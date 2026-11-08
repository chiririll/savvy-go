package ledger

import (
	"context"
	"fmt"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

// Reconciliation is a point that pins an account's balance: at the end of Date
// the balance is ActualBalance (in the account's currency), whatever the
// payments before it add up to. There is one per account and date. Date is
// YYYY-MM-DD, as for a transaction.
type Reconciliation struct {
	ID            int64
	AccountID     int64
	Date          string
	ActualBalance money.Money
	Note          *string
	CreatedAt     *time.Time
}

// ReconciliationFields is what a caller writes: the point's account, date,
// balance and note. The balance carries the account's unit.
type ReconciliationFields struct {
	AccountID     int64
	Date          string
	ActualBalance money.Money
	Note          *string
}

func reconciliationOf(x sqlc.Reconciliation, unit money.Unit) Reconciliation {
	return Reconciliation{
		ID: x.ID, AccountID: x.AccountID, Date: x.Date, ActualBalance: money.New(x.ActualBalance, unit),
		Note: optString(x.Note), CreatedAt: optTime(x.CreatedAt),
	}
}

// reconciliationsOf converts rows of any accounts, reading each account's unit once.
func (r Repo) reconciliationsOf(ctx context.Context, rows []sqlc.Reconciliation) ([]Reconciliation, error) {
	units := map[int64]money.Unit{}
	out := make([]Reconciliation, len(rows))
	for i, x := range rows {
		unit, ok := units[x.AccountID]
		if !ok {
			u, err := r.AccountUnit(ctx, x.AccountID)
			if err != nil {
				return nil, err
			}
			if u == nil {
				return nil, fmt.Errorf("reconciliation %d: account %d is gone", x.ID, x.AccountID)
			}
			unit, units[x.AccountID] = *u, *u
		}
		out[i] = reconciliationOf(x, unit)
	}
	return out, nil
}

// UpsertReconciliation writes the point of f.AccountID on f.Date, replacing
// the balance and note of one already there (the id and created_at stay).
// The account must exist.
func (r Repo) UpsertReconciliation(ctx context.Context, f ReconciliationFields, now time.Time) (*Reconciliation, error) {
	x, err := r.q().UpsertReconciliation(ctx, sqlc.UpsertReconciliationParams{
		AccountID: f.AccountID, Date: f.Date, ActualBalance: f.ActualBalance.Minor(),
		Note: db.NullString(f.Note), CreatedAt: db.TimeNS(now),
	})
	if err != nil {
		return nil, err
	}
	out := reconciliationOf(x, f.ActualBalance.Unit())
	return &out, nil
}

// Reconciliation returns nil for an unknown id.
func (r Repo) Reconciliation(ctx context.Context, id int64) (*Reconciliation, error) {
	x, err := r.q().GetReconciliation(ctx, id)
	if err != nil {
		return nilIfMissing[Reconciliation](err)
	}
	rows, err := r.reconciliationsOf(ctx, []sqlc.Reconciliation{x})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

// DeleteReconciliation reports whether a point was removed.
func (r Repo) DeleteReconciliation(ctx context.Context, id int64) (bool, error) {
	n, err := r.q().DeleteReconciliation(ctx, id)
	return n > 0, err
}

// AccountReconciliations lists the points of one account, oldest date first.
func (r Repo) AccountReconciliations(ctx context.Context, accountID int64) ([]Reconciliation, error) {
	rows, err := r.q().ListAccountReconciliations(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return r.reconciliationsOf(ctx, rows)
}

// ReconciliationsBetween lists the points of all accounts dated from..to
// inclusive ("" leaves that end open), by date, then account. A caller that
// wants some accounts filters the result.
func (r Repo) ReconciliationsBetween(ctx context.Context, from, to string) ([]Reconciliation, error) {
	rows, err := r.q().ListReconciliationsBetween(ctx, sqlc.ListReconciliationsBetweenParams{
		DateFrom: db.Narg(from), DateTo: db.Narg(to),
	})
	if err != nil {
		return nil, err
	}
	return r.reconciliationsOf(ctx, rows)
}

// LatestReconciliation is the newest point of the account dated on or before
// asOf; nil when there is none.
func (r Repo) LatestReconciliation(ctx context.Context, accountID int64, asOf string) (*Reconciliation, error) {
	x, err := r.q().LatestReconciliationAsOf(ctx, sqlc.LatestReconciliationAsOfParams{AccountID: accountID, Date: asOf})
	if err != nil {
		return nilIfMissing[Reconciliation](err)
	}
	rows, err := r.reconciliationsOf(ctx, []sqlc.Reconciliation{x})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

// RestoreReconciliation writes p back under its own id: it re-creates a point
// that was deleted, or puts the date, balance and note of one that exists back.
// A different point on the same account and date is a constraint error.
func (r Repo) RestoreReconciliation(ctx context.Context, p Reconciliation) (*Reconciliation, error) {
	x, err := r.q().RestoreReconciliation(ctx, sqlc.RestoreReconciliationParams{
		ID: p.ID, AccountID: p.AccountID, Date: p.Date, ActualBalance: p.ActualBalance.Minor(),
		Note: db.NullString(p.Note), CreatedAt: db.NullTime(p.CreatedAt),
	})
	if err != nil {
		return nil, err
	}
	out := reconciliationOf(x, p.ActualBalance.Unit())
	return &out, nil
}

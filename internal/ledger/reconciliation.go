package ledger

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"savvy-go/internal/audit"
	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

// EntityReconciliation is the entity type of a reconciliation point in the
// audit log.
const EntityReconciliation = "reconciliation"

var (
	// ErrNoReconcileItems is returned for a call with no account to reconcile.
	ErrNoReconcileItems = errors.New("reconcile at least one account")
	// ErrReconcileDate is returned for a date that is not a YYYY-MM-DD day.
	ErrReconcileDate = errors.New("the date must be a day as YYYY-MM-DD")
	// ErrReconcileFuture is returned for a date after today (in the app timezone):
	// a balance cannot be known before it happens.
	ErrReconcileFuture = errors.New("a reconciliation cannot be dated in the future")
	// ErrReconcileScale is returned for a balance with more decimal places than
	// the account's currency has. It is refused, not rounded, so a typo does
	// not pin the balance to a different amount than the one entered.
	ErrReconcileScale = errors.New("the balance has more decimal places than the currency")
	// ErrUnknownReconciliation is returned for a point id that does not exist.
	ErrUnknownReconciliation = errors.New("unknown reconciliation")
)

// Reconciliations are the points that pin an account's balance to the real
// one on a date. Writing them is audited and can be rolled back (Rollbacks).
type Reconciliations struct{ DB store.DB }

func (s Reconciliations) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// ReconcileItem is one account's real balance at the end of the date of a
// Reconcile call. Actual is in the account's currency.
type ReconcileItem struct {
	AccountID int64
	Actual    decimal.Decimal
	Note      *string
}

// ReconciliationView is a point with what the ledger expected at it. All
// amounts but DifferenceBase are in the account's currency. Difference is
// Actual - Expected (see ReconciliationCheck). DifferenceBase is Difference in
// the base currency at the current rates, nil when the account's currency has
// no rate.
type ReconciliationView struct {
	ID             int64
	AccountID      int64
	Date           string
	Actual         money.Money
	Expected       money.Money
	Difference     money.Money
	Note           *string
	DifferenceBase *money.Money
}

// ReconciliationFilter narrows List. No AccountIDs: every account. From and To
// (YYYY-MM-DD, inclusive) are open when empty.
type ReconciliationFilter struct {
	AccountIDs []int64
	From, To   string
}

// reconciliationSnapshot is how a point is stored in the audit log: enough to
// put it back under its id.
type reconciliationSnapshot struct {
	ID        int64      `json:"id"`
	AccountID int64      `json:"account_id"`
	Date      string     `json:"date"`
	Minor     int64      `json:"minor"`
	Note      *string    `json:"note,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

func snapshotOf(p *repo.Reconciliation) *reconciliationSnapshot {
	if p == nil {
		return nil
	}
	return &reconciliationSnapshot{
		ID: p.ID, AccountID: p.AccountID, Date: p.Date, Minor: p.ActualBalance.Minor(), Note: p.Note, CreatedAt: p.CreatedAt,
	}
}

func (x reconciliationSnapshot) point(unit money.Unit) repo.Reconciliation {
	return repo.Reconciliation{
		ID: x.ID, AccountID: x.AccountID, Date: x.Date, ActualBalance: money.New(x.Minor, unit), Note: x.Note,
		CreatedAt: x.CreatedAt,
	}
}

// Reconcile pins the balance of each account of items on date, all or none in
// one database transaction, with one audit batch. A point that exists for an
// account and date is replaced (an update in the log); one that would not
// change is left alone and not logged. now is the clock for "not in the
// future", judged in the app timezone. Views come back in the order of items.
//
// Errors (all checkable with errors.Is, wrapped with the item's position when
// one item is at fault): ErrNoReconcileItems, ErrReconcileDate,
// ErrReconcileFuture, ErrDuplicateAccount, ErrUnknownAccount, ErrServiceAccount,
// ErrReconcileScale, money.ErrOutOfRange.
func (s Reconciliations) Reconcile(
	ctx context.Context, date string, items []ReconcileItem, now time.Time,
) ([]ReconciliationView, error) {
	if len(items) == 0 {
		return nil, ErrNoReconcileItems
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil || day.Format("2006-01-02") != date {
		return nil, ErrReconcileDate
	}
	if date > now.In(appLocation).Format("2006-01-02") {
		return nil, ErrReconcileFuture
	}
	seen := map[int64]bool{}
	for _, it := range items {
		if seen[it.AccountID] {
			return nil, ErrDuplicateAccount
		}
		seen[it.AccountID] = true
	}
	batch := uuid.NewString()
	ids := make([]int64, len(items))
	err = store.Tx(ctx, s.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		for i, it := range items {
			fields, err := s.fields(ctx, r, date, it)
			if err != nil {
				return fmt.Errorf("item %d: %w", i+1, err)
			}
			before, err := pointOn(ctx, r, it.AccountID, date)
			if err != nil {
				return err
			}
			after, err := r.UpsertReconciliation(ctx, fields, now.UTC())
			if err != nil {
				return err
			}
			ids[i] = after.ID
			op := audit.OpCreate
			if before != nil {
				op = audit.OpUpdate
				if sameValues(*before, *after) {
					continue
				}
			}
			if _, err := (audit.Log{}).Record(ctx, d, audit.Change{
				EntityType: EntityReconciliation, EntityID: after.ID, Op: op, BatchID: batch,
				Before: audit.Snapshot(snapshotOf(before)), After: audit.Snapshot(snapshotOf(after)),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	accounts := make([]int64, len(items))
	for i, it := range items {
		accounts[i] = it.AccountID
	}
	views, err := s.List(ctx, ReconciliationFilter{AccountIDs: accounts, From: date, To: date})
	if err != nil {
		return nil, err
	}
	out := make([]ReconciliationView, 0, len(items))
	for _, id := range ids {
		for _, v := range views {
			if v.ID == id {
				out = append(out, v)
			}
		}
	}
	return out, nil
}

func sameValues(a, b repo.Reconciliation) bool {
	return a.ActualBalance.Minor() == b.ActualBalance.Minor() && (a.Note == nil) == (b.Note == nil) &&
		(a.Note == nil || *a.Note == *b.Note)
}

// fields checks one item against its account and builds what is written.
func (s Reconciliations) fields(ctx context.Context, r repo.Repo, date string, it ReconcileItem) (repo.ReconciliationFields, error) {
	acc, err := r.Account(ctx, it.AccountID)
	if err != nil {
		return repo.ReconciliationFields{}, err
	}
	if acc == nil {
		return repo.ReconciliationFields{}, ErrUnknownAccount
	}
	if acc.IsService || isServiceType(acc.Type) {
		return repo.ReconciliationFields{}, ErrServiceAccount
	}
	unit := currencyFrom(acc.Currency).Unit()
	if it.Actual.Exponent() < -int32(unit.Decimals) && !it.Actual.Equal(it.Actual.Round(int32(unit.Decimals))) {
		return repo.ReconciliationFields{}, ErrReconcileScale
	}
	actual, err := money.FromInput(it.Actual, unit)
	if err != nil {
		return repo.ReconciliationFields{}, err
	}
	return repo.ReconciliationFields{AccountID: it.AccountID, Date: date, ActualBalance: actual, Note: it.Note}, nil
}

// pointOn is the point of the account on exactly date, nil when there is none.
func pointOn(ctx context.Context, r repo.Repo, accountID int64, date string) (*repo.Reconciliation, error) {
	p, err := r.LatestReconciliation(ctx, accountID, date)
	if err != nil || p == nil || p.Date != date {
		return nil, err
	}
	return p, nil
}

// Delete removes the point id and logs it. ErrUnknownReconciliation when it
// does not exist.
func (s Reconciliations) Delete(ctx context.Context, id int64) error {
	return store.Tx(ctx, s.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		before, err := r.Reconciliation(ctx, id)
		if err != nil {
			return err
		}
		if before == nil {
			return ErrUnknownReconciliation
		}
		if _, err := r.DeleteReconciliation(ctx, id); err != nil {
			return err
		}
		_, err = audit.Log{}.Record(ctx, d, audit.Change{
			EntityType: EntityReconciliation, EntityID: id, Op: audit.OpDelete, Before: audit.Snapshot(snapshotOf(before)),
		})
		return err
	})
}

// List returns the points matching f with what the ledger expected at each,
// newest date first, then in the order of the accounts (their sort order, then
// id), then by point id. Expected and Difference are measured against the
// previous point of the account even when that one is outside the range, so
// the figures do not depend on the filter.
func (s Reconciliations) List(ctx context.Context, f ReconciliationFilter) ([]ReconciliationView, error) {
	rows, err := s.repo().Accounts(ctx, repo.AccountFilter{})
	if err != nil {
		return nil, err
	}
	accounts := map[int64]Account{}
	for _, row := range rows {
		accounts[row.ID] = accountFrom(row)
	}
	in, err := s.repo().ReconciliationsBetween(ctx, f.From, f.To)
	if err != nil {
		return nil, err
	}
	involved := map[int64]bool{}
	for _, p := range in {
		if len(f.AccountIDs) == 0 || slices.Contains(f.AccountIDs, p.AccountID) {
			involved[p.AccountID] = true
		}
	}
	inRange := map[int64]bool{}
	for _, p := range in {
		inRange[p.ID] = true
	}
	base, err := (Currencies{DB: s.DB}).Base(ctx)
	if err != nil {
		return nil, err
	}
	var out []ReconciliationView
	for id := range involved {
		acc, ok := accounts[id]
		if !ok {
			continue
		}
		checks, err := (Accounts{DB: s.DB}).ReconciliationChecks(ctx, acc)
		if err != nil {
			return nil, err
		}
		for _, c := range checks {
			if !inRange[c.Point.ID] {
				continue
			}
			v := ReconciliationView{
				ID: c.Point.ID, AccountID: id, Date: c.Point.Date, Actual: c.Point.ActualBalance, Expected: c.Expected,
				Difference: c.Difference, Note: c.Point.Note,
			}
			if base != nil && (acc.Currency.ID == base.ID || !acc.Currency.Rate.IsZero()) {
				if m, err := TryConvert(c.Difference, *acc.Currency, *base); err == nil {
					v.DifferenceBase = &m
				}
			}
			out = append(out, v)
		}
	}
	slices.SortFunc(out, func(a, b ReconciliationView) int {
		if a.Date != b.Date {
			if a.Date > b.Date {
				return -1
			}
			return 1
		}
		if a.AccountID != b.AccountID {
			x, y := accounts[a.AccountID], accounts[b.AccountID]
			if x.SortOrder != y.SortOrder {
				return x.SortOrder - y.SortOrder
			}
			return int(a.AccountID - b.AccountID)
		}
		return int(a.ID - b.ID)
	})
	return out, nil
}

// ReconciliationDay is the points of one date for the transaction list.
// DifferenceBase is the sum of the points' differences in the base currency
// (zero when none converts); Unconverted says some point had no rate and is
// not in the sum.
type ReconciliationDay struct {
	Date           string
	DifferenceBase money.Money
	Unconverted    bool
	Points         []ReconciliationView
}

// DayReconciliations groups the points dated from..to (inclusive, "" is open)
// by date, newest first, each with the sum of its differences in the base
// currency.
func (s Reconciliations) DayReconciliations(ctx context.Context, from, to string) ([]ReconciliationDay, error) {
	list, err := s.List(ctx, ReconciliationFilter{From: from, To: to})
	if err != nil {
		return nil, err
	}
	base, err := (Currencies{DB: s.DB}).Base(ctx)
	if err != nil {
		return nil, err
	}
	var unit money.Unit
	if base != nil {
		unit = base.Unit()
	}
	var out []ReconciliationDay
	for _, v := range list {
		if len(out) == 0 || out[len(out)-1].Date != v.Date {
			out = append(out, ReconciliationDay{Date: v.Date, DifferenceBase: money.Zero(unit)})
		}
		day := &out[len(out)-1]
		day.Points = append(day.Points, v)
		if v.DifferenceBase == nil {
			day.Unconverted = true
			continue
		}
		day.DifferenceBase = day.DifferenceBase.Add(*v.DifferenceBase)
	}
	return out, nil
}

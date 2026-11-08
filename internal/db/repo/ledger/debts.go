package ledger

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

// Debt is an account of type debt with its debts row. Principal and the
// account's balance are in the account currency; a balance is signed, so what
// is still owed is its absolute value.
type Debt struct {
	Account
	Direction    string // i_owe or owed_to_me
	Principal    money.Money
	DueDate      *string
	Counterparty *string
	Note         *string
	ClosedAt     *time.Time
}

// DebtFields is the writable part of a debt; Principal is already in the
// account currency.
type DebtFields struct {
	Direction    string
	Principal    money.Money
	DueDate      *string
	Counterparty *string
	Note         *string
}

type debtRow = sqlc.ListDebtsRow

func debtOf(x debtRow) Debt {
	a := accountOf(accountRow{
		ID: x.ID, Name: x.Name, Type: x.Type, IsService: x.IsService, CurrencyID: x.CurrencyID,
		InitialBalance: x.InitialBalance, IsActive: x.IsActive, SortOrder: x.SortOrder, CreatedAt: x.CreatedAt,
		CurrencyIDJoin: x.CurrencyIDJoin, Code: x.Code, CurrencyName: x.CurrencyName, Symbol: x.Symbol,
		Decimals: x.Decimals, IsBase: x.IsBase, Rate: x.Rate,
	})
	return Debt{
		Account: a, Direction: x.Direction, Principal: money.New(x.Principal, a.Currency.Unit()),
		DueDate: optString(x.DueDate), Counterparty: optString(x.Counterparty), Note: optString(x.Note),
		ClosedAt: optTime(x.ClosedAt),
	}
}

// Debts lists the debts, the open ones only when openOnly.
func (r Repo) Debts(ctx context.Context, openOnly bool) ([]Debt, error) {
	rows, err := r.q().ListDebts(ctx, db.Flag(openOnly))
	if err != nil {
		return nil, err
	}
	out := make([]Debt, len(rows))
	for i, x := range rows {
		out[i] = debtOf(x)
	}
	return out, nil
}

// Debt returns nil when the id is not a debt account.
func (r Repo) Debt(ctx context.Context, id int64) (*Debt, error) {
	x, err := r.q().GetDebt(ctx, id)
	if err != nil {
		return nilIfMissing[Debt](err)
	}
	d := debtOf(debtRow(x))
	return &d, nil
}

// InsertDebt adds the debts row of an account of type debt.
func (r Repo) InsertDebt(ctx context.Context, accountID int64, d DebtFields) error {
	return r.q().InsertDebt(ctx, sqlc.InsertDebtParams{
		AccountID: accountID, Direction: d.Direction, Principal: d.Principal.Minor(),
		DueDate: db.NullString(d.DueDate), Counterparty: db.NullString(d.Counterparty), Note: db.NullString(d.Note),
	})
}

func (r Repo) UpdateDebt(ctx context.Context, accountID int64, d DebtFields) error {
	return r.q().UpdateDebt(ctx, sqlc.UpdateDebtParams{
		Direction: d.Direction, Principal: d.Principal.Minor(), DueDate: db.NullString(d.DueDate),
		Counterparty: db.NullString(d.Counterparty), Note: db.NullString(d.Note), AccountID: accountID,
	})
}

// CloseDebt marks a debt settled as of at; ReopenDebt clears it.
func (r Repo) CloseDebt(ctx context.Context, accountID int64, at time.Time) error {
	return r.q().CloseDebt(ctx, sqlc.CloseDebtParams{ClosedAt: db.TimeNS(at), AccountID: accountID})
}

func (r Repo) ReopenDebt(ctx context.Context, accountID int64) error {
	return r.q().ReopenDebt(ctx, accountID)
}

// CountDebtHistory counts the payments and collections made on a debt.
func (r Repo) CountDebtHistory(ctx context.Context, id int64) int64 {
	n, _ := r.q().CountDebtHistory(ctx, id)
	return n
}

// DeleteDebt removes a debt account (its debts row goes with it) and the
// transaction that issued it.
func (r Repo) DeleteDebt(ctx context.Context, id int64) error {
	if err := r.q().DeleteDebtIssuance(ctx, id); err != nil {
		return err
	}
	return r.q().DeleteDebtAccount(ctx, id)
}

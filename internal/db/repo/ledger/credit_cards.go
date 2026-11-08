package ledger

import (
	"context"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

// The two ways a credit card's billing period runs. Only fixed_day is used;
// first_expense is in the schema for later.
const (
	CycleFixedDay     = "fixed_day"
	CycleFirstExpense = "first_expense"
)

// CreditCard is the billing terms of a credit card. Limit is in the account
// currency. StatementDay is set for fixed_day and PeriodDays for first_expense;
// the other is nil.
type CreditCard struct {
	Limit        money.Money
	Cycle        string
	StatementDay *int
	PeriodDays   *int
	GraceDays    int
}

// CardAccount is an account of type credit with its credit_cards row.
type CardAccount struct {
	Account
	Terms CreditCard
}

type cardRow = sqlc.ListCreditCardsRow

func cardOf(x cardRow) CardAccount {
	a := accountOf(accountRow{
		ID: x.ID, Name: x.Name, Type: x.Type, IsService: x.IsService, CurrencyID: x.CurrencyID,
		InitialBalance: x.InitialBalance, IsActive: x.IsActive, SortOrder: x.SortOrder, CreatedAt: x.CreatedAt,
		CurrencyIDJoin: x.CurrencyIDJoin, Code: x.Code, CurrencyName: x.CurrencyName, Symbol: x.Symbol,
		Decimals: x.Decimals, IsBase: x.IsBase, Rate: x.Rate,
	})
	return CardAccount{Account: a, Terms: CreditCard{
		Limit: money.New(x.CreditLimit, a.Currency.Unit()), Cycle: x.Cycle,
		StatementDay: optInt(x.StatementDay), PeriodDays: optInt(x.PeriodDays), GraceDays: int(x.GraceDays),
	}}
}

// CreditCards lists the credit cards, the active ones only when onlyActive.
func (r Repo) CreditCards(ctx context.Context, onlyActive bool) ([]CardAccount, error) {
	rows, err := r.q().ListCreditCards(ctx, db.Flag(onlyActive))
	if err != nil {
		return nil, err
	}
	out := make([]CardAccount, len(rows))
	for i, x := range rows {
		out[i] = cardOf(x)
	}
	return out, nil
}

// CreditCard returns nil when the id is not a credit card.
func (r Repo) CreditCard(ctx context.Context, id int64) (*CardAccount, error) {
	x, err := r.q().GetCreditCard(ctx, id)
	if err != nil {
		return nilIfMissing[CardAccount](err)
	}
	c := cardOf(cardRow(x))
	return &c, nil
}

// SaveCreditCard writes the terms of a card, replacing the ones it had. The
// limit is already in the account currency.
func (r Repo) SaveCreditCard(ctx context.Context, accountID int64, c CreditCard) error {
	return r.q().UpsertCreditCard(ctx, sqlc.UpsertCreditCardParams{
		AccountID: accountID, CreditLimit: c.Limit.Minor(), Cycle: c.Cycle,
		StatementDay: db.NullInt(c.StatementDay), PeriodDays: db.NullInt(c.PeriodDays), GraceDays: int64(c.GraceDays),
	})
}

// DeleteCreditCard removes a card account (its terms go with it).
func (r Repo) DeleteCreditCard(ctx context.Context, id int64) error {
	return r.q().DeleteCreditCardAccount(ctx, id)
}

// DayFlow is the money of an account on one day, in minor units: the net
// change, what came in (positive payments) and what went out (negative ones,
// so Outflow <= 0).
type DayFlow struct {
	Day                    string
	Delta, Inflow, Outflow int64
}

// DailyFlows is DailyChanges with the two directions apart, for all days.
func (r Repo) DailyFlows(ctx context.Context, id int64) ([]DayFlow, error) {
	rows, err := r.q().AccountDailyFlows(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]DayFlow, len(rows))
	for i, x := range rows {
		out[i] = DayFlow{Day: x.Day.String, Delta: x.Delta, Inflow: x.Inflow, Outflow: x.Outflow}
	}
	return out, nil
}

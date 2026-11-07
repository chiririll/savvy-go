// Package ledgerrepo reads and writes the finance tables of one space
// database: accounts, categories, currencies, budgets, tags and transactions.
// Package ledger owns the rules (conversion, balances, what may change); this
// package only moves rows. Amounts cross it as money.Money, so a number never
// loses its currency, and "no such row" is a nil result.
package ledger

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

// Repo works on one space database or on a transaction on it.
type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

func optTime(raw db.NString) *time.Time {
	if t, ok := db.ParseTime(raw); ok {
		return &t
	}
	return nil
}

func optString(raw db.NString) *string {
	if !raw.Valid {
		return nil
	}
	return &raw.String
}

func optInt64(raw db.NInt) *int64 {
	if !raw.Valid {
		return nil
	}
	return &raw.Int64
}

func optInt(raw db.NInt) *int {
	if !raw.Valid {
		return nil
	}
	v := int(raw.Int64)
	return &v
}

// Currencies

type Currency struct {
	ID                 int64
	Code, Name, Symbol string
	Decimals           int
	IsBase             bool
	Rate               decimal.Decimal
}

func (c Currency) Unit() money.Unit { return money.Unit{ID: c.ID, Decimals: c.Decimals} }

func currencyOf(id int64, code, name, symbol string, decimals, isBase int64, rate decimal.Decimal) Currency {
	return Currency{
		ID:       id,
		Code:     code,
		Name:     name,
		Symbol:   symbol,
		Decimals: int(decimals),
		IsBase:   isBase != 0,
		Rate:     rate,
	}
}

func (r Repo) Currencies(ctx context.Context) ([]Currency, error) {
	rows, err := r.q().ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Currency, len(rows))
	for i, x := range rows {
		out[i] = currencyOf(x.ID, x.Code, x.Name, x.Symbol, x.Decimals, x.IsBase, x.Rate)
	}
	return out, nil
}

// Currency, CurrencyByCode and BaseCurrency return nil when there is none.
func (r Repo) Currency(ctx context.Context, id int64) (*Currency, error) {
	x, err := r.q().GetCurrency(ctx, id)
	if err != nil {
		return nilIfMissing[Currency](err)
	}
	c := currencyOf(x.ID, x.Code, x.Name, x.Symbol, x.Decimals, x.IsBase, x.Rate)
	return &c, nil
}

func (r Repo) CurrencyByCode(ctx context.Context, code string) (*Currency, error) {
	x, err := r.q().GetCurrencyByCode(ctx, code)
	if err != nil {
		return nilIfMissing[Currency](err)
	}
	c := currencyOf(x.ID, x.Code, x.Name, x.Symbol, x.Decimals, x.IsBase, x.Rate)
	return &c, nil
}

func (r Repo) BaseCurrency(ctx context.Context) (*Currency, error) {
	x, err := r.q().GetBaseCurrency(ctx)
	if err != nil {
		return nilIfMissing[Currency](err)
	}
	c := currencyOf(x.ID, x.Code, x.Name, x.Symbol, x.Decimals, x.IsBase, x.Rate)
	return &c, nil
}

// nilIfMissing turns "no rows" into a nil result and keeps other errors.
func nilIfMissing[T any](err error) (*T, error) {
	if err == db.ErrNotFound {
		return nil, nil
	}
	return nil, err
}

func (r Repo) CountCurrencies(ctx context.Context) (int, error) {
	n, err := r.q().CountCurrencies(ctx)
	return int(n), err
}

func (r Repo) CurrencyCodes(ctx context.Context) ([]string, error) {
	return r.q().ListCurrencyCodes(ctx)
}

func (r Repo) ClearBaseCurrency(ctx context.Context) error { return r.q().ClearBaseCurrency(ctx) }

func (r Repo) InsertCurrency(ctx context.Context, c Currency, now time.Time) (int64, error) {
	res, err := r.q().InsertCurrency(ctx, sqlc.InsertCurrencyParams{
		Code: c.Code, Name: c.Name, Symbol: c.Symbol, Decimals: int64(c.Decimals),
		IsBase: db.BoolInt(c.IsBase), Rate: c.Rate, CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) UpdateCurrency(ctx context.Context, id int64, c Currency, now time.Time) error {
	return r.q().UpdateCurrency(ctx, sqlc.UpdateCurrencyParams{
		Name: c.Name, Symbol: c.Symbol, Decimals: int64(c.Decimals), IsBase: db.BoolInt(c.IsBase),
		Rate: c.Rate, UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) DeleteCurrency(ctx context.Context, id int64) error {
	return r.q().DeleteCurrency(ctx, id)
}

// CurrencyInUse reports whether an account or budget uses the currency.
func (r Repo) CurrencyInUse(ctx context.Context, id int64) bool {
	accounts, _ := r.q().CountAccountsForCurrency(ctx, id)
	budgets, _ := r.q().CountBudgetsForCurrency(ctx, id)
	return accounts > 0 || budgets > 0
}

// OtherRate is the stored rate of a non-base currency.
type OtherRate struct {
	ID   int64
	Rate decimal.Decimal
}

// OtherRates lists the rates of every currency but the one with id.
func (r Repo) OtherRates(ctx context.Context, id int64) ([]OtherRate, error) {
	rows, err := r.q().ListOtherCurrencyRates(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]OtherRate, len(rows))
	for i, x := range rows {
		out[i] = OtherRate{ID: x.ID, Rate: x.Rate}
	}
	return out, nil
}

func (r Repo) SetCurrencyRate(ctx context.Context, id int64, rate decimal.Decimal) error {
	return r.q().UpdateCurrencyRate(ctx, sqlc.UpdateCurrencyRateParams{Rate: rate, ID: id})
}

func (r Repo) SetBaseCurrency(ctx context.Context, id int64, now time.Time) error {
	return r.q().SetCurrencyBase(ctx, sqlc.SetCurrencyBaseParams{UpdatedAt: db.TimeNS(now), ID: id})
}

// Accounts

// Account is an account with its currency; amounts are in that currency. A
// debt or space account is a service account (IsService): pickers and net
// worth leave it out. The data of a debt is in Debt.
type Account struct {
	ID             int64
	Name, Type     string
	IsService      bool
	Currency       Currency
	InitialBalance money.Money
	IsActive       bool
	SortOrder      int
	CreatedAt      *time.Time
}

type accountRow = sqlc.ListAccountsRow

func accountOf(x accountRow) Account {
	cur := currencyOf(x.CurrencyIDJoin, x.Code, x.CurrencyName, x.Symbol, x.Decimals, x.IsBase, x.Rate)
	return Account{
		ID: x.ID, Name: x.Name, Type: x.Type, IsService: x.IsService.Int64 != 0, Currency: cur,
		InitialBalance: money.New(x.InitialBalance, cur.Unit()),
		IsActive:       x.IsActive != 0, SortOrder: int(x.SortOrder), CreatedAt: optTime(x.CreatedAt),
	}
}

// MoneyFromNull reads a nullable stored amount; nil when NULL.
func MoneyFromNull(v db.NInt, unit money.Unit) *money.Money { return db.MoneyFromNull(v, unit) }

// AccountFilter narrows Accounts; the zero value lists all. ExcludeService
// leaves out debt and space accounts, ExcludeSpace only the space accounts.
type AccountFilter struct{ OnlyActive, ExcludeService, ExcludeSpace bool }

func (r Repo) Accounts(ctx context.Context, f AccountFilter) ([]Account, error) {
	rows, err := r.q().ListAccounts(ctx, sqlc.ListAccountsParams{
		OnlyActive: db.Flag(f.OnlyActive), ExcludeService: db.Flag(f.ExcludeService), ExcludeSpace: db.Flag(f.ExcludeSpace),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Account, len(rows))
	for i, x := range rows {
		out[i] = accountOf(x)
	}
	return out, nil
}

// Account returns nil for an unknown id.
func (r Repo) Account(ctx context.Context, id int64) (*Account, error) {
	x, err := r.q().GetAccount(ctx, id)
	if err != nil {
		return nilIfMissing[Account](err)
	}
	a := accountOf(accountRow(x))
	return &a, nil
}

// SpaceAccountID is the id of the account of type space for a currency; nil
// when none has been made yet.
func (r Repo) SpaceAccountID(ctx context.Context, currencyID int64) (*int64, error) {
	id, err := r.q().GetSpaceAccountID(ctx, currencyID)
	if err != nil {
		return nilIfMissing[int64](err)
	}
	return &id, nil
}

// AccountFields is the writable part of an account; amounts are already in
// the account currency.
type AccountFields struct {
	Name, Type     string
	CurrencyID     int64
	InitialBalance money.Money
	IsActive       bool
	SortOrder      int64 // insert only
}

// NextAccountSortOrder is the next place at the end of the debts (debt) or of
// the other accounts.
func (r Repo) NextAccountSortOrder(ctx context.Context, debt bool) (int64, error) {
	return r.q().NextAccountSortOrder(ctx, db.Flag(debt))
}

func (r Repo) InsertAccount(ctx context.Context, a AccountFields, now time.Time) (int64, error) {
	res, err := r.q().InsertAccount(ctx, sqlc.InsertAccountParams{
		Name: a.Name, Type: a.Type, CurrencyID: a.CurrencyID, InitialBalance: a.InitialBalance.Minor(),
		IsActive: db.BoolInt(a.IsActive), SortOrder: a.SortOrder, CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) UpdateAccount(ctx context.Context, id int64, a AccountFields, now time.Time) error {
	return r.q().UpdateAccount(ctx, sqlc.UpdateAccountParams{
		Name: a.Name, Type: a.Type, CurrencyID: a.CurrencyID, InitialBalance: a.InitialBalance.Minor(),
		IsActive: db.BoolInt(a.IsActive), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// CountAccountPayments counts the payments on an account, i.e. the
// transactions that moved its money.
func (r Repo) CountAccountPayments(ctx context.Context, id int64) int64 {
	n, _ := r.q().CountAccountPayments(ctx, id)
	return n
}

func (r Repo) DeleteAccount(ctx context.Context, id int64) error { return r.q().DeleteAccount(ctx, id) }

func (r Repo) SetAccountSortOrder(ctx context.Context, id int64, order int) error {
	return r.q().SetAccountSortOrder(ctx, sqlc.SetAccountSortOrderParams{SortOrder: int64(order), ID: id})
}

// AccountUnit is the unit of an account's currency; nil for an unknown account.
func (r Repo) AccountUnit(ctx context.Context, id int64) (*money.Unit, error) {
	u, err := r.q().GetAccountUnit(ctx, id)
	if err != nil {
		return nilIfMissing[money.Unit](err)
	}
	return &money.Unit{ID: u.ID, Decimals: int(u.Decimals)}, nil
}

// CurrencyUnit is the unit of a currency; nil for an unknown currency.
func (r Repo) CurrencyUnit(ctx context.Context, id int64) (*money.Unit, error) {
	c, err := r.Currency(ctx, id)
	if c == nil || err != nil {
		return nil, err
	}
	u := c.Unit()
	return &u, nil
}

// DayChange is an account's net change on one day, in minor units.
type DayChange struct {
	Day   string
	Delta int64
}

// DailyChanges is the net change of the account's payments per day up to asOf
// ("" for all), oldest first. Add InitialBalance for a balance.
func (r Repo) DailyChanges(ctx context.Context, id int64, asOf string) ([]DayChange, error) {
	rows, err := r.q().AccountDailyDeltas(ctx, sqlc.AccountDailyDeltasParams{ID: id, AsOf: db.Narg(asOf)})
	if err != nil {
		return nil, err
	}
	out := make([]DayChange, len(rows))
	for i, x := range rows {
		out[i] = DayChange{Day: x.Day.String, Delta: x.Delta}
	}
	return out, nil
}

// DeltaSum is the sum of DailyChanges: what the account's payments add up to
// up to asOf ("" for all), in one row.
func (r Repo) DeltaSum(ctx context.Context, id int64, asOf string) (int64, error) {
	return r.q().AccountDeltaSum(ctx, sqlc.AccountDeltaSumParams{ID: id, AsOf: db.Narg(asOf)})
}

// Categories

// Tags

// Budgets

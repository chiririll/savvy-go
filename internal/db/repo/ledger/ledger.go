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

// Account is an account with its currency; amounts are in that currency.
type Account struct {
	ID                     int64
	Name, Type             string
	Currency               Currency
	InitialBalance         money.Money
	IsActive, IsPaidOff    bool
	SortOrder              int
	DebtType, DueDate      *string
	TargetAmount           *money.Money
	Counterparty, DebtDesc *string
	CreatedAt              *time.Time
}

type accountRow = sqlc.ListAccountsRow

func accountOf(x accountRow) Account {
	cur := currencyOf(x.CurrencyIDJoin, x.Code, x.CurrencyName, x.Symbol, x.Decimals, x.IsBase, x.Rate)
	return Account{
		ID: x.ID, Name: x.Name, Type: x.Type, Currency: cur,
		InitialBalance: money.New(x.InitialBalance, cur.Unit()),
		IsActive:       x.IsActive != 0, IsPaidOff: x.IsPaidOff != 0, SortOrder: int(x.SortOrder),
		DebtType: optString(x.DebtType), DueDate: optString(x.DueDate),
		TargetAmount: MoneyFromNull(x.TargetAmount, cur.Unit()),
		Counterparty: optString(x.Counterparty), DebtDesc: optString(x.DebtDescription),
		CreatedAt: optTime(x.CreatedAt),
	}
}

// MoneyFromNull reads a nullable stored amount; nil when NULL.
func MoneyFromNull(v db.NInt, unit money.Unit) *money.Money { return db.MoneyFromNull(v, unit) }

// AccountFilter narrows Accounts; the zero value lists all.
type AccountFilter struct{ OnlyActive, ExcludeDebts, OnlyDebts, UnpaidOnly bool }

func (r Repo) Accounts(ctx context.Context, f AccountFilter) ([]Account, error) {
	rows, err := r.q().ListAccounts(ctx, sqlc.ListAccountsParams{
		OnlyActive: db.Flag(f.OnlyActive), ExcludeDebts: db.Flag(f.ExcludeDebts),
		OnlyDebts: db.Flag(f.OnlyDebts), UnpaidOnly: db.Flag(f.UnpaidOnly),
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

// AccountFields is the writable part of an account; amounts are already in
// the account currency.
type AccountFields struct {
	Name, Type             string
	CurrencyID             int64
	InitialBalance         money.Money
	TargetAmount           *money.Money
	DueDate, DebtType      *string
	Counterparty, DebtDesc *string
	IsActive, IsPaidOff    bool
	SortOrder              int64 // insert only
}

func (r Repo) NextAccountSortOrder(ctx context.Context, debt bool) (int64, error) {
	return r.q().NextAccountSortOrder(ctx, db.Flag(debt))
}

func (r Repo) InsertAccount(ctx context.Context, a AccountFields, now time.Time) (int64, error) {
	res, err := r.q().InsertAccount(ctx, sqlc.InsertAccountParams{
		Name: a.Name, Type: a.Type, DebtType: db.NullString(a.DebtType), CurrencyID: a.CurrencyID,
		InitialBalance: a.InitialBalance.Minor(), TargetAmount: db.NullMoney(a.TargetAmount), DueDate: db.NullString(a.DueDate),
		IsPaidOff: db.BoolInt(
			a.IsPaidOff,
		), Counterparty: db.NullString(a.Counterparty), DebtDescription: db.NullString(a.DebtDesc),
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
		IsActive: db.BoolInt(a.IsActive), DebtType: db.NullString(a.DebtType), TargetAmount: db.NullMoney(a.TargetAmount),
		DueDate: db.NullString(a.DueDate), IsPaidOff: db.BoolInt(a.IsPaidOff), Counterparty: db.NullString(a.Counterparty),
		DebtDescription: db.NullString(a.DebtDesc), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// CountAccountTransactions counts the transactions that touch an account.
func (r Repo) CountAccountTransactions(ctx context.Context, id int64) int64 {
	n, _ := r.q().CountAccountTransactions(ctx, sqlc.CountAccountTransactionsParams{AccountID: id, ToAccountID: db.NI(id)})
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

// DailyChanges is the account's net change per day up to asOf ("" for all),
// oldest first.
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

// Debts

// DebtPaid is how much has been paid into a debt, in minor units of its currency.
func (r Repo) DebtPaid(ctx context.Context, debtID int64) (int64, error) {
	return r.q().SumDebtPayments(ctx, db.NI(debtID))
}

func (r Repo) MarkPaidOff(ctx context.Context, id int64) error {
	return r.q().MarkAccountPaidOff(ctx, id)
}

func (r Repo) Reopen(ctx context.Context, id int64, now time.Time) error {
	return r.q().ReopenAccount(ctx, sqlc.ReopenAccountParams{UpdatedAt: db.TimeNS(now), ID: id})
}

func (r Repo) CountDebtHistory(ctx context.Context, id int64) int64 {
	n, _ := r.q().CountDebtHistory(ctx, db.NI(id))
	return n
}

// DeleteDebt removes a debt account and the transaction that issued it.
func (r Repo) DeleteDebt(ctx context.Context, id int64) error {
	_ = r.q().DeleteDebtIssuance(ctx, db.NI(id))
	return r.q().DeleteDebtAccount(ctx, id)
}

// Categories

type Category struct {
	ID                int64
	Name, Type        string
	Icon, Color       *string
	IsDefault         bool
	TransactionsCount int
}

func categoryOf(id int64, name, typ string, icon, color db.NString, isDefault, count int64) Category {
	return Category{
		ID:                id,
		Name:              name,
		Type:              typ,
		Icon:              optString(icon),
		Color:             optString(color),
		IsDefault:         isDefault != 0,
		TransactionsCount: int(count),
	}
}

// Categories lists categories, of one type when typ is not empty.
func (r Repo) Categories(ctx context.Context, typ string) ([]Category, error) {
	rows, err := r.q().ListCategories(ctx, db.Narg(typ))
	if err != nil {
		return nil, err
	}
	out := make([]Category, len(rows))
	for i, x := range rows {
		out[i] = categoryOf(x.ID, x.Name, x.Type, x.Icon, x.Color, x.IsDefault, x.Count)
	}
	return out, nil
}

func (r Repo) Category(ctx context.Context, id int64) (*Category, error) {
	x, err := r.q().GetCategory(ctx, id)
	if err != nil {
		return nilIfMissing[Category](err)
	}
	c := categoryOf(x.ID, x.Name, x.Type, x.Icon, x.Color, x.IsDefault, x.Count)
	return &c, nil
}

func (r Repo) ClearDefaultCategory(ctx context.Context, typ string) error {
	return r.q().ClearDefaultCategory(ctx, typ)
}

func (r Repo) InsertCategory(ctx context.Context, c Category, now time.Time) (int64, error) {
	res, err := r.q().InsertCategory(ctx, sqlc.InsertCategoryParams{
		Name: c.Name, Type: c.Type, Icon: db.NullString(c.Icon), Color: db.NullString(c.Color),
		IsDefault: db.BoolInt(c.IsDefault), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) UpdateCategory(ctx context.Context, id int64, c Category, now time.Time) error {
	return r.q().UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		Name: c.Name, Type: c.Type, Icon: db.NullString(c.Icon), Color: db.NullString(c.Color),
		IsDefault: db.BoolInt(c.IsDefault), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// ReplaceCategory moves a category's transactions to another and deletes it.
func (r Repo) ReplaceCategory(ctx context.Context, id, successor int64) error {
	return store.Tx(ctx, r.DB, func(tx store.DB) error {
		q := db.Q(tx)
		if err := q.ReassignCategoryTransactions(ctx, sqlc.ReassignCategoryTransactionsParams{
			CategoryID: db.NI(successor), CategoryID_2: db.NI(id),
		}); err != nil {
			return err
		}
		return q.DeleteCategory(ctx, id)
	})
}

func (r Repo) DeleteCategory(ctx context.Context, id int64) error {
	return r.q().DeleteCategory(ctx, id)
}

func (r Repo) CountCategoriesOfType(ctx context.Context, typ string) int64 {
	n, _ := r.q().CountCategoriesByType(ctx, typ)
	return n
}

func (r Repo) SetCategoryDefault(ctx context.Context, id int64, now time.Time) error {
	return r.q().SetCategoryDefault(ctx, sqlc.SetCategoryDefaultParams{UpdatedAt: db.TimeNS(now), ID: id})
}

// CategoryAmount is one currency's share of a category's transactions.
type CategoryAmount struct {
	Count    int
	Minor    int64
	Decimals int
	Rate     decimal.Decimal
}

// CategoryStatistics lists, per currency, the category's confirmed
// transactions between start and end ("" for no bound).
func (r Repo) CategoryStatistics(ctx context.Context, id int64, start, end string) []CategoryAmount {
	rows, _ := r.q().CategoryStatistics(ctx, sqlc.CategoryStatisticsParams{
		CategoryID: db.NI(id), StartDate: db.Narg(start), EndDate: db.Narg(end),
	})
	out := make([]CategoryAmount, len(rows))
	for i, x := range rows {
		out[i] = CategoryAmount{Count: int(x.Cnt), Minor: x.Total, Decimals: int(x.Decimals), Rate: x.Rate}
	}
	return out
}

// Tags

type Tag struct {
	ID                int64
	Name              string
	TransactionsCount int
	CreatedAt         *time.Time
}

func tagOf(id int64, name string, created db.NString, count int64) Tag {
	return Tag{ID: id, Name: name, TransactionsCount: int(count), CreatedAt: optTime(created)}
}

func (r Repo) Tags(ctx context.Context) ([]Tag, error) {
	rows, err := r.q().ListTags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, x := range rows {
		out[i] = tagOf(x.ID, x.Name, x.CreatedAt, x.Count)
	}
	return out, nil
}

func (r Repo) Tag(ctx context.Context, id int64) (*Tag, error) {
	x, err := r.q().GetTag(ctx, id)
	if err != nil {
		return nilIfMissing[Tag](err)
	}
	t := tagOf(x.ID, x.Name, x.CreatedAt, x.Count)
	return &t, nil
}

func (r Repo) InsertTag(ctx context.Context, name string, now time.Time) (int64, error) {
	res, err := r.q().
		InsertTag(ctx, sqlc.InsertTagParams{Name: name, CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now)})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) RenameTag(ctx context.Context, id int64, name string, now time.Time) error {
	return r.q().UpdateTag(ctx, sqlc.UpdateTagParams{Name: name, UpdatedAt: db.TimeNS(now), ID: id})
}

func (r Repo) DeleteTag(ctx context.Context, id int64) error { return r.q().DeleteTag(ctx, id) }

// Budgets

type Budget struct {
	ID                 int64
	Name, Period       string
	Amount             money.Money
	StartDate, EndDate *string
	IsGlobal, IsActive bool
	NotifyAtPercent    *int
}

type BudgetFields struct {
	Name, Period       string
	Amount             money.Money
	StartDate, EndDate *string
	IsGlobal, IsActive bool
	NotifyAtPercent    *int
}

// Budgets lists budgets, or the one with id when id is not nil.
func (r Repo) Budgets(ctx context.Context, id *int64) ([]Budget, error) {
	rows, err := r.q().ListBudgets(ctx, db.NullInt64(id))
	if err != nil {
		return nil, err
	}
	out := make([]Budget, len(rows))
	for i, x := range rows {
		out[i] = Budget{
			ID: x.ID, Name: x.Name, Period: x.Period, Amount: money.New(x.Amount, money.Unit{ID: x.CurrencyID, Decimals: int(x.Decimals)}),
			StartDate: optString(
				x.StartDate,
			), EndDate: optString(x.EndDate), IsGlobal: x.IsGlobal != 0, IsActive: x.IsActive != 0,
			NotifyAtPercent: optInt(x.NotifyAtPercent),
		}
	}
	return out, nil
}

func (r Repo) InsertBudget(ctx context.Context, b BudgetFields, now time.Time) (int64, error) {
	res, err := r.q().InsertBudget(ctx, sqlc.InsertBudgetParams{
		Name: b.Name, Amount: b.Amount.Minor(), CurrencyID: b.Amount.Unit().ID, Period: b.Period,
		StartDate: db.NullString(b.StartDate), EndDate: db.NullString(b.EndDate),
		IsGlobal: db.BoolInt(b.IsGlobal), NotifyAtPercent: db.NullInt(b.NotifyAtPercent),
		IsActive: db.BoolInt(b.IsActive), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) UpdateBudget(ctx context.Context, id int64, b BudgetFields, now time.Time) error {
	return r.q().UpdateBudget(ctx, sqlc.UpdateBudgetParams{
		Name: b.Name, Amount: b.Amount.Minor(), CurrencyID: b.Amount.Unit().ID, Period: b.Period,
		StartDate: db.NullString(b.StartDate), EndDate: db.NullString(b.EndDate),
		IsGlobal: db.BoolInt(b.IsGlobal), NotifyAtPercent: db.NullInt(b.NotifyAtPercent),
		IsActive: db.BoolInt(b.IsActive), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) DeleteBudget(ctx context.Context, id int64) error { return r.q().DeleteBudget(ctx, id) }

func (r Repo) BudgetCategories(ctx context.Context, id int64) ([]Category, error) {
	rows, err := r.q().ListBudgetCategories(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Category, len(rows))
	for i, x := range rows {
		out[i] = categoryOf(x.ID, x.Name, x.Type, x.Icon, x.Color, x.IsDefault, x.TransactionsCount)
	}
	return out, nil
}

func (r Repo) BudgetTags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := r.q().ListBudgetTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, x := range rows {
		out[i] = tagOf(x.ID, x.Name, x.CreatedAt, x.TransactionsCount)
	}
	return out, nil
}

// SetBudgetCategories replaces the categories a budget covers.
func (r Repo) SetBudgetCategories(ctx context.Context, id int64, categoryIDs []int64) error {
	if err := r.q().DeleteBudgetCategories(ctx, id); err != nil {
		return err
	}
	for _, c := range categoryIDs {
		if err := r.q().InsertBudgetCategory(ctx, sqlc.InsertBudgetCategoryParams{BudgetID: id, CategoryID: c}); err != nil {
			return err
		}
	}
	return nil
}

// SetBudgetTags replaces the tags a budget covers.
func (r Repo) SetBudgetTags(ctx context.Context, id int64, tagIDs []int64) error {
	if err := r.q().DeleteBudgetTags(ctx, id); err != nil {
		return err
	}
	for _, t := range tagIDs {
		if err := r.q().InsertBudgetTag(ctx, sqlc.InsertBudgetTagParams{BudgetID: id, TagID: t}); err != nil {
			return err
		}
	}
	return nil
}

// MonthlyBudget is a monthly limit with what is needed to fold it into the
// base currency.
type MonthlyBudget struct {
	Amount money.Money
	Rate   decimal.Decimal
	IsBase bool
}

// GlobalMonthlyBudget returns the global monthly budget; nil when there is none.
func (r Repo) GlobalMonthlyBudget(ctx context.Context) *MonthlyBudget {
	x, err := r.q().GetGlobalMonthlyBudget(ctx)
	if err != nil {
		return nil
	}
	return &MonthlyBudget{
		Amount: money.New(x.Amount, money.Unit{ID: x.CurrencyID, Decimals: int(x.Decimals)}),
		Rate:   x.Rate,
		IsBase: x.IsBase != 0,
	}
}

// MonthlyBudgets lists the active monthly budgets.
func (r Repo) MonthlyBudgets(ctx context.Context) ([]MonthlyBudget, error) {
	rows, err := r.q().ListMonthlyBudgets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MonthlyBudget, len(rows))
	for i, x := range rows {
		out[i] = MonthlyBudget{
			Amount: money.New(x.Amount, money.Unit{ID: x.CurrencyID, Decimals: int(x.Decimals)}),
			Rate:   x.Rate,
			IsBase: x.IsBase != 0,
		}
	}
	return out, nil
}

package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

// Account is an account as stored; every amount is in its Currency. A debt is
// an account too: Balance is then what is still owed and TargetAmount what
// was owed in total.
type Account struct {
	ID             int64
	Name           string
	Type           string
	CurrencyID     int64
	InitialBalance money.Money
	IsActive       bool
	SortOrder      int
	DebtType       *string
	TargetAmount   *money.Money
	DueDate        *string
	IsPaidOff      bool
	Counterparty   *string
	DebtDesc       *string
	CreatedAt      *time.Time
	Currency       *Currency
	Balance        money.Money
}

// AccountInput is what create and update take. Amounts arrive as decimals
// because the request names the currency by id; they are rounded into the
// account currency's scale when stored.
type AccountInput struct {
	Name           string
	Type           string
	CurrencyID     int64
	InitialBalance decimal.Decimal
	IsActive       bool
	DebtType       *string
	TargetAmount   *decimal.Decimal
	DueDate        *string
	IsPaidOff      bool
	Counterparty   *string
	DebtDesc       *string
}

// Input is the account as an update input, for read-modify-write.
func (a Account) Input() AccountInput {
	in := AccountInput{
		Name: a.Name, Type: a.Type, CurrencyID: a.CurrencyID, InitialBalance: a.InitialBalance.Decimal(),
		IsActive: a.IsActive, DebtType: a.DebtType, DueDate: a.DueDate, IsPaidOff: a.IsPaidOff,
		Counterparty: a.Counterparty, DebtDesc: a.DebtDesc,
	}
	if a.TargetAmount != nil {
		d := a.TargetAmount.Decimal()
		in.TargetAmount = &d
	}
	return in
}

var (
	// ErrAccountCurrencyImmutable is returned when an update changes the currency
	// of an account: its amounts are stored in minor units of that currency.
	ErrAccountCurrencyImmutable = errors.New("account currency cannot be changed")
	// ErrUnknownAccount is returned when an account referenced by id does not exist.
	ErrUnknownAccount = errors.New("unknown account")
	// ErrUnknownCurrency is returned when a currency referenced by id does not exist.
	ErrUnknownCurrency = errors.New("unknown currency")
)

type Accounts struct{ DB store.DB }

func (s Accounts) All(ctx context.Context, onlyActive, excludeDebts bool) ([]Account, error) {
	return s.list(ctx, sqlc.ListAccountsParams{
		OnlyActive:   db.Flag(onlyActive),
		ExcludeDebts: db.Flag(excludeDebts),
	})
}

func (s Accounts) Debts(ctx context.Context, includeCompleted bool) ([]Account, error) {
	return s.list(ctx, sqlc.ListAccountsParams{
		OnlyDebts:  db.Flag(true),
		UnpaidOnly: db.Flag(!includeCompleted),
	})
}

func (s Accounts) ByID(ctx context.Context, id int64) (*Account, error) {
	r, err := db.Q(s.DB).GetAccount(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a := accountFromRow(sqlc.ListAccountsRow(r))
	if a.Balance, err = s.balance(ctx, a, ""); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s Accounts) Create(ctx context.Context, a AccountInput) (*Account, error) {
	if a.Type == "" {
		a.Type = "cash"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	unit, err := s.currencyUnit(ctx, a.CurrencyID)
	if err != nil {
		return nil, err
	}
	initial, target, err := accountAmounts(a, unit)
	if err != nil {
		return nil, err
	}
	next, err := db.Q(s.DB).NextAccountSortOrder(ctx, db.Flag(a.Type == "debt"))
	if err != nil {
		return nil, err
	}
	res, err := db.Q(s.DB).InsertAccount(ctx, sqlc.InsertAccountParams{
		Name: a.Name, Type: a.Type, DebtType: db.NullString(a.DebtType), CurrencyID: a.CurrencyID,
		InitialBalance: initial.Minor(),
		TargetAmount:   money.ToNullMinor(target), DueDate: db.NullString(a.DueDate),
		IsPaidOff: db.BoolInt(a.IsPaidOff), Counterparty: db.NullString(a.Counterparty), DebtDescription: db.NullString(a.DebtDesc),
		IsActive: db.BoolInt(a.IsActive), SortOrder: next, CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Accounts) Update(ctx context.Context, id int64, a AccountInput) (*Account, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return nil, err
	}
	if a.CurrencyID != cur.CurrencyID {
		return nil, ErrAccountCurrencyImmutable
	}
	initial, target, err := accountAmounts(a, cur.Currency.Unit())
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).UpdateAccount(ctx, sqlc.UpdateAccountParams{
		Name: a.Name, Type: a.Type, CurrencyID: cur.CurrencyID, InitialBalance: initial.Minor(),
		IsActive: db.BoolInt(a.IsActive), DebtType: db.NullString(a.DebtType),
		TargetAmount: money.ToNullMinor(target),
		DueDate:      db.NullString(a.DueDate), IsPaidOff: db.BoolInt(a.IsPaidOff), Counterparty: db.NullString(a.Counterparty),
		DebtDescription: db.NullString(a.DebtDesc), UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// accountAmounts checks the amounts entered for an account against the range
// and rounds them into the account currency.
func accountAmounts(a AccountInput, unit money.Unit) (initial money.Money, target *money.Money, err error) {
	if initial, err = money.FromInput(a.InitialBalance, unit); err != nil {
		return money.Money{}, nil, err
	}
	target, err = money.FromNullInput(a.TargetAmount, unit)
	return initial, target, err
}

func (s Accounts) Delete(ctx context.Context, id int64) error {
	n, _ := db.Q(s.DB).CountAccountTransactions(ctx, sqlc.CountAccountTransactionsParams{AccountID: id, ToAccountID: db.NI(id)})
	if n > 0 {
		return fmt.Errorf("has transactions")
	}
	return db.Q(s.DB).DeleteAccount(ctx, id)
}

func (s Accounts) Reorder(ctx context.Context, ids []int64) error {
	for i, id := range ids {
		if err := db.Q(s.DB).SetAccountSortOrder(ctx, sqlc.SetAccountSortOrderParams{SortOrder: int64(i), ID: id}); err != nil {
			return err
		}
	}
	return nil
}

// AccountsSummary totals account balances in Currency (nil when there is
// none), rounded to its decimals.
type AccountsSummary struct {
	Total    money.Money
	Currency *Currency
	Count    int
}

func (s Accounts) Summary(ctx context.Context, base *Currency) AccountsSummary {
	accts, _ := s.All(ctx, true, true)
	out := AccountsSummary{Currency: base, Count: len(accts)}
	if base != nil {
		out.Total = s.total(ctx, accts, *base, "")
	}
	return out
}

// TotalAt is Summary's total as of the end of asOf (YYYY-MM-DD).
func (s Accounts) TotalAt(ctx context.Context, base Currency, asOf string) money.Money {
	accts, _ := s.All(ctx, true, true)
	return s.total(ctx, accts, base, asOf)
}

// total converts the balances of accts as of asOf ("" for now, which reuses
// the loaded balances) to base, each rounded to base's scale, and adds them.
func (s Accounts) total(ctx context.Context, accts []Account, base Currency, asOf string) money.Money {
	sum := money.Zero(base.Unit())
	for _, a := range accts {
		bal := a.Balance
		if asOf != "" {
			var err error
			if bal, err = s.balance(ctx, a, asOf); err != nil {
				return money.Zero(base.Unit())
			}
		}
		sum = sum.Add(Convert(bal, *a.Currency, base))
	}
	return sum
}

func (s Accounts) list(ctx context.Context, arg sqlc.ListAccountsParams) ([]Account, error) {
	rows, err := db.Q(s.DB).ListAccounts(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, r := range rows {
		out = append(out, accountFromRow(r))
	}
	for i := range out {
		if out[i].Balance, err = s.balance(ctx, out[i], ""); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// balance is the account's balance at the end of asOf ("" for now). A debt's
// balance is what is still owed; it does not depend on the date.
func (s Accounts) balance(ctx context.Context, a Account, asOf string) (money.Money, error) {
	unit := a.Currency.Unit()
	if a.Type == "debt" {
		paid, err := db.Q(s.DB).SumDebtPayments(ctx, db.NI(a.ID))
		if err != nil {
			return money.Money{}, err
		}
		target := money.Zero(unit)
		if a.TargetAmount != nil {
			target = *a.TargetAmount
		}
		return target.Sub(money.New(paid, unit)), nil
	}
	deltas, err := s.deltas(ctx, a, asOf)
	if err != nil {
		return money.Money{}, err
	}
	bal := a.InitialBalance
	for _, d := range deltas {
		bal = bal.Add(d.change)
	}
	return bal, nil
}

// BalanceSeries is the balance at the end of each of dates, which must be
// ascending YYYY-MM-DD days, from one query instead of one per date.
func (s Accounts) BalanceSeries(ctx context.Context, a Account, dates []string) ([]money.Money, error) {
	out := make([]money.Money, len(dates))
	if len(dates) == 0 {
		return out, nil
	}
	if a.Type == "debt" {
		bal, err := s.balance(ctx, a, "")
		for i := range out {
			out[i] = bal
		}
		return out, err
	}
	deltas, err := s.deltas(ctx, a, dates[len(dates)-1])
	if err != nil {
		return nil, err
	}
	bal, next := a.InitialBalance, 0
	for i, date := range dates {
		for next < len(deltas) && deltas[next].day <= date {
			bal = bal.Add(deltas[next].change)
			next++
		}
		out[i] = bal
	}
	return out, nil
}

type dayChange struct {
	day    string
	change money.Money
}

// deltas is the account's net change per day up to asOf, oldest first.
func (s Accounts) deltas(ctx context.Context, a Account, asOf string) ([]dayChange, error) {
	rows, err := db.Q(s.DB).AccountDailyDeltas(ctx, sqlc.AccountDailyDeltasParams{ID: a.ID, AsOf: db.Narg(asOf)})
	if err != nil {
		return nil, err
	}
	unit := a.Currency.Unit()
	out := make([]dayChange, len(rows))
	for i, r := range rows {
		out[i] = dayChange{day: r.Day.String, change: money.New(r.Delta, unit)}
	}
	return out, nil
}

func (s Accounts) currencyUnit(ctx context.Context, currencyID int64) (money.Unit, error) {
	c, err := db.Q(s.DB).GetCurrency(ctx, currencyID)
	if errors.Is(err, sql.ErrNoRows) {
		return money.Unit{}, ErrUnknownCurrency
	}
	if err != nil {
		return money.Unit{}, err
	}
	return money.Unit{ID: c.ID, Decimals: int(c.Decimals)}, nil
}

// Unit returns the unit of the account currency, the scale of every amount
// stored for that account.
func (s Accounts) Unit(ctx context.Context, accountID int64) (money.Unit, error) {
	u, err := db.Q(s.DB).GetAccountUnit(ctx, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return money.Unit{}, ErrUnknownAccount
	}
	if err != nil {
		return money.Unit{}, err
	}
	return money.Unit{ID: u.ID, Decimals: int(u.Decimals)}, nil
}

func accountFromRow(r sqlc.ListAccountsRow) Account {
	a := Account{
		ID: r.ID, Name: r.Name, Type: r.Type, CurrencyID: r.CurrencyID,
		IsActive: r.IsActive != 0, SortOrder: int(r.SortOrder),
		IsPaidOff: r.IsPaidOff != 0,
	}
	cur := Currency{ID: r.CurrencyIDJoin, Code: r.Code, Name: r.CurrencyName, Symbol: r.Symbol, Decimals: int(r.Decimals), IsBase: r.IsBase != 0, Rate: r.Rate}
	a.Currency = &cur
	a.InitialBalance = money.New(r.InitialBalance, cur.Unit())
	if r.DebtType.Valid {
		a.DebtType = &r.DebtType.String
	}
	a.TargetAmount = money.FromNullMinor(r.TargetAmount, cur.Unit())
	if r.DueDate.Valid {
		a.DueDate = &r.DueDate.String
	}
	if r.Counterparty.Valid {
		a.Counterparty = &r.Counterparty.String
	}
	if r.DebtDescription.Valid {
		a.DebtDesc = &r.DebtDescription.String
	}
	if t, ok := parseNullTime(r.CreatedAt); ok {
		a.CreatedAt = &t
	}
	return a
}

func parseNullTime(raw sql.NullString) (time.Time, bool) {
	if !raw.Valid || raw.String == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw.String, time.UTC); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func nilOr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

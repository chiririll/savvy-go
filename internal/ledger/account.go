package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/ledger"
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

func (s Accounts) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Accounts) All(ctx context.Context, onlyActive, excludeDebts bool) ([]Account, error) {
	return s.list(ctx, repo.AccountFilter{OnlyActive: onlyActive, ExcludeDebts: excludeDebts})
}

func (s Accounts) Debts(ctx context.Context, includeCompleted bool) ([]Account, error) {
	return s.list(ctx, repo.AccountFilter{OnlyDebts: true, UnpaidOnly: !includeCompleted})
}

func (s Accounts) ByID(ctx context.Context, id int64) (*Account, error) {
	row, err := s.repo().Account(ctx, id)
	if row == nil || err != nil {
		return nil, err
	}
	a := accountFrom(*row)
	if a.Balance, err = s.Balance(ctx, a, ""); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s Accounts) Create(ctx context.Context, a AccountInput) (*Account, error) {
	if a.Type == "" {
		a.Type = "cash"
	}
	unit, err := s.currencyUnit(ctx, a.CurrencyID)
	if err != nil {
		return nil, err
	}
	initial, target, err := accountAmounts(a, unit)
	if err != nil {
		return nil, err
	}
	next, err := s.repo().NextAccountSortOrder(ctx, a.Type == "debt")
	if err != nil {
		return nil, err
	}
	id, err := s.repo().InsertAccount(ctx, repo.AccountFields{
		Name: a.Name, Type: a.Type, DebtType: a.DebtType, CurrencyID: a.CurrencyID, InitialBalance: initial,
		TargetAmount: target, DueDate: a.DueDate, IsPaidOff: a.IsPaidOff, Counterparty: a.Counterparty,
		DebtDesc: a.DebtDesc, IsActive: a.IsActive, SortOrder: next,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
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
	err = s.repo().UpdateAccount(ctx, id, repo.AccountFields{
		Name: a.Name, Type: a.Type, CurrencyID: cur.CurrencyID, InitialBalance: initial,
		IsActive: a.IsActive, DebtType: a.DebtType, TargetAmount: target, DueDate: a.DueDate,
		IsPaidOff: a.IsPaidOff, Counterparty: a.Counterparty, DebtDesc: a.DebtDesc,
	}, time.Now().UTC())
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
	if s.repo().CountAccountTransactions(ctx, id) > 0 {
		return fmt.Errorf("has transactions")
	}
	return s.repo().DeleteAccount(ctx, id)
}

func (s Accounts) Reorder(ctx context.Context, ids []int64) error {
	for i, id := range ids {
		if err := s.repo().SetAccountSortOrder(ctx, id, i); err != nil {
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
			if bal, err = s.Balance(ctx, a, asOf); err != nil {
				return money.Zero(base.Unit())
			}
		}
		sum = sum.Add(Convert(bal, *a.Currency, base))
	}
	return sum
}

func (s Accounts) list(ctx context.Context, f repo.AccountFilter) ([]Account, error) {
	rows, err := s.repo().Accounts(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]Account, len(rows))
	for i, r := range rows {
		out[i] = accountFrom(r)
		if out[i].Balance, err = s.Balance(ctx, out[i], ""); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Balance is the account's balance at the end of asOf ("" for now). A debt's
// balance is what is still owed; it does not depend on the date.
func (s Accounts) Balance(ctx context.Context, a Account, asOf string) (money.Money, error) {
	unit := a.Currency.Unit()
	if a.Type == "debt" {
		paid, err := s.repo().DebtPaid(ctx, a.ID)
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
		bal, err := s.Balance(ctx, a, "")
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
	rows, err := s.repo().DailyChanges(ctx, a.ID, asOf)
	if err != nil {
		return nil, err
	}
	unit := a.Currency.Unit()
	out := make([]dayChange, len(rows))
	for i, r := range rows {
		out[i] = dayChange{day: r.Day, change: money.New(r.Delta, unit)}
	}
	return out, nil
}

func (s Accounts) currencyUnit(ctx context.Context, currencyID int64) (money.Unit, error) {
	u, err := s.repo().CurrencyUnit(ctx, currencyID)
	if err != nil {
		return money.Unit{}, err
	}
	if u == nil {
		return money.Unit{}, ErrUnknownCurrency
	}
	return *u, nil
}

// Unit returns the unit of the account currency, the scale of every amount
// stored for that account.
func (s Accounts) Unit(ctx context.Context, accountID int64) (money.Unit, error) {
	u, err := s.repo().AccountUnit(ctx, accountID)
	if err != nil {
		return money.Unit{}, err
	}
	if u == nil {
		return money.Unit{}, ErrUnknownAccount
	}
	return *u, nil
}

func accountFrom(r repo.Account) Account {
	cur := currencyFrom(r.Currency)
	return Account{
		ID: r.ID, Name: r.Name, Type: r.Type, CurrencyID: r.Currency.ID, InitialBalance: r.InitialBalance,
		IsActive: r.IsActive, SortOrder: r.SortOrder, DebtType: r.DebtType, TargetAmount: r.TargetAmount,
		DueDate: r.DueDate, IsPaidOff: r.IsPaidOff, Counterparty: r.Counterparty, DebtDesc: r.DebtDesc,
		CreatedAt: r.CreatedAt, Currency: &cur,
	}
}

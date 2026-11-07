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

// Account is an account as stored; every amount is in its Currency and Balance
// is signed. A debt is an account too (type debt, a service account), but its
// terms are a Debt and are written only through Debts.
type Account struct {
	ID             int64
	Name           string
	Type           string
	IsService      bool
	CurrencyID     int64
	InitialBalance money.Money
	IsActive       bool
	SortOrder      int
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
}

// Input is the account as an update input, for read-modify-write.
func (a Account) Input() AccountInput {
	return AccountInput{
		Name: a.Name, Type: a.Type, CurrencyID: a.CurrencyID, InitialBalance: a.InitialBalance.Decimal(),
		IsActive: a.IsActive,
	}
}

var (
	// ErrAccountCurrencyImmutable is returned when an update changes the currency
	// of an account: its amounts are stored in minor units of that currency.
	ErrAccountCurrencyImmutable = errors.New("account currency cannot be changed")
	// ErrUnknownAccount is returned when an account referenced by id does not exist.
	ErrUnknownAccount = errors.New("unknown account")
	// ErrUnknownCurrency is returned when a currency referenced by id does not exist.
	ErrUnknownCurrency = errors.New("unknown currency")
	// ErrServiceAccount is returned when an account of type debt or space is
	// written through Accounts: debts belong to Debts, and a space account is
	// made by SpaceAccount.
	ErrServiceAccount = errors.New("a debt or space account cannot be changed as an account")
	// ErrCardAccount is returned when a credit card is written through Accounts:
	// it belongs to CreditCards, which keeps its terms with the account.
	ErrCardAccount = errors.New("a credit card cannot be changed as an account")
)

type Accounts struct{ DB store.DB }

func (s Accounts) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// All lists the accounts. excludeService leaves out debts and space accounts,
// which is what a picker and net worth want.
func (s Accounts) All(ctx context.Context, onlyActive, excludeService bool) ([]Account, error) {
	rows, err := s.repo().Accounts(ctx, repo.AccountFilter{OnlyActive: onlyActive, ExcludeService: excludeService})
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

// ForNetWorth lists the active accounts that make up net worth: the ordinary
// accounts and the debts (an asset when owed to the user, a liability when the
// user owes), not the space accounts, which stand for money that left the
// space.
func (s Accounts) ForNetWorth(ctx context.Context) ([]Account, error) {
	rows, err := s.repo().Accounts(ctx, repo.AccountFilter{OnlyActive: true, ExcludeSpace: true})
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

func (s Accounts) ByID(ctx context.Context, id int64) (*Account, error) {
	a, err := s.Bare(ctx, id)
	if a == nil || err != nil {
		return nil, err
	}
	if a.Balance, err = s.Balance(ctx, *a, ""); err != nil {
		return nil, err
	}
	return a, nil
}

// Bare is ByID without the balance, which costs a pass over every payment of
// the account. Use it when only the account itself is needed.
func (s Accounts) Bare(ctx context.Context, id int64) (*Account, error) {
	row, err := s.repo().Account(ctx, id)
	if row == nil || err != nil {
		return nil, err
	}
	a := accountFrom(*row)
	return &a, nil
}

// SpaceAccount is the id of the account that stands for all other spaces in
// one currency. It is made on first use, so a space without linked spaces has
// none.
func (s Accounts) SpaceAccount(ctx context.Context, currencyID int64) (int64, error) {
	existing, err := s.repo().SpaceAccountID(ctx, currencyID)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return *existing, nil
	}
	cur, err := s.repo().Currency(ctx, currencyID)
	if err != nil {
		return 0, err
	}
	if cur == nil {
		return 0, ErrUnknownCurrency
	}
	id, err := s.repo().InsertAccount(ctx, repo.AccountFields{
		Name: "Other spaces · " + cur.Code, Type: "space", CurrencyID: currencyID, IsActive: true,
	}, time.Now().UTC())
	if err != nil {
		// Another writer made it first: the unique index refused a second one.
		if again, gerr := s.repo().SpaceAccountID(ctx, currencyID); gerr == nil && again != nil {
			return *again, nil
		}
		return 0, err
	}
	return id, nil
}

func (s Accounts) Create(ctx context.Context, a AccountInput) (*Account, error) {
	if a.Type == "" {
		a.Type = "cash"
	}
	if isServiceType(a.Type) {
		return nil, ErrServiceAccount
	}
	if a.Type == TypeCredit {
		return nil, ErrCardAccount
	}
	unit, err := s.currencyUnit(ctx, a.CurrencyID)
	if err != nil {
		return nil, err
	}
	initial, err := money.FromInput(a.InitialBalance, unit)
	if err != nil {
		return nil, err
	}
	var id int64
	err = store.Tx(ctx, s.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		next, err := r.NextAccountSortOrder(ctx, false)
		if err != nil {
			return err
		}
		id, err = r.InsertAccount(ctx, repo.AccountFields{
			Name: a.Name, Type: a.Type, CurrencyID: a.CurrencyID, InitialBalance: initial,
			IsActive: a.IsActive, SortOrder: next,
		}, time.Now().UTC())
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// Update saves an account. A debt or space account is refused, and so is a
// change of type to one.
func (s Accounts) Update(ctx context.Context, id int64, a AccountInput) (*Account, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return nil, err
	}
	if cur.IsService || isServiceType(a.Type) {
		return nil, ErrServiceAccount
	}
	if cur.Type == TypeCredit || a.Type == TypeCredit {
		return nil, ErrCardAccount
	}
	if a.CurrencyID != cur.CurrencyID {
		return nil, ErrAccountCurrencyImmutable
	}
	initial, err := money.FromInput(a.InitialBalance, cur.Currency.Unit())
	if err != nil {
		return nil, err
	}
	err = s.repo().UpdateAccount(ctx, id, repo.AccountFields{
		Name: a.Name, Type: a.Type, CurrencyID: cur.CurrencyID, InitialBalance: initial, IsActive: a.IsActive,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func isServiceType(typ string) bool { return typ == "debt" || typ == "space" }

// Delete removes an account that has no payments. A debt is deleted by Debts.
func (s Accounts) Delete(ctx context.Context, id int64) error {
	if a, err := s.ByID(ctx, id); err == nil && a != nil && a.IsService {
		return ErrServiceAccount
	} else if err == nil && a != nil && a.Type == TypeCredit {
		return ErrCardAccount
	}
	if s.repo().CountAccountPayments(ctx, id) > 0 {
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

// Balance is the account's balance at the end of asOf ("" for now): the opening
// balance plus its payments, whatever the account is.
func (s Accounts) Balance(ctx context.Context, a Account, asOf string) (money.Money, error) {
	sum, err := s.repo().DeltaSum(ctx, a.ID, asOf)
	if err != nil {
		return money.Money{}, err
	}
	return a.InitialBalance.Add(money.New(sum, a.Currency.Unit())), nil
}

// BalanceSeries is the balance at the end of each of dates, which must be
// ascending YYYY-MM-DD days, from one query instead of one per date.
func (s Accounts) BalanceSeries(ctx context.Context, a Account, dates []string) ([]money.Money, error) {
	out := make([]money.Money, len(dates))
	if len(dates) == 0 {
		return out, nil
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
		ID: r.ID, Name: r.Name, Type: r.Type, IsService: r.IsService, CurrencyID: r.Currency.ID,
		InitialBalance: r.InitialBalance, IsActive: r.IsActive, SortOrder: r.SortOrder, CreatedAt: r.CreatedAt,
		Currency: &cur,
	}
}

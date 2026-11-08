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

// The two directions of a debt.
const (
	DebtIOwe     = "i_owe"
	DebtOwedToMe = "owed_to_me"
)

var (
	// ErrDebtPrincipal is returned for a debt without a positive amount.
	ErrDebtPrincipal = errors.New("a debt needs an amount above zero")
	// ErrDebtDirection is returned for a debt that is neither i_owe nor owed_to_me.
	ErrDebtDirection = errors.New("debt type must be i_owe or owed_to_me")
	// ErrDebtDirectionLocked is returned when the direction of a debt with
	// payments is changed: its payments would then run the wrong way.
	ErrDebtDirectionLocked = errors.New("the direction of a debt with payments cannot be changed")
)

// Debt is a debt account with its terms. Principal and the account's Balance
// are in the account currency. Balance is signed, positive when owed to the
// user and negative when the user owes; Owed is what is still to be paid.
type Debt struct {
	Account
	Direction    string // DebtIOwe or DebtOwedToMe
	Principal    money.Money
	DueDate      *string
	Counterparty *string
	Note         *string
	ClosedAt     *time.Time
}

// IsPaidOff reports whether the debt was settled and closed.
func (d Debt) IsPaidOff() bool { return d.ClosedAt != nil }

// Owed is what is still to be paid, whichever way the debt runs: the balance as
// a positive number, negative once the debt is overpaid.
func (d Debt) Owed() money.Money {
	if d.Direction == DebtIOwe {
		return d.Balance.Neg()
	}
	return d.Balance
}

// PaymentProgress is the share of the principal already paid, as a percentage
// rounded to one decimal and kept within 0..100: an overpaid debt is 100, a
// debt that has grown beyond its principal is 0.
func (d Debt) PaymentProgress() float64 {
	if !d.Principal.IsPositive() {
		return 0
	}
	paid := d.Principal.Sub(d.Owed())
	pct := paid.Decimal().Div(d.Principal.Decimal()).Mul(decimal.NewFromInt(100)).Round(1)
	return min(100, max(0, pct.InexactFloat64()))
}

// Edit is the debt as an update input, for read-modify-write.
func (d Debt) Edit() DebtEdit {
	return DebtEdit{
		Name: d.Name, Direction: d.Direction, CurrencyID: d.CurrencyID, Principal: d.Principal.Decimal(),
		DueDate: d.DueDate, Counterparty: d.Counterparty, Note: d.Note,
	}
}

// DebtEdit is what Debts.Update takes. Principal is rounded into the account
// currency's scale when stored.
type DebtEdit struct {
	Name         string
	Direction    string
	CurrencyID   int64
	Principal    decimal.Decimal
	DueDate      *string
	Counterparty *string
	Note         *string
}

type Debts struct {
	Accounts     Accounts
	Transactions Transactions
}

func (s Debts) repo() repo.Repo { return repo.Repo{DB: s.Accounts.DB} }

// All lists the debts, the open ones only unless includeCompleted.
func (s Debts) All(ctx context.Context, includeCompleted bool) ([]Debt, error) {
	rows, err := s.repo().Debts(ctx, !includeCompleted)
	if err != nil {
		return nil, err
	}
	out := make([]Debt, len(rows))
	for i, row := range rows {
		if out[i], err = s.debt(ctx, row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ByID returns nil when the id is not a debt.
func (s Debts) ByID(ctx context.Context, id int64) (*Debt, error) {
	row, err := s.repo().Debt(ctx, id)
	if row == nil || err != nil {
		return nil, err
	}
	d, err := s.debt(ctx, *row)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s Debts) debt(ctx context.Context, row repo.Debt) (Debt, error) {
	a := accountFrom(row.Account)
	bal, err := s.Accounts.Balance(ctx, a, "")
	if err != nil {
		return Debt{}, err
	}
	a.Balance = bal
	return Debt{
		Account: a, Direction: row.Direction, Principal: row.Principal, DueDate: row.DueDate,
		Counterparty: row.Counterparty, Note: row.Note, ClosedAt: row.ClosedAt,
	}, nil
}

// DebtInput describes a new debt. Origin "new" with an AccountID records the
// issuance as a transaction on that account and takes its currency; otherwise
// the debt starts from its amount as an opening balance.
type DebtInput struct {
	Name         string
	Direction    string
	CurrencyID   int64
	AccountID    int64
	Amount       decimal.Decimal
	Date         string
	Origin       string
	DueDate      *string
	Counterparty *string
	Description  *string
}

func (s Debts) Create(ctx context.Context, in DebtInput) (*Debt, error) {
	currencyID := in.CurrencyID
	issued := in.Origin == "new" && in.AccountID != 0
	if issued {
		src, err := s.Accounts.ByID(ctx, in.AccountID)
		if err != nil || src == nil {
			return nil, fmt.Errorf("source account")
		}
		if src.IsService {
			return nil, fmt.Errorf("cannot use debt source")
		}
		currencyID = src.CurrencyID
	}
	unit, err := s.Accounts.currencyUnit(ctx, currencyID)
	if err != nil {
		return nil, err
	}
	fields, err := debtFields(in.Direction, in.Amount, unit, in.DueDate, in.Counterparty, in.Description)
	if err != nil {
		return nil, err
	}
	opening := money.Zero(unit)
	if !issued {
		opening = signedPrincipal(fields.Direction, fields.Principal)
	}
	var id int64
	err = store.Tx(ctx, s.Accounts.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		next, err := r.NextAccountSortOrder(ctx, true)
		if err != nil {
			return err
		}
		if id, err = r.InsertAccount(ctx, repo.AccountFields{
			Name: in.Name, Type: "debt", CurrencyID: currencyID, InitialBalance: opening, IsActive: true, SortOrder: next,
		}, time.Now().UTC()); err != nil {
			return err
		}
		return r.InsertDebt(ctx, id, fields)
	})
	if err != nil {
		return nil, err
	}
	if issued {
		if err := s.recordIssuance(ctx, id, in); err != nil {
			_ = s.repo().DeleteDebt(ctx, id) // no debt without the issuance it was made for
			return nil, err
		}
	}
	return s.ByID(ctx, id)
}

// debtFields checks the terms of a debt and rounds the principal into unit.
func debtFields(
	direction string,
	amount decimal.Decimal,
	unit money.Unit,
	due, counterparty, note *string,
) (repo.DebtFields, error) {
	if direction != DebtIOwe && direction != DebtOwedToMe {
		return repo.DebtFields{}, ErrDebtDirection
	}
	principal, err := money.FromInput(amount, unit)
	if err != nil {
		return repo.DebtFields{}, err
	}
	if !principal.IsPositive() {
		return repo.DebtFields{}, ErrDebtPrincipal
	}
	return repo.DebtFields{
		Direction: direction, Principal: principal, DueDate: due, Counterparty: counterparty, Note: note,
	}, nil
}

// signedPrincipal is a principal as a balance: positive when owed to the user.
func signedPrincipal(direction string, principal money.Money) money.Money {
	if direction == DebtIOwe {
		return principal.Neg()
	}
	return principal
}

func (s Debts) recordIssuance(ctx context.Context, debtID int64, in DebtInput) error {
	typ := "debt_lend"
	if in.Direction == DebtIOwe {
		typ = "debt_borrow"
	}
	_, err := s.record(ctx, typ, debtID, in.AccountID, in.Amount, in.Date, in.Description)
	return err
}

// record writes a debt transaction: two payments, one on the debt account in
// its currency (amount) and one on the ordinary account in its own, converted
// at the current rates when the currencies differ. The debt account is signed
// like any account: lending and paying add to it, borrowing and collecting take
// from it, and the ordinary account moves the other way.
func (s Debts) record(
	ctx context.Context,
	typ string,
	debtID, accountID int64,
	amount decimal.Decimal,
	date string,
	desc *string,
) (*Transaction, error) {
	debt, err := s.Accounts.ByID(ctx, debtID)
	if err != nil || debt == nil || debt.Type != "debt" {
		return nil, ErrUnknownAccount
	}
	src, err := s.Accounts.ByID(ctx, accountID)
	if err != nil || src == nil {
		return nil, ErrUnknownAccount
	}
	onDebt, err := money.FromInput(amount, debt.Currency.Unit())
	if err != nil {
		return nil, err
	}
	if !onDebt.IsPositive() {
		return nil, ErrDebtPrincipal
	}
	onSource, err := TryConvert(onDebt, *debt.Currency, *src.Currency)
	if err != nil {
		return nil, err
	}
	if typ == "debt_lend" || typ == "debt_payment" {
		onSource = onSource.Neg() // the debt account grows, the ordinary one gives
	} else {
		onDebt = onDebt.Neg()
	}
	debtAmount, sourceAmount := onDebt.Decimal(), onSource.Decimal()
	return s.Transactions.As(ReasonDebt).Create(ctx, TxInput{
		Type: typ, Description: desc, Date: &date,
		Payments: []PaymentInput{
			{AccountID: accountID, Amount: &sourceAmount},
			{AccountID: debtID, Amount: &debtAmount},
		},
	})
}

// Update saves the terms of a debt. A new principal moves the opening balance
// with it, so that what is owed stays the principal less what was paid. The
// direction can change only while nothing was paid, which makes the debt as new.
func (s Debts) Update(ctx context.Context, id int64, e DebtEdit) (*Debt, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return nil, err
	}
	if e.CurrencyID != cur.CurrencyID {
		return nil, ErrAccountCurrencyImmutable
	}
	fields, err := debtFields(e.Direction, e.Principal, cur.Currency.Unit(), e.DueDate, e.Counterparty, e.Note)
	if err != nil {
		return nil, err
	}
	err = store.Tx(ctx, s.Accounts.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		opening, err := movedOpening(ctx, r, *cur, fields)
		if err != nil {
			return err
		}
		if err := r.UpdateAccount(ctx, id, repo.AccountFields{
			Name: e.Name, Type: "debt", CurrencyID: cur.CurrencyID, InitialBalance: opening, IsActive: cur.IsActive,
		}, time.Now().UTC()); err != nil {
			return err
		}
		return r.UpdateDebt(ctx, id, fields)
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// movedOpening is the opening balance of a debt after its terms changed.
func movedOpening(ctx context.Context, r repo.Repo, cur Debt, next repo.DebtFields) (money.Money, error) {
	if next.Direction != cur.Direction {
		if r.CountAccountPayments(ctx, cur.ID) > 0 {
			return money.Money{}, ErrDebtDirectionLocked
		}
		return signedPrincipal(next.Direction, next.Principal), nil
	}
	moved := signedPrincipal(next.Direction, next.Principal.Sub(cur.Principal))
	return cur.InitialBalance.Add(moved), nil
}

func (s Debts) Payment(
	ctx context.Context,
	debtID, accountID int64,
	amount decimal.Decimal,
	date string,
	desc *string,
	collect bool,
) (*Transaction, error) {
	debt, err := s.ByID(ctx, debtID)
	if err != nil || debt == nil {
		return nil, fmt.Errorf("not a debt")
	}
	if debt.IsPaidOff() {
		return nil, fmt.Errorf("already paid off")
	}
	typ := "debt_payment"
	if collect {
		typ = "debt_collection"
	}
	tx, err := s.record(ctx, typ, debt.ID, accountID, amount, date, desc)
	if err != nil {
		return nil, err
	}
	s.maybePayOff(ctx, debt.ID)
	return tx, nil
}

// maybePayOff closes a debt that has nothing left to pay.
func (s Debts) maybePayOff(ctx context.Context, id int64) {
	fresh, err := s.ByID(ctx, id)
	if err != nil || fresh == nil {
		return
	}
	if !fresh.Owed().IsPositive() {
		_ = s.repo().CloseDebt(ctx, id, time.Now().UTC())
	}
}

func (s Debts) Reopen(ctx context.Context, id int64) (*Debt, error) {
	d, err := s.ByID(ctx, id)
	if err != nil || d == nil {
		return nil, fmt.Errorf("not a debt")
	}
	if !d.IsPaidOff() {
		return nil, fmt.Errorf("not paid off")
	}
	if err := s.repo().ReopenDebt(ctx, id); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// DebtSummary totals open debts in the base currency (nil when there is none),
// rounded to its decimals.
type DebtSummary struct {
	IOwe     money.Money
	OwedToMe money.Money
	Count    int
	Currency *Currency
}

// Net is what is owed to the user less what the user owes.
func (s DebtSummary) Net() money.Money { return s.OwedToMe.Sub(s.IOwe) }

func (s Debts) Summary(ctx context.Context) DebtSummary {
	debts, _ := s.All(ctx, false)
	base, _ := (Currencies{DB: s.Accounts.DB}).Base(ctx)
	out := DebtSummary{Count: len(debts), Currency: base}
	if base == nil {
		return out
	}
	out.IOwe, out.OwedToMe = money.Zero(base.Unit()), money.Zero(base.Unit())
	for _, d := range debts {
		amt := Convert(d.Owed(), *d.Currency, *base)
		if d.Direction == DebtIOwe {
			out.IOwe = out.IOwe.Add(amt)
		} else {
			out.OwedToMe = out.OwedToMe.Add(amt)
		}
	}
	return out
}

func (s Debts) Delete(ctx context.Context, id int64) error {
	if s.repo().CountDebtHistory(ctx, id) > 0 {
		return fmt.Errorf("has history")
	}
	return s.repo().DeleteDebt(ctx, id)
}

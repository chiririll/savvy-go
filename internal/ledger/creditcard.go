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

// TypeCredit is the type of the account of a credit card.
const TypeCredit = "credit"

// CreditCard is the billing terms of a credit card.
type CreditCard = repo.CreditCard

// The two ways a billing period can run. Only CycleFixedDay is supported;
// CycleFirstExpense is in the schema for later and refused.
const (
	CycleFixedDay     = repo.CycleFixedDay
	CycleFirstExpense = repo.CycleFirstExpense
)

// MaxGraceDays is the longest grace period accepted. It keeps a due date within
// a few statements of today, which is how many CardState looks back over.
const MaxGraceDays = 60

// How many of the last statements decide the status: with a grace period of at
// most MaxGraceDays and statements at least 28 days apart, an older statement
// is past due for good.
const statementsChecked = 3

// The status of a card.
const (
	StatusOK      = "ok"
	StatusWarning = "warning" // a statement is waiting for its payment
	StatusDanger  = "danger"  // a payment is overdue, or the limit is exceeded

	ReasonPaymentDue = "payment_due"
	ReasonOverdue    = "overdue"
	ReasonOverLimit  = "over_limit"
)

var (
	ErrCardLimit = errors.New("a credit card needs a limit above zero")
	ErrCardDay   = errors.New("the statement day must be from 1 to 31")
	ErrCardGrace = errors.New("the grace period must be from 0 to 60 days")
	ErrCardCycle = errors.New("the billing period must close on a fixed day of the month")
	// ErrCardNotConvertible is returned for an account that cannot become a card:
	// only an ordinary bank account can.
	ErrCardNotConvertible = errors.New("only a bank account can become a credit card")
)

// ValidateCard checks the terms of a card.
func ValidateCard(c CreditCard) error {
	switch {
	case !c.Limit.IsPositive():
		return ErrCardLimit
	case c.GraceDays < 0 || c.GraceDays > MaxGraceDays:
		return ErrCardGrace
	case c.Cycle != CycleFixedDay || c.PeriodDays != nil:
		return ErrCardCycle
	case c.StatementDay == nil || *c.StatementDay < 1 || *c.StatementDay > 31:
		return ErrCardDay
	}
	return nil
}

// Statement is the last closed statement of a card. Balance is the debt at the
// end of the statement day (0 when the balance was not negative), Paid what
// came into the card after it, and AmountDue what is left, never below zero.
type Statement struct {
	Date      string // YYYY-MM-DD
	Balance   money.Money
	DueDate   string
	Paid      money.Money
	AmountDue money.Money
}

// CardStatus is what the UI shows as a triangle: Level ok, warning or danger,
// with the Reason, and the Amount and Date it is about (the amount to pay and
// its due date, or by how much the limit is exceeded and no date).
type CardStatus struct {
	Level  string
	Reason string
	Amount money.Money
	Date   string
}

// CardState is derived from the card's payments, never stored, so a payment
// entered later with a past date changes it.
type CardState struct {
	Used        money.Money // the debt, 0 when the balance is not negative
	Available   money.Money // limit + balance, below zero over the limit
	PeriodStart string      // the first day after the last statement
	PeriodEnd   string      // the next statement day
	Spent       money.Money // what went out of the card in the period so far
	Statement   Statement
	Status      CardStatus
}

// ComputeCardState reads the card as of today. initial is the account's opening
// balance and flows its confirmed payments per day, oldest first. Days after
// today are not counted. The card must be valid.
func ComputeCardState(c CreditCard, initial money.Money, flows []repo.DayFlow, today time.Time) CardState {
	unit := initial.Unit()
	todayS := dayString(today)
	book := newCardBook(initial, flows, todayS)

	out := CardState{}
	balance := book.balanceAt(todayS)
	out.Used = money.Max(balance.Neg(), money.Zero(unit))
	out.Available = c.Limit.Add(balance)

	last := lastStatement(*c.StatementDay, today)
	next := nextStatement(*c.StatementDay, last)
	out.PeriodStart = dayString(last.AddDate(0, 0, 1))
	out.PeriodEnd = dayString(next)
	out.Spent = book.outflowBetween(out.PeriodStart, todayS).Neg()

	var overdue, due *CardStatus
	stmt := last
	for i := 0; i < statementsChecked; i++ {
		s := book.statement(stmt, c.GraceDays, todayS)
		if i == 0 {
			out.Statement = s
		}
		if s.AmountDue.IsPositive() {
			st := CardStatus{Level: StatusWarning, Reason: ReasonPaymentDue, Amount: s.AmountDue, Date: s.DueDate}
			if todayS > s.DueDate {
				st.Level, st.Reason = StatusDanger, ReasonOverdue
				overdue = &st // the loop goes back in time, so the oldest ends up here
			} else if due == nil || s.DueDate < due.Date {
				due = &st // the nearest due date
			}
		}
		stmt = previousStatement(*c.StatementDay, stmt)
	}

	switch {
	case overdue != nil:
		out.Status = *overdue
	case out.Available.IsNegative():
		out.Status = CardStatus{Level: StatusDanger, Reason: ReasonOverLimit, Amount: out.Available.Neg()}
	case due != nil:
		out.Status = *due
	default:
		out.Status = CardStatus{Level: StatusOK, Amount: money.Zero(unit)}
	}
	return out
}

// cardBook is a card's payments up to today, for the questions a statement asks.
type cardBook struct {
	initial money.Money
	flows   []repo.DayFlow // up to today
}

func newCardBook(initial money.Money, flows []repo.DayFlow, today string) cardBook {
	b := cardBook{initial: initial}
	for _, f := range flows {
		if f.Day <= today {
			b.flows = append(b.flows, f)
		}
	}
	return b
}

func (b cardBook) balanceAt(day string) money.Money {
	bal := b.initial
	for _, f := range b.flows {
		if f.Day <= day {
			bal = bal.Add(money.New(f.Delta, b.initial.Unit()))
		}
	}
	return bal
}

// inflowBetween adds what came in on the days after from up to and including to.
func (b cardBook) inflowBetween(from, to string) money.Money {
	sum := money.Zero(b.initial.Unit())
	for _, f := range b.flows {
		if f.Day > from && f.Day <= to {
			sum = sum.Add(money.New(f.Inflow, b.initial.Unit()))
		}
	}
	return sum
}

// outflowBetween adds (a negative amount) what went out on the days from from
// to to, both included.
func (b cardBook) outflowBetween(from, to string) money.Money {
	sum := money.Zero(b.initial.Unit())
	for _, f := range b.flows {
		if f.Day >= from && f.Day <= to {
			sum = sum.Add(money.New(f.Outflow, b.initial.Unit()))
		}
	}
	return sum
}

func (b cardBook) statement(date time.Time, grace int, today string) Statement {
	unit := b.initial.Unit()
	day := dayString(date)
	bal := money.Max(b.balanceAt(day).Neg(), money.Zero(unit))
	paid := b.inflowBetween(day, today)
	return Statement{
		Date: day, Balance: bal, DueDate: dayString(date.AddDate(0, 0, grace)), Paid: paid,
		AmountDue: money.Max(bal.Sub(paid), money.Zero(unit)),
	}
}

// statementOn is the statement day of a month: day, or the month's last day
// when it has fewer.
func statementOn(day, year int, month time.Month) time.Time {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(year, month, min(day, last), 0, 0, 0, 0, time.UTC)
}

// lastStatement is the latest statement day before today: a period closes at
// the end of its statement day, so on that day it is still open.
func lastStatement(day int, today time.Time) time.Time {
	s := statementOn(day, today.Year(), today.Month())
	if !s.Before(dateOnly(today)) {
		return previousStatement(day, s)
	}
	return s
}

func previousStatement(day int, s time.Time) time.Time {
	first := time.Date(s.Year(), s.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	return statementOn(day, first.Year(), first.Month())
}

func nextStatement(day int, s time.Time) time.Time {
	first := time.Date(s.Year(), s.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return statementOn(day, first.Year(), first.Month())
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func dayString(t time.Time) string { return t.Format("2006-01-02") }

// Card is a credit card: an account of type credit with its terms. Like a Debt
// it is read and written only through its service, CreditCards; Accounts refuses
// it. The balance is signed like any account's, negative when money is owed.
type Card struct {
	Account
	Terms CreditCard
}

// CardInput is the terms of a card as the request gives them. Cycle defaults to
// fixed_day, the only one supported.
type CardInput struct {
	Limit        decimal.Decimal
	Cycle        string
	StatementDay int
	GraceDays    int
}

// NewCard describes a new card and its account.
type NewCard struct {
	Name           string
	CurrencyID     int64
	InitialBalance decimal.Decimal
	IsActive       bool
	Terms          CardInput
}

// CardEdit is what CreditCards.Update takes. The currency does not change.
type CardEdit struct {
	Name           string
	InitialBalance decimal.Decimal
	IsActive       bool
	Terms          CardInput
}

// Edit is the card as an update input, for read-modify-write.
func (c Card) Edit() CardEdit {
	t := CardInput{Limit: c.Terms.Limit.Decimal(), Cycle: c.Terms.Cycle, GraceDays: c.Terms.GraceDays}
	if c.Terms.StatementDay != nil {
		t.StatementDay = *c.Terms.StatementDay
	}
	return CardEdit{Name: c.Name, InitialBalance: c.InitialBalance.Decimal(), IsActive: c.IsActive, Terms: t}
}

// termsFor validates the terms of a card and puts the limit into unit.
func termsFor(in CardInput, unit money.Unit) (CreditCard, error) {
	limit, err := money.FromInput(in.Limit, unit)
	if err != nil {
		return CreditCard{}, err
	}
	cycle := in.Cycle
	if cycle == "" {
		cycle = CycleFixedDay
	}
	day := in.StatementDay
	c := CreditCard{Limit: limit, Cycle: cycle, StatementDay: &day, GraceDays: in.GraceDays}
	if err := ValidateCard(c); err != nil {
		return CreditCard{}, err
	}
	return c, nil
}

// CreditCards reads and writes credit cards.
type CreditCards struct{ Accounts Accounts }

func (s CreditCards) repo() repo.Repo { return repo.Repo{DB: s.Accounts.DB} }

// All lists the cards, the active ones only when onlyActive.
func (s CreditCards) All(ctx context.Context, onlyActive bool) ([]Card, error) {
	rows, err := s.repo().CreditCards(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]Card, len(rows))
	for i, row := range rows {
		if out[i], err = s.card(ctx, row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ByID returns nil when the id is not a credit card.
func (s CreditCards) ByID(ctx context.Context, id int64) (*Card, error) {
	row, err := s.repo().CreditCard(ctx, id)
	if row == nil || err != nil {
		return nil, err
	}
	c, err := s.card(ctx, *row)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s CreditCards) card(ctx context.Context, row repo.CardAccount) (Card, error) {
	a := accountFrom(row.Account)
	bal, err := s.Accounts.Balance(ctx, a, "")
	if err != nil {
		return Card{}, err
	}
	a.Balance = bal
	return Card{Account: a, Terms: row.Terms}, nil
}

// Create makes a card: an account of type credit with its terms, in one step.
func (s CreditCards) Create(ctx context.Context, in NewCard) (*Card, error) {
	unit, err := s.Accounts.currencyUnit(ctx, in.CurrencyID)
	if err != nil {
		return nil, err
	}
	initial, err := money.FromInput(in.InitialBalance, unit)
	if err != nil {
		return nil, err
	}
	terms, err := termsFor(in.Terms, unit)
	if err != nil {
		return nil, err
	}
	var id int64
	err = store.Tx(ctx, s.Accounts.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		next, err := r.NextAccountSortOrder(ctx, false)
		if err != nil {
			return err
		}
		if id, err = r.InsertAccount(ctx, repo.AccountFields{
			Name: in.Name, Type: TypeCredit, CurrencyID: in.CurrencyID, InitialBalance: initial,
			IsActive: in.IsActive, SortOrder: next,
		}, time.Now().UTC()); err != nil {
			return err
		}
		return r.SaveCreditCard(ctx, id, terms)
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// Convert makes a card of an ordinary bank account, which keeps its payments.
// This is how a v1.0 card, upgraded to a bank account, gets its terms. A card
// does not go back to another type.
func (s CreditCards) Convert(ctx context.Context, accountID int64, in CardInput) (*Card, error) {
	cur, err := s.Accounts.ByID(ctx, accountID)
	if err != nil || cur == nil {
		return nil, err
	}
	if cur.Type != "bank" {
		return nil, ErrCardNotConvertible
	}
	terms, err := termsFor(in, cur.Currency.Unit())
	if err != nil {
		return nil, err
	}
	err = store.Tx(ctx, s.Accounts.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		if err := r.UpdateAccount(ctx, accountID, repo.AccountFields{
			Name: cur.Name, Type: TypeCredit, CurrencyID: cur.CurrencyID, InitialBalance: cur.InitialBalance,
			IsActive: cur.IsActive,
		}, time.Now().UTC()); err != nil {
			return err
		}
		return r.SaveCreditCard(ctx, accountID, terms)
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, accountID)
}

// Update saves a card and its terms. The currency cannot change.
func (s CreditCards) Update(ctx context.Context, id int64, e CardEdit) (*Card, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return nil, err
	}
	initial, err := money.FromInput(e.InitialBalance, cur.Currency.Unit())
	if err != nil {
		return nil, err
	}
	terms, err := termsFor(e.Terms, cur.Currency.Unit())
	if err != nil {
		return nil, err
	}
	err = store.Tx(ctx, s.Accounts.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		if err := r.UpdateAccount(ctx, id, repo.AccountFields{
			Name: e.Name, Type: TypeCredit, CurrencyID: cur.CurrencyID, InitialBalance: initial, IsActive: e.IsActive,
		}, time.Now().UTC()); err != nil {
			return err
		}
		return r.SaveCreditCard(ctx, id, terms)
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// Delete removes a card that has no payments.
func (s CreditCards) Delete(ctx context.Context, id int64) error {
	if s.repo().CountAccountPayments(ctx, id) > 0 {
		return fmt.Errorf("has transactions")
	}
	return s.repo().DeleteCreditCard(ctx, id)
}

// State is the state of a card as of today (in the user's time zone, which the
// caller knows).
func (s CreditCards) State(ctx context.Context, c Card, today time.Time) (CardState, error) {
	flows, err := s.repo().DailyFlows(ctx, c.ID)
	if err != nil {
		return CardState{}, err
	}
	return ComputeCardState(c.Terms, c.InitialBalance, flows, today), nil
}

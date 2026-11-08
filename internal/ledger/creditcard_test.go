package ledger

import (
	"testing"
	"time"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

var usd2 = money.Unit{ID: 1, Decimals: 2}

func usdM(minor int64) money.Money { return money.New(minor, usd2) }

func intp(n int) *int { return &n }

// card is a fixed_day card with a 1000.00 limit.
func fixedCard(statementDay, grace int) CreditCard {
	return CreditCard{
		Limit: usdM(100000), Cycle: CycleFixedDay, StatementDay: intp(statementDay), GraceDays: grace,
	}
}

// spend and pay make the flows of one day.
func spendOut(day string, minor int64) repo.DayFlow {
	return repo.DayFlow{Day: day, Delta: -minor, Outflow: -minor}
}

func payIn(day string, minor int64) repo.DayFlow {
	return repo.DayFlow{Day: day, Delta: minor, Inflow: minor}
}

func cardAt(t *testing.T, c CreditCard, initial int64, today string, flows ...repo.DayFlow) CardState {
	t.Helper()
	d, err := time.Parse("2006-01-02", today)
	if err != nil {
		t.Fatal(err)
	}
	return ComputeCardState(c, usdM(initial), flows, nil, d)
}

func TestStatementDateIsTheLastClosedOne(t *testing.T) {
	c := fixedCard(20, 25)
	for _, tc := range []struct{ today, statement, start, end string }{
		{"2026-03-25", "2026-03-20", "2026-03-21", "2026-04-20"},
		{"2026-03-21", "2026-03-20", "2026-03-21", "2026-04-20"},
		// On the statement day the period is not closed yet.
		{"2026-03-20", "2026-02-20", "2026-02-21", "2026-03-20"},
		{"2026-03-05", "2026-02-20", "2026-02-21", "2026-03-20"},
		{"2026-01-05", "2025-12-20", "2025-12-21", "2026-01-20"},
	} {
		s := cardAt(t, c, 0, tc.today)
		if s.Statement.Date != tc.statement || s.PeriodStart != tc.start || s.PeriodEnd != tc.end {
			t.Errorf("today %s: statement %s period %s..%s, want %s %s..%s",
				tc.today, s.Statement.Date, s.PeriodStart, s.PeriodEnd, tc.statement, tc.start, tc.end)
		}
	}
}

func TestStatementDayPastMonthEndIsLastDay(t *testing.T) {
	c := fixedCard(31, 20)
	for _, tc := range []struct{ today, statement string }{
		{"2026-03-05", "2026-02-28"}, // February 2026 has 28 days
		{"2026-04-05", "2026-03-31"},
		{"2026-05-05", "2026-04-30"},
		{"2028-03-05", "2028-02-29"}, // leap year
	} {
		if got := cardAt(t, c, 0, tc.today).Statement.Date; got != tc.statement {
			t.Errorf("today %s: statement %s, want %s", tc.today, got, tc.statement)
		}
	}
}

func TestStatementBalanceIsDebtAtStatementDay(t *testing.T) {
	c := fixedCard(20, 25)
	// 20 Feb is the statement day, so that day's purchase is in the statement and
	// the one after it is in the next period.
	s := cardAt(t, c, 0, "2026-03-05",
		spendOut("2026-02-10", 30000), spendOut("2026-02-20", 20000), spendOut("2026-02-21", 5000))
	st := s.Statement
	if st.Date != "2026-02-20" || st.Balance.Minor() != 50000 {
		t.Fatalf("statement %+v, want 2026-02-20 with 500.00", st)
	}
	if st.DueDate != "2026-03-17" { // 20 Feb + 25 days
		t.Errorf("due %s, want 2026-03-17", st.DueDate)
	}
	if st.Paid.Minor() != 0 || st.AmountDue.Minor() != 50000 {
		t.Errorf("paid %s due %s, want 0 and 500", st.Paid, st.AmountDue)
	}
	// Spent in the period 21 Feb..20 Mar so far.
	if s.Spent.Minor() != 5000 {
		t.Errorf("spent %s, want 50", s.Spent)
	}
	if s.Used.Minor() != 55000 || s.Available.Minor() != 45000 {
		t.Errorf("used %s available %s, want 550 and 450", s.Used, s.Available)
	}
}

func TestOpeningDebtIsPartOfTheStatement(t *testing.T) {
	s := cardAt(t, fixedCard(20, 25), -12000, "2026-03-05")
	if s.Statement.Balance.Minor() != 12000 {
		t.Errorf("statement balance %s, want 120", s.Statement.Balance)
	}
}

func TestPaymentsReduceAmountDueAndClearStatus(t *testing.T) {
	c := fixedCard(20, 25)
	flows := []repo.DayFlow{spendOut("2026-02-10", 50000)}

	s := cardAt(t, c, 0, "2026-03-05", flows...)
	if s.Status.Level != StatusWarning || s.Status.Reason != ReasonPaymentDue ||
		s.Status.Amount.Minor() != 50000 || s.Status.Date != "2026-03-17" {
		t.Errorf("status %+v, want a warning for 500 due on 2026-03-17", s.Status)
	}

	part := append(flows, payIn("2026-03-06", 20000))
	s = cardAt(t, c, 0, "2026-03-10", part...)
	if s.Statement.Paid.Minor() != 20000 || s.Statement.AmountDue.Minor() != 30000 || s.Status.Level != StatusWarning {
		t.Errorf("after a part payment: %+v / %+v", s.Statement, s.Status)
	}

	full := append(flows, payIn("2026-03-06", 50000))
	s = cardAt(t, c, 0, "2026-03-10", full...)
	if s.Statement.AmountDue.Minor() != 0 || s.Status.Level != StatusOK {
		t.Errorf("after paying in full: %+v / %+v", s.Statement, s.Status)
	}

	// More than the statement is still not a negative amount due.
	over := append(flows, payIn("2026-03-06", 60000))
	if s = cardAt(t, c, 0, "2026-03-10", over...); s.Statement.AmountDue.Minor() != 0 {
		t.Errorf("amount due %s after an overpayment, want 0", s.Statement.AmountDue)
	}
}

func TestLaterPurchasesDoNotReduceAmountDue(t *testing.T) {
	flows := []repo.DayFlow{spendOut("2026-02-10", 50000), spendOut("2026-03-01", 70000)}
	s := cardAt(t, fixedCard(20, 25), 0, "2026-03-05", flows...)
	if s.Statement.AmountDue.Minor() != 50000 {
		t.Errorf("amount due %s, want 500", s.Statement.AmountDue)
	}
}

func TestMissedDueDateIsDangerUntilPaid(t *testing.T) {
	c := fixedCard(20, 25)
	flows := []repo.DayFlow{spendOut("2026-02-10", 50000)}

	// The due date is 17 March: that day is still in time, the next is not.
	if s := cardAt(t, c, 0, "2026-03-17", flows...); s.Status.Level != StatusWarning {
		t.Errorf("on the due date: %+v, want a warning", s.Status)
	}
	s := cardAt(t, c, 0, "2026-03-18", flows...)
	if s.Status.Level != StatusDanger || s.Status.Reason != ReasonOverdue ||
		s.Status.Amount.Minor() != 50000 || s.Status.Date != "2026-03-17" {
		t.Errorf("after the due date: %+v, want overdue 500 since 2026-03-17", s.Status)
	}

	// A payment entered later with a past date clears it, so a forgotten entry
	// leaves nothing stale.
	back := append(flows, payIn("2026-03-12", 50000))
	if s = cardAt(t, c, 0, "2026-03-18", back...); s.Status.Level != StatusOK {
		t.Errorf("after a backdated payment: %+v, want ok", s.Status)
	}
}

func TestOverdueSurvivesTheNextStatement(t *testing.T) {
	// 500 from February was never paid; on 21 March a new statement exists but
	// the February one is still past due.
	flows := []repo.DayFlow{spendOut("2026-02-10", 50000)}
	s := cardAt(t, fixedCard(20, 25), 0, "2026-03-25", flows...)
	if s.Statement.Date != "2026-03-20" || s.Statement.AmountDue.Minor() != 50000 {
		t.Fatalf("latest statement %+v", s.Statement)
	}
	if s.Status.Level != StatusDanger || s.Status.Reason != ReasonOverdue || s.Status.Date != "2026-03-17" {
		t.Errorf("status %+v, want the February statement overdue", s.Status)
	}
}

func TestOverLimitIsDanger(t *testing.T) {
	c := fixedCard(20, 25)
	s := cardAt(t, c, 0, "2026-03-05", spendOut("2026-03-01", 120000))
	if s.Status.Level != StatusDanger || s.Status.Reason != ReasonOverLimit || s.Status.Amount.Minor() != 20000 {
		t.Errorf("status %+v, want over the limit by 200", s.Status)
	}
	if s.Available.Minor() != -20000 {
		t.Errorf("available %s, want -200", s.Available)
	}
	// At exactly the limit it is not over.
	if s = cardAt(t, c, 0, "2026-03-05", spendOut("2026-03-01", 100000)); s.Status.Reason == ReasonOverLimit {
		t.Errorf("at the limit: %+v", s.Status)
	}
}

func TestPositiveBalanceIsOK(t *testing.T) {
	s := cardAt(t, fixedCard(20, 25), 0, "2026-03-05", payIn("2026-02-10", 5000))
	if s.Used.Minor() != 0 || s.Available.Minor() != 105000 || s.Statement.Balance.Minor() != 0 ||
		s.Status.Level != StatusOK {
		t.Errorf("overpaid card: %+v", s)
	}
}

// Paying 1020 to clear a debt of 1000 is -1020 on a bank account, +1000 on the
// card and a fee line: the card sees only its own payment.
func TestCommissionIsNotPaidOnTheCard(t *testing.T) {
	flows := []repo.DayFlow{spendOut("2026-02-10", 100000), payIn("2026-03-01", 100000)}
	s := cardAt(t, fixedCard(20, 25), 0, "2026-03-05", flows...)
	if s.Statement.Paid.Minor() != 100000 || s.Statement.AmountDue.Minor() != 0 || s.Status.Level != StatusOK {
		t.Errorf("%+v / %+v", s.Statement, s.Status)
	}
}

func TestFutureDaysAreNotCounted(t *testing.T) {
	s := cardAt(t, fixedCard(20, 25), 0, "2026-03-05", spendOut("2026-02-10", 10000), spendOut("2026-04-01", 99000))
	if s.Used.Minor() != 10000 {
		t.Errorf("used %s, want 100 (the April purchase has not happened)", s.Used)
	}
}

func TestCardTermsAreValidated(t *testing.T) {
	ok := fixedCard(20, 25)
	bad := []struct {
		name string
		edit func(*CreditCard)
	}{
		{"zero limit", func(c *CreditCard) { c.Limit = usdM(0) }},
		{"negative limit", func(c *CreditCard) { c.Limit = usdM(-1) }},
		{"day 0", func(c *CreditCard) { c.StatementDay = intp(0) }},
		{"day 32", func(c *CreditCard) { c.StatementDay = intp(32) }},
		{"no day", func(c *CreditCard) { c.StatementDay = nil }},
		{"negative grace", func(c *CreditCard) { c.GraceDays = -1 }},
		{"grace too long", func(c *CreditCard) { c.GraceDays = MaxGraceDays + 1 }},
		{"first_expense", func(c *CreditCard) { c.Cycle, c.StatementDay, c.PeriodDays = CycleFirstExpense, nil, intp(30) }},
		{"unknown cycle", func(c *CreditCard) { c.Cycle = "monthly" }},
		{"period days on fixed_day", func(c *CreditCard) { c.PeriodDays = intp(30) }},
	}
	if err := ValidateCard(ok); err != nil {
		t.Fatalf("a valid card: %v", err)
	}
	for _, tc := range bad {
		c := ok
		tc.edit(&c)
		if err := ValidateCard(c); err == nil {
			t.Errorf("%s: accepted", tc.name)
		}
	}
	if err := ValidateCard(CreditCard{Limit: usdM(1), Cycle: CycleFixedDay, StatementDay: intp(1), GraceDays: 0}); err != nil {
		t.Errorf("grace 0 and day 1: %v", err)
	}
}

func TestCardStateFollowsAPoint(t *testing.T) {
	c := fixedCard(20, 25)
	d, _ := time.Parse("2006-01-02", "2026-03-25")
	// Spent 500 on 03-10 and 30 on 03-18; a point on 03-15 says the card owes 200
	// (the 03-15 spending of 40 is already inside it).
	flows := []repo.DayFlow{spendOut("2026-03-10", 50000), spendOut("2026-03-15", 4000), spendOut("2026-03-18", 3000)}
	points := []Reconciliation{{Date: "2026-03-15", ActualBalance: usdM(-20000)}}
	s := ComputeCardState(c, usdM(0), flows, points, d)
	if s.Used.Minor() != 23000 {
		t.Errorf("used = %d, want 23000", s.Used.Minor())
	}
	if s.Available.Minor() != 100000-23000 {
		t.Errorf("available = %d", s.Available.Minor())
	}
	if s.Statement.Date != "2026-03-20" || s.Statement.Balance.Minor() != 23000 {
		t.Errorf("statement %+v", s.Statement)
	}
	// The statement before the point is still computed from the opening balance.
	d2, _ := time.Parse("2006-01-02", "2026-03-18")
	if s := ComputeCardState(c, usdM(0), flows, points, d2); s.Statement.Date != "2026-02-20" || s.Statement.Balance.Minor() != 0 {
		t.Errorf("older statement %+v", s.Statement)
	}
	// Without points the card is as before.
	if s := ComputeCardState(c, usdM(0), flows, nil, d); s.Used.Minor() != 57000 {
		t.Errorf("no point: used = %d", s.Used.Minor())
	}
}

func TestCreditCardServiceBalanceAndStateAgree(t *testing.T) {
	e := newMoneyEnv(t)
	cards := CreditCards{Accounts: Accounts{DB: e.db}}
	c, err := cards.Create(e.ctx, newCard("Gold", e.usd.ID, "1000", 20, 25))
	if err != nil {
		t.Fatal(err)
	}
	e.txOn(t, simple("expense", c.ID, "500"), "2026-03-10")
	e.point(t, c.Account, "2026-03-15", "-200")
	e.txOn(t, simple("expense", c.ID, "30"), "2026-03-18")
	c, _ = cards.ByID(e.ctx, c.ID)
	checkBal(t, "card balance", c.Balance.Decimal().String(), "-230")
	today, _ := time.Parse("2006-01-02", "2026-03-25")
	st, err := cards.State(e.ctx, *c, today)
	if err != nil {
		t.Fatal(err)
	}
	if st.Used.Decimal().String() != "230" || st.Available.Decimal().String() != "770" {
		t.Errorf("state %+v", st)
	}
}

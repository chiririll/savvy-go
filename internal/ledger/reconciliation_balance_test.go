package ledger

import (
	"testing"
	"time"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

// point pins the account's balance at the end of date.
func (e *moneyEnv) point(t *testing.T, a Account, date, actual string) {
	t.Helper()
	m, err := money.FromInput(dec(actual), a.Currency.Unit())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (repo.Repo{DB: e.db}).UpsertReconciliation(
		e.ctx, repo.ReconciliationFields{AccountID: a.ID, Date: date, ActualBalance: m}, time.Now().UTC(),
	); err != nil {
		t.Fatal(err)
	}
}

func (e *moneyEnv) balanceAt(t *testing.T, a Account, asOf string) string {
	t.Helper()
	m, err := (Accounts{DB: e.db}).Balance(e.ctx, a, asOf)
	if err != nil {
		t.Fatal(err)
	}
	return m.Decimal().String()
}

func checkBal(t *testing.T, what, got, want string) {
	t.Helper()
	if !dec(got).Equal(dec(want)) {
		t.Errorf("%s = %s, want %s", what, got, want)
	}
}

// usdAcc starts at 100. Payments: +50 on 01-10, -20 on 01-15, -5 on 01-20.
func pointEnv(t *testing.T) *moneyEnv {
	e := newMoneyEnv(t)
	e.txOn(t, simple("income", e.usdAcc.ID, "50"), "2024-01-10")
	e.txOn(t, simple("expense", e.usdAcc.ID, "20"), "2024-01-15")
	e.txOn(t, simple("expense", e.usdAcc.ID, "5"), "2024-01-20")
	return e
}

func TestBalanceBeforeOnAndAfterAPoint(t *testing.T) {
	e := pointEnv(t)
	a := e.usdAcc
	e.point(t, a, "2024-01-15", "200") // computed would be 130
	checkBal(t, "before", e.balanceAt(t, a, "2024-01-14"), "150")
	checkBal(t, "on the point date (payment of that day is inside actual)", e.balanceAt(t, a, "2024-01-15"), "200")
	checkBal(t, "after, before more payments", e.balanceAt(t, a, "2024-01-19"), "200")
	checkBal(t, "after a later payment", e.balanceAt(t, a, "2024-01-20"), "195")
	checkBal(t, "now", e.balanceAt(t, a, ""), "195")
	if got := e.balance(t, a.ID); !got.Equal(dec("195")) {
		t.Errorf("ByID balance = %s", got)
	}
}

func TestBackdatedPaymentClosesGapKeepsLaterBalance(t *testing.T) {
	e := pointEnv(t)
	a := e.usdAcc
	e.point(t, a, "2024-01-15", "200")
	acc := Accounts{DB: e.db}
	before, _ := acc.ReconciliationChecks(e.ctx, a)
	if len(before) != 1 {
		t.Fatalf("checks %+v", before)
	}
	checkBal(t, "expected", before[0].Expected.Decimal().String(), "130")
	checkBal(t, "difference", before[0].Difference.Decimal().String(), "70")

	e.txOn(t, simple("income", e.usdAcc.ID, "70"), "2024-01-12")
	after, _ := acc.ReconciliationChecks(e.ctx, a)
	checkBal(t, "expected after back-dated", after[0].Expected.Decimal().String(), "200")
	checkBal(t, "difference closed", after[0].Difference.Decimal().String(), "0")
	checkBal(t, "balance after the point", e.balanceAt(t, a, ""), "195")
	checkBal(t, "balance before the point shifts", e.balanceAt(t, a, "2024-01-14"), "220")
}

func TestDifferenceIsMeasuredAgainstThePreviousPoint(t *testing.T) {
	e := pointEnv(t)
	a := e.usdAcc
	e.point(t, a, "2024-01-15", "200") // initial 100 +50 -20 = 130 -> diff 70
	e.point(t, a, "2024-01-25", "180") // 200 + (-5) = 195 -> diff -15
	checks, err := (Accounts{DB: e.db}).ReconciliationChecks(e.ctx, a)
	if err != nil || len(checks) != 2 {
		t.Fatalf("%+v %v", checks, err)
	}
	for i, w := range [][3]string{{"2024-01-15", "130", "70"}, {"2024-01-25", "195", "-15"}} {
		c := checks[i]
		if c.Point.Date != w[0] {
			t.Errorf("point %d date %s", i, c.Point.Date)
		}
		checkBal(t, "expected", c.Expected.Decimal().String(), w[1])
		checkBal(t, "difference", c.Difference.Decimal().String(), w[2])
	}
	checkBal(t, "between", e.balanceAt(t, a, "2024-01-22"), "195")
	checkBal(t, "after second", e.balanceAt(t, a, ""), "180")
}

func TestPointReplacedOnSameDate(t *testing.T) {
	e := pointEnv(t)
	e.point(t, e.usdAcc, "2024-01-15", "200")
	e.point(t, e.usdAcc, "2024-01-15", "140")
	checkBal(t, "now", e.balanceAt(t, e.usdAcc, ""), "135")
	checks, _ := (Accounts{DB: e.db}).ReconciliationChecks(e.ctx, e.usdAcc)
	if len(checks) != 1 {
		t.Fatalf("%+v", checks)
	}
	checkBal(t, "difference", checks[0].Difference.Decimal().String(), "10")
}

func TestBalanceSeriesAcrossAPoint(t *testing.T) {
	e := pointEnv(t)
	a := e.usdAcc
	e.point(t, a, "2024-01-15", "200")
	e.point(t, a, "2024-01-25", "180")
	dates := []string{"2024-01-09", "2024-01-14", "2024-01-15", "2024-01-19", "2024-01-20", "2024-01-25", "2024-01-30"}
	got, err := (Accounts{DB: e.db}).BalanceSeries(e.ctx, a, dates)
	if err != nil {
		t.Fatal(err)
	}
	for i, w := range []string{"100", "150", "200", "200", "195", "180", "180"} {
		checkBal(t, dates[i], got[i].Decimal().String(), w)
		checkBal(t, dates[i]+" Balance()", e.balanceAt(t, a, dates[i]), w) // series and Balance agree
	}
	// A series that starts after the first point.
	got, _ = (Accounts{DB: e.db}).BalanceSeries(e.ctx, a, []string{"2024-01-22", "2024-01-26"})
	checkBal(t, "late start 1", got[0].Decimal().String(), "195")
	checkBal(t, "late start 2", got[1].Decimal().String(), "180")
}

func TestPointsPerAccountInDifferentCurrencies(t *testing.T) {
	e := newMoneyEnv(t) // jpy 3000, no decimals
	e.txOn(t, simple("expense", e.jpyAcc.ID, "500"), "2024-01-10")
	e.txOn(t, simple("expense", e.usdAcc.ID, "10"), "2024-01-10")
	e.point(t, e.jpyAcc, "2024-01-12", "2400")
	e.point(t, e.usdAcc, "2024-01-12", "95.5")
	e.txOn(t, simple("expense", e.jpyAcc.ID, "100"), "2024-01-13")
	checkBal(t, "jpy", e.balanceAt(t, e.jpyAcc, ""), "2300")
	checkBal(t, "usd", e.balanceAt(t, e.usdAcc, ""), "95.5")
	checks, _ := (Accounts{DB: e.db}).ReconciliationChecks(e.ctx, e.jpyAcc)
	checkBal(t, "jpy diff", checks[0].Difference.Decimal().String(), "-100")
	if checks[0].Difference.Unit() != e.jpyAcc.Currency.Unit() {
		t.Errorf("unit %v", checks[0].Difference.Unit())
	}
	checks, _ = (Accounts{DB: e.db}).ReconciliationChecks(e.ctx, e.usdAcc)
	checkBal(t, "usd diff", checks[0].Difference.Decimal().String(), "5.5")
}

func TestAccountWithoutPointsIsUnchanged(t *testing.T) {
	e := pointEnv(t)
	e.point(t, e.jpyAcc, "2024-01-01", "1") // another account's point changes nothing here
	checkBal(t, "now", e.balanceAt(t, e.usdAcc, ""), "125")
	checkBal(t, "asOf", e.balanceAt(t, e.usdAcc, "2024-01-15"), "130")
	checks, err := (Accounts{DB: e.db}).ReconciliationChecks(e.ctx, e.usdAcc)
	if err != nil || len(checks) != 0 {
		t.Errorf("checks %+v %v", checks, err)
	}
}

func TestPointInTheFutureDoesNotApplyYet(t *testing.T) {
	e := pointEnv(t)
	e.point(t, e.usdAcc, "2024-02-01", "999")
	checkBal(t, "before it", e.balanceAt(t, e.usdAcc, "2024-01-31"), "125")
	checkBal(t, "on it", e.balanceAt(t, e.usdAcc, "2024-02-01"), "999")
}

func TestNetTotalUsesPoints(t *testing.T) {
	e := pointEnv(t)
	e.point(t, e.usdAcc, "2024-01-15", "200")
	// jpy 3000 * 0.01 = 30 USD; usd 195.
	got := (Accounts{DB: e.db}).TotalAt(e.ctx, e.usd, "2024-01-31")
	checkBal(t, "total", got.Decimal().String(), "225")
}

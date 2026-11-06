package ledger_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/filter"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

// A report adds up lines, so these tests post transactions whose payments and
// lines differ on purpose: what moved is not what it was for.

type pay struct {
	account, minor int64
}

type line struct {
	category *int64
	currency int64
	minor    int64
}

// post stores a transaction of the given type, status and date, with the
// payments and lines given (amounts in minor units of their own currency).
func (f *fixture) post(t *testing.T, typ, status, date string, pays []pay, lines []line, tags ...int64) int64 {
	t.Helper()
	d := date
	id, err := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{Type: typ, Status: status, Date: &d}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	var ps []ledgerrepo.PaymentFields
	for _, p := range pays {
		a, _ := f.repo.Account(ctx, p.account)
		ps = append(ps, ledgerrepo.PaymentFields{AccountID: p.account, Amount: money.New(p.minor, a.Currency.Unit())})
	}
	if err := f.repo.ReplacePayments(ctx, id, ps); err != nil {
		t.Fatal(err)
	}
	var ls []ledgerrepo.LineFields
	for _, l := range lines {
		ls = append(ls, ledgerrepo.LineFields{CategoryID: l.category, Amount: money.New(l.minor, f.unit(l.currency))})
	}
	if err := f.repo.ReplaceLines(ctx, id, ls); err != nil {
		t.Fatal(err)
	}
	if len(tags) > 0 {
		if err := f.repo.SetTags(ctx, id, tags); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

// reportFixture is January 2026 in USD (the base) and JPY (0.0066 USD a yen):
//
//	01-02  expense   food 10.00 USD, paid from cash                      (tagged trip)
//	01-02  income    salary 1000.00 USD
//	01-03  expense   food 1000 JPY + other 500 JPY, paid from yen        (tagged trip)
//	01-04  expense   food 5.00 USD + food 300 JPY, paid from both accounts
//	01-05  transfer  100.00 USD out, 15000 JPY in, and a 2.00 USD fee line
//	01-06  expense   other 20.00 USD, but 50.00 USD handed over (change)
//	01-07  debt      lent 100.00 USD: no lines
//	01-08  expense   3.00 USD with no category
//
// and three that must not count: a pending expense, a skipped one, and one in
// February.
type reportFixture struct {
	*fixture
	other, fees, salary, trip int64
}

func newReportFixture(t *testing.T) *reportFixture {
	t.Helper()
	f := newFixture(t)
	now := time.Now().UTC()
	r := &reportFixture{fixture: f}
	r.other, _ = f.repo.InsertCategory(ctx, ledgerrepo.Category{Name: "Other", Type: "expense"}, now)
	r.fees, _ = f.repo.InsertCategory(ctx, ledgerrepo.Category{Name: "Fees", Type: "expense"}, now)
	r.salary, _ = f.repo.InsertCategory(ctx, ledgerrepo.Category{Name: "Salary", Type: "income"}, now)
	r.trip, _ = f.repo.InsertTag(ctx, "trip", now)
	debt := f.debt(t, "Ann", "owed_to_me", 10000)

	food, other, fees, salary := &f.food, &r.other, &r.fees, &r.salary
	f.post(t, "expense", "", "2026-01-02", []pay{{f.cash, -1000}}, []line{{food, f.usd, 1000}}, r.trip)
	f.post(t, "income", "", "2026-01-02", []pay{{f.cash, 100000}}, []line{{salary, f.usd, 100000}})
	f.post(t, "expense", "", "2026-01-03", []pay{{f.yen, -1500}},
		[]line{{food, f.jpy, 1000}, {other, f.jpy, 500}}, r.trip)
	f.post(t, "expense", "", "2026-01-04", []pay{{f.cash, -500}, {f.yen, -300}},
		[]line{{food, f.usd, 500}, {food, f.jpy, 300}})
	f.post(t, "transfer", "", "2026-01-05", []pay{{f.cash, -10000}, {f.yen, 15000}}, []line{{fees, f.usd, 200}})
	f.post(t, "expense", "", "2026-01-06", []pay{{f.cash, -5000}}, []line{{other, f.usd, 2000}})
	f.post(t, "debt_lend", "", "2026-01-07", []pay{{f.cash, -10000}, {debt, 10000}}, nil)
	f.post(t, "expense", "", "2026-01-08", []pay{{f.cash, -300}}, []line{{nil, f.usd, 300}})

	f.post(t, "expense", "pending", "2026-01-09", []pay{{f.cash, -999}}, []line{{food, f.usd, 999}})
	f.post(t, "expense", "skipped", "2026-01-09", []pay{{f.cash, -999}}, []line{{food, f.usd, 999}})
	f.post(t, "expense", "", "2026-02-01", []pay{{f.cash, -999}}, []line{{food, f.usd, 999}})
	return r
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func (f *fixture) usdUnit() money.Unit { return money.Unit{ID: f.usd, Decimals: 2} }

func (r *reportFixture) where(typ string) filter.ReportWhere {
	return filter.ReportWhere{Type: typ, Start: "2026-01-01", End: "2026-01-31"}
}

func wantMoney(t *testing.T, what string, got money.Money, want string, unit money.Unit) {
	t.Helper()
	if !got.Equal(money.FromDecimal(dec(want), unit)) {
		t.Errorf("%s = %s, want %s", what, got.Decimal(), want)
	}
}

func TestReportTotalsAreTheSumOfLines(t *testing.T) {
	r := newReportFixture(t)
	base := r.usdUnit()

	// Lines of every expense, a transfer's fee included, and the uncategorised
	// one; neither the change nor the debt are in it.
	wantMoney(t, "expenses", filter.SumByType(ctx, r.repo.DB, r.where("expense"), base), "51.88", base)
	wantMoney(t, "income", filter.SumByType(ctx, r.repo.DB, r.where("income"), base), "1000", base)

	w := r.where("expense")
	w.CategoryIDs = []int64{r.food}
	wantMoney(t, "food", filter.SumByType(ctx, r.repo.DB, w, base), "23.58", base)

	w = r.where("expense")
	w.TagIDs = []int64{r.trip}
	wantMoney(t, "trip", filter.SumByType(ctx, r.repo.DB, w, base), "19.90", base)

	// A transaction is in the report of an account when one of its payments is.
	w = r.where("expense")
	w.AccountIDs = []int64{r.yen}
	wantMoney(t, "paid from yen", filter.SumByType(ctx, r.repo.DB, w, base), "18.88", base)
}

func TestReportGroupsByCategoryLargestFirst(t *testing.T) {
	r := newReportFixture(t)
	base := r.usdUnit()
	got := filter.SumGroupedByCategory(ctx, r.repo.DB, r.where("expense"), base)
	want := []struct {
		id    int64
		total string
	}{{r.food, "23.58"}, {r.other, "23.30"}, {r.fees, "2"}}
	if len(got) != len(want) {
		t.Fatalf("categories = %+v, want %d (uncategorised lines have none)", got, len(want))
	}
	sum := money.Zero(base)
	for i, w := range want {
		if got[i].ID != w.id {
			t.Errorf("category %d is %d, want %d", i, got[i].ID, w.id)
		}
		wantMoney(t, got[i].Name, got[i].Total, w.total, base)
		sum = sum.Add(got[i].Total)
	}
	// The categories and the uncategorised 3.00 make up the whole.
	wantMoney(t, "categories + uncategorised", sum.Add(money.FromDecimal(dec("3"), base)),
		filter.SumByType(ctx, r.repo.DB, r.where("expense"), base).Decimal().String(), base)

	income := filter.SumGroupedByCategory(ctx, r.repo.DB, r.where("income"), base)
	if len(income) != 1 || income[0].ID != r.salary {
		t.Fatalf("income categories = %+v", income)
	}
}

func TestReportCountsATransactionOncePerDay(t *testing.T) {
	r := newReportFixture(t)
	base := r.usdUnit()
	days := filter.DailyTotals(ctx, r.repo.DB, r.where("expense"), base)
	want := map[string]struct {
		total string
		count int
	}{
		"2026-01-02": {"10", 1},
		"2026-01-03": {"9.90", 1}, // two lines
		"2026-01-04": {"6.98", 1}, // two lines in two currencies
		"2026-01-05": {"2", 1},    // the fee of a transfer
		"2026-01-06": {"20", 1},   // not the 50 that was handed over
		"2026-01-08": {"3", 1},
	}
	if len(days) != len(want) {
		t.Fatalf("days = %+v", days)
	}
	for _, d := range days {
		w, ok := want[d.Day]
		if !ok {
			t.Errorf("unexpected day %s", d.Day)
			continue
		}
		wantMoney(t, d.Day, d.Total, w.total, base)
		if d.Count != w.count {
			t.Errorf("%s: %d transactions, want %d", d.Day, d.Count, w.count)
		}
	}
}

func TestReportGroupsByPeriod(t *testing.T) {
	r := newReportFixture(t)
	base := r.usdUnit()
	got := filter.GroupedByPeriod(ctx, r.repo.DB, r.where("expense"), "month", base)
	if len(got) != 1 || got[0].Key != "2026-01-01" {
		t.Fatalf("months = %+v", got)
	}
	wantMoney(t, "January", got[0].Total, "51.88", base)

	// 2026-01-04 is a Sunday: the week of the 4th holds the 4th to the 8th.
	weeks := filter.GroupedByPeriod(ctx, r.repo.DB, r.where("expense"), "week", base)
	byWeek := map[string]money.Money{}
	for _, w := range weeks {
		byWeek[w.Key] = w.Total
	}
	if len(byWeek) != 2 {
		t.Fatalf("weeks = %+v", weeks)
	}
	wantMoney(t, "week of 01-04", byWeek["2026-01-04"], "31.98", base)
	wantMoney(t, "week of 12-28", byWeek["2025-12-28"], "19.90", base)
}

func TestTopTransactionsAreRankedByTheirLines(t *testing.T) {
	r := newReportFixture(t)
	base := r.usdUnit()
	top := filter.TopTransactions(ctx, r.repo.DB, r.where("expense"), 3, base)
	if len(top) != 3 {
		t.Fatalf("top = %+v", top)
	}
	// 20.00 (the change is not in it), then 10.00, then the yen dinner 9.90.
	for i, want := range []string{"20", "10", "9.90"} {
		wantMoney(t, "top "+want, top[i].Amount, want, base)
	}
	if top[0].AccID != r.cash || top[2].AccID != r.yen {
		t.Errorf("accounts = %d, %d; want cash %d and yen %d", top[0].AccID, top[2].AccID, r.cash, r.yen)
	}
	// The category of a transaction is the one of its largest line.
	if !top[2].CatID.Valid || top[2].CatID.Int64 != r.food {
		t.Errorf("category of the yen dinner = %+v, want food", top[2].CatID)
	}
	if more := filter.TopTransactions(ctx, r.repo.DB, r.where("income"), 5, base); len(more) != 1 || more[0].AccID != r.cash {
		t.Errorf("income top = %+v", more)
	}
}

func TestBudgetSpentSumsLinesInBudgetCurrency(t *testing.T) {
	r := newReportFixture(t)
	usd, jpy := r.usdUnit(), r.unit(r.jpy)
	spent := func(cats, tags []int64, in money.Unit, rate string) money.Money {
		t.Helper()
		got, err := filter.BudgetSpent(ctx, r.repo.DB, "2026-01-01", "2026-01-31", cats, tags, in, dec(rate))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	wantMoney(t, "all", spent(nil, nil, usd, "1"), "51.88", usd)
	wantMoney(t, "food", spent([]int64{r.food}, nil, usd, "1"), "23.58", usd)
	wantMoney(t, "food and fees", spent([]int64{r.food, r.fees}, nil, usd, "1"), "25.58", usd)
	wantMoney(t, "trip", spent(nil, []int64{r.trip}, usd, "1"), "19.90", usd)
	wantMoney(t, "no category at all", spent([]int64{}, nil, usd, "1"), "0", usd)
	// 51.88 USD at 0.0066 USD a yen: 7860.6 yen, rounded into whole yen.
	wantMoney(t, "all in yen", spent(nil, nil, jpy, "0.0066"), "7861", jpy)
}

func TestCategoryUsageCountsTransactionsNotLines(t *testing.T) {
	r := newReportFixture(t)
	// Food is on three transactions in January and one each in the pending,
	// skipped and February ones; the one of 01-04 has two food lines.
	food, _ := r.repo.Category(ctx, r.food)
	if food.TransactionsCount != 6 {
		t.Fatalf("food is on %d transactions, want 6", food.TransactionsCount)
	}
	stats := r.repo.CategoryStatistics(ctx, r.food, "2026-01-01", "2026-01-31")
	var count int
	for _, s := range stats {
		count += s.Count
	}
	// Per currency: USD on 01-02 and 01-04, JPY on 01-03 and 01-04.
	if len(stats) != 2 || count != 4 {
		t.Fatalf("stats = %+v", stats)
	}
}

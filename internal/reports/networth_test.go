package reports

import (
	"testing"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

func (e *seriesEnv) budget(t *testing.T, in ledger.BudgetInput) {
	t.Helper()
	if in.Period == "" {
		in.Period = "monthly"
	}
	if _, err := (ledger.Budgets{DB: e.db}).Create(e.ctx, in); err != nil {
		t.Fatal(err)
	}
}

func TestPaceBudgetAddsEveryCurrencyAtItsRate(t *testing.T) {
	e := newSeriesEnv(t)
	e.budget(t, ledger.BudgetInput{Name: "usd", Amount: dec("50"), CurrencyID: &e.usd.ID})
	e.budget(
		t,
		ledger.BudgetInput{Name: "yen", Amount: dec("10000"), CurrencyID: &e.jpy.ID},
	) // 100 USD
	e.budget(
		t,
		ledger.BudgetInput{Name: "weekly", Amount: dec("999"), CurrencyID: &e.usd.ID, Period: "weekly"},
	) // not monthly
	e.budget(
		t,
		ledger.BudgetInput{Name: "off", Amount: dec("999"), CurrencyID: &e.usd.ID, IsActive: ptrTo(false)},
	) // not active

	got := (Service{DB: e.db}).ExpensePace(e.ctx, janRange)
	if len(got.Months) != 1 || got.Months[0].Budget == nil {
		t.Fatalf("months %+v", got.Months)
	}
	wantMoney(t, "budget", *got.Months[0].Budget, usd(e.moneyEnv, "150"))
}

func TestPaceGlobalBudgetStandsForTheWholeScope(t *testing.T) {
	e := newSeriesEnv(t)
	e.budget(t, ledger.BudgetInput{Name: "usd", Amount: dec("50"), CurrencyID: &e.usd.ID})
	e.budget(t, ledger.BudgetInput{Name: "all", Amount: dec("400"), CurrencyID: &e.usd.ID, IsGlobal: ptrTo(true)})
	got := (Service{DB: e.db}).ExpensePace(e.ctx, janRange)
	wantMoney(t, "global budget", *got.Months[0].Budget, usd(e.moneyEnv, "400"))
}

func TestPaceBudgetFollowsTheReportScope(t *testing.T) {
	e := newSeriesEnv(t)
	tag, err := (ledger.Tags{DB: e.db}).Create(e.ctx, "trip")
	if err != nil {
		t.Fatal(err)
	}
	e.budget(t, ledger.BudgetInput{Name: "all", Amount: dec("400"), CurrencyID: &e.usd.ID})
	e.budget(
		t,
		ledger.BudgetInput{
			Name:           "food",
			Amount:         dec("30"),
			CurrencyID:     &e.usd.ID,
			CategoryIDs:    []int64{e.food.ID},
			HasCategoryIDs: true,
		},
	)
	e.budget(
		t,
		ledger.BudgetInput{Name: "trip", Amount: dec("7"), CurrencyID: &e.usd.ID, TagIDs: []int64{tag.ID}, HasTagIDs: true},
	)
	svc := Service{DB: e.db}

	scoped := func(f Filter) *money.Money {
		f.PeriodType, f.StartDate, f.EndDate = "custom", "2024-01-13", "2024-01-15"
		return svc.ExpensePace(e.ctx, f).Months[0].Budget
	}
	if b := scoped(Filter{CategoryIDs: []int64{e.food.ID}}); b == nil || b.Decimal().String() != "30" {
		t.Errorf("a category sees the budgets tied to it: %v", b)
	}
	if b := scoped(Filter{TagIDs: []int64{tag.ID}}); b == nil || b.Decimal().String() != "7" {
		t.Errorf("a tag sees the budgets tied to it: %v", b)
	}
	if b := scoped(Filter{CategoryIDs: []int64{e.food.ID}, TagIDs: []int64{tag.ID}}); b == nil ||
		b.Decimal().String() != "37" {
		t.Errorf("both: %v", b)
	}
	if b := scoped(Filter{CategoryIDs: []int64{e.fun.ID}}); b != nil {
		t.Errorf("a category with no budget has none: %v", b)
	}
	if b := scoped(Filter{AccountIDs: []int64{e.usdAcc.ID}}); b != nil {
		t.Errorf("budgets are not kept per account: %v", b)
	}
}

func TestPaceAccumulatesSpendingDayByDay(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "10", "2024-01-13")
	e.add(t, "expense", e.jpyAcc.ID, &e.food.ID, "500", "2024-01-15") // 5 USD
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "99", "2024-01-20")  // outside the range

	got := (Service{DB: e.db}).ExpensePace(e.ctx, janRange)
	if len(got.Months) != 1 {
		t.Fatalf("months %+v", got.Months)
	}
	m := got.Months[0]
	if m.DaysInMonth != 31 || len(m.DailyExpenses) != 31 || m.MonthStart != "2024-01-01" || m.MonthEnd != "2024-01-31" {
		t.Fatalf("month %+v", m)
	}
	if m.CurrentDay != nil {
		t.Errorf("a past month has no current day: %d", *m.CurrentDay)
	}
	for day, want := range map[int]string{11: "0", 12: "10", 13: "10", 14: "15", 30: "15"} { // index = day-1
		wantMoney(t, "spent by day", m.DailyExpenses[day], usd(e.moneyEnv, want))
	}
	wantMoney(t, "total", m.TotalSpent, usd(e.moneyEnv, "15"))
	if m.Budget != nil {
		t.Error("no budgets, no budget line")
	}
}

func TestNetWorthChangeAgainstThePreviousPeriod(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "20", "2024-01-10")
	e.add(t, "income", e.usdAcc.ID, &e.salary.ID, "55", "2024-02-10")
	f := Filter{PeriodType: "month", PeriodValue: "2024-02", CompareWith: "previous_period"}

	got := (Service{DB: e.db}).NetWorth(e.ctx, f)
	// Opening balances are USD 100 and JPY 3000 at 0.01.
	wantMoney(t, "now", got.Current, usd(e.moneyEnv, "165"))
	if got.Previous == nil {
		t.Fatal("no previous")
	}
	wantMoney(t, "before", *got.Previous, usd(e.moneyEnv, "110"))
	wantMoney(t, "change", got.Change, usd(e.moneyEnv, "55"))
	if got.ChangePercent != 50 {
		t.Errorf("change %v%%, want 50", got.ChangePercent)
	}
}

func TestNetWorthCoversChosenAccountsOnly(t *testing.T) {
	e := newSeriesEnv(t)
	got := (Service{DB: e.db}).NetWorth(e.ctx, Filter{AccountIDs: []int64{e.jpyAcc.ID}})
	wantMoney(t, "yen account only", got.Current, usd(e.moneyEnv, "30"))
	if len(got.Accounts) != 1 || got.Accounts[0].Name != "jpy" || got.Accounts[0].Percentage != 100 {
		t.Errorf("accounts %+v", got.Accounts)
	}
}

func TestNetWorthHistoryStepsByPeriod(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "20", "2024-01-20")
	e.add(t, "income", e.usdAcc.ID, &e.salary.ID, "50", "2024-03-05")
	f := Filter{PeriodType: "custom", StartDate: "2024-01-15", EndDate: "2024-03-10"}

	got := (Service{DB: e.db}).NetWorthHistory(e.ctx, f, "month")
	if len(got.Dates) != 3 || got.Dates[0] != "2024-01-15" || got.Dates[1] != "2024-02-01" ||
		got.Dates[2] != "2024-03-01" {
		t.Fatalf("dates %v", got.Dates)
	}
	for i, want := range []string{"110", "110", "160"} { // end of January, end of February, 10 March
		wantMoney(t, got.Dates[i], got.Values[i], usd(e.moneyEnv, want))
	}

	days := (Service{DB: e.db}).NetWorthHistory(
		e.ctx,
		Filter{PeriodType: "custom", StartDate: "2024-01-19", EndDate: "2024-01-21"},
		"day",
	)
	for i, want := range []string{"130", "110", "110"} {
		wantMoney(t, days.Dates[i], days.Values[i], usd(e.moneyEnv, want))
	}
}

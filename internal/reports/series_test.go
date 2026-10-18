package reports

import (
	"context"
	"testing"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

// seriesEnv is the USD/JPY/BTC space with a few categorised transactions on
// known days, so the series reports have something to group.
type seriesEnv struct {
	*moneyEnv
	salary, food, fun ledger.Category
}

func newSeriesEnv(t *testing.T) *seriesEnv {
	t.Helper()
	e := &seriesEnv{moneyEnv: newMoneyEnv(t)}
	cats := ledger.Categories{DB: e.db}
	mk := func(name, typ string) ledger.Category {
		c, err := cats.Create(e.ctx, ledger.Category{Name: name, Type: typ})
		if err != nil {
			t.Fatal(err)
		}
		return *c
	}
	e.salary, e.food, e.fun = mk("Salary", "income"), mk("Food", "expense"), mk("Fun", "expense")
	return e
}

func (e *seriesEnv) add(t *testing.T, typ string, accountID int64, categoryID *int64, amount, date string) {
	t.Helper()
	if _,
		err := e.txs.Create(e.ctx,
		ledger.TxInput{Type: typ, AccountID: accountID, CategoryID: categoryID, Amount: dec(amount),
			Date: &date}); err != nil {
		t.Fatal(err)
	}
}

func usd(e *moneyEnv, s string) money.Money { return money.FromDecimal(dec(s), e.usd.Unit()) }

func wantMoney(t *testing.T, what string, got, want money.Money) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("%s = %s, want %s", what, got, want)
	}
}

func TestCashFlowOverTimeAccumulatesDaily(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "income", e.usdAcc.ID, &e.salary.ID, "100", "2024-01-13")
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "30", "2024-01-14")
	e.add(t, "expense", e.jpyAcc.ID, &e.food.ID, "1000", "2024-01-14") // 10 USD
	// Pending transactions are plans, not money moved.
	pending := "pending"
	date := "2024-01-15"
	if _,
		err := e.txs.Create(e.ctx,
		ledger.TxInput{Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("999"), Date: &date,
			Status: &pending}); err != nil {
		t.Fatal(err)
	}

	got := (Service{DB: e.db}).CashFlowOverTime(e.ctx, janRange, "day")
	if got.Currency != "USD" || len(got.Items) != 3 {
		t.Fatalf("report %+v", got)
	}
	wantDates := []string{"2024-01-13", "2024-01-14", "2024-01-15"}
	wantIncome := []string{"100", "0", "0"}
	wantExpenses := []string{"0", "40", "0"}
	wantBalance := []string{"100", "60", "60"}
	for i, p := range got.Items {
		if p.Date != wantDates[i] {
			t.Errorf("point %d date %s", i, p.Date)
		}
		wantMoney(t, p.Date+" income", p.Income, usd(e.moneyEnv, wantIncome[i]))
		wantMoney(t, p.Date+" expenses", p.Expenses, usd(e.moneyEnv, wantExpenses[i]))
		wantMoney(t, p.Date+" balance", p.Balance, usd(e.moneyEnv, wantBalance[i]))
		if p.PrevIncome != nil || p.PrevBalance != nil {
			t.Errorf("%s has comparison figures without a comparison", p.Date)
		}
	}
}

func TestCashFlowPairsPointsWithComparison(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "7", "2024-01-10")
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "20", "2024-01-14")
	f := janRange
	f.CompareWith = "previous_period" // 2024-01-10 .. 2024-01-12

	got := (Service{DB: e.db}).CashFlowOverTime(e.ctx, f, "day")
	first := got.Items[0]
	if first.PrevExpenses == nil || first.PrevBalance == nil {
		t.Fatalf("no comparison figures: %+v", first)
	}
	wantMoney(t, "previous expenses", *first.PrevExpenses, usd(e.moneyEnv, "7"))
	wantMoney(t, "previous balance", *first.PrevBalance, usd(e.moneyEnv, "-7"))
	wantMoney(t, "current expenses", got.Items[1].Expenses, usd(e.moneyEnv, "20"))
	wantMoney(t, "previous balance carries on", *got.Items[2].PrevBalance, usd(e.moneyEnv, "-7"))
}

func TestCashFlowMonthlyPointPerMonth(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "5", "2024-01-31")
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "8", "2024-04-01")
	f := Filter{PeriodType: "custom", StartDate: "2024-01-31", EndDate: "2024-04-02"}

	got := (Service{DB: e.db}).CashFlowOverTime(e.ctx, f, "month")
	if len(got.Items) != 4 {
		t.Fatalf("%d points, want January to April", len(got.Items))
	}
	wantMoney(t, "January", got.Items[0].Expenses, usd(e.moneyEnv, "5"))
	wantMoney(t, "April", got.Items[3].Expenses, usd(e.moneyEnv, "8"))
	wantMoney(t, "final balance", got.Items[3].Balance, usd(e.moneyEnv, "-13"))
}

func TestExpensesCategoryComparesWithEarlierPeriod(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "30", "2024-02-10")
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "20", "2024-01-10")
	e.add(t, "expense", e.usdAcc.ID, &e.fun.ID, "5", "2024-02-11")
	e.add(t, "expense", e.usdAcc.ID, &e.fun.ID, "500", "2024-03-01") // outside both periods
	f := Filter{PeriodType: "month", PeriodValue: "2024-02", CompareWith: "previous_period"}

	got := (Service{DB: e.db}).ExpensesByCategory(e.ctx, f)
	byName := map[string]CategoryExpense{}
	for _, c := range got.Categories {
		byName[c.Name] = c
	}
	if len(byName) != 2 {
		t.Fatalf("categories %+v", got.Categories)
	}
	wantMoney(t, "food now", byName["Food"].Current, usd(e.moneyEnv, "30"))
	wantMoney(t, "food before", byName["Food"].Previous, usd(e.moneyEnv, "20"))
	wantMoney(t, "fun now", byName["Fun"].Current, usd(e.moneyEnv, "5"))
	wantMoney(t, "fun before", byName["Fun"].Previous, usd(e.moneyEnv, "0"))
	if byName["Food"].Icon == "" || byName["Food"].Color == "" {
		t.Errorf("a category without an icon or colour gets defaults: %+v", byName["Food"])
	}
}

func TestExpensesCategoryWithoutComparisonZeroPrevious(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "30", "2024-02-10")
	got := (Service{DB: e.db}).ExpensesByCategory(e.ctx, Filter{PeriodType: "month", PeriodValue: "2024-02"})
	if len(got.Categories) != 1 {
		t.Fatalf("categories %+v", got.Categories)
	}
	wantMoney(t, "previous", got.Categories[0].Previous, usd(e.moneyEnv, "0"))
}

func TestCategorySummaryTotalsTheType(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "30", "2024-01-14")
	e.add(t, "expense", e.jpyAcc.ID, &e.fun.ID, "500", "2024-01-14") // 5 USD
	e.add(t, "income", e.usdAcc.ID, &e.salary.ID, "100", "2024-01-14")

	cats, total := (Service{DB: e.db}).CategorySummary(e.ctx, janRange, "expense")
	wantMoney(t, "total", total, usd(e.moneyEnv, "35"))
	if len(cats) != 2 {
		t.Fatalf("categories %+v", cats)
	}
	var sum money.Money = money.Zero(e.usd.Unit())
	for _, c := range cats {
		if c.Type != "expense" || c.TotalAmount == nil {
			t.Fatalf("category %+v", c)
		}
		sum = sum.Add(*c.TotalAmount)
	}
	wantMoney(t, "categories add up to the total", sum, total)
}

func TestTxCategoryGivesSharesOfTotal(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "75", "2024-01-14")
	e.add(t, "expense", e.usdAcc.ID, &e.fun.ID, "25", "2024-01-14")

	got := (Service{DB: e.db}).TxByCategory(e.ctx, janRange, "expense")
	wantMoney(t, "total", got.Total, usd(e.moneyEnv, "100"))
	shares := map[string]float64{}
	for _, c := range got.Items {
		shares[c.Name] = c.Percentage
	}
	if shares["Food"] != 75 || shares["Fun"] != 25 {
		t.Errorf("shares %v", shares)
	}

	empty := (Service{DB: e.db}).TxByCategory(e.ctx, Filter{PeriodType: "month", PeriodValue: "2020-01"}, "expense")
	if len(empty.Items) != 0 || !empty.Total.IsZero() {
		t.Errorf("an empty period has no shares: %+v", empty)
	}
}

func TestTxDynamicsTotalLinePerCategory(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "10", "2024-01-13")
	e.add(t, "expense", e.usdAcc.ID, &e.fun.ID, "4", "2024-01-15")
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "1", "2024-01-15")

	got := (Service{DB: e.db}).TxDynamics(e.ctx, janRange, "expense", "day")
	if len(got.Dates) != 3 || got.Dates[0] != "2024-01-13" || got.Dates[2] != "2024-01-15" {
		t.Fatalf("dates %v", got.Dates)
	}
	if len(got.Datasets) != 3 || got.Datasets[0].ID != 0 || got.Datasets[0].Name != "Total" {
		t.Fatalf("datasets %+v", got.Datasets)
	}
	wantTotal := []string{"10", "0", "5"}
	for i, m := range got.Datasets[0].Data {
		wantMoney(t, "total "+got.Dates[i], m, usd(e.moneyEnv, wantTotal[i]))
	}
	for _, ds := range got.Datasets[1:] {
		if len(ds.Data) != len(got.Dates) {
			t.Errorf("%s has %d values for %d dates", ds.Name, len(ds.Data), len(got.Dates))
		}
		if ds.Name == "Fun" {
			wantMoney(t, "fun on the 15th", ds.Data[2], usd(e.moneyEnv, "4"))
			wantMoney(t, "fun on the 13th", ds.Data[0], usd(e.moneyEnv, "0"))
		}
	}
	income := (Service{DB: e.db}).TxDynamics(e.ctx, janRange, "income", "day")
	if got.Datasets[0].Color == income.Datasets[0].Color {
		t.Error("income and expense lines must differ in colour")
	}
}

func TestTxTopLargestFirstInBase(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "10", "2024-01-13")
	e.add(t, "expense", e.jpyAcc.ID, nil, "5000", "2024-01-14") // 50 USD, no category
	e.add(t, "expense", e.usdAcc.ID, &e.fun.ID, "30", "2024-01-15")

	svc := Service{DB: e.db}
	got := svc.TxTop(e.ctx, janRange, "expense", 2)
	if len(got.Items) != 2 {
		t.Fatalf("limit not honoured: %+v", got.Items)
	}
	wantMoney(t, "first", got.Items[0].Amount, usd(e.moneyEnv, "50"))
	wantMoney(t, "second", got.Items[1].Amount, usd(e.moneyEnv, "30"))
	if got.Items[0].Category != nil || got.Items[0].Account.Name != "jpy" {
		t.Errorf("the yen expense has no category: %+v", got.Items[0])
	}
	if c := got.Items[1].Category; c == nil || c.Name != "Fun" {
		t.Errorf("category %+v", c)
	}
	if def := svc.TxTop(e.ctx, janRange, "expense", 0); len(def.Items) != 3 {
		t.Errorf("a missing limit means the default of ten: %d items", len(def.Items))
	}
}

func TestFiltersNarrowEveryReport(t *testing.T) {
	e := newSeriesEnv(t)
	tag, err := (ledger.Tags{DB: e.db}).Create(e.ctx, "trip")
	if err != nil {
		t.Fatal(err)
	}
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "10", "2024-01-14")
	e.add(t, "expense", e.jpyAcc.ID, &e.food.ID, "1000", "2024-01-14") // 10 USD
	e.add(t, "expense", e.usdAcc.ID, &e.fun.ID, "3", "2024-01-14")
	date := "2024-01-14"
	tagged, err := e.txs.Create(
		e.ctx,
		ledger.TxInput{Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("4"), Date: &date, TagIDs: []int64{tag.ID}},
	)
	if err != nil || tagged == nil {
		t.Fatal(err)
	}

	svc := Service{DB: e.db}
	total := func(f Filter) money.Money {
		f.PeriodType, f.StartDate, f.EndDate = "custom", "2024-01-13", "2024-01-15"
		return svc.TxSummary(e.ctx, f, "expense").Total
	}
	wantMoney(t, "no filter", total(Filter{}), usd(e.moneyEnv, "27"))
	wantMoney(t, "one account", total(Filter{AccountIDs: []int64{e.jpyAcc.ID}}), usd(e.moneyEnv, "10"))
	wantMoney(t, "one category", total(Filter{CategoryIDs: []int64{e.fun.ID}}), usd(e.moneyEnv, "3"))
	wantMoney(t, "one tag", total(Filter{TagIDs: []int64{tag.ID}}), usd(e.moneyEnv, "4"))
	wantMoney(
		t,
		"account and category",
		total(Filter{AccountIDs: []int64{e.usdAcc.ID}, CategoryIDs: []int64{e.food.ID}}),
		usd(e.moneyEnv, "10"),
	)
}

func TestHeatmapCoversEveryDayFindsPeak(t *testing.T) {
	e := newSeriesEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "10", "2024-01-13")
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "5", "2024-01-15")
	e.add(t, "expense", e.usdAcc.ID, &e.fun.ID, "7", "2024-01-15")

	got := (Service{DB: e.db}).Heatmap(e.ctx, janRange)
	if len(got.Items) != 3 {
		t.Fatalf("days %+v", got.Items)
	}
	wantMoney(t, "peak", got.Max, usd(e.moneyEnv, "12"))
	if last := got.Items[2]; last.Count != 2 {
		t.Errorf("transactions on the 15th: %d", last.Count)
	}
	wantMoney(t, "an empty day", got.Items[1].Value, usd(e.moneyEnv, "0"))
}

func TestReportsEmptySpaceEmptyNotBroken(t *testing.T) {
	svc := Service{DB: openSpaceDB(t)}
	ctx := context.Background()
	if got := svc.Overview(ctx, janRange); got.Currency != "" {
		t.Errorf("without a currency there is no base: %+v", got)
	}
	if got := svc.TxTop(ctx, janRange, "expense", 5); len(got.Items) != 0 {
		t.Errorf("items %+v", got.Items)
	}
	if got := svc.CashFlowOverTime(ctx, janRange, "day"); len(got.Items) != 3 {
		t.Errorf("one point per day even without data: %d", len(got.Items))
	}
}

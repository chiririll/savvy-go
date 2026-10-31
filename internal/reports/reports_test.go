package reports

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/ledger"
	"savvy-go/internal/migrate"
	"savvy-go/internal/money"
)

var janRange = Filter{PeriodType: "custom", StartDate: "2024-01-13", EndDate: "2024-01-15"}

func TestNetWorthConvertsEachAccountBase(t *testing.T) {
	e := newMoneyEnv(t)
	got := (Service{DB: e.db}).NetWorth(e.ctx, Filter{})
	// USD 100 + JPY 3000*0.01 + BTC 0.
	if total := got.Current; !total.Equal(money.FromDecimal(dec("130"), e.usd.Unit())) {
		t.Fatalf("current = %s (%+v), want 130 USD", total, total.Unit())
	}
	accounts := got.Accounts
	if len(accounts) != 3 || accounts[0].Name != "usd" || accounts[1].Name != "jpy" {
		t.Fatalf("accounts must be largest first: %v", accounts)
	}
	if pct := accounts[0].Percentage; pct != 76.9 {
		t.Fatalf("usd share = %v, want 76.9", pct)
	}
}

func TestTxSummaryAveragesRoundBaseScale(t *testing.T) {
	e := newMoneyEnv(t)
	e.tx(t, "expense", e.usdAcc.ID, "10.00")
	got := (Service{DB: e.db}).TxSummary(e.ctx, janRange, "expense")
	base := e.usd.Unit()
	gotAmounts := map[string]money.Money{"total": got.Total, "avgPerDay": got.AvgPerDay, "avgPerWeek": got.AvgPerWeek}
	for key, want := range map[string]string{"total": "10", "avgPerDay": "3.33", "avgPerWeek": "23.33"} {
		if m := gotAmounts[key]; !m.Equal(money.FromDecimal(dec(want), base)) {
			t.Errorf("%s = %s, want %s", key, m, want)
		}
	}
	if got.DaysInPeriod != 3 {
		t.Errorf("daysInPeriod = %v", got.DaysInPeriod)
	}
}

// With JPY (no decimals) as the base currency every report amount is a whole
// number of yen; nothing may be forced to two decimals.
func TestReportsFollowTheBaseCurrencyScale(t *testing.T) {
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	ctx := context.Background()
	if err := migrate.Space.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	curs := ledger.Currencies{DB: sqlDB}
	jpy, err := curs.Create(ctx, ledger.Currency{Code: "JPY", Name: "JPY", Symbol: "J", Decimals: 0, Rate: dec("1")})
	if err != nil {
		t.Fatal(err)
	}
	usd, err := curs.Create(ctx, ledger.Currency{Code: "USD", Name: "USD", Symbol: "$", Decimals: 2, Rate: dec("150")})
	if err != nil {
		t.Fatal(err)
	}
	accts := ledger.Accounts{DB: sqlDB}
	yen, _ := accts.Create(
		ctx,
		ledger.AccountInput{Name: "yen", Type: "cash", CurrencyID: jpy.ID, InitialBalance: dec("5000"), IsActive: true},
	)
	dollars, _ := accts.Create(
		ctx,
		ledger.AccountInput{Name: "usd", Type: "cash", CurrencyID: usd.ID, InitialBalance: dec("10"), IsActive: true},
	)
	date := "2024-01-15"
	txs := ledger.Transactions{DB: sqlDB}
	for _, in := range []ledger.TxInput{
		{Type: "expense", AccountID: yen.ID, Amount: dec("1000"), Date: &date},
		{Type: "expense", AccountID: dollars.ID, Amount: dec("1.00"), Date: &date}, // 150 JPY
	} {
		if _, err := txs.Create(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	reports := Service{DB: sqlDB}
	base := jpy.Unit()

	got := reports.TxSummary(ctx, janRange, "expense")
	if total := got.Total; !total.Equal(money.FromDecimal(dec("1150"), base)) {
		t.Fatalf("total = %s (%+v), want 1150 JPY", total, total.Unit())
	}
	if avg := got.AvgPerDay; avg.Minor() != 383 { // 1150 / 3, whole yen
		t.Fatalf("avgPerDay = %s, want 383", avg)
	}

	// Net worth: 5000 - 1000 JPY plus (10 - 1) USD at 150 = 1350 JPY.
	if nw := reports.NetWorth(ctx, Filter{}).Current; nw.Minor() != 5350 {
		t.Fatalf("net worth = %s, want 5350 JPY", nw)
	}
}

func TestMoneyFlowSplitsIncomeAcrossExpenses(t *testing.T) {
	e := newMoneyEnv(t)
	cats := ledger.Categories{DB: e.db}
	salary, _ := cats.Create(e.ctx, ledger.Category{Name: "Salary", Type: "income"})
	food, _ := cats.Create(e.ctx, ledger.Category{Name: "Food", Type: "expense"})
	date := "2024-01-15"
	for _, in := range []ledger.TxInput{
		{Type: "income", AccountID: e.usdAcc.ID, CategoryID: &salary.ID, Amount: dec("100"), Date: &date},
		{Type: "expense", AccountID: e.usdAcc.ID, CategoryID: &food.ID, Amount: dec("25"), Date: &date},
	} {
		if _, err := e.txs.Create(e.ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	got := (Service{DB: e.db}).MoneyFlow(e.ctx, janRange)
	totals := map[string]money.Money{
		"income":   got.Totals.Income,
		"expenses": got.Totals.Expenses,
		"savings":  got.Totals.Savings,
	}
	base := e.usd.Unit()
	for key, want := range map[string]string{"income": "100", "expenses": "25", "savings": "75"} {
		if m := totals[key]; !m.Equal(money.FromDecimal(dec(want), base)) {
			t.Errorf("%s = %s, want %s", key, m, want)
		}
	}
	links := got.Links
	if len(links) != 2 {
		t.Fatalf("links = %v, want salary->food and salary->savings", links)
	}
	var sum money.Money = money.Zero(base)
	for _, l := range links {
		sum = sum.Add(l.Value)
	}
	if !sum.Equal(money.FromDecimal(dec("100"), base)) {
		t.Fatalf("links add up to %s, want the whole income", sum)
	}
}

func TestExpensePaceBudgetConvertedToBase(t *testing.T) {
	e := newMoneyEnv(t)
	if _, err := (ledger.Budgets{DB: e.db}).Create(e.ctx, ledger.BudgetInput{
		Name: "yen budget", Amount: dec("10000"), CurrencyID: &e.jpy.ID, Period: "monthly", IsGlobal: ptrTo(true),
	}); err != nil {
		t.Fatal(err)
	}
	got := (Service{DB: e.db}).ExpensePace(e.ctx, janRange)
	months := got.Months
	if len(months) == 0 {
		t.Fatal("no months")
	}
	// 10000 JPY at 0.01 USD per JPY.
	if b := months[0].Budget; b == nil || !b.Equal(money.FromDecimal(dec("100"), e.usd.Unit())) {
		t.Fatalf("budget = %v, want 100 USD", b)
	}
}

func TestOverviewSavingsRate(t *testing.T) {
	e := newMoneyEnv(t)
	cats := ledger.Categories{DB: e.db}
	salary, _ := cats.Create(e.ctx, ledger.Category{Name: "Salary", Type: "income"})
	date := "2024-01-15"
	if _,
		err := e.txs.Create(e.ctx,
		ledger.TxInput{Type: "income", AccountID: e.usdAcc.ID, CategoryID: &salary.ID, Amount: dec("200"),
			Date: &date}); err != nil {
		t.Fatal(err)
	}
	e.tx(t, "expense", e.usdAcc.ID, "50")
	got := (Service{DB: e.db}).Overview(e.ctx, janRange)
	rate := got.SavingsRate.Value
	if rate != 75 {
		t.Fatalf("savings rate = %v, want 75", rate)
	}
	net := got.NetCashFlow.Value
	if !net.Equal(money.FromDecimal(dec("150"), e.usd.Unit())) {
		t.Fatalf("net = %s, want 150", net)
	}
}

// An empty period must give empty lists, never nil: Go encodes a nil slice as
// JSON null, which the dashboard cannot read.
func TestByCategoryEmptyPeriodIsNotNil(t *testing.T) {
	e := newMoneyEnv(t)
	q := (Service{DB: e.db}).open(e.ctx, janRange)
	if got := q.byCategory("expense", janRange.Range(time.Now())); got == nil {
		t.Fatal("byCategory = nil, want an empty slice")
	}
}

func TestFilterValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		f    Filter
		ok   bool
	}{
		{"no dates is the default period", Filter{PeriodType: "custom"}, true},
		{"other period types are not checked", Filter{PeriodType: "month", StartDate: "nope"}, true},
		{"valid range", Filter{PeriodType: "custom", StartDate: "2025-01-01", EndDate: "2025-01-31"}, true},
		{"one day", Filter{PeriodType: "custom", StartDate: "2025-01-01", EndDate: "2025-01-01"}, true},
		{"end before start", Filter{PeriodType: "custom", StartDate: "2025-02-01", EndDate: "2025-01-31"}, false},
		{"unreadable start", Filter{PeriodType: "custom", StartDate: "yesterday", EndDate: "2025-01-31"}, false},
		{"only one date", Filter{PeriodType: "custom", StartDate: "2025-01-01"}, false},
	} {
		if got := tc.f.Validate() == nil; got != tc.ok {
			t.Errorf("%s: valid = %v, want %v", tc.name, got, tc.ok)
		}
	}
}

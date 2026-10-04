package domain

import (
	"context"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
	"savvy-go/internal/money"
)

var janRange = ReportFilter{PeriodType: "custom", StartDate: "2024-01-13", EndDate: "2024-01-15"}

func asMoney(t *testing.T, v any) money.Money {
	t.Helper()
	m, ok := v.(money.Money)
	if !ok {
		t.Fatalf("%v (%T) is not money.Money", v, v)
	}
	return m
}

func TestNetWorthConvertsEachAccountToBase(t *testing.T) {
	e := newMoneyEnv(t)
	got := (Reports{DB: e.db}).NetWorth(e.ctx, ReportFilter{})
	// USD 100 + JPY 3000*0.01 + BTC 0.
	if total := asMoney(t, got["current"]); !total.Equal(money.FromDecimal(dec("130"), e.usd.Unit())) {
		t.Fatalf("current = %s (%+v), want 130 USD", total, total.Unit())
	}
	accounts := got["accounts"].([]map[string]any)
	if len(accounts) != 3 || accounts[0]["name"] != "usd" || accounts[1]["name"] != "jpy" {
		t.Fatalf("accounts must be largest first: %v", accounts)
	}
	if pct := accounts[0]["percentage"].(float64); pct != 76.9 {
		t.Fatalf("usd share = %v, want 76.9", pct)
	}
}

func TestTxSummaryAveragesRoundToBaseScale(t *testing.T) {
	e := newMoneyEnv(t)
	e.tx(t, "expense", e.usdAcc.ID, "10.00")
	got := (Reports{DB: e.db}).TxSummary(e.ctx, janRange, "expense")
	base := e.usd.Unit()
	for key, want := range map[string]string{"total": "10", "avgPerDay": "3.33", "avgPerWeek": "23.33"} {
		if m := asMoney(t, got[key]); !m.Equal(money.FromDecimal(dec(want), base)) {
			t.Errorf("%s = %s, want %s", key, m, want)
		}
	}
	if got["daysInPeriod"] != 3 {
		t.Errorf("daysInPeriod = %v", got["daysInPeriod"])
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
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	curs := Currencies{DB: sqlDB}
	jpy, err := curs.Create(ctx, Currency{Code: "JPY", Name: "JPY", Symbol: "J", Decimals: 0, Rate: dec("1")})
	if err != nil {
		t.Fatal(err)
	}
	usd, err := curs.Create(ctx, Currency{Code: "USD", Name: "USD", Symbol: "$", Decimals: 2, Rate: dec("150")})
	if err != nil {
		t.Fatal(err)
	}
	accts := Accounts{DB: sqlDB}
	yen, _ := accts.Create(ctx, AccountInput{Name: "yen", Type: "cash", CurrencyID: jpy.ID, InitialBalance: dec("5000"), IsActive: true})
	dollars, _ := accts.Create(ctx, AccountInput{Name: "usd", Type: "cash", CurrencyID: usd.ID, InitialBalance: dec("10"), IsActive: true})
	date := "2024-01-15"
	txs := Transactions{DB: sqlDB}
	for _, in := range []TxInput{
		{Type: "expense", AccountID: yen.ID, Amount: dec("1000"), Date: &date},
		{Type: "expense", AccountID: dollars.ID, Amount: dec("1.00"), Date: &date}, // 150 JPY
	} {
		if _, err := txs.Create(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	reports := Reports{DB: sqlDB}
	base := jpy.Unit()

	got := reports.TxSummary(ctx, janRange, "expense")
	if total := asMoney(t, got["total"]); !total.Equal(money.FromDecimal(dec("1150"), base)) {
		t.Fatalf("total = %s (%+v), want 1150 JPY", total, total.Unit())
	}
	if avg := asMoney(t, got["avgPerDay"]); avg.Minor() != 383 { // 1150 / 3, whole yen
		t.Fatalf("avgPerDay = %s, want 383", avg)
	}

	// Net worth: 5000 - 1000 JPY plus (10 - 1) USD at 150 = 1350 JPY.
	if nw := asMoney(t, reports.NetWorth(ctx, ReportFilter{})["current"]); nw.Minor() != 5350 {
		t.Fatalf("net worth = %s, want 5350 JPY", nw)
	}
}

func TestMoneyFlowSplitsIncomeAcrossExpenses(t *testing.T) {
	e := newMoneyEnv(t)
	cats := Categories{DB: e.db}
	salary, _ := cats.Create(e.ctx, Category{Name: "Salary", Type: "income"})
	food, _ := cats.Create(e.ctx, Category{Name: "Food", Type: "expense"})
	date := "2024-01-15"
	for _, in := range []TxInput{
		{Type: "income", AccountID: e.usdAcc.ID, CategoryID: &salary.ID, Amount: dec("100"), Date: &date},
		{Type: "expense", AccountID: e.usdAcc.ID, CategoryID: &food.ID, Amount: dec("25"), Date: &date},
	} {
		if _, err := e.txs.Create(e.ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	got := (Reports{DB: e.db}).MoneyFlow(e.ctx, janRange)
	totals := got["totals"].(map[string]any)
	base := e.usd.Unit()
	for key, want := range map[string]string{"income": "100", "expenses": "25", "savings": "75"} {
		if m := asMoney(t, totals[key]); !m.Equal(money.FromDecimal(dec(want), base)) {
			t.Errorf("%s = %s, want %s", key, m, want)
		}
	}
	links := got["links"].([]map[string]any)
	if len(links) != 2 {
		t.Fatalf("links = %v, want salary->food and salary->savings", links)
	}
	var sum money.Money = money.Zero(base)
	for _, l := range links {
		sum = sum.Add(asMoney(t, l["value"]))
	}
	if !sum.Equal(money.FromDecimal(dec("100"), base)) {
		t.Fatalf("links add up to %s, want the whole income", sum)
	}
}

func TestExpensePaceBudgetIsConvertedToBase(t *testing.T) {
	e := newMoneyEnv(t)
	if _, err := (Budgets{DB: e.db}).Create(e.ctx, BudgetInput{
		Name: "yen budget", Amount: dec("10000"), CurrencyID: &e.jpy.ID, Period: "monthly", IsGlobal: ptrTo(true),
	}); err != nil {
		t.Fatal(err)
	}
	got := (Reports{DB: e.db}).ExpensePace(e.ctx, janRange)
	months := got["months"].([]map[string]any)
	if len(months) == 0 {
		t.Fatal("no months")
	}
	// 10000 JPY at 0.01 USD per JPY.
	if b := asMoney(t, months[0]["budget"]); !b.Equal(money.FromDecimal(dec("100"), e.usd.Unit())) {
		t.Fatalf("budget = %s (%+v), want 100 USD", b, b.Unit())
	}
}

func TestOverviewSavingsRate(t *testing.T) {
	e := newMoneyEnv(t)
	cats := Categories{DB: e.db}
	salary, _ := cats.Create(e.ctx, Category{Name: "Salary", Type: "income"})
	date := "2024-01-15"
	if _, err := e.txs.Create(e.ctx, TxInput{Type: "income", AccountID: e.usdAcc.ID, CategoryID: &salary.ID, Amount: dec("200"), Date: &date}); err != nil {
		t.Fatal(err)
	}
	e.tx(t, "expense", e.usdAcc.ID, "50")
	got := (Reports{DB: e.db}).Overview(e.ctx, janRange)
	rate := got["savingsRate"].(map[string]any)["value"].(float64)
	if rate != 75 {
		t.Fatalf("savings rate = %v, want 75", rate)
	}
	net := asMoney(t, got["netCashFlow"].(map[string]any)["value"])
	if !net.Equal(money.FromDecimal(dec("150"), e.usd.Unit())) {
		t.Fatalf("net = %s, want 150", net)
	}
}

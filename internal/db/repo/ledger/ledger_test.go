package ledger_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/dbtest"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

var ctx = context.Background()

// fixture is a space with a base currency (USD), a yen account and a few
// transactions.
type fixture struct {
	repo      ledgerrepo.Repo
	usd, jpy  int64
	cash, yen int64
	food      int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	r := ledgerrepo.Repo{DB: dbtest.Space(t)}
	now := time.Now().UTC()
	must := func(id int64, err error) int64 {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	f := &fixture{repo: r}
	f.usd = must(
		r.InsertCurrency(
			ctx,
			ledgerrepo.Currency{
				Code:     "USD",
				Name:     "Dollar",
				Symbol:   "$",
				Decimals: 2,
				IsBase:   true,
				Rate:     decimal.NewFromInt(1),
			},
			now,
		),
	)
	f.jpy = must(
		r.InsertCurrency(
			ctx,
			ledgerrepo.Currency{Code: "JPY", Name: "Yen", Symbol: "¥", Decimals: 0, Rate: decimal.RequireFromString("0.0066")},
			now,
		),
	)
	f.cash = must(
		r.InsertAccount(
			ctx,
			ledgerrepo.AccountFields{
				Name:           "Cash",
				Type:           "cash",
				CurrencyID:     f.usd,
				InitialBalance: money.New(10000, money.Unit{ID: f.usd, Decimals: 2}),
				IsActive:       true,
			},
			now,
		),
	)
	f.yen = must(
		r.InsertAccount(
			ctx,
			ledgerrepo.AccountFields{
				Name:           "Yen",
				Type:           "cash",
				CurrencyID:     f.jpy,
				InitialBalance: money.New(0, money.Unit{ID: f.jpy}),
				IsActive:       true,
			},
			now,
		),
	)
	f.food = must(r.InsertCategory(ctx, ledgerrepo.Category{Name: "Food", Type: "expense"}, now))
	return f
}

func (f *fixture) expense(t *testing.T, account int64, minor int64, date string) int64 {
	t.Helper()
	acc, _ := f.repo.Account(ctx, account)
	d := date
	id, err := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{
		Type: "expense", Status: "confirmed", AccountID: account, CategoryID: &f.food,
		Amount: money.New(minor, acc.Currency.Unit()), Date: &d,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestMissingRowsAreNil(t *testing.T) {
	f := newFixture(t)
	if a, err := f.repo.Account(ctx, 999); a != nil || err != nil {
		t.Fatalf("account: %v, %v", a, err)
	}
	if c, err := f.repo.Category(ctx, 999); c != nil || err != nil {
		t.Fatalf("category: %v, %v", c, err)
	}
	if c, err := f.repo.CurrencyByCode(ctx, "XXX"); c != nil || err != nil {
		t.Fatalf("currency: %v, %v", c, err)
	}
	if tx, err := f.repo.Transaction(ctx, 999); tx != nil || err != nil {
		t.Fatalf("transaction: %v, %v", tx, err)
	}
	if u, err := f.repo.AccountUnit(ctx, 999); u != nil || err != nil {
		t.Fatalf("unit: %v, %v", u, err)
	}
}

func TestAccountsCarryTheirCurrencyUnit(t *testing.T) {
	f := newFixture(t)
	a, _ := f.repo.Account(ctx, f.yen)
	if a.Currency.Code != "JPY" || a.Currency.Decimals != 0 ||
		a.InitialBalance.Unit() != (money.Unit{ID: f.jpy, Decimals: 0}) {
		t.Fatalf("yen account = %+v", a)
	}
	if u, _ := f.repo.AccountUnit(ctx, f.cash); u == nil || *u != (money.Unit{ID: f.usd, Decimals: 2}) {
		t.Fatalf("unit = %v", u)
	}
	list, _ := f.repo.Accounts(ctx, ledgerrepo.AccountFilter{})
	if len(list) != 2 {
		t.Fatalf("%d accounts", len(list))
	}
	base, _ := f.repo.BaseCurrency(ctx)
	if base == nil || base.Code != "USD" || !base.IsBase {
		t.Fatalf("base = %+v", base)
	}
}

func TestTransactionAmountsKeepTheirUnits(t *testing.T) {
	f := newFixture(t)
	id := f.expense(t, f.yen, 1500, "2026-01-02")
	tx, _ := f.repo.Transaction(ctx, id)
	if tx.Amount.Minor() != 1500 || tx.Amount.Unit().Decimals != 0 || tx.ToAmount != nil || tx.Date == nil ||
		*tx.Date != "2026-01-02" {
		t.Fatalf("tx = %+v", tx)
	}
	changes, _ := f.repo.DailyChanges(ctx, f.yen, "")
	if len(changes) != 1 || changes[0].Day != "2026-01-02" || changes[0].Delta != -1500 {
		t.Fatalf("changes = %+v", changes)
	}
	// The as-of bound excludes later days.
	if early, _ := f.repo.DailyChanges(ctx, f.yen, "2026-01-01"); len(early) != 0 {
		t.Fatalf("changes before the transaction = %+v", early)
	}
}

func TestTagsOfTransactionsReplacement(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	a, _ := f.repo.InsertTag(ctx, "a", now)
	b, _ := f.repo.InsertTag(ctx, "b", now)
	tx := f.expense(t, f.cash, 100, "2026-01-02")

	if err := f.repo.SetTags(ctx, tx, []int64{a, b}); err != nil {
		t.Fatal(err)
	}
	got, _ := f.repo.TagsOf(ctx, []int64{tx})
	if len(got) != 2 || got[0].TransactionID != tx {
		t.Fatalf("tags = %+v", got)
	}
	if err := f.repo.SetTags(ctx, tx, []int64{b}); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.repo.TagsOf(ctx, []int64{tx}); len(got) != 1 || got[0].ID != b {
		t.Fatalf("after replacing = %+v", got)
	}
	if tag, _ := f.repo.Tag(ctx, b); tag.TransactionsCount != 1 {
		t.Fatalf("count = %d", tag.TransactionsCount)
	}
}

func TestReplacingACategoryMovesItsTransactions(t *testing.T) {
	f := newFixture(t)
	other, _ := f.repo.InsertCategory(ctx, ledgerrepo.Category{Name: "Other", Type: "expense"}, time.Now().UTC())
	tx := f.expense(t, f.cash, 100, "2026-01-02")
	if err := f.repo.ReplaceCategory(ctx, f.food, other); err != nil {
		t.Fatal(err)
	}
	if c, _ := f.repo.Category(ctx, f.food); c != nil {
		t.Fatal("the replaced category is still there")
	}
	if got, _ := f.repo.Transaction(ctx, tx); got.CategoryID == nil || *got.CategoryID != other {
		t.Fatalf("category = %v", got.CategoryID)
	}
}

func TestBudgetsKeepTheirCurrencyScope(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	pct := 80
	id, err := f.repo.InsertBudget(ctx, ledgerrepo.BudgetFields{
		Name: "Food", Period: "monthly", Amount: money.New(50000, money.Unit{ID: f.usd, Decimals: 2}),
		IsGlobal: false, IsActive: true, NotifyAtPercent: &pct,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SetBudgetCategories(ctx, id, []int64{f.food}); err != nil {
		t.Fatal(err)
	}
	list, _ := f.repo.Budgets(ctx, &id)
	if len(list) != 1 || list[0].Amount.Minor() != 50000 || list[0].NotifyAtPercent == nil ||
		*list[0].NotifyAtPercent != 80 {
		t.Fatalf("budgets = %+v", list)
	}
	if cats, _ := f.repo.BudgetCategories(ctx, id); len(cats) != 1 || cats[0].ID != f.food {
		t.Fatalf("categories = %+v", cats)
	}

	monthly, _ := f.repo.MonthlyBudgets(ctx)
	if len(monthly) != 1 || !monthly[0].IsBase {
		t.Fatalf("monthly = %+v", monthly)
	}
	if f.repo.GlobalMonthlyBudget(ctx) != nil {
		t.Fatal("a scoped budget is not the global one")
	}
	// USD is used by an account and the budget, JPY by an account; a currency
	// nothing refers to is free to delete.
	spare, _ := f.repo.InsertCurrency(
		ctx,
		ledgerrepo.Currency{Code: "EUR", Name: "Euro", Symbol: "E", Decimals: 2, Rate: decimal.NewFromInt(1)},
		now,
	)
	if !f.repo.CurrencyInUse(ctx, f.usd) || !f.repo.CurrencyInUse(ctx, f.jpy) || f.repo.CurrencyInUse(ctx, spare) {
		t.Fatal("CurrencyInUse must be true for USD and JPY and false for a spare currency")
	}
}

func TestCategoryStatisticsGroupsByCurrency(t *testing.T) {
	f := newFixture(t)
	f.expense(t, f.cash, 1000, "2026-01-02")
	f.expense(t, f.yen, 500, "2026-01-03")
	stats := f.repo.CategoryStatistics(ctx, f.food, "", "")
	if len(stats) != 2 {
		t.Fatalf("stats = %+v", stats)
	}
	var total int
	for _, s := range stats {
		total += s.Count
	}
	if total != 2 {
		t.Fatalf("count = %d", total)
	}
}

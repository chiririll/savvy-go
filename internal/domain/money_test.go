package domain

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/filter"
	"savvy-go/internal/migrate"
)

type moneyEnv struct {
	ctx            context.Context
	db             *sql.DB
	usd, jpy, btc  Currency
	usdAcc, jpyAcc Account
	btcAcc         Account
	txs            Transactions
}

func newMoneyEnv(t *testing.T) *moneyEnv {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	ctx := context.Background()
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	env := &moneyEnv{ctx: ctx, db: sqlDB, txs: Transactions{DB: sqlDB}}
	curs := Currencies{DB: sqlDB}
	mk := func(code string, decimals int, rate string) Currency {
		c, err := curs.Create(ctx, Currency{Code: code, Name: code, Symbol: code, Decimals: decimals, Rate: decimal.RequireFromString(rate)})
		if err != nil {
			t.Fatal(err)
		}
		return *c
	}
	env.usd = mk("USD", 2, "1")
	env.jpy = mk("JPY", 0, "0.01")
	env.btc = mk("BTC", 8, "50000")
	accts := Accounts{DB: sqlDB}
	acct := func(name string, cur Currency, initial string) Account {
		a, err := accts.Create(ctx, Account{Name: name, Type: "cash", CurrencyID: cur.ID, InitialBalance: decimal.RequireFromString(initial), IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		return *a
	}
	env.usdAcc = acct("usd", env.usd, "100")
	env.jpyAcc = acct("jpy", env.jpy, "3000")
	env.btcAcc = acct("btc", env.btc, "0")
	return env
}

func (e *moneyEnv) tx(t *testing.T, typ string, accountID int64, amount string) *Transaction {
	t.Helper()
	date := "2024-01-15"
	tx, err := e.txs.Create(e.ctx, TxInput{Type: typ, AccountID: accountID, Amount: decimal.RequireFromString(amount), Date: &date})
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestBalanceIsExactForFloatUnfriendlySums(t *testing.T) {
	e := newMoneyEnv(t)
	acct, err := (Accounts{DB: e.db}).Create(e.ctx, Account{Name: "cash2", Type: "cash", CurrencyID: e.usd.ID, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	e.tx(t, "income", acct.ID, "0.10")
	e.tx(t, "income", acct.ID, "0.20")
	got, err := (Accounts{DB: e.db}).ByID(e.ctx, acct.ID)
	if err != nil || got == nil {
		t.Fatal(err)
	}
	if !got.Balance.Equal(dec("0.3")) {
		t.Fatalf("balance = %s, want exactly 0.3", got.Balance)
	}
}

func TestAmountsAreStoredAsMinorUnitsPerCurrency(t *testing.T) {
	e := newMoneyEnv(t)
	btcTx := e.tx(t, "income", e.btcAcc.ID, "0.00012345")
	jpyTx := e.tx(t, "expense", e.jpyAcc.ID, "1500")
	e.tx(t, "expense", e.usdAcc.ID, "10.50")

	var btcRaw, jpyRaw int64
	if err := e.db.QueryRow(`SELECT amount FROM transactions WHERE id = ?`, btcTx.ID).Scan(&btcRaw); err != nil {
		t.Fatal(err)
	}
	if err := e.db.QueryRow(`SELECT amount FROM transactions WHERE id = ?`, jpyTx.ID).Scan(&jpyRaw); err != nil {
		t.Fatal(err)
	}
	if btcRaw != 12345 || jpyRaw != 1500 {
		t.Fatalf("raw amounts btc=%d jpy=%d, want 12345 and 1500", btcRaw, jpyRaw)
	}
	if !btcTx.Amount.Equal(dec("0.00012345")) || !jpyTx.Amount.Equal(dec("1500")) {
		t.Fatalf("read back btc=%s jpy=%s", btcTx.Amount, jpyTx.Amount)
	}

	base, _ := (Currencies{DB: e.db}).Base(e.ctx)
	sum := (Accounts{DB: e.db}).Summary(e.ctx, base)
	// USD 100-10.50=89.50, JPY 3000-1500=1500 -> 15.00, BTC 0.00012345*50000=6.1725
	if got := sum["total_balance"].(interface{ String() string }).String(); got != "110.67" {
		t.Fatalf("total_balance = %s, want 110.67", got)
	}
}

func TestCrossCurrencyReportSumsFoldToBase(t *testing.T) {
	e := newMoneyEnv(t)
	e.tx(t, "expense", e.usdAcc.ID, "10.00")
	e.tx(t, "expense", e.jpyAcc.ID, "500")
	e.tx(t, "expense", e.btcAcc.ID, "0.0001")

	w := filter.ReportWhere{Type: "expense", Start: "2024-01-01", End: "2024-01-31"}
	// 10.00 + 500*0.01 + 0.0001*50000 = 20
	if got := filter.SumByType(e.ctx, e.db, w); !got.Equal(dec("20")) {
		t.Fatalf("SumByType = %s, want 20", got)
	}
	days := filter.DailyTotals(e.ctx, e.db, w)
	if len(days) != 1 || !days[0].Total.Equal(dec("20")) || days[0].Count != 3 {
		t.Fatalf("DailyTotals = %+v", days)
	}
	top := filter.TopTransactions(e.ctx, e.db, w, 2)
	if len(top) != 2 || !top[0].Amount.Equal(dec("10")) || !top[1].Amount.Equal(dec("5")) {
		t.Fatalf("TopTransactions = %+v", top)
	}
}

func TestCategoryStatisticsFoldsCurrencies(t *testing.T) {
	e := newMoneyEnv(t)
	cats := Categories{DB: e.db}
	cat, err := cats.Create(e.ctx, Category{Name: "Food", Type: "expense"})
	if err != nil {
		t.Fatal(err)
	}
	date := "2024-01-15"
	for _, in := range []struct {
		acct   int64
		amount string
	}{{e.usdAcc.ID, "10.00"}, {e.jpyAcc.ID, "500"}} {
		if _, err := e.txs.Create(e.ctx, TxInput{Type: "expense", AccountID: in.acct, CategoryID: &cat.ID, Amount: dec(in.amount), Date: &date}); err != nil {
			t.Fatal(err)
		}
	}
	stats, err := cats.Statistics(e.ctx, cat.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if stats["transactions_count"] != 2 {
		t.Fatalf("count = %v", stats["transactions_count"])
	}
	if got := stats["total_amount"].(interface{ String() string }).String(); got != "15.00" {
		t.Fatalf("total_amount = %s, want 15.00", got)
	}
}

func TestItemTotalsAreExact(t *testing.T) {
	e := newMoneyEnv(t)
	date := "2024-01-15"
	tx, err := e.txs.Create(e.ctx, TxInput{
		Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("0.30"), Date: &date,
		Items: []TxItem{{Name: "gum", Quantity: dec("3"), PricePerUnit: dec("0.10")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Items) != 1 || !tx.Items[0].TotalPrice.Equal(dec("0.30")) || !tx.Items[0].Quantity.Equal(dec("3")) {
		t.Fatalf("items = %+v", tx.Items)
	}
}

func TestDebtIsPaidOffExactly(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	debts := Debts{Accounts: accts, Transactions: e.txs}
	debt, err := debts.Create(e.ctx, "Loan", "i_owe", e.usd.ID, 0, dec("0.30"), "", "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"0.10", "0.20"} {
		if _, err := debts.Payment(e.ctx, debt.ID, e.usdAcc.ID, dec(part), "2024-01-15", nil, false); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := accts.ByID(e.ctx, debt.ID)
	if got == nil || !got.Balance.IsZero() || !got.IsPaidOff {
		t.Fatalf("debt after full payment = %+v", got)
	}
}

func TestFormulaIsExact(t *testing.T) {
	if got := evaluateFormula("0.1 + 0.2", nil); !got.Equal(dec("0.3")) {
		t.Fatalf("0.1+0.2 = %s", got)
	}
	tx := &Transaction{Amount: dec("0.1")}
	if got := evaluateFormula("{{transaction.amount}} * 3", tx); !got.Equal(dec("0.3")) {
		t.Fatalf("amount*3 = %s", got)
	}
	if got := evaluateFormula("{{transaction.amount}} / 0", tx); !got.IsZero() {
		t.Fatalf("division by zero should yield 0, got %s", got)
	}
}

func TestCurrencyConversionIsExact(t *testing.T) {
	jpy := Currency{ID: 2, Rate: dec("0.01")}
	usd := Currency{ID: 1, IsBase: true, Rate: dec("1")}
	if got := Convert(dec("1500"), jpy, usd); !got.Equal(dec("15")) {
		t.Fatalf("JPY->USD = %s", got)
	}
	if got := Convert(dec("15"), usd, jpy); !got.Equal(dec("1500")) {
		t.Fatalf("USD->JPY = %s", got)
	}
}

func rawAmount(t *testing.T, e *moneyEnv, id int64) int64 {
	t.Helper()
	var v int64
	if err := e.db.QueryRow(`SELECT amount FROM transactions WHERE id = ?`, id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAccountCurrencyIsImmutable(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	eur, err := (Currencies{DB: e.db}).Create(e.ctx, Currency{Code: "EUR", Name: "EUR", Symbol: "E", Decimals: 2, Rate: dec("1.1")})
	if err != nil {
		t.Fatal(err)
	}
	a, err := accts.Create(e.ctx, Account{Name: "fixed", Type: "cash", CurrencyID: e.usd.ID, InitialBalance: dec("100"), IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	tx := e.tx(t, "expense", a.ID, "10.50")

	// Any other currency is rejected, even one with the same decimals.
	for _, other := range []Currency{*eur, e.btc, e.jpy} {
		cur, _ := accts.ByID(e.ctx, a.ID)
		cur.CurrencyID = other.ID
		if _, err := accts.Update(e.ctx, a.ID, *cur); !errors.Is(err, ErrAccountCurrencyImmutable) {
			t.Fatalf("switch to %s = %v, want ErrAccountCurrencyImmutable", other.Code, err)
		}
	}
	unchanged, _ := accts.ByID(e.ctx, a.ID)
	if unchanged.CurrencyID != e.usd.ID || rawAmount(t, e, tx.ID) != 1050 {
		t.Fatalf("rejected switch must change nothing: currency=%d raw=%d", unchanged.CurrencyID, rawAmount(t, e, tx.ID))
	}

	// Other fields still update when the currency is left as is.
	cur, _ := accts.ByID(e.ctx, a.ID)
	cur.Name = "renamed"
	got, err := accts.Update(e.ctx, a.ID, *cur)
	if err != nil || got.Name != "renamed" || got.CurrencyID != e.usd.ID {
		t.Fatalf("update without currency change: %+v, %v", got, err)
	}
}

func TestCurrencyDecimalsAreImmutable(t *testing.T) {
	e := newMoneyEnv(t)
	curs := Currencies{DB: e.db}
	eur, err := curs.Create(e.ctx, Currency{Code: "EUR", Name: "EUR", Symbol: "E", Decimals: 2, Rate: dec("1.1")})
	if err != nil {
		t.Fatal(err)
	}
	changed := *eur
	changed.Decimals = 4
	if _, err := curs.Update(e.ctx, eur.ID, changed); !errors.Is(err, ErrDecimalsImmutable) {
		t.Fatalf("change decimals of unused currency = %v, want ErrDecimalsImmutable", err)
	}
	renamed := *eur
	renamed.Name = "Euro"
	if got, err := curs.Update(e.ctx, eur.ID, renamed); err != nil || got.Name != "Euro" {
		t.Fatalf("update with same decimals: %v", err)
	}
}

func TestCurrencyUsedByBudgetCannotBeDeleted(t *testing.T) {
	e := newMoneyEnv(t)
	curs := Currencies{DB: e.db}
	eur, err := curs.Create(e.ctx, Currency{Code: "EUR", Name: "EUR", Symbol: "E", Decimals: 2, Rate: dec("1.1")})
	if err != nil {
		t.Fatal(err)
	}
	budget, err := (Budgets{DB: e.db}).Create(e.ctx, BudgetInput{Name: "b", Amount: dec("100"), CurrencyID: &eur.ID, Period: "monthly"})
	if err != nil {
		t.Fatal(err)
	}
	if err := curs.Delete(e.ctx, eur.ID); err == nil || err.Error() != "in use" {
		t.Fatalf("delete with budget = %v, want in use", err)
	}
	if err := (Budgets{DB: e.db}).Delete(e.ctx, budget.ID); err != nil {
		t.Fatal(err)
	}
	if err := curs.Delete(e.ctx, eur.ID); err != nil {
		t.Fatalf("delete when unused: %v", err)
	}
}

func TestUnknownAccountOrCurrencyIsAnError(t *testing.T) {
	e := newMoneyEnv(t)
	if _, err := (Accounts{DB: e.db}).Create(e.ctx, Account{Name: "ghost", Type: "cash", CurrencyID: 999, IsActive: true}); !errors.Is(err, ErrUnknownCurrency) {
		t.Fatalf("create with unknown currency = %v, want ErrUnknownCurrency", err)
	}
	date := "2024-01-15"
	if _, err := e.txs.Create(e.ctx, TxInput{Type: "expense", AccountID: 999, Amount: dec("1"), Date: &date}); !errors.Is(err, ErrUnknownAccount) {
		t.Fatalf("transaction on unknown account = %v, want ErrUnknownAccount", err)
	}
}

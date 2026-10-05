package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func (e *moneyEnv) txOn(t *testing.T, in TxInput, date string) *Transaction {
	t.Helper()
	in.Date = &date
	tx, err := e.txs.Create(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func (e *moneyEnv) balance(t *testing.T, id int64) decimal.Decimal {
	t.Helper()
	a, err := (Accounts{DB: e.db}).ByID(e.ctx, id)
	if err != nil || a == nil {
		t.Fatalf("account %d: %v", id, err)
	}
	return a.Balance.Decimal()
}

// Every transaction type moves the balance the way the ledger query says:
// income, debt collection and borrowing add, everything else takes, and a
// transfer adds its destination amount to the destination account.
func TestBalanceFollowsEveryTransactionType(t *testing.T) {
	e := newMoneyEnv(t)
	usd, jpy := e.usdAcc.ID, e.jpyAcc.ID // start at 100 USD and 3000 JPY
	debt, err := (Debts{Accounts: Accounts{DB: e.db}, Transactions: e.txs}).Create(
		e.ctx, "loan", "owed_to_me", e.usd.ID, 0, dec("1"), "", "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	in := func(typ string, amount string, to *int64, toAmount string) TxInput {
		x := TxInput{Type: typ, AccountID: usd, Amount: dec(amount), ToAccountID: to}
		if toAmount != "" {
			d := dec(toAmount)
			x.ToAmount = &d
		}
		return x
	}
	e.txOn(t, in("income", "50", nil, ""), "2024-01-10")               // +50
	e.txOn(t, in("expense", "20", nil, ""), "2024-01-11")              // -20
	e.txOn(t, in("transfer", "10", &jpy, "1000"), "2024-01-12")        // -10 USD, +1000 JPY
	e.txOn(t, in("debt_lend", "5", &debt.ID, "5"), "2024-01-13")       // -5
	e.txOn(t, in("debt_collection", "2", &debt.ID, "2"), "2024-01-14") // +2

	if got := e.balance(t, usd); !got.Equal(dec("117")) { // 100+50-20-10-5+2
		t.Errorf("usd = %s, want 117", got)
	}
	if got := e.balance(t, jpy); !got.Equal(dec("4000")) { // 3000+1000
		t.Errorf("jpy = %s, want 4000", got)
	}
}

func TestBalanceIgnoresPendingAndLaterTransactions(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	pending := "pending"
	date := "2024-02-01"
	if _, err := e.txs.Create(e.ctx, TxInput{Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("30"), Date: &date, Status: &pending}); err != nil {
		t.Fatal(err)
	}
	e.txOn(t, TxInput{Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("10")}, "2024-01-20")

	acct, _ := accts.ByID(e.ctx, e.usdAcc.ID)
	if got := acct.Balance.Decimal(); !got.Equal(dec("90")) {
		t.Errorf("balance = %s, want 90 (pending ignored)", got)
	}
	series, err := accts.BalanceSeries(e.ctx, *acct, []string{"2024-01-19", "2024-01-20", "2024-01-21"})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"100", "90", "90"} {
		if !series[i].Decimal().Equal(dec(want)) {
			t.Errorf("series[%d] = %s, want %s", i, series[i], want)
		}
	}
}

// A transfer from an account to itself both takes and adds, as it always has.
func TestSelfTransferNetsToItsAmountDifference(t *testing.T) {
	e := newMoneyEnv(t)
	self := e.usdAcc.ID
	d := dec("7")
	e.txOn(t, TxInput{Type: "transfer", AccountID: self, ToAccountID: &self, Amount: dec("10"), ToAmount: &d}, "2024-01-10")
	if got := e.balance(t, self); !got.Equal(dec("97")) { // 100 - 10 + 7
		t.Errorf("balance = %s, want 97", got)
	}
}

// The series is the running total of the same ledger, so it must agree with
// the balance computed date by date, in every currency.
func TestBalanceSeriesMatchesBalanceOnEveryDay(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	days := []string{"2024-03-01", "2024-03-02", "2024-03-03", "2024-03-04", "2024-03-05"}
	for i, day := range days {
		if i%2 == 0 {
			e.txOn(t, TxInput{Type: "income", AccountID: e.btcAcc.ID, Amount: dec("0.00012345")}, day)
		}
		e.txOn(t, TxInput{Type: "expense", AccountID: e.jpyAcc.ID, Amount: dec("150")}, day)
		e.txOn(t, TxInput{Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("1.10")}, day)
	}
	for _, ac := range []Account{e.usdAcc, e.jpyAcc, e.btcAcc} {
		a, _ := accts.ByID(e.ctx, ac.ID)
		series, err := accts.BalanceSeries(e.ctx, *a, days)
		if err != nil {
			t.Fatal(err)
		}
		for i, day := range days {
			want, err := accts.balance(e.ctx, *a, day)
			if err != nil {
				t.Fatal(err)
			}
			if !series[i].Equal(want) {
				t.Errorf("%s %s: series %s, balance %s", a.Name, day, series[i], want)
			}
		}
	}
}

func TestDebtBalanceDoesNotDependOnTheDate(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	debts := Debts{Accounts: accts, Transactions: e.txs}
	debt, err := debts.Create(e.ctx, "loan", "i_owe", e.usd.ID, 0, dec("100"), "", "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := debts.Payment(e.ctx, debt.ID, e.usdAcc.ID, dec("40"), "2024-01-15", nil, false); err != nil {
		t.Fatal(err)
	}
	a, _ := accts.ByID(e.ctx, debt.ID)
	series, err := accts.BalanceSeries(e.ctx, *a, []string{"2023-01-01", "2024-12-31"})
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range series {
		if !s.Decimal().Equal(dec("60")) {
			t.Errorf("series[%d] = %s, want 60 remaining", i, s)
		}
	}
}

func TestNewAccountsGetIncreasingSortOrders(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	a, err := accts.Create(e.ctx, AccountInput{Name: "first-after-seed", Type: "cash", CurrencyID: e.usd.ID, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	b, err := accts.Create(e.ctx, AccountInput{Name: "next", Type: "cash", CurrencyID: e.usd.ID, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if b.SortOrder != a.SortOrder+1 {
		t.Errorf("sort orders %d then %d, want consecutive", a.SortOrder, b.SortOrder)
	}
	// Debts have their own sequence, which starts at zero.
	d, err := accts.Create(e.ctx, AccountInput{Name: "d", Type: "debt", CurrencyID: e.usd.ID, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if d.SortOrder != 0 {
		t.Errorf("first debt sort order = %d, want 0", d.SortOrder)
	}
}

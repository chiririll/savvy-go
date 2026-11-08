package ledger

import (
	"errors"
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
	debt, err := newDebts(e).Create(
		e.ctx, DebtInput{Name: "loan", Direction: "owed_to_me", CurrencyID: e.usd.ID, Amount: dec("1")})
	if err != nil {
		t.Fatal(err)
	}
	debts := newDebts(e)
	e.txOn(t, simple("income", usd, "50"), "2024-01-10")                                                   // +50
	e.txOn(t, simple("expense", usd, "20"), "2024-01-11")                                                  // -20
	e.txOn(t, move(usd, jpy, "10", "1000"), "2024-01-12")                                                  // -10 USD, +1000 JPY
	if _, err := debts.record(e.ctx, "debt_lend", debt.ID, usd, dec("5"), "2024-01-13", nil); err != nil { // -5
		t.Fatal(err)
	}
	if _, err := debts.record(e.ctx, "debt_collection", debt.ID, usd, dec("2"), "2024-01-14", nil); err != nil { // +2
		t.Fatal(err)
	}

	if got := e.balance(t, usd); !got.Equal(dec("117")) { // 100+50-20-10-5+2
		t.Errorf("usd = %s, want 117", got)
	}
	if got := e.balance(t, jpy); !got.Equal(dec("4000")) { // 3000+1000
		t.Errorf("jpy = %s, want 4000", got)
	}
}

func TestBalanceIgnoresPendingLaterTransactions(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	pending := "pending"
	date := "2024-02-01"
	later := simple("expense", e.usdAcc.ID, "30")
	later.Date, later.Status = &date, &pending
	if _, err := e.txs.Create(e.ctx, later); err != nil {
		t.Fatal(err)
	}
	e.txOn(t, simple("expense", e.usdAcc.ID, "10"), "2024-01-20")

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

// A transfer from an account to itself would be one payment of a net amount.
// That shape exists only in converted v1.0 data; new input is refused.
func TestSelfTransferIsRefused(t *testing.T) {
	e := newMoneyEnv(t)
	net, date := dec("-3"), "2024-01-10"
	_, err := e.txs.Create(e.ctx, TxInput{
		Type: "transfer", Date: &date, Payments: []PaymentInput{{AccountID: e.usdAcc.ID, Amount: &net}},
	})
	if !errors.Is(err, ErrTransferSigns) {
		t.Fatalf("self transfer: %v, want %v", err, ErrTransferSigns)
	}
}

// The series is the running total of the same ledger, so it must agree with
// the balance computed date by date, in every currency.
func TestBalanceSeriesMatchesBalanceEveryDay(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	days := []string{"2024-03-01", "2024-03-02", "2024-03-03", "2024-03-04", "2024-03-05"}
	for i, day := range days {
		if i%2 == 0 {
			e.txOn(t, simple("income", e.btcAcc.ID, "0.00012345"), day)
		}
		e.txOn(t, simple("expense", e.jpyAcc.ID, "150"), day)
		e.txOn(t, simple("expense", e.usdAcc.ID, "1.10"), day)
	}
	for _, ac := range []Account{e.usdAcc, e.jpyAcc, e.btcAcc} {
		a, _ := accts.ByID(e.ctx, ac.ID)
		series, err := accts.BalanceSeries(e.ctx, *a, days)
		if err != nil {
			t.Fatal(err)
		}
		for i, day := range days {
			want, err := accts.Balance(e.ctx, *a, day)
			if err != nil {
				t.Fatal(err)
			}
			if !series[i].Equal(want) {
				t.Errorf("%s %s: series %s, balance %s", a.Name, day, series[i], want)
			}
		}
	}
}

func TestDebtBalanceIsSignedAndFollowsTheDate(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	debts := Debts{Accounts: accts, Transactions: e.txs}
	debt, err := debts.Create(e.ctx, DebtInput{Name: "loan", Direction: "i_owe", CurrencyID: e.usd.ID, Amount: dec("100")})
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
	// A debt is an account like any other: what I owe is negative, and a
	// payment moves it from the day it was made.
	for i, want := range []string{"-100", "-60"} {
		if !series[i].Decimal().Equal(dec(want)) {
			t.Errorf("series[%d] = %s, want %s", i, series[i], want)
		}
	}
}

func TestNewAccountsGetIncreasingSortOrders(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	a, err := accts.Create(
		e.ctx,
		AccountInput{Name: "first-after-seed", Type: "cash", CurrencyID: e.usd.ID, IsActive: true},
	)
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
	d, err := newDebts(e).Create(e.ctx, DebtInput{Name: "d", Direction: DebtIOwe, CurrencyID: e.usd.ID, Amount: dec("1")})
	if err != nil {
		t.Fatal(err)
	}
	if d.SortOrder != 0 {
		t.Errorf("first debt sort order = %d, want 0", d.SortOrder)
	}
}

// Add stores what Create does, and Bare reads an account or category without
// the figures that cost a pass over the ledger.
func TestAddMatchesCreateAndBareSkipsTotals(t *testing.T) {
	e := newMoneyEnv(t)
	date := "2024-02-01"
	in := simple("income", e.usdAcc.ID, "25")
	in.Date = &date
	id, err := e.txs.Add(e.ctx, in)
	if err != nil || id == 0 {
		t.Fatalf("Add = %d, %v", id, err)
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("125")) {
		t.Errorf("balance after Add = %s, want 125", got)
	}
	bare, err := (Accounts{DB: e.db}).Bare(e.ctx, e.usdAcc.ID)
	if err != nil || bare == nil || bare.Name != e.usdAcc.Name || !bare.Balance.IsZero() {
		t.Errorf("Bare = %+v, %v", bare, err)
	}
}

// AddMany stores a list in one go, in order, or none of it when one is refused.
func TestAddManyIsAllOrNone(t *testing.T) {
	e := newMoneyEnv(t)
	date := "2024-02-01"
	mk := func(account int64, amount string) TxInput {
		in := simple("income", account, amount)
		in.Date = &date
		return in
	}
	ids, err := e.txs.AddMany(e.ctx, []TxInput{mk(e.usdAcc.ID, "10"), mk(e.usdAcc.ID, "5")})
	if err != nil || len(ids) != 2 || ids[0] >= ids[1] {
		t.Fatalf("AddMany = %v, %v", ids, err)
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("115")) {
		t.Fatalf("balance = %s, want 115", got)
	}
	if _, err := e.txs.AddMany(e.ctx, []TxInput{mk(e.usdAcc.ID, "1"), mk(0, "1")}); err == nil {
		t.Fatal("a list with a bad transaction was accepted")
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("115")) {
		t.Fatalf("a refused list changed the balance to %s", got)
	}
}

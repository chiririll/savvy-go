package ledger

import (
	"testing"

	"savvy-go/internal/money"
)

func newDebts(e *moneyEnv) Debts {
	return Debts{Accounts: Accounts{DB: e.db}, Transactions: e.txs}
}

func balanceOf(t *testing.T, e *moneyEnv, id int64) money.Money {
	t.Helper()
	a, err := (Accounts{DB: e.db}).ByID(e.ctx, id)
	if err != nil || a == nil {
		t.Fatalf("account %d: %v", id, err)
	}
	return a.Balance
}

func TestLendingFromAccountRecordsIssuanceCurrency(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)

	// The caller names another currency; the source account decides.
	debt, err := debts.Create(e.ctx, DebtInput{
		Name: "Ann", DebtType: "owed_to_me", CurrencyID: e.usd.ID, AccountID: e.jpyAcc.ID,
		Amount: dec("1000"), Date: "2024-01-15", Origin: "new",
	})
	if err != nil {
		t.Fatal(err)
	}
	if debt.CurrencyID != e.jpy.ID {
		t.Errorf("debt currency %d, want the source account's %d", debt.CurrencyID, e.jpy.ID)
	}
	if !debt.Balance.Equal(money.FromDecimal(dec("1000"), e.jpy.Unit())) {
		t.Errorf("owed to me %s, want 1000", debt.Balance)
	}
	if got := balanceOf(t, e, e.jpyAcc.ID); got.Minor() != 2000 {
		t.Errorf("the money left the account: %s, want 2000", got)
	}
}

func TestBorrowingBringsMoneyIn(t *testing.T) {
	e := newMoneyEnv(t)
	_, err := newDebts(e).Create(e.ctx, DebtInput{
		Name: "Bank", DebtType: "i_owe", AccountID: e.usdAcc.ID, Amount: dec("50.25"), Date: "2024-01-15", Origin: "new",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := balanceOf(t, e, e.usdAcc.ID); !got.Equal(money.FromDecimal(dec("150.25"), e.usd.Unit())) {
		t.Errorf("account after borrowing %s, want 150.25", got)
	}
}

func TestDebtCannotBeFundedFromDebt(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	first, err := debts.Create(e.ctx, DebtInput{Name: "a", DebtType: "i_owe", CurrencyID: e.usd.ID, Amount: dec("10")})
	if err != nil {
		t.Fatal(err)
	}
	if _,
		err := debts.Create(e.ctx,
		DebtInput{Name: "b", DebtType: "i_owe", AccountID: first.ID, Amount: dec("5"), Date: "2024-01-15",
			Origin: "new"}); err == nil {
		t.Error("a debt cannot be the source of another")
	}
	if _,
		err := debts.Create(e.ctx,
		DebtInput{Name: "c", DebtType: "i_owe", AccountID: 9999, Amount: dec("5"), Date: "2024-01-15",
			Origin: "new"}); err == nil {
		t.Error("an unknown source account was accepted")
	}
}

func TestAnExistingDebtMovesNoMoney(t *testing.T) {
	e := newMoneyEnv(t)
	debt, err := newDebts(
		e,
	).Create(e.ctx,
		DebtInput{Name: "old", DebtType: "i_owe", CurrencyID: e.usd.ID, AccountID: e.usdAcc.ID, Amount: dec("40"),
			Origin: "existing"})
	if err != nil {
		t.Fatal(err)
	}
	if debt.Balance.Decimal().String() != "40" {
		t.Errorf("debt %s", debt.Balance)
	}
	if got := balanceOf(t, e, e.usdAcc.ID); got.Decimal().String() != "100" {
		t.Errorf("a debt that already existed leaves the account at %s", got)
	}
}

func TestPaymentsSettleDebtReopenRestoresIt(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	accts := Accounts{DB: e.db}
	debt, err := debts.Create(e.ctx, DebtInput{Name: "loan", DebtType: "i_owe", CurrencyID: e.usd.ID, Amount: dec("30")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := debts.Reopen(e.ctx, debt.ID); err == nil {
		t.Error("an open debt cannot be reopened")
	}

	if _, err := debts.Payment(e.ctx, debt.ID, e.usdAcc.ID, dec("10"), "2024-01-15", nil, false); err != nil {
		t.Fatal(err)
	}
	if d, _ := accts.ByID(e.ctx, debt.ID); d.IsPaidOff || d.Balance.Decimal().String() != "20" {
		t.Fatalf("after a part payment %+v", d)
	}
	if _, err := debts.Payment(e.ctx, debt.ID, e.usdAcc.ID, dec("20"), "2024-01-16", nil, false); err != nil {
		t.Fatal(err)
	}
	if d, _ := accts.ByID(e.ctx, debt.ID); !d.IsPaidOff || !d.Balance.IsZero() {
		t.Fatalf("after the last payment %+v", d)
	}
	if got := balanceOf(t, e, e.usdAcc.ID); got.Decimal().String() != "70" {
		t.Errorf("paid 30 out of 100: %s", got)
	}
	if _, err := debts.Payment(e.ctx, debt.ID, e.usdAcc.ID, dec("1"), "2024-01-17", nil, false); err == nil {
		t.Error("a paid-off debt takes no more payments")
	}

	reopened, err := debts.Reopen(e.ctx, debt.ID)
	if err != nil || reopened.IsPaidOff {
		t.Fatalf("reopen: %+v, %v", reopened, err)
	}
}

func TestCollectingPaysBackDebtOwedMe(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	debt, err := debts.Create(
		e.ctx,
		DebtInput{
			Name:      "Ann",
			DebtType:  "owed_to_me",
			AccountID: e.usdAcc.ID,
			Amount:    dec("60"),
			Date:      "2024-01-10",
			Origin:    "new",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := debts.Payment(e.ctx, debt.ID, e.usdAcc.ID, dec("25"), "2024-01-15", nil, true); err != nil {
		t.Fatal(err)
	}
	if got := balanceOf(t, e, e.usdAcc.ID); got.Decimal().String() != "65" { // 100 - 60 + 25
		t.Errorf("account %s, want 65", got)
	}
	if got := balanceOf(t, e, debt.ID); got.Decimal().String() != "35" {
		t.Errorf("still owed %s, want 35", got)
	}
}

func TestPaymentSomethingNotDebtIsRefused(t *testing.T) {
	e := newMoneyEnv(t)
	if _, err := newDebts(e).Payment(e.ctx, e.usdAcc.ID, e.jpyAcc.ID, dec("1"), "2024-01-15", nil, false); err == nil {
		t.Error("an ordinary account is not a debt")
	}
	if _, err := newDebts(e).Reopen(e.ctx, e.usdAcc.ID); err == nil {
		t.Error("an ordinary account cannot be reopened")
	}
}

func TestDebtSummaryFoldsIntoBase(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)

	empty := debts.Summary(e.ctx)
	if empty.Count != 0 || !empty.Net().IsZero() {
		t.Errorf("no debts: %+v", empty)
	}

	mk := func(name, typ string, cur Currency, amount string) *Account {
		d, err := debts.Create(e.ctx, DebtInput{Name: name, DebtType: typ, CurrencyID: cur.ID, Amount: dec(amount)})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	mk("rent", "i_owe", e.usd, "100")
	mk("friend", "i_owe", e.jpy, "5000") // 50 USD
	mk("client", "owed_to_me", e.usd, "30")
	paid := mk("done", "owed_to_me", e.usd, "10")
	if _, err := debts.Payment(e.ctx, paid.ID, e.usdAcc.ID, dec("10"), "2024-01-15", nil, true); err != nil {
		t.Fatal(err)
	}

	sum := debts.Summary(e.ctx)
	if sum.Count != 3 {
		t.Errorf("only open debts count: %d", sum.Count)
	}
	if sum.Currency == nil || sum.Currency.Code != "USD" {
		t.Fatalf("currency %+v", sum.Currency)
	}
	if got := sum.IOwe.Decimal().String(); got != "150" {
		t.Errorf("I owe %s, want 150", got)
	}
	if got := sum.OwedToMe.Decimal().String(); got != "30" {
		t.Errorf("owed to me %s, want 30", got)
	}
	if got := sum.Net().Decimal().String(); got != "-120" {
		t.Errorf("net %s, want -120", got)
	}
	if all, _ := debts.All(e.ctx, true); len(all) != 4 {
		t.Errorf("All with completed lists %d debts", len(all))
	}
}

func TestDeleteRefusesADebtWithHistory(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	clean, _ := debts.Create(e.ctx, DebtInput{Name: "clean", DebtType: "i_owe", CurrencyID: e.usd.ID, Amount: dec("10")})
	used, _ := debts.Create(e.ctx, DebtInput{Name: "used", DebtType: "i_owe", CurrencyID: e.usd.ID, Amount: dec("10")})
	if _, err := debts.Payment(e.ctx, used.ID, e.usdAcc.ID, dec("1"), "2024-01-15", nil, false); err != nil {
		t.Fatal(err)
	}

	if err := debts.Delete(e.ctx, used.ID); err == nil {
		t.Error("a debt with payments cannot be deleted")
	}
	if err := debts.Delete(e.ctx, clean.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := (Accounts{DB: e.db}).ByID(e.ctx, clean.ID); got != nil {
		t.Error("deleted debt still found")
	}
}

func TestReorderSetsOrderOfTheList(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	if err := accts.Reorder(e.ctx, []int64{e.btcAcc.ID, e.usdAcc.ID, e.jpyAcc.ID}); err != nil {
		t.Fatal(err)
	}
	list, err := accts.All(e.ctx, true, true)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, a := range list {
		names = append(names, a.Name)
	}
	if len(names) != 3 || names[0] != "btc" || names[1] != "usd" || names[2] != "jpy" {
		t.Errorf("order %v", names)
	}
}

func TestTotalFoldsBalancesAsOfDate(t *testing.T) {
	e := newMoneyEnv(t)
	date := func(s string) *string { return &s }
	for _, in := range []TxInput{
		{Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("10"), Date: date("2024-01-10")},
		{Type: "expense", AccountID: e.jpyAcc.ID, Amount: dec("1000"), Date: date("2024-01-20")}, // 10 USD
	} {
		if _, err := e.txs.Create(e.ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	accts := Accounts{DB: e.db}
	// Opening balances: USD 100 + JPY 3000 * 0.01.
	for asOf, want := range map[string]string{"2024-01-01": "130", "2024-01-15": "120", "2024-01-31": "110"} {
		if got := accts.TotalAt(e.ctx, e.usd, asOf); got.Decimal().String() != want || got.Unit() != e.usd.Unit() {
			t.Errorf("total as of %s = %s, want %s", asOf, got, want)
		}
	}
}

func TestSetDescriptionChangesOnlyTheDescription(t *testing.T) {
	e := newMoneyEnv(t)
	tx := e.tx(t, "expense", e.usdAcc.ID, "12.34")
	if err := e.txs.SetDescription(e.ctx, tx.ID, "new text"); err != nil {
		t.Fatal(err)
	}
	got, _ := e.txs.ByID(e.ctx, tx.ID)
	if got.Description == nil || *got.Description != "new text" || !got.Amount.Equal(tx.Amount) || got.Type != "expense" {
		t.Errorf("after SetDescription %+v", got)
	}
}

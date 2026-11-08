package ledger

import (
	"errors"
	"testing"
	"time"
)

func newCard(name string, currencyID int64, limit string, day, grace int) NewCard {
	return NewCard{
		Name: name, CurrencyID: currencyID, IsActive: true,
		Terms: CardInput{Limit: dec(limit), StatementDay: day, GraceDays: grace},
	}
}

func TestCreditCardRoundTrip(t *testing.T) {
	e := newMoneyEnv(t)
	cards := CreditCards{Accounts: Accounts{DB: e.db}}
	c, err := cards.Create(e.ctx, newCard("Gold", e.usd.ID, "1500.50", 20, 25))
	if err != nil {
		t.Fatal(err)
	}
	if c.Type != TypeCredit || c.IsService || c.Terms.Limit.String() != "1500.5" || c.Terms.Cycle != CycleFixedDay ||
		*c.Terms.StatementDay != 20 || c.Terms.GraceDays != 25 {
		t.Fatalf("created %+v", c)
	}

	// The list holds only cards; Accounts lists the card as a plain account.
	list, err := cards.All(e.ctx, true)
	if err != nil || len(list) != 1 || list[0].ID != c.ID {
		t.Fatalf("cards %+v %v", list, err)
	}
	plain, err := cards.Accounts.ByID(e.ctx, c.ID)
	if err != nil || plain == nil || plain.Type != TypeCredit {
		t.Fatalf("account of the card %+v %v", plain, err)
	}
	if other, _ := cards.ByID(e.ctx, e.usdAcc.ID); other != nil {
		t.Errorf("an ordinary account is a card: %+v", other)
	}

	// An update with the card as input keeps its terms; a new input changes them.
	edit := c.Edit()
	edit.Name = "Gold Plus"
	if c, err = cards.Update(e.ctx, c.ID, edit); err != nil || c.Name != "Gold Plus" || c.Terms.GraceDays != 25 {
		t.Fatalf("update that keeps the terms: %+v %v", c, err)
	}
	edit = c.Edit()
	edit.Terms.Limit, edit.Terms.GraceDays = dec("2000"), 30
	if c, err = cards.Update(e.ctx, c.ID, edit); err != nil || c.Terms.Limit.String() != "2000" || c.Terms.GraceDays != 30 {
		t.Fatalf("update of the terms: %+v %v", c, err)
	}

	if err := cards.Delete(e.ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, e, "credit_cards"); n != 0 {
		t.Errorf("%d card rows left after deleting the card", n)
	}
	if gone, _ := cards.Accounts.ByID(e.ctx, c.ID); gone != nil {
		t.Errorf("the account is still there: %+v", gone)
	}
}

func countRows(t *testing.T, e *moneyEnv, table string) int {
	t.Helper()
	var n int
	if err := e.db.QueryRowContext(e.ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreditCardRules(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	cards := CreditCards{Accounts: accts}

	if _, err := cards.Create(e.ctx, newCard("x", e.usd.ID, "0", 20, 25)); !errors.Is(err, ErrCardLimit) {
		t.Errorf("a card with no limit: %v", err)
	}
	in := newCard("x", e.usd.ID, "100", 20, 25)
	in.Terms.Cycle = CycleFirstExpense
	if _, err := cards.Create(e.ctx, in); !errors.Is(err, ErrCardCycle) {
		t.Errorf("first_expense: %v", err)
	}
	if n := countRows(t, e, "credit_cards") + countRows(t, e, "accounts WHERE type = 'credit'"); n != 0 {
		t.Errorf("a refused card left %d rows", n)
	}

	// A card is written only through CreditCards, never as an account.
	if _, err := accts.Create(e.ctx, AccountInput{Name: "x", Type: TypeCredit, CurrencyID: e.usd.ID, IsActive: true}); !errors.Is(err, ErrCardAccount) {
		t.Errorf("creating a card as an account: %v", err)
	}
	card, err := cards.Create(e.ctx, newCard("Gold", e.usd.ID, "100", 20, 25))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := accts.Update(e.ctx, card.ID, card.Account.Input()); !errors.Is(err, ErrCardAccount) {
		t.Errorf("updating a card as an account: %v", err)
	}
	if err := accts.Delete(e.ctx, card.ID); !errors.Is(err, ErrCardAccount) {
		t.Errorf("deleting a card as an account: %v", err)
	}
	up := e.usdAcc.Input()
	up.Type = TypeCredit
	if _, err := accts.Update(e.ctx, e.usdAcc.ID, up); !errors.Is(err, ErrCardAccount) {
		t.Errorf("an account made a card by a type change: %v", err)
	}

	// A bank account becomes a card (a converted v1.0 card) and keeps its
	// payments; nothing else does.
	bank, err := accts.Create(e.ctx, AccountInput{Name: "old card", Type: "bank", CurrencyID: e.usd.ID, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	e.txOn(t, simple("expense", bank.ID, "40"), "2026-02-10")
	got, err := cards.Convert(e.ctx, bank.ID, CardInput{Limit: dec("500"), StatementDay: 5, GraceDays: 20})
	if err != nil || got.Type != TypeCredit || got.Balance.String() != "-40" || got.Terms.GraceDays != 20 {
		t.Fatalf("bank to credit: %+v %v", got, err)
	}
	if _, err := cards.Convert(e.ctx, bank.ID, CardInput{Limit: dec("500"), StatementDay: 5}); !errors.Is(err, ErrCardNotConvertible) {
		t.Errorf("a card converted again: %v", err)
	}
	cash, _ := accts.Create(e.ctx, AccountInput{Name: "wallet", Type: "cash", CurrencyID: e.usd.ID, IsActive: true})
	if _, err := cards.Convert(e.ctx, cash.ID, CardInput{Limit: dec("500"), StatementDay: 5}); !errors.Is(err, ErrCardNotConvertible) {
		t.Errorf("cash to credit: %v", err)
	}
	if _, err := cards.Convert(e.ctx, bank.ID+999, CardInput{Limit: dec("500"), StatementDay: 5}); err != nil {
		t.Errorf("an unknown account: %v", err)
	}
	if err := cards.Delete(e.ctx, bank.ID); err == nil {
		t.Error("deleted a card that has payments")
	}
}

// The card's state reads its confirmed payments. Paying 1020 from a bank
// account to clear a debt of 1000 is a transfer with a fee line: the card sees
// +1000, the 20 is spending.
func TestCardStateReadsPaymentsAndIgnoresPendingOnes(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	cards := CreditCards{Accounts: accts}
	card, err := cards.Create(e.ctx, newCard("Gold", e.usd.ID, "5000", 20, 25))
	if err != nil {
		t.Fatal(err)
	}
	fees, err := (Categories{DB: e.db}).Create(e.ctx, Category{Name: "Fees", Type: "expense"})
	if err != nil {
		t.Fatal(err)
	}
	e.txOn(t, simple("expense", card.ID, "1000"), "2026-02-10")
	pending := "pending"
	later := simple("expense", card.ID, "300")
	date := "2026-02-12"
	later.Date, later.Status = &date, &pending
	if _, err := e.txs.Create(e.ctx, later); err != nil {
		t.Fatal(err)
	}
	today := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	read := func() CardState {
		t.Helper()
		c, err := cards.ByID(e.ctx, card.ID)
		if err != nil || c == nil {
			t.Fatalf("card: %v %v", c, err)
		}
		st, err := cards.State(e.ctx, *c, today)
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	st := read()
	if st.Statement.AmountDue.String() != "1000" || st.Status.Level != StatusWarning {
		t.Fatalf("before paying: %+v / %+v", st.Statement, st.Status)
	}

	// The bank account pays 1020: 1000 reaches the card and 20 is the commission.
	out, in := dec("1020").Neg(), dec("1000")
	fee := dec("20")
	date = "2026-03-01"
	if _, err := e.txs.Create(e.ctx, TxInput{
		Type: "transfer", Date: &date,
		Payments: []PaymentInput{{AccountID: e.usdAcc.ID, Amount: &out}, {AccountID: card.ID, Amount: &in}},
		Lines:    []LineInput{{CategoryID: &fees.ID, CurrencyID: e.usd.ID, Amount: fee}},
	}); err != nil {
		t.Fatal(err)
	}
	st = read()
	if st.Statement.Paid.String() != "1000" || st.Statement.AmountDue.String() != "0" || st.Status.Level != StatusOK {
		t.Errorf("after paying: %+v / %+v", st.Statement, st.Status)
	}
	if got, _ := cards.ByID(e.ctx, card.ID); !got.Balance.IsZero() {
		t.Errorf("balance %s, want 0 (the pending 300 does not count)", got.Balance)
	}
}

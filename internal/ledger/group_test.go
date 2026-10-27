package ledger

import (
	"errors"
	"testing"

	"savvy-go/internal/db/filter"
)

const groupDay = "2024-01-15"

func (e *moneyEnv) groups() Groups { return Groups{DB: e.db} }

func (e *moneyEnv) createGroup(t *testing.T, legs ...TxInput) *Group {
	t.Helper()
	g, err := e.groups().Create(e.ctx, legs)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func (e *moneyEnv) leg(typ string, account int64, amount string) TxInput {
	day := groupDay
	return TxInput{Type: typ, AccountID: account, Amount: dec(amount), Date: &day}
}

func (e *moneyEnv) exchangeLeg(from int64, amount string, to int64, toAmount string) TxInput {
	in := e.leg("transfer", from, amount)
	in.ToAccountID = &to
	d := dec(toAmount)
	in.ToAmount = &d
	return in
}

func (e *moneyEnv) txCount(t *testing.T) int {
	t.Helper()
	_, total, err := e.txs.Filtered(e.ctx, filter.TxFilter{}, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	return total
}

// Rent is 45 USD. 50 USD are handed over and 500 JPY come back. That is an
// expense of exactly 45 USD plus an exchange of 5 USD into 500 JPY, so the
// rent never depends on today's rate.
func TestGroupRentPaidWithChangeInAnotherCurrency(t *testing.T) {
	e := newMoneyEnv(t)
	g := e.createGroup(t,
		e.leg("expense", e.usdAcc.ID, "45"),
		e.exchangeLeg(e.usdAcc.ID, "5", e.jpyAcc.ID, "500"),
	)
	if len(g.Legs) != 2 || g.UUID == "" {
		t.Fatalf("group = %+v", g)
	}
	for _, leg := range g.Legs {
		if leg.GroupUUID == nil || *leg.GroupUUID != g.UUID {
			t.Errorf("leg %d group = %v, want %s", leg.ID, leg.GroupUUID, g.UUID)
		}
	}
	if got := g.Legs[0].Amount.Decimal(); !got.Equal(dec("45")) {
		t.Errorf("rent = %s, want 45", got)
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("50")) { // 100 - 45 - 5
		t.Errorf("usd = %s, want 50", got)
	}
	if got := e.balance(t, e.jpyAcc.ID); !got.Equal(dec("3500")) { // 3000 + 500
		t.Errorf("jpy = %s, want 3500", got)
	}
}

func TestGroupPaidInTwoCurrencies(t *testing.T) {
	e := newMoneyEnv(t)
	e.createGroup(t,
		e.leg("expense", e.usdAcc.ID, "20"),
		e.leg("expense", e.jpyAcc.ID, "1000"),
	)
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("80")) {
		t.Errorf("usd = %s, want 80", got)
	}
	if got := e.balance(t, e.jpyAcc.ID); !got.Equal(dec("2000")) {
		t.Errorf("jpy = %s, want 2000", got)
	}
}

func TestGroupPaidByCashAndCard(t *testing.T) {
	e := newMoneyEnv(t)
	card, err := (Accounts{DB: e.db}).Create(e.ctx, AccountInput{
		Name: "card", Type: "bank", CurrencyID: e.usd.ID, InitialBalance: dec("200"), IsActive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	e.createGroup(t,
		e.leg("expense", e.usdAcc.ID, "15"),
		e.leg("expense", card.ID, "25"),
	)
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("85")) {
		t.Errorf("cash = %s, want 85", got)
	}
	if got := e.balance(t, card.ID); !got.Equal(dec("175")) {
		t.Errorf("card = %s, want 175", got)
	}
}

func TestGroupRefusesWhatCannotBeOneReceipt(t *testing.T) {
	e := newMoneyEnv(t)
	other := "2024-01-16"
	otherDay := e.leg("expense", e.jpyAcc.ID, "10")
	otherDay.Date = &other
	future := "2999-01-01"
	futureLeg := e.leg("expense", e.jpyAcc.ID, "10")
	futureLeg.Date = &future
	pending := "pending"
	pendingLeg := e.leg("expense", e.jpyAcc.ID, "10")
	pendingLeg.Status = &pending
	noDate := e.leg("expense", e.jpyAcc.ID, "10")
	noDate.Date = nil

	cases := map[string][]TxInput{
		"no legs":          nil,
		"one leg":          {e.leg("expense", e.usdAcc.ID, "1")},
		"different days":   {e.leg("expense", e.usdAcc.ID, "1"), otherDay},
		"future leg":       {e.leg("expense", e.usdAcc.ID, "1"), futureLeg},
		"pending leg":      {e.leg("expense", e.usdAcc.ID, "1"), pendingLeg},
		"leg without date": {e.leg("expense", e.usdAcc.ID, "1"), noDate},
		"debt leg":         {e.leg("expense", e.usdAcc.ID, "1"), e.leg("debt_lend", e.jpyAcc.ID, "1")},
		"space transfer":   {e.leg("expense", e.usdAcc.ID, "1"), e.leg("transfer_out", e.jpyAcc.ID, "1")},
	}
	for name, legs := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := e.groups().Create(e.ctx, legs); !errors.Is(err, ErrGroupInvalid) {
				t.Errorf("err = %v, want ErrGroupInvalid", err)
			}
			if n := e.txCount(t); n != 0 {
				t.Errorf("%d transactions written", n)
			}
		})
	}
}

// A bad second leg must not leave the first one behind.
func TestGroupWritesAllLegsOrNone(t *testing.T) {
	e := newMoneyEnv(t)
	bad := e.leg("expense", 9999, "1") // no such account
	if _, err := e.groups().Create(e.ctx, []TxInput{e.leg("expense", e.usdAcc.ID, "5"), bad}); err == nil {
		t.Fatal("want error")
	}
	if n := e.txCount(t); n != 0 {
		t.Errorf("%d transactions written, want none", n)
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("100")) {
		t.Errorf("usd = %s, want 100", got)
	}
}

// One leg changes only with its group.
func TestGroupLegRefusesSingleLegChanges(t *testing.T) {
	e := newMoneyEnv(t)
	g := e.createGroup(t, e.leg("expense", e.usdAcc.ID, "1"), e.leg("expense", e.jpyAcc.ID, "100"))
	leg := g.Legs[0]

	// The flags are the payment's: it can be edited and deleted, a leg cannot be copied.
	if a := leg.Allowed(); a != (TxActions{Edit: true, Delete: true}) {
		t.Errorf("allowed = %+v, want edit and delete", a)
	}
	if _, err := e.txs.Update(e.ctx, leg.ID, e.leg("expense", e.usdAcc.ID, "2")); !errors.Is(err, ErrGroupLeg) {
		t.Errorf("update err = %v, want ErrGroupLeg", err)
	}
	if err := e.txs.Delete(e.ctx, leg.ID); !errors.Is(err, ErrGroupLeg) {
		t.Errorf("delete err = %v, want ErrGroupLeg", err)
	}
	if _, err := e.txs.Duplicate(e.ctx, leg.ID); !errors.Is(err, ErrGroupLeg) {
		t.Errorf("duplicate err = %v, want ErrGroupLeg", err)
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("99")) {
		t.Errorf("usd = %s, want 99", got)
	}
}

func TestGroupUpdateReplacesAllLegs(t *testing.T) {
	e := newMoneyEnv(t)
	g := e.createGroup(t,
		e.leg("expense", e.usdAcc.ID, "45"),
		e.exchangeLeg(e.usdAcc.ID, "5", e.jpyAcc.ID, "500"),
	)
	got, err := e.groups().Update(e.ctx, g.UUID, []TxInput{
		e.leg("expense", e.usdAcc.ID, "40"),
		e.exchangeLeg(e.usdAcc.ID, "10", e.jpyAcc.ID, "1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != g.UUID || len(got.Legs) != 2 {
		t.Fatalf("group = %+v", got)
	}
	if b := e.balance(t, e.usdAcc.ID); !b.Equal(dec("50")) { // 100 - 40 - 10
		t.Errorf("usd = %s, want 50", b)
	}
	if b := e.balance(t, e.jpyAcc.ID); !b.Equal(dec("4000")) { // 3000 + 1000
		t.Errorf("jpy = %s, want 4000", b)
	}
	if n := e.txCount(t); n != 2 {
		t.Errorf("%d transactions, want 2", n)
	}
	// An update that breaks the rules changes nothing.
	if _, err := e.groups().Update(e.ctx, g.UUID, []TxInput{e.leg("expense", e.usdAcc.ID, "1")}); !errors.Is(err, ErrGroupInvalid) {
		t.Errorf("err = %v, want ErrGroupInvalid", err)
	}
	if b := e.balance(t, e.usdAcc.ID); !b.Equal(dec("50")) {
		t.Errorf("usd = %s after refused update, want 50", b)
	}
}

func TestGroupDeleteRemovesAllLegs(t *testing.T) {
	e := newMoneyEnv(t)
	g := e.createGroup(t,
		e.leg("expense", e.usdAcc.ID, "45"),
		e.exchangeLeg(e.usdAcc.ID, "5", e.jpyAcc.ID, "500"),
	)
	if err := e.groups().Delete(e.ctx, g.UUID); err != nil {
		t.Fatal(err)
	}
	if n := e.txCount(t); n != 0 {
		t.Errorf("%d transactions left", n)
	}
	if b := e.balance(t, e.usdAcc.ID); !b.Equal(dec("100")) {
		t.Errorf("usd = %s, want 100", b)
	}
	if got, err := e.groups().ByUUID(e.ctx, g.UUID); err != nil || got != nil {
		t.Errorf("ByUUID = %v, %v, want nil", got, err)
	}
}

// The list stays one row per leg, so an account filter shows that account's
// leg only; each row says which group it belongs to.
func TestGroupLegsListedPerLegWithTheirGroup(t *testing.T) {
	e := newMoneyEnv(t)
	g := e.createGroup(t,
		e.leg("expense", e.usdAcc.ID, "45"),
		e.exchangeLeg(e.usdAcc.ID, "5", e.jpyAcc.ID, "500"),
	)
	e.tx(t, "expense", e.btcAcc.ID, "0") // not in the group
	list, total, err := e.txs.Filtered(e.ctx, filter.TxFilter{AccountID: e.usdAcc.ID}, 1, 100)
	if err != nil || total != 2 {
		t.Fatalf("total = %d, err = %v", total, err)
	}
	for _, tx := range list {
		if tx.GroupUUID == nil || *tx.GroupUUID != g.UUID {
			t.Errorf("tx %d group = %v, want %s", tx.ID, tx.GroupUUID, g.UUID)
		}
	}
	all, _, _ := e.txs.Filtered(e.ctx, filter.TxFilter{}, 1, 100)
	var loose int
	for _, tx := range all {
		if tx.GroupUUID == nil {
			loose++
		}
	}
	if loose != 1 {
		t.Errorf("%d ungrouped, want 1", loose)
	}
}

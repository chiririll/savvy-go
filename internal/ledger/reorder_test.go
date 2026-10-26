package ledger

import (
	"slices"
	"testing"

	"savvy-go/internal/db/filter"
)

func dayIDs(t *testing.T, e *moneyEnv, dir string) []int64 {
	t.Helper()
	list, _, err := e.txs.Filtered(e.ctx, filter.TxFilter{SortBy: "date", SortDir: dir}, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, tx := range list {
		ids = append(ids, tx.ID)
	}
	return ids
}

func TestReorderChangesOrderWithinTheDay(t *testing.T) {
	e := newMoneyEnv(t)
	a := e.tx(t, "expense", e.usdAcc.ID, "1")
	b := e.tx(t, "expense", e.usdAcc.ID, "2")
	c := e.tx(t, "expense", e.usdAcc.ID, "3")

	if got := dayIDs(t, e, "asc"); !slices.Equal(got, []int64{a.ID, b.ID, c.ID}) {
		t.Fatalf("default asc = %v", got)
	}
	if err := e.txs.Reorder(e.ctx, "2024-01-15", []int64{c.ID, a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	if got := dayIDs(t, e, "asc"); !slices.Equal(got, []int64{c.ID, a.ID, b.ID}) {
		t.Errorf("asc = %v, want c a b", got)
	}
	if got := dayIDs(t, e, "desc"); !slices.Equal(got, []int64{b.ID, a.ID, c.ID}) {
		t.Errorf("desc = %v, want b a c", got)
	}

	// A transaction added later still goes after the day's existing ones.
	d := e.tx(t, "expense", e.usdAcc.ID, "4")
	if got := dayIDs(t, e, "asc"); !slices.Equal(got, []int64{c.ID, a.ID, b.ID, d.ID}) {
		t.Errorf("after add asc = %v", got)
	}
}

func TestReorderAcceptsASubsetOfTheDay(t *testing.T) {
	e := newMoneyEnv(t)
	a := e.tx(t, "expense", e.usdAcc.ID, "1")
	b := e.tx(t, "expense", e.usdAcc.ID, "2")
	c := e.tx(t, "expense", e.usdAcc.ID, "3")
	// Only a and c are shown (a filter); swapping them leaves b between.
	if err := e.txs.Reorder(e.ctx, "2024-01-15", []int64{c.ID, a.ID}); err != nil {
		t.Fatal(err)
	}
	if got := dayIDs(t, e, "asc"); !slices.Equal(got, []int64{c.ID, b.ID, a.ID}) {
		t.Errorf("asc = %v, want c b a", got)
	}
}

func TestReorderRejectsOtherDaysAndUnknownIDs(t *testing.T) {
	e := newMoneyEnv(t)
	a := e.tx(t, "expense", e.usdAcc.ID, "1")
	other := "2024-02-01"
	o, err := e.txs.Create(e.ctx, TxInput{Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("5"), Date: &other})
	if err != nil {
		t.Fatal(err)
	}
	for name, ids := range map[string][]int64{
		"other day": {a.ID, o.ID},
		"unknown":   {a.ID, 9999},
		"duplicate": {a.ID, a.ID},
	} {
		if err := e.txs.Reorder(e.ctx, "2024-01-15", ids); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if err := e.txs.Reorder(e.ctx, "not a date", []int64{a.ID}); err == nil {
		t.Error("bad date: want an error")
	}
}

func TestMovingATransactionToAnotherDayDropsItsManualPlace(t *testing.T) {
	e := newMoneyEnv(t)
	a := e.tx(t, "expense", e.usdAcc.ID, "1")
	b := e.tx(t, "expense", e.usdAcc.ID, "2")
	if err := e.txs.Reorder(e.ctx, "2024-01-15", []int64{b.ID, a.ID}); err != nil {
		t.Fatal(err)
	}
	day := "2024-01-16"
	if _, err := e.txs.Update(e.ctx, a.ID, TxInput{
		Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("1"), Date: &day,
	}); err != nil {
		t.Fatal(err)
	}
	var order *int64
	if err := e.db.QueryRow(`SELECT day_order FROM transactions WHERE id = ?`, a.ID).Scan(&order); err != nil {
		t.Fatal(err)
	}
	if order != nil {
		t.Errorf("day_order = %d, want NULL after the date changed", *order)
	}
	// Saving it again on the same day keeps a manual place.
	if err := e.txs.Reorder(e.ctx, "2024-01-15", []int64{b.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.txs.Update(e.ctx, b.ID, TxInput{
		Type: "expense", AccountID: e.usdAcc.ID, Amount: dec("9"), Date: ptr("2024-01-15"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := e.db.QueryRow(`SELECT day_order FROM transactions WHERE id = ?`, b.ID).Scan(&order); err != nil || order == nil {
		t.Errorf("day_order after same-day edit = %v, %v; want kept", order, err)
	}
}

func ptr[T any](v T) *T { return &v }

package ledger

import (
	"testing"

	"savvy-go/internal/db/filter"
)

// A page loads the items and tags of all its transactions in one query each;
// every line and tag must still land on its own transaction, in its currency.
func TestPageHydrationKeepsItemsTags(t *testing.T) {
	e := newMoneyEnv(t)
	tags := Tags{DB: e.db}
	food, err := tags.Create(e.ctx, "food")
	if err != nil {
		t.Fatal(err)
	}
	trip, err := tags.Create(e.ctx, "trip")
	if err != nil {
		t.Fatal(err)
	}
	date := "2024-01-15"
	mk := func(acct int64, amount string, items []TxItemInput, tagIDs []int64) int64 {
		tx, err := e.txs.Create(
			e.ctx,
			TxInput{Type: "expense", AccountID: acct, Amount: dec(amount), Date: &date, Items: items, TagIDs: tagIDs},
		)
		if err != nil {
			t.Fatal(err)
		}
		return tx.ID
	}
	usdTx := mk(
		e.usdAcc.ID,
		"3",
		[]TxItemInput{
			{Name: "a", Quantity: dec("1"), PricePerUnit: dec("1")},
			{Name: "b", Quantity: dec("2"), PricePerUnit: dec("1")},
		},
		[]int64{food.ID, trip.ID},
	)
	jpyTx := mk(
		e.jpyAcc.ID,
		"500",
		[]TxItemInput{{Name: "c", Quantity: dec("1"), PricePerUnit: dec("500")}},
		[]int64{trip.ID},
	)
	bare := mk(e.usdAcc.ID, "1", nil, nil)

	page, total, err := e.txs.Filtered(e.ctx, filter.TxFilter{}, 1, 25)
	if err != nil || total != 3 {
		t.Fatalf("page: %d rows, %v", total, err)
	}
	byID := map[int64]Transaction{}
	for _, tx := range page {
		byID[tx.ID] = tx
	}

	if got := byID[usdTx]; len(got.Items) != 2 || got.Items[0].Name != "a" || got.Items[1].Name != "b" ||
		len(got.Tags) != 2 {
		t.Fatalf("usd tx items %+v tags %+v", got.Items, got.Tags)
	}
	if got := byID[jpyTx]; len(got.Items) != 1 || got.Items[0].Name != "c" || len(got.Tags) != 1 ||
		got.Tags[0].ID != trip.ID {
		t.Fatalf("jpy tx items %+v tags %+v", got.Items, got.Tags)
	}
	if got := byID[jpyTx].Items[0].TotalPrice; got.Unit() != e.jpy.Unit() || !got.Decimal().Equal(dec("500")) {
		t.Fatalf("jpy item total = %s %+v, want 500 JPY", got, got.Unit())
	}
	if got := byID[bare]; got.Items == nil || got.Tags == nil || len(got.Items) != 0 || len(got.Tags) != 0 {
		t.Fatalf("bare tx must have empty, non-nil items and tags: %+v %+v", got.Items, got.Tags)
	}
	// Transactions of the same account share one loaded account.
	if byID[usdTx].Account == nil || byID[usdTx].Account != byID[bare].Account {
		t.Fatal("expected the account to be loaded once and shared")
	}
}

// The summary totals what the list shows: the same filter narrows both.
func TestSummaryFollowsTheFilter(t *testing.T) {
	e := newMoneyEnv(t)
	date := "2024-01-15"
	mk := func(typ string, acct int64, amount, desc string) {
		t.Helper()
		if _, err := e.txs.Create(e.ctx, TxInput{
			Type: typ, AccountID: acct, Amount: dec(amount), Date: &date, Description: &desc,
		}); err != nil {
			t.Fatal(err)
		}
	}
	mk("income", e.usdAcc.ID, "100", "salary")
	mk("expense", e.usdAcc.ID, "30", "armchair")
	mk("expense", e.usdAcc.ID, "5", "coffee")
	mk("expense", e.jpyAcc.ID, "500", "armchair cover")

	all := e.txs.Summary(e.ctx, filter.TxFilter{}, false)
	if all.Count != 4 {
		t.Fatalf("unfiltered count = %d, want 4", all.Count)
	}
	usd := e.txs.Summary(e.ctx, filter.TxFilter{Currencies: []string{"usd"}}, false)
	if usd.Count != 3 || !usd.Income.Decimal().Equal(dec("100")) || !usd.Expense.Decimal().Equal(dec("35")) {
		t.Fatalf("usd = %d, income %s, expense %s", usd.Count, usd.Income, usd.Expense)
	}
	chairs := e.txs.Summary(e.ctx, filter.TxFilter{Search: "armchair", Currencies: []string{"USD"}}, false)
	if chairs.Count != 1 || !chairs.Income.IsZero() || !chairs.Expense.Decimal().Equal(dec("30")) {
		t.Fatalf("armchair = %d, income %s, expense %s", chairs.Count, chairs.Income, chairs.Expense)
	}
	// Type and status narrow the list, but the summary stays income and
	// expense in the status it was asked for.
	if got := e.txs.Summary(e.ctx, filter.TxFilter{Type: "transfer"}, false); got.Count != 0 {
		t.Fatalf("transfer filter counted %d", got.Count)
	}
	if got := e.txs.Summary(e.ctx, filter.TxFilter{Status: "pending"}, false); got.Count != 4 {
		t.Fatalf("a status filter must not change which status the summary covers: %d", got.Count)
	}
	if got := e.txs.Summary(e.ctx, filter.TxFilter{Search: "nothing"}, false); got.Count != 0 || !got.Expense.IsZero() {
		t.Fatalf("no match = %d, expense %s", got.Count, got.Expense)
	}
}

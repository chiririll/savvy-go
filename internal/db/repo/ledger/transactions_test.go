package ledger_test

import (
	"testing"
	"time"

	"savvy-go/internal/db/filter"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

func (f *fixture) unit(id int64) money.Unit {
	switch id {
	case f.usd:
		return money.Unit{ID: f.usd, Decimals: 2}
	default:
		return money.Unit{ID: f.jpy, Decimals: 0}
	}
}

// transfer stores a transfer of out from the cash account into the yen
// account, with the fee as a line of the sending side.
func (f *fixture) transfer(t *testing.T, out, in int64, date string) int64 {
	t.Helper()
	d := date
	id, err := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{Type: "transfer", Date: &d}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.ReplacePayments(ctx, id, []ledgerrepo.PaymentFields{
		{AccountID: f.cash, Amount: money.New(-out, f.unit(f.usd))},
		{AccountID: f.yen, Amount: money.New(in, f.unit(f.jpy))},
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestTransactionRoundTripsPaymentsLinesAndTags(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	tag, _ := f.repo.InsertTag(ctx, "trip", now)
	other, _ := f.repo.InsertCategory(ctx, ledgerrepo.Category{Name: "Fees", Type: "expense"}, now)
	desc, date, note := "Dinner abroad", "2026-03-04", "service"
	by := int64(7)
	id, err := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{
		Type: "expense", Description: &desc, Date: &date, CreatedBy: &by,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Paid from the cash account in dollars; the lines are in two currencies.
	if err := f.repo.ReplacePayments(ctx, id, []ledgerrepo.PaymentFields{
		{AccountID: f.cash, Amount: money.New(-1250, f.unit(f.usd))},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.ReplaceLines(ctx, id, []ledgerrepo.LineFields{
		{CategoryID: &f.food, Amount: money.New(1000, f.unit(f.usd))},
		{CategoryID: &other, Amount: money.New(250, f.unit(f.usd)), Note: &note},
		{Amount: money.New(300, f.unit(f.jpy))},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SetTags(ctx, id, []int64{tag}); err != nil {
		t.Fatal(err)
	}

	got, err := f.repo.Transaction(ctx, id)
	if err != nil || got == nil {
		t.Fatalf("transaction = %v, %v", got, err)
	}
	if got.Type != "expense" || got.Status != "confirmed" || *got.Description != desc || *got.Date != date ||
		*got.CreatedBy != 7 || got.CreatedAt == nil || got.ExternalID != nil || got.DayOrder != nil {
		t.Fatalf("row = %+v", got)
	}
	if len(got.Payments) != 1 || got.Payments[0].AccountID != f.cash ||
		got.Payments[0].Amount != money.New(-1250, f.unit(f.usd)) || got.Payments[0].SpaceTransferUUID != nil {
		t.Fatalf("payments = %+v", got.Payments)
	}
	if len(got.Lines) != 3 || got.Lines[0].Seq != 0 || got.Lines[2].Seq != 2 ||
		*got.Lines[0].CategoryID != f.food || *got.Lines[1].Note != note || got.Lines[2].CategoryID != nil ||
		got.Lines[2].Amount != money.New(300, f.unit(f.jpy)) {
		t.Fatalf("lines = %+v", got.Lines)
	}
	if len(got.Tags) != 1 || got.Tags[0].ID != tag {
		t.Fatalf("tags = %+v", got.Tags)
	}
}

func TestSavingRewritesLinesAndUpdatesPaymentsInPlace(t *testing.T) {
	f := newFixture(t)
	d := f.transfer(t, 1000, 150000, "2026-03-04")
	// The yen payment is the cross-space movement of a transfer record.
	if _, err := f.repo.DB.ExecContext(ctx,
		`INSERT INTO space_transfers (uuid, payload) VALUES ('u1', '{"version":1}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.DB.ExecContext(ctx,
		`UPDATE payments SET space_transfer_uuid = 'u1' WHERE transaction_id = ? AND account_id = ?`, d, f.yen); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.ReplaceLines(ctx, d, []ledgerrepo.LineFields{
		{CategoryID: &f.food, Amount: money.New(10, f.unit(f.usd))},
		{CategoryID: &f.food, Amount: money.New(20, f.unit(f.usd))},
	}); err != nil {
		t.Fatal(err)
	}

	date := "2026-03-05"
	if err := f.repo.UpdateTransaction(ctx, d, ledgerrepo.TxFields{Type: "transfer", Date: &date}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	// The cash payment changes, the yen payment stays with its uuid, and a
	// third account is added and then dropped again.
	if err := f.repo.ReplacePayments(ctx, d, []ledgerrepo.PaymentFields{
		{AccountID: f.cash, Amount: money.New(-1100, f.unit(f.usd))},
		{AccountID: f.yen, Amount: money.New(160000, f.unit(f.jpy))},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.ReplaceLines(ctx, d, []ledgerrepo.LineFields{
		{CategoryID: &f.food, Amount: money.New(100, f.unit(f.usd))},
	}); err != nil {
		t.Fatal(err)
	}

	got, _ := f.repo.Transaction(ctx, d)
	if *got.Date != date || len(got.Payments) != 2 {
		t.Fatalf("tx = %+v", got)
	}
	byAccount := map[int64]ledgerrepo.Payment{}
	for _, p := range got.Payments {
		byAccount[p.AccountID] = p
	}
	if byAccount[f.cash].Amount.Minor() != -1100 || byAccount[f.yen].Amount.Minor() != 160000 {
		t.Fatalf("payments = %+v", got.Payments)
	}
	if u := byAccount[f.yen].SpaceTransferUUID; u == nil || *u != "u1" {
		t.Fatalf("the transfer uuid was lost: %v", u)
	}
	if len(got.Lines) != 1 || got.Lines[0].Seq != 0 || got.Lines[0].Amount.Minor() != 100 {
		t.Fatalf("lines = %+v", got.Lines)
	}

	// A payment whose account is not in the list any more is deleted.
	if err := f.repo.ReplacePayments(ctx, d, []ledgerrepo.PaymentFields{
		{AccountID: f.cash, Amount: money.New(-1100, f.unit(f.usd))},
	}); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.repo.Transaction(ctx, d); len(got.Payments) != 1 || got.Payments[0].AccountID != f.cash {
		t.Fatalf("payments = %+v", got.Payments)
	}
}

func TestMovingToAnotherDayDropsTheManualPlace(t *testing.T) {
	f := newFixture(t)
	a := f.expense(t, f.cash, 100, "2026-03-04")
	b := f.expense(t, f.cash, 200, "2026-03-04")
	if err := f.repo.SetDayOrder(ctx, a, 500); err != nil {
		t.Fatal(err)
	}
	slots, _ := f.repo.DaySlots(ctx, []int64{a, b})
	if len(slots) != 2 {
		t.Fatalf("slots = %+v", slots)
	}
	if got, _ := f.repo.Transaction(ctx, a); got.DayOrder == nil || *got.DayOrder != 500 {
		t.Fatalf("day order = %v", got.DayOrder)
	}

	same := "2026-03-04T10:00:00Z"
	if err := f.repo.UpdateTransaction(ctx, a, ledgerrepo.TxFields{Type: "expense", Date: &same}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.repo.Transaction(ctx, a); got.DayOrder == nil {
		t.Fatal("the place within the day was dropped by a change inside the day")
	}
	next := "2026-03-05"
	if err := f.repo.UpdateTransaction(ctx, a, ledgerrepo.TxFields{Type: "expense", Date: &next}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.repo.Transaction(ctx, a); got.DayOrder != nil {
		t.Fatalf("day order = %v after moving to another day", *got.DayOrder)
	}
}

func TestPendingTransactionIsConfirmedOrSkipped(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	date := "2026-04-01"
	pending, _ := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{Type: "expense", Status: "pending", Date: &date}, now)
	skipped, _ := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{Type: "expense", Status: "pending", Date: &date}, now)
	if err := f.repo.ConfirmTransaction(ctx, pending, "2026-04-02", now); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SkipTransaction(ctx, skipped, now); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.repo.Transaction(ctx, pending); got.Status != "confirmed" || *got.Date != "2026-04-02" {
		t.Fatalf("confirmed = %+v", got)
	}
	if got, _ := f.repo.Transaction(ctx, skipped); got.Status != "skipped" {
		t.Fatalf("skipped = %+v", got)
	}
}

func TestDeletingATransactionRemovesWhatBelongsToIt(t *testing.T) {
	f := newFixture(t)
	tag, _ := f.repo.InsertTag(ctx, "x", time.Now().UTC())
	id := f.expense(t, f.cash, 100, "2026-03-04")
	keep := f.expense(t, f.cash, 300, "2026-03-04")
	_ = f.repo.SetTags(ctx, id, []int64{tag})
	if err := f.repo.DeleteTransaction(ctx, id); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.repo.Transaction(ctx, id); got != nil {
		t.Fatalf("still there: %+v", got)
	}
	for _, table := range []string{"payments", "transaction_lines", "transaction_tag"} {
		var n int
		if err := f.repo.DB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM `+table+` WHERE transaction_id = ?`, id).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s: %d rows left, %v", table, n, err)
		}
	}
	if got, _ := f.repo.Transaction(ctx, keep); got == nil || len(got.Payments) != 1 || len(got.Lines) != 1 {
		t.Fatalf("the other transaction lost rows: %+v", got)
	}
	if changes, _ := f.repo.DailyChanges(ctx, f.cash, ""); len(changes) != 1 || changes[0].Delta != -300 {
		t.Fatalf("changes = %+v", changes)
	}
}

func TestListFiltersReadPaymentsLinesAndTags(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	other, _ := f.repo.InsertCategory(ctx, ledgerrepo.Category{Name: "Other", Type: "expense"}, now)
	tag, _ := f.repo.InsertTag(ctx, "work", now)

	lunch := f.expense(t, f.cash, 1000, "2026-01-02")
	yen := f.expense(t, f.yen, 500, "2026-01-03")
	moved := f.transfer(t, 2000, 300000, "2026-01-04")
	split := f.expense(t, f.cash, 700, "2026-02-01")
	_ = f.repo.ReplaceLines(ctx, split, []ledgerrepo.LineFields{
		{CategoryID: &f.food, Amount: money.New(300, f.unit(f.usd))},
		{CategoryID: &other, Amount: money.New(400, f.unit(f.usd))},
	})
	_ = f.repo.SetTags(ctx, lunch, []int64{tag})

	ids := func(fl filter.TxFilter) []int64 {
		t.Helper()
		list, total, err := f.repo.Filtered(ctx, fl, 50, 0)
		if err != nil {
			t.Fatal(err)
		}
		if total != len(list) {
			t.Fatalf("total %d, listed %d", total, len(list))
		}
		out := make([]int64, len(list))
		for i, x := range list {
			out[i] = x.ID
		}
		return out
	}
	same := func(got []int64, want ...int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("got %v, want %v", got, want)
			}
		}
	}

	same(ids(filter.TxFilter{}), split, moved, yen, lunch)
	same(ids(filter.TxFilter{SortDir: "asc"}), lunch, yen, moved, split)
	// A transfer is on both of its accounts.
	same(ids(filter.TxFilter{AccountID: f.yen}), moved, yen)
	same(ids(filter.TxFilter{AccountID: f.cash}), split, moved, lunch)
	same(ids(filter.TxFilter{Type: "transfer"}), moved)
	// A split transaction is in each of its categories, once.
	same(ids(filter.TxFilter{CategoryID: other}), split)
	same(ids(filter.TxFilter{CategoryID: f.food}), split, yen, lunch)
	same(ids(filter.TxFilter{CategoryIDs: []int64{f.food, other}}), split, yen, lunch)
	same(ids(filter.TxFilter{TagIDs: []int64{tag}}), lunch)
	same(ids(filter.TxFilter{StartDate: "2026-01-03", EndDate: "2026-01-31"}), moved, yen)

	// Paging keeps the total of all matches and loads payments of the page only.
	page, total, _ := f.repo.Filtered(ctx, filter.TxFilter{}, 2, 1)
	if total != 4 || len(page) != 2 || page[0].ID != moved || len(page[0].Payments) != 2 || len(page[1].Lines) != 1 {
		t.Fatalf("page = %+v of %d", page, total)
	}
	if page[0].Lines == nil || page[0].Tags == nil {
		t.Fatal("a transaction without lines or tags lists empty slices, not nil")
	}
}

func TestListSortsByMovedBaseCurrency(t *testing.T) {
	f := newFixture(t)
	// 100000 yen at 0.0066 is $660; $1000 is more; the transfer moves $20 out
	// and 3000 yen ($19.80) in and counts as the larger side, not as the sum.
	small := f.expense(t, f.yen, 100000, "2026-01-01")
	big := f.expense(t, f.cash, 100000, "2026-01-02")
	moved := f.transfer(t, 2000, 3000, "2026-01-03")

	list, _, err := f.repo.Filtered(ctx, filter.TxFilter{SortBy: "amount"}, 10, 0)
	if err != nil || len(list) != 3 || list[0].ID != big || list[1].ID != small || list[2].ID != moved {
		t.Fatalf("by amount = %+v, %v", list, err)
	}
	list, _, _ = f.repo.Filtered(ctx, filter.TxFilter{SortBy: "amount", SortDir: "asc"}, 10, 0)
	if list[0].ID != moved || list[2].ID != big {
		t.Fatalf("ascending = %+v", list)
	}
}

func TestSummaryRowsAreConfirmedLines(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	date := "2026-01-01"
	f.expense(t, f.yen, 500, date)
	f.transfer(t, 1000, 150000, date) // no lines: not in the summary
	pending, _ := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{Type: "expense", Status: "pending", Date: &date}, now)
	_ = f.repo.ReplaceLines(ctx, pending, []ledgerrepo.LineFields{{Amount: money.New(900, f.unit(f.usd))}})

	rows, err := f.repo.SummaryRows(ctx, filter.TxFilter{}, "confirmed")
	if err != nil || len(rows) != 1 || rows[0].Type != "expense" || rows[0].Minor != 500 || rows[0].Decimals != 0 ||
		rows[0].IsBase {
		t.Fatalf("confirmed = %+v, %v", rows, err)
	}
	if rows, _ := f.repo.SummaryRows(ctx, filter.TxFilter{}, "pending"); len(rows) != 1 || rows[0].Minor != 900 {
		t.Fatalf("pending = %+v", rows)
	}
}

func TestCategoryAndDescriptionCanBeSetAlone(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	other, _ := f.repo.InsertCategory(ctx, ledgerrepo.Category{Name: "Other", Type: "expense"}, now)
	id := f.expense(t, f.cash, 100, "2026-01-01")
	before, _ := f.repo.Transaction(ctx, id)
	chosen := before.Lines[0].CategoryID
	// One line the user categorised and one left without a category.
	if err := f.repo.ReplaceLines(ctx, id, []ledgerrepo.LineFields{
		{CategoryID: chosen, Amount: money.New(60, f.unit(f.usd))},
		{Amount: money.New(40, f.unit(f.usd))},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SetTransactionCategory(ctx, id, other, now); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SetTransactionDescription(ctx, id, "renamed", now); err != nil {
		t.Fatal(err)
	}
	got, _ := f.repo.Transaction(ctx, id)
	if *got.Lines[0].CategoryID != *chosen || *got.Lines[1].CategoryID != other {
		t.Fatalf("a categorised line keeps its category and an uncategorised one gets it: %+v", got.Lines)
	}
	if *got.Description != "renamed" || got.Payments[0].Amount.Minor() != -100 {
		t.Fatalf("tx = %+v", got)
	}
}

func TestExternalIDIsLookedUp(t *testing.T) {
	f := newFixture(t)
	ext := "csv:1:abc"
	if has, _ := f.repo.HasExternalID(ctx, ext); has {
		t.Fatal("found an id nothing was stored with")
	}
	id, _ := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{Type: "expense", ExternalID: &ext}, time.Now().UTC())
	if has, _ := f.repo.HasExternalID(ctx, ext); !has {
		t.Fatal("did not find the external id")
	}
	if got, _ := f.repo.Transaction(ctx, id); got.ExternalID == nil || *got.ExternalID != ext {
		t.Fatalf("external id = %v", got.ExternalID)
	}
}

func TestFilteredMoreReportsAnotherPageWithoutCounting(t *testing.T) {
	f := newFixture(t)
	for _, date := range []string{"2026-03-01", "2026-03-02", "2026-03-03"} {
		f.transfer(t, 1000, 150000, date)
	}
	for _, c := range []struct {
		limit int
		rows  int
		more  bool
	}{{2, 2, true}, {3, 3, false}, {10, 3, false}} {
		page, more, err := f.repo.FilteredMore(ctx, filter.TxFilter{}, c.limit, 0)
		if err != nil || len(page) != c.rows || more != c.more {
			t.Fatalf("limit %d: %d rows, more %v, err %v", c.limit, len(page), more, err)
		}
	}
}

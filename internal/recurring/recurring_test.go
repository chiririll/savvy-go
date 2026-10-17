package recurring

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/ledger"
)

func ptr[T any](v T) *T { return &v }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestNextRunDate(t *testing.T) {
	tests := []struct {
		name string
		rec  Recurring
		want string
	}{
		{"daily", Recurring{Frequency: "daily", Interval: 1, NextRunDate: "2024-02-28"}, "2024-02-29"},
		{"daily crosses a year end", Recurring{Frequency: "daily", Interval: 3, NextRunDate: "2023-12-30"}, "2024-01-02"},
		{"weekly keeps the weekday", Recurring{Frequency: "weekly", Interval: 2, NextRunDate: "2024-01-01"}, "2024-01-15"},
		{
			"weekly moves to the chosen weekday",
			Recurring{Frequency: "weekly", Interval: 1, DayOfWeek: ptr(5), NextRunDate: "2024-01-01"},
			"2024-01-12",
		},
		{
			"weekly on the same weekday",
			Recurring{Frequency: "weekly", Interval: 1, DayOfWeek: ptr(1), NextRunDate: "2024-01-01"},
			"2024-01-08",
		},
		{"monthly", Recurring{Frequency: "monthly", Interval: 1, NextRunDate: "2024-01-15"}, "2024-02-15"},
		{
			"monthly clamps to a short month",
			Recurring{Frequency: "monthly", Interval: 1, DayOfMonth: ptr(31), NextRunDate: "2024-01-31"},
			"2024-02-29",
		},
		{
			"monthly returns to the day after a short month",
			Recurring{Frequency: "monthly", Interval: 1, DayOfMonth: ptr(31), NextRunDate: "2024-02-29"},
			"2024-03-31",
		},
		{
			"monthly interval over a year end",
			Recurring{Frequency: "monthly", Interval: 3, NextRunDate: "2024-11-10"},
			"2025-02-10",
		},
		{"yearly", Recurring{Frequency: "yearly", Interval: 1, NextRunDate: "2023-06-01"}, "2024-06-01"},
		{
			"interval below one counts as one",
			Recurring{Frequency: "daily", Interval: 0, NextRunDate: "2024-01-01"},
			"2024-01-02",
		},
		{
			"unknown frequency steps by days",
			Recurring{Frequency: "fortnightly", Interval: 2, NextRunDate: "2024-01-01"},
			"2024-01-03",
		},
		{
			"a timestamp is cut to its date",
			Recurring{Frequency: "daily", Interval: 1, NextRunDate: "2024-01-01T10:00:00Z"},
			"2024-01-02",
		},
		{"an unreadable date is left alone", Recurring{Frequency: "daily", Interval: 1, NextRunDate: "soon"}, "soon"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rec.calculateNextRunDate(); got != tc.want {
				t.Errorf("next run = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestWithinSchedule(t *testing.T) {
	tests := []struct {
		name string
		rec  Recurring
		want bool
	}{
		{"active without an end", Recurring{IsActive: true, NextRunDate: "2030-01-01"}, true},
		{"active with an empty end", Recurring{IsActive: true, NextRunDate: "2030-01-01", EndDate: ptr("")}, true},
		{"inactive", Recurring{IsActive: false, NextRunDate: "2024-01-01"}, false},
		{"on the end date", Recurring{IsActive: true, NextRunDate: "2024-03-01", EndDate: ptr("2024-03-01")}, true},
		{"past the end date", Recurring{IsActive: true, NextRunDate: "2024-03-02", EndDate: ptr("2024-03-01")}, false},
	}
	for _, tc := range tests {
		if got := tc.rec.withinSchedule(); got != tc.want {
			t.Errorf("%s: withinSchedule = %v, want %v", tc.name, got, tc.want)
		}
	}
}

type env struct {
	ctx      context.Context
	svc      Service
	usd, eur ledger.Currency
	usdAcct  ledger.Account
	usdAcct2 ledger.Account
	eurAcct  ledger.Account
}

func newEnv(t *testing.T) *env {
	t.Helper()
	d := dbtest.Space(t)
	ctx := context.Background()
	curs := ledger.Currencies{DB: d}
	mk := func(code, rate string) ledger.Currency {
		c, err := curs.Create(ctx, ledger.Currency{Code: code, Name: code, Symbol: code, Decimals: 2, Rate: dec(rate)})
		if err != nil {
			t.Fatal(err)
		}
		return *c
	}
	e := &env{ctx: ctx, svc: Service{DB: d, Txs: ledger.Transactions{DB: d}}}
	e.usd, e.eur = mk("USD", "1"), mk("EUR", "2")
	accts := ledger.Accounts{DB: d}
	acct := func(name string, c ledger.Currency) ledger.Account {
		a, err := accts.Create(ctx, ledger.AccountInput{Name: name, Type: "cash", CurrencyID: c.ID, IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		return *a
	}
	e.usdAcct, e.usdAcct2, e.eurAcct = acct("usd", e.usd), acct("usd2", e.usd), acct("eur", e.eur)
	return e
}

func (e *env) create(t *testing.T, in Input) *Recurring {
	t.Helper()
	rec, err := e.svc.Create(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// pending lists the pending occurrences of a template.
func (e *env) pending(t *testing.T, recurringID int64) []int64 {
	t.Helper()
	rows, err := e.svc.DB.QueryContext(
		e.ctx,
		`SELECT id FROM transactions WHERE recurring_transaction_id = ? AND status = 'pending' ORDER BY id`,
		recurringID,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func (e *env) tx(t *testing.T, id int64) *ledger.Transaction {
	t.Helper()
	tx, err := e.svc.Txs.ByID(e.ctx, id)
	if err != nil || tx == nil {
		t.Fatalf("transaction %d: %v", id, err)
	}
	return tx
}

func TestCreateOpensOnePendingOccurrenceOnTheStartDate(t *testing.T) {
	e := newEnv(t)
	rec := e.create(
		t,
		Input{
			Type:        "expense",
			AccountID:   e.usdAcct.ID,
			Amount:      dec("12.50"),
			Frequency:   "monthly",
			StartDate:   "2024-01-15",
			Description: ptr("rent"),
		},
	)
	if !rec.IsActive || rec.Interval != 1 || rec.NextRunDate != "2024-01-15" {
		t.Errorf("defaults %+v", rec)
	}
	ids := e.pending(t, rec.ID)
	if len(ids) != 1 {
		t.Fatalf("pending occurrences %v, want one", ids)
	}
	tx := e.tx(t, ids[0])
	if tx.Amount.Decimal().String() != "12.5" || *tx.Date != "2024-01-15" || *tx.Description != "rent" ||
		tx.Status != "pending" {
		t.Errorf("occurrence %+v", tx)
	}
}

func TestCreateInactiveOrPastItsEndOpensNothing(t *testing.T) {
	e := newEnv(t)
	inactive := e.create(
		t,
		Input{
			Type:      "expense",
			AccountID: e.usdAcct.ID,
			Amount:    dec("1"),
			Frequency: "daily",
			StartDate: "2024-01-15",
			IsActive:  ptr(false),
		},
	)
	ended := e.create(
		t,
		Input{
			Type:      "expense",
			AccountID: e.usdAcct.ID,
			Amount:    dec("1"),
			Frequency: "daily",
			StartDate: "2024-01-15",
			EndDate:   ptr("2024-01-01"),
		},
	)
	for _, rec := range []*Recurring{inactive, ended} {
		if ids := e.pending(t, rec.ID); len(ids) != 0 {
			t.Errorf("template %d opened %v", rec.ID, ids)
		}
	}
}

func TestEstimatedFlagReachesTheOccurrence(t *testing.T) {
	e := newEnv(t)
	rec := e.create(
		t,
		Input{
			Type:        "expense",
			AccountID:   e.usdAcct.ID,
			Amount:      dec("5"),
			Frequency:   "monthly",
			StartDate:   "2024-01-15",
			IsEstimated: ptr(true),
		},
	)
	if !rec.IsEstimated || !e.tx(t, e.pending(t, rec.ID)[0]).IsEstimated {
		t.Error("an estimated template must open estimated occurrences")
	}
}

func TestAdvanceMovesTheScheduleAndOpensTheNextOccurrence(t *testing.T) {
	e := newEnv(t)
	rec := e.create(
		t,
		Input{
			Type:       "expense",
			AccountID:  e.usdAcct.ID,
			Amount:     dec("10"),
			Frequency:  "monthly",
			StartDate:  "2024-01-31",
			DayOfMonth: ptr(31),
		},
	)
	first := e.pending(t, rec.ID)[0]

	// The scheduler confirms the open occurrence, then asks the template to advance.
	if _, err := e.svc.Txs.Confirm(e.ctx, first, "2024-01-31", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.AdvanceAfterOccurrence(e.ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.svc.ByID(e.ctx, rec.ID)
	if got.NextRunDate != "2024-02-29" || got.LastRunDate == nil {
		t.Errorf("after advancing: next %s last %v", got.NextRunDate, got.LastRunDate)
	}
	ids := e.pending(t, rec.ID)
	if len(ids) != 1 || ids[0] == first {
		t.Fatalf("pending after advancing %v (first %d)", ids, first)
	}
	if date := *e.tx(t, ids[0]).Date; date != "2024-02-29" {
		t.Errorf("next occurrence dated %s", date)
	}

	// Advancing again while an occurrence is still open must not stack a second one.
	if err := e.svc.AdvanceAfterOccurrence(e.ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if n := len(e.pending(t, rec.ID)); n != 1 {
		t.Errorf("%d pending occurrences, want 1", n)
	}
}

func TestAdvancePastTheEndDateStopsTheSeries(t *testing.T) {
	e := newEnv(t)
	rec := e.create(
		t,
		Input{
			Type:      "expense",
			AccountID: e.usdAcct.ID,
			Amount:    dec("10"),
			Frequency: "daily",
			StartDate: "2024-01-01",
			EndDate:   ptr("2024-01-01"),
		},
	)
	if _, err := e.svc.Txs.Confirm(e.ctx, e.pending(t, rec.ID)[0], "2024-01-01", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.AdvanceAfterOccurrence(e.ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if ids := e.pending(t, rec.ID); len(ids) != 0 {
		t.Errorf("a finished series opened %v", ids)
	}
}

func TestEnsureUpcomingHealsOnlyTemplatesWithoutAnOpenOccurrence(t *testing.T) {
	e := newEnv(t)
	healthy := e.create(
		t,
		Input{Type: "expense", AccountID: e.usdAcct.ID, Amount: dec("1"), Frequency: "daily", StartDate: "2024-01-01"},
	)
	lost := e.create(
		t,
		Input{Type: "expense", AccountID: e.usdAcct.ID, Amount: dec("2"), Frequency: "daily", StartDate: "2024-01-01"},
	)
	off := e.create(
		t,
		Input{
			Type:      "expense",
			AccountID: e.usdAcct.ID,
			Amount:    dec("3"),
			Frequency: "daily",
			StartDate: "2024-01-01",
			IsActive:  ptr(false),
		},
	)
	if _, err := e.svc.Txs.Skip(e.ctx, e.pending(t, lost.ID)[0]); err != nil {
		t.Fatal(err)
	}
	before := e.pending(t, healthy.ID)

	for range 2 { // idempotent
		if err := e.svc.EnsureUpcoming(e.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(e.pending(t, lost.ID)); n != 1 {
		t.Errorf("lost occurrence recreated %d times, want 1", n)
	}
	if got := e.pending(t, healthy.ID); len(got) != 1 || got[0] != before[0] {
		t.Errorf("healthy template touched: %v -> %v", before, got)
	}
	if n := len(e.pending(t, off.ID)); n != 0 {
		t.Errorf("inactive template opened %d", n)
	}
}

func TestUpdateReplansTheOpenOccurrence(t *testing.T) {
	e := newEnv(t)
	rec := e.create(
		t,
		Input{Type: "expense", AccountID: e.usdAcct.ID, Amount: dec("10"), Frequency: "monthly", StartDate: "2024-01-15"},
	)
	open := e.pending(t, rec.ID)[0]

	upd, err := e.svc.Update(e.ctx, rec.ID, Input{Amount: dec("20"), Description: ptr("new")})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Amount.Decimal().String() != "20" || upd.Frequency != "monthly" || upd.AccountID != e.usdAcct.ID {
		t.Errorf("omitted fields must keep their value: %+v", upd)
	}
	if ids := e.pending(t, rec.ID); len(ids) != 1 || ids[0] != open {
		t.Fatalf("the open occurrence is rewritten, not duplicated: %v", ids)
	}
	if tx := e.tx(t, open); tx.Amount.Decimal().String() != "20" || *tx.Description != "new" {
		t.Errorf("occurrence %+v", tx)
	}
}

func TestUpdateStartDateRestartsTheSchedule(t *testing.T) {
	e := newEnv(t)
	rec := e.create(
		t,
		Input{Type: "expense", AccountID: e.usdAcct.ID, Amount: dec("10"), Frequency: "monthly", StartDate: "2024-01-15"},
	)
	upd, err := e.svc.Update(e.ctx, rec.ID, Input{StartDate: "2024-03-01"})
	if err != nil {
		t.Fatal(err)
	}
	if upd.NextRunDate != "2024-03-01" {
		t.Errorf("next run %s", upd.NextRunDate)
	}
	if date := *e.tx(t, e.pending(t, rec.ID)[0]).Date; date != "2024-03-01" {
		t.Errorf("open occurrence dated %s", date)
	}
}

func TestDeactivatingStopsOccurrencesAndReactivatingOpensOne(t *testing.T) {
	e := newEnv(t)
	rec := e.create(
		t,
		Input{Type: "expense", AccountID: e.usdAcct.ID, Amount: dec("10"), Frequency: "daily", StartDate: "2024-01-01"},
	)
	if _, err := e.svc.Txs.Skip(e.ctx, e.pending(t, rec.ID)[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Update(e.ctx, rec.ID, Input{IsActive: ptr(false)}); err != nil {
		t.Fatal(err)
	}
	if n := len(e.pending(t, rec.ID)); n != 0 {
		t.Fatalf("inactive template opened %d", n)
	}
	if _, err := e.svc.Update(e.ctx, rec.ID, Input{IsActive: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if n := len(e.pending(t, rec.ID)); n != 1 {
		t.Errorf("reactivated template opened %d, want 1", n)
	}
}

func TestUpdateOfAMissingTemplateIsNil(t *testing.T) {
	e := newEnv(t)
	got, err := e.svc.Update(e.ctx, 999, Input{Amount: dec("1")})
	if err != nil || got != nil {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestTagsFollowTheTemplateIntoItsOccurrences(t *testing.T) {
	e := newEnv(t)
	tag, err := (ledger.Tags{DB: e.svc.DB}).Create(e.ctx, "weekly")
	if err != nil {
		t.Fatal(err)
	}
	rec := e.create(
		t,
		Input{
			Type:      "expense",
			AccountID: e.usdAcct.ID,
			Amount:    dec("1"),
			Frequency: "daily",
			StartDate: "2024-01-01",
			TagIDs:    []int64{tag.ID},
		},
	)
	if len(rec.Tags) != 1 || rec.Tags[0].ID != tag.ID {
		t.Fatalf("template tags %+v", rec.Tags)
	}
	if tags := e.tx(t, e.pending(t, rec.ID)[0]).Tags; len(tags) != 1 || tags[0].ID != tag.ID {
		t.Errorf("occurrence tags %+v", tags)
	}
	// An update without tag ids keeps them; one with an empty list clears them.
	upd, _ := e.svc.Update(e.ctx, rec.ID, Input{Description: ptr("x")})
	if len(upd.Tags) != 1 {
		t.Errorf("tags lost by an unrelated update: %+v", upd.Tags)
	}
	upd, _ = e.svc.Update(e.ctx, rec.ID, Input{HasTagIDs: true})
	if len(upd.Tags) != 0 {
		t.Errorf("tags not cleared: %+v", upd.Tags)
	}
}

// A transfer between currencies delivers the converted amount, rounded to the
// destination's scale, unless the template names the amount itself.
func TestTransferDeliversTheConvertedOrTheStatedAmount(t *testing.T) {
	e := newEnv(t)
	converted := e.create(
		t,
		Input{
			Type:        "transfer",
			AccountID:   e.usdAcct.ID,
			ToAccountID: &e.eurAcct.ID,
			Amount:      dec("10"),
			Frequency:   "monthly",
			StartDate:   "2024-01-15",
		},
	)
	stated := e.create(
		t,
		Input{
			Type:        "transfer",
			AccountID:   e.usdAcct.ID,
			ToAccountID: &e.eurAcct.ID,
			Amount:      dec("10"),
			ToAmount:    ptr(dec("4.99")),
			Frequency:   "monthly",
			StartDate:   "2024-01-15",
		},
	)
	same := e.create(
		t,
		Input{
			Type:        "transfer",
			AccountID:   e.usdAcct.ID,
			ToAccountID: &e.usdAcct2.ID,
			Amount:      dec("10"),
			Frequency:   "monthly",
			StartDate:   "2024-01-15",
		},
	)

	want := map[int64]string{converted.ID: "5", stated.ID: "4.99", same.ID: "10"}
	for id, w := range want {
		ids := e.pending(t, id)
		if len(ids) == 0 {
			t.Fatalf("template %d opened nothing", id)
		}
		tx := e.tx(t, ids[0])
		if tx.ToAmount == nil || tx.ToAmount.Decimal().String() != w {
			t.Errorf("template %d delivers %v, want %s", id, tx.ToAmount, w)
		}
		if tx.Amount.Decimal().String() != "10" {
			t.Errorf("template %d takes %s from the source", id, tx.Amount.Decimal())
		}
	}
}

func TestAllUpcomingAndDelete(t *testing.T) {
	e := newEnv(t)
	a := e.create(
		t,
		Input{Type: "expense", AccountID: e.usdAcct.ID, Amount: dec("1"), Frequency: "daily", StartDate: "2024-01-01"},
	)
	b := e.create(
		t,
		Input{
			Type:      "expense",
			AccountID: e.usdAcct.ID,
			Amount:    dec("1"),
			Frequency: "daily",
			StartDate: "2024-01-02",
			IsActive:  ptr(false),
		},
	)
	c := e.create(
		t,
		Input{Type: "expense", AccountID: e.usdAcct.ID, Amount: dec("1"), Frequency: "daily", StartDate: "2024-01-03"},
	)

	all, err := e.svc.All(e.ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("All: %d, %v", len(all), err)
	}
	up, err := e.svc.Upcoming(e.ctx, 0)
	if err != nil || len(up) != 2 {
		t.Fatalf("Upcoming lists the active ones only: %d, %v", len(up), err)
	}
	if up[0].ID != a.ID || up[1].ID != c.ID {
		t.Errorf("Upcoming order: %d, %d", up[0].ID, up[1].ID)
	}
	one, _ := e.svc.Upcoming(e.ctx, 1)
	if len(one) != 1 {
		t.Errorf("limit not honoured: %d", len(one))
	}
	if up[0].Account == nil || up[0].Account.Name != "usd" {
		t.Errorf("accounts are loaded: %+v", up[0].Account)
	}

	if err := e.svc.Delete(e.ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.svc.ByID(e.ctx, b.ID); got != nil {
		t.Error("deleted template still found")
	}
	if all, _ := e.svc.All(e.ctx); len(all) != 2 {
		t.Errorf("All after delete: %d", len(all))
	}
}

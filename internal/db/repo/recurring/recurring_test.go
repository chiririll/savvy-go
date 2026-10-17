package recurring_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/dbtest"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	recurringrepo "savvy-go/internal/db/repo/recurring"
	"savvy-go/internal/money"
)

var ctx = context.Background()

type fixture struct {
	repo      recurringrepo.Repo
	usd       money.Unit
	acct, tag int64
	day       int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	d := dbtest.Space(t)
	l := ledgerrepo.Repo{DB: d}
	now := time.Now().UTC()
	cur, err := l.InsertCurrency(
		ctx,
		ledgerrepo.Currency{Code: "USD", Name: "Dollar", Symbol: "$", Decimals: 2, IsBase: true, Rate: decimal.NewFromInt(1)},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	unit := money.Unit{ID: cur, Decimals: 2}
	acct, _ := l.InsertAccount(
		ctx,
		ledgerrepo.AccountFields{
			Name:           "Cash",
			Type:           "cash",
			CurrencyID:     cur,
			InitialBalance: money.New(0, unit),
			IsActive:       true,
		},
		now,
	)
	tag, _ := l.InsertTag(ctx, "monthly", now)
	return &fixture{repo: recurringrepo.Repo{DB: d}, usd: unit, acct: acct, tag: tag}
}

func (f *fixture) fields() recurringrepo.Fields {
	day := 15
	desc := "Rent"
	return recurringrepo.Fields{
		Type: "expense", Frequency: "monthly", AccountID: f.acct, Amount: money.New(80000, f.usd),
		IsEstimated: true, IsActive: true, Description: &desc, Interval: 1, DayOfMonth: &day,
		StartDate: "2026-01-15", NextRunDate: "2026-01-15",
	}
}

func TestTemplateRoundTripKeepsOptionalFields(t *testing.T) {
	f := newFixture(t)
	if r, err := f.repo.ByID(ctx, 999); r != nil || err != nil {
		t.Fatalf("unknown template = %v, %v", r, err)
	}
	id, err := f.repo.Insert(ctx, f.fields(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	r, _ := f.repo.ByID(ctx, id)
	if r == nil || r.Amount != money.New(80000,
		f.usd) || r.ToAmount != nil || r.ToAccountID != nil || r.CategoryID != nil ||
		!r.IsEstimated || !r.IsActive || r.DayOfMonth == nil || *r.DayOfMonth != 15 || r.DayOfWeek != nil ||
		r.EndDate != nil || r.LastRunDate != nil ||
		r.Description == nil ||
		*r.Description != "Rent" {
		t.Fatalf("template = %+v", r)
	}

	up := f.fields()
	up.Amount, up.IsActive, up.NextRunDate = money.New(90000, f.usd), false, "2026-02-15"
	end := "2026-12-31"
	up.EndDate = &end
	if err := f.repo.Update(ctx, id, up, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	r, _ = f.repo.ByID(ctx, id)
	if r.Amount.Minor() != 90000 || r.IsActive || r.NextRunDate != "2026-02-15" || r.EndDate == nil || *r.EndDate != end {
		t.Fatalf("updated = %+v", r)
	}
}

func TestListCanKeepOnlyActiveTemplates(t *testing.T) {
	f := newFixture(t)
	_, _ = f.repo.Insert(ctx, f.fields(), time.Now().UTC())
	off := f.fields()
	off.IsActive = false
	_, _ = f.repo.Insert(ctx, off, time.Now().UTC())

	if all, _ := f.repo.List(ctx, false, 100); len(all) != 2 {
		t.Fatalf("all = %d", len(all))
	}
	if active, _ := f.repo.List(ctx, true, 100); len(active) != 1 || !active[0].IsActive {
		t.Fatalf("active = %+v", active)
	}
	if capped, _ := f.repo.List(ctx, false, 1); len(capped) != 1 {
		t.Fatalf("limit ignored: %d", len(capped))
	}
}

func TestAdvanceTagsDelete(t *testing.T) {
	f := newFixture(t)
	id, _ := f.repo.Insert(ctx, f.fields(), time.Now().UTC())

	if err := f.repo.Advance(ctx, id, "2026-01-15", "2026-02-15", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if r, _ := f.repo.ByID(ctx, id); r.NextRunDate != "2026-02-15" || r.LastRunDate == nil ||
		*r.LastRunDate != "2026-01-15" {
		t.Fatalf("advanced = %+v", r)
	}

	if err := f.repo.SetTags(ctx, id, []int64{f.tag}); err != nil {
		t.Fatal(err)
	}
	if tags, _ := f.repo.Tags(ctx, id); len(tags) != 1 || tags[0].Name != "monthly" || tags[0].CreatedAt == nil {
		t.Fatalf("tags = %+v", tags)
	}
	_ = f.repo.SetTags(ctx, id, nil)
	if tags, _ := f.repo.Tags(ctx, id); len(tags) != 0 {
		t.Fatalf("tags after clearing = %+v", tags)
	}

	if pending, _ := f.repo.PendingTransactionID(ctx, id); pending != 0 {
		t.Fatalf("pending = %d, want none", pending)
	}
	if err := f.repo.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	if r, _ := f.repo.ByID(ctx, id); r != nil {
		t.Fatal("deleted template is still there")
	}
}

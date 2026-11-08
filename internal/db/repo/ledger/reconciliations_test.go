package ledger_test

import (
	"testing"
	"time"

	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

func (f *fixture) point(t *testing.T, account int64, date string, minor int64, decimals int) *ledgerrepo.Reconciliation {
	t.Helper()
	cur := f.usd
	if account == f.yen {
		cur = f.jpy
	}
	got, err := f.repo.UpsertReconciliation(ctx, ledgerrepo.ReconciliationFields{
		AccountID: account, Date: date, ActualBalance: money.New(minor, money.Unit{ID: cur, Decimals: decimals}),
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestUpsertReconciliationReplacesSameDate(t *testing.T) {
	f := newFixture(t)
	note := "statement"
	first, err := f.repo.UpsertReconciliation(ctx, ledgerrepo.ReconciliationFields{
		AccountID: f.cash, Date: "2026-03-01", ActualBalance: money.New(10000, money.Unit{ID: f.usd, Decimals: 2}), Note: &note,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if first.CreatedAt == nil || first.Note == nil || *first.Note != note {
		t.Fatalf("first = %+v", first)
	}
	second := f.point(t, f.cash, "2026-03-01", -250, 2)
	if second.ID != first.ID || second.ActualBalance.Minor() != -250 || second.Note != nil {
		t.Fatalf("second = %+v, want same id, -250, no note", second)
	}
	list, _ := f.repo.AccountReconciliations(ctx, f.cash)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	if _, err := f.repo.UpsertReconciliation(ctx, ledgerrepo.ReconciliationFields{
		AccountID: 99999, Date: "2026-03-01", ActualBalance: money.New(1, money.Unit{ID: f.usd, Decimals: 2}),
	}, time.Now().UTC()); err == nil {
		t.Fatal("unknown account accepted")
	}
}

func TestReconciliationGetAndDelete(t *testing.T) {
	f := newFixture(t)
	p := f.point(t, f.yen, "2026-03-01", 5000, 0)
	got, err := f.repo.Reconciliation(ctx, p.ID)
	if err != nil || got == nil {
		t.Fatalf("get = %v, %v", got, err)
	}
	if want := money.New(5000, money.Unit{ID: f.jpy, Decimals: 0}); !got.ActualBalance.Equal(want) {
		t.Fatalf("balance = %v, want %v", got.ActualBalance, want)
	}
	if missing, err := f.repo.Reconciliation(ctx, 777); err != nil || missing != nil {
		t.Fatalf("missing = %v, %v", missing, err)
	}
	if ok, err := f.repo.DeleteReconciliation(ctx, p.ID); err != nil || !ok {
		t.Fatalf("delete = %v, %v", ok, err)
	}
	if ok, err := f.repo.DeleteReconciliation(ctx, p.ID); err != nil || ok {
		t.Fatalf("second delete = %v, %v", ok, err)
	}
}

func TestAccountReconciliationsOldestFirst(t *testing.T) {
	f := newFixture(t)
	f.point(t, f.cash, "2026-03-10", 3, 2)
	f.point(t, f.cash, "2026-01-10", 1, 2)
	f.point(t, f.cash, "2026-02-10", 2, 2)
	f.point(t, f.yen, "2026-01-01", 9, 0)
	list, err := f.repo.AccountReconciliations(ctx, f.cash)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].Date != "2026-01-10" || list[1].Date != "2026-02-10" || list[2].Date != "2026-03-10" {
		t.Fatalf("list = %+v", list)
	}
}

func TestLatestReconciliationOnOrBefore(t *testing.T) {
	f := newFixture(t)
	f.point(t, f.cash, "2026-01-10", 1, 2)
	f.point(t, f.cash, "2026-02-10", 2, 2)
	f.point(t, f.yen, "2026-02-20", 7, 0)
	for asOf, want := range map[string]int64{"2026-01-10": 1, "2026-02-09": 1, "2026-02-10": 2, "2026-12-31": 2} {
		got, err := f.repo.LatestReconciliation(ctx, f.cash, asOf)
		if err != nil || got == nil || got.ActualBalance.Minor() != want {
			t.Fatalf("as of %s = %+v, %v; want %d", asOf, got, err, want)
		}
	}
	if got, err := f.repo.LatestReconciliation(ctx, f.cash, "2026-01-09"); err != nil || got != nil {
		t.Fatalf("before first = %+v, %v", got, err)
	}
}

func TestReconciliationsBetween(t *testing.T) {
	f := newFixture(t)
	f.point(t, f.cash, "2026-01-10", 1, 2)
	f.point(t, f.yen, "2026-02-10", 2, 0)
	f.point(t, f.cash, "2026-03-10", 3, 2)
	all, _ := f.repo.ReconciliationsBetween(ctx, "", "")
	if len(all) != 3 {
		t.Fatalf("all = %d", len(all))
	}
	mid, err := f.repo.ReconciliationsBetween(ctx, "2026-02-10", "2026-03-09")
	if err != nil || len(mid) != 1 || mid[0].AccountID != f.yen || mid[0].ActualBalance.Unit().Decimals != 0 {
		t.Fatalf("mid = %+v, %v", mid, err)
	}
	from, _ := f.repo.ReconciliationsBetween(ctx, "2026-02-11", "")
	to, _ := f.repo.ReconciliationsBetween(ctx, "", "2026-01-10")
	if len(from) != 1 || from[0].Date != "2026-03-10" || len(to) != 1 || to[0].Date != "2026-01-10" {
		t.Fatalf("from = %+v to = %+v", from, to)
	}
}

func TestReconciliationsDieWithTheirAccount(t *testing.T) {
	f := newFixture(t)
	p := f.point(t, f.cash, "2026-01-10", 1, 2)
	if err := f.repo.DeleteAccount(ctx, f.cash); err != nil {
		t.Fatal(err)
	}
	if got, err := f.repo.Reconciliation(ctx, p.ID); err != nil || got != nil {
		t.Fatalf("after account delete = %+v, %v", got, err)
	}
}

func TestRestoreReconciliationKeepsIDAfterDelete(t *testing.T) {
	f := newFixture(t)
	p := f.point(t, f.cash, "2026-03-01", 10000, 2)
	if ok, err := f.repo.DeleteReconciliation(ctx, p.ID); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
	note := "back"
	p.Note = &note
	got, err := f.repo.RestoreReconciliation(ctx, *p)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != p.ID || got.Date != p.Date || got.Note == nil || *got.Note != "back" {
		t.Fatalf("restored = %+v, want id %d", got, p.ID)
	}
	// Restoring over the live row replaces its balance and note.
	p.ActualBalance = money.New(500, p.ActualBalance.Unit())
	got, err = f.repo.RestoreReconciliation(ctx, *p)
	if err != nil || got.ActualBalance.Minor() != 500 {
		t.Fatalf("overwrite = %+v, %v", got, err)
	}
}

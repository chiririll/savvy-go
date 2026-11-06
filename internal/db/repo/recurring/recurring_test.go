package recurring_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	recurringrepo "savvy-go/internal/db/repo/recurring"
)

var ctx = context.Background()

type fixture struct {
	repo   recurringrepo.Repo
	ledger ledgerrepo.Repo
	rule   int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	d := dbtest.Space(t)
	f := &fixture{repo: recurringrepo.Repo{DB: d}, ledger: ledgerrepo.Repo{DB: d}}
	id, err := f.repo.Insert(ctx, fields(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	f.rule = id
	return f
}

func fields() recurringrepo.Fields {
	return recurringrepo.Fields{
		Template: `{"type":"expense"}`,
		Schedule: `{"frequency":"monthly","interval":1,"start":"2026-01-15"}`,
		IsActive: true,
	}
}

// occur records a transaction as the occurrence of a due date.
func (f *fixture) occur(t *testing.T, due, status string) int64 {
	t.Helper()
	tx, err := f.ledger.InsertTransaction(ctx, ledgerrepo.TxFields{Type: "expense", Status: status, Date: &due}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if added, err := f.repo.AddInstance(ctx, f.rule, due, tx); err != nil || !added {
		t.Fatalf("add instance %s = %v, %v", due, added, err)
	}
	return tx
}

func TestRuleRoundTripKeepsTemplateAndSchedule(t *testing.T) {
	f := newFixture(t)
	if r, err := f.repo.ByID(ctx, 999); r != nil || err != nil {
		t.Fatalf("unknown rule = %v, %v", r, err)
	}
	r, _ := f.repo.ByID(ctx, f.rule)
	if r == nil || r.Template != fields().Template || r.Schedule != fields().Schedule || !r.IsActive ||
		r.Error != nil || r.LastDue != nil || r.LastRun != nil || r.CreatedAt == nil {
		t.Fatalf("rule = %+v", r)
	}

	up := fields()
	up.Template = `{"type":"income"}`
	if err := f.repo.Stop(ctx, f.rule, "the account is gone", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	r, _ = f.repo.ByID(ctx, f.rule)
	if r.IsActive || r.Error == nil || *r.Error != "the account is gone" {
		t.Fatalf("stopped = %+v", r)
	}

	// Saving the rule again clears the error.
	if err := f.repo.Update(ctx, f.rule, up, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	r, _ = f.repo.ByID(ctx, f.rule)
	if r.Template != up.Template || !r.IsActive || r.Error != nil {
		t.Fatalf("updated = %+v", r)
	}
}

func TestListCanKeepOnlyActiveRules(t *testing.T) {
	f := newFixture(t)
	off := fields()
	off.IsActive = false
	if _, err := f.repo.Insert(ctx, off, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if all, _ := f.repo.List(ctx, false); len(all) != 2 {
		t.Fatalf("all = %d", len(all))
	}
	if active, _ := f.repo.List(ctx, true); len(active) != 1 || active[0].ID != f.rule {
		t.Fatalf("active = %+v", active)
	}
}

func TestDatesAreDerivedFromInstances(t *testing.T) {
	f := newFixture(t)
	f.occur(t, "2026-01-15", "confirmed")
	f.occur(t, "2026-02-15", "skipped")
	pending := f.occur(t, "2026-03-15", "pending")

	r, _ := f.repo.ByID(ctx, f.rule)
	if r.LastDue == nil || *r.LastDue != "2026-03-15" {
		t.Fatalf("last due = %v, want the pending one", r.LastDue)
	}
	if r.LastRun == nil || *r.LastRun != "2026-01-15" {
		t.Fatalf("last run = %v, want the confirmed one, not skipped or pending", r.LastRun)
	}
	if r.PendingDue == nil || *r.PendingDue != "2026-03-15" {
		t.Fatalf("pending due = %v", r.PendingDue)
	}
	if p, _ := f.repo.Pending(ctx, f.rule); p == nil || p.DueDate != "2026-03-15" || *p.TransactionID != pending {
		t.Fatalf("pending = %+v", p)
	}
	if rule, due, ok, _ := f.repo.OccurrenceOf(ctx, pending); !ok || rule != f.rule || due != "2026-03-15" {
		t.Fatalf("occurrence of = %d %s %v", rule, due, ok)
	}
	if _, _, ok, _ := f.repo.OccurrenceOf(ctx, 999); ok {
		t.Fatal("a transaction no rule made belongs to a rule")
	}
}

func TestADueDateIsHandledOnce(t *testing.T) {
	f := newFixture(t)
	f.occur(t, "2026-01-15", "pending")
	again, _ := f.ledger.InsertTransaction(ctx, ledgerrepo.TxFields{Type: "expense", Status: "pending"}, time.Now().UTC())
	if added, err := f.repo.AddInstance(ctx, f.rule, "2026-01-15", again); err != nil || added {
		t.Fatalf("second instance of a due date = %v, %v", added, err)
	}
}

func TestADeletedOccurrenceKeepsItsDueDate(t *testing.T) {
	f := newFixture(t)
	f.occur(t, "2026-01-15", "confirmed")
	tx := f.occur(t, "2026-02-15", "pending")
	if err := f.ledger.DeleteTransaction(ctx, tx); err != nil {
		t.Fatal(err)
	}

	r, _ := f.repo.ByID(ctx, f.rule)
	if r.LastDue == nil || *r.LastDue != "2026-02-15" {
		t.Fatalf("last due after deleting the occurrence = %v, want it kept", r.LastDue)
	}
	if p, _ := f.repo.Pending(ctx, f.rule); p != nil {
		t.Fatalf("a deleted occurrence is still pending: %+v", p)
	}
	inst, _ := f.repo.Instances(ctx, f.rule)
	if len(inst) != 2 || inst[1].DueDate != "2026-02-15" || inst[1].TransactionID != nil {
		t.Fatalf("instances = %+v", inst)
	}
}

func TestMovingAnInstanceKeepsOtherDates(t *testing.T) {
	f := newFixture(t)
	f.occur(t, "2026-01-15", "confirmed")
	f.occur(t, "2026-02-15", "pending")

	if moved, _ := f.repo.MoveInstance(ctx, f.rule, "2026-02-15", "2026-01-15"); moved {
		t.Fatal("moved onto a date the rule already handled")
	}
	if moved, _ := f.repo.MoveInstance(ctx, f.rule, "2026-02-15", "2026-02-20"); !moved {
		t.Fatal("move to a free date failed")
	}
	if p, _ := f.repo.Pending(ctx, f.rule); p == nil || p.DueDate != "2026-02-20" {
		t.Fatalf("pending after the move = %+v", p)
	}
}

func TestDeletingARuleKeepsItsPendingTransaction(t *testing.T) {
	f := newFixture(t)
	tx := f.occur(t, "2026-01-15", "pending")
	if err := f.repo.Delete(ctx, f.rule); err != nil {
		t.Fatal(err)
	}
	if r, _ := f.repo.ByID(ctx, f.rule); r != nil {
		t.Fatal("deleted rule is still there")
	}
	if inst, _ := f.repo.Instances(ctx, f.rule); len(inst) != 0 {
		t.Fatalf("instances survived: %+v", inst)
	}
	if got, _ := f.ledger.Transaction(ctx, tx); got == nil || got.Status != "pending" {
		t.Fatalf("the transaction = %+v, want it kept as an ordinary pending one", got)
	}
}

package recurring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// fakeLedger makes the pending transactions of templates the way internal/ledger
// will: one transaction row dated with the due date. A template that contains
// "broken" does not validate.
type fakeLedger struct{ d store.DB }

func (f fakeLedger) Validate(_ context.Context, template json.RawMessage) error {
	if bytes.Contains(template, []byte("broken")) {
		return errors.New("the account does not exist")
	}
	return nil
}

func (f fakeLedger) Create(ctx context.Context, template json.RawMessage, due string) (int64, error) {
	desc := string(template)
	return ledgerrepo.Repo{DB: f.d}.InsertTransaction(ctx,
		ledgerrepo.TxFields{Type: "expense", Status: "pending", Description: &desc, Date: &due}, time.Now().UTC())
}

func (f fakeLedger) Replan(ctx context.Context, txID int64, template json.RawMessage, due string) error {
	desc := string(template)
	return ledgerrepo.Repo{DB: f.d}.UpdateTransaction(ctx, txID,
		ledgerrepo.TxFields{Type: "expense", Description: &desc, Date: &due}, time.Now().UTC())
}

type env struct {
	ctx    context.Context
	svc    Service
	ledger ledgerrepo.Repo
}

func newEnv(t *testing.T) *env {
	t.Helper()
	d := dbtest.Space(t)
	return &env{
		ctx: context.Background(), ledger: ledgerrepo.Repo{DB: d},
		svc: Service{DB: d, Ledger: func(h store.DB) Occurrences { return fakeLedger{h} }},
	}
}

func monthly(start string) Schedule {
	return Schedule{Frequency: "monthly", Interval: 1, Start: start}
}

func (e *env) create(t *testing.T, template string, s Schedule) *Rule {
	t.Helper()
	r, err := e.svc.Create(e.ctx, Input{Template: json.RawMessage(template), Schedule: s})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// pending is the open occurrence of a rule: its due date and transaction.
func (e *env) pending(t *testing.T, rule int64) (string, *ledgerrepo.Tx) {
	t.Helper()
	p, err := e.svc.repo(e.svc.DB).Pending(e.ctx, rule)
	if err != nil || p == nil {
		t.Fatalf("pending of rule %d = %v, %v", rule, p, err)
	}
	tx, _ := e.ledger.Transaction(e.ctx, *p.TransactionID)
	return p.DueDate, tx
}

func (e *env) noPending(t *testing.T, rule int64) {
	t.Helper()
	if p, _ := e.svc.repo(e.svc.DB).Pending(e.ctx, rule); p != nil {
		t.Fatalf("rule %d has a pending occurrence: %+v", rule, p)
	}
}

// settle confirms or skips the pending occurrence of a rule, then advances,
// as the ledger does in one database transaction.
func (e *env) settle(t *testing.T, rule int64, skip bool) {
	t.Helper()
	_, tx := e.pending(t, rule)
	err := store.Tx(e.ctx, e.svc.DB, func(d store.DB) error {
		l := ledgerrepo.Repo{DB: d}
		var err error
		if skip {
			err = l.SkipTransaction(e.ctx, tx.ID, time.Now().UTC())
		} else {
			err = l.ConfirmTransaction(e.ctx, tx.ID, *tx.Date, time.Now().UTC())
		}
		if err != nil {
			return err
		}
		return e.svc.AdvanceAfterOccurrence(e.ctx, d, rule)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateOpensPendingOccurrenceAtStart(t *testing.T) {
	e := newEnv(t)
	r := e.create(t, `{"type":"expense"}`, monthly("2026-01-15"))
	due, tx := e.pending(t, r.ID)
	if due != "2026-01-15" || tx.Date == nil || *tx.Date != "2026-01-15" || tx.Status != "pending" {
		t.Fatalf("occurrence = %s %+v", due, tx)
	}
	if r.NextRunDate == nil || *r.NextRunDate != "2026-01-15" || r.LastRunDate != nil || !r.IsActive || r.Error != nil {
		t.Fatalf("rule = %+v", r)
	}
}

func TestConfirmAndSkipCreateTheNextOccurrence(t *testing.T) {
	e := newEnv(t)
	r := e.create(t, `{"type":"expense"}`, monthly("2026-01-15"))

	e.settle(t, r.ID, false)
	if due, _ := e.pending(t, r.ID); due != "2026-02-15" {
		t.Fatalf("after confirming, next = %s", due)
	}
	e.settle(t, r.ID, true)
	if due, _ := e.pending(t, r.ID); due != "2026-03-15" {
		t.Fatalf("after skipping, next = %s", due)
	}
	got, _ := e.svc.ByID(e.ctx, r.ID)
	if got.LastRunDate == nil || *got.LastRunDate != "2026-01-15" || *got.NextRunDate != "2026-03-15" {
		t.Fatalf("rule = last %v next %v, want the confirmed 2026-01-15 and the open 2026-03-15",
			got.LastRunDate, got.NextRunDate)
	}
	// Advancing again, as a crash recovery or a double call, makes no second occurrence.
	if err := e.svc.AdvanceAfterOccurrence(e.ctx, e.svc.DB, r.ID); err != nil {
		t.Fatal(err)
	}
	if inst, _ := e.svc.repo(e.svc.DB).Instances(e.ctx, r.ID); len(inst) != 3 {
		t.Fatalf("instances = %+v", inst)
	}
}

func TestADeletedOccurrenceIsNotGeneratedAgain(t *testing.T) {
	e := newEnv(t)
	r := e.create(t, `{"type":"expense"}`, monthly("2026-01-15"))
	e.settle(t, r.ID, false)
	_, tx := e.pending(t, r.ID) // 2026-02-15

	// Deleting the pending occurrence acts like a skip: the ledger deletes it and advances.
	err := store.Tx(e.ctx, e.svc.DB, func(d store.DB) error {
		if err := (ledgerrepo.Repo{DB: d}).DeleteTransaction(e.ctx, tx.ID); err != nil {
			return err
		}
		return e.svc.AdvanceAfterOccurrence(e.ctx, d, r.ID)
	})
	if err != nil {
		t.Fatal(err)
	}
	if due, _ := e.pending(t, r.ID); due != "2026-03-15" {
		t.Fatalf("next after deleting 2026-02-15 = %s", due)
	}

	// The job finds nothing to make, and the deleted date stays handled.
	if err := e.svc.EnsureUpcoming(e.ctx); err != nil {
		t.Fatal(err)
	}
	inst, _ := e.svc.repo(e.svc.DB).Instances(e.ctx, r.ID)
	if len(inst) != 3 || inst[1].DueDate != "2026-02-15" || inst[1].TransactionID != nil {
		t.Fatalf("instances = %+v", inst)
	}
}

func TestTheScheduleEndsTheSeries(t *testing.T) {
	e := newEnv(t)
	s := monthly("2026-01-15")
	s.End = ptr("2026-02-20")
	r := e.create(t, `{"type":"expense"}`, s)
	e.settle(t, r.ID, false)
	e.settle(t, r.ID, false) // 2026-02-15, the last one
	e.noPending(t, r.ID)
	if got, _ := e.svc.ByID(e.ctx, r.ID); got.NextRunDate != nil {
		t.Fatalf("an ended rule has a next date: %v", *got.NextRunDate)
	}

	// A rule whose start is already past its end opens nothing.
	past := monthly("2026-05-01")
	past.End = ptr("2026-04-01")
	if _, err := e.svc.Create(e.ctx, Input{Template: json.RawMessage(`{}`), Schedule: past}); !errors.Is(err, ErrSchedule) {
		t.Fatalf("end before start = %v, want ErrSchedule", err)
	}
}

func TestEnsureUpcomingRecoversAfterACrash(t *testing.T) {
	e := newEnv(t)
	r := e.create(t, `{"type":"expense"}`, monthly("2026-01-15"))
	off := false
	inactive, _ := e.svc.Create(e.ctx, Input{Template: json.RawMessage(`{}`), Schedule: monthly("2026-01-15"), IsActive: &off})
	e.noPending(t, inactive.ID)

	// The confirmation committed, the next occurrence did not.
	_, tx := e.pending(t, r.ID)
	if err := e.ledger.ConfirmTransaction(e.ctx, tx.ID, *tx.Date, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	e.noPending(t, r.ID)

	if err := e.svc.EnsureUpcoming(e.ctx); err != nil {
		t.Fatal(err)
	}
	if due, _ := e.pending(t, r.ID); due != "2026-02-15" {
		t.Fatalf("recovered occurrence = %s", due)
	}
	e.noPending(t, inactive.ID)
	// A second run changes nothing.
	if err := e.svc.EnsureUpcoming(e.ctx); err != nil {
		t.Fatal(err)
	}
	if inst, _ := e.svc.repo(e.svc.DB).Instances(e.ctx, r.ID); len(inst) != 2 {
		t.Fatalf("instances = %+v", inst)
	}
}

// The startup job stops a rule whose template or schedule was broken outside the
// service, with the reason, before it makes anything, and goes on with the others.
func TestEnsureUpcomingStopsBrokenRulesWithTheReason(t *testing.T) {
	e := newEnv(t)
	good := e.create(t, `{"type":"expense"}`, monthly("2026-01-15"))
	badTemplate := e.create(t, `{"type":"expense"}`, monthly("2026-01-15"))
	badSchedule := e.create(t, `{"type":"expense"}`, monthly("2026-01-15"))
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := e.svc.DB.ExecContext(e.ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	// Their open occurrences are gone (a restored older backup), so a job that did not stop them would make new ones.
	for _, id := range []int64{badTemplate.ID, badSchedule.ID} {
		exec(`DELETE FROM transactions WHERE id IN (SELECT transaction_id FROM recurring_instances WHERE recurring_transaction_id = ?)`, id)
	}
	exec(`UPDATE recurring_transactions SET template = '{"broken":true}' WHERE id = ?`, badTemplate.ID)
	exec(`UPDATE recurring_transactions SET schedule = '{"frequency":"hourly"}' WHERE id = ?`, badSchedule.ID)

	if err := e.svc.EnsureUpcoming(e.ctx); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		rule   int64
		reason string
	}{{badTemplate.ID, "the account does not exist"}, {badSchedule.ID, "invalid schedule"}} {
		got, _ := e.svc.ByID(e.ctx, tc.rule)
		if got.IsActive || got.Error == nil || !bytes.Contains([]byte(*got.Error), []byte(tc.reason)) {
			t.Fatalf("rule %d = active %v error %v, want stopped with %q", tc.rule, got.IsActive, got.Error, tc.reason)
		}
		e.noPending(t, tc.rule)
	}
	if got, _ := e.svc.ByID(e.ctx, good.ID); !got.IsActive || got.Error != nil {
		t.Fatalf("the good rule was stopped: %+v", got)
	}
	if due, _ := e.pending(t, good.ID); due != "2026-01-15" {
		t.Fatalf("the good rule's occurrence = %s, want it kept", due)
	}
}

func TestUpdateReplansOpenOccurrenceAndClearsError(t *testing.T) {
	e := newEnv(t)
	r := e.create(t, `{"type":"expense","n":1}`, monthly("2026-01-15"))
	_, tx := e.pending(t, r.ID)

	if err := e.svc.repo(e.svc.DB).Stop(e.ctx, r.ID, "was broken", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.Update(e.ctx, r.ID, Input{Template: json.RawMessage(`{"type":"expense","n":2}`), Schedule: monthly("2026-01-15")})
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsActive || got.Error != nil {
		t.Fatalf("rule = %+v, want active with the error cleared", got)
	}
	_, now := e.pending(t, r.ID)
	if now.ID != tx.ID || now.Description == nil || *now.Description != `{"type":"expense","n":2}` {
		t.Fatalf("occurrence = %+v, want the same transaction with the new template", now)
	}
}

func TestUpdateFirstOccurrenceFollowsNewStart(t *testing.T) {
	e := newEnv(t)
	r := e.create(t, `{}`, monthly("2026-01-15"))
	if _, err := e.svc.Update(e.ctx, r.ID, Input{Template: json.RawMessage(`{}`), Schedule: monthly("2026-02-01")}); err != nil {
		t.Fatal(err)
	}
	due, tx := e.pending(t, r.ID)
	if due != "2026-02-01" || *tx.Date != "2026-02-01" {
		t.Fatalf("occurrence = %s %v, want it moved to the new start", due, *tx.Date)
	}

	// Once the series has history, a new start does not move what is already handled.
	e.settle(t, r.ID, false)
	if _, err := e.svc.Update(e.ctx, r.ID, Input{Template: json.RawMessage(`{}`), Schedule: monthly("2025-06-01")}); err != nil {
		t.Fatal(err)
	}
	if due, _ := e.pending(t, r.ID); due != "2026-03-01" {
		t.Fatalf("occurrence = %s, want it unmoved", due)
	}
}

func TestDeactivatingStopsOccurrencesReactivatingOpensOne(t *testing.T) {
	e := newEnv(t)
	r := e.create(t, `{}`, monthly("2026-01-15"))
	off := false
	if _, err := e.svc.Update(e.ctx, r.ID, Input{Template: json.RawMessage(`{}`), Schedule: monthly("2026-01-15"), IsActive: &off}); err != nil {
		t.Fatal(err)
	}
	e.settle(t, r.ID, false)
	e.noPending(t, r.ID)

	if _, err := e.svc.Update(e.ctx, r.ID, Input{Template: json.RawMessage(`{}`), Schedule: monthly("2026-01-15")}); err != nil {
		t.Fatal(err)
	}
	if due, _ := e.pending(t, r.ID); due != "2026-02-15" {
		t.Fatalf("reactivated occurrence = %s", due)
	}
}

func TestRefusedRulesStoreNothing(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.Create(e.ctx, Input{Template: json.RawMessage(`{"broken":1}`), Schedule: monthly("2026-01-15")}); err == nil {
		t.Fatal("a template the ledger refuses was saved")
	}
	if _, err := e.svc.Create(e.ctx, Input{Template: json.RawMessage(`{}`), Schedule: Schedule{Frequency: "daily"}}); !errors.Is(err, ErrSchedule) {
		t.Fatalf("schedule without an interval = %v, want ErrSchedule", err)
	}
	if all, _ := e.svc.All(e.ctx); len(all) != 0 {
		t.Fatalf("rules = %+v", all)
	}
	if e.ledger.CountTransactions(e.ctx) != 0 {
		t.Fatal("an occurrence was left behind")
	}
	if got, err := e.svc.Update(e.ctx, 999, Input{Template: json.RawMessage(`{}`), Schedule: monthly("2026-01-15")}); got != nil || err != nil {
		t.Fatalf("update of a missing rule = %v, %v", got, err)
	}
}

func TestUpcomingOrdersByNextDate(t *testing.T) {
	e := newEnv(t)
	late := e.create(t, `{}`, monthly("2026-03-01"))
	early := e.create(t, `{}`, monthly("2026-01-01"))
	mid := e.create(t, `{}`, monthly("2026-02-01"))
	off := false
	_, _ = e.svc.Create(e.ctx, Input{Template: json.RawMessage(`{}`), Schedule: monthly("2025-01-01"), IsActive: &off})

	got, err := e.svc.Upcoming(e.ctx, 2)
	if err != nil || len(got) != 2 || got[0].ID != early.ID || got[1].ID != mid.ID {
		t.Fatalf("upcoming = %+v, %v (late is %d)", got, err, late.ID)
	}
	if all, _ := e.svc.All(e.ctx); len(all) != 4 {
		t.Fatalf("all = %d", len(all))
	}
}

func TestDeleteRuleKeepsPendingTransaction(t *testing.T) {
	e := newEnv(t)
	r := e.create(t, `{}`, monthly("2026-01-15"))
	_, tx := e.pending(t, r.ID)
	if rule, _ := e.svc.RuleOf(e.ctx, tx.ID); rule != r.ID {
		t.Fatalf("rule of the occurrence = %d", rule)
	}
	if rule, _ := e.svc.RuleOf(e.ctx, 999); rule != 0 {
		t.Fatalf("rule of a transaction nobody made = %d", rule)
	}
	if err := e.svc.Delete(e.ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.ledger.Transaction(e.ctx, tx.ID); got == nil || got.Status != "pending" {
		t.Fatalf("occurrence after deleting the rule = %+v", got)
	}
	if rule, _ := e.svc.RuleOf(e.ctx, tx.ID); rule != 0 {
		t.Fatal("the transaction still belongs to the deleted rule")
	}
}

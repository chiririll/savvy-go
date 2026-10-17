package automation_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	automationrepo "savvy-go/internal/db/repo/automation"
)

var ctx = context.Background()

func rule(name, trigger string, priority int, active bool) automationrepo.Fields {
	return automationrepo.Fields{
		Name: name, TriggerType: trigger, Priority: priority, Conditions: `{"match":"all"}`, Actions: `[]`,
		IsActive: active,
	}
}

func TestRulesAreListedByPriorityFiltered(t *testing.T) {
	r := automationrepo.Repo{DB: dbtest.Space(t)}
	now := time.Now().UTC()
	late, err := r.Insert(ctx, rule("late", "on_transaction_create", 90, true), now)
	if err != nil {
		t.Fatal(err)
	}
	early, _ := r.Insert(ctx, rule("early", "on_transaction_create", 10, true), now)
	_, _ = r.Insert(ctx, rule("off", "on_transaction_update", 50, false), now)

	all, _ := r.List(ctx, automationrepo.Filter{})
	if len(all) != 3 || all[0].ID != early || all[1].Name != "off" || all[2].ID != late {
		t.Fatalf("order = %+v", all)
	}
	if on, _ := r.List(ctx, automationrepo.Filter{ActiveOnly: true, TriggerType: "on_transaction_create"}); len(on) != 2 {
		t.Fatalf("active created rules = %d", len(on))
	}
	one, _ := r.List(ctx, automationrepo.Filter{ID: late})
	if len(one) != 1 || one[0].Conditions != `{"match":"all"}` || one[0].Description != nil || one[0].LastRunAt != nil ||
		one[0].CreatedAt == nil || !one[0].IsActive || one[0].StopProcessing {
		t.Fatalf("rule = %+v", one)
	}
}

func TestRuleEditing(t *testing.T) {
	r := automationrepo.Repo{DB: dbtest.Space(t)}
	now := time.Now().UTC()
	id, _ := r.Insert(ctx, rule("a", "on_transaction_create", 50, true), now)

	desc := "Groceries"
	up := rule("renamed", "on_transaction_create", 50, true)
	up.Description, up.StopProcessing = &desc, true
	if err := r.Update(ctx, id, up, now); err != nil {
		t.Fatal(err)
	}
	if err := r.SetPriority(ctx, id, 5); err != nil {
		t.Fatal(err)
	}
	if err := r.SetActive(ctx, id, false, now); err != nil {
		t.Fatal(err)
	}
	if err := r.CountRun(ctx, id, now); err != nil {
		t.Fatal(err)
	}
	got, _ := r.List(ctx, automationrepo.Filter{ID: id})
	g := got[0]
	if g.Name != "renamed" || g.Description == nil || *g.Description != desc || !g.StopProcessing || g.Priority != 5 ||
		g.IsActive || g.RunsCount != 1 || g.LastRunAt == nil {
		t.Fatalf("rule = %+v", g)
	}

	if err := r.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	if left, _ := r.List(ctx, automationrepo.Filter{}); len(left) != 0 {
		t.Fatalf("%d rules left", len(left))
	}
}

func TestLogsKeepTheirOptionalParts(t *testing.T) {
	r := automationrepo.Repo{DB: dbtest.Space(t)}
	now := time.Now().UTC().Truncate(time.Second)
	id, _ := r.Insert(ctx, rule("a", "on_transaction_create", 50, true), now)

	if err := r.AddLog(ctx, id, 42, "success", `[{"type":"set_category"}]`, "", now); err != nil {
		t.Fatal(err)
	}
	if err := r.AddLog(ctx, id, 43, "error", "", "boom", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	logs, err := r.Logs(ctx, id)
	if err != nil || len(logs) != 2 {
		t.Fatalf("logs = %+v, %v", logs, err)
	}
	byTx := map[int64]automationrepo.Log{}
	for _, l := range logs {
		byTx[*l.TriggerEntityID] = l
	}
	ok, bad := byTx[42], byTx[43]
	if ok.Status != "success" || ok.ActionsExecuted != `[{"type":"set_category"}]` || ok.ErrorMessage != nil ||
		ok.TriggerEntityType == nil || *ok.TriggerEntityType != "Transaction" || !ok.CreatedAt.Equal(now) {
		t.Fatalf("success log = %+v", ok)
	}
	if bad.ActionsExecuted != "" || bad.ErrorMessage == nil || *bad.ErrorMessage != "boom" {
		t.Fatalf("error log = %+v", bad)
	}
}

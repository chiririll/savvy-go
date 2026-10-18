package automation

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/db/filter"
	"savvy-go/internal/ledger"
)

func ptr[T any](v T) *T { return &v }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

type env struct {
	ctx          context.Context
	svc          Service
	usd          ledger.Currency
	wallet, bank ledger.Account
	food, fun    ledger.Category
	tagA, tagB   ledger.Tag
}

func newEnv(t *testing.T) *env {
	t.Helper()
	d := dbtest.Space(t)
	ctx := context.Background()
	e := &env{ctx: ctx, svc: Service{DB: d, Txs: ledger.Transactions{DB: d}}}
	cur, err := (ledger.Currencies{DB: d}).Create(
		ctx,
		ledger.Currency{Code: "USD", Name: "USD", Symbol: "$", Decimals: 2, Rate: dec("1")},
	)
	if err != nil {
		t.Fatal(err)
	}
	e.usd = *cur
	accts := ledger.Accounts{DB: d}
	acct := func(name string) ledger.Account {
		a, err := accts.Create(
			ctx,
			ledger.AccountInput{Name: name, Type: "cash", CurrencyID: cur.ID, InitialBalance: dec("1000"), IsActive: true},
		)
		if err != nil {
			t.Fatal(err)
		}
		return *a
	}
	e.wallet, e.bank = acct("wallet"), acct("bank")
	cats := ledger.Categories{DB: d}
	cat := func(name string) ledger.Category {
		c, err := cats.Create(ctx, ledger.Category{Name: name, Type: "expense"})
		if err != nil {
			t.Fatal(err)
		}
		return *c
	}
	e.food, e.fun = cat("Food"), cat("Fun")
	tags := ledger.Tags{DB: d}
	tag := func(name string) ledger.Tag {
		tg, err := tags.Create(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		return *tg
	}
	e.tagA, e.tagB = tag("a"), tag("b")
	return e
}

func (e *env) expense(t *testing.T, amount, desc string) *ledger.Transaction {
	t.Helper()
	date := "2024-01-15"
	tx, err := e.svc.Txs.Create(
		e.ctx,
		ledger.TxInput{Type: "expense", AccountID: e.wallet.ID, Amount: dec(amount), Description: &desc, Date: &date},
	)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func (e *env) reload(t *testing.T, id int64) *ledger.Transaction {
	t.Helper()
	tx, err := e.svc.Txs.ByID(e.ctx, id)
	if err != nil || tx == nil {
		t.Fatalf("transaction %d: %v", id, err)
	}
	return tx
}

func (e *env) rule(t *testing.T, in Input) *Rule {
	t.Helper()
	if in.TriggerType == "" {
		in.TriggerType = "on_transaction_create"
	}
	r, err := e.svc.Create(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func when(field, op string, value any) map[string]any {
	return map[string]any{"match": "all", "conditions": []any{map[string]any{"field": field, "op": op, "value": value}}}
}

func tagIDs(tx *ledger.Transaction) map[int64]bool {
	out := map[int64]bool{}
	for _, tg := range tx.Tags {
		out[tg.ID] = true
	}
	return out
}

func TestCreateFillsDefaults(t *testing.T) {
	e := newEnv(t)
	r := e.rule(t, Input{Name: "r"})
	if r.Priority != 50 || !r.IsActive || r.StopProcessing || r.RunsCount != 0 {
		t.Errorf("defaults %+v", r)
	}
	if len(r.Actions) != 0 {
		t.Errorf("actions %v", r.Actions)
	}
	if r.Conditions["match"] != "all" {
		t.Errorf("a rule without conditions matches all: %v", r.Conditions)
	}
}

// The older clients said "operator"; it is stored as "op".
func TestConditionsKeepTheirMeaningWhenStored(t *testing.T) {
	e := newEnv(t)
	r := e.rule(t, Input{Name: "r", Conditions: map[string]any{
		"conditions": []any{map[string]any{"field": "type", "operator": "equals", "value": "expense"}},
	}})
	conds := r.Conditions["conditions"].([]any)
	first := conds[0].(map[string]any)
	if first["op"] != "equals" || first["operator"] != nil {
		t.Errorf("condition %v", first)
	}
	if !evaluateConditions(r.Conditions, e.expense(t, "1", "x")) {
		t.Error("a stored rule must still match what it matched when written")
	}
}

func TestUpdateKeepsWhatIsNotSent(t *testing.T) {
	e := newEnv(t)
	r := e.rule(t, Input{
		Name: "r", Priority: 7, Conditions: when("type", "equals", "expense"),
		Actions: []map[string]any{{"type": "set_category", "category_id": e.food.ID}}, StopProcessing: ptr(true),
	})
	upd, err := e.svc.Update(e.ctx, r.ID, Input{Name: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != "renamed" || upd.Priority != 7 || !upd.StopProcessing || !upd.IsActive ||
		upd.TriggerType != r.TriggerType {
		t.Errorf("updated %+v", upd)
	}
	if len(upd.Actions) != 1 || len(upd.Conditions["conditions"].([]any)) != 1 {
		t.Errorf("actions or conditions lost: %+v", upd)
	}
	if got, err := e.svc.Update(e.ctx, 999, Input{Name: "x"}); got != nil || err != nil {
		t.Errorf("unknown rule: %v, %v", got, err)
	}
}

func TestToggleReorderDelete(t *testing.T) {
	e := newEnv(t)
	a, b := e.rule(t, Input{Name: "a", Priority: 10}), e.rule(t, Input{Name: "b", Priority: 20})

	off, err := e.svc.Toggle(e.ctx, a.ID)
	if err != nil || off.IsActive {
		t.Fatalf("toggled off: %+v, %v", off, err)
	}
	on, _ := e.svc.Toggle(e.ctx, a.ID)
	if !on.IsActive {
		t.Error("toggling twice restores the rule")
	}
	if got, err := e.svc.Toggle(e.ctx, 999); got != nil || err != nil {
		t.Errorf("unknown rule: %v, %v", got, err)
	}

	err = e.svc.Reorder(e.ctx, []struct {
		ID       int64 `json:"id"`
		Priority int   `json:"priority"`
	}{{b.ID, 1}, {a.ID, 2}})
	if err != nil {
		t.Fatal(err)
	}
	all, _ := e.svc.All(e.ctx)
	if len(all) != 2 || all[0].ID != b.ID || all[1].ID != a.ID {
		t.Errorf("rules are listed by priority: %+v", all)
	}

	if err := e.svc.Delete(e.ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.svc.ByID(e.ctx, a.ID); got != nil {
		t.Error("deleted rule still found")
	}
}

func TestProcessAppliesEveryAction(t *testing.T) {
	e := newEnv(t)
	e.rule(t, Input{
		Name: "everything", Conditions: when("description", "contains", "coffee"),
		Actions: []map[string]any{
			{"type": "set_category", "category_id": float64(e.food.ID)},
			{"type": "add_tags", "tag_ids": []any{float64(e.tagA.ID), float64(e.tagB.ID)}},
			{"type": "remove_tags", "tag_ids": []any{float64(e.tagB.ID)}},
			{"type": "set_description", "value": "Cafe: {{transaction.description}} ({{transaction.amount}})"},
		},
	})
	tx := e.expense(t, "4.50", "Morning coffee")
	e.svc.Process(e.ctx, "on_transaction_create", tx)

	got := e.reload(t, tx.ID)
	if got.CategoryID == nil || *got.CategoryID != e.food.ID {
		t.Errorf("category %v", got.CategoryID)
	}
	if tags := tagIDs(got); !tags[e.tagA.ID] || tags[e.tagB.ID] || len(tags) != 1 {
		t.Errorf("tags %v (a added, b added then removed)", got.Tags)
	}
	if got.Description == nil || *got.Description != "Cafe: Morning coffee (4.5)" {
		t.Errorf("description %v", got.Description)
	}
}

func TestProcessSkipsInapplicableRules(t *testing.T) {
	e := newEnv(t)
	setFood := []map[string]any{{"type": "set_category", "category_id": float64(e.food.ID)}}
	e.rule(t, Input{Name: "no match", Conditions: when("description", "contains", "tesla"), Actions: setFood})
	e.rule(t, Input{Name: "other trigger", TriggerType: "on_transaction_update", Actions: setFood})
	e.rule(t, Input{Name: "off", IsActive: ptr(false), Actions: setFood})

	tx := e.expense(t, "4", "coffee")
	e.svc.Process(e.ctx, "on_transaction_create", tx)
	if got := e.reload(t, tx.ID); got.CategoryID != nil {
		t.Errorf("a rule that should not run set category %v", *got.CategoryID)
	}

	e.rule(t, Input{Name: "on", Actions: setFood})
	pending := "pending"
	date := "2024-02-01"
	plan, err := e.svc.Txs.Create(
		e.ctx,
		ledger.TxInput{Type: "expense", AccountID: e.wallet.ID, Amount: dec("4"), Date: &date, Status: &pending},
	)
	if err != nil {
		t.Fatal(err)
	}
	e.svc.Process(e.ctx, "on_transaction_create", plan)
	if got := e.reload(t, plan.ID); got.CategoryID != nil {
		t.Error("a pending transaction is a plan; rules wait until it is confirmed")
	}
	e.svc.Process(e.ctx, "on_transaction_create", nil) // must not panic
}

func TestProcessRunsRulesPriorityStopsAsked(t *testing.T) {
	e := newEnv(t)
	set := func(id int64) []map[string]any {
		return []map[string]any{{"type": "set_category", "category_id": float64(id)}}
	}
	// Created out of order: the lower number runs first, so Fun is set last.. unless stopped.
	late := e.rule(t, Input{Name: "late", Priority: 20, Actions: set(e.fun.ID)})
	first := e.rule(t, Input{Name: "first", Priority: 10, Actions: set(e.food.ID)})

	tx := e.expense(t, "4", "x")
	e.svc.Process(e.ctx, "on_transaction_create", tx)
	if got := e.reload(t, tx.ID); *got.CategoryID != e.fun.ID {
		t.Errorf("both ran in priority order, so the later one wins; category %d", *got.CategoryID)
	}

	if _, err := e.svc.Update(e.ctx, first.ID, Input{StopProcessing: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	tx2 := e.expense(t, "5", "y")
	e.svc.Process(e.ctx, "on_transaction_create", tx2)
	if got := e.reload(t, tx2.ID); *got.CategoryID != e.food.ID {
		t.Errorf("a stopping rule ends the run; category %d", *got.CategoryID)
	}

	ran := func(id int64) int { r, _ := e.svc.ByID(e.ctx, id); return r.RunsCount }
	if ran(first.ID) != 2 || ran(late.ID) != 1 {
		t.Errorf("run counts: first %d, late %d", ran(first.ID), ran(late.ID))
	}
	if r, _ := e.svc.ByID(e.ctx, first.ID); r.LastRunAt == nil {
		t.Error("last run time not set")
	}
}

func TestProcessLogsSuccessFailure(t *testing.T) {
	e := newEnv(t)
	good := e.rule(
		t,
		Input{
			Name:     "good",
			Priority: 1,
			Actions:  []map[string]any{{"type": "set_category", "category_id": float64(e.food.ID)}},
		},
	)
	// A category that does not exist: the write is refused.
	bad := e.rule(
		t,
		Input{Name: "bad", Priority: 2, Actions: []map[string]any{{"type": "set_category", "category_id": float64(9999)}}},
	)
	after := e.rule(
		t,
		Input{Name: "after", Priority: 3, Actions: []map[string]any{{"type": "set_description", "value": "done"}}},
	)

	tx := e.expense(t, "4", "x")
	e.svc.Process(e.ctx, "on_transaction_create", tx)

	logs := func(id int64) []Log {
		l, err := e.svc.Logs(e.ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return l
	}
	if l := logs(good.ID); len(l) != 1 || l[0].Status != "success" || l[0].TriggerEntityID == nil ||
		*l[0].TriggerEntityID != tx.ID {
		t.Errorf("good rule log %+v", l)
	}
	if l := logs(bad.ID); len(l) != 1 || l[0].Status != "error" || l[0].ErrorMessage == nil || *l[0].ErrorMessage == "" {
		t.Errorf("bad rule log %+v", l)
	}
	if r, _ := e.svc.ByID(e.ctx, bad.ID); r.RunsCount != 0 {
		t.Errorf("a failed run is not counted: %d", r.RunsCount)
	}
	if l := logs(after.ID); len(l) != 1 || l[0].Status != "success" {
		t.Errorf("a failing rule must not stop the ones after it: %+v", l)
	}
	if got := e.reload(t, tx.ID); *got.Description != "done" {
		t.Errorf("description %q", *got.Description)
	}
}

func TestActionsNeedMoreInputDoNothing(t *testing.T) {
	e := newEnv(t)
	e.rule(t, Input{Name: "vague", Actions: []map[string]any{
		{"type": "set_category"},
		{"type": "add_tags"},
		{"type": "remove_tags", "tag_ids": []any{}},
		{"type": "set_description", "value": ""},
		{"type": "create_transfer", "to_account_id": float64(e.bank.ID)},
		{"type": "unknown"},
	}})
	tx := e.expense(t, "4", "keep me")
	e.svc.Process(e.ctx, "on_transaction_create", tx)
	got := e.reload(t, tx.ID)
	if got.CategoryID != nil || len(got.Tags) != 0 || *got.Description != "keep me" {
		t.Errorf("transaction changed: %+v", got)
	}
}

func TestCreateTransferActionMovesFormulaAmount(t *testing.T) {
	e := newEnv(t)
	e.rule(t, Input{
		Name: "round up", Conditions: when("type", "equals", "expense"),
		Actions: []map[string]any{{
			"type": "create_transfer", "from_account_id": "{{transaction.account_id}}",
			"to_account_id": float64(e.bank.ID), "amount_formula": "{{transaction.amount}} * 0.1",
			"description": "Savings for {{transaction.description}}",
		}},
	})
	tx := e.expense(t, "33.35", "lunch")
	e.svc.Process(e.ctx, "on_transaction_create", tx)

	all, _, err := e.svc.Txs.Filtered(e.ctx, filter.TxFilter{Type: "transfer"}, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("%d transfers, want 1", len(all))
	}
	transfer := &all[0]
	if transfer.Amount.Decimal().String() != "3.34" { // 3.335 rounds to the cent, never to a float
		t.Errorf("transfer amount %s", transfer.Amount.Decimal())
	}
	if transfer.AccountID != e.wallet.ID || transfer.ToAccountID == nil || *transfer.ToAccountID != e.bank.ID {
		t.Errorf("transfer route %+v", transfer)
	}
}

func TestTestReportsMatchWithoutRunningAnything(t *testing.T) {
	e := newEnv(t)
	r := e.rule(
		t,
		Input{
			Name:       "r",
			Conditions: when("description", "contains", "coffee"),
			Actions:    []map[string]any{{"type": "set_category", "category_id": float64(e.food.ID)}},
		},
	)
	hit, miss := e.expense(t, "1", "coffee"), e.expense(t, "1", "tea")

	got, err := e.svc.Test(e.ctx, r.ID, hit.ID)
	if err != nil || got["conditionsMatch"] != true {
		t.Fatalf("hit: %v, %v", got, err)
	}
	got, _ = e.svc.Test(e.ctx, r.ID, miss.ID)
	if got["conditionsMatch"] != false {
		t.Errorf("miss: %v", got)
	}
	if e.reload(t, hit.ID).CategoryID != nil {
		t.Error("testing a rule must not run it")
	}
	if _, err := e.svc.Test(e.ctx, r.ID, 999); err == nil {
		t.Error("an unknown transaction is an error")
	}
	if got, err := e.svc.Test(e.ctx, 999, hit.ID); got != nil || err != nil {
		t.Errorf("unknown rule: %v, %v", got, err)
	}
}

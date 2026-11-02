//go:build integration

package httpserver

import (
	"net/http"
	"testing"
)

// A transaction goes in and comes out as payments and lines; the soft check
// reports a difference without refusing the save.
func TestTransactionPaymentsAndLinesOnTheWire(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("wire-tx@test.com", "secret1", roleEditor), false)
	acc, cat := seedMoney(t, a, sess)

	res := a.do("POST", "/api/transactions", flowIn("expense", acc, cat, "12.34", "2024-01-15"), sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	tx := body["data"].(map[string]any)
	if got := len(tx["payments"].([]any)); got != 1 {
		t.Fatalf("payments = %v", tx["payments"])
	}
	pay := tx["payments"].([]any)[0].(map[string]any)
	line := tx["lines"].([]any)[0].(map[string]any)
	if pay["amount"] != -12.34 || pay["accountId"] != float64(acc) || line["amount"] != 12.34 ||
		line["categoryId"] != float64(cat) || tx["amount"] != 12.34 || tx["difference"] != nil {
		t.Fatalf("payment %v line %v amount %v difference %v", pay, line, tx["amount"], tx["difference"])
	}
	if actions := tx["actions"].(map[string]any); actions["edit"] != true || actions["editDetails"] != true {
		t.Errorf("actions = %v", actions)
	}
	id := itoa(int64(tx["id"].(float64)))

	// Paying more than the lines say is reported, not refused.
	off := flowIn("expense", acc, cat, "10", "2024-01-15")
	off["payments"] = []map[string]any{{"account_id": acc, "amount": "-15"}}
	res = a.do("POST", "/api/transactions", off, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated || body["data"].(map[string]any)["difference"] != float64(5) {
		t.Errorf("a difference: %d %v", res.StatusCode, body)
	}

	// The flat shape of older versions is told apart from a mistake in the new one.
	old := map[string]any{"type": "expense", "account_id": acc, "amount": 5, "date": "2024-01-15"}
	status(t, a.do("POST", "/api/transactions", old, sess.Token, sess.CSRF), http.StatusUnprocessableEntity, "flat shape")

	// A PATCH keeps what it leaves out; an explicit null clears the description.
	status(t, a.do("PATCH", "/api/transactions/"+id, map[string]any{"description": "Lunch"}, sess.Token, sess.CSRF), 200, "describe")
	res = a.do("PATCH", "/api/transactions/"+id, map[string]any{"date": "2024-01-16"}, sess.Token, sess.CSRF)
	tx = decodeJSON(t, res)["data"].(map[string]any)
	if tx["description"] != "Lunch" || tx["date"] != "2024-01-16" || len(tx["payments"].([]any)) != 1 ||
		len(tx["lines"].([]any)) != 1 {
		t.Errorf("patch dropped something: %v", tx)
	}
	res = a.do("PATCH", "/api/transactions/"+id, map[string]any{"description": nil}, sess.Token, sess.CSRF)
	if got := decodeJSON(t, res)["data"].(map[string]any)["description"]; got != nil {
		t.Errorf("description = %v after null, want null", got)
	}
}

// A recurring rule keeps one pending occurrence: confirming or skipping it
// makes the next, and the open one can be neither edited nor deleted.
func TestRecurringRuleOccurrencesOnTheWire(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("wire-rec@test.com", "secret1", roleEditor), false)
	acc, cat := seedMoney(t, a, sess)

	template := flowIn("expense", acc, cat, "50", "")
	delete(template, "date")
	res := a.do("POST", "/api/recurring", map[string]any{
		"template": template,
		"schedule": map[string]any{"frequency": "monthly", "interval": 1, "day_of_month": 5, "start": "2030-01-05"},
	}, sess.Token, sess.CSRF)
	rule := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create rule %d %v", res.StatusCode, rule)
	}
	if got := rule["data"].(map[string]any)["nextRunDate"]; got != "2030-01-05" {
		t.Errorf("nextRunDate = %v", got)
	}

	pending := func() []map[string]any {
		var out []map[string]any
		data := decodeJSON(t, a.do("GET", "/api/transactions?status=pending", nil, sess.Token, ""))["data"].([]any)
		for _, row := range data {
			out = append(out, row.(map[string]any))
		}
		return out
	}
	open := pending()
	if len(open) != 1 || open[0]["date"] != "2030-01-05" {
		t.Fatalf("the rule's first occurrence: %v", open)
	}
	id := itoa(int64(open[0]["id"].(float64)))
	actions := open[0]["actions"].(map[string]any)
	if actions["confirm"] != true || actions["skip"] != true || actions["edit"] != false || actions["delete"] != false {
		t.Errorf("an open occurrence is confirmed or skipped: %v", actions)
	}
	if rs := open[0]["restrictions"].([]any); len(rs) != 1 || rs[0].(map[string]any)["reason"] != "recurring_pending" {
		t.Errorf("restrictions = %v", open[0]["restrictions"])
	}
	status(t, a.do("PATCH", "/api/transactions/"+id, map[string]any{"description": "x"}, sess.Token, sess.CSRF), 422, "edit")
	status(t, a.do("DELETE", "/api/transactions/"+id, nil, sess.Token, sess.CSRF), 422, "delete")

	status(t, a.do("POST", "/api/transactions/"+id+"/confirm", map[string]any{"date": "2024-01-15"}, sess.Token, sess.CSRF),
		200, "confirm")
	open = pending()
	if len(open) != 1 || open[0]["date"] != "2030-02-05" {
		t.Fatalf("confirming makes the next occurrence: %v", open)
	}
	id = itoa(int64(open[0]["id"].(float64)))
	status(t, a.do("POST", "/api/transactions/"+id+"/skip", nil, sess.Token, sess.CSRF), 200, "skip")
	open = pending()
	if len(open) != 1 || open[0]["date"] != "2030-03-05" {
		t.Fatalf("skipping makes the next occurrence: %v", open)
	}

	// A PATCH of the rule keeps what it leaves out.
	ruleID := itoa(int64(rule["data"].(map[string]any)["id"].(float64)))
	res = a.do("PATCH", "/api/recurring/"+ruleID, map[string]any{"is_active": false}, sess.Token, sess.CSRF)
	got := decodeJSON(t, res)["data"].(map[string]any)
	if res.StatusCode != 200 || got["isActive"] != false || got["template"] == nil ||
		got["schedule"].(map[string]any)["start"] != "2030-01-05" {
		t.Errorf("patched rule: %d %v", res.StatusCode, got)
	}
}

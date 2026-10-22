package httpserver

import (
	"testing"
	"time"
)

func TestAutomationSetsCategoryOnCreate(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("auto@test.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	res := a.do("POST", "/api/categories", map[string]any{"name": "Auto", "type": "expense"}, sess.Token, sess.CSRF)
	autoCat := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))

	res = a.do("POST", "/api/automation-rules", map[string]any{
		"name": "Tag groceries", "trigger_type": "on_transaction_create", "priority": 10,
		"conditions": map[string]any{
			"match": "all",
			"conditions": []map[string]any{
				{"field": "type", "op": "equals", "value": "expense"},
				{"field": "amount", "op": "gte", "value": 10},
			},
		},
		"actions": []map[string]any{
			{"type": "set_category", "category_id": autoCat},
		},
		"is_active": true,
	}, sess.Token, sess.CSRF)
	ruleBody := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("rule %d %v", res.StatusCode, ruleBody)
	}

	today := time.Now().UTC().Format("2006-01-02")
	res = a.do("POST", "/api/transactions", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 25, "date": today,
	}, sess.Token, sess.CSRF)
	tx := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("tx %d %v", res.StatusCode, tx)
	}
	cat := tx["data"].(map[string]any)["category"].(map[string]any)
	if int64(cat["id"].(float64)) != autoCat {
		t.Fatalf("category not rewritten %v", tx)
	}

	ruleID := int64(ruleBody["data"].(map[string]any)["id"].(float64))
	res = a.do("GET", "/api/automation-rules/"+itoa(ruleID)+"/logs", nil, sess.Token, "")
	logs := decodeJSON(t, res)
	if res.StatusCode != 200 || len(logs["data"].([]any)) != 1 {
		t.Fatalf("logs %d %v", res.StatusCode, logs)
	}

	res = a.do("POST", "/api/automation-rules/"+itoa(ruleID)+"/test", map[string]any{
		"transaction_id": int64(tx["data"].(map[string]any)["id"].(float64)),
	}, sess.Token, sess.CSRF)
	testBody := decodeJSON(t, res)
	if res.StatusCode != 200 || testBody["conditionsMatch"] != true {
		t.Fatalf("test %d %v", res.StatusCode, testBody)
	}
}

package httpserver

import (
	"testing"
	"time"

	"github.com/chiririll/savvy-plus/internal/auth"
)

func seedMoney(t *testing.T, a *testApp, sess *auth.Issued) (accID, catID int64) {
	t.Helper()
	res := a.do("POST", "/api/currencies", map[string]any{
		"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1,
	}, sess.Token, sess.CSRF)
	usdID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	res = a.do("POST", "/api/accounts", map[string]any{
		"name": "Cash", "type": "cash", "currency_id": usdID, "initial_balance": 1000,
	}, sess.Token, sess.CSRF)
	accID = int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	res = a.do("POST", "/api/categories", map[string]any{"name": "Food", "type": "expense"}, sess.Token, sess.CSRF)
	catID = int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	return accID, catID
}

func TestRecurringInactiveCreatesNoPending(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)

	res := a.do("POST", "/api/recurring", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 120,
		"frequency": "monthly", "interval": 1, "day_of_month": time.Now().UTC().Day(),
		"start_date": time.Now().UTC().Format("2006-01-02"), "is_active": false,
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	id := int64(body["data"].(map[string]any)["id"].(float64))
	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	list := decodeJSON(t, res)["data"].([]any)
	for _, item := range list {
		if rec, _ := item.(map[string]any)["recurringTransactionId"].(float64); int64(rec) == id {
			t.Fatal("inactive should not spawn pending")
		}
	}
}

func TestRecurringCreatesPendingAndConfirmSpawnsNext(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw2@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	today := time.Now().UTC().Format("2006-01-02")

	res := a.do("POST", "/api/recurring", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 80,
		"frequency": "daily", "interval": 1, "start_date": today, "is_active": true,
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	templateID := int64(body["data"].(map[string]any)["id"].(float64))

	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	if decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64) != 1000 {
		t.Fatal("pending must not change balance")
	}

	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	pending := findRecurringTx(decodeJSON(t, res)["data"].([]any), templateID)
	if pending == nil {
		t.Fatal("expected pending occurrence")
	}

	res = a.do("POST", "/api/transactions/"+itoa(int64(pending["id"].(float64)))+"/confirm", map[string]any{}, sess.Token, sess.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("confirm %d %v", res.StatusCode, decodeJSON(t, res))
	} else {
		res.Body.Close()
	}

	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	if decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64) != 920 {
		t.Fatal("confirm should deduct")
	}
	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	if findRecurringTx(decodeJSON(t, res)["data"].([]any), templateID) == nil {
		t.Fatal("next pending should spawn")
	}
}

func TestRecurringEndDateStopsNextPending(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw3@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	today := time.Now().UTC().Format("2006-01-02")

	res := a.do("POST", "/api/recurring", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 40,
		"frequency": "daily", "interval": 1, "start_date": today, "end_date": today, "is_active": true,
	}, sess.Token, sess.CSRF)
	templateID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	pending := findRecurringTx(decodeJSON(t, res)["data"].([]any), templateID)
	res = a.do("POST", "/api/transactions/"+itoa(int64(pending["id"].(float64)))+"/confirm", map[string]any{}, sess.Token, sess.CSRF)
	res.Body.Close()
	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	if findRecurringTx(decodeJSON(t, res)["data"].([]any), templateID) != nil {
		t.Fatal("should not spawn after end date")
	}
}

func TestRecurringSkipSpawnsNextWithoutBalanceChange(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw4@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	today := time.Now().UTC().Format("2006-01-02")

	res := a.do("POST", "/api/recurring", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 30,
		"frequency": "daily", "interval": 1, "start_date": today, "is_active": true,
	}, sess.Token, sess.CSRF)
	templateID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	pending := findRecurringTx(decodeJSON(t, res)["data"].([]any), templateID)
	res = a.do("POST", "/api/transactions/"+itoa(int64(pending["id"].(float64)))+"/skip", map[string]any{}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 200 || body["data"].(map[string]any)["status"] != "skipped" {
		t.Fatalf("skip %d %v", res.StatusCode, body)
	}
	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	if decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64) != 1000 {
		t.Fatal("skip must not change balance")
	}
	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	if findRecurringTx(decodeJSON(t, res)["data"].([]any), templateID) == nil {
		t.Fatal("skip should spawn next pending")
	}
}

func TestBudgetProgressCountsConfirmedExpenses(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("b@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	today := time.Now().UTC().Format("2006-01-02")

	res := a.do("POST", "/api/transactions", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 40, "date": today,
	}, sess.Token, sess.CSRF)
	res.Body.Close()

	res = a.do("POST", "/api/budgets", map[string]any{
		"name": "Food", "amount": 100, "period": "monthly", "is_global": false,
		"category_ids": []int64{catID}, "is_active": true,
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("budget %d %v", res.StatusCode, body)
	}
	prog := body["data"].(map[string]any)["progress"].(map[string]any)
	if prog["spent"].(float64) != 40 || prog["remaining"].(float64) != 60 {
		t.Fatalf("progress %v", prog)
	}
}

func TestAutomationSetsCategoryOnCreate(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("auto@test.com", "secret1", auth.RoleReadWrite)
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
	if res.StatusCode != 200 || testBody["conditions_match"] != true {
		t.Fatalf("test %d %v", res.StatusCode, testBody)
	}
}

func findRecurringTx(list []any, templateID int64) map[string]any {
	for _, item := range list {
		m := item.(map[string]any)
		if rec, ok := m["recurringTransactionId"].(float64); ok && int64(rec) == templateID {
			return m
		}
	}
	return nil
}

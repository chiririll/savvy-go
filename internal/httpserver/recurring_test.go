package httpserver

import (
	"testing"
	"time"
)

func TestRecurringInactiveCreatesNoPending(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@test.com", "secret1", roleEditor)
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
	u := a.createUser("rw2@test.com", "secret1", roleEditor)
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

func TestEstimatedAmountCarriesToPendingAndConfirmReplacesIt(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("est@test.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	today := time.Now().UTC().Format("2006-01-02")

	res := a.do("POST", "/api/recurring", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 80,
		"frequency": "monthly", "interval": 1, "start_date": today, "is_estimated": true,
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 || body["data"].(map[string]any)["isEstimated"] != true {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	templateID := int64(body["data"].(map[string]any)["id"].(float64))

	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	pending := findRecurringTx(decodeJSON(t, res)["data"].([]any), templateID)
	if pending == nil || pending["isEstimated"] != true {
		t.Fatalf("pending occurrence should be estimated: %v", pending)
	}

	res = a.do("POST", "/api/transactions/"+itoa(int64(pending["id"].(float64)))+"/confirm",
		map[string]any{"amount": 95.5}, sess.Token, sess.CSRF)
	out := decodeJSON(t, res)["data"].(map[string]any)
	if res.StatusCode != 200 || out["isEstimated"] != false || out["amount"].(float64) != 95.5 {
		t.Fatalf("confirm should store the real amount: %d %v", res.StatusCode, out)
	}
	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	if bal := decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64); bal != 904.5 {
		t.Fatalf("balance %v, want 904.5", bal)
	}

	// The template keeps its estimate, so the next occurrence is estimated too.
	res = a.do("GET", "/api/transactions?status=pending", nil, sess.Token, "")
	next := findRecurringTx(decodeJSON(t, res)["data"].([]any), templateID)
	if next == nil || next["isEstimated"] != true || next["amount"].(float64) != 80 {
		t.Fatalf("next occurrence: %v", next)
	}
}

func TestRecurringEndDateStopsNextPending(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw3@test.com", "secret1", roleEditor)
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
	u := a.createUser("rw4@test.com", "secret1", roleEditor)
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

func findRecurringTx(list []any, templateID int64) map[string]any {
	for _, item := range list {
		m := item.(map[string]any)
		if rec, ok := m["recurringTransactionId"].(float64); ok && int64(rec) == templateID {
			return m
		}
	}
	return nil
}

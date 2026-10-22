package httpserver

import (
	"testing"
	"time"
)

func TestBudgetProgressCountsConfirmedExpenses(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("b@test.com", "secret1", roleEditor)
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

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// PATCH /transactions/bulk changes only the fields sent, answers in the order
// sent, and a refused item leaves every transaction as it was.
func TestBulkUpdateIsAllOrNone(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	usd := apiID(t, a, sess, "/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1})
	acc := apiID(t, a, sess, "/api/accounts",
		map[string]any{"name": "Cash", "type": "cash", "currency_id": usd, "initial_balance": 0})
	first := apiID(t, a, sess, "/api/transactions", flowBody("expense", acc, 12, "2024-01-15"))
	second := apiID(t, a, sess, "/api/transactions", flowBody("expense", acc, 3, "2024-01-16"))

	patch := func(items ...map[string]any) (int, map[string]any) {
		res := a.do("PATCH", "/api/transactions/bulk", map[string]any{"transactions": items}, sess.Token, sess.CSRF)
		return res.StatusCode, decodeJSON(t, res)
	}

	code, out := patch(map[string]any{"id": first, "description": "lunch"}, map[string]any{"id": second, "date": "2024-01-20"})
	data, _ := out["data"].([]any)
	if code != http.StatusOK || len(data) != 2 {
		t.Fatalf("update = %d %v", code, out)
	}
	one, two := data[0].(map[string]any), data[1].(map[string]any)
	if one["id"] != float64(first) || one["description"] != "lunch" || two["id"] != float64(second) || two["date"] == "2024-01-16" {
		t.Fatalf("answer = %v", data)
	}

	if code, out = patch(map[string]any{"id": first, "description": "dinner"}, map[string]any{"id": 999999}); code != http.StatusNotFound {
		t.Fatalf("unknown id = %d %v, want 404", code, out)
	}
	if code, out = patch(map[string]any{"id": first}, map[string]any{"id": first}); code != http.StatusUnprocessableEntity {
		t.Fatalf("repeated id = %d %v, want 422", code, out)
	}
	if code, out = patch(map[string]any{"description": "no id"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("missing id = %d %v, want 422", code, out)
	}
	if code, out = patch(map[string]any{"id": first, "amount": 5}); code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown field = %d %v, want 422", code, out)
	}
	if code, out = patch(); code != http.StatusUnprocessableEntity {
		t.Fatalf("empty list = %d %v, want 422", code, out)
	}

	res := a.do("GET", "/api/transactions/"+itoa(first), nil, sess.Token, "")
	if got := decodeJSON(t, res)["data"].(map[string]any); got["description"] != "lunch" {
		t.Fatalf("a refused call changed the transaction: %v", got["description"])
	}
}

//go:build integration

package integration

import (
	"net/http"
	"strconv"
	"testing"
)

// A retry with the same external key is answered 409 naming the first
// transaction, so nothing is added twice; the key finds the transaction again.
func TestExternalIDMakesCreateIdempotent(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	usd := apiID(t, a, sess, "/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1})
	acc := apiID(t, a, sess, "/api/accounts",
		map[string]any{"name": "Cash", "type": "cash", "currency_id": usd, "initial_balance": 0})

	body := flowBody("expense", acc, 12, "2024-01-15")
	body["external_id"] = "F6HDVMNQ-5105"
	res := a.do("POST", "/api/transactions", body, sess.Token, sess.CSRF)
	out := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("first create = %d %v", res.StatusCode, out)
	}
	data := out["data"].(map[string]any)
	if data["externalSource"] != "ui" || data["externalId"] != "F6HDVMNQ-5105" {
		t.Fatalf("answer lacks the key: %v", data)
	}
	first := data["id"]

	res = a.do("POST", "/api/transactions", body, sess.Token, sess.CSRF)
	out = decodeJSON(t, res)
	if res.StatusCode != http.StatusConflict || out["transaction_id"] != first || out["external_id"] != "F6HDVMNQ-5105" {
		t.Fatalf("retry = %d %v, want 409 naming transaction %v", res.StatusCode, out, first)
	}

	bulk := map[string]any{"transactions": []any{flowBody("expense", acc, 1, "2024-01-16"), body}}
	res = a.do("POST", "/api/transactions/bulk", bulk, sess.Token, sess.CSRF)
	out = decodeJSON(t, res)
	if res.StatusCode != http.StatusConflict || out["transaction_id"] != first {
		t.Fatalf("bulk retry = %d %v", res.StatusCode, out)
	}

	res = a.do("GET", "/api/transactions?external_id=F6HDVMNQ-5105&external_source=ui", nil, sess.Token, "")
	list := decodeJSON(t, res)["data"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["id"] != first {
		t.Fatalf("lookup by key = %v", list)
	}
	res = a.do("GET", "/api/transactions?external_id=F6HDVMNQ-5105&external_source=api", nil, sess.Token, "")
	if list = decodeJSON(t, res)["data"].([]any); len(list) != 0 {
		t.Fatalf("another source found %v", list)
	}
	res = a.do("GET", "/api/transactions", nil, sess.Token, "")
	if list = decodeJSON(t, res)["data"].([]any); len(list) != 1 {
		t.Fatalf("%d transactions stored, want 1: the refused calls must add none", len(list))
	}

	// Sent on update, the key is refused, not ignored.
	put := map[string]any{"external_id": "other"}
	path := "/api/transactions/" + strconv.FormatInt(int64(first.(float64)), 10)
	status(t, a.do("PATCH", path, put, sess.Token, sess.CSRF), http.StatusUnprocessableEntity, "key on update")
}

// Find-similar tells, for each item, which entered transactions look like it,
// by money and day or by the external id, without storing anything.
func TestFindSimilarTransactions(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	usd := apiID(t, a, sess, "/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1})
	acc := apiID(t, a, sess, "/api/accounts",
		map[string]any{"name": "Cash", "type": "cash", "currency_id": usd, "initial_balance": 0})
	body := flowBody("expense", acc, 12, "2024-01-15")
	body["external_id"] = "R-1"
	first := apiID(t, a, sess, "/api/transactions", body)

	req := map[string]any{"items": []any{
		map[string]any{"key": "bank-1", "date": "2024-01-16", "amount": "12.00", "currency": "USD"},
		map[string]any{"key": "receipt", "external_id": "R-1"},
		map[string]any{"key": "far", "date": "2024-03-01", "amount": "12", "currency": "USD"},
	}}
	res := a.do("POST", "/api/transactions/find-similar", req, sess.Token, sess.CSRF)
	out := decodeJSON(t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("find-similar = %d %v", res.StatusCode, out)
	}
	items := out["data"].([]any)
	matches := func(i int) []any { return items[i].(map[string]any)["matches"].([]any) }
	if len(items) != 3 || items[0].(map[string]any)["key"] != "bank-1" {
		t.Fatalf("answer = %v", items)
	}
	m := matches(0)
	if len(m) != 1 || m[0].(map[string]any)["transaction"].(map[string]any)["id"] != float64(first) {
		t.Fatalf("by money = %v", m)
	}
	if len(matches(1)) != 1 || len(matches(2)) != 0 {
		t.Fatalf("by key %v, far %v", matches(1), matches(2))
	}
	// An unknown currency refuses the whole request and names the item.
	bad := map[string]any{"items": []any{map[string]any{"date": "2024-01-16", "amount": "12", "currency": "EUR"}}}
	res = a.do("POST", "/api/transactions/find-similar", bad, sess.Token, sess.CSRF)
	status(t, res, http.StatusUnprocessableEntity, "unknown currency EUR")
}

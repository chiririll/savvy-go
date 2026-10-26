//go:build integration

package httpserver

import (
	"net/http"
	"testing"

	"savvy-go/internal/auth"
)

// apiID creates something through the API and returns the id of the "data" it answers with.
func apiID(t *testing.T, a *testApp, sess *auth.Issued, path string, body map[string]any) int64 {
	t.Helper()
	res := a.do("POST", path, body, sess.Token, sess.CSRF)
	out := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST %s = %d %v", path, res.StatusCode, out)
	}
	return int64(out["data"].(map[string]any)["id"].(float64))
}

// A value the schema forbids is a client error, not a stored row or a server error.
func TestInvalidValuesAreClientErrors(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	usd := apiID(
		t,
		a,
		sess,
		"/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1},
	)
	acc := apiID(
		t,
		a,
		sess,
		"/api/accounts",
		map[string]any{"name": "Cash", "type": "cash", "currency_id": usd, "initial_balance": 0},
	)

	for _, c := range []struct {
		name, path string
		body       map[string]any
	}{
		{"unknown account type", "/api/accounts",
			map[string]any{"name": "A", "type": "wallet", "currency_id": usd, "initial_balance": 0}},
		{"negative transaction amount", "/api/transactions",
			map[string]any{"type": "income", "account_id": acc, "amount": -5, "date": "2024-01-15"}},
		{"unknown transaction type", "/api/transactions",
			map[string]any{"type": "gift", "account_id": acc, "amount": 5, "date": "2024-01-15"}},
	} {
		status(t, a.do("POST", c.path, c.body, sess.Token, sess.CSRF), http.StatusUnprocessableEntity, c.name)
	}
	list := decodeJSON(t, a.do("GET", "/api/transactions", nil, sess.Token, ""))["data"].([]any)
	if len(list) != 0 {
		t.Fatalf("refused requests stored %d transactions", len(list))
	}
}

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

// Reordering sets the order of a day's transactions in the list.
func TestReorderTransactionsWithinADay(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("reorder@test.com", "secret1", roleEditor), false)
	accID, _ := seedMoney(t, a, sess)
	var ids []int64
	for _, amount := range []int{1, 2, 3} {
		ids = append(ids, apiID(t, a, sess, "/api/transactions", map[string]any{
			"type": "expense", "account_id": accID, "amount": amount, "date": "2024-01-15",
		}))
	}
	listed := func() []int64 {
		data := decodeJSON(t, a.do("GET", "/api/transactions?sort_by=date&sort_direction=asc", nil, sess.Token, ""))["data"].([]any)
		var out []int64
		for _, row := range data {
			out = append(out, int64(row.(map[string]any)["id"].(float64)))
		}
		return out
	}

	want := []int64{ids[2], ids[0], ids[1]}
	status(t, a.do("POST", "/api/transactions/reorder",
		map[string]any{"date": "2024-01-15", "ids": want}, sess.Token, sess.CSRF), http.StatusOK, "reorder")
	got := listed()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}

	status(t, a.do("POST", "/api/transactions/reorder",
		map[string]any{"date": "2024-02-01", "ids": want}, sess.Token, sess.CSRF),
		http.StatusUnprocessableEntity, "ids of another day")
	status(t, a.do("POST", "/api/transactions/reorder",
		map[string]any{"date": "2024-01-15"}, sess.Token, sess.CSRF),
		http.StatusUnprocessableEntity, "no ids")
}

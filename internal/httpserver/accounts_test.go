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

// A period that cannot be read, or ends before it starts, is refused: a silent
// fallback to the default period would show data the client never asked for.
func TestInvalidPeriodIsAClientError(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)

	for _, c := range []struct{ name, path string }{
		{"summary, end before start", "/api/categories-summary?type=expense&start_date=2025-02-01&end_date=2025-01-01"},
		{"summary, unreadable date", "/api/categories-summary?type=expense&start_date=nope&end_date=2025-01-01"},
		{"summary, one date only", "/api/categories-summary?type=expense&start_date=2025-01-01"},
		{"balance history", "/api/accounts-balance-history?start_date=2025-02-01&end_date=2025-01-01"},
		{"balance comparison", "/api/accounts-balance-comparison?start_date=2025-02-01&end_date=2025-01-01"},
	} {
		status(t, a.do("GET", c.path, nil, sess.Token, ""), http.StatusUnprocessableEntity, c.name)
	}
	status(t, a.do("GET", "/api/categories-summary?type=expense&start_date=2025-01-01&end_date=2025-01-31", nil, sess.Token, ""), http.StatusOK, "valid range")
	status(t, a.do("GET", "/api/categories-summary?type=expense", nil, sess.Token, ""), http.StatusOK, "no range")
}

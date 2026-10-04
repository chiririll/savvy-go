package httpserver

import (
	"testing"

	"savvy-go/internal/auth"
)

// TestSchemaConstraintsRejectBadInput verifies the schema CHECK constraints
// turn invalid values into a client error instead of storing them.
func TestSchemaConstraintsRejectBadInput(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)

	res := a.do("POST", "/api/currencies", map[string]any{
		"code": "USD", "name": "USD", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1,
	}, sess.Token, sess.CSRF)
	usd := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))

	cases := []struct {
		name, path string
		body       map[string]any
	}{
		{"unknown account type", "/api/accounts", map[string]any{"name": "A", "type": "wallet", "currency_id": usd, "initial_balance": 0}},
		{"zero transaction amount", "/api/transactions", map[string]any{"type": "income", "account_id": 1, "amount": 0, "date": "2024-01-15"}},
		{"unknown transaction type", "/api/transactions", map[string]any{"type": "gift", "account_id": 1, "amount": 5, "date": "2024-01-15"}},
	}
	// A valid account for the transaction cases.
	ok := a.do("POST", "/api/accounts", map[string]any{"name": "Cash", "type": "cash", "currency_id": usd, "initial_balance": 0}, sess.Token, sess.CSRF)
	if ok.StatusCode != 201 {
		t.Fatalf("create account %d", ok.StatusCode)
	}
	for _, c := range cases {
		res := a.do("POST", c.path, c.body, sess.Token, sess.CSRF)
		if res.StatusCode < 400 || res.StatusCode >= 500 {
			t.Errorf("%s: status %d, want 4xx (%v)", c.name, res.StatusCode, decodeJSON(t, res))
		}
	}
}

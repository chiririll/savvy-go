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

// An account may name its currency by code instead of id: the currency is then added to the
// space from the catalog, with its rate to the base currency.
func TestAccountCreatedFromACatalogCurrency(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	apiID(
		t,
		a,
		sess,
		"/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1},
	)

	res := a.do("POST", "/api/accounts", map[string]any{
		"name": "Barclays", "type": "bank", "currency_code": "GBP", "initial_balance": 100,
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	cur := body["data"].(map[string]any)["currency"].(map[string]any)
	if cur["code"] != "GBP" || cur["decimals"] != float64(2) || cur["isBase"] != false || cur["rate"].(float64) <= 0 {
		t.Fatalf("currency added from the catalog: %v", cur)
	}
	list := decodeJSON(t, a.do("GET", "/api/currencies", nil, sess.Token, ""))["data"].([]any)
	if len(list) != 2 {
		t.Fatalf("the space has %d currencies, want USD and GBP", len(list))
	}
}

// The summary adds the balances of accounts in different currencies in the base currency.
func TestAccountSummaryConvertsToTheBaseCurrency(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	usd := apiID(
		t,
		a,
		sess,
		"/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1},
	)
	eur := apiID(
		t,
		a,
		sess,
		"/api/currencies",
		map[string]any{"code": "EUR", "name": "Euro", "symbol": "€", "decimals": 2, "rate": 2},
	)
	apiID(
		t,
		a,
		sess,
		"/api/accounts",
		map[string]any{"name": "Cash", "type": "cash", "currency_id": usd, "initial_balance": 50},
	)
	apiID(
		t,
		a,
		sess,
		"/api/accounts",
		map[string]any{"name": "Euro cash", "type": "cash", "currency_id": eur, "initial_balance": 10},
	)

	body := decodeJSON(t, a.do("GET", "/api/accounts?with_summary=1", nil, sess.Token, ""))
	summary, _ := body["summary"].(map[string]any)
	// A currency's rate is its value in the base currency: 10 EUR at 2 is 20 USD.
	if summary["currency"] != "USD" || summary["totalBalance"] != float64(70) || summary["accountsCount"] != float64(2) {
		t.Fatalf("summary %v", summary)
	}
}

func TestDeletedAccountIsGone(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	usd := apiID(
		t,
		a,
		sess,
		"/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1},
	)
	id := itoa(
		apiID(
			t,
			a,
			sess,
			"/api/accounts",
			map[string]any{"name": "Cash", "type": "cash", "currency_id": usd, "initial_balance": 0},
		),
	)

	status(t, a.do("DELETE", "/api/accounts/"+id, nil, sess.Token, sess.CSRF), http.StatusNoContent, "delete")
	status(t, a.do("GET", "/api/accounts/"+id, nil, sess.Token, ""), http.StatusNotFound, "get after delete")
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
		{"unknown account type", "/api/accounts", map[string]any{"name": "A", "type": "wallet", "currency_id": usd, "initial_balance": 0}},
		{"negative transaction amount", "/api/transactions", map[string]any{"type": "income", "account_id": acc, "amount": -5, "date": "2024-01-15"}},
		{"unknown transaction type", "/api/transactions", map[string]any{"type": "gift", "account_id": acc, "amount": 5, "date": "2024-01-15"}},
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

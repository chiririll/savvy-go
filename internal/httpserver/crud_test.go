package httpserver

import (
	"net/http"
	"testing"

	"github.com/chiririll/savvy-plus/internal/auth"
)

func TestCurrencyAccountCategoryTagCRUD(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)

	res := a.do("POST", "/api/currencies", map[string]any{
		"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1,
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("currency create %d %v", res.StatusCode, body)
	}
	usdID := int64(body["data"].(map[string]any)["id"].(float64))

	res = a.do("POST", "/api/currencies", map[string]any{
		"code": "EUR", "name": "Euro", "symbol": "€", "decimals": 2, "rate": 0.9,
	}, sess.Token, sess.CSRF)
	if res.StatusCode != 201 {
		t.Fatalf("eur %d %v", res.StatusCode, decodeJSON(t, res))
	} else {
		res.Body.Close()
	}

	res = a.do("GET", "/api/currencies", nil, sess.Token, "")
	list := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("list %d", res.StatusCode)
	}
	if len(list["data"].([]any)) != 2 {
		t.Fatalf("currencies %v", list)
	}

	res = a.do("POST", "/api/currencies/convert", map[string]any{
		"amount": 10, "from_currency_id": usdID, "to_currency_id": usdID,
	}, sess.Token, sess.CSRF)
	conv := decodeJSON(t, res)
	if conv["result"].(float64) != 10 {
		t.Fatalf("convert %v", conv)
	}

	res = a.do("POST", "/api/accounts", map[string]any{
		"name": "Cash", "type": "cash", "currency_id": usdID, "initial_balance": 50,
	}, sess.Token, sess.CSRF)
	acc := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("account %d %v", res.StatusCode, acc)
	}
	if acc["data"].(map[string]any)["currentBalance"].(float64) != 50 {
		t.Fatalf("balance %v", acc)
	}
	accID := int64(acc["data"].(map[string]any)["id"].(float64))

	res = a.do("POST", "/api/accounts", map[string]any{
		"name": "Barclays", "type": "bank", "currency_code": "GBP", "initial_balance": 100,
	}, sess.Token, sess.CSRF)
	if res.StatusCode != 201 {
		t.Fatalf("catalog account %d %v", res.StatusCode, decodeJSON(t, res))
	} else {
		res.Body.Close()
	}

	res = a.do("GET", "/api/accounts?with_summary=1&exclude_debts=1", nil, sess.Token, "")
	listed := decodeJSON(t, res)
	if listed["summary"] == nil {
		t.Fatalf("summary %v", listed)
	}

	res = a.do("POST", "/api/categories", map[string]any{"name": "Food", "type": "expense"}, sess.Token, sess.CSRF)
	cat := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("category %d %v", res.StatusCode, cat)
	}
	catID := int64(cat["data"].(map[string]any)["id"].(float64))

	res = a.do("GET", "/api/categories/"+itoa(catID)+"/statistics", nil, sess.Token, "")
	if res.StatusCode != 200 {
		t.Fatalf("stats %d", res.StatusCode)
	}
	res.Body.Close()

	res = a.do("POST", "/api/tags", map[string]any{"name": "groceries"}, sess.Token, sess.CSRF)
	if res.StatusCode != 201 {
		t.Fatalf("tag %d %v", res.StatusCode, decodeJSON(t, res))
	} else {
		res.Body.Close()
	}

	ro := a.createUser("ro@test.com", "secret1", auth.RoleReadOnly)
	ros := a.issue(ro, false)
	res = a.do("POST", "/api/tags", map[string]any{"name": "blocked"}, ros.Token, ros.CSRF)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("readonly write %d", res.StatusCode)
	}
	res.Body.Close()

	res = a.do("DELETE", "/api/accounts/"+itoa(accID), nil, sess.Token, sess.CSRF)
	if res.StatusCode != 204 {
		t.Fatalf("delete account %d", res.StatusCode)
	}
}

func TestCurrencyAuthz(t *testing.T) {
	a := newTestApp(t)
	res := a.do("GET", "/api/currencies", nil, "", "")
	if res.StatusCode != 401 {
		t.Fatalf("unauth %d", res.StatusCode)
	}
	res.Body.Close()
}

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

func TestCreditCardThroughTheAPI(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("card@test.com", "secret1", roleEditor), false)
	usd := apiID(t, a, sess, "/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1})

	// A card needs its terms; a card cannot be made as an account.
	status(t, a.do("POST", "/api/credit-cards",
		map[string]any{"name": "Gold", "currency_id": usd}, sess.Token, sess.CSRF),
		http.StatusUnprocessableEntity, "a card without terms")
	status(t, a.do("POST", "/api/credit-cards", map[string]any{
		"name": "Gold", "currency_id": usd, "limit": 1000, "statement_day": 40, "grace_days": 25,
	}, sess.Token, sess.CSRF), http.StatusUnprocessableEntity, "a statement day of 40")
	status(t, a.do("POST", "/api/accounts",
		map[string]any{"name": "Gold", "type": "credit", "currency_id": usd}, sess.Token, sess.CSRF),
		http.StatusUnprocessableEntity, "a card as an account")

	card := apiID(t, a, sess, "/api/credit-cards", map[string]any{
		"name": "Gold", "currency_id": usd, "limit": 1000, "statement_day": 20, "grace_days": 25,
	})

	cc := decodeJSON(t, a.do("GET", "/api/credit-cards/"+itoa(card), nil, sess.Token, ""))["data"].(map[string]any)
	if cc["limit"].(float64) != 1000 || cc["statementDay"].(float64) != 20 || cc["graceDays"].(float64) != 25 ||
		cc["cycle"] != "fixed_day" || cc["type"] != "credit" {
		t.Errorf("terms %v", cc)
	}
	st := cc["status"].(map[string]any)
	if st["level"] != "ok" || cc["available"].(float64) != 1000 {
		t.Errorf("a new card: status %v available %v", st, cc["available"])
	}

	// An update that sends only one term keeps the others.
	res := a.do("PATCH", "/api/credit-cards/"+itoa(card), map[string]any{"grace_days": 30}, sess.Token, sess.CSRF)
	cc = decodeJSON(t, res)["data"].(map[string]any)
	if cc["graceDays"].(float64) != 30 || cc["statementDay"].(float64) != 20 || cc["limit"].(float64) != 1000 {
		t.Errorf("after the update: %v", cc)
	}
	status(t, a.do("PATCH", "/api/accounts/"+itoa(card), map[string]any{"name": "x"}, sess.Token, sess.CSRF),
		http.StatusUnprocessableEntity, "a card changed as an account")

	// A bank account becomes a card and keeps its id.
	bank := apiID(t, a, sess, "/api/accounts",
		map[string]any{"name": "Old card", "type": "bank", "currency_id": usd})
	res = a.do("POST", "/api/credit-cards",
		map[string]any{"account_id": bank, "limit": 500, "statement_day": 5, "grace_days": 20}, sess.Token, sess.CSRF)
	if got := decodeJSON(t, res)["data"].(map[string]any); int64(got["id"].(float64)) != bank || got["type"] != "credit" {
		t.Errorf("converted: %v", got)
	}

	// Both are cards, and an account of the list is not one unless it is.
	list := decodeJSON(t, a.do("GET", "/api/credit-cards", nil, sess.Token, ""))["data"].([]any)
	if len(list) != 2 {
		t.Errorf("cards %v", list)
	}

	// Cards are in the account list too, with their terms and state; the type
	// query narrows the list.
	all := decodeJSON(t, a.do("GET", "/api/accounts", nil, sess.Token, ""))["data"].([]any)
	if len(all) != 2 {
		t.Fatalf("accounts %v", all)
	}
	cards := decodeJSON(t, a.do("GET", "/api/accounts?type=credit", nil, sess.Token, ""))["data"].([]any)
	if len(cards) != 2 {
		t.Fatalf("credit accounts %v", cards)
	}
	info, ok := cards[0].(map[string]any)["creditCard"].(map[string]any)
	if !ok || info["limit"].(float64) == 0 || info["status"].(map[string]any)["level"] != "ok" {
		t.Errorf("card in the account list: %v", cards[0])
	}
	plain := apiID(t, a, sess, "/api/accounts", map[string]any{"name": "Wallet", "type": "cash", "currency_id": usd})
	for _, q := range []struct {
		query string
		want  int
	}{{"type=cash", 1}, {"type=credit,cash", 3}, {"type=bank", 0}} {
		got := decodeJSON(t, a.do("GET", "/api/accounts?"+q.query, nil, sess.Token, ""))["data"].([]any)
		if len(got) != q.want {
			t.Errorf("?%s: %d accounts, want %d", q.query, len(got), q.want)
		}
	}
	for _, row := range decodeJSON(t, a.do("GET", "/api/accounts?type=cash", nil, sess.Token, ""))["data"].([]any) {
		if _, has := row.(map[string]any)["creditCard"]; has || int64(row.(map[string]any)["id"].(float64)) != plain {
			t.Errorf("an ordinary account has card terms: %v", row)
		}
	}

	status(t, a.do("DELETE", "/api/credit-cards/"+itoa(card), nil, sess.Token, sess.CSRF), http.StatusNoContent, "delete")
}

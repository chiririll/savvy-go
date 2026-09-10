package httpserver

import (
	"testing"
	"time"

	"savvy-go/internal/auth"
)

func TestPendingAndConfirmedTransactions(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)

	res := a.do("POST", "/api/currencies", map[string]any{
		"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1,
	}, sess.Token, sess.CSRF)
	usdID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))

	res = a.do("POST", "/api/accounts", map[string]any{
		"name": "Cash", "type": "cash", "currency_id": usdID, "initial_balance": 1000,
	}, sess.Token, sess.CSRF)
	accID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))

	res = a.do("POST", "/api/categories", map[string]any{"name": "Food", "type": "expense"}, sess.Token, sess.CSRF)
	catID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))

	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	res = a.do("POST", "/api/transactions", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 50, "date": tomorrow,
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 || body["data"].(map[string]any)["status"] != "pending" {
		t.Fatalf("pending %d %v", res.StatusCode, body)
	}
	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	if decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64) != 1000 {
		t.Fatal("pending should not change balance")
	}

	today := time.Now().UTC().Format("2006-01-02")
	res = a.do("POST", "/api/transactions", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 50, "date": today,
	}, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != 201 || body["data"].(map[string]any)["status"] != "confirmed" {
		t.Fatalf("confirmed %d %v", res.StatusCode, body)
	}
	txID := int64(body["data"].(map[string]any)["id"].(float64))
	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	if decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64) != 950 {
		t.Fatal("confirmed should deduct")
	}

	res = a.do("POST", "/api/transactions/"+itoa(txID)+"/duplicate", map[string]any{}, sess.Token, sess.CSRF)
	if res.StatusCode != 201 {
		t.Fatalf("dup %d %v", res.StatusCode, decodeJSON(t, res))
	} else {
		res.Body.Close()
	}
}

func TestDebtCreateLendAndBorrow(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("d@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)

	res := a.do("POST", "/api/currencies", map[string]any{
		"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1,
	}, sess.Token, sess.CSRF)
	usdID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	res = a.do("POST", "/api/accounts", map[string]any{
		"name": "Cash", "type": "cash", "currency_id": usdID, "initial_balance": 1000,
	}, sess.Token, sess.CSRF)
	accID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))

	res = a.do("POST", "/api/debts", map[string]any{
		"origin": "new", "name": "Loan to Ivan", "debt_type": "owed_to_me",
		"account_id": accID, "amount": 200, "date": "2026-09-01",
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("lend %d %v", res.StatusCode, body)
	}
	if body["data"].(map[string]any)["remainingDebt"].(float64) != 200 {
		t.Fatalf("remaining %v", body)
	}
	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	if decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64) != 800 {
		t.Fatal("lend should debit cash")
	}

	res = a.do("POST", "/api/debts", map[string]any{
		"origin": "new", "name": "Borrowed", "debt_type": "i_owe",
		"account_id": accID, "amount": 300, "date": "2026-09-01",
	}, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("borrow %d %v", res.StatusCode, body)
	}
	if body["data"].(map[string]any)["remainingDebt"].(float64) != 300 {
		t.Fatalf("borrow remaining %v", body)
	}
	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	bal := decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64)
	if bal != 1100 {
		t.Fatalf("after borrow cash want 1100 got %v", bal)
	}
}

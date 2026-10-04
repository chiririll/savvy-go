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
	if body["data"].(map[string]any)["currentBalance"].(float64) != 200 {
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
	if body["data"].(map[string]any)["currentBalance"].(float64) != 300 {
		t.Fatalf("borrow remaining %v", body)
	}
	res = a.do("GET", "/api/accounts/"+itoa(accID), nil, sess.Token, "")
	bal := decodeJSON(t, res)["data"].(map[string]any)["currentBalance"].(float64)
	if bal != 1100 {
		t.Fatalf("after borrow cash want 1100 got %v", bal)
	}
}

func listTxIDs(t *testing.T, a *testApp, sess *auth.Issued, query string) (ids []int64, total float64) {
	t.Helper()
	res := a.do("GET", "/api/transactions"+query, nil, sess.Token, "")
	body := decodeJSON(t, res)
	for _, item := range body["data"].([]any) {
		ids = append(ids, int64(item.(map[string]any)["id"].(float64)))
	}
	return ids, body["meta"].(map[string]any)["total"].(float64)
}

func TestTransactionsListSortAndFilters(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("sort@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	mk := func(path string, payload map[string]any) int64 {
		res := a.do("POST", path, payload, sess.Token, sess.CSRF)
		body := decodeJSON(t, res)
		if res.StatusCode != 201 {
			t.Fatalf("%s %d %v", path, res.StatusCode, body)
		}
		return int64(body["data"].(map[string]any)["id"].(float64))
	}
	usd := mk("/api/currencies", map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1})
	eur := mk("/api/currencies", map[string]any{"code": "EUR", "name": "Euro", "symbol": "E", "decimals": 2, "is_base": false, "rate": 2})
	usdAcc := mk("/api/accounts", map[string]any{"name": "USD", "type": "cash", "currency_id": usd, "initial_balance": 0})
	eurAcc := mk("/api/accounts", map[string]any{"name": "EUR", "type": "cash", "currency_id": eur, "initial_balance": 0})
	food := mk("/api/categories", map[string]any{"name": "Food", "type": "expense"})
	fun := mk("/api/categories", map[string]any{"name": "Fun", "type": "expense"})
	tag := mk("/api/tags", map[string]any{"name": "trip"})

	// raw 100 USD = 100 base; raw 80 EUR = 160 base; raw 30 USD = 30 base; undated pending
	big := mk("/api/transactions", map[string]any{"type": "expense", "account_id": usdAcc, "category_id": food, "amount": 100, "date": "2026-01-02"})
	eurTx := mk("/api/transactions", map[string]any{"type": "expense", "account_id": eurAcc, "category_id": fun, "amount": 80, "date": "2026-01-03", "tag_ids": []int64{tag}})
	small := mk("/api/transactions", map[string]any{"type": "expense", "account_id": usdAcc, "category_id": food, "amount": 30, "date": "2026-01-01"})
	undated := mk("/api/transactions", map[string]any{"type": "expense", "account_id": usdAcc, "amount": 5})

	eq := func(name string, got, want []int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s: got %v want %v", name, got, want)
			}
		}
	}

	got, _ := listTxIDs(t, a, sess, "?status=confirmed&sort_by=amount&sort_direction=desc")
	eq("amount desc (base currency)", got, []int64{eurTx, big, small})
	got, _ = listTxIDs(t, a, sess, "?status=confirmed&sort_by=amount&sort_direction=asc")
	eq("amount asc", got, []int64{small, big, eurTx})

	got, _ = listTxIDs(t, a, sess, "?sort_by=date&sort_direction=asc")
	eq("date asc undated last", got, []int64{small, big, eurTx, undated})
	got, _ = listTxIDs(t, a, sess, "?sort_by=date&sort_direction=desc")
	eq("date desc undated last", got, []int64{eurTx, big, small, undated})
	got, _ = listTxIDs(t, a, sess, "?sort_by=bogus;DROP&sort_direction=x")
	eq("unknown sort falls back to date desc", got, []int64{eurTx, big, small, undated})

	got, total := listTxIDs(t, a, sess, "?category_ids[]="+itoa(food)+"&sort_by=date&sort_direction=asc")
	eq("category_ids filter", got, []int64{small, big})
	if total != 2 {
		t.Fatalf("total %v", total)
	}
	got, total = listTxIDs(t, a, sess, "?tag_ids[]="+itoa(tag))
	eq("tag_ids filter", got, []int64{eurTx})
	if total != 1 {
		t.Fatalf("tag total %v", total)
	}
}

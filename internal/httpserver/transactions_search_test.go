//go:build integration

package httpserver

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestTransactionListSearch(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	acc, cat := seedMoney(t, a, sess)
	for _, tx := range []map[string]any{
		{"description": "Morning Coffee", "amount": 3.5},
		{"description": "Кресло", "amount": 1879},
		{"description": "Bread", "amount": 2},
	} {
		tx["type"], tx["account_id"], tx["category_id"], tx["date"] = "expense", acc, cat, "2024-01-15"
		res := a.do("POST", "/api/transactions", tx, sess.Token, sess.CSRF)
		status(t, res, http.StatusCreated, "create")
	}

	count := func(q string) int {
		t.Helper()
		out := decodeJSON(t, a.do("GET", "/api/transactions?q="+url.QueryEscape(q), nil, sess.Token, ""))
		total := int(out["meta"].(map[string]any)["total"].(float64))
		if n := len(out["data"].([]any)); n != total {
			t.Fatalf("q=%q: %d rows but total %d", q, n, total)
		}
		return total
	}
	for q, want := range map[string]int{
		"": 3, "coffee": 1, "COFFEE": 1, "кресло": 1, "1 879": 1, "1879": 1, "18.79": 0, "3.50": 1, "coffee 1879": 0,
	} {
		if got := count(q); got != want {
			t.Errorf("q=%q found %d, want %d", q, got, want)
		}
	}
}

func TestListSummaryFollowsTheSearch(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rw@test.com", "secret1", roleEditor), false)
	acc, cat := seedMoney(t, a, sess)
	for _, tx := range []map[string]any{
		{"description": "Coffee", "amount": 3},
		{"description": "Coffee beans", "amount": 5},
		{"description": "Bread", "amount": 2},
	} {
		tx["type"], tx["account_id"], tx["category_id"], tx["date"] = "expense", acc, cat, "2024-01-15"
		status(t, a.do("POST", "/api/transactions", tx, sess.Token, sess.CSRF), http.StatusCreated, "create")
	}

	summary := func(q string) map[string]any {
		t.Helper()
		out := decodeJSON(t, a.do("GET", "/api/transactions?with_summary=1&q="+url.QueryEscape(q), nil, sess.Token, ""))
		return out["summary"].(map[string]any)
	}
	if got := summary("coffee"); got["transactionsCount"].(float64) != 2 || fmt.Sprint(got["expense"]) != "8" {
		t.Errorf("coffee summary = %v, want 2 transactions and 8 spent", got)
	}
	if got := summary(""); got["transactionsCount"].(float64) != 3 || fmt.Sprint(got["expense"]) != "10" {
		t.Errorf("unfiltered summary = %v, want 3 transactions and 10 spent", got)
	}
}

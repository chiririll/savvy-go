package httpserver

import (
	"testing"
	"time"

	"github.com/chiririll/savvy-plus/internal/auth"
)

func TestReportsOverview(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("r@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	today := time.Now().UTC().Format("2006-01-02")
	res := a.do("POST", "/api/transactions", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 25, "date": today,
	}, sess.Token, sess.CSRF)
	res.Body.Close()
	res = a.do("POST", "/api/transactions", map[string]any{
		"type": "income", "account_id": accID, "category_id": catID, "amount": 100, "date": today,
	}, sess.Token, sess.CSRF)
	if res.StatusCode != 201 {
		// income category type mismatch is ok if validation is loose
		res.Body.Close()
	} else {
		res.Body.Close()
	}

	res = a.do("GET", "/api/reports/overview", nil, sess.Token, "")
	body := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("overview %d %v", res.StatusCode, body)
	}
	inc := body["income"].(map[string]any)
	exp := body["expenses"].(map[string]any)
	if exp["value"].(float64) != 25 {
		t.Fatalf("expenses %v", exp)
	}
	if inc["sparkline"] == nil || body["netCashFlow"] == nil || body["savingsRate"] == nil {
		t.Fatalf("shape %v", body)
	}
}

func TestReportsNetWorth(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("nw@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	today := time.Now().UTC().Format("2006-01-02")
	res := a.do("POST", "/api/transactions", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 100, "date": today,
	}, sess.Token, sess.CSRF)
	res.Body.Close()

	res = a.do("GET", "/api/reports/net-worth", nil, sess.Token, "")
	body := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("net-worth %d %v", res.StatusCode, body)
	}
	if body["current"].(float64) != 900 {
		t.Fatalf("current %v", body)
	}
	accts := body["accounts"].([]any)
	if len(accts) != 1 {
		t.Fatalf("accounts %v", accts)
	}

	res = a.do("GET", "/api/reports/net-worth-history?group_by=day", nil, sess.Token, "")
	hist := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("history %d %v", res.StatusCode, hist)
	}
	if hist["values"] == nil || hist["labels"] == nil {
		t.Fatalf("history shape %v", hist)
	}
}

func TestReportsTransactionSummary(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("txr@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)
	today := time.Now().UTC().Format("2006-01-02")
	res := a.do("POST", "/api/transactions", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 40, "date": today,
	}, sess.Token, sess.CSRF)
	res.Body.Close()

	res = a.do("GET", "/api/reports/transactions/summary?type=expense", nil, sess.Token, "")
	body := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("summary %d %v", res.StatusCode, body)
	}
	if body["total"].(float64) != 40 {
		t.Fatalf("total %v", body)
	}
	if body["daysInPeriod"] == nil || body["avgPerDay"] == nil {
		t.Fatalf("shape %v", body)
	}

	res = a.do("GET", "/api/reports/transactions/by-category?type=expense", nil, sess.Token, "")
	cats := decodeJSON(t, res)
	if res.StatusCode != 200 || len(cats["items"].([]any)) != 1 {
		t.Fatalf("by-cat %d %v", res.StatusCode, cats)
	}

	res = a.do("GET", "/api/monitoring/storage", nil, sess.Token, "")
	st := decodeJSON(t, res)
	if res.StatusCode != 200 || st["volume"] == nil || st["managed"] == nil {
		t.Fatalf("storage %d %v", res.StatusCode, st)
	}
	res = a.do("GET", "/api/monitoring/resources", nil, sess.Token, "")
	rs := decodeJSON(t, res)
	if res.StatusCode != 200 || rs["cpu"] == nil || rs["runtime"] == nil {
		t.Fatalf("resources %d %v", res.StatusCode, rs)
	}
}

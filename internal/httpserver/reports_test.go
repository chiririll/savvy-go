package httpserver

import (
	"testing"
	"time"

	"savvy-go/internal/auth"
)

// reportFixture is a space with an expense of 25 and an income of 100 today, on an account
// that started with 1000.
func reportFixture(t *testing.T) (*testApp, *auth.Issued) {
	t.Helper()
	a := newTestApp(t)
	sess := a.issue(a.createUser("r@test.com", "secret1", roleEditor), false)
	accID, expenseCat := seedMoney(t, a, sess)
	incomeCat := apiID(t, a, sess, "/api/categories", map[string]any{"name": "Pay", "type": "income"})
	today := time.Now().UTC().Format("2006-01-02")
	apiID(t, a, sess, "/api/transactions", map[string]any{"type": "expense", "account_id": accID, "category_id": expenseCat, "amount": 25, "date": today})
	apiID(t, a, sess, "/api/transactions", map[string]any{"type": "income", "account_id": accID, "category_id": incomeCat, "amount": 100, "date": today})
	return a, sess
}

func getReport(t *testing.T, a *testApp, sess *auth.Issued, path string) map[string]any {
	t.Helper()
	res := a.do("GET", path, nil, sess.Token, "")
	body := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("GET %s = %d %v", path, res.StatusCode, body)
	}
	return body
}

func TestReportsOverview(t *testing.T) {
	a, sess := reportFixture(t)
	body := getReport(t, a, sess, "/api/reports/overview")

	value := func(key string) float64 { return body[key].(map[string]any)["value"].(float64) }
	if value("income") != 100 || value("expenses") != 25 || value("netCashFlow") != 75 || value("savingsRate") != 75 {
		t.Fatalf("overview %v", body)
	}
	if body["currency"] != "USD" {
		t.Fatalf("currency %v", body["currency"])
	}
}

func TestReportsNetWorth(t *testing.T) {
	a, sess := reportFixture(t)

	now := getReport(t, a, sess, "/api/reports/net-worth")
	if now["current"] != float64(1075) || len(now["accounts"].([]any)) != 1 {
		t.Fatalf("net worth %v", now)
	}

	history := getReport(t, a, sess, "/api/reports/net-worth-history?group_by=day")
	dates, values := history["dates"].([]any), history["values"].([]any)
	if len(dates) == 0 || len(dates) != len(values) {
		t.Fatalf("history has %d dates and %d values", len(dates), len(values))
	}
	if first, last := values[0], values[len(values)-1]; first != float64(1000) || last != float64(1075) {
		t.Fatalf("history runs from %v to %v, want 1000 to 1075", first, last)
	}
}

func TestReportsTransactionSummary(t *testing.T) {
	a, sess := reportFixture(t)

	summary := getReport(t, a, sess, "/api/reports/transactions/summary?type=expense")
	if summary["total"] != float64(25) {
		t.Fatalf("summary %v", summary)
	}
	byCategory := getReport(t, a, sess, "/api/reports/transactions/by-category?type=expense")
	items := byCategory["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("by category %v", byCategory)
	}
}

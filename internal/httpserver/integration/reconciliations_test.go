//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"savvy-go/internal/auth"
)

type recoEnv struct {
	a     *testApp
	sess  *auth.Issued
	acc   int64
	other int64
}

func newRecoEnv(t *testing.T) *recoEnv {
	t.Helper()
	a := newTestApp(t)
	sess := a.issue(a.createUser("reco@test.com", "secret1", roleEditor), false)
	acc, _ := seedMoney(t, a, sess) // Cash, USD (base), opening balance 1000
	res := a.do("GET", "/api/currencies", nil, sess.Token, "")
	usd := int64(decodeJSON(t, res)["data"].([]any)[0].(map[string]any)["id"].(float64))
	other := apiID(t, a, sess, "/api/accounts",
		map[string]any{"name": "Bank", "type": "bank", "currency_id": usd, "initial_balance": 200})
	return &recoEnv{a: a, sess: sess, acc: acc, other: other}
}

func (e *recoEnv) post(body any) (*http.Response, map[string]any) {
	e.a.t.Helper()
	res := e.a.do("POST", "/api/reconciliations", body, e.sess.Token, e.sess.CSRF)
	return res, decodeJSON(e.a.t, res)
}

func (e *recoEnv) list(path string) []any {
	e.a.t.Helper()
	res := e.a.do("GET", path, nil, e.sess.Token, "")
	out := decodeJSON(e.a.t, res)
	if res.StatusCode != http.StatusOK {
		e.a.t.Fatalf("GET %s = %d %v", path, res.StatusCode, out)
	}
	return out["data"].([]any)
}

func TestReconcileAndListPoints(t *testing.T) {
	e := newRecoEnv(t)
	res, out := e.post(map[string]any{
		"date": "2024-01-31",
		"items": []any{
			map[string]any{"accountId": e.acc, "actualBalance": 950.5, "note": "counted"},
			map[string]any{"accountId": e.other, "actualBalance": 200},
		},
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST = %d %v", res.StatusCode, out)
	}
	data := out["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("points = %v", data)
	}
	p := data[0].(map[string]any)
	if p["accountId"].(float64) != float64(e.acc) || p["date"] != "2024-01-31" || p["actualBalance"].(float64) != 950.5 ||
		p["expectedBalance"].(float64) != 1000 || p["difference"].(float64) != -49.5 ||
		p["differenceBase"].(float64) != -49.5 || p["note"] != "counted" || p["id"] == nil {
		t.Fatalf("point = %v", p)
	}
	if q := data[1].(map[string]any); q["note"] != nil || q["difference"].(float64) != 0 {
		t.Fatalf("second point = %v", q)
	}

	if got := e.list("/api/reconciliations"); len(got) != 2 {
		t.Fatalf("list = %v", got)
	}
	if got := e.list(fmt.Sprintf("/api/reconciliations?account_ids=%d", e.other)); len(got) != 1 {
		t.Fatalf("filtered by account = %v", got)
	}
	if got := e.list("/api/reconciliations?start_date=2024-02-01"); len(got) != 0 {
		t.Fatalf("filtered by date = %v", got)
	}
	if got := e.list(fmt.Sprintf("/api/accounts/%d/reconciliations", e.acc)); len(got) != 1 {
		t.Fatalf("account list = %v", got)
	}
	status(t, e.a.do("GET", "/api/accounts/9999/reconciliations", nil, e.sess.Token, ""), 404, "missing account")

	// The same account and day again replaces the point.
	_, out = e.post(map[string]any{"date": "2024-01-31",
		"items": []any{map[string]any{"accountId": e.acc, "actualBalance": 990}}})
	if len(e.list("/api/reconciliations")) != 2 {
		t.Fatal("a repeated point must replace, not add")
	}
	if id := out["data"].([]any)[0].(map[string]any)["id"]; id != p["id"] {
		t.Fatalf("id changed %v -> %v", p["id"], id)
	}
}

func TestReconcileRefusals(t *testing.T) {
	e := newRecoEnv(t)
	debt := apiID(t, e.a, e.sess, "/api/debts", map[string]any{
		"origin": "new", "name": "Loan", "debt_type": "owed_to_me", "account_id": e.acc, "amount": 20,
		"date": "2024-01-01",
	})
	item := func(id int64, v any) map[string]any { return map[string]any{"accountId": id, "actualBalance": v} }
	for _, c := range []struct {
		name string
		body map[string]any
	}{
		{"no items", map[string]any{"date": "2024-01-31", "items": []any{}}},
		{"bad date", map[string]any{"date": "31.01.2024", "items": []any{item(e.acc, 1)}}},
		{"future", map[string]any{"date": "2999-01-01", "items": []any{item(e.acc, 1)}}},
		{"duplicate", map[string]any{"date": "2024-01-31", "items": []any{item(e.acc, 1), item(e.acc, 2)}}},
		{"unknown account", map[string]any{"date": "2024-01-31", "items": []any{item(9999, 1)}}},
		{"debt account", map[string]any{"date": "2024-01-31", "items": []any{item(debt, 1)}}},
		{"too many decimals", map[string]any{"date": "2024-01-31", "items": []any{item(e.acc, 1.005)}}},
		{"out of range", map[string]any{"date": "2024-01-31", "items": []any{item(e.acc, 1e15)}}},
		{"missing balance", map[string]any{"date": "2024-01-31", "items": []any{map[string]any{"accountId": e.acc}}}},
		{"atomic: second item fails", map[string]any{"date": "2024-01-31",
			"items": []any{item(e.acc, 5), item(9999, 1)}}},
	} {
		res, out := e.post(c.body)
		if res.StatusCode != http.StatusUnprocessableEntity || out["message"] == nil {
			t.Fatalf("%s: %d %v", c.name, res.StatusCode, out)
		}
	}
	if got := e.list("/api/reconciliations"); len(got) != 0 {
		t.Fatalf("refused requests stored %v", got)
	}
}

func TestDeleteReconciliation(t *testing.T) {
	e := newRecoEnv(t)
	_, out := e.post(map[string]any{"date": "2024-01-31",
		"items": []any{map[string]any{"accountId": e.acc, "actualBalance": 900}}})
	id := int64(out["data"].([]any)[0].(map[string]any)["id"].(float64))
	path := fmt.Sprintf("/api/reconciliations/%d", id)
	status(t, e.a.do("DELETE", path, nil, e.sess.Token, e.sess.CSRF), 204, "delete")
	status(t, e.a.do("DELETE", path, nil, e.sess.Token, e.sess.CSRF), 404, "delete again")
	if len(e.list("/api/reconciliations")) != 0 {
		t.Fatal("point not deleted")
	}
}

func TestReconciliationDays(t *testing.T) {
	e := newRecoEnv(t)
	e.post(map[string]any{"date": "2024-01-31", "items": []any{
		map[string]any{"accountId": e.acc, "actualBalance": 950},
		map[string]any{"accountId": e.other, "actualBalance": 210},
	}})
	e.post(map[string]any{"date": "2024-02-10", "items": []any{map[string]any{"accountId": e.acc, "actualBalance": 950}}})
	days := e.list("/api/reconciliations-days?start_date=2024-01-01&end_date=2024-02-28")
	if len(days) != 2 {
		t.Fatalf("days = %v", days)
	}
	d := days[1].(map[string]any)
	if d["date"] != "2024-01-31" || d["differenceBase"].(float64) != -40 || d["unconverted"] != false ||
		len(d["points"].([]any)) != 2 {
		t.Fatalf("day = %v", d)
	}
	if days[0].(map[string]any)["date"] != "2024-02-10" {
		t.Fatalf("newest first expected: %v", days)
	}
	// The dates are optional: without them every day is answered.
	if all := e.list("/api/reconciliations-days"); len(all) != 2 {
		t.Fatalf("all days = %v", all)
	}
	if from := e.list("/api/reconciliations-days?start_date=2024-02-01"); len(from) != 1 {
		t.Fatalf("days from February = %v", from)
	}
}

func TestReconciliationRoles(t *testing.T) {
	e := newRecoEnv(t)
	body := map[string]any{"date": "2024-01-31",
		"items": []any{map[string]any{"accountId": e.acc, "actualBalance": 900}}}
	viewer := e.a.issue(e.a.createUser("view@test.com", "secret1", roleViewer), false)
	status(t, e.a.do("POST", "/api/reconciliations", body, viewer.Token, viewer.CSRF), 403, "viewer writes")
	e.post(body)
	status(t, e.a.do("GET", "/api/reconciliations", nil, viewer.Token, ""), 200, "viewer reads")
	status(t, e.a.do("GET", "/api/reconciliations-days?start_date=2024-01-01&end_date=2024-02-01", nil, viewer.Token, ""),
		200, "viewer reads days")
	status(t, e.a.do("DELETE", "/api/reconciliations/1", nil, viewer.Token, viewer.CSRF), 403, "viewer deletes")

	// Another space's member is a stranger here.
	stranger := e.a.issue(e.a.createUser("x@test.com", "secret1", auth.RoleUser), false)
	status(t, e.a.inSpace(e.a.space.ID, "GET", "/api/reconciliations", nil, stranger), 404, "other space")
}

func TestReconciliationTokenPermissions(t *testing.T) {
	e := newRecoEnv(t)
	u := e.a.createUser("tok@test.com", "secret1", roleEditor)
	body := map[string]any{"date": "2024-01-31",
		"items": []any{map[string]any{"accountId": e.acc, "actualBalance": 900}}}

	none, _ := e.a.issueGrant(u, auth.Grant{Permissions: []string{"tags:write"}, AllSpaces: true}, nil)
	res := e.a.bearer("GET", "/api/reconciliations", nil, none)
	if res.StatusCode != 403 || decodeJSON(t, res)["message"] != "This API token has no accounts:read permission." {
		t.Fatal("a token without accounts:read must be refused naming the permission")
	}
	status(t, e.a.bearer("GET", "/api/reconciliations-days?start_date=2024-01-01&end_date=2024-02-01", nil, none), 403, "days")

	read, _ := e.a.issueGrant(u, auth.Grant{Permissions: []string{"accounts:read"}, AllSpaces: true}, nil)
	status(t, e.a.bearer("GET", "/api/reconciliations", nil, read), 200, "read")
	res = e.a.bearer("POST", "/api/reconciliations", body, read)
	if res.StatusCode != 403 || decodeJSON(t, res)["message"] != "This API token has no accounts:write permission." {
		t.Fatal("write with a read grant must name accounts:write")
	}

	write, _ := e.a.issueGrant(u, auth.Grant{Permissions: []string{"accounts:write"}, AllSpaces: true}, nil)
	status(t, e.a.bearer("POST", "/api/reconciliations", body, write), 201, "write")
	res = e.a.bearer("DELETE", "/api/reconciliations/1", nil, write)
	if res.StatusCode != 403 || decodeJSON(t, res)["message"] != "This API token has no accounts:delete permission." {
		t.Fatal("delete needs accounts:delete")
	}
	del, _ := e.a.issueGrant(u, auth.Grant{Permissions: []string{"accounts:delete"}, AllSpaces: true}, nil)
	status(t, e.a.bearer("DELETE", "/api/reconciliations/1", nil, del), 204, "delete")
}

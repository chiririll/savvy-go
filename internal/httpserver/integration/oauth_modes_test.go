//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const cimdClientID = "https://app.example/oauth/client.json"

type docFetcher map[string]string

func (f docFetcher) Fetch(_ context.Context, u string) ([]byte, string, error) {
	if d, ok := f[u]; ok {
		return []byte(d), "", nil
	}
	return nil, "", errors.New("404")
}

func (a *testApp) useCIMD(allowHost string) {
	a.t.Helper()
	a.svc.OAuth.Fetcher = docFetcher{cimdClientID: fmt.Sprintf(
		`{"client_id":%q,"client_name":"Doc App","redirect_uris":[%q]}`, cimdClientID, oauthRedirect)}
	if allowHost != "" {
		if err := a.svc.OAuth.AddCIMDHost(context.Background(), allowHost); err != nil {
			a.t.Fatal(err)
		}
	}
}

func (a *testApp) exchangeForm(clientID, code string) url.Values {
	return url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code},
		"code_verifier": {oauthVerifier}, "redirect_uri": {oauthRedirect},
	}
}

func TestCIMDClientCanConnectOnAllowedHost(t *testing.T) {
	a := newTestApp(t)
	a.useCIMD("app.example")
	u := a.issue(a.createUser("c@a.com", "secret1", roleEditor), false)

	res := a.do("GET", "/api/oauth/authorize?client_id="+url.QueryEscape(cimdClientID)+"&redirect_uri="+url.QueryEscape(oauthRedirect), nil, u.Token, u.CSRF)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("consent info = %d", res.StatusCode)
	}
	data := decodeJSON(t, res)["data"].(map[string]any)
	if data["client_name"] != "Doc App" || data["client_host"] != "app.example" {
		t.Fatalf("consent info = %v", data)
	}

	code := codeFrom(t, a.approve(u.Token, u.CSRF, cimdClientID, true))
	if st, body := a.token(a.exchangeForm(cimdClientID, code)); st != 200 || body["access_token"] == nil {
		t.Fatalf("token = %d %v", st, body)
	}
}

func TestCIMDClientRefusedOffTheAllowlist(t *testing.T) {
	a := newTestApp(t)
	a.useCIMD("")
	u := a.issue(a.createUser("c@a.com", "secret1", roleEditor), false)
	res := a.do("GET", "/api/oauth/authorize?client_id="+url.QueryEscape(cimdClientID)+"&redirect_uri="+url.QueryEscape(oauthRedirect), nil, u.Token, u.CSRF)
	if res.StatusCode != http.StatusBadRequest && res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("consent info = %d, want a client error", res.StatusCode)
	}
}

func TestOwnConfidentialClientNeedsItsSecret(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@a.com", "secret1", "admin")
	as := a.issue(admin, false)
	res := a.do("POST", "/api/admin/oauth/clients", map[string]any{
		"name": "Mine", "redirect_uris": []string{oauthRedirect}, "confidential": true,
	}, as.Token, as.CSRF)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", res.StatusCode)
	}
	d := decodeJSON(t, res)["data"].(map[string]any)
	id, secret := d["client_id"].(string), d["client_secret"].(string)
	if id == "" || secret == "" || d["confidential"] != true || d["kind"] != "manual" {
		t.Fatalf("created = %v", d)
	}

	// The list never shows the secret again.
	list := decodeJSON(t, a.do("GET", "/api/admin/oauth/clients", nil, as.Token, as.CSRF))["data"].([]any)
	if _, leaked := list[0].(map[string]any)["client_secret"]; leaked {
		t.Fatal("list leaks the secret")
	}

	user := a.issue(a.createUser("u@a.com", "secret1", roleEditor), false)
	exchange := func(extra url.Values, basic [2]string) (int, map[string]any) {
		code := codeFrom(t, a.approve(user.Token, user.CSRF, id, true))
		form := a.exchangeForm(id, code)
		for k, v := range extra {
			form[k] = v
		}
		req, _ := http.NewRequest("POST", a.srv.URL+"/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if basic[0] != "" {
			req.SetBasicAuth(basic[0], basic[1])
		}
		r, err := a.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return r.StatusCode, decodeJSON(t, r)
	}
	if st, b := exchange(nil, [2]string{}); st == 200 || b["error"] != "invalid_client" {
		t.Fatalf("no secret = %d %v", st, b)
	}
	if st, b := exchange(url.Values{"client_secret": {"wrong"}}, [2]string{}); st == 200 || b["error"] != "invalid_client" {
		t.Fatalf("wrong secret = %d %v", st, b)
	}
	if st, b := exchange(url.Values{"client_secret": {secret}}, [2]string{}); st != 200 {
		t.Fatalf("client_secret_post = %d %v", st, b)
	}
	if st, b := exchange(url.Values{"client_id": nil}, [2]string{id, secret}); st != 200 {
		t.Fatalf("client_secret_basic = %d %v", st, b)
	}
}

func TestAdminManagesCIMDHosts(t *testing.T) {
	a := newTestApp(t)
	admin := a.issue(a.createUser("admin@a.com", "secret1", "admin"), false)
	hosts := func() []any {
		return decodeJSON(t, a.do("GET", "/api/admin/oauth/cimd-hosts", nil, admin.Token, admin.CSRF))["data"].([]any)
	}
	if len(hosts()) != 0 {
		t.Fatal("allowlist must start empty")
	}
	add := func(host string) int {
		return a.do("POST", "/api/admin/oauth/cimd-hosts", map[string]any{"host": host}, admin.Token, admin.CSRF).StatusCode
	}
	if st := add("https://bad/path"); st != http.StatusUnprocessableEntity {
		t.Fatalf("bad host = %d", st)
	}
	if st := add("Claude.ai"); st != http.StatusCreated {
		t.Fatalf("add = %d", st)
	}
	if h := hosts(); len(h) != 1 || h[0].(map[string]any)["host"] != "claude.ai" {
		t.Fatalf("hosts = %v", h)
	}
	if r := a.do("DELETE", "/api/admin/oauth/cimd-hosts/claude.ai", nil, admin.Token, admin.CSRF); r.StatusCode != http.StatusNoContent {
		t.Fatalf("remove = %d", r.StatusCode)
	}
	if r := a.do("DELETE", "/api/admin/oauth/cimd-hosts/claude.ai", nil, admin.Token, admin.CSRF); r.StatusCode != http.StatusNotFound {
		t.Fatalf("remove again = %d", r.StatusCode)
	}
}

func TestOnlyAdminsManageOwnClientsAndHosts(t *testing.T) {
	a := newTestApp(t)
	u := a.issue(a.createUser("u@a.com", "secret1", roleEditor), false)
	for _, c := range [][2]string{
		{"GET", "/api/admin/oauth/cimd-hosts"}, {"POST", "/api/admin/oauth/cimd-hosts"}, {"POST", "/api/admin/oauth/clients"},
	} {
		if r := a.do(c[0], c[1], map[string]any{"host": "x.example"}, u.Token, u.CSRF); r.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s = %d, want 403", c[0], c[1], r.StatusCode)
		}
	}
}

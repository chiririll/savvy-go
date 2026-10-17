package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/sso"
)

func TestIdentityProviderCRUDPresets(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	rw := a.createUser("rw@test.com", "secret1", roleEditor)
	adm := a.issue(admin, false)
	user := a.issue(rw, false)

	denied := a.do("POST", "/api/identity-providers", map[string]any{
		"name": "X", "slug": "x", "preset": "google", "fields": map[string]any{},
	}, user.Token, user.CSRF)
	if denied.StatusCode != 403 {
		t.Fatalf("non-admin %d", denied.StatusCode)
	}
	denied.Body.Close()

	presets := a.do("GET", "/api/auth/sso/presets", nil, adm.Token, "")
	if presets.StatusCode != 200 {
		t.Fatalf("presets %d", presets.StatusCode)
	}
	raw, _ := io.ReadAll(presets.Body)
	if !strings.Contains(string(raw), `"key":"github"`) || !strings.Contains(string(raw), `"key":"custom_saml"`) {
		t.Fatalf("catalog %s", raw)
	}
	ro := a.createUser("ro@test.com", "secret1", roleViewer)
	roIss := a.issue(ro, false)
	forbid := a.do("GET", "/api/auth/sso/presets", nil, roIss.Token, "")
	if forbid.StatusCode != 403 {
		t.Fatalf("readonly presets %d", forbid.StatusCode)
	}
	forbid.Body.Close()

	badSlug := a.do("POST", "/api/identity-providers", map[string]any{
		"name": "Bad", "slug": "Not A Slug", "preset": "google", "fields": map[string]any{},
	}, adm.Token, adm.CSRF)
	if badSlug.StatusCode != 422 {
		t.Fatalf("slug %d %v", badSlug.StatusCode, decodeJSON(t, badSlug))
	}

	missing := a.do("POST", "/api/identity-providers", map[string]any{
		"name": "Okta", "slug": "okta", "preset": "okta",
		"fields": map[string]any{"client_id": "cid"},
	}, adm.Token, adm.CSRF)
	miss := decodeJSON(t, missing)
	if missing.StatusCode != 422 || miss["error"] != "missing_field" {
		t.Fatalf("missing %d %v", missing.StatusCode, miss)
	}

	created := a.do("POST", "/api/identity-providers", map[string]any{
		"name": "Company Okta", "slug": "company-okta", "preset": "okta", "enabled": true,
		"fields": map[string]any{"domain": "example.okta.com", "client_id": "cid", "client_secret": "shh"},
	}, adm.Token, adm.CSRF)
	if created.StatusCode != 201 {
		t.Fatalf("create %d %v", created.StatusCode, decodeJSON(t, created))
	}
	body := decodeJSON(t, created)
	if body["protocol"] != "oidc" || body["hasClientSecret"] != true {
		t.Fatalf("create body %v", body)
	}
	enc, _ := json.Marshal(body)
	if strings.Contains(string(enc), "shh") {
		t.Fatal("leaked secret")
	}

	id := int64(body["id"].(float64))
	patch := a.do("PATCH", "/api/identity-providers/"+itoa(id), map[string]any{
		"name": "GH Renamed", "fields": map[string]any{"domain": "example.okta.com", "client_id": "cid", "client_secret": ""},
	}, adm.Token, adm.CSRF)
	if patch.StatusCode != 200 {
		t.Fatalf("patch %d %v", patch.StatusCode, decodeJSON(t, patch))
	}
	patch.Body.Close()
	var secret string
	_ = a.db.QueryRow(`SELECT secrets FROM identity_providers WHERE id=?`, id).Scan(&secret)
	if !strings.Contains(secret, "shh") {
		t.Fatalf("kept secret %s", secret)
	}

	pub := a.do("GET", "/api/auth/sso/providers", nil, "", "")
	pubBody := decodeJSON(t, pub)
	data, _ := pubBody["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("public %v", pubBody)
	}
}

func TestSSOTicketExchange(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", roleEditor)
	ticket, err := a.svc.SSO.IssueTicket(context.Background(), u.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	res := a.do("POST", "/api/auth/sso/exchange", map[string]string{"ticket": ticket}, "", "")
	if res.StatusCode != 200 {
		t.Fatalf("exchange %d %v", res.StatusCode, decodeJSON(t, res))
	}
	body := decodeJSON(t, res)
	user, _ := body["user"].(map[string]any)
	if user["email"] != "u@test.com" {
		t.Fatalf("user %v", body)
	}
	if body["token"] != nil {
		t.Fatal("must not leak token")
	}
	if cookieNamed(res, "svy_session") == nil {
		t.Fatal("session cookie")
	}
	replay := a.do("POST", "/api/auth/sso/exchange", map[string]string{"ticket": ticket}, "", "")
	if replay.StatusCode != 410 {
		t.Fatalf("replay %d", replay.StatusCode)
	}
	replay.Body.Close()

	unknown := a.do("POST", "/api/auth/sso/exchange", map[string]string{"ticket": "nope"}, "", "")
	if unknown.StatusCode != 410 {
		t.Fatalf("unknown %d", unknown.StatusCode)
	}
	unknown.Body.Close()

	u2 := a.createUser("tf@test.com", "secret1", roleEditor)
	_, _ = a.db.Exec(`UPDATE users SET two_factor_enabled=1, two_factor_confirmed=1 WHERE id=?`, u2.ID)
	t2, _ := a.svc.SSO.IssueTicket(context.Background(), u2.ID, true)
	ch := a.do("POST", "/api/auth/sso/exchange", map[string]string{"ticket": t2}, "", "")
	chBody := decodeJSON(t, ch)
	if chBody["requires_2fa"] != true || chBody["two_factor_token"] == nil {
		t.Fatalf("2fa ticket %v", chBody)
	}
}

func TestSSOGithubSignInProvisionsUser(t *testing.T) {
	a := newTestApp(t)
	a.createUser("root@test.com", "secret1", auth.RoleAdmin)
	_, err := a.svc.SSO.Create(context.Background(), sso.IdPWrite{
		Name: "GitHub", Slug: "github", Preset: "github", Enabled: boolPtr(true),
		Fields: map[string]any{"client_id": "cid", "client_secret": "csecret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "gho_x", "token_type": "bearer"})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 4242, "login": "octo", "name": "Octo", "email": nil})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": "octo@test.com", "primary": true, "verified": true},
		})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	a.svc.SSO.HTTP = &http.Client{Transport: rewriteHost{base: ts.URL}}

	redir := doNoFollow(t, a, "GET", "/api/auth/sso/github/redirect")
	if redir.StatusCode != 302 {
		t.Fatalf("redirect %d", redir.StatusCode)
	}
	loc := redir.Header.Get("Location")
	if !strings.Contains(loc, "github.com/login/oauth/authorize") {
		t.Fatalf("loc %s", loc)
	}
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")
	cb := doNoFollow(t, a, "GET", "/api/auth/sso/github/callback?code=abc&state="+state)
	if cb.StatusCode != 302 {
		t.Fatalf("callback %d", cb.StatusCode)
	}
	cbLoc := cb.Header.Get("Location")
	if !strings.Contains(cbLoc, "ticket=") {
		t.Fatalf("callback loc %s", cbLoc)
	}
	q, _ := url.Parse(cbLoc)
	ex := a.do("POST", "/api/auth/sso/exchange", map[string]string{"ticket": q.Query().Get("ticket")}, "", "")
	if ex.StatusCode != 200 {
		t.Fatalf("exchange %d %v", ex.StatusCode, decodeJSON(t, ex))
	}
	exBody := decodeJSON(t, ex)
	user, _ := exBody["user"].(map[string]any)
	if user["email"] != "octo@test.com" {
		t.Fatalf("provisioned %v", exBody)
	}

}

func TestSSOSAMLMetadataRedirectRejectedResponse(t *testing.T) {
	a := newTestApp(t)

	_, err := a.svc.SSO.Create(context.Background(), sso.IdPWrite{
		Name: "Corp SAML", Slug: "saml", Preset: "custom_saml", Enabled: boolPtr(true),
		Fields: map[string]any{
			"idp_entity_id": "https://idp.example/saml",
			"idp_sso_url":   "https://idp.example/saml/sso",
			"idp_x509_cert": "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := a.do("GET", "/api/auth/sso/saml/metadata", nil, "", "")
	b, _ := io.ReadAll(meta.Body)
	if meta.StatusCode != 200 || !strings.Contains(string(b), "EntityDescriptor") ||
		!strings.Contains(string(b), "AssertionConsumerService") {
		t.Fatalf("metadata %d %s", meta.StatusCode, b)
	}
	if !strings.Contains(meta.Header.Get("Content-Type"), "xml") {
		t.Fatalf("ctype %s", meta.Header.Get("Content-Type"))
	}
	sso := doNoFollow(t, a, "GET", "/api/auth/sso/saml/redirect")
	if sso.StatusCode != 302 || !strings.Contains(sso.Header.Get("Location"), "SAMLRequest=") {
		t.Fatalf("saml redirect %d %s", sso.StatusCode, sso.Header.Get("Location"))
	}
	form := url.Values{"RelayState": {"nope"}, "SAMLResponse": {"PGhlbGxvPg=="}}
	req, _ := http.NewRequest("POST", a.srv.URL+"/api/auth/sso/saml/acs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	acs, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	acs.Body.Close()
	if acs.StatusCode != 302 || !strings.Contains(acs.Header.Get("Location"), "error=") {
		t.Fatalf("acs %d %s", acs.StatusCode, acs.Header.Get("Location"))
	}
}

func TestSSOProvisioningLinkJIT(t *testing.T) {
	a := newTestApp(t)
	a.createUser("root@test.com", "secret1", auth.RoleAdmin)
	existing := a.createUser("jane@test.com", "secret1", roleEditor)
	p, err := a.svc.SSO.Create(context.Background(), sso.IdPWrite{
		Name: "IdP", Slug: "idp", Preset: "custom_oidc", Enabled: boolPtr(true),
		Fields: map[string]any{
			"discovery_url": "https://idp.test/.well-known/openid-configuration",
			"client_id":     "c", "client_secret": "s",
		},
		RoleMapping: []any{map[string]any{"claim": "groups", "operator": "contains", "value": "admins", "role": "admin"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	user, _, err := a.svc.SSO.Provision(context.Background(), *p, sso.NormalizedIdentity{
		Subject: "ext-1", Email: "jane@test.com", EmailVerified: true, Name: "Jane",
	})
	if err != nil || user.ID != existing.ID {
		t.Fatalf("link %v %v", user, err)
	}
	_, _, err = a.svc.SSO.Provision(context.Background(), *p, sso.NormalizedIdentity{
		Subject: "ext-2", Email: "jane@test.com", EmailVerified: false, Name: "Jane",
	})
	if err == nil {
		t.Fatal("unverified should not JIT when email exists without link")
	}
	fresh, _, err := a.svc.SSO.Provision(context.Background(), *p, sso.NormalizedIdentity{
		Subject: "ext-3", Email: "new@test.com", EmailVerified: true, Name: "New",
		Groups: []string{"admins"}, Raw: map[string]any{"groups": []any{"admins"}},
	})
	if err != nil || fresh.Email != "new@test.com" || fresh.Role != auth.RoleAdmin || !fresh.IsSSOOnly {
		t.Fatalf("jit %+v %v", fresh, err)
	}
	if list, err := a.svc.Spaces.ForUser(context.Background(), fresh.ID); err != nil || len(list) != 0 {
		t.Fatalf("jit user has spaces %v %v", list, err)
	}
}

func doNoFollow(t *testing.T, a *testApp, method, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, a.srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func boolPtr(v bool) *bool { return &v }

type rewriteHost struct{ base string }

// RoundTrip sends the requests meant for GitHub to the test server.
func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	u, _ := url.Parse(r.base)
	if req.URL.Host == "github.com" || req.URL.Host == "api.github.com" {
		req.URL.Host, req.URL.Scheme = u.Host, u.Scheme
	}
	return http.DefaultTransport.RoundTrip(req)
}

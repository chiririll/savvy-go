package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/domain"
)

func TestTwoFactorEnableConfirmLoginAndRecovery(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("otp@test.com", "secret1", auth.RoleReadWrite)
	iss := a.issue(u, false)

	en := a.do("POST", "/api/auth/2fa/enable", map[string]any{}, iss.Token, iss.CSRF)
	if en.StatusCode != 200 {
		t.Fatalf("enable %d %v", en.StatusCode, decodeJSON(t, en))
	}
	enBody := decodeJSON(t, en)
	secret, _ := enBody["secret"].(string)
	if secret == "" || enBody["qr_code_url"] == nil {
		t.Fatalf("enable body %v", enBody)
	}

	bad := a.do("POST", "/api/auth/2fa/confirm", map[string]string{"code": "000000"}, iss.Token, iss.CSRF)
	if bad.StatusCode != 422 {
		t.Fatalf("bad confirm %d", bad.StatusCode)
	}
	bad.Body.Close()

	code := auth.TOTPNow(secret)
	ok := a.do("POST", "/api/auth/2fa/confirm", map[string]string{"code": code}, iss.Token, iss.CSRF)
	if ok.StatusCode != 200 {
		t.Fatalf("confirm %d %v", ok.StatusCode, decodeJSON(t, ok))
	}
	confirmed := decodeJSON(t, ok)
	codes, _ := confirmed["recovery_codes"].([]any)
	if len(codes) != 8 {
		t.Fatalf("codes %v", confirmed)
	}

	login := a.do("POST", "/api/auth/login", map[string]string{"email": "otp@test.com", "password": "secret1"}, "", "")
	loginBody := decodeJSON(t, login)
	tok, _ := loginBody["two_factor_token"].(string)
	if tok == "" {
		t.Fatalf("login 2fa %v", loginBody)
	}

	verify := a.do("POST", "/api/auth/2fa/verify", map[string]any{
		"two_factor_token": tok, "code": auth.TOTPNow(secret),
	}, "", "")
	if verify.StatusCode != 200 {
		t.Fatalf("verify %d %v", verify.StatusCode, decodeJSON(t, verify))
	}
	if cookieNamed(verify, "svy_session") == nil {
		t.Fatal("expected session after 2fa")
	}
	verify.Body.Close()

	login2 := a.do("POST", "/api/auth/login", map[string]string{"email": "otp@test.com", "password": "secret1"}, "", "")
	tok2, _ := decodeJSON(t, login2)["two_factor_token"].(string)
	rec := a.do("POST", "/api/auth/2fa/verify", map[string]any{
		"two_factor_token": tok2, "code": codes[0],
	}, "", "")
	if rec.StatusCode != 200 {
		t.Fatalf("recovery verify %d %v", rec.StatusCode, decodeJSON(t, rec))
	}
	rec.Body.Close()

	st := a.do("GET", "/api/auth/2fa/status", nil, iss.Token, "")
	stBody := decodeJSON(t, st)
	if stBody["enabled"] != true {
		t.Fatalf("status %v", stBody)
	}
	if rem, _ := stBody["recovery_codes_remaining"].(float64); rem != 7 {
		t.Fatalf("remaining %v", stBody)
	}

	off := a.do("POST", "/api/auth/2fa/disable", map[string]string{"code": auth.TOTPNow(secret)}, iss.Token, iss.CSRF)
	if off.StatusCode != 200 {
		t.Fatalf("disable %d %v", off.StatusCode, decodeJSON(t, off))
	}
}

func TestLegacyLaravelTOTPStillVerifies(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	appKey := "base64:" + base64.StdEncoding.EncodeToString(key)
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := auth.EncryptLaravel(appKey, secret)
	if err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t)
	a.s.twoFactor.AppKey = appKey
	u := a.createUser("legacy2fa@test.com", "secret1", auth.RoleReadWrite)
	_, _ = a.db.Exec(`UPDATE users SET two_factor_enabled=1, two_factor_confirmed=1, two_factor_secret=? WHERE id=?`, ct, u.ID)

	login := a.do("POST", "/api/auth/login", map[string]string{"email": "legacy2fa@test.com", "password": "secret1"}, "", "")
	tok, _ := decodeJSON(t, login)["two_factor_token"].(string)
	if tok == "" {
		t.Fatal("expected 2fa challenge")
	}
	res := a.do("POST", "/api/auth/2fa/verify", map[string]any{
		"two_factor_token": tok, "code": auth.TOTPNow(secret),
	}, "", "")
	if res.StatusCode != 200 {
		t.Fatalf("legacy verify %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()

	var stored string
	_ = a.db.QueryRow(`SELECT two_factor_secret FROM users WHERE id=?`, u.ID).Scan(&stored)
	if stored != secret {
		t.Fatalf("secret not persisted plaintext %q", stored)
	}
}

func TestWebauthnOptionsAndCredentialCRUD(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("pk@test.com", "secret1", auth.RoleReadWrite)
	iss := a.issue(u, false)

	opts := a.do("POST", "/api/auth/webauthn/register/options", map[string]any{}, iss.Token, iss.CSRF)
	if opts.StatusCode != 200 {
		t.Fatalf("register options %d %v", opts.StatusCode, decodeJSON(t, opts))
	}
	body := decodeJSON(t, opts)
	if body["token"] == nil {
		t.Fatalf("options %v", body)
	}
	opt, _ := body["options"].(map[string]any)
	if opt["challenge"] == nil || opt["rp"] == nil || opt["publicKey"] != nil {
		t.Fatalf("creation options %v", body["options"])
	}

	bad := a.do("POST", "/api/auth/webauthn/register/verify", map[string]any{
		"token": "nope", "response": map[string]any{"id": "x"},
	}, iss.Token, iss.CSRF)
	if bad.StatusCode != 422 {
		t.Fatalf("bad verify %d", bad.StatusCode)
	}
	bad.Body.Close()

	loginOpts := a.do("POST", "/api/auth/webauthn/login/options", map[string]any{}, "", "")
	if loginOpts.StatusCode != 200 {
		t.Fatalf("login options %d %v", loginOpts.StatusCode, decodeJSON(t, loginOpts))
	}
	loginOpts.Body.Close()

	_, _ = a.db.Exec(`INSERT INTO webauthn_credentials (user_id, credential_id, name, record, created_at)
		VALUES (?, 'cred-1', 'Laptop', '{}', '2026-01-01T00:00:00Z')`, u.ID)
	list := a.do("GET", "/api/auth/webauthn/credentials", nil, iss.Token, "")
	listed := decodeJSON(t, list)
	creds, _ := listed["credentials"].([]any)
	if len(creds) != 1 {
		t.Fatalf("list %v", listed)
	}
	id := int64(creds[0].(map[string]any)["id"].(float64))
	ren := a.do("PATCH", "/api/auth/webauthn/credentials/"+itoa(id), map[string]string{"name": "Phone"}, iss.Token, iss.CSRF)
	if ren.StatusCode != 200 {
		t.Fatalf("rename %d", ren.StatusCode)
	}
	ren.Body.Close()
	del := a.do("DELETE", "/api/auth/webauthn/credentials/"+itoa(id), map[string]any{}, iss.Token, iss.CSRF)
	if del.StatusCode != 200 {
		t.Fatalf("delete %d", del.StatusCode)
	}
	del.Body.Close()
}

func TestIdentityProviderCRUDAndPresets(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	rw := a.createUser("rw@test.com", "secret1", auth.RoleReadWrite)
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
	ro := a.createUser("ro@test.com", "secret1", auth.RoleReadOnly)
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
	u := a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
	ticket, err := a.s.sso.IssueTicket(context.Background(), u.ID, false)
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

	u2 := a.createUser("tf@test.com", "secret1", auth.RoleReadWrite)
	_, _ = a.db.Exec(`UPDATE users SET two_factor_enabled=1, two_factor_confirmed=1 WHERE id=?`, u2.ID)
	t2, _ := a.s.sso.IssueTicket(context.Background(), u2.ID, true)
	ch := a.do("POST", "/api/auth/sso/exchange", map[string]string{"ticket": t2}, "", "")
	chBody := decodeJSON(t, ch)
	if chBody["requires_2fa"] != true || chBody["two_factor_token"] == nil {
		t.Fatalf("2fa ticket %v", chBody)
	}
}

func TestSSOGithubCallbackAndSAMLMetadata(t *testing.T) {
	a := newTestApp(t)
	a.createUser("root@test.com", "secret1", auth.RoleAdmin)
	_, err := a.s.sso.Create(context.Background(), domain.IdPWrite{
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
	a.s.sso.HTTP = &http.Client{Transport: rewriteHost{base: ts.URL}}

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

	saml, err := a.s.sso.Create(context.Background(), domain.IdPWrite{
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
	_ = saml
	meta := a.do("GET", "/api/auth/sso/saml/metadata", nil, "", "")
	b, _ := io.ReadAll(meta.Body)
	if meta.StatusCode != 200 || !strings.Contains(string(b), "EntityDescriptor") || !strings.Contains(string(b), "AssertionConsumerService") {
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

func TestSSOProvisioningLinkAndJIT(t *testing.T) {
	a := newTestApp(t)
	a.createUser("root@test.com", "secret1", auth.RoleAdmin)
	existing := a.createUser("jane@test.com", "secret1", auth.RoleReadWrite)
	p, err := a.s.sso.Create(context.Background(), domain.IdPWrite{
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
	user, _, err := a.s.sso.Provision(context.Background(), *p, domain.NormalizedIdentity{
		Subject: "ext-1", Email: "jane@test.com", EmailVerified: true, Name: "Jane",
	})
	if err != nil || user.ID != existing.ID {
		t.Fatalf("link %v %v", user, err)
	}
	_, _, err = a.s.sso.Provision(context.Background(), *p, domain.NormalizedIdentity{
		Subject: "ext-2", Email: "jane@test.com", EmailVerified: false, Name: "Jane",
	})
	if err == nil {
		t.Fatal("unverified should not JIT when email exists without link")
	}
	fresh, _, err := a.s.sso.Provision(context.Background(), *p, domain.NormalizedIdentity{
		Subject: "ext-3", Email: "new@test.com", EmailVerified: true, Name: "New",
		Groups: []string{"admins"}, Raw: map[string]any{"groups": []any{"admins"}},
	})
	if err != nil || fresh.Email != "new@test.com" || fresh.Role != auth.RoleAdmin || !fresh.IsSSOOnly {
		t.Fatalf("jit %+v %v", fresh, err)
	}
}

func TestSSOPresetHelpers(t *testing.T) {
	disco := "https://login.microsoftonline.com/{tenantid}/v2.0"
	if !domain.MatchesIssuer("entra", disco, map[string]any{"iss": "https://login.microsoftonline.com/abc-123/v2.0", "tid": "abc-123"}) {
		t.Fatal("entra tenant match")
	}
	if domain.MatchesIssuer("entra", disco, map[string]any{"iss": "https://login.microsoftonline.com/evil/v2.0", "tid": "abc-123"}) {
		t.Fatal("entra evil")
	}
	p := domain.IdentityProvider{Preset: "okta", Config: map[string]any{"domain": "example.okta.com"}}
	if got := domain.PresetDiscoveryURL(p); got != "https://example.okta.com/.well-known/openid-configuration" {
		t.Fatalf("okta disco %s", got)
	}
	p.Config["auth_server_id"] = "aus1a2b3c"
	if got := domain.PresetDiscoveryURL(p); got != "https://example.okta.com/oauth2/aus1a2b3c/.well-known/openid-configuration" {
		t.Fatalf("okta custom %s", got)
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

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	u, _ := url.Parse(r.base)
	switch req.URL.Host {
	case "github.com":
		req.URL.Host = u.Host
		req.URL.Scheme = u.Scheme
		if strings.HasPrefix(req.URL.Path, "/login/oauth/") {
			// keep path
		}
	case "api.github.com":
		req.URL.Host = u.Host
		req.URL.Scheme = u.Scheme
	}
	return http.DefaultTransport.RoundTrip(req)
}

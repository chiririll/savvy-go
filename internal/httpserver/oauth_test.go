package httpserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/oauth"
)

const (
	oauthRedirect = "https://claude.ai/api/mcp/auth_callback"
	oauthVerifier = "vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv"
)

func oauthChallenge() string {
	sum := sha256.Sum256([]byte(oauthVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (a *testApp) getJSON(path string) (int, map[string]any) {
	a.t.Helper()
	res, err := a.client.Get(a.srv.URL + path)
	if err != nil {
		a.t.Fatal(err)
	}
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func (a *testApp) registerClient(redirects ...string) string {
	a.t.Helper()
	if len(redirects) == 0 {
		redirects = []string{oauthRedirect}
	}
	a.s.oauth.OpenRegistration(oauth.RegistrationWindow)
	raw, _ := json.Marshal(map[string]any{"client_name": "Claude", "redirect_uris": redirects})
	res, err := a.client.Post(a.srv.URL+"/oauth/register", "application/json", bytes.NewReader(raw))
	if err != nil {
		a.t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		a.t.Fatalf("register: %d", res.StatusCode)
	}
	return decodeJSON(a.t, res)["client_id"].(string)
}

// registerClientNoOpen registers inside a window somebody else opened.
func (a *testApp) registerClientNoOpen() {
	a.t.Helper()
	raw, _ := json.Marshal(map[string]any{"client_name": "Claude", "redirect_uris": []string{oauthRedirect}})
	res, err := a.client.Post(a.srv.URL+"/oauth/register", "application/json", bytes.NewReader(raw))
	if err != nil {
		a.t.Fatal(err)
	}
	status(a.t, res, http.StatusCreated, "register in the window")
}

func (a *testApp) token(form url.Values) (int, map[string]any) {
	a.t.Helper()
	res, err := a.client.PostForm(a.srv.URL+"/oauth/token", form)
	if err != nil {
		a.t.Fatal(err)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		a.t.Errorf("token response Cache-Control = %q", cc)
	}
	return res.StatusCode, decodeJSON(a.t, res)
}

// approve plays the consent screen: the SPA posts the user's decision.
func (a *testApp) approve(sessToken, csrf, clientID string, approve bool) string {
	a.t.Helper()
	res := a.do("POST", "/api/oauth/authorize", map[string]any{
		"client_id": clientID, "redirect_uri": oauthRedirect, "state": "st-1",
		"code_challenge": oauthChallenge(), "code_challenge_method": "S256", "approve": approve,
	}, sessToken, csrf)
	if res.StatusCode != http.StatusOK {
		a.t.Fatalf("approve: %d", res.StatusCode)
	}
	data := decodeJSON(a.t, res)["data"].(map[string]any)
	return data["redirect_url"].(string)
}

// connect plays the whole sign-in of an app and returns its client id and access token.
func (a *testApp) connect(sess *auth.Issued) (clientID, access string) {
	a.t.Helper()
	id := a.registerClient()
	code := codeFrom(a.t, a.approve(sess.Token, sess.CSRF, id, true))
	status2, tok := a.token(url.Values{"grant_type": {"authorization_code"}, "client_id": {id}, "code": {code}, "code_verifier": {oauthVerifier}, "redirect_uri": {oauthRedirect}})
	if status2 != 200 {
		a.t.Fatalf("token = %d %v", status2, tok)
	}
	return id, tok["access_token"].(string)
}

func codeFrom(t *testing.T, redirectURL string) string {
	t.Helper()
	u, err := url.Parse(redirectURL)
	if err != nil || !strings.HasPrefix(redirectURL, oauthRedirect+"?") {
		t.Fatalf("redirect = %q (%v)", redirectURL, err)
	}
	if u.Query().Get("state") != "st-1" {
		t.Fatalf("state lost: %q", redirectURL)
	}
	return u.Query().Get("code")
}

func TestOAuthMetadata(t *testing.T) {
	a := newTestApp(t)
	code, pr := a.getJSON("/.well-known/oauth-protected-resource")
	if code != 200 || pr["resource"] != "http://localhost:8080/mcp" {
		t.Fatalf("protected resource = %d %v", code, pr)
	}
	if servers, _ := pr["authorization_servers"].([]any); len(servers) != 1 || servers[0] != "http://localhost:8080" {
		t.Fatalf("authorization_servers = %v", pr["authorization_servers"])
	}
	// MCP clients also try the path-suffixed form (RFC 9728 §3.1).
	if code, _ := a.getJSON("/.well-known/oauth-protected-resource/mcp"); code != 200 {
		t.Fatalf("suffixed protected resource = %d", code)
	}

	// Registration is advertised only while an admin has it open.
	if _, closed := func() (int, map[string]any) { return a.getJSON("/.well-known/oauth-authorization-server") }(); closed["registration_endpoint"] != nil {
		t.Fatalf("closed registration is advertised: %v", closed)
	}
	a.s.oauth.OpenRegistration(oauth.RegistrationWindow)
	code, as := a.getJSON("/.well-known/oauth-authorization-server")
	if code != 200 || as["issuer"] != "http://localhost:8080" ||
		as["authorization_endpoint"] != "http://localhost:8080/oauth/authorize" ||
		as["token_endpoint"] != "http://localhost:8080/oauth/token" ||
		as["registration_endpoint"] != "http://localhost:8080/oauth/register" {
		t.Fatalf("authorization server = %d %v", code, as)
	}
	if m, _ := as["code_challenge_methods_supported"].([]any); len(m) != 1 || m[0] != "S256" {
		t.Fatalf("PKCE methods = %v", as["code_challenge_methods_supported"])
	}
	if m, _ := as["token_endpoint_auth_methods_supported"].([]any); len(m) != 1 || m[0] != "none" {
		t.Fatalf("auth methods = %v", as["token_endpoint_auth_methods_supported"])
	}
}

func TestOAuthNeedsAppURL(t *testing.T) {
	a := newTestApp(t)
	cfg, _ := testConfig(t)
	cfg.AppURL = ""
	s := New(cfg, a.store, testKeys(t, cfg))
	for _, p := range []string{"/.well-known/oauth-authorization-server", "/.well-known/oauth-protected-resource"} {
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s without app_url = %d", p, rec.Code)
		}
	}
}

func TestOAuthRegisterRejectsBadRedirects(t *testing.T) {
	a := newTestApp(t)
	a.s.oauth.OpenRegistration(oauth.RegistrationWindow)
	for _, uris := range [][]string{{}, {"http://evil.example/cb"}, {"javascript:alert(1)"}} {
		raw, _ := json.Marshal(map[string]any{"client_name": "X", "redirect_uris": uris})
		res, _ := a.client.Post(a.srv.URL+"/oauth/register", "application/json", bytes.NewReader(raw))
		body := decodeJSON(t, res)
		if res.StatusCode != http.StatusBadRequest || body["error"] != "invalid_redirect_uri" {
			t.Errorf("%v -> %d %v", uris, res.StatusCode, body)
		}
	}
}

func TestOAuthAuthorizeRedirectsToConsentOnlyForRegisteredRedirects(t *testing.T) {
	a := newTestApp(t)
	id := a.registerClient()
	noFollow := *a.client
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	q := url.Values{"response_type": {"code"}, "client_id": {id}, "redirect_uri": {oauthRedirect}, "state": {"st-1"},
		"code_challenge": {oauthChallenge()}, "code_challenge_method": {"S256"}}
	res, err := noFollow.Get(a.srv.URL + "/oauth/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := url.Parse(res.Header.Get("Location"))
	if res.StatusCode != http.StatusFound || loc.Path != "/oauth/consent" || loc.Query().Get("client_id") != id {
		t.Fatalf("authorize = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// A redirect URI that was never registered must never be redirected to.
	q.Set("redirect_uri", "https://evil.example/cb")
	res, _ = noFollow.Get(a.srv.URL + "/oauth/authorize?" + q.Encode())
	if res.StatusCode != http.StatusBadRequest || res.Header.Get("Location") != "" {
		t.Fatalf("unregistered redirect = %d %q", res.StatusCode, res.Header.Get("Location"))
	}
	q.Set("redirect_uri", oauthRedirect)
	q.Set("client_id", "unknown")
	res, _ = noFollow.Get(a.srv.URL + "/oauth/authorize?" + q.Encode())
	if res.StatusCode != http.StatusBadRequest || res.Header.Get("Location") != "" {
		t.Fatalf("unknown client = %d", res.StatusCode)
	}
}

func TestOAuthAuthorizeSendsProtocolErrorsToTheClient(t *testing.T) {
	a := newTestApp(t)
	id := a.registerClient()
	noFollow := *a.client
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	q := url.Values{"response_type": {"token"}, "client_id": {id}, "redirect_uri": {oauthRedirect}, "state": {"st-1"},
		"code_challenge": {oauthChallenge()}, "code_challenge_method": {"S256"}}
	res, _ := noFollow.Get(a.srv.URL + "/oauth/authorize?" + q.Encode())
	loc, _ := url.Parse(res.Header.Get("Location"))
	if res.StatusCode != http.StatusFound || !strings.HasPrefix(loc.String(), oauthRedirect) ||
		loc.Query().Get("error") != "unsupported_response_type" || loc.Query().Get("state") != "st-1" {
		t.Fatalf("implicit grant = %d %s", res.StatusCode, loc)
	}
}

func TestOAuthConsentNeedsASessionNotAToken(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("oauth@a.com", "secret1", roleEditor)
	id := a.registerClient()
	body := map[string]any{"client_id": id, "redirect_uri": oauthRedirect, "code_challenge": oauthChallenge(), "code_challenge_method": "S256", "approve": true}

	status(t, a.do("POST", "/api/oauth/authorize", body, "", ""), http.StatusUnauthorized, "anonymous")

	sess := a.issue(u, false)
	status(t, a.do("POST", "/api/oauth/authorize", body, sess.Token, ""), 419, "no CSRF")

	raw, _ := a.issueToken(u, "read-write", nil)
	status(t, a.bearer("POST", "/api/oauth/authorize", body, raw), http.StatusForbidden, "an API token must not mint tokens")
}

func TestOAuthConsentShowsTheClientName(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("oauth@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	id := a.registerClient()
	res := a.do("GET", "/api/oauth/authorize?client_id="+id+"&redirect_uri="+url.QueryEscape(oauthRedirect), nil, sess.Token, sess.CSRF)
	data, _ := decodeJSON(t, res)["data"].(map[string]any)
	if res.StatusCode != 200 || data["client_name"] != "Claude" {
		t.Fatalf("consent info = %d %v", res.StatusCode, data)
	}
	res = a.do("GET", "/api/oauth/authorize?client_id="+id+"&redirect_uri="+url.QueryEscape("https://evil.example/cb"), nil, sess.Token, sess.CSRF)
	status(t, res, http.StatusBadRequest, "unregistered redirect")
}

func TestOAuthFullFlowGivesAReadOnlyToken(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("oauth@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	id := a.registerClient()

	code := codeFrom(t, a.approve(sess.Token, sess.CSRF, id, true))
	code2, tok := a.token(url.Values{"grant_type": {"authorization_code"}, "client_id": {id}, "code": {code}, "code_verifier": {oauthVerifier}, "redirect_uri": {oauthRedirect}})
	if code2 != 200 || tok["token_type"] != "Bearer" || tok["access_token"] == "" || tok["refresh_token"] == "" {
		t.Fatalf("token = %d %v", code2, tok)
	}

	// The access token is a normal read-only API token: it reads, it cannot write.
	access := tok["access_token"].(string)
	res := a.bearer("GET", "/api/accounts", nil, access)
	if res.StatusCode != 200 {
		t.Fatalf("read with access token = %d", res.StatusCode)
	}
	res = a.bearer("POST", "/api/accounts", map[string]any{"name": "X"}, access)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("write with access token = %d, want 403", res.StatusCode)
	}

	// Refresh rotates; the old pair stops working.
	status2, next := a.token(url.Values{"grant_type": {"refresh_token"}, "client_id": {id}, "refresh_token": {tok["refresh_token"].(string)}})
	if status2 != 200 || next["access_token"] == access {
		t.Fatalf("refresh = %d %v", status2, next)
	}
	if res := a.bearer("GET", "/api/accounts", nil, access); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old access token = %d, want 401", res.StatusCode)
	}
	status3, errBody := a.token(url.Values{"grant_type": {"refresh_token"}, "client_id": {id}, "refresh_token": {tok["refresh_token"].(string)}})
	if status3 != http.StatusBadRequest || errBody["error"] != "invalid_grant" {
		t.Fatalf("replayed refresh = %d %v", status3, errBody)
	}
}

func TestOAuthDenyReturnsAccessDenied(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("oauth@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	id := a.registerClient()
	u2, err := url.Parse(a.approve(sess.Token, sess.CSRF, id, false))
	if err != nil || u2.Query().Get("error") != "access_denied" || u2.Query().Get("state") != "st-1" || u2.Query().Get("code") != "" {
		t.Fatalf("deny = %v %v", u2, err)
	}
}

func TestOAuthTokenErrors(t *testing.T) {
	a := newTestApp(t)
	id := a.registerClient()
	cases := map[string]url.Values{
		"unsupported_grant_type": {"grant_type": {"password"}, "client_id": {id}},
		"invalid_client":         {"grant_type": {"authorization_code"}, "client_id": {"unknown"}, "code": {"x"}},
		"invalid_grant":          {"grant_type": {"authorization_code"}, "client_id": {id}, "code": {"x"}, "code_verifier": {oauthVerifier}, "redirect_uri": {oauthRedirect}},
	}
	for want, form := range cases {
		_, body := a.token(form)
		if body["error"] != want {
			t.Errorf("%s: got %v", want, body)
		}
	}
}

func TestOAuthRegistrationIsClosedByDefault(t *testing.T) {
	a := newTestApp(t)
	raw, _ := json.Marshal(map[string]any{"client_name": "Claude", "redirect_uris": []string{oauthRedirect}})
	res, _ := a.client.Post(a.srv.URL+"/oauth/register", "application/json", bytes.NewReader(raw))
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusNotFound || body["error"] != nil || strings.Contains(strings.ToLower(body["message"].(string)), "admin") {
		t.Fatalf("closed registration must look absent: %d %v", res.StatusCode, body)
	}
}

func TestAdminOpensRegistrationForOneApp(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@a.com", "secret1", "admin")
	sess := a.issue(admin, false)
	state := func(method string) map[string]any {
		res := a.do(method, "/api/admin/oauth/registration", nil, sess.Token, sess.CSRF)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s registration = %d", method, res.StatusCode)
		}
		return decodeJSON(t, res)["data"].(map[string]any)
	}
	if d := state("GET"); d["open"] != false || d["expires_at"] != nil {
		t.Fatalf("initial = %v", d)
	}
	d := state("POST")
	if d["open"] != true || d["expires_at"] == nil {
		t.Fatalf("after open = %v", d)
	}
	until, err := time.Parse(time.RFC3339Nano, d["expires_at"].(string))
	if err != nil || time.Until(until) > 5*time.Minute || time.Until(until) < 4*time.Minute {
		t.Fatalf("expires_at = %v (%v)", d["expires_at"], err)
	}

	a.registerClientNoOpen()
	if d := state("GET"); d["open"] != false {
		t.Fatalf("registering must close the window: %v", d)
	}

	state("POST")
	if d := state("DELETE"); d["open"] != false {
		t.Fatalf("after close = %v", d)
	}
}

func TestOnlyAdminSessionsControlRegistration(t *testing.T) {
	a := newTestApp(t)
	user := a.createUser("u@a.com", "secret1", roleEditor)
	sess := a.issue(user, false)
	status(t, a.do("POST", "/api/admin/oauth/registration", nil, sess.Token, sess.CSRF), http.StatusForbidden, "a normal user")
	status(t, a.do("POST", "/api/admin/oauth/registration", nil, "", ""), http.StatusUnauthorized, "anonymous")

	admin := a.createUser("admin@a.com", "secret1", "admin")
	raw, _ := a.issueToken(admin, "read-write", nil)
	status(t, a.bearer("POST", "/api/admin/oauth/registration", nil, raw), http.StatusForbidden, "an admin's API token")
	if _, open := a.s.oauth.RegistrationOpenUntil(); open {
		t.Fatal("a refused request opened registration")
	}
}

func TestAdminSeesWhereTheRegistrationCameFrom(t *testing.T) {
	a := newTestApp(t)
	admin := a.issue(a.createUser("admin@a.com", "secret1", "admin"), false)
	a.s.oauth.OpenRegistration(oauth.RegistrationWindow)
	raw, _ := json.Marshal(map[string]any{"client_name": "Claude", "redirect_uris": []string{oauthRedirect}})
	req, _ := http.NewRequest("POST", a.srv.URL+"/oauth/register", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Claude-MCP/9.9")
	req.Header.Set("Origin", "https://claude.ai")
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	res, err := a.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	status(t, res, http.StatusCreated, "register")

	list := decodeJSON(t, a.do("GET", "/api/admin/oauth/clients", nil, admin.Token, admin.CSRF))["data"].([]any)
	if len(list) != 1 {
		t.Fatalf("clients = %v", list)
	}
	c := list[0].(map[string]any)
	if c["name"] != "Claude" || c["ip"] != "203.0.113.7" || c["user_agent"] != "Claude-MCP/9.9" || c["origin"] != "https://claude.ai" ||
		c["approved_at"] != nil || c["registered_at"] == nil || c["redirect_uris"].([]any)[0] != oauthRedirect {
		t.Fatalf("client = %v", c)
	}
}

func TestRemovingAnAppCutsItOffAtOnce(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("u@a.com", "secret1", roleEditor), false)
	admin := a.issue(a.createUser("admin@a.com", "secret1", "admin"), false)
	id, access := a.connect(user)
	if res := a.bearer("GET", "/api/spaces", nil, access); res.StatusCode != http.StatusOK {
		t.Fatalf("read = %d", res.StatusCode)
	}
	status(t, a.do("DELETE", "/api/admin/oauth/clients/"+id, nil, admin.Token, admin.CSRF), http.StatusNoContent, "remove")
	if res := a.bearer("GET", "/api/spaces", nil, access); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("read after removal = %d, want 401", res.StatusCode)
	}
	status(t, a.do("DELETE", "/api/admin/oauth/clients/"+id, nil, admin.Token, admin.CSRF), http.StatusNotFound, "remove twice")
}

func TestOnlyAdminSessionsListAndRemoveApps(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("u@a.com", "secret1", roleEditor), false)
	id, access := a.connect(user)
	status(t, a.do("GET", "/api/admin/oauth/clients", nil, user.Token, user.CSRF), http.StatusForbidden, "list as a normal user")
	status(t, a.do("DELETE", "/api/admin/oauth/clients/"+id, nil, user.Token, user.CSRF), http.StatusForbidden, "remove as a normal user")
	// A token, even an admin's, cannot manage apps; neither can the app itself.
	adminUser := a.createUser("admin@a.com", "secret1", "admin")
	raw, _ := a.issueToken(adminUser, "read-write", nil)
	status(t, a.bearer("GET", "/api/admin/oauth/clients", nil, raw), http.StatusForbidden, "list with a token")
	status(t, a.bearer("DELETE", "/api/admin/oauth/clients/"+id, nil, access), http.StatusForbidden, "remove with its own token")
}

func TestNoPageCanBeFramed(t *testing.T) {
	a := newTestApp(t)
	for _, path := range []string{"/oauth/consent", "/oauth/authorize", "/api/auth/status", "/livez", "/"} {
		res, err := a.client.Get(a.srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if got := res.Header.Get("X-Frame-Options"); got != "DENY" {
			t.Errorf("%s: X-Frame-Options = %q", path, got)
		}
		if got := res.Header.Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors 'none'") {
			t.Errorf("%s: Content-Security-Policy = %q", path, got)
		}
	}
}

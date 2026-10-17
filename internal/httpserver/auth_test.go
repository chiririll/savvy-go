package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
	"savvy-go/internal/spacefill"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
)

// Test roles for createUser: admin is a server admin and admin of the shared
// test space; editor and viewer are plain users with that role in it.
const (
	roleEditor = spaces.RoleEditor
	roleViewer = spaces.RoleViewer
)

type testApp struct {
	t      *testing.T
	db     *sql.DB // the server database
	store  *sqlite.Store
	space  *spaces.Space // the space every createUser member belongs to
	srv    *httptest.Server
	s      *Server
	svc    services
	client *http.Client

	bootstrap *bootstrap.Guard // the first-run password of the server
}

// openTestStore opens a store in cfg.DataDir with the embedded migrations.
func openTestStore(t *testing.T, cfg appconfig.Config) *sqlite.Store {
	t.Helper()
	st, err := sqlite.OpenApp(context.Background(), cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// testKeys is a signing key in the test's data directory.
func testKeys(t *testing.T, cfg appconfig.Config) *signing.Holder {
	t.Helper()
	key, err := signing.Load(cfg.DataDir, false)
	if err != nil {
		t.Fatal(err)
	}
	return signing.NewHolder(key)
}

func newTestApp(t *testing.T, opts ...Option) *testApp {
	t.Helper()
	cfg, _ := testConfig(t)
	st := openTestStore(t, cfg)
	guard := bootstrap.New(cfg.DataDir, bootstrap.Binary)
	opts = append([]Option{WithRates(currencydata.NewOffline()), WithBootstrap(guard)}, opts...)
	s := newServer(cfg, st, testKeys(t, cfg), opts...)
	space, err := servicesOf(s).Spaces.Create(context.Background(), "Test", nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return &testApp{t: t, db: st.SQL(), store: st, space: space, srv: ts, s: s, svc: servicesOf(s), client: ts.Client(), bootstrap: guard}
}

// spaceDB is the database of the shared test space.
func (a *testApp) spaceDB() store.DB {
	a.t.Helper()
	d, err := a.store.Space(context.Background(), a.space.ID)
	if err != nil {
		a.t.Fatal(err)
	}
	return d
}

// createUser creates a user. role is a server role (admin, user, guest) or,
// for editor and viewer, a plain user with that role in the shared space; an
// admin also administers the shared space.
func (a *testApp) createUser(email, password, role string, opts ...func(*auth.User)) *auth.User {
	a.t.Helper()
	serverRole, spaceRole := role, ""
	switch role {
	case auth.RoleAdmin:
		spaceRole = spaces.RoleAdmin
	case roleEditor, roleViewer:
		serverRole, spaceRole = auth.RoleUser, role
	}
	u, err := a.svc.Users.Create(context.Background(), "U", email, &password, serverRole)
	if err != nil {
		a.t.Fatal(err)
	}
	if spaceRole != "" {
		if err := a.svc.Spaces.SetMember(context.Background(), a.space.ID, u, spaceRole); err != nil {
			a.t.Fatal(err)
		}
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

func (a *testApp) issue(u *auth.User, remember bool) *auth.Issued {
	a.t.Helper()
	issued, err := a.svc.Sessions.Issue(context.Background(), u, "127.0.0.1", "test", remember)
	if err != nil {
		a.t.Fatal(err)
	}
	return issued
}

// dataRoots are the first path segments of the space data routes.
func dataRoots(s *Server) map[string]bool {
	r := chi.NewRouter()
	routing.SpaceWriteRoutes(r, s.deps())
	roots := map[string]bool{}
	_ = chi.Walk(r, func(_, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		roots[strings.SplitN(strings.TrimPrefix(route, "/"), "/", 2)[0]] = true
		return nil
	})
	return roots
}

// apiPath sends data requests written as /api/<data> to the shared test
// space, /api/spaces/{id}/<data>.
func (a *testApp) apiPath(path string) string {
	if a.space == nil {
		return path
	}
	return spacedPath(a.s, a.space.ID, path)
}

func spacedPath(s *Server, spaceID int64, path string) string {
	rest, ok := strings.CutPrefix(path, "/api/")
	if !ok {
		return path
	}
	root, _, _ := strings.Cut(rest, "/")
	root, _, _ = strings.Cut(root, "?")
	if dataRoots(s)[root] {
		return "/api/spaces/" + strconv.FormatInt(spaceID, 10) + "/" + rest
	}
	return path
}

func (a *testApp) request(method, path string, body any, token, csrf string) *http.Request {
	a.t.Helper()
	path = a.apiPath(path)
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			a.t.Fatal(err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, a.srv.URL+path, rdr)
	if err != nil {
		a.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "svy_session", Value: token})
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	return req
}

func (a *testApp) do(method, path string, body any, token, csrf string) *http.Response {
	a.t.Helper()
	res, err := a.client.Do(a.request(method, path, body, token, csrf))
	if err != nil {
		a.t.Fatal(err)
	}
	return res
}

func decodeJSON(t *testing.T, res *http.Response) map[string]any {
	t.Helper()
	defer res.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	return out
}

// toString renders a decoded JSON value as text, for asserting that a response
// mentions some phrase. JSON escaping applies, so keep phrases free of quotes,
// angle brackets and ampersands.
func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func cookieNamed(res *http.Response, name string) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestSessionSmokeRegisterMeLogout(t *testing.T) {
	a := newTestApp(t)
	res := a.register(a.bootstrap.Password(), nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register %d %v", res.StatusCode, decodeJSON(t, res))
	}
	body := decodeJSON(t, res)
	user, _ := body["user"].(map[string]any)
	if user["email"] != "a@a.com" {
		t.Fatalf("user %v", body)
	}
	session := cookieNamed(res, "svy_session")
	csrf := cookieNamed(res, "svy_csrf")
	if session == nil || !session.HttpOnly {
		t.Fatalf("session cookie %+v", session)
	}
	if csrf == nil || csrf.HttpOnly {
		t.Fatalf("csrf cookie %+v", csrf)
	}
	var n int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM auth_sessions`).Scan(&n)
	if n != 1 {
		t.Fatalf("sessions %d", n)
	}

	me := a.do("GET", "/api/auth/me", nil, session.Value, "")
	if me.StatusCode != 200 {
		t.Fatalf("me %d", me.StatusCode)
	}
	me.Body.Close()

	bare := a.do("POST", "/api/auth/logout", map[string]any{}, session.Value, "")
	if bare.StatusCode != 419 {
		t.Fatalf("logout without csrf %d", bare.StatusCode)
	}
	bare.Body.Close()

	ok := a.do("POST", "/api/auth/logout", map[string]any{}, session.Value, csrf.Value)
	if ok.StatusCode != 200 {
		t.Fatalf("logout %d", ok.StatusCode)
	}
	ok.Body.Close()

	st := a.do("GET", "/api/auth/2fa/status", nil, session.Value, "")
	if st.StatusCode != 401 {
		t.Fatalf("protected after logout %d", st.StatusCode)
	}
	st.Body.Close()
}

func TestHostPrefixedCookieOverTLS(t *testing.T) {
	cfg, _ := testConfig(t)
	guard := bootstrap.New(cfg.DataDir, bootstrap.Binary)
	s := New(cfg.WithAppURL("https://savvy.test"), openTestStore(t, cfg), testKeys(t, cfg), WithBootstrap(guard))
	ts := httptest.NewTLSServer(s.Handler())
	t.Cleanup(ts.Close)
	client := ts.Client()

	raw, _ := json.Marshal(map[string]string{"name": "A", "email": "a@a.com", "password": "secret12", "bootstrap_password": guard.Password()})
	req, _ := http.NewRequest("POST", ts.URL+"/api/auth/register", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 201 {
		t.Fatalf("status %d", res.StatusCode)
	}
	host := cookieNamed(res, "__Host-svy_session")
	if host == nil || !host.Secure || host.Path != "/" {
		t.Fatalf("host cookie %+v", host)
	}

	me, err := http.NewRequest("GET", ts.URL+"/api/auth/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	me.AddCookie(&http.Cookie{Name: "__Host-svy_session", Value: host.Value})
	meRes, err := client.Do(me)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeJSON(t, meRes)
	user, _ := body["user"].(map[string]any)
	if user["email"] != "a@a.com" {
		t.Fatalf("me %v", body)
	}
}

func TestSessionIdleAbsoluteRevokeAndRefresh(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", roleEditor)
	issued := a.issue(u, false)

	_, _ = a.db.Exec(`UPDATE auth_sessions SET idle_expires_at = ? WHERE id = ?`,
		time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano), issued.Session.ID)
	res := a.do("GET", "/api/auth/2fa/status", nil, issued.Token, "")
	if res.StatusCode != 401 {
		t.Fatalf("idle %d", res.StatusCode)
	}
	res.Body.Close()

	issued = a.issue(u, false)
	_, _ = a.db.Exec(`UPDATE auth_sessions SET absolute_expires_at = ? WHERE id = ?`,
		time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano), issued.Session.ID)
	res = a.do("GET", "/api/auth/2fa/status", nil, issued.Token, "")
	if res.StatusCode != 401 {
		t.Fatalf("absolute %d", res.StatusCode)
	}
	res.Body.Close()

	issued = a.issue(u, false)
	_ = a.svc.Sessions.Revoke(context.Background(), issued.Session)
	res = a.do("GET", "/api/auth/2fa/status", nil, issued.Token, "")
	if res.StatusCode != 401 {
		t.Fatalf("revoked %d", res.StatusCode)
	}
	res.Body.Close()

	issued = a.issue(u, false)
	_, _ = a.db.Exec(`UPDATE auth_sessions SET idle_expires_at = ? WHERE id = ?`,
		time.Now().Add(5*time.Minute).UTC().Format(time.RFC3339Nano), issued.Session.ID)
	res = a.do("GET", "/api/auth/2fa/status", nil, issued.Token, "")
	if res.StatusCode != 200 {
		t.Fatalf("touch %d", res.StatusCode)
	}
	res.Body.Close()
	var idle string
	_ = a.db.QueryRow(`SELECT idle_expires_at FROM auth_sessions WHERE id = ?`, issued.Session.ID).Scan(&idle)
	parsed, _ := time.Parse(time.RFC3339Nano, idle)
	if !parsed.After(time.Now().Add(60 * time.Minute)) {
		t.Fatalf("idle not slid: %s", idle)
	}
}

func TestRememberMeRotation(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", roleEditor)
	issued := a.issue(u, true)
	oldHash := issued.Session.TokenHash
	_, _ = a.db.Exec(`UPDATE auth_sessions SET refreshed_at = ? WHERE id = ?`,
		time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339Nano), issued.Session.ID)
	res := a.do("GET", "/api/auth/2fa/status", nil, issued.Token, "")
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	if cookieNamed(res, "svy_session") != nil {
		t.Fatal("should not rotate on ordinary API use")
	}
	res.Body.Close()
	var hash string
	_ = a.db.QueryRow(`SELECT token_hash FROM auth_sessions WHERE id = ?`, issued.Session.ID).Scan(&hash)
	if hash != oldHash {
		t.Fatal("hash rotated on status")
	}

	_, _ = a.db.Exec(`UPDATE auth_sessions SET refreshed_at = ? WHERE id = ?`,
		time.Now().Add(-24*time.Hour-time.Minute).UTC().Format(time.RFC3339Nano), issued.Session.ID)
	me := a.do("GET", "/api/auth/me", nil, issued.Token, "")
	body := decodeJSON(t, me)
	if body["refresh_at"] == nil {
		t.Fatalf("refresh_at %v", body)
	}
	if cookieNamed(me, "svy_session") == nil {
		t.Fatal("expected rotated cookie")
	}
	_ = a.db.QueryRow(`SELECT token_hash FROM auth_sessions WHERE id = ?`, issued.Session.ID).Scan(&hash)
	if hash == oldHash {
		t.Fatal("expected new hash")
	}

	issued = a.issue(u, true)
	me = a.do("GET", "/api/auth/me", nil, issued.Token, "")
	if cookieNamed(me, "svy_session") != nil {
		t.Fatal("should not refresh before a day")
	}
	me.Body.Close()

	issued = a.issue(u, false)
	_, _ = a.db.Exec(`UPDATE auth_sessions SET refreshed_at = ? WHERE id = ?`,
		time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339Nano), issued.Session.ID)
	me = a.do("GET", "/api/auth/me", nil, issued.Token, "")
	body = decodeJSON(t, me)
	if cookieNamed(me, "svy_session") != nil {
		t.Fatal("browser session should not rotate")
	}
	if body["refresh_at"] != nil {
		t.Fatalf("refresh_at %v", body["refresh_at"])
	}
}

func TestLoginCookiePersistence(t *testing.T) {
	a := newTestApp(t)
	a.createUser("u@test.com", "secret1", roleEditor)
	sessionLogin := a.do("POST", "/api/auth/login", map[string]any{
		"email": "u@test.com", "password": "secret1", "remember_me": false,
	}, "", "")
	c := cookieNamed(sessionLogin, "svy_session")
	sessionLogin.Body.Close()
	if c == nil || (!c.Expires.IsZero() && c.MaxAge > 0) {
		t.Fatalf("session cookie should be session-lived %+v", c)
	}

	remember := a.do("POST", "/api/auth/login", map[string]any{
		"email": "u@test.com", "password": "secret1", "remember_me": true,
	}, "", "")
	rc := cookieNamed(remember, "svy_session")
	remember.Body.Close()
	if rc == nil || rc.Expires.Before(time.Now().Add(6*24*time.Hour)) {
		t.Fatalf("remember cookie %+v", rc)
	}
}

func TestLoginSSOOnlyAnd2FAChallenge(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("sso@test.com", "secret1", roleEditor)
	_, _ = a.db.Exec(`UPDATE users SET is_sso_only = 1 WHERE id = ?`, u.ID)
	res := a.do("POST", "/api/auth/login", map[string]string{"email": "sso@test.com", "password": "secret1"}, "", "")
	if res.StatusCode != 422 {
		t.Fatalf("sso-only %d", res.StatusCode)
	}
	res.Body.Close()

	u2 := a.createUser("u@test.com", "secret1", roleEditor)
	_, _ = a.db.Exec(`UPDATE users SET two_factor_enabled = 1, two_factor_confirmed = 1, two_factor_secret = 'SECRET' WHERE id = ?`, u2.ID)
	res = a.do("POST", "/api/auth/login", map[string]string{"email": "u@test.com", "password": "secret1"}, "", "")
	body := decodeJSON(t, res)
	if body["requires_2fa"] != true || body["two_factor_token"] == nil {
		t.Fatalf("2fa %v", body)
	}
	if cookieNamed(res, "svy_session") != nil {
		t.Fatal("no session cookie during 2fa")
	}
	n, _ := a.svc.Challenges.Count(context.Background())
	if n != 1 {
		t.Fatalf("challenges %d", n)
	}
}

func TestTwoFactorChallengePeekAndConsume(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", roleEditor)
	tok, err := a.svc.Challenges.Issue(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.svc.Challenges.Resolve(context.Background(), tok)
	if err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("resolve %v %v", got, err)
	}
	got, _ = a.svc.Challenges.Resolve(context.Background(), tok)
	if got == nil {
		t.Fatal("peek should not consume")
	}
	got, _ = a.svc.Challenges.Consume(context.Background(), tok)
	if got == nil {
		t.Fatal("first consume")
	}
	got, _ = a.svc.Challenges.Consume(context.Background(), tok)
	if got != nil {
		t.Fatal("second consume")
	}

	tok, _ = a.svc.Challenges.Issue(context.Background(), u)
	_, _ = a.db.Exec(`UPDATE two_factor_challenges SET expires_at = ?`, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano))
	got, _ = a.svc.Challenges.Resolve(context.Background(), tok)
	if got != nil {
		t.Fatal("expired challenge")
	}
}

func TestPasswordLoginToggle(t *testing.T) {
	a := newTestApp(t)
	a.createUser("u@test.com", "secret1", roleEditor)
	insertSSO(t, a.db, true)
	_ = settings.Store{DB: a.db}.Set(context.Background(), "password_login_enabled", false)

	res := a.do("POST", "/api/auth/login", map[string]string{"email": "u@test.com", "password": "secret1"}, "", "")
	if res.StatusCode != 422 {
		t.Fatalf("blocked %d", res.StatusCode)
	}
	res.Body.Close()

	_, _ = a.db.Exec(`DELETE FROM identity_providers`)
	res = a.do("POST", "/api/auth/login", map[string]string{"email": "u@test.com", "password": "secret1"}, "", "")
	if res.StatusCode != 200 {
		t.Fatalf("self-heal no provider %d", res.StatusCode)
	}
	res.Body.Close()

	insertSSO(t, a.db, false)
	res = a.do("POST", "/api/auth/login", map[string]string{"email": "u@test.com", "password": "secret1"}, "", "")
	if res.StatusCode != 200 {
		t.Fatalf("self-heal disabled provider %d", res.StatusCode)
	}
	res.Body.Close()

	_, _ = a.db.Exec(`UPDATE identity_providers SET enabled = 1`)
	st := a.do("GET", "/api/auth/status", nil, "", "")
	body := decodeJSON(t, st)
	if body["password_login_enabled"] != false {
		t.Fatalf("status disabled %v", body)
	}
	_ = settings.Store{DB: a.db}.Set(context.Background(), "password_login_enabled", true)
	st = a.do("GET", "/api/auth/status", nil, "", "")
	body = decodeJSON(t, st)
	if body["password_login_enabled"] != true {
		t.Fatalf("status enabled %v", body)
	}

	admin, _ := a.svc.Users.Create(context.Background(), "Ad", "ad@test.com", ptr("secret1"), auth.RoleAdmin)
	iss := a.issue(admin, false)
	_, _ = a.db.Exec(`DELETE FROM identity_providers`)
	res = a.do("PATCH", "/api/settings", map[string]any{"password_login_enabled": false}, iss.Token, iss.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != 422 || body["error"] != "sso_required" {
		t.Fatalf("refuse disable %d %v", res.StatusCode, body)
	}
	insertSSO(t, a.db, true)
	res = a.do("PATCH", "/api/settings", map[string]any{"password_login_enabled": false}, iss.Token, iss.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("allow disable %d %v", res.StatusCode, decodeJSON(t, res))
	} else {
		res.Body.Close()
	}
	store := settings.Store{DB: a.db}
	if store.Bool(context.Background(), "password_login_enabled", true) {
		t.Fatal("expected disabled")
	}
}

func insertSSO(t *testing.T, sqlDB *sql.DB, enabled bool) {
	t.Helper()
	en := 0
	if enabled {
		en = 1
	}
	_, err := sqlDB.Exec(`
		INSERT INTO identity_providers (name, slug, protocol, preset, enabled, config, default_role)
		VALUES ('IdP', 'idp', 'oidc', 'custom_oidc', ?, '{"discovery_url":"https://idp.test","client_id":"c"}', 'read-only')`, en)
	if err != nil {
		t.Fatal(err)
	}
}

func ptr(s string) *string { return &s }

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

// spaceCreator creates spaces with the server's own services, as the create-space route does.
func (a *testApp) spaceCreator() spacefill.Creator {
	return spacefill.Creator{Spaces: a.svc.Spaces, Store: a.svc.Store, Rates: a.svc.Rates}
}

// An unknown email, an SSO-only account and a wrong password must be
// indistinguishable by status and body (the bcrypt work is covered in the
// session package).
func TestLoginAnswersTheSameForUnknownAndWrong(t *testing.T) {
	a := newTestApp(t)
	a.createUser("real@test.com", "secret1", roleEditor)
	sso := a.createUser("sso@test.com", "secret1", roleEditor)
	_, _ = a.db.Exec(`UPDATE users SET is_sso_only = 1 WHERE id = ?`, sso.ID)
	var first map[string]any
	for _, email := range []string{"real@test.com", "nobody@test.com", "sso@test.com"} {
		res := a.do("POST", "/api/auth/login", map[string]string{"email": email, "password": "wrong-pass"}, "", "")
		if res.StatusCode != 422 {
			t.Fatalf("%s: %d", email, res.StatusCode)
		}
		body := decodeJSON(t, res)
		if first == nil {
			first = body
		} else if fmt.Sprint(body) != fmt.Sprint(first) {
			t.Errorf("%s: body %v differs from %v", email, body, first)
		}
	}
}

func TestPlainCookieDoesNotAuthenticateOverHTTPS(t *testing.T) {
	cfg, _ := testConfig(t)
	s := New(cfg.WithAppURL("https://savvy.test"), openTestStore(t, cfg), testKeys(t, cfg))
	svc := servicesOf(s)
	pw := "secret12"
	u, err := svc.Users.Create(context.Background(), "U", "u@test.com", &pw, "")
	if err != nil {
		t.Fatal(err)
	}
	issued, err := svc.Sessions.Issue(context.Background(), u, "127.0.0.1:1", "test", false)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]int{"svy_session": 401, "__Host-svy_session": 200} {
		req := httptest.NewRequest("GET", "/api/auth/2fa/status", nil)
		req.AddCookie(&http.Cookie{Name: name, Value: issued.Token})
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("%s: %d, want %d", name, rec.Code, want)
		}
	}
}

func TestForwardedProtoDoesNotMakeCookiesSecure(t *testing.T) {
	a := newTestApp(t) // app_url is http://
	a.createUser("u@test.com", "secret1", roleEditor)
	raw, _ := json.Marshal(map[string]string{"email": "u@test.com", "password": "secret1"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	a.s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("login %d", rec.Code)
	}
	res := rec.Result()
	if c := cookieNamed(res, "svy_session"); c == nil || c.Secure {
		t.Errorf("session cookie %+v, want plain and not Secure", c)
	}
	if cookieNamed(res, "__Host-svy_session") != nil {
		t.Error("host-prefixed cookie set on an http:// instance")
	}
}

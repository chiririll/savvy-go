package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
	"savvy-go/internal/settings"
)

type testApp struct {
	t      *testing.T
	db     *sql.DB
	srv    *httptest.Server
	s      *Server
	client *http.Client
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	cfg, _ := testConfig(t)
	sqlDB, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}
	s := New(cfg, sqlDB)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(func() {
		ts.Close()
		_ = sqlDB.Close()
	})
	return &testApp{t: t, db: sqlDB, srv: ts, s: s, client: ts.Client()}
}

func (a *testApp) createUser(email, password, role string, opts ...func(*auth.User)) *auth.User {
	a.t.Helper()
	u, err := a.s.users.Create(context.Background(), "U", email, &password, role)
	if err != nil {
		a.t.Fatal(err)
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

func (a *testApp) issue(u *auth.User, remember bool) *auth.Issued {
	a.t.Helper()
	issued, err := a.s.sessions.Issue(context.Background(), u, "127.0.0.1", "test", remember)
	if err != nil {
		a.t.Fatal(err)
	}
	return issued
}

func (a *testApp) do(method, path string, body any, token, csrf string) *http.Response {
	a.t.Helper()
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
	res, err := a.client.Do(req)
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
	res := a.do("POST", "/api/auth/register", map[string]string{
		"name": "A", "email": "a@a.com", "password": "secret1",
	}, "", "")
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
	sqlDB, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := migrate.Up(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}
	s := New(cfg, sqlDB)
	ts := httptest.NewTLSServer(s.Handler())
	t.Cleanup(ts.Close)
	client := ts.Client()

	raw, _ := json.Marshal(map[string]string{"name": "A", "email": "a@a.com", "password": "secret1"})
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
	u := a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
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
	_ = a.s.sessions.Revoke(context.Background(), issued.Session)
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
	u := a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
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
	a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
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
	u := a.createUser("sso@test.com", "secret1", auth.RoleReadWrite)
	_, _ = a.db.Exec(`UPDATE users SET is_sso_only = 1 WHERE id = ?`, u.ID)
	res := a.do("POST", "/api/auth/login", map[string]string{"email": "sso@test.com", "password": "secret1"}, "", "")
	if res.StatusCode != 422 {
		t.Fatalf("sso-only %d", res.StatusCode)
	}
	res.Body.Close()

	u2 := a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
	_, _ = a.db.Exec(`UPDATE users SET two_factor_enabled = 1, two_factor_confirmed = 1, two_factor_secret = 'SECRET' WHERE id = ?`, u2.ID)
	res = a.do("POST", "/api/auth/login", map[string]string{"email": "u@test.com", "password": "secret1"}, "", "")
	body := decodeJSON(t, res)
	if body["requires_2fa"] != true || body["two_factor_token"] == nil {
		t.Fatalf("2fa %v", body)
	}
	if cookieNamed(res, "svy_session") != nil {
		t.Fatal("no session cookie during 2fa")
	}
	n, _ := a.s.challenges.Count(context.Background())
	if n != 1 {
		t.Fatalf("challenges %d", n)
	}
}

func TestTwoFactorChallengePeekAndConsume(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
	tok, err := a.s.challenges.Issue(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.s.challenges.Resolve(context.Background(), tok)
	if err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("resolve %v %v", got, err)
	}
	got, _ = a.s.challenges.Resolve(context.Background(), tok)
	if got == nil {
		t.Fatal("peek should not consume")
	}
	got, _ = a.s.challenges.Consume(context.Background(), tok)
	if got == nil {
		t.Fatal("first consume")
	}
	got, _ = a.s.challenges.Consume(context.Background(), tok)
	if got != nil {
		t.Fatal("second consume")
	}

	tok, _ = a.s.challenges.Issue(context.Background(), u)
	_, _ = a.db.Exec(`UPDATE two_factor_challenges SET expires_at = ?`, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano))
	got, _ = a.s.challenges.Resolve(context.Background(), tok)
	if got != nil {
		t.Fatal("expired challenge")
	}
}

func TestChangePasswordAndLogoutOthers(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
	current := a.issue(u, false)
	other := a.issue(u, false)

	res := a.do("PUT", "/api/auth/password", map[string]string{
		"current_password": "secret1", "password": "newsecret1", "password_confirmation": "newsecret1",
	}, current.Token, current.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("change %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	ok := a.do("GET", "/api/auth/2fa/status", nil, current.Token, "")
	if ok.StatusCode != 200 {
		t.Fatalf("current %d", ok.StatusCode)
	}
	ok.Body.Close()
	denied := a.do("GET", "/api/auth/2fa/status", nil, other.Token, "")
	if denied.StatusCode != 401 {
		t.Fatalf("other %d", denied.StatusCode)
	}
	denied.Body.Close()

	u, _ = a.s.users.ByID(context.Background(), u.ID)
	if !auth.CheckPassword(*u.Password, "newsecret1") {
		t.Fatal("password not updated")
	}

	u = a.createUser("w@test.com", "secret1", auth.RoleReadWrite)
	current = a.issue(u, false)
	other = a.issue(u, false)
	res = a.do("PUT", "/api/auth/password", map[string]string{
		"current_password": "wrong-password", "password": "newsecret1", "password_confirmation": "newsecret1",
	}, current.Token, current.CSRF)
	if res.StatusCode != 422 {
		t.Fatalf("wrong current %d", res.StatusCode)
	}
	res.Body.Close()
	ok = a.do("GET", "/api/auth/2fa/status", nil, other.Token, "")
	if ok.StatusCode != 200 {
		t.Fatalf("other still valid %d", ok.StatusCode)
	}
	ok.Body.Close()

	sso := a.createUser("sso2@test.com", "secret1", auth.RoleReadWrite)
	_, _ = a.db.Exec(`UPDATE users SET is_sso_only = 1 WHERE id = ?`, sso.ID)
	sso, _ = a.s.users.ByID(context.Background(), sso.ID)
	iss := a.issue(sso, false)
	res = a.do("PUT", "/api/auth/password", map[string]string{
		"current_password": "secret1", "password": "newsecret1", "password_confirmation": "newsecret1",
	}, iss.Token, iss.CSRF)
	if res.StatusCode != 422 {
		t.Fatalf("sso password change %d", res.StatusCode)
	}
	res.Body.Close()

	u = a.createUser("lo@test.com", "secret1", auth.RoleReadWrite)
	current = a.issue(u, false)
	other = a.issue(u, false)
	res = a.do("POST", "/api/auth/logout-others", map[string]any{}, current.Token, current.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 200 || body["revoked"].(float64) != 1 {
		t.Fatalf("logout-others %d %v", res.StatusCode, body)
	}
	ok = a.do("GET", "/api/auth/2fa/status", nil, current.Token, "")
	ok.Body.Close()
	if ok.StatusCode != 200 {
		t.Fatalf("keep current %d", ok.StatusCode)
	}
	denied = a.do("GET", "/api/auth/2fa/status", nil, other.Token, "")
	denied.Body.Close()
	if denied.StatusCode != 401 {
		t.Fatalf("revoke other %d", denied.StatusCode)
	}
}

func TestPasswordTokensAndUsers(t *testing.T) {
	a := newTestApp(t)
	admin, err := a.s.users.Create(context.Background(), "Admin", "admin@test.com", ptr("secret1"), auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	sess := a.issue(admin, false)

	res := a.do("POST", "/api/users", map[string]any{
		"name": "Active", "email": "active@test.com", "password": "password1", "role": "read-only",
	}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	data := body["data"].(map[string]any)
	if data["email"] != "active@test.com" || data["isInactive"] != false || data["token"] != nil {
		t.Fatalf("active user %v", data)
	}

	res = a.do("POST", "/api/users", map[string]any{
		"name": "Invited", "email": "invited@test.com", "role": "read-write",
	}, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("invite %d %v", res.StatusCode, body)
	}
	data = body["data"].(map[string]any)
	if data["isInactive"] != true || data["token"] == nil {
		t.Fatalf("invite %v", data)
	}
	inviteTok := data["token"].(string)

	rw, _ := a.s.users.Create(context.Background(), "RW", "rw@test.com", ptr("secret1"), auth.RoleReadWrite)
	rws := a.issue(rw, false)
	res = a.do("POST", "/api/users", map[string]any{"name": "Nope", "email": "nope@test.com"}, rws.Token, rws.CSRF)
	if res.StatusCode != 403 {
		t.Fatalf("forbid %d", res.StatusCode)
	}
	res.Body.Close()

	_, _ = a.s.users.Create(context.Background(), "Pending", "pending@test.com", nil, auth.RoleReadOnly)
	res = a.do("POST", "/api/auth/login", map[string]string{"email": "pending@test.com", "password": "anything1"}, "", "")
	if res.StatusCode != 422 {
		t.Fatalf("inactive login %d", res.StatusCode)
	}
	res.Body.Close()

	res = a.do("GET", "/api/auth/password/"+inviteTok, nil, "", "")
	prev := decodeJSON(t, res)
	if res.StatusCode != 200 || prev["email"] != "invited@test.com" || prev["isInactive"] != true {
		t.Fatalf("preview %d %v", res.StatusCode, prev)
	}
	res = a.do("GET", "/api/auth/password/not-a-real-token", nil, "", "")
	if res.StatusCode != 404 {
		t.Fatalf("bad token %d", res.StatusCode)
	}
	res.Body.Close()

	res = a.do("POST", "/api/auth/password/"+inviteTok, map[string]string{
		"password": "newpass12", "password_confirmation": "newpass12",
	}, "", "")
	acc := decodeJSON(t, res)
	user := acc["user"].(map[string]any)
	if res.StatusCode != 200 || user["email"] != "invited@test.com" {
		t.Fatalf("accept %d %v", res.StatusCode, acc)
	}
	res = a.do("POST", "/api/auth/password/"+inviteTok, map[string]string{
		"password": "otherpass", "password_confirmation": "otherpass",
	}, "", "")
	if res.StatusCode != 404 {
		t.Fatalf("reuse %d", res.StatusCode)
	}
	res.Body.Close()
	res = a.do("POST", "/api/auth/login", map[string]string{"email": "invited@test.com", "password": "newpass12"}, "", "")
	if res.StatusCode != 200 {
		t.Fatalf("login after accept %d", res.StatusCode)
	}
	res.Body.Close()

	target, _ := a.s.users.Create(context.Background(), "Target", "target@test.com", ptr("oldpass12"), auth.RoleReadOnly)
	first, _, _ := a.s.tokens.Issue(context.Background(), target)
	res = a.do("POST", "/api/users/"+itoa(target.ID)+"/password-token", map[string]any{}, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	newTok := body["data"].(map[string]any)["token"].(string)
	if newTok == first {
		t.Fatal("token not rotated")
	}
	res = a.do("GET", "/api/auth/password/"+first, nil, "", "")
	if res.StatusCode != 404 {
		t.Fatalf("old token %d", res.StatusCode)
	}
	res.Body.Close()
	res = a.do("GET", "/api/auth/password/"+newTok, nil, "", "")
	prev = decodeJSON(t, res)
	if prev["isInactive"] != false {
		t.Fatalf("reset preview %v", prev)
	}

	res = a.do("POST", "/api/users/"+itoa(admin.ID)+"/password-token", map[string]any{}, sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Fatalf("self reset %d", res.StatusCode)
	}
	res.Body.Close()

	sso, _ := a.s.users.Create(context.Background(), "Sso", "sso@test.com", ptr("x"), auth.RoleReadOnly)
	_, _ = a.db.Exec(`UPDATE users SET is_sso_only = 1 WHERE id = ?`, sso.ID)
	res = a.do("POST", "/api/users/"+itoa(sso.ID)+"/password-token", map[string]any{}, sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Fatalf("sso reset %d", res.StatusCode)
	}
	res.Body.Close()

	inv, _ := a.s.users.Create(context.Background(), "Inv2", "inv2@test.com", nil, auth.RoleReadOnly)
	issued, _, _ := a.s.tokens.Issue(context.Background(), inv)
	res = a.do("PATCH", "/api/users/"+itoa(inv.ID), map[string]string{"password": "setbyadmin"}, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != 200 || body["data"].(map[string]any)["isInactive"] != false {
		t.Fatalf("admin set password %d %v", res.StatusCode, body)
	}
	res = a.do("GET", "/api/auth/password/"+issued, nil, "", "")
	if res.StatusCode != 404 {
		t.Fatalf("revoked after admin set %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestPasswordLoginToggle(t *testing.T) {
	a := newTestApp(t)
	a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
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

	admin, _ := a.s.users.Create(context.Background(), "Ad", "ad@test.com", ptr("secret1"), auth.RoleAdmin)
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

func TestCSRFOnLogout(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", auth.RoleReadWrite)
	issued := a.issue(u, false)
	res := a.do("POST", "/api/auth/logout", map[string]any{}, issued.Token, "")
	if res.StatusCode != 419 {
		t.Fatalf("csrf %d", res.StatusCode)
	}
	res.Body.Close()
	res = a.do("POST", "/api/auth/logout", map[string]any{}, issued.Token, issued.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("logout %d", res.StatusCode)
	}
	res.Body.Close()
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

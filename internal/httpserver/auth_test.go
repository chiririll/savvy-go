//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"savvy-go/internal/firstrun"
	"savvy-go/internal/spaces/spacefill"
)

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
	guard := firstrun.New(cfg.DataDir, firstrun.Binary)
	s := New(cfg.WithAppURL("https://savvy.test"), openTestStore(t, cfg), testKeys(t, cfg), WithBootstrap(guard))
	ts := httptest.NewTLSServer(s.Handler())
	t.Cleanup(ts.Close)
	client := ts.Client()

	raw, _ := json.Marshal(
		map[string]string{"name": "A", "email": "a@a.com", "password": "secret12", "bootstrap_password": guard.Password()},
	)
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

func TestSessionIdleAbsoluteRevokeRefresh(t *testing.T) {
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

func ptr(s string) *string { return &s }

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

// spaceCreator creates spaces with the server's own services, as the create-space route does.
func (a *testApp) spaceCreator() spacefill.Creator {
	return spacefill.Creator{Spaces: a.svc.Spaces, Store: a.svc.Store, Rates: a.svc.Rates}
}

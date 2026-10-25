//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/settings"
)

func TestLoginSSOOnly2FAChallenge(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("sso@test.com", "secret1", roleEditor)
	_, _ = a.db.Exec(`UPDATE users SET is_sso_only = 1 WHERE id = ?`, u.ID)
	res := a.do("POST", "/api/auth/login", map[string]string{"email": "sso@test.com", "password": "secret1"}, "", "")
	if res.StatusCode != 422 {
		t.Fatalf("sso-only %d", res.StatusCode)
	}
	res.Body.Close()

	u2 := a.createUser("u@test.com", "secret1", roleEditor)
	_, _ = a.db.Exec(
		`UPDATE users SET two_factor_enabled = 1, two_factor_confirmed = 1, two_factor_secret = 'SECRET' WHERE id = ?`,
		u2.ID,
	)
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

func TestTwoFactorChallengePeekConsume(t *testing.T) {
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
	_, _ = a.db.Exec(
		`UPDATE two_factor_challenges SET expires_at = ?`,
		time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano),
	)
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
		VALUES ('IdP', 'idp', 'oidc', 'custom_oidc', ?,
			'{"discovery_url":"https://idp.test","client_id":"c"}', 'read-only')`, en)
	if err != nil {
		t.Fatal(err)
	}
}

// An unknown email, an SSO-only account and a wrong password must be
// indistinguishable by status and body (the bcrypt work is covered in the
// session package).
func TestLoginAnswersSameForUnknownWrong(t *testing.T) {
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

func TestPlainCookieNotAuthenticateOverHTTPS(t *testing.T) {
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

func TestForwardedProtoNotMakeCookiesSecure(t *testing.T) {
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

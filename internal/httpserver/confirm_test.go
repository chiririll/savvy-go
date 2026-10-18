package httpserver

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/permissions"
	"savvy-go/internal/auth/totp"
)

// A stolen session must not be enough to add a lasting credential: each of these
// requests also carries the password (or, with 2FA on, an authenticator code).

// sensitive is one request that changes a credential, with the extra fields it takes.
type sensitive struct {
	name   string
	method string
	path   func(t *testing.T, a *testApp, u *auth.User) string
	body   func() map[string]any
}

func sensitiveRequests() []sensitive {
	grant := func() map[string]any {
		return map[string]any{"name": "Zapier", "permissions": allPerms(permissions.Read), "all_spaces": true}
	}
	fixed := func(p string) func(*testing.T, *testApp, *auth.User) string {
		return func(*testing.T, *testApp, *auth.User) string { return p }
	}
	passkey := func(t *testing.T, a *testApp, u *auth.User) string {
		t.Helper()
		if _, err := a.db.Exec(`INSERT INTO webauthn_credentials (user_id, credential_id, name, record, created_at)
			VALUES (?, 'cred-1', 'Laptop', '{}', '2026-01-01T00:00:00Z')`, u.ID); err != nil {
			t.Fatal(err)
		}
		var id int64
		if err := a.db.QueryRow(`SELECT id FROM webauthn_credentials WHERE user_id = ?`, u.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return "/api/auth/webauthn/credentials/" + strconv.FormatInt(id, 10)
	}
	token := func(t *testing.T, a *testApp, u *auth.User) string {
		t.Helper()
		_, tok := a.issueToken(u, tokRead, nil)
		return "/api/auth/api-tokens/" + strconv.FormatInt(tok.ID, 10)
	}
	return []sensitive{
		{
			"register a passkey",
			"POST",
			fixed("/api/auth/webauthn/register/options"),
			func() map[string]any { return map[string]any{} },
		},
		{"delete a passkey", "DELETE", passkey, func() map[string]any { return map[string]any{} }},
		{"enable 2FA", "POST", fixed("/api/auth/2fa/enable"), func() map[string]any { return map[string]any{} }},
		{"issue an API token", "POST", fixed("/api/auth/api-tokens"), grant},
		{"widen an API token", "PATCH", token, func() map[string]any {
			return map[string]any{"permissions": allPerms(permissions.Write), "all_spaces": true}
		}},
	}
}

func with(body map[string]any, extra map[string]any) map[string]any {
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func TestSensitiveChangesNeedThePassword(t *testing.T) {
	for _, tc := range sensitiveRequests() {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t)
			u := a.createUser("c@test.com", "secret1", roleEditor)
			sess := a.issue(u, false)
			path := tc.path(t, a, u)

			res := a.do(tc.method, path, tc.body(), sess.Token, sess.CSRF)
			if body := decodeJSON(t, res); res.StatusCode != http.StatusForbidden || body["error"] != "confirmation_required" {
				t.Fatalf("no password: %d %v", res.StatusCode, body)
			}

			res = a.do(tc.method, path, with(tc.body(), map[string]any{"current_password": "wrong-pass"}), sess.Token, sess.CSRF)
			if body := decodeJSON(t, res); res.StatusCode != http.StatusUnprocessableEntity ||
				body["error"] != "confirmation_invalid" {
				t.Fatalf("wrong password: %d %v", res.StatusCode, body)
			}

			res = a.do(tc.method, path, with(tc.body(), map[string]any{"current_password": "secret1"}), sess.Token, sess.CSRF)
			res.Body.Close()
			if res.StatusCode >= 300 {
				t.Fatalf("right password: %d", res.StatusCode)
			}
		})
	}
}

// With 2FA on, the password is not enough: whoever has a stolen session and a
// phished password still lacks the authenticator.
func TestSensitiveChangesNeedTwoFactorCode(t *testing.T) {
	a := newTestApp(t)
	secret, _ := twoFactorUser(t, a)
	u, err := a.svc.Users.ByEmail(t.Context(), "otp@test.com")
	if err != nil {
		t.Fatal(err)
	}
	sess := a.issue(u, false)
	body := func(extra map[string]any) map[string]any {
		return with(map[string]any{"name": "Zapier", "permissions": allPerms(permissions.Read), "all_spaces": true}, extra)
	}

	res := a.do("POST", "/api/auth/api-tokens", body(map[string]any{"current_password": "secret1"}), sess.Token, sess.CSRF)
	if b := decodeJSON(t, res); res.StatusCode != http.StatusForbidden || b["error"] != "confirmation_required" {
		t.Fatalf("password only: %d %v", res.StatusCode, b)
	}
	res = a.do("POST", "/api/auth/api-tokens", body(map[string]any{"code": "000000"}), sess.Token, sess.CSRF)
	if b := decodeJSON(t, res); res.StatusCode != http.StatusUnprocessableEntity || b["error"] != "confirmation_invalid" {
		t.Fatalf("wrong code: %d %v", res.StatusCode, b)
	}
	res = a.do(
		"POST",
		"/api/auth/api-tokens",
		body(map[string]any{"code": totp.At(secret, time.Now())}),
		sess.Token,
		sess.CSRF,
	)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("right code: %d", res.StatusCode)
	}
}

// A code works once, here as everywhere: a second change needs a fresh one.
func TestSensitiveChangeNotAcceptReplayedCode(t *testing.T) {
	a := newTestApp(t)
	secret, _ := twoFactorUser(t, a)
	u, _ := a.svc.Users.ByEmail(t.Context(), "otp@test.com")
	sess := a.issue(u, false)
	code := totp.At(secret, time.Now())
	req := map[string]any{"name": "Zapier", "permissions": allPerms(permissions.Read), "all_spaces": true, "code": code}

	res := a.do("POST", "/api/auth/api-tokens", req, sess.Token, sess.CSRF)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("first use: %d", res.StatusCode)
	}
	res = a.do("POST", "/api/auth/api-tokens", req, sess.Token, sess.CSRF)
	res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("replayed code: %d", res.StatusCode)
	}
}

// An account that signs in with SSO and has neither a password nor 2FA has
// nothing to confirm with, so it is not locked out of setting one up.
func TestSensitiveChangesPassWithNothingConfirm(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("sso@test.com", "secret1", roleEditor)
	if _, err := a.db.Exec(`UPDATE users SET password = NULL, is_sso_only = 1 WHERE id = ?`, u.ID); err != nil {
		t.Fatal(err)
	}
	u, _ = a.svc.Users.ByID(t.Context(), u.ID)
	sess := a.issue(u, false)

	res := a.do("POST", "/api/auth/2fa/enable", map[string]any{}, sess.Token, sess.CSRF)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("enable 2FA without a password: %d", res.StatusCode)
	}
}

// Guessing the password through these routes counts against the same limit as
// signing in, so a stolen session is no way round it.
func TestWrongConfirmationsCountTowardAccountLimit(t *testing.T) {
	a := newTestApp(t, WithConfig(testFile("[security]\nip_rate_limit = 0\naccount_rate_limit = 3\n")))
	freezeLimits(a)
	u := a.createUser("c@test.com", "secret1", roleEditor)
	sess := a.issue(u, false)

	for i := range 3 {
		res := a.do("POST", "/api/auth/2fa/enable", map[string]any{"current_password": "wrong-pass"}, sess.Token, sess.CSRF)
		res.Body.Close()
		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("guess %d: %d", i+1, res.StatusCode)
		}
	}
	res := a.do("POST", "/api/auth/2fa/enable", map[string]any{"current_password": "secret1"}, sess.Token, sess.CSRF)
	res.Body.Close()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("right password after too many guesses: %d", res.StatusCode)
	}
}

// Renaming a passkey and revoking a token only take access away or label it, so
// they ask for nothing.
func TestHarmlessCredentialChangesAskForNothing(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("c@test.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	_, tok := a.issueToken(u, tokRead, nil)

	res := a.do("DELETE", "/api/auth/api-tokens/"+strconv.FormatInt(tok.ID, 10), nil, sess.Token, sess.CSRF)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("revoke a token: %d", res.StatusCode)
	}
}

// Approving an app mints it a token, so it asks like issuing one does. Declining
// gives nothing away and asks for nothing.
func TestOAuthConsentNeedsPasswordApprove(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("oauth@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	id := a.registerClient()
	decide := func(approve bool, extra map[string]any) *http.Response {
		return a.do("POST", "/api/oauth/authorize", with(map[string]any{
			"client_id": id, "redirect_uri": oauthRedirect, "state": "st-1",
			"code_challenge": oauthChallenge(), "code_challenge_method": "S256", "approve": approve, "all_spaces": true,
		}, extra), sess.Token, sess.CSRF)
	}

	res := decide(true, nil)
	if body := decodeJSON(t, res); res.StatusCode != http.StatusForbidden || body["error"] != "confirmation_required" {
		t.Fatalf("no password: %d %v", res.StatusCode, body)
	}
	res = decide(true, map[string]any{"current_password": "wrong-pass"})
	if body := decodeJSON(t, res); res.StatusCode != http.StatusUnprocessableEntity ||
		body["error"] != "confirmation_invalid" {
		t.Fatalf("wrong password: %d %v", res.StatusCode, body)
	}
	status(t, decide(false, nil), http.StatusOK, "declining")
	status(t, decide(true, map[string]any{"current_password": "secret1"}), http.StatusOK, "right password")
}

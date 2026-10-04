package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"savvy-go/internal/auth"
)

// bearer performs a request authenticated only by an API token.
func (a *testApp) bearer(method, path string, body any, token string) *http.Response {
	a.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequest(method, a.srv.URL+a.apiPath(path), &buf)
	if err != nil {
		a.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	return res
}

func (a *testApp) issueToken(u *auth.User, scope string, exp *time.Time) (string, *auth.APIToken) {
	a.t.Helper()
	raw, tok, err := a.s.apiTokens.Issue(context.Background(), u, "app", scope, exp)
	if err != nil {
		a.t.Fatal(err)
	}
	return raw, tok
}

func status(t *testing.T, res *http.Response, want int, label string) {
	t.Helper()
	res.Body.Close()
	if res.StatusCode != want {
		t.Fatalf("%s: got %d want %d", label, res.StatusCode, want)
	}
}

func TestAPITokenCreateListRevoke(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)

	res := a.do("POST", "/api/auth/api-tokens", map[string]any{"name": "Zapier", "scope": "read-write"}, sess.Token, sess.CSRF)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %d", res.StatusCode)
	}
	data := decodeJSON(t, res)["data"].(map[string]any)
	raw, _ := data["token"].(string)
	if len(raw) < 20 || raw[:4] != "svy_" {
		t.Fatalf("token %q", raw)
	}

	var stored string
	if err := a.db.QueryRow(`SELECT token_hash FROM api_tokens`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == raw || stored != auth.HashToken(raw) {
		t.Fatalf("raw token must not be stored")
	}

	list := decodeJSON(t, a.do("GET", "/api/auth/api-tokens", nil, sess.Token, ""))
	items := list["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("list %v", list)
	}
	if _, leaked := items[0].(map[string]any)["token"]; leaked {
		t.Fatal("list must not expose the token")
	}

	id := strconv.Itoa(int(data["id"].(float64)))
	status(t, a.do("DELETE", "/api/auth/api-tokens/"+id, nil, sess.Token, sess.CSRF), 200, "revoke")
	status(t, a.bearer("GET", "/api/accounts", nil, raw), 401, "revoked token")
	status(t, a.do("DELETE", "/api/auth/api-tokens/"+id, nil, sess.Token, sess.CSRF), 404, "revoke twice")
}

func TestAPITokenBearerSkipsCSRF(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	raw, _ := a.issueToken(u, auth.APIScopeReadWrite, nil)

	status(t, a.bearer("GET", "/api/accounts", nil, raw), 200, "bearer GET")
	res := a.bearer("POST", "/api/tags", map[string]any{"name": "from-api"}, raw)
	if res.StatusCode != 201 && res.StatusCode != 200 {
		t.Fatalf("bearer POST %d", res.StatusCode)
	}
	res.Body.Close()

	// Cookie auth is still CSRF-protected.
	sess := a.issue(u, false)
	status(t, a.do("POST", "/api/tags", map[string]any{"name": "x"}, sess.Token, ""), 419, "cookie without csrf")
}

func TestAPITokenReadScopeBlocksWrites(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	raw, _ := a.issueToken(u, auth.APIScopeRead, nil)

	status(t, a.bearer("GET", "/api/tags", nil, raw), 200, "read GET")
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "nope"}, raw), 403, "read POST")
}

// P4: a token never writes more than its owner may in the space, whatever
// its scope; a viewer's read-write token reads but cannot write there.
func TestP4ViewerReadWriteTokenCannotWrite(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("ro@a.com", "secret1", roleViewer)
	raw, _ := a.issueToken(u, auth.APIScopeReadWrite, nil)

	status(t, a.bearer("GET", "/api/tags", nil, raw), 200, "viewer token GET")
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "nope"}, raw), 403, "viewer token POST")
}

func TestAPITokenInvalidExpiredAndNoCookieFallback(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	past := time.Now().Add(-time.Hour)
	expired, _ := a.issueToken(u, auth.APIScopeRead, &past)

	status(t, a.bearer("GET", "/api/accounts", nil, expired), 401, "expired")
	status(t, a.bearer("GET", "/api/accounts", nil, "svy_garbage"), 401, "garbage")
	status(t, a.bearer("GET", "/api/accounts", nil, "not-a-token"), 401, "wrong prefix")

	// A valid session cookie must not rescue a bad bearer.
	sess := a.issue(u, false)
	req, _ := http.NewRequest("GET", a.srv.URL+a.apiPath("/api/accounts"), nil)
	req.AddCookie(&http.Cookie{Name: "svy_session", Value: sess.Token})
	req.Header.Set("Authorization", "Bearer svy_garbage")
	res, err := a.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	status(t, res, 401, "bad bearer + good cookie")
}

func TestAPITokenForbiddenOnSessionOnlyAndAdminRoutes(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@a.com", "secret1", auth.RoleAdmin)
	raw, _ := a.issueToken(admin, auth.APIScopeReadWrite, nil)

	status(t, a.bearer("POST", "/api/auth/api-tokens", map[string]any{"name": "x"}, raw), 403, "mint via token")
	status(t, a.bearer("GET", "/api/auth/api-tokens", nil, raw), 403, "list via token")
	status(t, a.bearer("GET", "/api/backups", nil, raw), 403, "backups")
	status(t, a.bearer("PATCH", "/api/settings", map[string]any{}, raw), 403, "settings update")
	status(t, a.bearer("POST", "/api/auth/2fa/enable", nil, raw), 403, "2fa")
	status(t, a.bearer("GET", "/api/identity-providers", nil, raw), 403, "admin route")
	status(t, a.bearer("POST", "/api/users", map[string]any{}, raw), 403, "admin write")
}

func TestAPITokensDeletedWithUser(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	a.issueToken(u, auth.APIScopeRead, nil)
	if _, err := a.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	if err := a.s.users.Delete(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM api_tokens`).Scan(&n)
	if n != 0 {
		t.Fatalf("tokens left: %d", n)
	}
}

func TestAPITokenRevokeAndListAreScopedToOwner(t *testing.T) {
	a := newTestApp(t)
	alice := a.createUser("alice@a.com", "secret1", roleEditor)
	bob := a.createUser("bob@a.com", "secret1", roleEditor)
	aliceRaw, aliceTok := a.issueToken(alice, auth.APIScopeRead, nil)
	a.issueToken(bob, auth.APIScopeRead, nil)
	sess := a.issue(bob, false)

	items := decodeJSON(t, a.do("GET", "/api/auth/api-tokens", nil, sess.Token, ""))["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("bob sees %d tokens", len(items))
	}
	id := strconv.FormatInt(aliceTok.ID, 10)
	status(t, a.do("DELETE", "/api/auth/api-tokens/"+id, nil, sess.Token, sess.CSRF), 404, "revoke foreign token")
	status(t, a.bearer("GET", "/api/accounts", nil, aliceRaw), 200, "foreign token still valid")
}

func TestAPITokenDemotedUserLosesWrite(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	raw, _ := a.issueToken(u, auth.APIScopeReadWrite, nil)
	if err := a.s.spaces.SetMember(context.Background(), a.space.ID, u, roleViewer); err != nil {
		t.Fatal(err)
	}

	status(t, a.bearer("GET", "/api/tags", nil, raw), 200, "demoted GET")
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "nope"}, raw), 403, "demoted POST")
}

func TestAPITokenStoreValidation(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'x'
	}

	cases := []struct {
		label string
		body  any
		field string
	}{
		{"empty name", map[string]any{"name": "  "}, "name"},
		{"long name", map[string]any{"name": string(long)}, "name"},
		{"unknown scope", map[string]any{"name": "x", "scope": "admin"}, "scope"},
		{"past expiry", map[string]any{"name": "x", "expires_at": time.Now().Add(-time.Hour).Format(time.RFC3339)}, "expires_at"},
		{"bad expiry", map[string]any{"name": "x", "expires_at": "2030-01-01"}, "expires_at"},
		{"malformed body", "not an object", "name"},
	}
	for _, c := range cases {
		res := a.do("POST", "/api/auth/api-tokens", c.body, sess.Token, sess.CSRF)
		if res.StatusCode != http.StatusUnprocessableEntity {
			res.Body.Close()
			t.Fatalf("%s: got %d want 422", c.label, res.StatusCode)
		}
		errs, _ := decodeJSON(t, res)["errors"].(map[string]any)
		if _, ok := errs[c.field]; !ok {
			t.Fatalf("%s: no error for %q in %v", c.label, c.field, errs)
		}
	}

	var n int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM api_tokens`).Scan(&n)
	if n != 0 {
		t.Fatalf("invalid requests created %d tokens", n)
	}
}

func TestAPITokenDefaultScopeAndFutureExpiry(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	exp := time.Now().Add(24 * time.Hour).Truncate(time.Second)

	res := a.do("POST", "/api/auth/api-tokens", map[string]any{"name": "ci", "expires_at": exp.Format(time.RFC3339)}, sess.Token, sess.CSRF)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %d", res.StatusCode)
	}
	data := decodeJSON(t, res)["data"].(map[string]any)
	if data["scope"] != auth.APIScopeRead {
		t.Fatalf("default scope %v", data["scope"])
	}
	if data["expires_at"] != exp.UTC().Format(time.RFC3339) {
		t.Fatalf("expires_at %v", data["expires_at"])
	}
	status(t, a.bearer("GET", "/api/accounts", nil, data["token"].(string)), 200, "unexpired token")
}

func TestAPITokenLimit(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	for range 50 {
		a.issueToken(u, auth.APIScopeRead, nil)
	}
	sess := a.issue(u, false)

	status(t, a.do("POST", "/api/auth/api-tokens", map[string]any{"name": "51st"}, sess.Token, sess.CSRF), 422, "over limit")
	if _, _, err := a.s.apiTokens.Issue(context.Background(), u, "x", auth.APIScopeRead, nil); err != auth.ErrTokenLimit {
		t.Fatalf("issue over limit: %v", err)
	}
}

func TestAPITokenTouchesLastUsed(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	raw, tok := a.issueToken(u, auth.APIScopeRead, nil)
	lastUsed := func() string {
		var v sql.NullString
		if err := a.db.QueryRow(`SELECT last_used_at FROM api_tokens WHERE id = ?`, tok.ID).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v.String
	}
	setLastUsed := func(ago time.Duration) string {
		v := time.Now().Add(-ago).UTC().Format(time.RFC3339Nano)
		if _, err := a.db.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, v, tok.ID); err != nil {
			t.Fatal(err)
		}
		return v
	}

	status(t, a.bearer("GET", "/api/accounts", nil, raw), 200, "first use")
	if lastUsed() == "" {
		t.Fatal("last_used_at not set on first use")
	}

	recent := setLastUsed(30 * time.Second)
	status(t, a.bearer("GET", "/api/accounts", nil, raw), 200, "recent use")
	if lastUsed() != recent {
		t.Fatal("last_used_at rewritten within the touch interval")
	}

	stale := setLastUsed(2 * time.Minute)
	status(t, a.bearer("GET", "/api/accounts", nil, raw), 200, "stale use")
	if lastUsed() == stale {
		t.Fatal("last_used_at not refreshed after the touch interval")
	}
}

func TestAPITokenMalformedAuthorizationHeader(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)

	for _, h := range []string{"Basic dXNlcjpwYXNz", "Bearer", "Bearer   ", "svy_nobearerprefix"} {
		req, _ := http.NewRequest("GET", a.srv.URL+a.apiPath("/api/accounts"), nil)
		req.AddCookie(&http.Cookie{Name: "svy_session", Value: sess.Token})
		req.Header.Set("Authorization", h)
		res, err := a.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		status(t, res, 401, "Authorization: "+h)
	}
}

func TestAPITokenForbiddenOnCredentialRoutes(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	raw, _ := a.issueToken(u, auth.APIScopeReadWrite, nil)
	sess := a.issue(u, false)

	status(t, a.bearer("POST", "/api/auth/logout", nil, raw), 403, "logout")
	status(t, a.bearer("POST", "/api/auth/logout-others", nil, raw), 403, "logout others")
	status(t, a.bearer("PUT", "/api/auth/password", map[string]any{"current_password": "secret1", "password": "secret2", "password_confirmation": "secret2"}, raw), 403, "change password")
	status(t, a.bearer("GET", "/api/auth/2fa/status", nil, raw), 403, "2fa status")
	status(t, a.bearer("GET", "/api/auth/webauthn/credentials", nil, raw), 403, "webauthn list")
	status(t, a.bearer("DELETE", "/api/auth/webauthn/credentials/1", nil, raw), 403, "webauthn delete")
	status(t, a.bearer("DELETE", "/api/auth/api-tokens/1", nil, raw), 403, "revoke via token")
	status(t, a.bearer("POST", "/api/backups", nil, raw), 403, "backup create")
	status(t, a.bearer("DELETE", "/api/backups/x.sqlite", nil, raw), 403, "backup delete")

	// The blocked logout-others must have left existing sessions intact.
	status(t, a.do("GET", "/api/accounts", nil, sess.Token, ""), 200, "session still valid")
}

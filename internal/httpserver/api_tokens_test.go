//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/permissions"
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

// Shorthands for tokens that reach every space of their owner: tokRead reads
// everything, tokReadWrite also writes. Tests of finer grants use issueGrant.
const (
	tokRead      = "read"
	tokReadWrite = "read-write"
)

func allPerms(level string) []string {
	out := make([]string, len(permissions.APIResources))
	for i, r := range permissions.APIResources {
		out[i] = permissions.New(r, level)
	}
	return out
}

func (a *testApp) issueToken(u *auth.User, scope string, exp *time.Time) (string, *auth.APIToken) {
	a.t.Helper()
	level := permissions.Read
	if scope == tokReadWrite {
		level = permissions.Delete
	}
	return a.issueGrant(u, auth.Grant{Permissions: allPerms(level), AllSpaces: true}, exp)
}

func (a *testApp) issueGrant(u *auth.User, g auth.Grant, exp *time.Time) (string, *auth.APIToken) {
	a.t.Helper()
	raw, tok, err := a.svc.APITokens.Issue(context.Background(), u, "app", g, exp)
	if err != nil {
		a.t.Fatal(err)
	}
	return raw, tok
}

// confirmed adds the password every credential change asks for. createUser gives
// its users the password "secret1".
func confirmed(body map[string]any) map[string]any {
	out := map[string]any{"current_password": "secret1"}
	maps.Copy(out, body)
	return out
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

	res := a.do(
		"POST",
		"/api/auth/api-tokens",
		confirmed(map[string]any{"name": "Zapier", "permissions": allPerms(permissions.Write), "all_spaces": true}),
		sess.Token,
		sess.CSRF,
	)
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
	raw, _ := a.issueToken(u, tokReadWrite, nil)

	status(t, a.bearer("GET", "/api/accounts", nil, raw), 200, "bearer GET")
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "from-api"}, raw), 201, "bearer POST")

	// Cookie auth is still CSRF-protected.
	sess := a.issue(u, false)
	status(t, a.do("POST", "/api/tags", map[string]any{"name": "x"}, sess.Token, ""), 419, "cookie without csrf")
}

func TestAPITokenReadScopeBlocksWrites(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	raw, _ := a.issueToken(u, tokRead, nil)

	status(t, a.bearer("GET", "/api/tags", nil, raw), 200, "read GET")
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "nope"}, raw), 403, "read POST")
}

func TestAPITokensDeletedWithUser(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	a.issueToken(u, tokRead, nil)
	if _, err := a.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	if err := a.svc.Users.Delete(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM api_tokens`).Scan(&n)
	if n != 0 {
		t.Fatalf("tokens left: %d", n)
	}
}

func TestAPITokenRevokeListScopedOwner(t *testing.T) {
	a := newTestApp(t)
	alice := a.createUser("alice@a.com", "secret1", roleEditor)
	bob := a.createUser("bob@a.com", "secret1", roleEditor)
	aliceRaw, aliceTok := a.issueToken(alice, tokRead, nil)
	a.issueToken(bob, tokRead, nil)
	sess := a.issue(bob, false)

	items := decodeJSON(t, a.do("GET", "/api/auth/api-tokens", nil, sess.Token, ""))["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("bob sees %d tokens", len(items))
	}
	id := strconv.FormatInt(aliceTok.ID, 10)
	status(t, a.do("DELETE", "/api/auth/api-tokens/"+id, nil, sess.Token, sess.CSRF), 404, "revoke foreign token")
	status(t, a.bearer("GET", "/api/accounts", nil, aliceRaw), 200, "foreign token still valid")
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
		{
			"unknown permission",
			map[string]any{"name": "x", "all_spaces": true, "permissions": []string{"admin:all"}},
			"permissions",
		},
		{
			"unknown resource",
			map[string]any{"name": "x", "all_spaces": true, "permissions": []string{"users:read"}},
			"permissions",
		},
		{"no spaces", map[string]any{"name": "x", "permissions": []string{"accounts:read"}}, "space_ids"},
		{
			"all and some spaces",
			map[string]any{"name": "x", "all_spaces": true, "space_ids": []int64{1}, "permissions": []string{"accounts:read"}},
			"space_ids",
		},
		{
			"foreign space",
			map[string]any{"name": "x", "space_ids": []int64{9999}, "permissions": []string{"accounts:read"}},
			"space_ids",
		},
		{
			"past expiry",
			map[string]any{"name": "x", "expires_at": time.Now().Add(-time.Hour).Format(time.RFC3339)},
			"expires_at",
		},
		{"bad expiry", map[string]any{"name": "x", "expires_at": "2030-01-01"}, "expires_at"},
		{"malformed body", "not an object", "name"},
	}
	for _, c := range cases {
		body := c.body
		if m, ok := body.(map[string]any); ok {
			body = confirmed(m)
		}
		res := a.do("POST", "/api/auth/api-tokens", body, sess.Token, sess.CSRF)
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

func TestAPITokenGrantFutureExpiry(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	exp := time.Now().Add(24 * time.Hour).Truncate(time.Second)

	res := a.do(
		"POST",
		"/api/auth/api-tokens",
		confirmed(
			map[string]any{
				"name":        "ci",
				"all_spaces":  true,
				"permissions": []string{"accounts:read"},
				"expires_at":  exp.Format(time.RFC3339),
			},
		),
		sess.Token,
		sess.CSRF,
	)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %d", res.StatusCode)
	}
	data := decodeJSON(t, res)["data"].(map[string]any)
	if perms, _ := data["permissions"].([]any); len(perms) != 1 || perms[0] != "accounts:read" {
		t.Fatalf("permissions %v", data["permissions"])
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
		a.issueToken(u, tokRead, nil)
	}
	sess := a.issue(u, false)

	status(
		t,
		a.do("POST", "/api/auth/api-tokens", confirmed(map[string]any{"name": "51st"}), sess.Token, sess.CSRF),
		422,
		"over limit",
	)
	if _, _,
		err := a.svc.APITokens.Issue(context.Background(), u, "x",
		auth.Grant{Permissions: allPerms(permissions.Read), AllSpaces: true}, nil); err != auth.ErrTokenLimit {
		t.Fatalf("issue over limit: %v", err)
	}
}

func TestAPITokenTouchesLastUsed(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	raw, tok := a.issueToken(u, tokRead, nil)
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

// A bearer that cannot be used is a 401 even when the same request carries a good session
// cookie: a bad credential is never rescued by another one.
func TestUnusableBearerCredentialsAreRejected(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", roleEditor)
	past := time.Now().Add(-time.Hour)
	expired, _ := a.issueToken(u, tokRead, &past)
	sess := a.issue(u, false)

	for _, c := range []struct{ name, header string }{
		{"expired token", "Bearer " + expired},
		{"unknown token", "Bearer svy_garbage"},
		{"token without the prefix", "Bearer not-a-token"},
		{"other scheme", "Basic dXNlcjpwYXNz"},
		{"empty bearer", "Bearer   "},
		{"no scheme", "svy_nobearerprefix"},
	} {
		for _, cookie := range []bool{false, true} {
			req, _ := http.NewRequest("GET", a.srv.URL+a.apiPath("/api/accounts"), nil)
			req.Header.Set("Authorization", c.header)
			if cookie {
				req.AddCookie(&http.Cookie{Name: "svy_session", Value: sess.Token})
			}
			res, err := a.client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			status(t, res, 401, fmt.Sprintf("%s (session cookie: %v)", c.name, cookie))
		}
	}
}

// Managing credentials, backups and the server is for people in a session: an API token is
// refused there, whoever it belongs to and however much it may do elsewhere.
func TestAPITokensRefusedSessionOnlyRoutes(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@a.com", "secret1", auth.RoleAdmin)
	raw, _ := a.issueToken(admin, tokReadWrite, nil)
	sess := a.issue(admin, false)

	for _, r := range []struct{ method, path string }{
		{"POST", "/api/auth/api-tokens"},
		{"GET", "/api/auth/api-tokens"},
		{"DELETE", "/api/auth/api-tokens/1"},
		{"POST", "/api/auth/logout"},
		{"POST", "/api/auth/logout-others"},
		{"PUT", "/api/auth/password"},
		{"POST", "/api/auth/2fa/enable"},
		{"GET", "/api/auth/2fa/status"},
		{"GET", "/api/auth/webauthn/credentials"},
		{"DELETE", "/api/auth/webauthn/credentials/1"},
		{"GET", "/api/backups"},
		{"POST", "/api/backups"},
		{"DELETE", "/api/backups/x.sqlite"},
		{"PATCH", "/api/settings"},
		{"GET", "/api/identity-providers"},
		{"POST", "/api/users"},
	} {
		status(t, a.bearer(r.method, r.path, map[string]any{}, raw), 403, r.method+" "+r.path)
	}
	// The refused logout-others ended no session.
	status(t, a.do("GET", "/api/accounts", nil, sess.Token, ""), 200, "session still valid")
}

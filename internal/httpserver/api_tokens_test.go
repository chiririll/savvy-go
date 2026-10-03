package httpserver

import (
	"bytes"
	"context"
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
	req, err := http.NewRequest(method, a.srv.URL+path, &buf)
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
	u := a.createUser("rw@a.com", "secret1", auth.RoleReadWrite)
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
	u := a.createUser("rw@a.com", "secret1", auth.RoleReadWrite)
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
	u := a.createUser("rw@a.com", "secret1", auth.RoleReadWrite)
	raw, _ := a.issueToken(u, auth.APIScopeRead, nil)

	status(t, a.bearer("GET", "/api/tags", nil, raw), 200, "read GET")
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "nope"}, raw), 403, "read POST")
}

func TestAPITokenReadOnlyUserCannotMintReadWrite(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("ro@a.com", "secret1", auth.RoleReadOnly)
	sess := a.issue(u, false)

	status(t, a.do("POST", "/api/auth/api-tokens", map[string]any{"name": "x", "scope": "read-write"}, sess.Token, sess.CSRF), 422, "rw token")
	status(t, a.do("POST", "/api/auth/api-tokens", map[string]any{"name": "x", "scope": "read"}, sess.Token, sess.CSRF), 201, "read token")
}

func TestAPITokenInvalidExpiredAndNoCookieFallback(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@a.com", "secret1", auth.RoleReadWrite)
	past := time.Now().Add(-time.Hour)
	expired, _ := a.issueToken(u, auth.APIScopeRead, &past)

	status(t, a.bearer("GET", "/api/accounts", nil, expired), 401, "expired")
	status(t, a.bearer("GET", "/api/accounts", nil, "svy_garbage"), 401, "garbage")
	status(t, a.bearer("GET", "/api/accounts", nil, "not-a-token"), 401, "wrong prefix")

	// A valid session cookie must not rescue a bad bearer.
	sess := a.issue(u, false)
	req, _ := http.NewRequest("GET", a.srv.URL+"/api/accounts", nil)
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
	u := a.createUser("rw@a.com", "secret1", auth.RoleReadWrite)
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

package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"savvy-go/internal/oauth"
)

func TestOAuthRegistrationClosedByDefault(t *testing.T) {
	a := newTestApp(t)
	raw, _ := json.Marshal(map[string]any{"client_name": "Claude", "redirect_uris": []string{oauthRedirect}})
	res, err := a.client.Post(a.srv.URL+"/oauth/register", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	body := decodeJSON(t, res)
	// A plain "not found": nothing says a window exists or an admin can open it.
	msg, _ := body["message"].(string)
	if res.StatusCode != http.StatusNotFound || body["error"] != nil || msg != "Not found." {
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
	status(
		t,
		a.do("POST", "/api/admin/oauth/registration", nil, sess.Token, sess.CSRF),
		http.StatusForbidden,
		"a normal user",
	)
	status(t, a.do("POST", "/api/admin/oauth/registration", nil, "", ""), http.StatusUnauthorized, "anonymous")

	admin := a.createUser("admin@a.com", "secret1", "admin")
	raw, _ := a.issueToken(admin, "read-write", nil)
	status(t, a.bearer("POST", "/api/admin/oauth/registration", nil, raw), http.StatusForbidden, "an admin's API token")
	if _, open := a.svc.OAuth.RegistrationOpenUntil(); open {
		t.Fatal("a refused request opened registration")
	}
}

func TestAdminSeesWhereRegistrationCameFrom(t *testing.T) {
	a := newTestApp(t)
	admin := a.issue(a.createUser("admin@a.com", "secret1", "admin"), false)
	a.svc.OAuth.OpenRegistration(oauth.RegistrationWindow)
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
	if c["name"] != "Claude" || c["ip"] != "203.0.113.7" || c["user_agent"] != "Claude-MCP/9.9" ||
		c["origin"] != "https://claude.ai" ||
		c["approved_at"] != nil ||
		c["registered_at"] == nil ||
		c["redirect_uris"].([]any)[0] != oauthRedirect {
		t.Fatalf("client = %v", c)
	}
}

func TestRemovingAppCutsItOffOnce(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("u@a.com", "secret1", roleEditor), false)
	admin := a.issue(a.createUser("admin@a.com", "secret1", "admin"), false)
	id, access, refresh := a.connectWithRefresh(user)
	if res := a.bearer("GET", "/api/spaces", nil, access); res.StatusCode != http.StatusOK {
		t.Fatalf("read = %d", res.StatusCode)
	}
	status(t, a.do("DELETE", "/api/admin/oauth/clients/"+id, nil, admin.Token, admin.CSRF), http.StatusNoContent, "remove")
	if res := a.bearer("GET", "/api/spaces", nil, access); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("read after removal = %d, want 401", res.StatusCode)
	}
	if _,
		body := a.token(url.Values{"grant_type": {"refresh_token"}, "client_id": {id},
		"refresh_token": {refresh}}); body["error"] == nil {
		t.Fatalf("a removed app refreshed its tokens: %v", body)
	}
	status(
		t,
		a.do("DELETE", "/api/admin/oauth/clients/"+id, nil, admin.Token, admin.CSRF),
		http.StatusNotFound,
		"remove twice",
	)
}

func TestOnlyAdminSessionsListRemoveApps(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("u@a.com", "secret1", roleEditor), false)
	id, access := a.connect(user)
	status(
		t,
		a.do("GET", "/api/admin/oauth/clients", nil, user.Token, user.CSRF),
		http.StatusForbidden,
		"list as a normal user",
	)
	status(
		t,
		a.do("DELETE", "/api/admin/oauth/clients/"+id, nil, user.Token, user.CSRF),
		http.StatusForbidden,
		"remove as a normal user",
	)
	// A token, even an admin's, cannot manage apps; neither can the app itself.
	adminUser := a.createUser("admin@a.com", "secret1", "admin")
	raw, _ := a.issueToken(adminUser, "read-write", nil)
	status(t, a.bearer("GET", "/api/admin/oauth/clients", nil, raw), http.StatusForbidden, "list with a token")
	status(
		t,
		a.bearer("DELETE", "/api/admin/oauth/clients/"+id, nil, access),
		http.StatusForbidden,
		"remove with its own token",
	)
}

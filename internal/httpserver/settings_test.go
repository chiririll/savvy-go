package httpserver

import (
	"context"
	"testing"

	"savvy-go/internal/auth"
)

// Server setup (onboarding) reads config_writable to know whether the public
// URL can be edited, and records on the server that it has been dealt with.
func TestServerSetupSettings(t *testing.T) {
	a := newTestApp(t)
	editor := a.issue(a.createUser("e@test.com", "secret1", roleEditor), false)
	admin := a.issue(a.createUser("a@test.com", "secret1", auth.RoleAdmin), false)

	body := decodeJSON(t, a.do("GET", "/api/settings", nil, admin.Token, ""))
	if _, ok := body["config_writable"].(bool); !ok {
		t.Fatalf("config_writable missing or not a bool: %v", body["config_writable"])
	}
	// An existing server never went through onboarding: nothing is pending.
	if v, ok := body["onboarding_server_done"]; !ok || v != nil {
		t.Fatalf("server setup is pending on an existing server: %v (present %v)", v, ok)
	}

	status(t, a.do("PATCH", "/api/settings", map[string]any{"onboarding_server_done": true}, editor.Token, editor.CSRF), 403, "editor finishes server setup")
	res := a.do("PATCH", "/api/settings", map[string]any{"onboarding_server_done": true}, admin.Token, admin.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["onboarding_server_done"] != true {
		t.Fatalf("admin finish %d %v", res.StatusCode, body)
	}
}

// Creating the first admin is what makes server setup pending, so the app can
// bring the admin back to it after a reload or a new tab.
func TestFirstAdminLeavesServerSetupPending(t *testing.T) {
	a := newTestApp(t)
	status(t, a.register(a.bootstrap.Password(), nil), 201, "register")
	all, err := a.s.settings.All(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if all["onboarding_server_done"] != false {
		t.Fatalf("server setup not pending after the first admin: %v", all["onboarding_server_done"])
	}
}

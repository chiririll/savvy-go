//go:build integration

package httpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/config"
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

	status(
		t,
		a.do("PATCH", "/api/settings", map[string]any{"onboarding_server_done": true}, editor.Token, editor.CSRF),
		403,
		"editor finishes server setup",
	)
	res := a.do("PATCH", "/api/settings", map[string]any{"onboarding_server_done": true}, admin.Token, admin.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["onboarding_server_done"] != true {
		t.Fatalf("admin finish %d %v", res.StatusCode, body)
	}
}

func TestAdminChangesThePublicURLFromThePanel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[server]\napp_url = \""+testURL+"\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := config.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Wired as the process wires it: the URL is read from the file and saved to it.
	server := config.Of(file, ServerSection)
	a := newTestAppCfg(t, func(c appconfig.Config) appconfig.Config {
		return c.WithServer("UTC", time.UTC, func() string { return server.Get().AppURL }).
			WithAppURLSetter(func(u string) error { return SetAppURL(file, u) })
	}, WithConfig(file))
	if err := file.Finish(); err != nil {
		t.Fatal(err)
	}
	admin := a.issue(a.createUser("a@test.com", "secret1", auth.RoleAdmin), false)

	res := a.do("PATCH", "/api/settings", map[string]any{"app_url": "http://savvy.example.org/"}, admin.Token, admin.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 200 || body["app_url"] != "http://savvy.example.org" || body["config_writable"] != true {
		t.Fatalf("change %d %v", res.StatusCode, body)
	}
	if saved, _ := os.ReadFile(path); !strings.Contains(string(saved), "http://savvy.example.org") {
		t.Errorf("not written to the config file:\n%s", saved)
	}

	for _, bad := range []any{"savvy.example.org", 5, nil} {
		res = a.do("PATCH", "/api/settings", map[string]any{"app_url": bad}, admin.Token, admin.CSRF)
		body = decodeJSON(t, res)
		if res.StatusCode != 422 || body["errors"] == nil {
			t.Errorf("app_url %v: %d %v", bad, res.StatusCode, body)
		}
	}
	if got := a.svc.Cfg.AppURL(); got != "http://savvy.example.org" {
		t.Errorf("a rejected URL was applied: %q", got)
	}

	res = a.do("PATCH", "/api/settings", map[string]any{"app_url": ""}, admin.Token, admin.CSRF)
	res.Body.Close()
	if got := a.svc.Cfg.AppURL(); got != "" {
		t.Errorf("the URL was not cleared: %q", got)
	}
}

func TestPublicURLLockedOnReadOnlyFile(t *testing.T) {
	a := newTestApp(t)
	admin := a.issue(a.createUser("a@test.com", "secret1", auth.RoleAdmin), false)

	res := a.do("PATCH", "/api/settings", map[string]any{"app_url": "https://other.example.org"}, admin.Token, admin.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 409 {
		t.Fatalf("read-only file: %d %v", res.StatusCode, body)
	}
	if got := a.svc.Cfg.AppURL(); got != testURL {
		t.Errorf("the URL changed: %q", got)
	}

	// Sending the URL it already has is not a change.
	res = a.do("PATCH", "/api/settings", map[string]any{"app_url": testURL + "/"}, admin.Token, admin.CSRF)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("an unchanged URL: %d", res.StatusCode)
	}
}

// Creating the first admin is what makes server setup pending, so the app can
// bring the admin back to it after a reload or a new tab.
func TestFirstAdminLeavesServerSetupPending(t *testing.T) {
	a := newTestApp(t)
	status(t, a.register(a.bootstrap.Password(), nil), 201, "register")
	all, err := a.svc.Settings.All(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if all["onboarding_server_done"] != false {
		t.Fatalf("server setup not pending after the first admin: %v", all["onboarding_server_done"])
	}
}

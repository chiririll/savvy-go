package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/config"
)

// start reads the config file with body as the builder does and returns its path.
func start(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if body != "" {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Init(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSecuritySection(t *testing.T) {
	start(t, "[security]\nsession_ttl = 30\ncsrf_header = \" X-Tok \"\n")
	sec := config.Of(auth.SecuritySection)
	if err := config.Finish(); err != nil {
		t.Fatal(err)
	}
	got := sec.Get()
	if got.SessionTTL != 30*time.Minute || got.RememberTTL != 7*24*time.Hour || got.ChallengeTTL != 5*time.Minute {
		t.Fatalf("lifetimes %+v", got)
	}
	if got.CSRFHeader != "X-Tok" || got.SessionCookie != "svy_session" || got.CSRFCookie != "svy_csrf" {
		t.Fatalf("names %+v", got)
	}
	if d := auth.DefaultSecurity(); d.SessionTTL != 24*time.Hour || d.CSRFHeader != "X-CSRF-Token" {
		t.Errorf("DefaultSecurity %+v", d)
	}
}

func TestSecuritySectionRefusesAZeroLifetime(t *testing.T) {
	start(t, "[security]\nremember_ttl = 0\n")
	config.Of(auth.SecuritySection)
	if err := config.Finish(); err == nil || !strings.Contains(err.Error(), "remember_ttl") {
		t.Fatalf("zero lifetime: %v", err)
	}
}

func TestServerAndPathsSections(t *testing.T) {
	dir := t.TempDir()
	path := start(t, "[server]\napp_url = \" https://x.example/ \"\ntimezone = \"Nowhere/Land\"\n\n[paths]\ndata = \""+filepath.ToSlash(dir)+"/d\"\n\n[unknown]\nkey = 1\n")
	server := config.Of(serverSection)
	paths := config.Of(pathsSection)
	if err := config.Finish(); err != nil {
		t.Fatal(err)
	}
	cfg := NewConfig(server.Get(), paths.Get())
	if cfg.TZ != "UTC" || cfg.Location != time.UTC {
		t.Errorf("an unknown time zone must fall back to UTC: %q", cfg.TZ)
	}
	if cfg.DataDir != filepath.ToSlash(dir)+"/d" || cfg.UploadsDir != filepath.Join(cfg.DataDir, "uploads") || cfg.ListenAddr != DefaultListen {
		t.Errorf("paths %+v", cfg)
	}
	if got := server.Get().AppURL; got != "https://x.example" {
		t.Errorf("AppURL %q", got)
	}
	text, _ := os.ReadFile(path)
	if strings.Contains(string(text), "unknown") || !strings.HasPrefix(string(text), "# Go Savvy configuration") {
		t.Errorf("file:\n%s", text)
	}
}

// AppURL follows the file: a change through the section shows at once.
func TestAppURLFollowsTheSection(t *testing.T) {
	start(t, "[server]\napp_url = \"https://x.example\"\n")
	server := config.Of(serverSection)
	if err := config.Finish(); err != nil {
		t.Fatal(err)
	}
	svc := &Services{appURL: func() string { return server.Get().AppURL }}
	if err := serverSection.Update(func(s *serverTable) { s.AppURL = "https://y.example///" }); err != nil {
		t.Fatal(err)
	}
	if got := svc.AppURL(); got != "https://y.example" {
		t.Errorf("AppURL after a change %q", got)
	}
}

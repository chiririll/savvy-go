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

// start reads the config file with body as the builder does.
func start(t *testing.T, body string) *config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if body != "" {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f, err := config.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestSecuritySection(t *testing.T) {
	f := start(t, "[security]\nsession_ttl = 30\ncsrf_header = \" X-Tok \"\n")
	sec := config.Of(f, auth.SecuritySection)
	if err := f.Finish(); err != nil {
		t.Fatal(err)
	}
	got := sec.Get()
	if got.SessionTTL != 30*time.Minute || got.RememberTTL != 7*24*time.Hour || got.ChallengeTTL != 5*time.Minute {
		t.Fatalf("lifetimes %+v", got)
	}
	if got.CSRFHeader != "X-Tok" || got.SessionCookie != "svy_session" || got.CSRFCookie != "svy_csrf" {
		t.Fatalf("names %+v", got)
	}
	if d := (auth.SecurityConfig{}).Defaults(); d.SessionTTL != 60*24 || d.CSRFHeader != "X-CSRF-Token" {
		t.Errorf("DefaultSecurity %+v", d)
	}
}

func TestSecuritySectionRefusesAZeroLifetime(t *testing.T) {
	f := start(t, "[security]\nremember_ttl = 0\n")
	config.Of(f, auth.SecuritySection)
	if err := f.Finish(); err == nil || !strings.Contains(err.Error(), "remember_ttl") {
		t.Fatalf("zero lifetime: %v", err)
	}
}

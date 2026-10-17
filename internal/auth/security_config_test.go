package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/config"
)

func TestDecodeSecurity(t *testing.T) {
	good := SecurityConfig{}.Defaults()
	sec, err := decodeSecurity(good)
	if err != nil {
		t.Fatal(err)
	}
	if sec.SessionTTL != 24*time.Hour || sec.RememberTTL != 7*24*time.Hour || sec.ChallengeTTL != 5*time.Minute {
		t.Errorf("lifetimes are minutes: %+v", sec)
	}
	if sec.IPRateLimit != 10 || sec.AccountRateLimit != 5 {
		t.Errorf("default rate limits: %+v", sec)
	}
	if sec.ResetTTL != time.Hour {
		t.Errorf("default reset link lifetime: %+v", sec)
	}
	off := good
	off.IPRateLimit, off.AccountRateLimit = 0, 0
	if _, err := decodeSecurity(off); err != nil {
		t.Errorf("zero turns a limit off: %v", err)
	}
	padded := good
	padded.Cookie, padded.CSRFHeader = "  sid ", " X-T "
	if sec, _ := decodeSecurity(padded); sec.SessionCookie != "sid" || sec.CSRFHeader != "X-T" {
		t.Errorf("names are trimmed: %+v", sec)
	}
	for name, mutate := range map[string]func(*SecurityConfig){
		"zero session":      func(c *SecurityConfig) { c.SessionTTL = 0 },
		"negative remember": func(c *SecurityConfig) { c.RememberTTL = -1 },
		"zero challenge":    func(c *SecurityConfig) { c.ChallengeTTL = 0 },
		"zero reset":        func(c *SecurityConfig) { c.ResetTTL = 0 },
		"blank cookie":      func(c *SecurityConfig) { c.Cookie = " " },
		"blank csrf cookie": func(c *SecurityConfig) { c.CSRFCookie = "" },
		"blank csrf header": func(c *SecurityConfig) { c.CSRFHeader = "" },
		"negative ip rate":  func(c *SecurityConfig) { c.IPRateLimit = -1 },
		"negative account":  func(c *SecurityConfig) { c.AccountRateLimit = -1 },
	} {
		c := good
		mutate(&c)
		if _, err := decodeSecurity(c); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// TestSecuritySectionFromFile covers what decodeSecurity cannot: the TOML keys
// and the error naming the offending one.
func TestSecuritySectionFromFile(t *testing.T) {
	load := func(body string) (*config.Config, error) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return config.Open(path)
	}

	f, err := load("[security]\nsession_ttl = 30\n")
	if err != nil {
		t.Fatal(err)
	}
	sec := config.Of(f, SecuritySection)
	if err := f.Finish(); err != nil {
		t.Fatal(err)
	}
	if got := sec.Get(); got.SessionTTL != 30*time.Minute || got.RememberTTL != 7*24*time.Hour {
		t.Errorf("file value and default: %+v", got)
	}

	f, err = load("[security]\nremember_ttl = 0\n")
	if err != nil {
		t.Fatal(err)
	}
	config.Of(f, SecuritySection)
	if err := f.Finish(); err == nil || !strings.Contains(err.Error(), "remember_ttl") {
		t.Errorf("zero lifetime: %v", err)
	}
}

package auth

import (
	"testing"
	"time"
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
	padded := good
	padded.Cookie, padded.CSRFHeader = "  sid ", " X-T "
	if sec, _ := decodeSecurity(padded); sec.SessionCookie != "sid" || sec.CSRFHeader != "X-T" {
		t.Errorf("names are trimmed: %+v", sec)
	}
	for name, mutate := range map[string]func(*SecurityConfig){
		"zero session":      func(c *SecurityConfig) { c.SessionTTL = 0 },
		"negative remember": func(c *SecurityConfig) { c.RememberTTL = -1 },
		"zero challenge":    func(c *SecurityConfig) { c.ChallengeTTL = 0 },
		"blank cookie":      func(c *SecurityConfig) { c.Cookie = " " },
		"blank csrf cookie": func(c *SecurityConfig) { c.CSRFCookie = "" },
		"blank csrf header": func(c *SecurityConfig) { c.CSRFHeader = "" },
	} {
		c := good
		mutate(&c)
		if _, err := decodeSecurity(c); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

package auth

import (
	"errors"
	"strings"
	"time"

	"savvy-go/internal/config"
)

// Security is how long sign-ins last and what the session cookies are called.
type Security struct {
	// SessionTTL and RememberTTL are how long a session lasts without and with "remember me".
	SessionTTL, RememberTTL time.Duration
	// ChallengeTTL is how long a second-factor or passkey challenge can be answered.
	ChallengeTTL  time.Duration
	SessionCookie string
	CSRFCookie    string
	CSRFHeader    string
}

// SecuritySettings is the [security] table of the config file.
type SecuritySettings struct {
	SessionTTL   int    `toml:"session_ttl" comment:"Lifetimes in minutes."`
	RememberTTL  int    `toml:"remember_ttl"`
	ChallengeTTL int    `toml:"challenge_ttl"`
	Cookie       string `toml:"session_cookie" comment:"Cookie and header names."`
	CSRFCookie   string `toml:"csrf_cookie"`
	CSRFHeader   string `toml:"csrf_header"`
}

// Defaults are the values of the keys the file lacks.
func (SecuritySettings) Defaults() SecuritySettings {
	return SecuritySettings{
		SessionTTL:   60 * 24,
		RememberTTL:  60 * 24 * 7,
		ChallengeTTL: 5,
		Cookie:       "svy_session",
		CSRFCookie:   "svy_csrf",
		CSRFHeader:   "X-CSRF-Token",
	}
}

// SecuritySection is where the config file keeps SecuritySettings.
var SecuritySection = config.Section[SecuritySettings, Security]{Name: "security", Decode: decodeSecurity}

// decodeSecurity refuses a lifetime that is not positive and a name that is empty.
func decodeSecurity(c SecuritySettings) (Security, error) {
	switch {
	case c.SessionTTL <= 0:
		return Security{}, errors.New("session_ttl must be a positive number of minutes")
	case c.RememberTTL <= 0:
		return Security{}, errors.New("remember_ttl must be a positive number of minutes")
	case c.ChallengeTTL <= 0:
		return Security{}, errors.New("challenge_ttl must be a positive number of minutes")
	case strings.TrimSpace(c.Cookie) == "":
		return Security{}, errors.New("session_cookie is empty")
	case strings.TrimSpace(c.CSRFCookie) == "":
		return Security{}, errors.New("csrf_cookie is empty")
	case strings.TrimSpace(c.CSRFHeader) == "":
		return Security{}, errors.New("csrf_header is empty")
	}
	return Security{
		SessionTTL:    time.Duration(c.SessionTTL) * time.Minute,
		RememberTTL:   time.Duration(c.RememberTTL) * time.Minute,
		ChallengeTTL:  time.Duration(c.ChallengeTTL) * time.Minute,
		SessionCookie: strings.TrimSpace(c.Cookie),
		CSRFCookie:    strings.TrimSpace(c.CSRFCookie),
		CSRFHeader:    strings.TrimSpace(c.CSRFHeader),
	}, nil
}

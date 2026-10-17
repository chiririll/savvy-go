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
	ChallengeTTL time.Duration
	// ResetTTL is how long a link to reset the password of an active account works. The link
	// that sets the first password of an invited account is not bound by it.
	ResetTTL      time.Duration
	SessionCookie string
	CSRFCookie    string
	CSRFHeader    string
	// IPRateLimit and AccountRateLimit are the sign-in attempts a minute one address and one
	// account may make. 0 is no limit.
	IPRateLimit, AccountRateLimit int
}

// SecurityConfig is the [security] table of the config file.
type SecurityConfig struct {
	SessionTTL   int    `toml:"session_ttl" comment:"Lifetimes in minutes."`
	RememberTTL  int    `toml:"remember_ttl"`
	ChallengeTTL int    `toml:"challenge_ttl"`
	ResetTTL     int    `toml:"reset_ttl"`
	Cookie       string `toml:"session_cookie" comment:"Cookie and header names."`
	CSRFCookie   string `toml:"csrf_cookie"`
	CSRFHeader   string `toml:"csrf_header"`

	IPRateLimit      int `toml:"ip_rate_limit" comment:"Sign-in attempts per minute from one address and against one account. 0 is no limit."`
	AccountRateLimit int `toml:"account_rate_limit"`
}

// Defaults are the values of the keys the file lacks.
func (SecurityConfig) Defaults() SecurityConfig {
	return SecurityConfig{
		SessionTTL:   60 * 24,
		RememberTTL:  60 * 24 * 7,
		ChallengeTTL: 5,
		ResetTTL:     60,
		Cookie:       "svy_session",
		CSRFCookie:   "svy_csrf",
		CSRFHeader:   "X-CSRF-Token",

		IPRateLimit:      10,
		AccountRateLimit: 5,
	}
}

// SecuritySection is where the config file keeps SecurityConfig.
var SecuritySection = config.Section[SecurityConfig, Security]{Name: "security", Decode: decodeSecurity}

// decodeSecurity refuses a lifetime that is not positive and a name that is empty.
func decodeSecurity(c SecurityConfig) (Security, error) {
	switch {
	case c.SessionTTL <= 0:
		return Security{}, errors.New("session_ttl must be a positive number of minutes")
	case c.RememberTTL <= 0:
		return Security{}, errors.New("remember_ttl must be a positive number of minutes")
	case c.ChallengeTTL <= 0:
		return Security{}, errors.New("challenge_ttl must be a positive number of minutes")
	case c.ResetTTL <= 0:
		return Security{}, errors.New("reset_ttl must be a positive number of minutes")
	case strings.TrimSpace(c.Cookie) == "":
		return Security{}, errors.New("session_cookie is empty")
	case strings.TrimSpace(c.CSRFCookie) == "":
		return Security{}, errors.New("csrf_cookie is empty")
	case strings.TrimSpace(c.CSRFHeader) == "":
		return Security{}, errors.New("csrf_header is empty")
	case c.IPRateLimit < 0:
		return Security{}, errors.New("ip_rate_limit must be zero or a positive number of attempts a minute")
	case c.AccountRateLimit < 0:
		return Security{}, errors.New("account_rate_limit must be zero or a positive number of attempts a minute")
	}
	return Security{
		SessionTTL:    time.Duration(c.SessionTTL) * time.Minute,
		RememberTTL:   time.Duration(c.RememberTTL) * time.Minute,
		ChallengeTTL:  time.Duration(c.ChallengeTTL) * time.Minute,
		ResetTTL:      time.Duration(c.ResetTTL) * time.Minute,
		SessionCookie: strings.TrimSpace(c.Cookie),
		CSRFCookie:    strings.TrimSpace(c.CSRFCookie),
		CSRFHeader:    strings.TrimSpace(c.CSRFHeader),

		IPRateLimit:      c.IPRateLimit,
		AccountRateLimit: c.AccountRateLimit,
	}, nil
}

package auth

import (
	"net/http"
	"time"

	"github.com/chiririll/savvy-plus/internal/config"
)

func SessionCookieName(cfg config.Config, secure bool) string {
	if secure {
		return "__Host-" + cfg.SessionCookie
	}
	return cfg.SessionCookie
}

func ReadToken(r *http.Request, cfg config.Config) string {
	if c, err := r.Cookie("__Host-" + cfg.SessionCookie); err == nil && c.Value != "" {
		return c.Value
	}
	if c, err := r.Cookie(cfg.SessionCookie); err == nil {
		return c.Value
	}
	return ""
}

func IsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

func SetSessionCookies(w http.ResponseWriter, r *http.Request, cfg config.Config, token, csrf string, remember bool) {
	secure := IsSecure(r)
	var expires time.Time
	maxAge := 0
	if remember {
		expires = time.Now().Add(cfg.RememberTTL)
		maxAge = int(cfg.RememberTTL.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName(cfg, secure),
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.CSRFCookie,
		Value:    csrf,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookies(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	secure := IsSecure(r)
	past := time.Unix(0, 0)
	forget := func(name string, sec, httpOnly bool) {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  past,
			MaxAge:   -1,
			Secure:   sec,
			HttpOnly: httpOnly,
			SameSite: http.SameSiteLaxMode,
		})
	}
	forget("__Host-"+cfg.SessionCookie, secure, true)
	forget(cfg.SessionCookie, false, true)
	forget(cfg.CSRFCookie, secure, false)
}

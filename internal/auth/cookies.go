package auth

import (
	"net/http"
	"time"
)

// SessionCookieName is the cookie that holds the session token; the __Host- prefix is used
// over HTTPS.
func (sec Security) SessionCookieName(secure bool) string {
	if secure {
		return "__Host-" + sec.SessionCookie
	}
	return sec.SessionCookie
}

// ReadToken is the session token the request carries, or "". Over HTTPS only the __Host-
// cookie counts, over HTTP only the plain one: a plain cookie planted by a sibling domain or
// over HTTP never signs anyone in on an HTTPS instance.
func (sec Security) ReadToken(r *http.Request, secure bool) string {
	if c, err := r.Cookie(sec.SessionCookieName(secure)); err == nil {
		return c.Value
	}
	return ""
}

// SetSessionCookies sends the session and CSRF cookies. remember makes them outlive the browser session.
func (sec Security) SetSessionCookies(w http.ResponseWriter, secure bool, token, csrf string, remember bool) {
	var expires time.Time
	maxAge := 0
	if remember {
		expires = time.Now().Add(sec.RememberTTL)
		maxAge = int(sec.RememberTTL.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sec.SessionCookieName(secure),
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     sec.CSRFCookie,
		Value:    csrf,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookies expires the session and CSRF cookies.
func (sec Security) ClearSessionCookies(w http.ResponseWriter, secure bool) {
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
	forget("__Host-"+sec.SessionCookie, secure, true)
	forget(sec.SessionCookie, false, true)
	forget(sec.CSRFCookie, secure, false)
}

// Package guard holds the middleware that decides who may reach a route:
// sign-in, CSRF, roles in the instance and in a space, and API token grants.
package guard

import (
	"context"
	"net/http"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx"
)

// Guard is built once by the server from its own services.
type Guard struct {
	AppURL    func() string // the public URL of the instance, empty when unset
	Security  auth.Security
	Sessions  auth.Sessions
	APITokens auth.APITokens
	Space     SpaceDeps
	// EmailRequired reports whether the instance holds back unverified users.
	EmailRequired func(context.Context) bool
}

// bearerToken returns the raw Authorization: Bearer value and whether the header was present.
func bearerToken(r *http.Request) (string, bool) {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if h == "" {
		return "", false
	}
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:]), true
	}
	return "", true
}

// RequireSession authenticates via Authorization: Bearer (API token) when the header is
// present, otherwise via the session cookie. A bad bearer never falls back to the cookie.
func (g *Guard) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if raw, present := bearerToken(r); present {
			t, err := g.APITokens.Resolve(r.Context(), raw)
			if err != nil || t == nil || t.User == nil {
				httpx.WriteMessage(w, http.StatusUnauthorized, "Unauthenticated.")
				return
			}
			ctx := httpx.WithToken(httpx.WithUser(r.Context(), t.User), t)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		token := g.Security.ReadToken(r)
		sess, err := g.Sessions.Resolve(r.Context(), token)
		if err != nil || sess == nil || sess.User == nil {
			httpx.WriteMessage(w, http.StatusUnauthorized, "Unauthenticated.")
			return
		}
		_ = g.Sessions.Touch(r.Context(), sess)
		ctx := httpx.WithSession(httpx.WithUser(r.Context(), sess.User), sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (g *Guard) RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if httpx.Token(r) != nil {
			next.ServeHTTP(w, r)
			return
		}
		sess := httpx.Session(r)
		header := r.Header.Get(g.Security.CSRFHeader)
		if sess == nil || header == "" || !secureCompare(sess.CSRF, header) {
			httpx.WriteMessage(w, 419, "CSRF token mismatch.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (g *Guard) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := httpx.User(r)
		if u == nil || !u.IsAdmin() || httpx.Token(r) != nil {
			httpx.WriteMessage(w, http.StatusForbidden, "Forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SessionOnly rejects API-token requests for credential and instance-level endpoints.
func (g *Guard) SessionOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httpx.Token(r) != nil {
			httpx.WriteMessage(w, http.StatusForbidden, "Not available for API tokens.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// NeedAppURL gates features that hand absolute URLs or origins to browsers and
// identity providers (SSO, passkeys). Like Gitea's ROOT_URL, nothing is guessed
// from request headers: without APP_URL these features are simply unavailable.
func (g *Guard) NeedAppURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.AppURL() == "" {
			httpx.WriteMessage(w, http.StatusServiceUnavailable, "APP_URL is not configured.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// verificationExempt are the routes an unverified user may still call while
// verification is required: leaving, and asking for a new link.
var verificationExempt = map[string]bool{
	"/api/auth/logout":              true,
	"/api/auth/verify-email/resend": true,
}

// RequireVerifiedEmail holds back, while the instance requires it, every
// signed-in user whose address is not confirmed. It covers API tokens, so a
// token cannot do what its owner could not.
func (g *Guard) RequireVerifiedEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := httpx.User(r)
		if u != nil && !u.EmailVerified && !verificationExempt[r.URL.Path] && g.EmailRequired(r.Context()) {
			httpx.WriteCoded(w, http.StatusForbidden, "Confirm your email address to continue.", "email_unverified")
			return
		}
		next.ServeHTTP(w, r)
	})
}

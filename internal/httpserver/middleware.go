package httpserver

import (
	"context"
	"net/http"
	"strings"

	"savvy-go/internal/auth"
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxSession
	ctxAPIToken
)

func userFrom(r *http.Request) *auth.User {
	u, _ := r.Context().Value(ctxUser).(*auth.User)
	return u
}

func sessionFrom(r *http.Request) *auth.Session {
	s, _ := r.Context().Value(ctxSession).(*auth.Session)
	return s
}

func apiTokenFrom(r *http.Request) *auth.APIToken {
	t, _ := r.Context().Value(ctxAPIToken).(*auth.APIToken)
	return t
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

// requireSession authenticates via Authorization: Bearer (API token) when the header is
// present, otherwise via the session cookie. A bad bearer never falls back to the cookie.
func (s *Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if raw, present := bearerToken(r); present {
			t, err := s.apiTokens.Resolve(r.Context(), raw)
			if err != nil || t == nil || t.User == nil {
				writeMessage(w, http.StatusUnauthorized, "Unauthenticated.")
				return
			}
			ctx := context.WithValue(r.Context(), ctxUser, t.User)
			ctx = context.WithValue(ctx, ctxAPIToken, t)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		token := auth.ReadToken(r, s.cfg)
		sess, err := s.sessions.Resolve(r.Context(), token)
		if err != nil || sess == nil || sess.User == nil {
			writeMessage(w, http.StatusUnauthorized, "Unauthenticated.")
			return
		}
		_ = s.sessions.Touch(r.Context(), sess)
		ctx := context.WithValue(r.Context(), ctxUser, sess.User)
		ctx = context.WithValue(ctx, ctxSession, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if apiTokenFrom(r) != nil {
			next.ServeHTTP(w, r)
			return
		}
		sess := sessionFrom(r)
		header := r.Header.Get(s.cfg.CSRFHeader)
		if sess == nil || header == "" || !secureCompare(sess.CSRF, header) {
			writeMessage(w, 419, "CSRF token mismatch.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r)
		if u == nil || !u.IsAdmin() || apiTokenFrom(r) != nil {
			writeMessage(w, http.StatusForbidden, "Forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// sessionOnly rejects API-token requests for credential and instance-level endpoints.
func (s *Server) sessionOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiTokenFrom(r) != nil {
			writeMessage(w, http.StatusForbidden, "Not available for API tokens.")
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

func looksLikeEmail(s string) bool {
	s = strings.TrimSpace(s)
	at := strings.IndexByte(s, '@')
	return at > 0 && at < len(s)-3 && strings.Contains(s[at:], ".")
}

func locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if loc := strings.TrimSpace(r.Header.Get("X-Locale")); loc != "" {
			w.Header().Set("Content-Language", loc)
		}
		next.ServeHTTP(w, r)
	})
}

// noFraming forbids showing any page inside another site's frame, so a hidden
// iframe cannot put its own button over ours (clickjacking), notably over
// "Allow access" on the OAuth consent page. Nothing here is meant to be embedded.
func noFraming(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func noStoreAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// needAppURL gates features that hand absolute URLs or origins to browsers and
// identity providers (SSO, passkeys). Like Gitea's ROOT_URL, nothing is guessed
// from request headers: without APP_URL these features are simply unavailable.
func (s *Server) needAppURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.appURL(r.Context()) == "" {
			writeMessage(w, http.StatusServiceUnavailable, "APP_URL is not configured.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// appURL is the public URL of the instance, empty when unset.
func (s *Server) appURL(ctx context.Context) string {
	return s.cfg.URL()
}

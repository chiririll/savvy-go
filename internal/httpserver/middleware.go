package httpserver

import (
	"net/http"
	"strings"
)

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

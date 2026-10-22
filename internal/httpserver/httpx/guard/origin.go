package guard

import (
	"mime"
	"net/http"
	"net/url"
	"strings"

	"savvy-go/internal/httpserver/httpx"
)

// PublicOrigin protects the public POSTs (sign-in, 2FA, password links) that no session
// cookie guards, so RequireCSRF cannot. Without it a cross-site <form enctype="text/plain">
// could sign the victim in to the attacker's account. For every unsafe method it
//   - rejects a cross-site request: Sec-Fetch-Site must be same-origin (or none) when the
//     browser sends it, otherwise Origin must be the origin of app_url (of the request's own
//     Host when app_url is unset), and
//   - requires a JSON body, which a plain HTML form cannot send.
//
// Requests without Origin and Sec-Fetch-Site (curl, scripts) pass: they are not browsers
// riding on someone's ambient authority.
func (g *Guard) PublicOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		// An identity provider posts its SAML response here from its own site, and
		// the response is signed.
		if isSSOCallback(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !g.sameOrigin(r) {
			httpx.WriteMessage(w, http.StatusForbidden, "Cross-site request refused.")
			return
		}
		if r.ContentLength != 0 && !allowedBodyType(r) {
			httpx.WriteMessage(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isSSOCallback(path string) bool {
	return strings.HasPrefix(path, "/api/auth/sso/") && strings.HasSuffix(path, "/acs")
}

// uploadPaths take a file, not JSON.
func allowedBodyType(r *http.Request) bool {
	p := r.URL.Path
	if p == "/api/onboarding/restore" || strings.HasPrefix(p, "/api/uploads/") {
		return true
	}
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mt == "application/json"
}

func (g *Guard) sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" {
		return site == "same-origin" || site == "none"
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	want := g.Cfg.AppURL()
	if want == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		want = scheme + "://" + r.Host
	}
	a, err1 := url.Parse(origin)
	b, err2 := url.Parse(want)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

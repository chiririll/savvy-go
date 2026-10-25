package guard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"savvy-go/internal/appconfig"
)

func TestPublicOrigin(t *testing.T) {
	g := &Guard{Cfg: appconfig.Config{}.WithAppURL("https://savvy.example")}
	h := g.PublicOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	const j = "application/json"
	cases := []struct {
		name, method, path, ctype string
		hdr                       map[string]string
		want                      int
	}{
		{"same-origin json", "POST", "/api/auth/login", j, map[string]string{"Origin": "https://savvy.example"}, 204},
		{"json with charset", "POST", "/api/auth/login", j + "; charset=utf-8", nil, 204},
		{"no headers (script)", "POST", "/api/auth/login", j, nil, 204},
		{"text/plain form", "POST", "/api/auth/login", "text/plain", nil, 415},
		{"no content type", "POST", "/api/auth/login", "", nil, 415},
		{"foreign origin", "POST", "/api/auth/login", j, map[string]string{"Origin": "https://evil.example"}, 403},
		{"null origin", "POST", "/api/auth/login", j, map[string]string{"Origin": "null"}, 403},
		{"cross-site fetch", "POST", "/api/auth/login", j, map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
		{"same-site other origin", "POST", "/api/auth/login", j, map[string]string{"Sec-Fetch-Site": "same-site"}, 403},
		{"same-origin fetch", "POST", "/api/auth/login", j, map[string]string{"Sec-Fetch-Site": "same-origin"}, 204},
		{"GET untouched", "GET", "/api/auth/status", "", map[string]string{"Origin": "https://evil.example"}, 204},
		{
			"SAML ACS cross-site form",
			"POST",
			"/api/auth/sso/okta/acs",
			"application/x-www-form-urlencoded",
			map[string]string{"Origin": "https://idp.example", "Sec-Fetch-Site": "cross-site"},
			204,
		},
		{"restore multipart", "POST", "/api/onboarding/restore", "multipart/form-data; boundary=x", nil, 204},
		{
			"restore multipart foreign",
			"POST",
			"/api/onboarding/restore",
			"multipart/form-data; boundary=x",
			map[string]string{"Origin": "https://evil.example"},
			403,
		},
		{"upload part", "PUT", "/api/uploads/1/parts/1", "application/octet-stream", nil, 204},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, strings.NewReader("{}"))
		if c.ctype != "" {
			r.Header.Set("Content-Type", c.ctype)
		}
		for k, v := range c.hdr {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: got %d, want %d", c.name, w.Code, c.want)
		}
	}
}

func TestPublicOriginFallsBackToHost(t *testing.T) {
	g := &Guard{}
	h := g.PublicOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for origin, want := range map[string]int{"http://example.com": 204, "http://evil.example": 403} {
		r := httptest.NewRequest("POST", "http://example.com/api/auth/login", strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Errorf("%s: got %d, want %d", origin, w.Code, want)
		}
	}
}

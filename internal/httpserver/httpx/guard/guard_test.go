package guard

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"savvy-go/internal/appconfig"
)

func TestSecureComesFromAppURLNotHeaders(t *testing.T) {
	plain := httptest.NewRequest("GET", "/", nil)
	forwarded := httptest.NewRequest("GET", "/", nil)
	forwarded.Header.Set("X-Forwarded-Proto", "https")
	tlsReq := httptest.NewRequest("GET", "/", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	cases := []struct {
		name   string
		appURL string
		r      *http.Request
		want   bool
	}{
		{"https app_url", "https://savvy.example", plain, true},
		{"HTTPS in capitals", "HTTPS://savvy.example", plain, true},
		{"http app_url ignores forwarded header", "http://savvy.example", forwarded, false},
		{"http app_url ignores TLS", "http://savvy.example", tlsReq, false},
		{"no app_url ignores forwarded header", "", forwarded, false},
		{"no app_url uses TLS", "", tlsReq, true},
		{"no app_url, plain", "", plain, false},
	}
	for _, c := range cases {
		g := &Guard{Cfg: appconfig.Config{}.WithAppURL(c.appURL)}
		if got := g.Secure(c.r); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

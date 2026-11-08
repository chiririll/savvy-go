package safehttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestBlockedAddr(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", "10.0.0.5", "172.16.3.1", "192.168.1.1",
		"169.254.169.254", "100.64.0.1", "0.0.0.0", "fe80::1", "fc00::1",
		"::ffff:127.0.0.1", "224.0.0.1",
	}
	for _, s := range blocked {
		if !blockedAddr(netip.MustParseAddr(s)) {
			t.Errorf("%s must be blocked", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		if blockedAddr(netip.MustParseAddr(s)) {
			t.Errorf("%s must be allowed", s)
		}
	}
}

func TestFetchRejectsNonHTTPS(t *testing.T) {
	f := New()
	for _, u := range []string{"http://example.com/x", "ftp://example.com/x", "https://user:pw@example.com/x", "https:///x", "not a url"} {
		if _, _, err := f.Fetch(context.Background(), u); err == nil {
			t.Errorf("%q must be rejected", u)
		}
	}
}

func TestFetchRejectsPrivateLiteral(t *testing.T) {
	f := New()
	_, _, err := f.Fetch(context.Background(), "https://127.0.0.1/x")
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("want blocked error, got %v", err)
	}
}

// serve starts a TLS server and returns a client that may reach its loopback address.
func serve(t *testing.T, h http.HandlerFunc) (*Client, string) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	t.Cleanup(srv.Close)
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	c := newClient(func(a netip.Addr) bool { return !a.IsLoopback() }, &tls.Config{RootCAs: pool})
	return c, srv.URL + "/client.json"
}

func TestFetchReturnsBodyAndCacheControl(t *testing.T) {
	c, u := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	body, cc, err := c.Fetch(context.Background(), u)
	if err != nil || string(body) != `{"ok":true}` || cc != "max-age=60" {
		t.Fatalf("body=%q cc=%q err=%v", body, cc, err)
	}
}

func TestFetchRefusesRedirects(t *testing.T) {
	c, u := serve(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://169.254.169.254/latest/meta-data", http.StatusFound)
	})
	if _, _, err := c.Fetch(context.Background(), u); err == nil {
		t.Fatal("a redirect must not be followed or accepted")
	}
}

func TestFetchRefusesErrorStatus(t *testing.T) {
	c, u := serve(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusNotFound) })
	if _, _, err := c.Fetch(context.Background(), u); err == nil {
		t.Fatal("a 404 must fail")
	}
}

func TestFetchEnforcesSizeCap(t *testing.T) {
	big := strings.Repeat("a", MaxBody+1)
	c, u := serve(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(big)) })
	if _, _, err := c.Fetch(context.Background(), u); err == nil {
		t.Fatal("an oversized document must fail")
	}
	exact := strings.Repeat("a", MaxBody)
	c, u = serve(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(exact)) })
	if body, _, err := c.Fetch(context.Background(), u); err != nil || len(body) != MaxBody {
		t.Fatalf("a document at the cap must work: %d, %v", len(body), err)
	}
}

func TestFetchRespectsContext(t *testing.T) {
	c, u := serve(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, _, err := c.Fetch(ctx, u); err == nil {
		t.Fatal("a cancelled context must stop the fetch")
	}
}

func TestDefaultClientCannotReachLoopbackServer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	// Via a host name, so only the dial-time check can stop it.
	u := strings.Replace(srv.URL, "127.0.0.1", "localhost", 1) + "/x"
	_, _, err := New().Fetch(context.Background(), u)
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("want blocked at dial time, got %v", err)
	}
}

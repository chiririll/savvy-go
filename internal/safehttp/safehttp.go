// Package safehttp fetches small documents from untrusted URLs without
// letting the caller reach internal networks (SSRF). https only, no
// redirects, addresses checked after DNS resolution, bounded time and size.
package safehttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
	"time"
)

const (
	// MaxBody is the largest document accepted (CIMD recommends 5 KB).
	MaxBody = 5 << 10
	timeout = 5 * time.Second
)

// Client downloads documents. Core packages declare their own small
// interface for it (they must not import net/http) so tests can inject a fake.
type Client struct {
	http    *http.Client
	blocked func(netip.Addr) bool
}

var errBlocked = errors.New("safehttp: blocked address")

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// blockedAddr reports whether a is not a public unicast address.
func blockedAddr(a netip.Addr) bool {
	a = a.Unmap()
	return !a.IsValid() || a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() ||
		a.IsLinkLocalMulticast() || a.IsMulticast() || a.IsUnspecified() ||
		a.IsInterfaceLocalMulticast() || cgnat.Contains(a)
}

// control runs after DNS resolution, on the address actually dialed.
func control(_, address string, _ syscall.RawConn) error { return checkDial(address, blockedAddr) }

func checkDial(address string, blocked func(netip.Addr) bool) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	a, err := netip.ParseAddr(host)
	if err != nil || blocked(a) {
		return fmt.Errorf("%w: %s", errBlocked, host)
	}
	return nil
}

// New returns a Client with the SSRF-safe dialer.
func New() *Client { return newClient(blockedAddr, nil) }

// newClient lets tests allow the loopback address of an httptest server and
// trust its certificate; production always uses New.
func newClient(blocked func(netip.Addr) bool, tlsConfig *tls.Config) *Client {
	d := &net.Dialer{
		Timeout: timeout,
		Control: func(_, address string, _ syscall.RawConn) error { return checkDial(address, blocked) },
	}
	return &Client{blocked: blocked, http: &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:         d.DialContext,
			TLSClientConfig:     tlsConfig,
			Proxy:               nil, // a proxy would dial on our behalf, skipping the check
			TLSHandshakeTimeout: timeout,
			DisableKeepAlives:   true,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// Fetch GETs rawURL. Redirects are not followed and count as failure.
func (c *Client) Fetch(ctx context.Context, rawURL string) (body []byte, cacheControl string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Hostname() == "" || u.User != nil {
		return nil, "", errors.New("safehttp: need an https URL without credentials")
	}
	// Literal IPs skip DNS, so the dialer sees them too, but fail early.
	if a, err := netip.ParseAddr(u.Hostname()); err == nil && c.blocked(a) {
		return nil, "", fmt.Errorf("%w: %s", errBlocked, u.Hostname())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("safehttp: status %d", resp.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(resp.Body, MaxBody+1))
	if err != nil {
		return nil, "", err
	}
	if len(body) > MaxBody {
		return nil, "", errors.New("safehttp: document too large")
	}
	return body, resp.Header.Get("Cache-Control"), nil
}

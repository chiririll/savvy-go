package oauth_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

const cimdURL = "https://app.example/oauth/client.json"

type fakeFetcher struct {
	docs  map[string]string
	cache string
	err   error
	calls int
}

func (f *fakeFetcher) Fetch(_ context.Context, u string) ([]byte, string, error) {
	f.calls++
	if f.err != nil {
		return nil, "", f.err
	}
	d, ok := f.docs[u]
	if !ok {
		return nil, "", errors.New("404")
	}
	return []byte(d), f.cache, nil
}

func doc(id, extra string) string {
	return fmt.Sprintf(`{"client_id":%q,"client_name":"App","redirect_uris":[%q]%s}`, id, redirect, extra)
}

func cimdEnv(t *testing.T, allow ...string) (*env, *fakeFetcher) {
	t.Helper()
	e := newEnv(t)
	f := &fakeFetcher{docs: map[string]string{cimdURL: doc(cimdURL, "")}}
	e.svc.Fetcher = f
	for _, h := range allow {
		if err := e.svc.AddCIMDHost(context.Background(), h); err != nil {
			t.Fatal(err)
		}
	}
	return e, f
}

func TestCIMDDeniedWithoutAllowlist(t *testing.T) {
	e, f := cimdEnv(t)
	_, err := e.svc.CheckAuthorize(context.Background(), cimdURL, redirect)
	wantCode(t, err, "invalid_client")
	if f.calls != 0 {
		t.Fatal("must not fetch from a host that is not allowed")
	}
}

func TestCIMDAllowedHostWorksEndToEnd(t *testing.T) {
	e, _ := cimdEnv(t, "app.example")
	ctx := context.Background()
	c, err := e.svc.CheckAuthorize(ctx, cimdURL, redirect)
	if err != nil || c.Name != "App" || c.Kind != "cimd" {
		t.Fatalf("client = %+v, err = %v", c, err)
	}
	code := e.code(t, cimdURL, vOK)
	if _, err := e.svc.Exchange(ctx, cimdURL, code, vOK, redirect); err != nil {
		t.Fatal(err)
	}
}

func TestCIMDRejectsBadDocuments(t *testing.T) {
	cases := map[string]string{
		"id mismatch":    doc("https://other.example/x", ""),
		"secret method":  doc(cimdURL, `,"token_endpoint_auth_method":"client_secret_basic"`),
		"bad redirect":   fmt.Sprintf(`{"client_id":%q,"redirect_uris":["http://evil.example/cb"]}`, cimdURL),
		"no redirects":   fmt.Sprintf(`{"client_id":%q,"redirect_uris":[]}`, cimdURL),
		"not json":       `nope`,
		"has secret key": doc(cimdURL, `,"client_secret":"x"`),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			e, f := cimdEnv(t, "app.example")
			f.docs[cimdURL] = body
			_, err := e.svc.CheckAuthorize(context.Background(), cimdURL, redirect)
			wantCode(t, err, "invalid_client")
		})
	}
}

func TestCIMDFetchFailureIsInvalidClient(t *testing.T) {
	e, f := cimdEnv(t, "app.example")
	f.err = errors.New("boom")
	_, err := e.svc.CheckAuthorize(context.Background(), cimdURL, redirect)
	wantCode(t, err, "invalid_client")
}

func TestCIMDRejectsUnsafeURLs(t *testing.T) {
	for _, u := range []string{
		"http://app.example/c.json", "https://app.example", "https://app.example/c.json#x",
		"https://u:p@app.example/c.json", "https://app.example/a/../c.json", "https://127.0.0.1/c.json",
	} {
		e, f := cimdEnv(t, "app.example")
		f.docs[u] = doc(u, "")
		_, err := e.svc.CheckAuthorize(context.Background(), u, redirect)
		if err == nil {
			t.Errorf("%q must be rejected", u)
		}
	}
}

func TestCIMDCacheAndExpiry(t *testing.T) {
	e, f := cimdEnv(t, "app.example")
	f.cache = "max-age=600"
	ctx := context.Background()
	_, _ = e.svc.CheckAuthorize(ctx, cimdURL, redirect)
	_, _ = e.svc.CheckAuthorize(ctx, cimdURL, redirect)
	if f.calls != 1 {
		t.Fatalf("calls = %d, want cache hit", f.calls)
	}
	e.now = e.now.Add(11 * time.Minute)
	f.docs[cimdURL] = fmt.Sprintf(`{"client_id":%q,"client_name":"Renamed","redirect_uris":[%q]}`, cimdURL, redirect)
	c, err := e.svc.CheckAuthorize(ctx, cimdURL, redirect)
	if err != nil || f.calls != 2 || c.Name != "Renamed" {
		t.Fatalf("calls=%d client=%+v err=%v", f.calls, c, err)
	}
}

func TestCIMDCacheIsBounded(t *testing.T) {
	e, f := cimdEnv(t, "app.example")
	f.cache = "max-age=1"
	ctx := context.Background()
	_, _ = e.svc.CheckAuthorize(ctx, cimdURL, redirect)
	e.now = e.now.Add(10 * time.Second)
	_, _ = e.svc.CheckAuthorize(ctx, cimdURL, redirect)
	if f.calls != 1 {
		t.Fatalf("max-age below the floor must still be cached, calls = %d", f.calls)
	}
}

func TestRemovingHostStopsExistingCIMDClient(t *testing.T) {
	e, _ := cimdEnv(t, "app.example")
	ctx := context.Background()
	code := e.code(t, cimdURL, vOK)
	if _, err := e.svc.RemoveCIMDHost(ctx, "app.example"); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Exchange(ctx, cimdURL, code, vOK, redirect)
	wantCode(t, err, "invalid_grant")
}

func TestAddCIMDHostNormalises(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	for _, bad := range []string{"", "https://x.example", "x.example/path", "*.x.example", "x y", "127.0.0.1", "x.example:8443"} {
		if err := e.svc.AddCIMDHost(ctx, bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
	if err := e.svc.AddCIMDHost(ctx, " Claude.AI "); err != nil {
		t.Fatal(err)
	}
	if hs, _ := e.svc.CIMDHosts(ctx); len(hs) != 1 || hs[0] != "claude.ai" {
		t.Fatalf("hosts = %v", hs)
	}
}

func TestManualClientSecret(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	c, secret, err := e.svc.CreateClient(ctx, "Mine", []string{redirect}, true, e.user)
	if err != nil || secret == "" || c.SecretHash == secret {
		t.Fatalf("client=%+v secret=%q err=%v", c, secret, err)
	}
	if _, err := e.svc.AuthenticateClient(ctx, c.ID, ""); err == nil {
		t.Fatal("missing secret must fail")
	}
	if _, err := e.svc.AuthenticateClient(ctx, c.ID, secret+"x"); err == nil {
		t.Fatal("wrong secret must fail")
	}
	if _, err := e.svc.AuthenticateClient(ctx, c.ID, secret); err != nil {
		t.Fatal(err)
	}
}

func TestPublicManualClientNeedsNoSecret(t *testing.T) {
	e := newEnv(t)
	c, secret, err := e.svc.CreateClient(context.Background(), "Pub", []string{redirect}, false, e.user)
	if err != nil || secret != "" {
		t.Fatalf("secret=%q err=%v", secret, err)
	}
	if _, err := e.svc.AuthenticateClient(context.Background(), c.ID, ""); err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticateUnknownClient(t *testing.T) {
	e := newEnv(t)
	_, err := e.svc.AuthenticateClient(context.Background(), "nope", "")
	wantCode(t, err, "invalid_client")
}

func TestDCRClientIgnoresPresentedSecret(t *testing.T) {
	e := newEnv(t)
	id := e.client(t)
	if _, err := e.svc.AuthenticateClient(context.Background(), id, ""); err != nil {
		t.Fatal(err)
	}
}

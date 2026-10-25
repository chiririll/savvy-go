//go:build integration

package httpserver

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestPublicPostsRefuseCrossSiteForms(t *testing.T) {
	a := newTestApp(t)
	post := func(ctype, origin string) int {
		req, err := http.NewRequest(
			"POST",
			a.srv.URL+"/api/auth/login",
			strings.NewReader(`{"email":"a@b.c","password":"x"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", ctype)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := a.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode == http.StatusForbidden && strings.Contains(string(raw), "Cross-site") {
			return -1 // refused by the origin check
		}
		return res.StatusCode
	}
	if got := post("text/plain", ""); got != http.StatusUnsupportedMediaType {
		t.Errorf("text/plain: %d", got)
	}
	if got := post("application/json", "https://evil.example"); got != -1 {
		t.Errorf("foreign origin: %d", got)
	}
	if got := post("application/json", testURL); got == http.StatusUnsupportedMediaType || got == -1 {
		t.Errorf("same-origin JSON refused: %d (origin %s)", got, testURL)
	}
}

// OAuth endpoints sit outside /api and take cross-origin form posts by design.
func TestOAuthTokenAcceptsCrossOrigin(t *testing.T) {
	a := newTestApp(t)
	req, _ := http.NewRequest(
		"POST",
		a.srv.URL+"/oauth/token",
		strings.NewReader(url.Values{"grant_type": {"authorization_code"}}.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://claude.ai")
	res, err := a.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusUnsupportedMediaType {
		t.Errorf("token endpoint refused cross-origin: %d", res.StatusCode)
	}
}

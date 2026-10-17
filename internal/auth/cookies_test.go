package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadTokenTrustsOnlyNameTransport(t *testing.T) {
	sec := Security{SessionCookie: "svy_session"}
	req := func(names ...string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		for _, n := range names {
			r.AddCookie(&http.Cookie{Name: n, Value: n + "-token"})
		}
		return r
	}
	if got := sec.ReadToken(req("svy_session"), true); got != "" {
		t.Errorf("HTTPS read a plain cookie: %q", got)
	}
	if got := sec.ReadToken(req("__Host-svy_session"), true); got != "__Host-svy_session-token" {
		t.Errorf("HTTPS host cookie: %q", got)
	}
	if got := sec.ReadToken(req("svy_session", "__Host-svy_session"), true); got != "__Host-svy_session-token" {
		t.Errorf("HTTPS with both: %q", got)
	}
	if got := sec.ReadToken(req("__Host-svy_session"), false); got != "" {
		t.Errorf("HTTP read a host cookie: %q", got)
	}
	if got := sec.ReadToken(req("svy_session"), false); got != "svy_session-token" {
		t.Errorf("HTTP plain cookie: %q", got)
	}
}

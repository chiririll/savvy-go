package sso

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAuthorizeURLCarriesPKCEUnguessableState(t *testing.T) {
	e := newSSOEnv(t, nil)
	loc, err := e.svc.AuthorizeURL(e.ctx, *e.p, "/spaces/3")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(loc)
	q := u.Query()
	if !strings.HasPrefix(loc, e.idp.srv.URL+"/authorize?") {
		t.Fatalf("goes to the discovered endpoint: %s", loc)
	}
	if q.Get("client_id") != "cid" || q.Get("response_type") != "code" ||
		q.Get("redirect_uri") != "https://app.test/api/auth/sso/test/callback" {
		t.Errorf("query %v", q)
	}
	if q.Get("scope") != "openid profile email" || q.Get("code_challenge_method") != "S256" || len(q.Get("state")) < 32 {
		t.Errorf("scope, PKCE or state: %v", q)
	}

	st, err := e.svc.ConsumeState(e.ctx, q.Get("state"))
	if err != nil || st == nil {
		t.Fatalf("state: %v, %v", st, err)
	}
	sum := sha256.Sum256([]byte(st.CodeVerifier))
	if q.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(sum[:]) {
		t.Error("the challenge is not the SHA-256 of the stored verifier")
	}
	if st.RedirectAfter != "/spaces/3" || st.ProviderID != e.p.ID {
		t.Errorf("state %+v", st)
	}
	if again, _ := e.svc.ConsumeState(e.ctx, q.Get("state")); again != nil {
		t.Error("a state can be used once")
	}
	if none, _ := e.svc.ConsumeState(e.ctx, "nope"); none != nil {
		t.Error("an unknown state resolved")
	}
}

func TestRedirectAfterLoginStaysThisSite(t *testing.T) {
	for in, want := range map[string]string{
		"/spaces/1":               "/spaces/1",
		"  /spaces/1  ":           "/spaces/1",
		"":                        "",
		"//evil.example/x":        "",
		"https://evil.example/":   "",
		"javascript://x":          "",
		"spaces/1":                "",
		"/redirect?to=http://x/y": "",
	} {
		if got := sanitizeRedirect(in); got != want {
			t.Errorf("sanitizeRedirect(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCallbackReturnsTheVerifiedIdentity(t *testing.T) {
	e := newSSOEnv(t, nil)
	claims := e.idp.idToken("sub-1", "ann@test.com", true)
	claims["groups"] = []string{"staff", "admins"}
	e.idp.next = claims
	e.idp.userinfo = map[string]any{"email": "forged@evil.example", "nickname": "annie"}

	id, err := e.login(t)
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "sub-1" || id.Email != "ann@test.com" || !id.EmailVerified || id.Name != "Ann" {
		t.Errorf("identity %+v", id)
	}
	if len(id.Groups) != 2 || id.Groups[1] != "admins" {
		t.Errorf("groups %v", id.Groups)
	}
	// The signed token wins; the userinfo only fills in what the token lacks.
	if id.Raw["email"] != "ann@test.com" || id.Raw["nickname"] != "annie" {
		t.Errorf("claims %v", id.Raw)
	}
	if f := e.idp.lastForm; f.Get("code") != "abc" || f.Get("client_secret") != "shh" || f.Get("code_verifier") == "" ||
		f.Get("grant_type") != "authorization_code" {
		t.Errorf("token request %v", f)
	}
}

func TestCallbackRefusesWhatNotValidLogin(t *testing.T) {
	e := newSSOEnv(t, nil)
	good := func() map[string]any { return e.idp.idToken("sub-1", "ann@test.com", true) }

	tests := []struct {
		name  string
		setup func()
		query func(state string) url.Values
		code  string
	}{
		{"the provider reports an error", func() {}, func(string) url.Values {
			return url.Values{"error": {"access_denied"}, "error_description": {"User said no"}}
		}, "idp_error"},
		{"no code", func() {}, func(s string) url.Values { return url.Values{"state": {s}} }, "invalid_request"},
		{"no state", func() {}, func(string) url.Values { return url.Values{"code": {"abc"}} }, "invalid_request"},
		{
			"an invented state",
			func() {},
			func(string) url.Values { return url.Values{"code": {"abc"}, "state": {"forged"}} },
			"invalid_state",
		},
		{"no ID token", func() { e.idp.tokenBody = map[string]any{"access_token": "at"} }, nil, "id_token_missing"},
		{"a token for another application", func() {
			c := good()
			c["aud"] = "someone-else"
			e.idp.next, e.idp.tokenBody = c, nil
		}, nil, "id_token_invalid_or_audience"},
		{"a token from another issuer", func() {
			c := good()
			c["iss"] = "https://evil.example"
			e.idp.next, e.idp.tokenBody = c, nil
		}, nil, "issuer_mismatch"},
		{"an expired token", func() {
			c := good()
			c["exp"] = time.Now().Add(-time.Hour).Unix()
			c["iat"] = time.Now().Add(-2 * time.Hour).Unix()
			e.idp.next, e.idp.tokenBody = c, nil
		}, nil, "id_token_invalid"},
	}
	for _, tc := range tests {
		e.idp.tokenBody = nil
		e.idp.next = good()
		tc.setup()
		loc, err := e.svc.AuthorizeURL(e.ctx, *e.p, "")
		if err != nil {
			t.Fatal(err)
		}
		u, _ := url.Parse(loc)
		q := url.Values{"code": {"abc"}, "state": {u.Query().Get("state")}}
		if tc.query != nil {
			q = tc.query(u.Query().Get("state"))
		}
		_, err = e.svc.HandleCallback(e.ctx, *e.p, q)
		got := ssoCode(err)
		if got == "" || (tc.code != "id_token_invalid_or_audience" && got != tc.code) {
			t.Errorf("%s: error %q, want %q", tc.name, got, tc.code)
		}
		if tc.code == "id_token_invalid_or_audience" && got != "audience_mismatch" {
			t.Errorf("%s: error %q, want audience_mismatch", tc.name, got)
		}
	}
}

func TestStateBelongsProviderThatIssuedIt(t *testing.T) {
	e := newSSOEnv(t, nil)
	e.idp.next = e.idp.idToken("sub-1", "ann@test.com", true)
	loc, _ := e.svc.AuthorizeURL(e.ctx, *e.p, "")
	u, _ := url.Parse(loc)

	other := *e.p
	other.ID += 100
	_, err := e.svc.HandleCallback(e.ctx, other, url.Values{"code": {"abc"}, "state": {u.Query().Get("state")}})
	if ssoCode(err) != "invalid_state" {
		t.Errorf("a state replayed against another provider: %q", ssoCode(err))
	}
	// The state is spent even by a failed attempt.
	_, err = e.svc.HandleCallback(e.ctx, *e.p, url.Values{"code": {"abc"}, "state": {u.Query().Get("state")}})
	if ssoCode(err) != "invalid_state" {
		t.Errorf("a spent state worked again: %q", ssoCode(err))
	}
}

func TestDiscoveryErrorsAreReportedNotLeaked(t *testing.T) {
	e := newSSOEnv(t, nil)
	e.idp.srv.Close()
	_, err := e.svc.AuthorizeURL(e.ctx, *e.p, "")
	if ssoCode(err) != "discovery_failed" {
		t.Errorf("error %q", ssoCode(err))
	}
	if err := e.svc.TestConnection(e.ctx, *e.p); ssoCode(err) != "discovery_failed" {
		t.Errorf("TestConnection %q", ssoCode(err))
	}
}

func TestTestConnectionReachesTheDiscoveryDocument(t *testing.T) {
	e := newSSOEnv(t, nil)
	if err := e.svc.TestConnection(e.ctx, *e.p); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.TestConnection(e.ctx, IdentityProvider{Protocol: "oauth2"}); err != nil {
		t.Errorf("a protocol without a check is fine: %v", err)
	}
	noURL := *e.p
	noURL.Config = map[string]any{}
	if ssoCode(e.svc.TestConnection(e.ctx, noURL)) != "missing_field" {
		t.Error("a provider without a discovery URL")
	}
}

func TestScopes(t *testing.T) {
	tests := []struct {
		name string
		p    IdentityProvider
		want string
	}{
		{"defaults", IdentityProvider{}, "openid profile email"},
		{"okta also asks for groups", IdentityProvider{Preset: "okta"}, "openid profile email groups"},
		{
			"a string",
			IdentityProvider{Config: map[string]any{"scopes": "email  offline_access"}},
			"openid email offline_access",
		},
		{
			"a list, openid never twice",
			IdentityProvider{Config: map[string]any{"scopes": []any{"openid", "groups", " ", "groups"}}},
			"openid groups",
		},
	}
	for _, tc := range tests {
		if got := strings.Join(oidcScopes(tc.p), " "); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestPKCEPairs(t *testing.T) {
	v1, c1 := pkce()
	v2, _ := pkce()
	sum := sha256.Sum256([]byte(v1))
	if v1 == v2 || len(v1) < 43 || c1 != base64.RawURLEncoding.EncodeToString(sum[:]) {
		t.Errorf("verifier %q challenge %q", v1, c1)
	}
}

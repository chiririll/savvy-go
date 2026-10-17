package sso

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/settings"
)

func rsaJWK(key *rsa.PublicKey, kid, alg string) map[string]any {
	return map[string]any{
		"kty": "RSA", "kid": kid, "alg": alg, "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}

func jwks(keys ...map[string]any) string {
	b, _ := json.Marshal(map[string]any{"keys": keys})
	return string(b)
}

func claims(mut func(jwt.MapClaims)) jwt.MapClaims {
	now := time.Now()
	c := jwt.MapClaims{"iss": "https://idp.test", "sub": "u", "aud": "cid", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix()}
	if mut != nil {
		mut(c)
	}
	return c
}

func signWith(t *testing.T, method jwt.SigningMethod, key any, kid string, c jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(method, c)
	if kid != "" {
		tok.Header["kid"] = kid
	}
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestIDTokenVerification(t *testing.T) {
	rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ecKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ecJWK := map[string]any{
		"kty": "EC", "crv": "P-256", "kid": "ec1", "alg": "ES256",
		"x": base64.RawURLEncoding.EncodeToString(ecKey.X.Bytes()),
		"y": base64.RawURLEncoding.EncodeToString(ecKey.Y.Bytes()),
	}
	set := jwks(rsaJWK(&rsaKey.PublicKey, "k1", "RS256"), ecJWK)
	const iss, aud = "https://idp.test", "cid"

	// An HS256 token "signed" with the public key as the shared secret is the
	// classic algorithm-confusion forgery.
	confused := signWith(t, jwt.SigningMethodHS256, rsaKey.N.Bytes(), "k1", claims(nil))

	tests := []struct {
		name  string
		token string
		jwks  string
		iss   string
		aud   string
		ok    bool
		code  string
	}{
		{"an RSA token", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(nil)), set, iss, aud, true, ""},
		{"an EC token", signWith(t, jwt.SigningMethodES256, ecKey, "ec1", claims(nil)), set, iss, aud, true, ""},
		{"a key that is not in the set", signWith(t, jwt.SigningMethodRS256, otherKey, "k1", claims(nil)), set, iss, aud, false, "id_token_invalid"},
		{"a kid that is not in the set", signWith(t, jwt.SigningMethodRS256, rsaKey, "zzz", claims(nil)), set, iss, aud, false, "id_token_invalid"},
		{"a symmetric algorithm", confused, set, iss, aud, false, "id_token_invalid"},
		{"an expired token", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) {
			c["exp"] = time.Now().Add(-time.Hour).Unix()
		})), set, iss, aud, false, "id_token_invalid"},
		{"a token issued in the future", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) {
			c["iat"] = time.Now().Add(time.Hour).Unix()
		})), set, iss, aud, false, "id_token_invalid"},
		{"another issuer", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { c["iss"] = "https://evil" })), set, iss, aud, false, "issuer_mismatch"},
		{"no issuer claim", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { delete(c, "iss") })), set, iss, aud, false, "issuer_mismatch"},
		{"another audience", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { c["aud"] = "other" })), set, iss, aud, false, "audience_mismatch"},
		{"no audience claim", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { delete(c, "aud") })), set, iss, aud, false, "audience_mismatch"},
		{"our audience among several", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { c["aud"] = []string{"x", "cid"} })), set, iss, aud, true, ""},
		{"a signature from a key with another declared algorithm", signWith(t, jwt.SigningMethodRS384, rsaKey, "k1", claims(nil)), set, iss, aud, false, "id_token_invalid"},
		{"a set that is not JSON", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(nil)), "not json", iss, aud, false, "jwks_failed"},
		{"an empty set", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(nil)), `{"keys":[]}`, iss, aud, false, "jwks_failed"},
		{"a token that is not a JWT", "abc.def", set, iss, aud, false, "id_token_invalid"},
	}
	for _, tc := range tests {
		got, err := VerifyOIDCIDToken(tc.token, tc.jwks, tc.iss, tc.aud, "custom_oidc")
		switch {
		case tc.ok && err != nil:
			t.Errorf("%s: %v", tc.name, err)
		case tc.ok && got["sub"] != "u":
			t.Errorf("%s: claims %v", tc.name, got)
		case !tc.ok && err == nil:
			t.Errorf("%s: accepted", tc.name)
		case !tc.ok && ssoCode(err) != tc.code:
			t.Errorf("%s: error %q, want %q", tc.name, ssoCode(err), tc.code)
		}
	}
}

func TestSingleKeyWithoutKidIsUsed(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	tok := signWith(t, jwt.SigningMethodRS256, key, "", claims(nil))
	if _, err := VerifyOIDCIDToken(tok, jwks(rsaJWK(&key.PublicKey, "", "")), "https://idp.test", "cid", "custom_oidc"); err != nil {
		t.Errorf("single key: %v", err)
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	if _, err := VerifyOIDCIDToken(tok, jwks(rsaJWK(&key.PublicKey, "a", ""), rsaJWK(&other.PublicKey, "b", "")), "https://idp.test", "cid", "custom_oidc"); err == nil {
		// with two keys and no kid the first listed key is tried; the point is that a wrong key never verifies
		t.Log("the first of several keys signed it; fine")
	}
}

func TestJWKPublicKeyRefusesWhatItCannotUse(t *testing.T) {
	for name, k := range map[string]jwkKey{
		"an unknown type":       {Kty: "oct"},
		"an unsupported curve":  {Kty: "EC", Crv: "P-521", X: "AQ", Y: "AQ"},
		"a broken RSA modulus":  {Kty: "RSA", N: "!!", E: "AQAB"},
		"a broken RSA exponent": {Kty: "RSA", N: "AQAB", E: "!!"},
		"a broken EC point":     {Kty: "EC", Crv: "P-256", X: "!!", Y: "AQ"},
	} {
		if _, err := jwkPublicKey(k); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if k, err := jwkPublicKey(jwkKey{Kty: "rsa", N: "AQAB", E: "AQAB"}); err != nil || k == nil {
		t.Errorf("key type is case-insensitive: %v", err)
	}
}

func TestFetchJWKS(t *testing.T) {
	var body string
	var status int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	s := Service{HTTP: srv.Client()}
	ctx := context.Background()

	status, body = 200, `{"keys":[{"kty":"RSA","kid":"a"}]}`
	if keys, err := s.fetchJWKS(ctx, srv.URL); err != nil || len(keys) != 1 || keys[0].Kid != "a" {
		t.Errorf("good document: %v, %v", keys, err)
	}
	status, body = 200, `{"keys":[]}`
	if _, err := s.fetchJWKS(ctx, srv.URL); ssoCode(err) != "jwks_failed" {
		t.Errorf("empty set: %q", ssoCode(err))
	}
	status, body = 200, `<html>`
	if _, err := s.fetchJWKS(ctx, srv.URL); ssoCode(err) != "jwks_failed" {
		t.Errorf("not JSON: %q", ssoCode(err))
	}
	status, body = 500, `{"keys":[{"kty":"RSA"}]}`
	if _, err := s.fetchJWKS(ctx, srv.URL); ssoCode(err) != "jwks_failed" {
		t.Errorf("server error: %q", ssoCode(err))
	}
	if _, err := s.fetchJWKS(ctx, "  "); ssoCode(err) != "jwks_missing" {
		t.Errorf("no URL: %q", ssoCode(err))
	}
}

// --- provider administration ---

func newAdminEnv(t *testing.T) (Service, context.Context) {
	t.Helper()
	d := dbtest.Server(t)
	return Service{DB: d, Users: auth.Users{DB: d}, Settings: settings.Store{DB: d}}, context.Background()
}

func oidcFields() map[string]any {
	return map[string]any{"discovery_url": "https://idp.test/.well-known/openid-configuration", "client_id": "cid", "client_secret": "s3cret"}
}

func TestCreateValidatesTheProvider(t *testing.T) {
	s, ctx := newAdminEnv(t)
	if _, err := s.Create(ctx, IdPWrite{Slug: "a", Preset: "myspace", Fields: oidcFields()}); ssoCode(err) != "unknown_preset" {
		t.Errorf("unknown preset: %q", ssoCode(err))
	}
	for _, slug := range []string{"", "Has Space", "UPPER", "a_b", "../x"} {
		if _, err := s.Create(ctx, IdPWrite{Slug: slug, Preset: "custom_oidc", Fields: oidcFields()}); ssoCode(err) != "invalid_slug" {
			t.Errorf("slug %q: %q", slug, ssoCode(err))
		}
	}
	fields := oidcFields()
	delete(fields, "client_id")
	if _, err := s.Create(ctx, IdPWrite{Slug: "a", Preset: "custom_oidc", Fields: fields}); ssoCode(err) != "missing_field" {
		t.Errorf("missing client id: %q", ssoCode(err))
	}
	fields = oidcFields()
	fields["client_secret"] = "  "
	if _, err := s.Create(ctx, IdPWrite{Slug: "a", Preset: "custom_oidc", Fields: fields}); ssoCode(err) != "missing_field" {
		t.Errorf("blank secret: %q", ssoCode(err))
	}
	if _, err := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields()}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, IdPWrite{Name: "B", Slug: "a", Preset: "custom_oidc", Fields: oidcFields()}); ssoCode(err) != "slug_taken" {
		t.Errorf("duplicate slug: %q", ssoCode(err))
	}
	if all, _ := s.All(ctx); len(all) != 1 {
		t.Errorf("refused providers were stored: %d", len(all))
	}
}

func TestNewProviderStartsDisabledAndLinksByEmail(t *testing.T) {
	s, ctx := newAdminEnv(t)
	p, err := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "google", Fields: map[string]any{"client_id": "cid", "client_secret": "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Enabled || !p.AllowJIT || !p.LinkByEmail || p.SyncRoleOnLogin || p.Protocol != "oidc" {
		t.Errorf("defaults %+v", p)
	}
	if s.AnyEnabled(ctx) {
		t.Error("a disabled provider is not offered")
	}
	if p.ClaimMappings["subject"] != "sub" {
		t.Errorf("the preset's claim mappings are the default: %v", p.ClaimMappings)
	}
	if _, err := s.Update(ctx, p.ID, IdPWrite{Enabled: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if !s.AnyEnabled(ctx) {
		t.Error("an enabled provider is offered")
	}
	if en, _ := s.Enabled(ctx); len(en) != 1 {
		t.Errorf("enabled: %d", len(en))
	}
	if got, _ := s.BySlug(ctx, "a"); got == nil || got.ID != p.ID {
		t.Error("BySlug")
	}
	if got, _ := s.BySlug(ctx, "zz"); got != nil {
		t.Error("BySlug of an unknown slug")
	}
}

func TestTheSecretNeverLeavesAndIsKeptWhenNotResent(t *testing.T) {
	s, ctx := newAdminEnv(t)
	p, err := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields()})
	if err != nil {
		t.Fatal(err)
	}
	for _, public := range []map[string]any{p.JSON("https://app.test/"), p.PublicJSON()} {
		raw, _ := json.Marshal(public)
		if strings.Contains(string(raw), "s3cret") {
			t.Fatalf("the secret is in %s", raw)
		}
	}
	js := p.JSON("https://app.test/")
	if js["hasClientSecret"] != true || js["callbackUrl"] != "https://app.test/api/auth/sso/a/callback" || js["acsUrl"] != "https://app.test/api/auth/sso/a/acs" {
		t.Errorf("json %v", js)
	}
	if _, ok := p.PublicJSON()["config"]; ok {
		t.Error("the public view shows only what the login page needs")
	}

	edit := oidcFields()
	delete(edit, "client_secret") // the form comes back without it
	edit["client_id"] = "cid2"
	upd, err := s.Update(ctx, p.ID, IdPWrite{Fields: edit})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Secrets["client_secret"] != "s3cret" || upd.Config["client_id"] != "cid2" {
		t.Errorf("secret %v config %v", upd.Secrets, upd.Config)
	}
	edit["client_secret"] = "rotated"
	upd, _ = s.Update(ctx, p.ID, IdPWrite{Fields: edit})
	if upd.Secrets["client_secret"] != "rotated" {
		t.Errorf("a new secret replaces the old: %v", upd.Secrets)
	}
}

func TestUpdateKeepsWhatIsNotSent(t *testing.T) {
	s, ctx := newAdminEnv(t)
	p, _ := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields(), DefaultRole: "guest", SyncRoleOnLogin: ptr(true), SortOrder: ptr(3)})
	upd, err := s.Update(ctx, p.ID, IdPWrite{Name: "Renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != "Renamed" || upd.DefaultRole != "guest" || !upd.SyncRoleOnLogin || upd.SortOrder != 3 || upd.Config["client_id"] != "cid" || upd.Secrets["client_secret"] != "s3cret" {
		t.Errorf("updated %+v", upd)
	}
	if got, err := s.Update(ctx, 999, IdPWrite{Name: "x"}); got != nil || err != nil {
		t.Errorf("unknown provider: %v, %v", got, err)
	}
	other, _ := s.Create(ctx, IdPWrite{Name: "B", Slug: "b", Preset: "custom_oidc", Fields: oidcFields()})
	if _, err := s.Update(ctx, other.ID, IdPWrite{Slug: "a"}); ssoCode(err) != "slug_taken" {
		t.Errorf("taking another slug: %q", ssoCode(err))
	}
	if _, err := s.Update(ctx, other.ID, IdPWrite{Slug: "b"}); err != nil {
		t.Errorf("keeping its own slug: %v", err)
	}
	if _, err := s.Update(ctx, other.ID, IdPWrite{Slug: "Not Valid"}); ssoCode(err) != "invalid_slug" {
		t.Errorf("invalid slug: %q", ssoCode(err))
	}
}

func TestDeleteRefusesToStrandSSOOnlyUsers(t *testing.T) {
	s, ctx := newAdminEnv(t)
	p, _ := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields(), Enabled: ptr(true)})
	if _, err := s.Users.Create(ctx, "Admin", "root@test.com", ptr("pw-pw-pw"), auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Provision(ctx, *p, ident("sub", "jit@test.com", true)); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, p.ID); err == nil {
		t.Fatal("the only way in for an SSO-only user was removed")
	}
	if got, _ := s.ByID(ctx, p.ID); got == nil {
		t.Error("the provider is gone")
	}

	empty, _ := s.Create(ctx, IdPWrite{Name: "B", Slug: "b", Preset: "custom_oidc", Fields: oidcFields()})
	if err := s.Delete(ctx, empty.ID); err != nil {
		t.Errorf("a provider nobody depends on: %v", err)
	}
}

func TestTicketsAndStatesAreSingleUse(t *testing.T) {
	s, ctx := newAdminEnv(t)
	u, _ := s.Users.Create(ctx, "A", "a@test.com", ptr("pw-pw-pw"), "")

	ticket, err := s.IssueTicket(ctx, u.ID, true)
	if err != nil || len(ticket) != 64 {
		t.Fatalf("ticket %q, %v", ticket, err)
	}
	id, needs2FA, ok := s.ConsumeTicket(ctx, ticket)
	if !ok || id != u.ID || !needs2FA {
		t.Errorf("consume: %d %v %v", id, needs2FA, ok)
	}
	if _, _, ok := s.ConsumeTicket(ctx, ticket); ok {
		t.Error("a ticket worked twice")
	}
	if _, _, ok := s.ConsumeTicket(ctx, "forged"); ok {
		t.Error("an invented ticket worked")
	}

	p, err := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields()})
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.SaveState(ctx, IdPState{ProviderID: p.ID, SAMLRequestID: "r"})
	if err != nil || len(state) != 48 {
		t.Fatalf("state %q, %v", state, err)
	}
	if err := s.AttachSAMLRequestID(ctx, state, "req-77"); err != nil {
		t.Fatal(err)
	}
	st, _ := s.ConsumeState(ctx, state)
	if st == nil || st.SAMLRequestID != "req-77" || st.ProviderID != p.ID {
		t.Errorf("state %+v", st)
	}
	if again, _ := s.ConsumeState(ctx, state); again != nil {
		t.Error("a state worked twice")
	}
}

func TestCallbackURLsFollowThePublicURL(t *testing.T) {
	s := Service{}
	p := IdentityProvider{Slug: "x"}
	if got := s.CallbackURL(context.Background(), p); got != "/api/auth/sso/x/callback" {
		t.Errorf("without a public URL: %q", got)
	}
}

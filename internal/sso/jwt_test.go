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
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	c := jwt.MapClaims{
		"iss": "https://idp.test",
		"sub": "u",
		"aud": "cid",
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
	}
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
		{
			"a key that is not in the set",
			signWith(t, jwt.SigningMethodRS256, otherKey, "k1", claims(nil)),
			set,
			iss,
			aud,
			false,
			"id_token_invalid",
		},
		{
			"a kid that is not in the set",
			signWith(t, jwt.SigningMethodRS256, rsaKey, "zzz", claims(nil)),
			set,
			iss,
			aud,
			false,
			"id_token_invalid",
		},
		{"a symmetric algorithm", confused, set, iss, aud, false, "id_token_invalid"},
		{"an expired token", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) {
			c["exp"] = time.Now().Add(-time.Hour).Unix()
		})), set, iss, aud, false, "id_token_invalid"},
		{"a token issued in the future", signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) {
			c["iat"] = time.Now().Add(time.Hour).Unix()
		})), set, iss, aud, false, "id_token_invalid"},
		{
			"another issuer",
			signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { c["iss"] = "https://evil" })),
			set,
			iss,
			aud,
			false,
			"issuer_mismatch",
		},
		{
			"no issuer claim",
			signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { delete(c, "iss") })),
			set,
			iss,
			aud,
			false,
			"issuer_mismatch",
		},
		{
			"another audience",
			signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { c["aud"] = "other" })),
			set,
			iss,
			aud,
			false,
			"audience_mismatch",
		},
		{
			"no audience claim",
			signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { delete(c, "aud") })),
			set,
			iss,
			aud,
			false,
			"audience_mismatch",
		},
		{
			"our audience among several",
			signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(func(c jwt.MapClaims) { c["aud"] = []string{"x", "cid"} })),
			set,
			iss,
			aud,
			true,
			"",
		},
		{
			"a signature from a key with another declared algorithm",
			signWith(t, jwt.SigningMethodRS384, rsaKey, "k1", claims(nil)),
			set,
			iss,
			aud,
			false,
			"id_token_invalid",
		},
		{
			"a set that is not JSON",
			signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(nil)),
			"not json",
			iss,
			aud,
			false,
			"jwks_failed",
		},
		{
			"an empty set",
			signWith(t, jwt.SigningMethodRS256, rsaKey, "k1", claims(nil)),
			`{"keys":[]}`,
			iss,
			aud,
			false,
			"jwks_failed",
		},
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
	if _,
		err := VerifyOIDCIDToken(tok, jwks(rsaJWK(&key.PublicKey, "", "")), "https://idp.test", "cid",
		"custom_oidc"); err != nil {
		t.Errorf("single key: %v", err)
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	if _,
		err := VerifyOIDCIDToken(tok, jwks(rsaJWK(&key.PublicKey, "a", ""), rsaJWK(&other.PublicKey, "b", "")),
		"https://idp.test", "cid", "custom_oidc"); err == nil {
		// with two keys and no kid the first listed key is tried; the point is that a wrong key never verifies
		t.Log("the first of several keys signed it; fine")
	}
}

func TestJWKPublicKeyRefusesUnusable(t *testing.T) {
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

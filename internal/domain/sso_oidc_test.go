package domain

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifyOIDCIDTokenAcceptsSignedAndRejectsForged(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks, err := json.Marshal(map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "https://idp.test", "sub": "user-1", "aud": "cid",
		"email": "a@test.com", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "k1"
	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := VerifyOIDCIDToken(signed, string(jwks), "https://idp.test", "cid", "custom_oidc")
	if err != nil || stringify(got["sub"]) != "user-1" {
		t.Fatalf("valid token %v %v", got, err)
	}

	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	forged, err := tok.SignedString(other)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyOIDCIDToken(forged, string(jwks), "https://idp.test", "cid", "custom_oidc"); err == nil {
		t.Fatal("forged token accepted")
	}

	parts := splitJWT(signed)
	unsigned := parts[0] + "." + parts[1] + "."
	if _, err := VerifyOIDCIDToken(unsigned, string(jwks), "https://idp.test", "cid", "custom_oidc"); err == nil {
		t.Fatal("unsigned token accepted")
	}
	if _, err := VerifyOIDCIDToken("", string(jwks), "https://idp.test", "cid", "custom_oidc"); err == nil {
		t.Fatal("empty token accepted")
	}
}

func splitJWT(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '.' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}

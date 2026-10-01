package domain

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type jwksDoc struct {
	Keys []jwkKey `json:"keys"`
}

func (s SSO) fetchJWKS(ctx context.Context, uri string) ([]jwkKey, error) {
	if strings.TrimSpace(uri) == "" {
		return nil, ssoErr("jwks_missing", "The identity provider did not advertise a JWKS URI.", 502)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client().Do(req)
	if err != nil {
		return nil, ssoErr("jwks_failed", "Unable to load JWKS.", 502)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, ssoErr("jwks_failed", "Unable to load JWKS.", 502)
	}
	var doc jwksDoc
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil || len(doc.Keys) == 0 {
		return nil, ssoErr("jwks_failed", "Invalid JWKS document.", 502)
	}
	return doc.Keys, nil
}

func jwkPublicKey(k jwkKey) (any, error) {
	switch strings.ToUpper(k.Kty) {
	case "RSA":
		n, err := b64u(k.N)
		if err != nil {
			return nil, err
		}
		e, err := b64u(k.E)
		if err != nil {
			return nil, err
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}, nil
	case "EC":
		if k.Crv != "P-256" {
			return nil, fmt.Errorf("unsupported curve %s", k.Crv)
		}
		x, err := b64u(k.X)
		if err != nil {
			return nil, err
		}
		y, err := b64u(k.Y)
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}, nil
	default:
		return nil, fmt.Errorf("unsupported kty %s", k.Kty)
	}
}

func b64u(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
}

// VerifyOIDCIDToken checks a compact JWT against a JWKS document.
func VerifyOIDCIDToken(rawToken, jwksJSON, issuer, audience, preset string) (map[string]any, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, ssoErr("id_token_missing", "The identity provider did not return an ID token.", 401)
	}
	var doc jwksDoc
	if err := json.Unmarshal([]byte(jwksJSON), &doc); err != nil || len(doc.Keys) == 0 {
		return nil, ssoErr("jwks_failed", "Invalid JWKS document.", 502)
	}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "PS256"}),
		jwt.WithLeeway(time.Minute),
		jwt.WithIssuedAt(),
	)
	tok, err := parser.Parse(rawToken, func(t *jwt.Token) (any, error) {
		alg, _ := t.Header["alg"].(string)
		if strings.EqualFold(alg, "none") || alg == "" {
			return nil, fmt.Errorf("rejected alg")
		}
		kid, _ := t.Header["kid"].(string)
		var match *jwkKey
		for i := range doc.Keys {
			k := &doc.Keys[i]
			if kid != "" && k.Kid != kid {
				continue
			}
			if k.Alg != "" && alg != "" && k.Alg != alg {
				continue
			}
			match = k
			break
		}
		if match == nil && kid == "" && len(doc.Keys) == 1 {
			match = &doc.Keys[0]
		}
		if match == nil {
			return nil, fmt.Errorf("no matching jwk")
		}
		return jwkPublicKey(*match)
	})
	if err != nil || tok == nil || !tok.Valid {
		return nil, ssoErr("id_token_invalid", "The ID token signature is invalid.", 401)
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ssoErr("id_token_invalid", "The ID token is invalid.", 401)
	}
	if issuer != "" && !MatchesIssuer(preset, issuer, claims) {
		return nil, ssoErr("issuer_mismatch", "The token issuer does not match the discovery document.", 401)
	}
	if audience != "" {
		if !audienceOK(claims, audience) {
			return nil, ssoErr("audience_mismatch", "The ID token audience does not match this application.", 401)
		}
	}
	out := map[string]any{}
	for k, v := range claims {
		out[k] = v
	}
	return out, nil
}

func audienceOK(claims jwt.MapClaims, want string) bool {
	raw, ok := claims["aud"]
	if !ok {
		return false
	}
	switch t := raw.(type) {
	case string:
		return t == want
	case []any:
		for _, v := range t {
			if stringify(v) == want {
				return true
			}
		}
	}
	return false
}

func (s SSO) verifyIDToken(ctx context.Context, disco *oidcDiscovery, p IdentityProvider, raw string) (map[string]any, error) {
	keys, err := s.fetchJWKS(ctx, disco.JWKSURI)
	if err != nil {
		return nil, err
	}
	doc, _ := json.Marshal(jwksDoc{Keys: keys})
	return VerifyOIDCIDToken(raw, string(doc), disco.Issuer, cfgString(p.Config, "client_id"), p.Preset)
}

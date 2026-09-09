package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (s SSO) CallbackURL(p IdentityProvider) string {
	return strings.TrimRight(s.AppURL, "/") + "/api/auth/sso/" + p.Slug + "/callback"
}

func (s SSO) ACSURL(p IdentityProvider) string {
	return strings.TrimRight(s.AppURL, "/") + "/api/auth/sso/" + p.Slug + "/acs"
}

func (s SSO) AuthorizeURL(ctx context.Context, p IdentityProvider, redirectAfter string) (string, error) {
	if p.Preset == "custom_saml" || p.Protocol == "saml" {
		return s.samlRedirect(ctx, p, redirectAfter)
	}
	if p.Preset == "github" {
		state, err := s.SaveState(ctx, IdPState{ProviderID: p.ID, RedirectAfter: sanitizeRedirect(redirectAfter)})
		if err != nil {
			return "", err
		}
		u, _ := url.Parse("https://github.com/login/oauth/authorize")
		q := u.Query()
		q.Set("client_id", cfgString(p.Config, "client_id"))
		q.Set("redirect_uri", s.CallbackURL(p))
		q.Set("response_type", "code")
		q.Set("scope", "read:user user:email")
		q.Set("state", state)
		u.RawQuery = q.Encode()
		return u.String(), nil
	}
	disco, err := s.discover(ctx, p)
	if err != nil {
		return "", err
	}
	verifier, challenge := pkce()
	state, err := s.SaveState(ctx, IdPState{
		ProviderID: p.ID, CodeVerifier: verifier, RedirectAfter: sanitizeRedirect(redirectAfter),
	})
	if err != nil {
		return "", err
	}
	scopes := oidcScopes(p)
	u, err := url.Parse(disco.AuthorizationEndpoint)
	if err != nil {
		return "", ssoErr("idp_error", "Invalid authorization endpoint.", 502)
	}
	q := u.Query()
	q.Set("client_id", cfgString(p.Config, "client_id"))
	q.Set("redirect_uri", s.CallbackURL(p))
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

func (s SSO) discover(ctx context.Context, p IdentityProvider) (*oidcDiscovery, error) {
	raw := PresetDiscoveryURL(p)
	if raw == "" {
		return nil, ssoErr("missing_field", "Discovery URL is not configured.", 422)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client().Do(req)
	if err != nil {
		return nil, ssoErr("discovery_failed", "Unable to reach the identity provider.", 502)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, ssoErr("discovery_failed", "Unable to reach the identity provider.", 502)
	}
	var d oidcDiscovery
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return nil, ssoErr("discovery_failed", "Invalid discovery document.", 502)
	}
	return &d, nil
}

func (s SSO) HandleCallback(ctx context.Context, p IdentityProvider, q url.Values) (NormalizedIdentity, error) {
	if errDesc := q.Get("error"); errDesc != "" {
		msg := q.Get("error_description")
		if msg == "" {
			msg = errDesc
		}
		return NormalizedIdentity{}, ssoErr("idp_error", msg, 401)
	}
	code, state := q.Get("code"), q.Get("state")
	if code == "" || state == "" {
		return NormalizedIdentity{}, ssoErr("invalid_request", "The SSO request is invalid.", 400)
	}
	st, err := s.ConsumeState(ctx, state)
	if err != nil || st == nil || st.ProviderID != p.ID {
		return NormalizedIdentity{}, ssoErr("invalid_state", "The SSO state is invalid or expired.", 401)
	}
	if p.Preset == "github" {
		return s.githubIdentity(ctx, p, code)
	}
	return s.oidcIdentity(ctx, p, code, st.CodeVerifier)
}

func (s SSO) githubIdentity(ctx context.Context, p IdentityProvider, code string) (NormalizedIdentity, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", s.CallbackURL(p))
	form.Set("client_id", cfgString(p.Config, "client_id"))
	form.Set("client_secret", secretString(p.Secrets, "client_secret"))
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := s.client().Do(req)
	if err != nil {
		return NormalizedIdentity{}, ssoErr("token_exchange_failed", "Token exchange failed.", 502)
	}
	defer res.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.NewDecoder(res.Body).Decode(&tok)
	if tok.AccessToken == "" {
		return NormalizedIdentity{}, ssoErr("token_exchange_failed", "Token exchange failed.", 502)
	}
	claims, err := s.getJSON(ctx, "https://api.github.com/user", tok.AccessToken)
	if err != nil {
		return NormalizedIdentity{}, ssoErr("userinfo_failed", "Unable to load the user profile.", 502)
	}
	email, verified := s.githubEmail(ctx, tok.AccessToken)
	id := extractIdentity(p, claims, map[string]any{"email": email, "email_verified": verified})
	if !verified || email == "" {
		id.Email = ""
		id.EmailVerified = false
	}
	return id, nil
}

func (s SSO) githubEmail(ctx context.Context, token string) (string, bool) {
	body, err := s.getJSONList(ctx, "https://api.github.com/user/emails", token)
	if err != nil {
		return "", false
	}
	for _, entry := range body {
		m, _ := entry.(map[string]any)
		if m == nil {
			continue
		}
		if asBoolAny(m["primary"]) && asBoolAny(m["verified"]) {
			email, _ := m["email"].(string)
			return email, true
		}
	}
	return "", false
}

func (s SSO) oidcIdentity(ctx context.Context, p IdentityProvider, code, verifier string) (NormalizedIdentity, error) {
	disco, err := s.discover(ctx, p)
	if err != nil {
		return NormalizedIdentity{}, err
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", s.CallbackURL(p))
	form.Set("client_id", cfgString(p.Config, "client_id"))
	form.Set("client_secret", secretString(p.Secrets, "client_secret"))
	if verifier != "" {
		form.Set("code_verifier", verifier)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, disco.TokenEndpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := s.client().Do(req)
	if err != nil {
		return NormalizedIdentity{}, ssoErr("token_exchange_failed", "Token exchange failed.", 502)
	}
	defer res.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	_ = json.NewDecoder(res.Body).Decode(&tok)
	claims := map[string]any{}
	if tok.IDToken != "" {
		if payload, err := jwtPayload(tok.IDToken); err == nil {
			claims = payload
			if disco.Issuer != "" && !MatchesIssuer(p.Preset, disco.Issuer, claims) {
				return NormalizedIdentity{}, ssoErr("issuer_mismatch", "The token issuer does not match the discovery document.", 401)
			}
		}
	}
	if tok.AccessToken != "" && disco.UserinfoEndpoint != "" {
		if extra, err := s.getJSON(ctx, disco.UserinfoEndpoint, tok.AccessToken); err == nil {
			for k, v := range extra {
				if _, exists := claims[k]; !exists {
					claims[k] = v
				}
			}
		}
	}
	if len(claims) == 0 {
		return NormalizedIdentity{}, ssoErr("userinfo_failed", "Unable to load the user profile.", 502)
	}
	return extractIdentity(p, claims, nil), nil
}

func extractIdentity(p IdentityProvider, claims map[string]any, overrides map[string]any) NormalizedIdentity {
	maps := p.ClaimMappings
	if maps == nil {
		if preset, ok := PresetByKey(p.Preset); ok {
			maps = preset.DefaultClaims
		}
	}
	get := func(key string) any {
		if overrides != nil {
			if v, ok := overrides[key]; ok {
				return v
			}
		}
		raw := maps[key]
		return claimValue(claims, raw)
	}
	id := NormalizedIdentity{Raw: claims}
	id.Subject = stringify(get("subject"))
	id.Email = stringify(get("email"))
	id.Name = stringify(get("name"))
	if v := get("email_verified"); v != nil {
		id.EmailVerified = asBoolAny(v)
	}
	if g := get("groups"); g != nil {
		if arr, ok := asAnySlice(g); ok {
			for _, v := range arr {
				id.Groups = append(id.Groups, stringify(v))
			}
		} else if s := stringify(g); s != "" {
			id.Groups = []string{s}
		}
	}
	return id
}

func claimValue(claims map[string]any, mapping any) any {
	switch t := mapping.(type) {
	case nil:
		return nil
	case string:
		return claims[t]
	case []any:
		for _, c := range t {
			key := stringify(c)
			if v, ok := claims[key]; ok && v != nil && stringify(v) != "" {
				return v
			}
		}
	}
	return nil
}

func (s SSO) getJSON(ctx context.Context, rawURL, token string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Savvy")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", res.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s SSO) getJSONList(ctx context.Context, rawURL, token string) ([]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Savvy")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var out []any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s SSO) TestConnection(ctx context.Context, p IdentityProvider) error {
	switch p.Protocol {
	case "oidc":
		_, err := s.discover(ctx, p)
		return err
	case "saml":
		_ = s.SamlMetadata(p)
		return nil
	default:
		return nil
	}
}

func pkce() (verifier, challenge string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	verifier = strings.TrimRight(base64.RawURLEncoding.EncodeToString(b), "=")
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

func jwtPayload(tok string) (map[string]any, error) {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("jwt")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, err
		}
	}
	var out map[string]any
	return out, json.Unmarshal(raw, &out)
}

func oidcScopes(p IdentityProvider) []string {
	seen := map[string]bool{"openid": true}
	out := []string{"openid"}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	if raw, ok := p.Config["scopes"]; ok {
		switch t := raw.(type) {
		case string:
			for _, p := range strings.Fields(t) {
				add(p)
			}
		case []any:
			for _, v := range t {
				add(stringify(v))
			}
		}
	}
	if len(out) == 1 {
		for _, s := range []string{"profile", "email"} {
			add(s)
		}
		if p.Preset == "okta" {
			add("groups")
		}
	}
	return out
}

func sanitizeRedirect(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "//") || strings.Contains(s, "://") {
		return ""
	}
	if !strings.HasPrefix(s, "/") {
		return ""
	}
	return s
}

func asBoolAny(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	default:
		return false
	}
}

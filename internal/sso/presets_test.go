package sso

import "testing"

func TestEntraIssuerMustMatchTenantToken(t *testing.T) {
	discovery := "https://login.microsoftonline.com/{tenantid}/v2.0"
	tenant := "abc-123"
	for _, c := range []struct {
		name   string
		issuer string
		want   bool
	}{
		{"issuer of the same tenant", "https://login.microsoftonline.com/abc-123/v2.0", true},
		{"issuer of another tenant", "https://login.microsoftonline.com/evil/v2.0", false},
	} {
		got := MatchesIssuer("entra", discovery, map[string]any{"iss": c.issuer, "tid": tenant})
		if got != c.want {
			t.Errorf("%s: MatchesIssuer = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOktaDiscoveryURLFollowsAuthorizationServer(t *testing.T) {
	p := IdentityProvider{Preset: "okta", Config: map[string]any{"domain": "example.okta.com"}}
	if got, want := PresetDiscoveryURL(p), "https://example.okta.com/.well-known/openid-configuration"; got != want {
		t.Errorf("org authorization server: %s, want %s", got, want)
	}
	p.Config["auth_server_id"] = "aus1a2b3c"
	if got, want := PresetDiscoveryURL(p),
		"https://example.okta.com/oauth2/aus1a2b3c/.well-known/openid-configuration"; got != want {
		t.Errorf("custom authorization server: %s, want %s", got, want)
	}
}

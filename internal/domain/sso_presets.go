package domain

import "strings"

type SSOField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Group       string `json:"group"`
	Placeholder string `json:"placeholder,omitempty"`
}

type SSOPreset struct {
	Key           string         `json:"key"`
	Label         string         `json:"label"`
	Protocol      string         `json:"protocol"`
	Fields        []SSOField     `json:"fields"`
	DefaultClaims map[string]any `json:"default_claim_mappings"`
}

func clientFields() []SSOField {
	return []SSOField{
		{Key: "client_id", Label: "Client ID", Type: "text", Required: true, Group: "config"},
		{Key: "client_secret", Label: "Client secret", Type: "password", Required: true, Secret: true, Group: "secrets"},
	}
}

func oidcClaims() map[string]any {
	return map[string]any{"subject": "sub", "email": "email", "email_verified": "email_verified", "name": "name", "groups": "groups"}
}

func SSOPresetCatalog() []SSOPreset {
	return []SSOPreset{
		{Key: "entra", Label: "Microsoft Entra ID", Protocol: "oidc", Fields: append([]SSOField{
			{Key: "tenant", Label: "Tenant", Type: "text", Required: true, Group: "config", Placeholder: "common / organizations / <tenant-guid>"},
		}, clientFields()...), DefaultClaims: map[string]any{
			"subject": "sub", "email": []any{"email", "preferred_username", "upn"}, "email_verified": "xms_edov", "name": "name", "groups": "groups",
		}},
		{Key: "github", Label: "GitHub", Protocol: "oauth2", Fields: clientFields(), DefaultClaims: map[string]any{
			"subject": "id", "email": "email", "name": "name", "groups": nil,
		}},
		{Key: "google", Label: "Google", Protocol: "oidc", Fields: clientFields(), DefaultClaims: map[string]any{
			"subject": "sub", "email": "email", "email_verified": "email_verified", "name": "name", "groups": nil,
		}},
		{Key: "okta", Label: "Okta", Protocol: "oidc", Fields: append([]SSOField{
			{Key: "domain", Label: "Domain", Type: "text", Required: true, Group: "config", Placeholder: "example.okta.com"},
			{Key: "auth_server_id", Label: "Authorization server ID", Type: "text", Group: "config", Placeholder: "default (leave blank for the org server)"},
		}, clientFields()...), DefaultClaims: oidcClaims()},
		{Key: "gitlab", Label: "GitLab", Protocol: "oidc", Fields: append([]SSOField{
			{Key: "base_url", Label: "GitLab URL", Type: "url", Required: true, Group: "config", Placeholder: "https://gitlab.com"},
		}, clientFields()...), DefaultClaims: map[string]any{
			"subject": "sub", "email": "email", "email_verified": "email_verified", "name": "name", "groups": "groups_direct",
		}},
		{Key: "keycloak", Label: "Keycloak", Protocol: "oidc", Fields: append([]SSOField{
			{Key: "base_url", Label: "Keycloak URL", Type: "url", Required: true, Group: "config", Placeholder: "https://kc.example.com"},
			{Key: "realm", Label: "Realm", Type: "text", Required: true, Group: "config"},
		}, clientFields()...), DefaultClaims: oidcClaims()},
		{Key: "authentik", Label: "Authentik", Protocol: "oidc", Fields: append([]SSOField{
			{Key: "base_url", Label: "Authentik URL", Type: "url", Required: true, Group: "config", Placeholder: "https://auth.example.com"},
			{Key: "app_slug", Label: "Application slug", Type: "text", Required: true, Group: "config"},
		}, clientFields()...), DefaultClaims: oidcClaims()},
		{Key: "custom_oidc", Label: "OpenID Connect", Protocol: "oidc", Fields: append([]SSOField{
			{Key: "discovery_url", Label: "Discovery URL", Type: "url", Required: true, Group: "config", Placeholder: "https://idp.example.com/.well-known/openid-configuration"},
			{Key: "scopes", Label: "Scopes", Type: "text", Group: "config", Placeholder: "openid profile email"},
		}, clientFields()...), DefaultClaims: oidcClaims()},
		{Key: "custom_saml", Label: "SAML 2.0", Protocol: "saml", Fields: []SSOField{
			{Key: "idp_entity_id", Label: "IdP entity ID", Type: "text", Required: true, Group: "config"},
			{Key: "idp_sso_url", Label: "IdP SSO URL", Type: "url", Required: true, Group: "config"},
			{Key: "idp_x509_cert", Label: "IdP X.509 certificate", Type: "textarea", Required: true, Group: "config"},
			{Key: "idp_x509_cert_standby", Label: "Standby certificate", Type: "textarea", Group: "config"},
		}, DefaultClaims: map[string]any{
			"subject": nil,
			"email":   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"name":    "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			"groups":  "http://schemas.xmlsoap.org/claims/Group",
		}},
	}
}

func PresetByKey(key string) (SSOPreset, bool) {
	for _, p := range SSOPresetCatalog() {
		if p.Key == key {
			return p, true
		}
	}
	return SSOPreset{}, false
}

func PresetDiscoveryURL(p IdentityProvider) string {
	switch p.Preset {
	case "google":
		return "https://accounts.google.com/.well-known/openid-configuration"
	case "entra":
		tenant := cfgString(p.Config, "tenant")
		if tenant == "" {
			tenant = "common"
		}
		return "https://login.microsoftonline.com/" + tenant + "/v2.0/.well-known/openid-configuration"
	case "okta":
		domain := cfgString(p.Config, "domain")
		if domain != "" && !hasScheme(domain) {
			domain = "https://" + domain
		}
		domain = strings.TrimRight(domain, "/")
		if id := cfgString(p.Config, "auth_server_id"); id != "" {
			return domain + "/oauth2/" + id + "/.well-known/openid-configuration"
		}
		return domain + "/.well-known/openid-configuration"
	case "keycloak":
		return strings.TrimRight(cfgString(p.Config, "base_url"), "/") + "/realms/" + cfgString(p.Config, "realm") + "/.well-known/openid-configuration"
	case "authentik":
		return strings.TrimRight(cfgString(p.Config, "base_url"), "/") + "/application/o/" + cfgString(p.Config, "app_slug") + "/.well-known/openid-configuration"
	case "gitlab":
		return strings.TrimRight(cfgString(p.Config, "base_url"), "/") + "/.well-known/openid-configuration"
	case "custom_oidc":
		return cfgString(p.Config, "discovery_url")
	default:
		return cfgString(p.Config, "discovery_url")
	}
}

func hasScheme(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func MatchesIssuer(preset, discoveryIssuer string, claims map[string]any) bool {
	actual, _ := claims["iss"].(string)
	if actual == "" {
		return false
	}
	if preset == "entra" && strings.Contains(discoveryIssuer, "{tenantid}") {
		tid, _ := claims["tid"].(string)
		if tid == "" {
			return false
		}
		discoveryIssuer = strings.ReplaceAll(discoveryIssuer, "{tenantid}", tid)
	}
	return discoveryIssuer == actual
}

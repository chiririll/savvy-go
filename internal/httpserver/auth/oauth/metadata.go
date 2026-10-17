package oauth

import (
	"net/http"

	"savvy-go/internal/httpserver/httpx"
)

// oauthPreflight answers CORS preflights from browser-based MCP clients. The
// endpoints take no cookies, so any origin may call them.
func oauthPreflight(w http.ResponseWriter, r *http.Request) {
	httpx.OpenCORS(w)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().
		Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Session-Id, Mcp-Protocol-Version, Last-Event-ID")
	w.WriteHeader(http.StatusNoContent)
}

// mcpResourceURL is the identifier of the MCP endpoint, what clients bind tokens to.
func (h Handler) mcpResourceURL(r *http.Request) string { return h.Cfg.AppURL() + "/mcp" }

// resourceMetadata is the protected resource metadata of RFC 9728.
type resourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
	ResourceName           string   `json:"resource_name"`
}

// serverMetadata is the authorization server metadata of RFC 8414.
type serverMetadata struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	RegistrationEndpoint          string   `json:"registration_endpoint,omitempty"`
	ResponseTypesSupported        []string `json:"response_types_supported"`
	GrantTypesSupported           []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethods      []string `json:"token_endpoint_auth_methods_supported"`
	ScopesSupported               []string `json:"scopes_supported"`
}

func (h Handler) oauthResourceMetadata(w http.ResponseWriter, r *http.Request) {
	httpx.OpenCORS(w)
	base := h.Cfg.AppURL()
	httpx.WriteJSON(w, http.StatusOK, resourceMetadata{
		Resource:               h.mcpResourceURL(r),
		AuthorizationServers:   []string{base},
		ScopesSupported:        oauthScopes,
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Go Savvy",
	})
}

func (h Handler) oauthServerMetadata(w http.ResponseWriter, r *http.Request) {
	httpx.OpenCORS(w)
	base := h.Cfg.AppURL()
	meta := serverMetadata{
		Issuer:                        base,
		AuthorizationEndpoint:         base + "/oauth/authorize",
		TokenEndpoint:                 base + "/oauth/token",
		ResponseTypesSupported:        []string{"code"},
		GrantTypesSupported:           []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported: []string{"S256"},
		TokenEndpointAuthMethods:      []string{"none"},
		ScopesSupported:               oauthScopes,
	}
	// Registration exists only while an admin has it open; closed, nothing
	// hints that it could be, and clients must not cache the answer.
	if _, open := h.OAuth.RegistrationOpenUntil(); open {
		meta.RegistrationEndpoint = base + "/oauth/register"
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, meta)
}

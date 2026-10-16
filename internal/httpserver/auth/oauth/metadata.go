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
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Session-Id, Mcp-Protocol-Version, Last-Event-ID")
	w.WriteHeader(http.StatusNoContent)
}

// mcpResourceURL is the identifier of the MCP endpoint, what clients bind tokens to.
func (h Handler) mcpResourceURL(r *http.Request) string { return h.Cfg.URL() + "/mcp" }

func (h Handler) oauthResourceMetadata(w http.ResponseWriter, r *http.Request) {
	httpx.OpenCORS(w)
	base := h.Cfg.URL()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"resource":                 h.mcpResourceURL(r),
		"authorization_servers":    []string{base},
		"scopes_supported":         oauthScopes,
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Go Savvy",
	})
}

func (h Handler) oauthServerMetadata(w http.ResponseWriter, r *http.Request) {
	httpx.OpenCORS(w)
	base := h.Cfg.URL()
	meta := map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      oauthScopes,
	}
	// Registration exists only while an admin has it open; closed, nothing
	// hints that it could be, and clients must not cache the answer.
	if _, open := h.OAuth.RegistrationOpenUntil(); open {
		meta["registration_endpoint"] = base + "/oauth/register"
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, meta)
}

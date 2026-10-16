package oauth

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/mcpserver"
	"savvy-go/internal/oauth"
	"savvy-go/internal/spaces"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{
			Cfg: routing.Get[appconfig.Config](d), OAuth: routing.Get[*oauth.Service](d), Spaces: routing.Get[spaces.Service](d),
			MCP: mcpserver.Handler{Cfg: routing.Get[appconfig.Config](d), APITokens: routing.Get[auth.APITokens](d), Inner: d.Inner, Version: routing.Get[appconfig.Build](d).Version}.Endpoint(),
		}
		return []routing.Mount{
			routing.Credential(h.CredentialRoutes),
			routing.RootAppURL(h.Routes),
		}
	})
}

// Handler holds what the OAuth endpoints need. MCP is the endpoint they protect.
type Handler struct {
	Cfg    appconfig.Config
	OAuth  *oauth.Service
	Spaces spaces.Service
	MCP    http.Handler
}

// Routes mounts the OAuth endpoints and the MCP endpoint they protect.
func (h Handler) Routes(r chi.Router) {
	for _, p := range []string{
		"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/*",
		"/.well-known/oauth-authorization-server", "/oauth/register", "/oauth/token", "/mcp",
	} {
		r.Options(p, oauthPreflight)
	}
	r.Get("/.well-known/oauth-protected-resource", h.oauthResourceMetadata)
	r.Get("/.well-known/oauth-protected-resource/*", h.oauthResourceMetadata)
	r.Get("/.well-known/oauth-authorization-server", h.oauthServerMetadata)
	r.Post("/oauth/register", h.oauthRegister)
	r.Get("/oauth/authorize", h.oauthAuthorize)
	r.Post("/oauth/token", h.oauthToken)
	// Per method, not Handle: Handle matches OPTIONS too and would replace the preflight route.
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		r.Method(method, "/mcp", h.MCP)
	}
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Get("/oauth/authorize", h.oauthConsentInfo)
	r.Post("/oauth/authorize", h.oauthConsent)
}

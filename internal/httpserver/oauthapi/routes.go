package oauthapi

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/mcpserver"
	"savvy-go/internal/httpserver/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{
			Cfg: d.Cfg, OAuth: d.OAuth, Spaces: d.Spaces,
			MCP: mcpserver.Handler{Cfg: d.Cfg, APITokens: d.APITokens, Inner: d.Inner}.Endpoint(),
		}
		return []routing.Mount{
			routing.Credential(h.CredentialRoutes),
			routing.AdminSession(h.AdminRoutes),
			routing.RootAppURL(h.Routes),
		}
	})
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
	r.Handle("/mcp", h.MCP)
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Get("/oauth/authorize", h.oauthConsentInfo)
	r.Post("/oauth/authorize", h.oauthConsent)
}

func (h Handler) AdminRoutes(r chi.Router) {
	r.Get("/admin/oauth/registration", h.oauthRegistrationShow)
	r.Post("/admin/oauth/registration", h.oauthRegistrationOpen)
	r.Delete("/admin/oauth/registration", h.oauthRegistrationClose)
	r.Get("/admin/oauth/clients", h.oauthClientsIndex)
	r.Delete("/admin/oauth/clients/{id}", h.oauthClientDestroy)
}

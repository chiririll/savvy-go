// Package oauthapi is the OAuth 2.1 authorization server MCP clients sign in with.
package oauthapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"savvy-go/internal/config"
	"savvy-go/internal/oauth"
	"savvy-go/internal/spaces"
)

// The OAuth 2.1 authorization server MCP clients (Claude web) sign in with.
// Endpoints sit outside /api, at the paths the clients discover through
// /.well-known; the consent screen is an SPA page that talks to
// /api/oauth/authorize with the user's session. See internal/oauth.

const oauthBodyLimit = 64 << 10

// oauthScopes are advertised for clients that want to show something. They do
// not grant anything: what a token may do is the per-resource permissions the
// user ticked on the consent screen (see auth.MCPResources), read by default.
var oauthScopes = []string{"read", "write"}

// Handler holds what the OAuth endpoints need. MCP is the endpoint they protect.
type Handler struct {
	Cfg    config.Config
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

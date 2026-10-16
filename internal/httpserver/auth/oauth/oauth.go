// Package oauth is the OAuth 2.1 authorization server MCP clients sign in with.
package oauth

import (
	"net/http"

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
	AppURL func() string // the public URL of the instance, empty when unset
	OAuth  *oauth.Service
	Spaces spaces.Service
	MCP    http.Handler
}

// Package mcpserver is the MCP endpoint: tools that call the app's own REST routes as the caller.
package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	savvyauth "savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/platform/version"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The MCP server: tools for AI assistants, served over Streamable HTTP at
// /mcp. It has no data access of its own. Every tool calls one of the app's own
// routes in-process with the caller's bearer token, so roles, token
// permissions, token spaces, membership and the JSON shape (money, currencies) are exactly
// those of the REST API and cannot drift from it. The read tools are here, the
// write tools in write_tools.go; a token that was only given read access can
// call a write tool and gets the REST 403 naming the missing permission.

const (
	mcpMaxResult    = 256 << 10
	mcpMaxPerPage   = 100
	mcpInstructions = "Go Savvy is a multi-currency expense tracker. Data lives in spaces: call list_spaces first and pass a space's id as space_id. " +
		"Amounts are exact decimal numbers in the currency of the account or transaction they belong to; never add amounts of different currencies, " +
		"and always say which currency a figure is in. Dates are YYYY-MM-DD. " +
		"Tools that create, change or delete data work only when the user granted write access; they act as the user, so do not use them unless the user asked for the change, " +
		"and ask before deleting. Amounts are strings, for example \"12.50\"."
)

// Handler is the /mcp endpoint. Inner is the app's own router: tools call it
// in-process with the caller's bearer token, so the REST rules apply.
type Handler struct {
	Cfg       config.Config
	APITokens savvyauth.APITokens
	Inner     http.Handler
}

// Endpoint is the handler to mount at /mcp. The OAuth metadata URL is read per
// request because the public URL can change in the admin panel.
func (h Handler) Endpoint() http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		var perms []string
		if info := auth.TokenInfoFromContext(r.Context()); info != nil {
			perms = info.Scopes
		}
		return h.mcpServer(perms)
	}, &mcp.StreamableHTTPOptions{
		// Stateless: nothing to lose on restart, no session to hijack.
		Stateless:    true,
		JSONResponse: true,
		// The check guards unauthenticated localhost servers against DNS
		// rebinding. Here every request needs a bearer token a rebinding page
		// cannot send, and behind a reverse proxy it would reject the public Host.
		DisableLocalhostProtection: true,
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.OpenCORS(w)
		opts := &auth.RequireBearerTokenOptions{
			ResourceMetadataURL:    h.Cfg.URL() + "/.well-known/oauth-protected-resource",
			AllowMissingExpiration: true,
		}
		auth.RequireBearerToken(h.verifyMCPToken, opts)(streamable).ServeHTTP(w, r)
	})
}

// verifyMCPToken accepts any live API token: OAuth access tokens and tokens
// made by hand in Settings alike. A browser session is not enough. What the
// token may read is decided per call, by the REST routes the tools use.
func (h Handler) verifyMCPToken(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
	t, err := h.APITokens.Resolve(ctx, raw)
	if err != nil {
		return nil, err
	}
	if t == nil || t.User == nil {
		return nil, fmt.Errorf("%w: unknown or expired token", auth.ErrInvalidToken)
	}
	info := &auth.TokenInfo{Scopes: t.Permissions, UserID: strconv.FormatInt(t.UserID, 10)}
	if t.ExpiresAt != nil {
		info.Expiration = *t.ExpiresAt
	}
	return info, nil
}

// mcpNeed is the grant a tool's REST route asks of a token.
type mcpNeed struct{ resource, level string }

// mcpServer builds the tool set for a token holding perms. It is cheap and
// holds nothing per user: each call reads the caller's token from its own
// request. Tools the token has no grant for are left out of the list, so an
// assistant is not offered what would only fail. This is only a courtesy: a
// call to one anyway still reaches the REST route, which checks the grant and
// names the missing permission.
func (h Handler) mcpServer(perms []string) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: "go-savvy", Version: version.Value}, &mcp.ServerOptions{Instructions: mcpInstructions})
	h.addSimpleReadTools(srv)
	h.addQueryReadTools(srv)
	h.hideUngrantedTools(srv, h.addMCPWriteTools(srv), perms)
	return srv
}

// hideUngrantedTools leaves out of the tool list what the token has no grant for.
func (h Handler) hideUngrantedTools(srv *mcp.Server, needs map[string]mcpNeed, perms []string) {
	for name, resource := range mcpReadNeeds {
		needs[name] = mcpNeed{resource, savvyauth.PermRead}
	}
	token := &savvyauth.APIToken{Permissions: perms}
	srv.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			res, err := next(ctx, method, req)
			if list, ok := res.(*mcp.ListToolsResult); ok && err == nil {
				list.Tools = slices.DeleteFunc(list.Tools, func(t *mcp.Tool) bool {
					n, known := needs[t.Name]
					return known && !token.Allows(n.resource, n.level)
				})
			}
			return res, err
		}
	})
}

// Package routing lets each handler package register its own routes. A package calls
// Register from init in its routes.go and names the access tier of every route set with
// Public, Account, SpaceWrite and the like; Build mounts them all under the right
// middleware. The server only lists the packages to link in (modules.go).
package routing

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/apidocs"
	"savvy-go/internal/httpserver/guard"
	"savvy-go/internal/httpserver/mailing"
	"savvy-go/internal/httpserver/signin"
	"savvy-go/internal/jobs"
	"savvy-go/internal/ledger"
	"savvy-go/internal/oauth"
	"savvy-go/internal/refsync"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/sso"
	"savvy-go/internal/store"
	"savvy-go/internal/transfers"
	"savvy-go/internal/uploads"
)

// Deps are the services the server owns and its handlers draw on.
type Deps struct {
	Cfg       config.Config
	Store     store.Store
	Rates     ledger.RateSource
	Bootstrap *bootstrap.Guard
	Queue     *jobs.Queue
	Keys      *signing.Holder

	Users          auth.Users
	Sessions       auth.Sessions
	APITokens      auth.APITokens
	PasswordTokens auth.PasswordTokens
	Challenges     auth.Challenges
	Verifications  auth.EmailVerifications
	TwoFactor      auth.TwoFactor
	WebAuthn       auth.WebAuthn

	Mail      *mailing.Service
	Settings  settings.Store
	Spaces    spaces.Service
	Backups   backup.Service
	Transfers *transfers.Service
	Refsync   refsync.Service
	Uploads   uploads.Service
	SSO       *sso.Service
	OAuth     *oauth.Service
	Login     signin.Service
	Guard     *guard.Guard

	// Inner is the finished router, for handlers that call the API in-process (MCP).
	Inner http.Handler
}

// Tier is who may reach a set of routes, and so which middleware wraps it.
type Tier int

const (
	root         Tier = iota // outside /api, open
	rootAppURL               // outside /api, needs the public URL (OAuth discovery)
	public                   // /api, open
	credential               // /api, real sessions only: passwords, 2FA, passkeys, tokens
	account                  // /api, session or API token
	spaceRead                // /api/spaces/{space}, any member
	spaceAdmin               // /api/spaces/{space}, the space's admins in a real session
	spaceWrite               // /api/spaces/{space}, members who may write
	admin                    // /api, server admins
	adminSession             // /api, server admins in a real session
)

// Mount is a set of routes in a tier.
type Mount struct {
	tier   Tier
	routes func(chi.Router)
}

func Root(fn func(chi.Router)) Mount         { return Mount{root, fn} }
func RootAppURL(fn func(chi.Router)) Mount   { return Mount{rootAppURL, fn} }
func Public(fn func(chi.Router)) Mount       { return Mount{public, fn} }
func Credential(fn func(chi.Router)) Mount   { return Mount{credential, fn} }
func Account(fn func(chi.Router)) Mount      { return Mount{account, fn} }
func SpaceRead(fn func(chi.Router)) Mount    { return Mount{spaceRead, fn} }
func SpaceAdmin(fn func(chi.Router)) Mount   { return Mount{spaceAdmin, fn} }
func SpaceWrite(fn func(chi.Router)) Mount   { return Mount{spaceWrite, fn} }
func Admin(fn func(chi.Router)) Mount        { return Mount{admin, fn} }
func AdminSession(fn func(chi.Router)) Mount { return Mount{adminSession, fn} }

var (
	mu      sync.Mutex
	modules []func(*Deps) []Mount
)

// Register adds a package's routes. spec is its embedded openapi.yaml, nil when it
// documents nothing. build runs once per server, so a handler gets that server's Deps.
func Register(spec []byte, build func(*Deps) []Mount) {
	if spec != nil {
		apidocs.Add(spec)
	}
	mu.Lock()
	defer mu.Unlock()
	modules = append(modules, build)
}

// Build mounts every registered package on r.
func Build(r chi.Router, d *Deps) {
	mu.Lock()
	mods := append([]func(*Deps) []Mount(nil), modules...)
	mu.Unlock()

	byTier := map[Tier][]func(chi.Router){}
	for _, build := range mods {
		for _, m := range build(d) {
			byTier[m.tier] = append(byTier[m.tier], m.routes)
		}
	}
	mount := func(t Tier, r chi.Router) {
		for _, fn := range byTier[t] {
			fn(r)
		}
	}
	g := d.Guard

	mount(root, r)
	r.Route("/api", func(r chi.Router) {
		apidocs.Routes(r)
		mount(public, r)

		r.Group(func(r chi.Router) {
			r.Use(g.RequireSession)
			r.Use(g.RequireCSRF)
			r.Use(g.RequireVerifiedEmail)

			// Credential management is limited to real sessions, never API tokens.
			r.Group(func(r chi.Router) {
				r.Use(g.SessionOnly)
				mount(credential, r)
			})

			mount(account, r)

			r.Route("/spaces/{space}", func(r chi.Router) {
				r.Use(g.WithSpace)
				r.Use(g.RequireTokenGrant)
				mount(spaceRead, r)

				// Administering the space: its admins, in a real session.
				r.Group(func(r chi.Router) {
					r.Use(g.RequireSpaceAdmin)
					r.Use(g.SessionOnly)
					mount(spaceAdmin, r)
				})

				r.Group(func(r chi.Router) {
					r.Use(g.RequireWrite)
					mount(spaceWrite, r)
				})
			})

			// Users, instance settings, monitoring and the space overview are the
			// server admins'.
			r.Group(func(r chi.Router) {
				r.Use(g.RequireAdmin)
				mount(admin, r)
			})
			r.Group(func(r chi.Router) {
				r.Use(g.RequireAdmin)
				r.Use(g.SessionOnly)
				mount(adminSession, r)
			})
		})
	})

	// MCP clients (Claude web) discover and sign in through these; they need the public
	// URL because it is their issuer.
	r.Group(func(r chi.Router) {
		r.Use(g.NeedAppURL)
		mount(rootAppURL, r)
	})
}

// SpaceWriteRoutes mounts only the finance routes of a space on r, with no middleware. The
// tests use it to learn which path roots belong to a space.
func SpaceWriteRoutes(r chi.Router, d *Deps) {
	mu.Lock()
	mods := append([]func(*Deps) []Mount(nil), modules...)
	mu.Unlock()
	for _, build := range mods {
		for _, m := range build(d) {
			if m.tier == spaceWrite {
				m.routes(r)
			}
		}
	}
}

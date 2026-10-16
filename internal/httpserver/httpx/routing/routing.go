// Package routing lets each handler package register its own routes. A package calls
// Register from init in its routes.go and names the access tier of every route set with
// Public, Account, SpaceWrite and the like; Build mounts them all under the right
// middleware. The server only lists the packages to link in (modules.go).
package routing

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/login"
	"savvy-go/internal/locator"
)

// Deps are what the registered packages draw on: the locator that holds the core services,
// and the HTTP-level pieces built on them.
type Deps struct {
	Loc *locator.Locator

	Login login.Service
	Guard *guard.Guard

	// Inner is the finished router, for handlers that call the API in-process (MCP).
	Inner http.Handler
}

// Get is the T a package asks the locator for when its routes are built. A missing service
// is a wiring bug and panics at start.
func Get[T any](d *Deps) T { return locator.Get[T](d.Loc) }

// ConfigWritable is whether the application changes the config file. It does not for now:
// the file is edited by hand. Section.Update and config.Writable stay for when that is
// switched back on.
func (d *Deps) ConfigWritable() bool { return false }

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
	fallback                 // outside /api, answers what no other route does (the SPA)
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
func Fallback(fn func(chi.Router)) Mount     { return Mount{fallback, fn} }

var (
	mu      sync.Mutex
	modules []func(*Deps) []Mount
)

// Register adds a package's routes. build runs once per server, so a handler gets that
// server's Deps. A package that documents its routes hands its openapi.yaml to ui.AddSpec.
func Register(build func(*Deps) []Mount) {
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

	mount(fallback, r)
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

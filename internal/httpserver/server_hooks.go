package httpserver

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/locator"
	"savvy-go/internal/store"
)

// These reach into a built Server for the tests in httpserver/integration, which live in their
// own package and so cannot see its fields.

// Locator is the service locator the server was built from.
func (s *Server) Locator() *locator.Locator { return s.loc }

// Routes is the finished router, for walking the registered routes.
func (s *Server) Routes() chi.Routes { return s.mux }

// Guard is the request guard the routes share.
func (s *Server) Guard() *guard.Guard { return s.guard }

// RebuildGuard refreshes the guard from the locator, for a test that swapped a service in it.
func (s *Server) RebuildGuard() { *s.guard = *s.newGuard() }

// NewScope is the scope of a space for the given role, as the guard builds it for a request.
func (s *Server) NewScope(id int64, role string, d store.DB) *httpx.SpaceScope {
	return s.newScope(id, role, d)
}

// Deps are what the packages that register routes draw on.
func (s *Server) Deps() *routing.Deps { return s.deps() }

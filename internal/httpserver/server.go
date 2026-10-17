package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"savvy-go/internal/app"
	"savvy-go/internal/app/signin"
	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/login"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/locator"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
)

// Server is the HTTP front door: health probes, /api, and the Vite SPA.
type Server struct {
	loc *locator.Locator

	mux   *chi.Mux
	login login.Service
	guard *guard.Guard
}

// New builds the services and serves them. Tests use it; the process builds its own locator
// (internal/boot) and calls FromLocator.
func New(cfg appconfig.Config, st store.Store, keys *signing.Holder, opts ...Option) *Server {
	l, err := app.New(cfg, st, keys, opts...)
	if err != nil {
		panic(err) // New is for tests: a wiring bug should stop them
	}
	return FromLocator(l)
}

// FromLocator serves the services the caller built, so it can share them with other parts of
// the process (the scheduled jobs).
func FromLocator(l *locator.Locator) *Server {
	s := &Server{
		loc: l,
		login: login.Service{
			Policy:   locator.Get[signin.Policy](l),
			Security: locator.Get[auth.Security](l),
			Sessions: locator.Get[auth.Sessions](l),
		},
	}
	s.guard = s.newGuard()
	s.login.Guard = s.guard
	s.mux = s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(locale)
	r.Use(noStoreAPI)
	r.Use(noFraming)

	routing.Build(r, s.deps())

	return r
}

// deps are what the packages that register routes draw on.
func (s *Server) deps() *routing.Deps {
	return &routing.Deps{
		Loc: s.loc, Login: s.login, Guard: s.guard,
		// The MCP tools call the finished router, which exists after New.
		Inner: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }),
	}
}

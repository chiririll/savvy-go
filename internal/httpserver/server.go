package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"savvy-go/internal/app"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/auth/emailverify"
	"savvy-go/internal/httpserver/auth/signin"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/spa"
	"savvy-go/internal/signing"
	"savvy-go/internal/spacefill"
	"savvy-go/internal/store"
	"savvy-go/internal/webui"
)

// Server is the HTTP front door: health probes, /api, and the Vite SPA.
type Server struct {
	*app.Services

	mux    *chi.Mux
	spa    *spa.Handler // the frontend: index.html, icons and build/
	login  signin.Service
	guard  *guard.Guard
	verify emailverify.Handler
}

// New builds the services and serves them.
func New(cfg config.Config, st store.Store, keys *signing.Holder, opts ...Option) *Server {
	svc := app.New(cfg, st, keys, opts...)
	s := &Server{
		Services: svc,
		spa:      &spa.Handler{Assets: spa.Resolve(webui.FS())},
		login:    signin.Service{Cfg: cfg, Sessions: svc.Sessions, Settings: svc.Settings, SSO: svc.SSO},
	}
	s.guard = s.newGuard()
	s.verify = emailverify.Handler{Cfg: cfg, Mail: svc.Mail, Verifications: svc.Verifications, Users: svc.Users}
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

	r.Get("/*", s.spa.ServeHTTP)
	return r
}

func (s *Server) spaceCreator() spacefill.Creator {
	return spacefill.Creator{Spaces: s.Spaces, Store: s.Store, Rates: s.Rates}
}

// deps are what the packages that register routes draw on.
func (s *Server) deps() *routing.Deps {
	return &routing.Deps{
		Services: s.Services, Login: s.login, Guard: s.guard,
		// The MCP tools call the finished router, which exists after New.
		Inner: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }),
	}
}

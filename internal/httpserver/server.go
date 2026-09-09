package httpserver

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/chiririll/savvy-plus/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the HTTP front door: health probes, /api, and the Vite SPA.
type Server struct {
	cfg config.Config
	db  *sql.DB
	mux *chi.Mux
}

func New(cfg config.Config, sqlDB *sql.DB) *Server {
	s := &Server{cfg: cfg, db: sqlDB}
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

	r.Get("/livez", s.livez)
	r.Get("/readyz", s.readyz)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.livez)
		// Domain routes are registered as slices land (auth, CRUD, …).
	})

	r.Get("/*", s.spa)
	return r
}

func locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if loc := strings.TrimSpace(r.Header.Get("X-Locale")); loc != "" {
			w.Header().Set("Content-Language", loc)
		}
		next.ServeHTTP(w, r)
	})
}

func noStoreAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

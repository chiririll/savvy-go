package httpserver

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"savvy-go/internal/backup"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/httpserver/emailverify"
	"savvy-go/internal/httpserver/guard"
	"savvy-go/internal/httpserver/mailing"
	"savvy-go/internal/httpserver/routing"
	"savvy-go/internal/httpserver/signin"
	"savvy-go/internal/httpserver/spa"
	"savvy-go/internal/httpserver/spacefill"
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
	"savvy-go/internal/webui"
)

// Server is the HTTP front door: health probes, /api, and the Vite SPA.
// Server-level services use the server database; the services of a space are
// built per request by guard.WithSpace (see spaceScope).
type Server struct {
	cfg       config.Config
	store     store.Store
	mux       *chi.Mux
	keys      *signing.Holder
	rates     ledger.RateSource // currency catalog and exchange rates, shared by every space
	bootstrap *bootstrap.Guard  // first-run password; nil keeps registration closed
	spa       *spa.Handler      // the frontend: index.html, icons and build/
	queue     *jobs.Queue

	accountServices
	mail *mailing.Service

	settings  settings.Store
	spaces    spaces.Service
	backups   backup.Service
	transfers *transfers.Service
	refsync   refsync.Service
	uploads   uploads.Service
	sso       sso.Service
	guard     *guard.Guard
	login     signin.Service
	verify    emailverify.Handler
	oauth     *oauth.Service
}

func New(cfg config.Config, st store.Store, keys *signing.Holder, opts ...Option) *Server {
	ledger.SetLocation(cfg.Location)
	srv := st.Server()
	s := &Server{
		cfg:             cfg,
		spa:             &spa.Handler{Assets: spa.Resolve(webui.FS())},
		store:           st,
		keys:            keys,
		accountServices: newAccountServices(cfg, srv),
		settings:        settings.Store{DB: srv},
		spaces:          spaces.Service{Store: st},
		uploads:         uploads.Service{DB: srv, Root: cfg.UploadsDir, SignSecret: randomSecret()},
	}
	keyRing := signing.KeyRing{Holder: keys, Trusted: transfers.TrustedKeys(srv)}
	s.transfers = &transfers.Service{Spaces: s.spaces, Keys: keyRing}
	s.refsync = refsync.Service{Spaces: s.spaces}
	s.backups = backup.Service{Store: st, Keys: keyRing, Dir: cfg.BackupsDir, DataDir: cfg.DataDir, Transfers: s.transfers}
	for _, opt := range opts {
		opt(s)
	}
	if s.rates == nil {
		s.rates = currencydata.New()
	}
	if s.mail == nil {
		s.mail = mailing.New(cfg, nil)
	}
	s.sso = sso.Service{DB: srv, Users: s.users, Settings: s.settings, AppURL: cfg.URL}
	s.oauth = oauth.New(srv)
	s.login = signin.Service{Cfg: cfg, Sessions: s.sessions, Settings: s.settings, SSO: &s.sso}
	s.guard = s.newGuard()
	s.verify = emailverify.Handler{Cfg: cfg, Mail: s.mail, Verifications: s.verifications, Users: s.users}
	_ = os.MkdirAll(cfg.UploadsDir, 0o775)
	_ = os.MkdirAll(cfg.BackupsDir, 0o775)
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
	return spacefill.Creator{Spaces: s.spaces, Store: s.store, Rates: s.rates}
}

// deps are the services handed to the packages that register routes.
func (s *Server) deps() *routing.Deps {
	return &routing.Deps{
		Cfg: s.cfg, Store: s.store, Rates: s.rates, Bootstrap: s.bootstrap, Queue: s.queue, Keys: s.keys,
		Users: s.users, Sessions: s.sessions, APITokens: s.apiTokens, PasswordTokens: s.tokens,
		Challenges: s.challenges, Verifications: s.verifications, TwoFactor: s.twoFactor, WebAuthn: s.webauthn,
		Mail: s.mail, Settings: s.settings, Spaces: s.spaces, Backups: s.backups, Transfers: s.transfers,
		Refsync: s.refsync, Uploads: s.uploads, SSO: &s.sso, OAuth: s.oauth, Login: s.login, Guard: s.guard,
		// The MCP tools call the finished router, which exists after New.
		Inner: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }),
	}
}

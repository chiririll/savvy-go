package httpserver

import (
	"io/fs"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/backup"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
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
// built per request by withSpace (see spaceScope).
type Server struct {
	cfg       config.Config
	store     store.Store
	mux       *chi.Mux
	keys      *signing.Holder
	rates     ledger.RateSource // currency catalog and exchange rates, shared by every space
	bootstrap *bootstrap.Guard  // first-run password; nil keeps registration closed
	assets    fs.FS             // the frontend: index.html, icons and build/
	queue     *jobs.Queue

	accountServices
	mailState

	settings  settings.Store
	spaces    spaces.Service
	backups   backup.Service
	transfers *transfers.Service
	refsync   refsync.Service
	uploads   uploads.Service
	sso       sso.Service
	oauth     *oauth.Service
}

func New(cfg config.Config, st store.Store, keys *signing.Holder, opts ...Option) *Server {
	ledger.SetLocation(cfg.Location)
	srv := st.Server()
	s := &Server{
		cfg:             cfg,
		assets:          resolveAssets(webui.FS()),
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
	s.initMail(cfg)
	s.sso = sso.Service{DB: srv, Users: s.users, Settings: s.settings, AppURL: cfg.URL}
	s.oauth = oauth.New(srv)
	_ = os.MkdirAll(cfg.UploadsDir, 0o775)
	_ = os.MkdirAll(cfg.BackupsDir, 0o775)
	s.mux = s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

// Package app is the composition root of the core: it builds every service from the
// config, the store and the signing keys, so the HTTP layer only picks what it needs.
package app

import (
	"os"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/jobs"
	"savvy-go/internal/ledger"
	"savvy-go/internal/mailer"
	"savvy-go/internal/mailing"
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

// Services are the server-level services. Server-level ones use the server database;
// the services of a space are built per request from the space's own database.
type Services struct {
	Cfg   config.Config
	Store store.Store
	Keys  *signing.Holder
	Rates ledger.RateSource // currency catalog and exchange rates, shared by every space

	// Bootstrap is the first-run password; nil keeps registration closed.
	Bootstrap *bootstrap.Guard
	Queue     *jobs.Queue

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
}

// Option configures New.
type Option func(*Services)

// WithRates sets where currencies and exchange rates come from. The default is the public
// data files; share one source with the scheduled rate update so spaces do not each
// fetch their own.
func WithRates(src ledger.RateSource) Option { return func(s *Services) { s.Rates = src } }

// WithBootstrap sets the first-run password that registration needs while the server has
// no users. Without it, registration is always closed.
func WithBootstrap(g *bootstrap.Guard) Option { return func(s *Services) { s.Bootstrap = g } }

// WithQueue sets the worker pool that background jobs (imports) run on. Without it they
// run at once, inside the request.
func WithQueue(q *jobs.Queue) Option { return func(s *Services) { s.Queue = q } }

// WithMailer sets how email is sent. The default is the SMTP server of the [mailer]
// section of the config file; tests pass a recorder.
func WithMailer(m mailer.Mailer) Option {
	return func(s *Services) { s.Mail = mailing.New(s.Cfg, m) }
}

// New builds the services.
func New(cfg config.Config, st store.Store, keys *signing.Holder, opts ...Option) *Services {
	ledger.SetLocation(cfg.Location)
	db := st.Server()
	s := &Services{
		Cfg:            cfg,
		Store:          st,
		Keys:           keys,
		Users:          auth.Users{DB: db},
		Sessions:       auth.Sessions{DB: db, SessionTTL: cfg.SessionTTL, RememberTTL: cfg.RememberTTL},
		APITokens:      auth.APITokens{DB: db},
		PasswordTokens: auth.PasswordTokens{DB: db},
		Challenges:     auth.Challenges{DB: db, Cfg: cfg},
		Verifications:  auth.EmailVerifications{DB: db},
		TwoFactor:      auth.TwoFactor{DB: db, Users: auth.Users{DB: db}},
		WebAuthn:       auth.WebAuthn{DB: db, Cfg: cfg},
		Settings:       settings.Store{DB: db},
		Spaces:         spaces.Service{Store: st},
		Uploads:        uploads.Service{DB: db, Root: cfg.UploadsDir, SignSecret: randomSecret()},
	}
	keyRing := signing.KeyRing{Holder: keys, Trusted: transfers.TrustedKeys(db)}
	s.Transfers = &transfers.Service{Spaces: s.Spaces, Keys: keyRing}
	s.Refsync = refsync.Service{Spaces: s.Spaces}
	s.Backups = backup.Service{Store: st, Keys: keyRing, Dir: cfg.BackupsDir, DataDir: cfg.DataDir, Transfers: s.Transfers}
	for _, opt := range opts {
		opt(s)
	}
	if s.Rates == nil {
		s.Rates = currencydata.New()
	}
	if s.Mail == nil {
		s.Mail = mailing.New(cfg, nil)
	}
	s.SSO = &sso.Service{DB: db, Users: s.Users, Settings: s.Settings, AppURL: cfg.URL}
	s.OAuth = oauth.New(db)
	_ = os.MkdirAll(cfg.UploadsDir, 0o775)
	_ = os.MkdirAll(cfg.BackupsDir, 0o775)
	return s
}

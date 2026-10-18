// Package app registers the core services on a locator, from the config, the store and the
// signing keys. The HTTP layer asks the locator for what it needs.
package app

import (
	"os"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/firstrun"
	"savvy-go/internal/jobs"
	"savvy-go/internal/ledger"
	"savvy-go/internal/locator"
	"savvy-go/internal/mail/smtp"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
)

// inputs are what the process makes before the services and hands to New.
type inputs struct {
	conf   *config.Config // each package pulls its own sections from it; not shared
	sender smtp.Mailer
	rates  ledger.RateSource
	build  appconfig.Build
	guard  *firstrun.Guard
	queue  *jobs.Queue
}

// Option configures New.
type Option func(*inputs)

// WithRates sets where currencies and exchange rates come from. The default is the public
// data files; share one source with the scheduled rate update so spaces do not each
// fetch their own.
func WithRates(src ledger.RateSource) Option { return func(s *inputs) { s.rates = src } }

// WithBuild sets the release the services report. The default is "dev" in "development".
func WithBuild(b appconfig.Build) Option { return func(s *inputs) { s.build = b } }

// WithBootstrap sets the first-run password that registration needs while the server has
// no users. Without it, registration is always closed.
func WithBootstrap(g *firstrun.Guard) Option { return func(s *inputs) { s.guard = g } }

// WithQueue sets the worker pool that background jobs (imports) run on. Without it they
// run at once, inside the request.
func WithQueue(q *jobs.Queue) Option { return func(s *inputs) { s.queue = q } }

// WithConfig sets the config file the services pull their sections from. The default is an
// empty one, so every section has its defaults.
func WithConfig(f *config.Config) Option { return func(s *inputs) { s.conf = f } }

// WithMailer sets how email is sent, whatever the [mailer] section says; tests pass a
// recorder. The default is the SMTP server of the section.
func WithMailer(m smtp.Mailer) Option { return func(s *inputs) { s.sender = m } }

// New registers every service on a locator and builds them all, so a wiring bug fails the
// start. The caller owns the locator and closes it.
func New(cfg appconfig.Config, st store.Store, keys *signing.Holder, opts ...Option) (*locator.Locator, error) {
	in := &inputs{}
	for _, opt := range opts {
		opt(in)
	}
	if in.conf == nil {
		in.conf = config.Empty()
	}
	if in.build.Version == "" {
		in.build.Version = "dev"
	}
	if in.build.Env == "" {
		in.build.Env = "development"
	}
	if in.rates == nil {
		in.rates = currencydata.New()
	}
	ledger.SetLocation(cfg.Location)
	l := locator.New()
	// What the process makes before the services: they only read it.
	locator.Set[store.DB](l, st.Server())
	locator.Set(l, st)
	locator.Set(l, keys)
	locator.Set(l, in.conf)
	locator.Set(l, cfg)
	locator.Set(l, in.build)
	locator.Set(l, in.rates)
	locator.Set(l, in.guard)
	locator.Set(l, in.queue)
	if in.sender != nil {
		locator.Set(l, in.sender)
	}
	registerServices(l)
	if err := l.Build(); err != nil {
		return nil, err
	}
	_ = os.MkdirAll(cfg.UploadsDir, 0o775)
	_ = os.MkdirAll(cfg.BackupsDir, 0o775)
	return l, nil
}

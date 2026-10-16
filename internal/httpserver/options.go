package httpserver

import (
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/httpserver/mailing"
	"savvy-go/internal/ledger"
	"savvy-go/internal/mailer"
)

// Option configures New.
type Option func(*Server)

// WithRates sets where currencies and exchange rates come from. The default
// is the public data files; share one source with the scheduled rate update
// so spaces do not each fetch their own.
func WithRates(src ledger.RateSource) Option { return func(s *Server) { s.rates = src } }

// WithBootstrap sets the first-run password that registration needs while the
// server has no users. Without it, registration is always closed.
func WithBootstrap(g *bootstrap.Guard) Option { return func(s *Server) { s.bootstrap = g } }

// WithMailer sets how email is sent. The default is the SMTP server of the
// [mailer] section of the config file; tests pass a recorder.
func WithMailer(m mailer.Mailer) Option { return func(s *Server) { s.mail = mailing.New(s.cfg, m) } }

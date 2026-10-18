// Package mail is the mail side of the application: the mail server of the instance
// (the sender in use and the configuration it was built from) and the confirmation of
// email addresses. SMTP itself lives in mail/smtp.
package mail

import (
	"context"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/config"
	"savvy-go/internal/mail/smtp"
)

// NoAppURLMessage is why a link cannot be sent while app_url is unset.
const NoAppURLMessage = "The public URL is not set, so the link would point nowhere. Set app_url first."

// DefaultTimeout is the connection timeout when none is set.
const DefaultTimeout = smtp.DefaultTimeout

// Service is the mail server in use: the [mailer] section of the config file, read each
// time it is needed.
type Service struct {
	cfg     appconfig.Config
	file    *config.Config
	section *config.Ref[smtp.Setup]
	sender  smtp.Mailer // given to New: used whatever the section says
}

// New sends mail as the [mailer] section of f says. A non-nil sender replaces the SMTP one,
// which is how tests record the mail.
func New(f *config.Config, cfg appconfig.Config, sender smtp.Mailer) *Service {
	return &Service{cfg: cfg, file: f, section: config.Of(f, smtp.Section), sender: sender}
}

// Settings is a full replacement of the [mailer] section, as an administrator sends it.
type Settings struct {
	Enabled    bool
	Host       string
	Port       int
	Protocol   string
	User       string
	Password   *string // nil keeps the stored password; "" clears it
	From       string
	SkipVerify bool
	Timeout    int // seconds
}

// Save writes the [mailer] section of the config file and applies it at once. An enabled
// configuration that is not valid is refused with an smtp.InvalidError and nothing changes; a
// disabled one may be half filled in. It fails with config.ErrReadOnly when the file cannot
// be written.
func (s *Service) Save(in Settings) error {
	change := func(m *smtp.MailerConfig) { *m = in.mailerConfig(m.Password) }
	// The section turns a mistake into "mail is off", so it is refused by the check instead.
	return smtp.Section.Edit(s.file, change, func(m smtp.MailerConfig) error { return m.Config().Validate() })
}

// mailerConfig is the [mailer] table these settings make; stored is the password kept when none is given.
func (in Settings) mailerConfig(stored string) smtp.MailerConfig {
	password := stored
	if in.Password != nil {
		password = *in.Password
	}
	return smtp.MailerConfig{
		Enabled: in.Enabled, Host: in.Host, Port: in.Port, Protocol: in.Protocol, User: in.User,
		Password: password, From: in.From, SkipVerify: in.SkipVerify, Timeout: in.Timeout,
	}
}

// Check connects and signs in to the server the settings describe, as if they were enabled, without
// saving them or sending anything. A configuration that cannot work is refused with an smtp.InvalidError.
func (s *Service) Check(ctx context.Context, in Settings) error {
	stored, _ := s.Config()
	in.Enabled = true
	cfg := in.mailerConfig(stored.Password).Config()
	if err := cfg.Validate(); err != nil {
		return err
	}
	mailer := s.sender
	if mailer == nil {
		mailer = smtp.New(cfg)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*cfg.Timeout)
	defer cancel()
	return mailer.Check(ctx)
}

// Mailer is the current sender.
func (s *Service) Mailer() smtp.Mailer {
	if s.sender != nil {
		return s.sender
	}
	return smtp.New(s.section.Get().Config)
}

// Config is the current SMTP configuration and why it is switched off, if
// the section was invalid.
func (s *Service) Config() (smtp.Config, string) {
	setup := s.section.Get()
	return setup.Config, setup.Problem
}

// AppURL is the public URL of the instance, empty when unset.
func (s *Service) AppURL() string { return s.cfg.AppURL() }

// Send sends one email, giving up after a few connection timeouts.
func (s *Service) Send(ctx context.Context, m smtp.Message) error {
	cfg, _ := s.Config()
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, 3*timeout)
	defer cancel()
	return s.Mailer().Send(ctx, m)
}

// Status is whether mail works, and what is wrong with the mail section.
func (s *Service) Status() (enabled bool, problem string) {
	conf, problem := s.Config()
	return s.Mailer().Enabled() && conf.Enabled, problem
}

// OffReason says why mail cannot be used.
func (s *Service) OffReason() string {
	reason, hint := s.OffWhy()
	if hint == "" {
		return reason
	}
	return reason + " " + hint
}

// OffWhy splits OffReason into the fact and, when there is one, where
// an administrator fixes it. Pages that are not for administrators show the fact only.
func (s *Service) OffWhy() (reason, hint string) {
	if _, problem := s.Config(); problem != "" {
		return "Mail is switched off: " + problem + ".", ""
	}
	return "Mail is not configured.", "Set it up in the [mailer] section of the config file."
}

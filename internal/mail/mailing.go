// Package mail is the mail side of the application: the mail server of the instance
// (the sender in use and the settings it was built from) and the confirmation of
// email addresses. SMTP itself lives in mail/smtp.
package mail

import (
	"context"

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
	appURL  func() string
	section *config.Ref[smtp.Setup]
	sender  smtp.Mailer // given to New: used whatever the section says
}

// New sends mail as the [mailer] section of f says. A non-nil sender replaces the SMTP one,
// which is how tests record the mail.
func New(f *config.Config, appURL func() string, sender smtp.Mailer) *Service {
	return &Service{appURL: appURL, section: config.Of(f, smtp.Section), sender: sender}
}

// Mailer is the current sender.
func (s *Service) Mailer() smtp.Mailer {
	if s.sender != nil {
		return s.sender
	}
	return smtp.New(s.section.Get().Config)
}

// Settings is the current SMTP configuration and why it is switched off, if
// the section was invalid.
func (s *Service) Settings() (smtp.Config, string) {
	setup := s.section.Get()
	return setup.Config, setup.Problem
}

// AppURL is the public URL of the instance, empty when unset.
func (s *Service) AppURL() string { return s.appURL() }

// Send sends one email, giving up after a few connection timeouts.
func (s *Service) Send(ctx context.Context, m smtp.Message) error {
	cfg, _ := s.Settings()
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
	conf, problem := s.Settings()
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
	if _, problem := s.Settings(); problem != "" {
		return "Mail is switched off: " + problem + ".", ""
	}
	return "Mail is not configured.", "Set it up in the [mailer] section of the config file."
}

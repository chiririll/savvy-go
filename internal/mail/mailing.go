// Package mail is the mail side of the application: the mail server of the instance
// (the sender in use and the settings it was built from) and the confirmation of
// email addresses. SMTP itself lives in mail/smtp.
package mail

import (
	"context"
	"sync"
	"time"

	"savvy-go/internal/config"
	"savvy-go/internal/mail/smtp"
)

// NoAppURLMessage is why a link cannot be sent while app_url is unset.
const NoAppURLMessage = "The public URL is not set, so the link would point nowhere. Set app_url first."

const DefaultTimeout = 10 * time.Second

// Service is the mail server in use. An admin can change it, so it is guarded
// by a lock: read the sender through Mailer and the settings through Settings.
type Service struct {
	cfg config.Config

	// NewMailer builds a sender from settings; tests swap it to record
	// instead of dialling.
	NewMailer func(smtp.Config) smtp.Mailer

	mu      sync.RWMutex
	sender  smtp.Mailer
	conf    smtp.Config
	problem string
}

// New starts from the [mailer] section of cfg. A non-nil sender replaces the
// SMTP one, which is how tests record the mail.
func New(cfg config.Config, sender smtp.Mailer) *Service {
	s := &Service{cfg: cfg, NewMailer: smtp.New, sender: sender, conf: cfg.Mail, problem: cfg.MailProblem}
	if s.sender == nil {
		s.sender = s.NewMailer(cfg.Mail)
	}
	return s
}

// Mailer is the current sender. It changes when an admin saves new settings.
func (s *Service) Mailer() smtp.Mailer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sender
}

// Settings is the current SMTP configuration and why it is switched off, if
// a saved section was invalid.
func (s *Service) Settings() (smtp.Config, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conf, s.problem
}

// UseConfig replaces the config the service reads the URL and the config file from.
func (s *Service) UseConfig(cfg config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

func (s *Service) Config() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// SetProblem records why the saved mail section cannot be used.
func (s *Service) SetProblem(problem string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.problem = problem
}

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
	return "Mail is not configured.", "Set it up in Administration → System → Email."
}

// Apply validates, saves and activates next. A nil password keeps the stored
// one. It holds the lock for the whole save, so two admins cannot mix values.
func (s *Service) Apply(next smtp.Config, password *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next.Password = s.conf.Password
	if password != nil {
		next.Password = *password
	}
	if err := next.Validate(); err != nil {
		return err
	}
	if err := s.cfg.Live.SetMail(next); err != nil {
		return err
	}
	s.conf, s.problem = next, ""
	s.sender = s.NewMailer(next)
	return nil
}

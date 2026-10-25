// Package mailer sends plain-text email through an SMTP server. It is a leaf:
// it knows nothing of users, spaces or HTTP.
package smtp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	gomail "github.com/wneessen/go-mail"

	"savvy-go/internal/config"
)

// Protocols a server can be reached with.
const (
	STARTTLS = "starttls" // plain connection upgraded with STARTTLS (usually port 587); required
	SMTPS    = "smtps"    // TLS from the first byte (usually port 465)
	Plain    = "smtp"     // no encryption (port 25); for a trusted local relay only
)

// ErrNotConfigured is returned by a mailer that has no server.
var ErrNotConfigured = errors.New("mail is not configured")

// Config describes one SMTP server.
type Config struct {
	Enabled    bool
	Host       string
	Port       int
	Protocol   string
	User       string
	Password   string
	From       string // "Name <addr>" or "addr"
	SkipVerify bool   // accept any certificate; for self-signed servers only
	Timeout    time.Duration
}

// DefaultTimeout is the connection timeout when none is set.
const DefaultTimeout = 10 * time.Second

// InvalidError is a mistake in an enabled configuration. Field is the config
// key it is about ("host", "port", ...), so a form can show it by the field.
type InvalidError = config.FieldError

func invalid(field, format string, args ...any) error {
	return InvalidError{Field: field, Message: fmt.Sprintf(format, args...)}
}

// Validate reports what is wrong with an enabled configuration (an InvalidError).
// A disabled one is always valid.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	switch {
	case strings.TrimSpace(c.Host) == "":
		return invalid("host", "mailer.host is empty")
	case c.Port < 1 || c.Port > 65535:
		return invalid("port", "mailer.port %d is not a valid port", c.Port)
	case c.Protocol != STARTTLS && c.Protocol != SMTPS && c.Protocol != Plain:
		return invalid("protocol", "mailer.protocol %q must be %q, %q or %q", c.Protocol, STARTTLS, SMTPS, Plain)
	case c.User != "" && c.Password == "":
		return invalid("password", "mailer.password is empty but mailer.user is set")
	}
	if _, err := mail.ParseAddress(c.From); err != nil {
		return invalid("from", "mailer.from %q is not an email address", c.From)
	}
	return nil
}

// Message is one email.
type Message struct {
	To      string
	Subject string
	Text    string
}

// Mailer sends email. Callers hold the interface so tests can record messages.
type Mailer interface {
	// Enabled is false when no server is configured; Send then fails with ErrNotConfigured.
	Enabled() bool
	// Send delivers the message to the server.
	Send(ctx context.Context, m Message) error
	// Check connects and authenticates without sending anything.
	Check(ctx context.Context) error
}

// New returns the SMTP mailer for c, or a disabled one when c is not enabled.
// c must be valid.
func New(c Config) Mailer {
	if !c.Enabled {
		return Disabled{}
	}
	return &server{cfg: c}
}

// Disabled is the mailer of an instance without a mail server.
type Disabled struct{}

func (Disabled) Enabled() bool                       { return false }
func (Disabled) Send(context.Context, Message) error { return ErrNotConfigured }
func (Disabled) Check(context.Context) error         { return ErrNotConfigured }

type server struct{ cfg Config }

func (s *server) Enabled() bool { return true }

func (s *server) client() (*gomail.Client, error) {
	opts := []gomail.Option{
		gomail.WithPort(s.cfg.Port),
		gomail.WithTimeout(s.cfg.Timeout),
	}
	switch s.cfg.Protocol {
	case SMTPS:
		opts = append(opts, gomail.WithSSL())
	case Plain:
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	default:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}
	if s.cfg.User != "" {
		// Autodiscovery never offers a password mechanism on an unencrypted
		// connection; the admin chose "smtp" on purpose, so name one.
		mech := gomail.SMTPAuthAutoDiscover
		if s.cfg.Protocol == Plain {
			mech = gomail.SMTPAuthPlain
		}
		opts = append(opts,
			gomail.WithSMTPAuth(mech),
			gomail.WithUsername(s.cfg.User),
			gomail.WithPassword(s.cfg.Password),
		)
	}
	if s.cfg.Protocol != Plain {
		opts = append(opts, gomail.WithTLSConfig(&tls.Config{
			ServerName:         s.cfg.Host,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: s.cfg.SkipVerify, //nolint:gosec // opt-in for self-signed servers
		}))
	}
	return gomail.NewClient(s.cfg.Host, opts...)
}

func (s *server) Send(ctx context.Context, m Message) error {
	msg := gomail.NewMsg()
	if err := msg.From(s.cfg.From); err != nil {
		return fmt.Errorf("sender: %w", err)
	}
	if err := msg.To(m.To); err != nil {
		return fmt.Errorf("recipient: %w", err)
	}
	msg.Subject(m.Subject)
	msg.SetBodyString(gomail.TypeTextPlain, m.Text)
	c, err := s.client()
	if err != nil {
		return err
	}
	return c.DialAndSendWithContext(ctx, msg)
}

func (s *server) Check(ctx context.Context) error {
	c, err := s.client()
	if err != nil {
		return err
	}
	if err := c.DialWithContext(ctx); err != nil {
		return err
	}
	return c.Close()
}

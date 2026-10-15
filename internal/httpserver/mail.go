package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/mailer"
)

// WithMailer sets how email is sent. The default is the SMTP server of the
// [mailer] section of the config file; tests pass a recorder.
func WithMailer(m mailer.Mailer) Option { return func(s *Server) { s.mail = m } }

// mailer is the current sender. It changes when an admin saves new settings.
func (s *Server) mailer() mailer.Mailer {
	s.mailMu.RLock()
	defer s.mailMu.RUnlock()
	return s.mail
}

// mailSettings is the current SMTP configuration and why it is switched off, if
// a saved section was invalid.
func (s *Server) mailSettings() (mailer.Config, string) {
	s.mailMu.RLock()
	defer s.mailMu.RUnlock()
	return s.mailCfg, s.mailProblem
}

const defaultMailTimeout = 10 * time.Second

// sendMail sends one email, giving up after a few connection timeouts.
func (s *Server) sendMail(ctx context.Context, m mailer.Message) error {
	cfg, _ := s.mailSettings()
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultMailTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, 3*timeout)
	defer cancel()
	return s.mailer().Send(ctx, m)
}

// mailIndex shows the mail configuration. The password is never sent, only
// whether one is set. "writable" is false when the config file is read-only.
func (s *Server) mailIndex(w http.ResponseWriter, r *http.Request) {
	c, problem := s.mailSettings()
	writeData(w, http.StatusOK, map[string]any{
		"enabled":     s.mailer().Enabled(),
		"problem":     nullIfEmpty(problem),
		"host":        c.Host,
		"port":        c.Port,
		"protocol":    c.Protocol,
		"user":        c.User,
		"from":        c.From,
		"skipVerify":  c.SkipVerify,
		"timeout":     int(c.Timeout / time.Second),
		"passwordSet": c.Password != "",
		"appUrlSet":   s.cfg.URL() != "",
		"writable":    s.cfg.Live.Writable(),
	})
}

// mailUpdate replaces the whole mail configuration, writes it to the config
// file and starts using it at once. A missing "password" keeps the stored one;
// an empty string clears it. An enabled configuration must be valid; a disabled
// one is saved as it is, so it can be filled in before it is switched on.
func (s *Server) mailUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled    bool    `json:"enabled"`
		Host       string  `json:"host"`
		Port       int     `json:"port"`
		Protocol   string  `json:"protocol"`
		User       string  `json:"user"`
		Password   *string `json:"password"`
		From       string  `json:"from"`
		SkipVerify bool    `json:"skipVerify"`
		Timeout    int     `json:"timeout"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !s.cfg.Live.Writable() {
		writeMessage(w, http.StatusUnprocessableEntity, "The config file is read-only. Set the [mailer] section in config.toml.")
		return
	}
	next := mailer.Config{
		Enabled:    body.Enabled,
		Host:       strings.TrimSpace(body.Host),
		Port:       body.Port,
		Protocol:   strings.ToLower(strings.TrimSpace(body.Protocol)),
		User:       body.User,
		From:       strings.TrimSpace(body.From),
		SkipVerify: body.SkipVerify,
		Timeout:    time.Duration(body.Timeout) * time.Second,
	}
	if next.Timeout <= 0 {
		next.Timeout = defaultMailTimeout
	}
	if err := s.applyMail(next, body.Password); err != nil {
		var invalid mailer.InvalidError
		if errors.As(err, &invalid) {
			writeValidation(w, map[string][]string{invalid.Field: {invalid.Error()}})
			return
		}
		writeMessage(w, http.StatusUnprocessableEntity, "Could not save the config file: "+err.Error())
		return
	}
	s.mailIndex(w, r)
}

// applyMail validates, saves and activates next. A nil password keeps the stored
// one. It holds the lock for the whole save, so two admins cannot mix values.
func (s *Server) applyMail(next mailer.Config, password *string) error {
	s.mailMu.Lock()
	defer s.mailMu.Unlock()
	next.Password = s.mailCfg.Password
	if password != nil {
		next.Password = *password
	}
	if err := next.Validate(); err != nil {
		return err
	}
	if err := s.cfg.Live.SetMail(next); err != nil {
		return err
	}
	s.mailCfg, s.mailProblem = next, ""
	s.mail = s.newMailer(next)
	return nil
}

// mailTest checks the connection and sends a test message to the given address
// or, by default, to the admin who asked.
func (s *Server) mailTest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To string `json:"to"`
	}
	if r.ContentLength != 0 && !decodeBody(w, r, &body) {
		return
	}
	to := strings.TrimSpace(body.To)
	if to == "" {
		to = userFrom(r).Email
	}
	if !looksLikeEmail(to) {
		writeValidation(w, map[string][]string{"to": {"The email field must be a valid email address."}})
		return
	}
	if !s.mailer().Enabled() {
		writeMessage(w, http.StatusUnprocessableEntity, s.mailOffReason())
		return
	}
	cfg, _ := s.mailSettings()
	ctx, cancel := context.WithTimeout(r.Context(), 3*max(cfg.Timeout, time.Second))
	defer cancel()
	if err := s.mailer().Check(ctx); err != nil {
		writeMessage(w, http.StatusUnprocessableEntity, "Could not connect to the mail server: "+err.Error())
		return
	}
	err := s.sendMail(r.Context(), mailer.Message{
		To:      to,
		Subject: "Go Savvy test email",
		Text:    "This is a test email from Go Savvy. Your mail settings work.\n",
	})
	if err != nil {
		writeMessage(w, http.StatusUnprocessableEntity, "The mail server did not accept the message: "+err.Error())
		return
	}
	writeData(w, http.StatusOK, map[string]any{"to": to})
}

// mailOffReason says why mail cannot be used.
func (s *Server) mailOffReason() string {
	if _, problem := s.mailSettings(); problem != "" {
		return "Mail is switched off: " + problem + "."
	}
	return "Mail is not configured. Set it up in Administration → System → Email."
}

const noAppURLMessage = "The public URL is not set, so the link would point nowhere. Set app_url first."

// resetMessage is the email with a one-time link to choose a new password.
func resetMessage(u *auth.User, link string, expires time.Time) mailer.Message {
	return mailer.Message{
		To:      u.Email,
		Subject: "Set your Go Savvy password",
		Text: fmt.Sprintf("Hello %s,\n\nAn administrator asked to reset your Go Savvy password. "+
			"Open this link to choose a new one:\n\n%s\n\n"+
			"The link works once and expires on %s. If you did not expect this email, you can ignore it.\n",
			oneLine(u.Name), link, expires.UTC().Format("2006-01-02 15:04 UTC")),
	}
}

// inviteMessage is the email with the link to join a space.
func inviteMessage(to, inviter, space, role, link string) mailer.Message {
	return mailer.Message{
		To:      to,
		Subject: "You are invited to " + oneLine(space) + " on Go Savvy",
		Text: fmt.Sprintf("%s invited you to the space \"%s\" on Go Savvy as %s.\n\n"+
			"Open this link to join:\n\n%s\n\nIf you did not expect this email, you can ignore it.\n",
			oneLine(inviter), oneLine(space), role, link),
	}
}

// oneLine keeps a name from breaking out of a header line.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

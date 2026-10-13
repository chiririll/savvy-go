package httpserver

import (
	"context"
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

const defaultMailTimeout = 10 * time.Second

// sendMail sends one email, giving up after a few connection timeouts.
func (s *Server) sendMail(ctx context.Context, m mailer.Message) error {
	timeout := s.cfg.Mail.Timeout
	if timeout <= 0 {
		timeout = defaultMailTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, 3*timeout)
	defer cancel()
	return s.mail.Send(ctx, m)
}

// mailIndex shows the mail configuration. It is read-only: the settings live in
// the config file, and the password is never sent, only whether one is set.
func (s *Server) mailIndex(w http.ResponseWriter, r *http.Request) {
	c := s.cfg.Mail
	writeData(w, http.StatusOK, map[string]any{
		"enabled":     s.mail.Enabled(),
		"problem":     nullIfEmpty(s.cfg.MailProblem),
		"host":        c.Host,
		"port":        c.Port,
		"protocol":    c.Protocol,
		"user":        c.User,
		"from":        c.From,
		"skipVerify":  c.SkipVerify,
		"passwordSet": c.Password != "",
		"appUrlSet":   s.cfg.URL() != "",
	})
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
	if !s.mail.Enabled() {
		writeMessage(w, http.StatusUnprocessableEntity, s.mailOffReason())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*max(s.cfg.Mail.Timeout, time.Second))
	defer cancel()
	if err := s.mail.Check(ctx); err != nil {
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
	if s.cfg.MailProblem != "" {
		return "Mail is switched off: " + s.cfg.MailProblem + "."
	}
	return "Mail is not configured. Add a [mailer] section to the config file and restart."
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

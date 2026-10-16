package mailapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/mailing"
	"savvy-go/internal/mailer"
)

// Handler is the admin's view of the mail settings.
type Handler struct{ Mail *mailing.Service }

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// index shows the mail configuration. The password is never sent, only
// whether one is set. "writable" is false when the config file is read-only.
func (h Handler) index(w http.ResponseWriter, r *http.Request) {
	c, problem := h.Mail.Settings()
	api.WriteData(w, http.StatusOK, map[string]any{
		"enabled":     h.Mail.Mailer().Enabled(),
		"problem":     nullIfEmpty(problem),
		"host":        c.Host,
		"port":        c.Port,
		"protocol":    c.Protocol,
		"user":        c.User,
		"from":        c.From,
		"skipVerify":  c.SkipVerify,
		"timeout":     int(c.Timeout / time.Second),
		"passwordSet": c.Password != "",
		"appUrlSet":   h.Mail.Config().URL() != "",
		"writable":    h.Mail.Config().Live.Writable(),
	})
}

// update replaces the whole mail configuration, writes it to the config
// file and starts using it at once. A missing "password" keeps the stored one;
// an empty string clears it. An enabled configuration must be valid; a disabled
// one is saved as it is, so it can be filled in before it is switched on.
func (h Handler) update(w http.ResponseWriter, r *http.Request) {
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
	if !api.DecodeBody(w, r, &body) {
		return
	}
	if !h.Mail.Config().Live.Writable() {
		api.WriteMessage(w, http.StatusUnprocessableEntity, "The config file is read-only. Set the [mailer] section in config.toml.")
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
		next.Timeout = mailing.DefaultTimeout
	}
	if err := h.Mail.Apply(next, body.Password); err != nil {
		var invalid mailer.InvalidError
		if errors.As(err, &invalid) {
			api.WriteValidation(w, map[string][]string{invalid.Field: {invalid.Error()}})
			return
		}
		api.WriteMessage(w, http.StatusUnprocessableEntity, "Could not save the config file: "+err.Error())
		return
	}
	h.index(w, r)
}

// test checks the connection and sends a test message to the given address
// or, by default, to the admin who asked.
func (h Handler) test(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To string `json:"to"`
	}
	if r.ContentLength != 0 && !api.DecodeBody(w, r, &body) {
		return
	}
	to := strings.TrimSpace(body.To)
	if to == "" {
		to = api.User(r).Email
	}
	if !api.LooksLikeEmail(to) {
		api.WriteValidation(w, map[string][]string{"to": {"The email field must be a valid email address."}})
		return
	}
	if !h.Mail.Mailer().Enabled() {
		api.WriteMessage(w, http.StatusUnprocessableEntity, h.Mail.OffReason())
		return
	}
	cfg, _ := h.Mail.Settings()
	ctx, cancel := context.WithTimeout(r.Context(), 3*max(cfg.Timeout, time.Second))
	defer cancel()
	if err := h.Mail.Mailer().Check(ctx); err != nil {
		api.WriteMessage(w, http.StatusUnprocessableEntity, "Could not connect to the mail server: "+err.Error())
		return
	}
	err := h.Mail.Send(r.Context(), mailer.Message{
		To:      to,
		Subject: "Go Savvy test email",
		Text:    "This is a test email from Go Savvy. Your mail settings work.\n",
	})
	if err != nil {
		api.WriteMessage(w, http.StatusUnprocessableEntity, "The mail server did not accept the message: "+err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"to": to})
}

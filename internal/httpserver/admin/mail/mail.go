package mail

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/mailer"
	"savvy-go/internal/mailing"
)

// Handler is the admin's view of the mail settings.
type Handler struct{ Mail *mailing.Service }

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// settingsBody is the mail configuration without its password.
type settingsBody struct {
	Enabled     bool    `json:"enabled"`
	Problem     *string `json:"problem"`
	Host        string  `json:"host"`
	Port        int     `json:"port"`
	Protocol    string  `json:"protocol"`
	User        string  `json:"user"`
	From        string  `json:"from"`
	SkipVerify  bool    `json:"skipVerify"`
	Timeout     int     `json:"timeout"`
	PasswordSet bool    `json:"passwordSet"`
	AppURLSet   bool    `json:"appUrlSet"`
	Writable    bool    `json:"writable"`
}

// testedBody names the address a test message went to.
type testedBody struct {
	To string `json:"to"`
}

// index shows the mail configuration. The password is never sent, only
// whether one is set. "writable" is false when the config file is read-only.
func (h Handler) index(w http.ResponseWriter, r *http.Request) {
	c, problem := h.Mail.Settings()
	httpx.WriteData(w, http.StatusOK, settingsBody{
		Enabled: h.Mail.Mailer().Enabled(), Problem: nullIfEmpty(problem),
		Host: c.Host, Port: c.Port, Protocol: c.Protocol, User: c.User, From: c.From,
		SkipVerify: c.SkipVerify, Timeout: int(c.Timeout / time.Second), PasswordSet: c.Password != "",
		AppURLSet: h.Mail.Config().URL() != "", Writable: h.Mail.Config().Live.Writable(),
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
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	if !h.Mail.Config().Live.Writable() {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "The config file is read-only. Set the [mailer] section in config.toml.")
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
			httpx.WriteValidation(w, map[string][]string{invalid.Field: {invalid.Error()}})
			return
		}
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Could not save the config file: "+err.Error())
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
	if r.ContentLength != 0 && !httpx.DecodeBody(w, r, &body) {
		return
	}
	to := strings.TrimSpace(body.To)
	if to == "" {
		to = httpx.User(r).Email
	}
	if !httpx.LooksLikeEmail(to) {
		httpx.WriteValidation(w, map[string][]string{"to": {"The email field must be a valid email address."}})
		return
	}
	if !h.Mail.Mailer().Enabled() {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, h.Mail.OffReason())
		return
	}
	cfg, _ := h.Mail.Settings()
	ctx, cancel := context.WithTimeout(r.Context(), 3*max(cfg.Timeout, time.Second))
	defer cancel()
	if err := h.Mail.Mailer().Check(ctx); err != nil {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Could not connect to the mail server: "+err.Error())
		return
	}
	err := h.Mail.Send(r.Context(), mailer.Message{
		To:      to,
		Subject: "Go Savvy test email",
		Text:    "This is a test email from Go Savvy. Your mail settings work.\n",
	})
	if err != nil {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "The mail server did not accept the message: "+err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, testedBody{To: to})
}

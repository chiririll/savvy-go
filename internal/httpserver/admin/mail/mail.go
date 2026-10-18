package mail

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/mail"
	"savvy-go/internal/mail/smtp"
)

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
// whether one is set. "writable" is false when the config file cannot be changed from the
// application: the settings are then shown, not edited.
func (h Handler) index(w http.ResponseWriter, r *http.Request) {
	c, problem := h.Mail.Config()
	httpx.WriteData(w, http.StatusOK, settingsBody{
		Enabled: h.Mail.Mailer().Enabled(), Problem: nullIfEmpty(problem),
		Host: c.Host, Port: c.Port, Protocol: c.Protocol, User: c.User, From: c.From,
		SkipVerify: c.SkipVerify, Timeout: int(c.Timeout / time.Second), PasswordSet: c.Password != "",
		AppURLSet: h.Mail.AppURL() != "", Writable: h.Writable(),
	})
}

// inputBody is every mail setting, as the panel sends them. Without a password the stored one
// is kept; an empty one clears it.
type inputBody struct {
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

func (b inputBody) settings() mail.Settings {
	return mail.Settings{
		Enabled: b.Enabled, Host: b.Host, Port: b.Port, Protocol: b.Protocol, User: b.User,
		Password: b.Password, From: b.From, SkipVerify: b.SkipVerify, Timeout: b.Timeout,
	}
}

// check tries the settings in the body, saved or not, by connecting and signing in. Nothing is
// saved or sent.
func (h Handler) check(w http.ResponseWriter, r *http.Request) {
	var body inputBody
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	err := h.Mail.Check(r.Context(), body.settings())
	var invalid smtp.InvalidError
	switch {
	case err == nil:
		httpx.WriteData(w, http.StatusOK, map[string]bool{"ok": true})
	case errors.As(err, &invalid):
		httpx.WriteConfigError(w, err)
	default:
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Could not connect to the mail server: "+err.Error())
	}
}

// update saves the mail settings to the config file and applies them at once. An enabled
// configuration that cannot work is refused with the field at fault.
func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	var body inputBody
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	err := h.Mail.Save(body.settings())
	if err != nil {
		httpx.WriteConfigError(w, err)
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
	cfg, _ := h.Mail.Config()
	ctx, cancel := context.WithTimeout(r.Context(), 3*max(cfg.Timeout, time.Second))
	defer cancel()
	if err := h.Mail.Mailer().Check(ctx); err != nil {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Could not connect to the mail server: "+err.Error())
		return
	}
	err := h.Mail.Send(r.Context(), smtp.Message{
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

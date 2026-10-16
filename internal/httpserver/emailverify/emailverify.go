// Package emailverify confirms email addresses: the emailed link, and the setting that requires it.
package emailverify

import (
	"context"
	"encoding/json"
	"net/http"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/mailing"

	"savvy-go/internal/auth"
	"savvy-go/internal/mailer"
)

// ErrUnverified is the error code of a request held back until the address is confirmed.
const ErrUnverified = "email_unverified"

// Handler holds what the confirmation routes need.
type Handler struct {
	Cfg           config.Config
	Mail          *mailing.Service
	Verifications auth.EmailVerifications
	Users         auth.Users
}

// CanRequire refuses to switch the requirement on when it
// would lock the admin out: links need mail and a public URL, and the admin
// switching it on must have confirmed their own address first.
func (h Handler) CanRequire(w http.ResponseWriter, r *http.Request) bool {
	if !h.Mail.Mailer().Enabled() || h.Cfg.URL() == "" {
		message := h.Mail.OffReason()
		if h.Mail.Mailer().Enabled() {
			message = mailing.NoAppURLMessage
		}
		api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Email verification needs working mail and a public URL. " + message,
			"error":   "mail_required",
		})
		return false
	}
	if !api.User(r).EmailVerified {
		api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Confirm your own email address first, or you would lock yourself out.",
			"error":   ErrUnverified,
		})
		return false
	}
	return true
}

// Send emails u a link that confirms their address. It says what
// is missing when mail cannot be sent.
func (h Handler) Send(ctx context.Context, u *auth.User) string {
	switch {
	case !h.Mail.Mailer().Enabled():
		return h.Mail.OffReason()
	case h.Cfg.URL() == "":
		return mailing.NoAppURLMessage
	}
	token, exp, err := h.Verifications.Issue(ctx, u)
	if err != nil {
		return "Could not create the link: " + err.Error()
	}
	if err := h.Mail.Send(ctx, mailer.EmailVerification(u.Email, u.Name, h.Cfg.URL()+"/verify-email/"+token, exp)); err != nil {
		return "Could not send the email: " + err.Error()
	}
	return ""
}

// verifyEmailResend sends the signed-in user a new confirmation link.
func (h Handler) verifyEmailResend(w http.ResponseWriter, r *http.Request) {
	u := api.User(r)
	if u.EmailVerified {
		api.WriteMessage(w, http.StatusUnprocessableEntity, "Your email address is already confirmed.")
		return
	}
	if problem := h.Send(r.Context(), u); problem != "" {
		api.WriteMessage(w, http.StatusUnprocessableEntity, problem)
		return
	}
	api.WriteMessage(w, http.StatusOK, "Confirmation email sent.")
}

// verifyEmail confirms the address named by the emailed token. It needs no
// session: the link may be opened on another device.
func (h Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	invalid := map[string][]string{"token": {"This link is invalid or has expired."}}
	u, err := h.Verifications.Consume(r.Context(), body.Token)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if u == nil {
		api.WriteValidation(w, invalid)
		return
	}
	if err := h.Users.SetEmailVerified(r.Context(), u.ID, true); err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteMessage(w, http.StatusOK, "Email address confirmed.")
}

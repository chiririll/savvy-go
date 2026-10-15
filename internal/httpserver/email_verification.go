package httpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"savvy-go/internal/auth"
	"savvy-go/internal/mailer"
)

const (
	settingRequireVerification = "require_email_verification"
	errEmailUnverified         = "email_unverified"
)

func (s *Server) emailVerificationRequired(ctx context.Context) bool {
	return s.settings.Bool(ctx, settingRequireVerification, false)
}

// verificationExempt are the routes an unverified user may still call while
// verification is required: leaving, and asking for a new link.
var verificationExempt = map[string]bool{
	"/api/auth/logout":              true,
	"/api/auth/verify-email/resend": true,
}

// requireVerifiedEmail holds back, while the instance requires it, every
// signed-in user whose address is not confirmed. It covers API tokens, so a
// token cannot do what its owner could not.
func (s *Server) requireVerifiedEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r)
		if u != nil && !u.EmailVerified && !verificationExempt[r.URL.Path] && s.emailVerificationRequired(r.Context()) {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"message": "Confirm your email address to continue.",
				"error":   errEmailUnverified,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// checkCanRequireVerification refuses to switch the requirement on when it
// would lock the admin out: links need mail and a public URL, and the admin
// switching it on must have confirmed their own address first.
func (s *Server) checkCanRequireVerification(w http.ResponseWriter, r *http.Request) bool {
	if !s.mailer().Enabled() || s.cfg.URL() == "" {
		message := s.mailOffReason()
		if s.mailer().Enabled() {
			message = noAppURLMessage
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Email verification needs working mail and a public URL. " + message,
			"error":   "mail_required",
		})
		return false
	}
	if !userFrom(r).EmailVerified {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Confirm your own email address first, or you would lock yourself out.",
			"error":   errEmailUnverified,
		})
		return false
	}
	return true
}

// sendVerification emails u a link that confirms their address. It says what
// is missing when mail cannot be sent.
func (s *Server) sendVerification(ctx context.Context, u *auth.User) string {
	switch {
	case !s.mailer().Enabled():
		return s.mailOffReason()
	case s.cfg.URL() == "":
		return noAppURLMessage
	}
	token, exp, err := s.verifications.Issue(ctx, u)
	if err != nil {
		return "Could not create the link: " + err.Error()
	}
	if err := s.sendMail(ctx, mailer.EmailVerification(u.Email, u.Name, s.cfg.URL()+"/verify-email/"+token, exp)); err != nil {
		return "Could not send the email: " + err.Error()
	}
	return ""
}

// verifyEmailResend sends the signed-in user a new confirmation link.
func (s *Server) verifyEmailResend(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if u.EmailVerified {
		writeMessage(w, http.StatusUnprocessableEntity, "Your email address is already confirmed.")
		return
	}
	if problem := s.sendVerification(r.Context(), u); problem != "" {
		writeMessage(w, http.StatusUnprocessableEntity, problem)
		return
	}
	writeMessage(w, http.StatusOK, "Confirmation email sent.")
}

// verifyEmail confirms the address named by the emailed token. It needs no
// session: the link may be opened on another device.
func (s *Server) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	invalid := map[string][]string{"token": {"This link is invalid or has expired."}}
	u, err := s.verifications.Consume(r.Context(), body.Token)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if u == nil {
		writeValidation(w, invalid)
		return
	}
	if err := s.users.SetEmailVerified(r.Context(), u.ID, true); err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeMessage(w, http.StatusOK, "Email address confirmed.")
}

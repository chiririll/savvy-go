// Package signin is what every way of signing in shares: starting a session,
// the answer the SPA expects, and the instance policy on how people may sign in.
package signin

import (
	"context"
	"net/http"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/settings"
	"savvy-go/internal/sso"
)

// SettingRequireVerification is the instance setting that holds back users
// whose email address is not confirmed.
const SettingRequireVerification = "require_email_verification"

type Service struct {
	Cfg      config.Config
	Sessions auth.Sessions
	Settings settings.Store
	SSO      *sso.Service
}

// Issue starts a session for u and answers with it.
func (s Service) Issue(w http.ResponseWriter, r *http.Request, u *auth.User, status int, remember bool) {
	issued, err := s.Sessions.Issue(r.Context(), u, r.RemoteAddr, r.UserAgent(), remember)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	auth.SetSessionCookies(w, r, s.Cfg, issued.Token, issued.CSRF, remember)
	s.WritePayload(w, r, status, u, issued.Session)
}

// WritePayload answers with the user and when the session ends.
func (s Service) WritePayload(w http.ResponseWriter, r *http.Request, status int, u *auth.User, sess *auth.Session) {
	var refresh *string // null when the session does not rotate
	if at := sess.RefreshDueAt(); at != nil {
		iso := at.UTC().Format(time.RFC3339)
		refresh = &iso
	}
	api.WriteJSON(w, status, map[string]any{
		"user":       u.SessionJSON(),
		"expires_at": sess.AbsoluteExpiresAt.UTC().Format(time.RFC3339),
		"refresh_at": refresh,
		// The SPA shows the "confirm your email" page instead of the app.
		"email_verification_required": s.EmailRequired(r.Context()),
	})
}

// PasswordLoginDisabled is true when an admin turned password sign-in off and
// at least one SSO provider is there to replace it.
func (s Service) PasswordLoginDisabled(ctx context.Context) bool {
	if s.Settings.Bool(ctx, "password_login_enabled", true) {
		return false
	}
	return s.SSO.AnyEnabled(ctx)
}

// EmailRequired is true while the instance holds back unverified users.
func (s Service) EmailRequired(ctx context.Context) bool {
	return s.Settings.Bool(ctx, SettingRequireVerification, false)
}

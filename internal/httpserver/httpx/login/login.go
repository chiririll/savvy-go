// Package login is what every way of signing in shares: starting a session and the
// answer the SPA expects. The policy on how people may sign in is in internal/app/signin.
package login

import (
	"net/http"
	"time"

	"savvy-go/internal/app/signin"
	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
)

// Service starts sessions. It embeds the sign-in policy, so callers ask it the
// policy questions too.
type Service struct {
	signin.Policy
	Cfg      config.Config
	Sessions auth.Sessions
}

// Issue starts a session for u and answers with it.
func (s Service) Issue(w http.ResponseWriter, r *http.Request, u *auth.User, status int, remember bool) {
	issued, err := s.Sessions.Issue(r.Context(), u, r.RemoteAddr, r.UserAgent(), remember)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	auth.SetSessionCookies(w, r, s.Cfg, issued.Token, issued.CSRF, remember)
	s.WritePayload(w, r, status, u, issued.Session)
}

// WritePayload answers with the user and when the session ends.
func (s Service) WritePayload(w http.ResponseWriter, r *http.Request, status int, u *auth.User, sess *auth.Session) {
	user := dto.NewSessionUser(*u)
	expires := sess.AbsoluteExpiresAt.UTC().Truncate(time.Second)
	var refresh *time.Time // null when the session does not rotate
	if at := sess.RefreshDueAt(); at != nil {
		t := at.UTC().Truncate(time.Second)
		refresh = &t
	}
	httpx.WriteJSON(w, status, dto.SessionPayload{
		User: &user, ExpiresAt: &expires, RefreshAt: refresh,
		// The SPA shows the "confirm your email" page instead of the app.
		EmailVerificationRequired: s.EmailRequired(r.Context()),
	})
}

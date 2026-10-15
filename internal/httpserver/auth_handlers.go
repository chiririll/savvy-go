package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/config"
	"savvy-go/internal/store"

	"github.com/go-chi/chi/v5"
)

func (s *Server) passwordLoginDisabled(r *http.Request) bool {
	if s.settings.Bool(r.Context(), "password_login_enabled", true) {
		return false
	}
	return s.enabledSSOExists(r)
}

func (s *Server) enabledSSOExists(r *http.Request) bool {
	return s.sso.AnyEnabled(r.Context())
}

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	n, err := s.users.Count(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Tells the setup page where to find the first-run password; the password
	// itself never leaves the server.
	var hint any
	if n == 0 && s.bootstrap.Active() {
		h := s.bootstrap.Hint()
		hint = map[string]any{"environment": h.Environment, "file": h.File}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"needs_registration":     n == 0,
		"password_login_enabled": !s.passwordLoginDisabled(r),
		"bootstrap_hint":         hint,
	})
}

// bootstrapVerify tells the setup page whether the bootstrap password is right,
// without using it up; register still checks it again.
func (s *Server) bootstrapVerify(w http.ResponseWriter, r *http.Request) {
	n, err := s.users.Count(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 || !s.bootstrap.Active() {
		writeMessage(w, http.StatusForbidden, "Registration is closed.")
		return
	}
	var body struct {
		BootstrapPassword string `json:"bootstrap_password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if s.bootstrap.Verify(body.BootstrapPassword) != nil {
		writeValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// authRegister creates the first admin. Registration is never open: it needs
// the bootstrap password, and only while the server has no users.
func (s *Server) authRegister(w http.ResponseWriter, r *http.Request) {
	n, err := s.users.Count(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 || !s.bootstrap.Active() {
		writeMessage(w, http.StatusForbidden, "Registration is closed.")
		return
	}
	var body struct {
		Name              string `json:"name"`
		Email             string `json:"email"`
		Password          string `json:"password"`
		BootstrapPassword string `json:"bootstrap_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	errs := map[string][]string{}
	if strings.TrimSpace(body.Name) == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if !looksLikeEmail(body.Email) {
		errs["email"] = []string{"The email field must be a valid email address."}
	}
	if len(body.Password) < 6 {
		errs["password"] = []string{"The password must be at least 6 characters."}
	}
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	var u *auth.User
	// Verify, create and invalidate are one step, so two parallel requests
	// with the right password still create one admin.
	err = s.bootstrap.Register(body.BootstrapPassword, func() error {
		if n, err := s.users.Count(r.Context()); err != nil {
			return err
		} else if n > 0 {
			return bootstrap.ErrClosed
		}
		pass := body.Password
		var err error
		if u, err = s.users.Create(r.Context(), strings.TrimSpace(body.Name), body.Email, &pass, auth.RoleAdmin); err != nil {
			return errEmailTaken
		}
		return nil
	})
	switch {
	case errors.Is(err, bootstrap.ErrWrongPassword):
		writeValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
	case errors.Is(err, bootstrap.ErrClosed):
		s.bootstrap.Invalidate()
		writeMessage(w, http.StatusForbidden, "Registration is closed.")
	case errors.Is(err, errEmailTaken):
		writeValidation(w, map[string][]string{"email": {"The email has already been taken."}})
	case err != nil:
		writeMessage(w, http.StatusInternalServerError, err.Error())
	default:
		// The admin is taken through server setup, also after a reload or on another device.
		if err := s.settings.Set(r.Context(), "onboarding_server_done", false); err != nil {
			slog.Warn("mark server setup pending", "err", err)
		}
		s.issueSession(w, r, u, 201, true)
	}
}

var errEmailTaken = errors.New("email taken")

func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	if s.passwordLoginDisabled(r) {
		writeValidation(w, map[string][]string{"email": {"Password sign-in is disabled. Use single sign-on instead."}})
		return
	}
	u, err := s.users.ByEmail(r.Context(), body.Email)
	if err != nil || u == nil || u.IsSSOOnly || u.IsInactive() || u.Password == nil || !auth.CheckPassword(*u.Password, body.Password) {
		writeValidation(w, map[string][]string{"email": {"Invalid credentials."}})
		return
	}
	if u.HasTwoFactor() {
		tok, err := s.challenges.Issue(r.Context(), u)
		if err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"requires_2fa":     true,
			"two_factor_token": tok,
		})
		return
	}
	s.issueSession(w, r, u, http.StatusOK, body.RememberMe)
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	token := auth.ReadToken(r, s.cfg)
	sess, err := s.sessions.Resolve(r.Context(), token)
	if err != nil || sess == nil || sess.User == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"user":       nil,
			"expires_at": nil,
			"refresh_at": nil,
		})
		return
	}
	refreshed, err := s.sessions.MaybeRefresh(r.Context(), sess)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if refreshed != nil {
		auth.SetSessionCookies(w, r, s.cfg, refreshed.Token, refreshed.CSRF, true)
	}
	s.writeSessionPayload(w, r, http.StatusOK, sess.User, sess)
}

func (s *Server) authChangePassword(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if u.IsSSOOnly || u.Password == nil {
		writeMessage(w, http.StatusUnprocessableEntity, "This account uses SSO and does not have a password.")
		return
	}
	var body struct {
		Current              string `json:"current_password"`
		Password             string `json:"password"`
		PasswordConfirmation string `json:"password_confirmation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"password": {"The given data was invalid."}})
		return
	}
	if !auth.CheckPassword(*u.Password, body.Current) {
		writeValidation(w, map[string][]string{"current_password": {"Current password is incorrect."}})
		return
	}
	if len(body.Password) < 8 {
		writeValidation(w, map[string][]string{"password": {"The password must be at least 8 characters."}})
		return
	}
	if body.Password != body.PasswordConfirmation {
		writeValidation(w, map[string][]string{"password": {"The password confirmation does not match."}})
		return
	}
	if err := s.users.UpdatePassword(r.Context(), u.ID, body.Password); err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sess := sessionFrom(r); sess != nil {
		_, _ = s.sessions.RevokeOthers(r.Context(), u.ID, sess.ID)
	}
	writeMessage(w, http.StatusOK, "Password updated.")
}

func (s *Server) authLogoutOthers(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	sess := sessionFrom(r)
	revoked := 0
	if sess != nil {
		revoked, _ = s.sessions.RevokeOthers(r.Context(), u.ID, sess.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Signed out of other devices.",
		"revoked": revoked,
	})
}

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	if sess := sessionFrom(r); sess != nil {
		_ = s.sessions.Revoke(r.Context(), sess)
	}
	auth.ClearSessionCookies(w, r, s.cfg)
	writeMessage(w, http.StatusOK, "Logged out.")
}

func (s *Server) passwordPreview(w http.ResponseWriter, r *http.Request) {
	row, err := s.tokens.Preview(r.Context(), chi.URLParam(r, "token"))
	if err != nil || row == nil || row.User == nil {
		writeMessage(w, http.StatusNotFound, "This link is invalid or has expired.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       row.User.Name,
		"email":      row.User.Email,
		"isInactive": row.User.IsInactive(),
		"expiresAt":  row.ExpiresAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
	})
}

func (s *Server) passwordAccept(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password             string `json:"password"`
		PasswordConfirmation string `json:"password_confirmation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Password) < 8 || body.Password != body.PasswordConfirmation {
		writeValidation(w, map[string][]string{"password": {"The password must be at least 8 characters."}})
		return
	}
	row, err := s.tokens.Consume(r.Context(), chi.URLParam(r, "token"))
	if err != nil || row == nil || row.User == nil {
		writeMessage(w, http.StatusNotFound, "This link is invalid or has expired.")
		return
	}
	if err := s.users.UpdatePassword(r.Context(), row.User.ID, body.Password); err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	u, _ := s.users.ByID(r.Context(), row.User.ID)
	_ = s.sessions.RevokeAll(r.Context(), u.ID)
	if u.HasTwoFactor() {
		tok, err := s.challenges.Issue(r.Context(), u)
		if err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requires_2fa": true, "two_factor_token": tok})
		return
	}
	s.issueSession(w, r, u, http.StatusOK, true)
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u *auth.User, status int, remember bool) {
	issued, err := s.sessions.Issue(r.Context(), u, r.RemoteAddr, r.UserAgent(), remember)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	auth.SetSessionCookies(w, r, s.cfg, issued.Token, issued.CSRF, remember)
	s.writeSessionPayload(w, r, status, u, issued.Session)
}

func (s *Server) writeSessionPayload(w http.ResponseWriter, r *http.Request, status int, u *auth.User, sess *auth.Session) {
	var refresh *string // null when the session does not rotate
	if at := sess.RefreshDueAt(); at != nil {
		iso := at.UTC().Format(time.RFC3339)
		refresh = &iso
	}
	payload := map[string]any{
		"user":       u.SessionJSON(),
		"expires_at": sess.AbsoluteExpiresAt.UTC().Format(time.RFC3339),
		"refresh_at": refresh,
		// The SPA shows the "confirm your email" page instead of the app.
		"email_verification_required": s.emailVerificationRequired(r.Context()),
	}
	writeJSON(w, status, payload)
}

func (s *Server) publicAuthRoutes(r chi.Router) {
	r.Get("/auth/status", s.authStatus)
	r.Get("/auth/me", s.authMe)
	r.Post("/auth/bootstrap/verify", s.bootstrapVerify)
	r.Post("/auth/register", s.authRegister)
	r.Post("/auth/login", s.authLogin)
	r.Get("/auth/password/{token}", s.passwordPreview)
	r.Post("/auth/password/{token}", s.passwordAccept)
}

func (s *Server) credentialAuthRoutes(r chi.Router) {
	r.Post("/auth/logout", s.authLogout)
	r.Post("/auth/logout-others", s.authLogoutOthers)
	r.Put("/auth/password", s.authChangePassword)
}

// accountServices are the server-level services behind sign-in and accounts.
type accountServices struct {
	users         auth.Users
	sessions      auth.Sessions
	apiTokens     auth.APITokens
	tokens        auth.PasswordTokens
	challenges    auth.Challenges
	verifications auth.EmailVerifications
	twoFactor     auth.TwoFactor
	webauthn      auth.WebAuthn
}

func newAccountServices(cfg config.Config, srv store.DB) accountServices {
	return accountServices{
		users:         auth.Users{DB: srv},
		sessions:      auth.Sessions{DB: srv, SessionTTL: cfg.SessionTTL, RememberTTL: cfg.RememberTTL},
		apiTokens:     auth.APITokens{DB: srv},
		tokens:        auth.PasswordTokens{DB: srv},
		challenges:    auth.Challenges{DB: srv, Cfg: cfg},
		verifications: auth.EmailVerifications{DB: srv},
		twoFactor:     auth.TwoFactor{DB: srv, Users: auth.Users{DB: srv}},
		webauthn:      auth.WebAuthn{DB: srv, Cfg: cfg},
	}
}

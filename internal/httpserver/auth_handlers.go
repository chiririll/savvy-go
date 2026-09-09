package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/chiririll/savvy-plus/internal/auth"
	"github.com/go-chi/chi/v5"
)

func (s *Server) passwordLoginDisabled(r *http.Request) bool {
	if s.settings.Bool(r.Context(), "password_login_enabled", true) {
		return false
	}
	return s.enabledSSOExists(r)
}

func (s *Server) enabledSSOExists(r *http.Request) bool {
	var n int
	_ = s.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM identity_providers WHERE enabled = 1`).Scan(&n)
	return n > 0
}

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	n, err := s.users.Count(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"needs_registration":     n == 0,
		"password_login_enabled": !s.passwordLoginDisabled(r),
	})
}

func (s *Server) authRegister(w http.ResponseWriter, r *http.Request) {
	n, err := s.users.Count(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 {
		writeMessage(w, http.StatusForbidden, "Registration is closed.")
		return
	}
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
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
	pass := body.Password
	u, err := s.users.Create(r.Context(), strings.TrimSpace(body.Name), body.Email, &pass, auth.RoleAdmin)
	if err != nil {
		writeValidation(w, map[string][]string{"email": {"The email has already been taken."}})
		return
	}
	s.issueSession(w, r, u, 201, true)
}

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
	s.writeSessionPayload(w, http.StatusOK, sess.User, sess)
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
	s.writeSessionPayload(w, status, u, issued.Session)
}

func (s *Server) writeSessionPayload(w http.ResponseWriter, status int, u *auth.User, sess *auth.Session) {
	expires, refresh := s.sessions.ClientTimes(sess)
	payload := map[string]any{
		"user":       u.SessionJSON(),
		"expires_at": expires,
		"refresh_at": refresh,
	}
	writeJSON(w, status, payload)
}

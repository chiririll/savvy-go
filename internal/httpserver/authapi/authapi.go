// Package authapi serves sign-in, registration, password links and sign-out.
package authapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/signin"
	"savvy-go/internal/settings"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/config"

	"github.com/go-chi/chi/v5"
)

// Handler holds what the account routes need.
type Handler struct {
	Cfg        config.Config
	Users      auth.Users
	Sessions   auth.Sessions
	Tokens     auth.PasswordTokens
	Challenges auth.Challenges
	Settings   settings.Store
	Bootstrap  *bootstrap.Guard
	Login      signin.Service
}

func (h Handler) authStatus(w http.ResponseWriter, r *http.Request) {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Tells the setup page where to find the first-run password; the password
	// itself never leaves the server.
	var hint any
	if n == 0 && h.Bootstrap.Active() {
		h := h.Bootstrap.Hint()
		hint = map[string]any{"environment": h.Environment, "file": h.File}
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"needs_registration":     n == 0,
		"password_login_enabled": !h.Login.PasswordLoginDisabled(r.Context()),
		"bootstrap_hint":         hint,
	})
}

// bootstrapVerify tells the setup page whether the bootstrap password is right,
// without using it up; register still checks it again.
func (h Handler) bootstrapVerify(w http.ResponseWriter, r *http.Request) {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 || !h.Bootstrap.Active() {
		api.WriteMessage(w, http.StatusForbidden, "Registration is closed.")
		return
	}
	var body struct {
		BootstrapPassword string `json:"bootstrap_password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if h.Bootstrap.Verify(body.BootstrapPassword) != nil {
		api.WriteValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// authRegister creates the first admin. Registration is never open: it needs
// the bootstrap password, and only while the server has no users.
func (h Handler) authRegister(w http.ResponseWriter, r *http.Request) {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 || !h.Bootstrap.Active() {
		api.WriteMessage(w, http.StatusForbidden, "Registration is closed.")
		return
	}
	var body struct {
		Name              string `json:"name"`
		Email             string `json:"email"`
		Password          string `json:"password"`
		BootstrapPassword string `json:"bootstrap_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	errs := map[string][]string{}
	if strings.TrimSpace(body.Name) == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if !api.LooksLikeEmail(body.Email) {
		errs["email"] = []string{"The email field must be a valid email address."}
	}
	if len(body.Password) < 6 {
		errs["password"] = []string{"The password must be at least 6 characters."}
	}
	if len(errs) > 0 {
		api.WriteValidation(w, errs)
		return
	}
	var u *auth.User
	// Verify, create and invalidate are one step, so two parallel requests
	// with the right password still create one admin.
	err = h.Bootstrap.Register(body.BootstrapPassword, func() error {
		if n, err := h.Users.Count(r.Context()); err != nil {
			return err
		} else if n > 0 {
			return bootstrap.ErrClosed
		}
		pass := body.Password
		var err error
		if u, err = h.Users.Create(r.Context(), strings.TrimSpace(body.Name), body.Email, &pass, auth.RoleAdmin); err != nil {
			return errEmailTaken
		}
		return nil
	})
	switch {
	case errors.Is(err, bootstrap.ErrWrongPassword):
		api.WriteValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
	case errors.Is(err, bootstrap.ErrClosed):
		h.Bootstrap.Invalidate()
		api.WriteMessage(w, http.StatusForbidden, "Registration is closed.")
	case errors.Is(err, errEmailTaken):
		api.WriteValidation(w, map[string][]string{"email": {"The email has already been taken."}})
	case err != nil:
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
	default:
		// The admin is taken through server setup, also after a reload or on another device.
		if err := h.Settings.Set(r.Context(), "onboarding_server_done", false); err != nil {
			slog.Warn("mark server setup pending", "err", err)
		}
		h.Login.Issue(w, r, u, 201, true)
	}
}

var errEmailTaken = errors.New("email taken")

func (h Handler) authLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	if h.Login.PasswordLoginDisabled(r.Context()) {
		api.WriteValidation(w, map[string][]string{"email": {"Password sign-in is disabled. Use single sign-on instead."}})
		return
	}
	u, err := h.Users.ByEmail(r.Context(), body.Email)
	if err != nil || u == nil || u.IsSSOOnly || u.IsInactive() || u.Password == nil || !auth.CheckPassword(*u.Password, body.Password) {
		api.WriteValidation(w, map[string][]string{"email": {"Invalid credentials."}})
		return
	}
	if u.HasTwoFactor() {
		tok, err := h.Challenges.Issue(r.Context(), u)
		if err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		api.WriteJSON(w, http.StatusOK, map[string]any{
			"requires_2fa":     true,
			"two_factor_token": tok,
		})
		return
	}
	h.Login.Issue(w, r, u, http.StatusOK, body.RememberMe)
}

func (h Handler) authMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	token := auth.ReadToken(r, h.Cfg)
	sess, err := h.Sessions.Resolve(r.Context(), token)
	if err != nil || sess == nil || sess.User == nil {
		api.WriteJSON(w, http.StatusOK, map[string]any{
			"user":       nil,
			"expires_at": nil,
			"refresh_at": nil,
		})
		return
	}
	refreshed, err := h.Sessions.MaybeRefresh(r.Context(), sess)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if refreshed != nil {
		auth.SetSessionCookies(w, r, h.Cfg, refreshed.Token, refreshed.CSRF, true)
	}
	h.Login.WritePayload(w, r, http.StatusOK, sess.User, sess)
}

func (h Handler) authChangePassword(w http.ResponseWriter, r *http.Request) {
	u := api.User(r)
	if u.IsSSOOnly || u.Password == nil {
		api.WriteMessage(w, http.StatusUnprocessableEntity, "This account uses SSO and does not have a password.")
		return
	}
	var body struct {
		Current              string `json:"current_password"`
		Password             string `json:"password"`
		PasswordConfirmation string `json:"password_confirmation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"password": {"The given data was invalid."}})
		return
	}
	if !auth.CheckPassword(*u.Password, body.Current) {
		api.WriteValidation(w, map[string][]string{"current_password": {"Current password is incorrect."}})
		return
	}
	if len(body.Password) < 8 {
		api.WriteValidation(w, map[string][]string{"password": {"The password must be at least 8 characters."}})
		return
	}
	if body.Password != body.PasswordConfirmation {
		api.WriteValidation(w, map[string][]string{"password": {"The password confirmation does not match."}})
		return
	}
	if err := h.Users.UpdatePassword(r.Context(), u.ID, body.Password); err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sess := api.Session(r); sess != nil {
		_, _ = h.Sessions.RevokeOthers(r.Context(), u.ID, sess.ID)
	}
	api.WriteMessage(w, http.StatusOK, "Password updated.")
}

func (h Handler) authLogoutOthers(w http.ResponseWriter, r *http.Request) {
	u := api.User(r)
	sess := api.Session(r)
	revoked := 0
	if sess != nil {
		revoked, _ = h.Sessions.RevokeOthers(r.Context(), u.ID, sess.ID)
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"message": "Signed out of other devices.",
		"revoked": revoked,
	})
}

func (h Handler) authLogout(w http.ResponseWriter, r *http.Request) {
	if sess := api.Session(r); sess != nil {
		_ = h.Sessions.Revoke(r.Context(), sess)
	}
	auth.ClearSessionCookies(w, r, h.Cfg)
	api.WriteMessage(w, http.StatusOK, "Logged out.")
}

func (h Handler) passwordPreview(w http.ResponseWriter, r *http.Request) {
	row, err := h.Tokens.Preview(r.Context(), chi.URLParam(r, "token"))
	if err != nil || row == nil || row.User == nil {
		api.WriteMessage(w, http.StatusNotFound, "This link is invalid or has expired.")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"name":       row.User.Name,
		"email":      row.User.Email,
		"isInactive": row.User.IsInactive(),
		"expiresAt":  row.ExpiresAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
	})
}

func (h Handler) passwordAccept(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password             string `json:"password"`
		PasswordConfirmation string `json:"password_confirmation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Password) < 8 || body.Password != body.PasswordConfirmation {
		api.WriteValidation(w, map[string][]string{"password": {"The password must be at least 8 characters."}})
		return
	}
	row, err := h.Tokens.Consume(r.Context(), chi.URLParam(r, "token"))
	if err != nil || row == nil || row.User == nil {
		api.WriteMessage(w, http.StatusNotFound, "This link is invalid or has expired.")
		return
	}
	if err := h.Users.UpdatePassword(r.Context(), row.User.ID, body.Password); err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	u, _ := h.Users.ByID(r.Context(), row.User.ID)
	_ = h.Sessions.RevokeAll(r.Context(), u.ID)
	if u.HasTwoFactor() {
		tok, err := h.Challenges.Issue(r.Context(), u)
		if err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		api.WriteJSON(w, http.StatusOK, map[string]any{"requires_2fa": true, "two_factor_token": tok})
		return
	}
	h.Login.Issue(w, r, u, http.StatusOK, true)
}

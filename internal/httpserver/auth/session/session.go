// Package session serves sign-in, registration, password links and sign-out.
package session

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/password"
	"savvy-go/internal/firstrun"

	"github.com/go-chi/chi/v5"
)

func (h Handler) authStatus(w http.ResponseWriter, r *http.Request) {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Tells the setup page where to find the first-run password; the password
	// itself never leaves the server.
	var hint *dto.BootstrapHint
	if n == 0 && h.Bootstrap.Active() {
		b := h.Bootstrap.Hint()
		hint = &dto.BootstrapHint{Environment: b.Environment, File: b.File}
	}
	httpx.WriteJSON(w, http.StatusOK, dto.SetupStatus{
		NeedsRegistration:    n == 0,
		PasswordLoginEnabled: !h.Login.PasswordLoginDisabled(r.Context()),
		BootstrapHint:        hint,
	})
}

// bootstrapVerify tells the setup page whether the bootstrap password is right,
// without using it up; register still checks it again.
func (h Handler) bootstrapVerify(w http.ResponseWriter, r *http.Request) {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 || !h.Bootstrap.Active() {
		httpx.WriteMessage(w, http.StatusForbidden, "Registration is closed.")
		return
	}
	var body struct {
		BootstrapPassword string `json:"bootstrap_password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if h.Bootstrap.Verify(body.BootstrapPassword) != nil {
		httpx.WriteValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// authRegister creates the first admin. Registration is never open: it needs
// the bootstrap password, and only while the server has no users.
func (h Handler) authRegister(w http.ResponseWriter, r *http.Request) {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 || !h.Bootstrap.Active() {
		httpx.WriteMessage(w, http.StatusForbidden, "Registration is closed.")
		return
	}
	var body struct {
		Name              string `json:"name"`
		Email             string `json:"email"`
		Password          string `json:"password"`
		BootstrapPassword string `json:"bootstrap_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	errs := map[string][]string{}
	if strings.TrimSpace(body.Name) == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if !httpx.LooksLikeEmail(body.Email) {
		errs["email"] = []string{"The email field must be a valid email address."}
	}
	if password.TooShort(body.Password) {
		errs["password"] = []string{password.TooShortMessage}
	}
	if len(errs) > 0 {
		httpx.WriteValidation(w, errs)
		return
	}
	var u *auth.User
	// Verify, create and invalidate are one step, so two parallel requests
	// with the right password still create one admin.
	err = h.Bootstrap.Register(body.BootstrapPassword, func() error {
		if n, err := h.Users.Count(r.Context()); err != nil {
			return err
		} else if n > 0 {
			return firstrun.ErrClosed
		}
		pass := body.Password
		var err error
		if u, err = h.Users.Create(r.Context(), strings.TrimSpace(body.Name), body.Email, &pass, auth.RoleAdmin); err != nil {
			return errEmailTaken
		}
		return nil
	})
	switch {
	case errors.Is(err, firstrun.ErrWrongPassword):
		httpx.WriteValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
	case errors.Is(err, firstrun.ErrClosed):
		h.Bootstrap.Invalidate()
		httpx.WriteMessage(w, http.StatusForbidden, "Registration is closed.")
	case errors.Is(err, errEmailTaken):
		httpx.WriteValidation(w, map[string][]string{"email": {"The email has already been taken."}})
	case err != nil:
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
	default:
		// The admin is taken through server setup, also after a reload or on another device.
		if err := h.Settings.Set(r.Context(), "onboarding_server_done", false); err != nil {
			slog.Warn("mark server setup pending", "err", err)
		}
		h.Login.Issue(w, r, u, 201, true)
	}
}

var errEmailTaken = errors.New("email taken")

// credentialsOK runs exactly one password check whatever u is, so an unknown,
// SSO-only or inactive account takes as long as a wrong password. check gets a
// nil hash when there is none to compare with and must then report false.
func credentialsOK(check func(hash *string, plain string) bool, u *auth.User, plain string) bool {
	var hash *string
	if u != nil && !u.IsSSOOnly && !u.IsInactive() {
		hash = u.Password
	}
	ok := check(hash, plain)
	return ok && hash != nil
}

func (h Handler) authLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	if h.Login.PasswordLoginDisabled(r.Context()) {
		httpx.WriteValidation(w, map[string][]string{"email": {"Password sign-in is disabled. Use single sign-on instead."}})
		return
	}
	if !h.Guard.AllowAccount(w, body.Email) {
		return
	}
	u, err := h.Users.ByEmail(r.Context(), body.Email)
	if err != nil {
		u = nil
	}
	if !credentialsOK(password.CheckOrBurn, u, body.Password) {
		httpx.WriteValidation(w, map[string][]string{"email": {"Invalid credentials."}})
		return
	}
	if u.HasTwoFactor() {
		tok, err := h.Challenges.Issue(r.Context(), u)
		if err != nil {
			httpx.WriteChallengeError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, dto.NewTwoFactorRequired(tok))
		return
	}
	h.Login.Issue(w, r, u, http.StatusOK, body.RememberMe)
}

func (h Handler) authMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	token := h.Security.ReadToken(r, h.Guard.Secure(r))
	sess, err := h.Sessions.Resolve(r.Context(), token)
	if err != nil || sess == nil || sess.User == nil {
		httpx.WriteJSON(w, http.StatusOK, dto.SessionPayload{})
		return
	}
	refreshed, err := h.Sessions.MaybeRefresh(r.Context(), sess)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if refreshed != nil {
		h.Security.SetSessionCookies(w, h.Guard.Secure(r), refreshed.Token, refreshed.CSRF, true)
	}
	h.Login.WritePayload(w, r, http.StatusOK, sess.User, sess)
}

func (h Handler) authChangePassword(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	if u.IsSSOOnly || u.Password == nil {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "This account uses SSO and does not have a password.")
		return
	}
	var body struct {
		Current              string `json:"current_password"`
		Password             string `json:"password"`
		PasswordConfirmation string `json:"password_confirmation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"password": {"The given data was invalid."}})
		return
	}
	if !password.Check(*u.Password, body.Current) {
		httpx.WriteValidation(w, map[string][]string{"current_password": {"Current password is incorrect."}})
		return
	}
	if password.TooShort(body.Password) {
		httpx.WriteValidation(w, map[string][]string{"password": {password.TooShortMessage}})
		return
	}
	if body.Password != body.PasswordConfirmation {
		httpx.WriteValidation(w, map[string][]string{"password": {"The password confirmation does not match."}})
		return
	}
	if err := h.Users.UpdatePassword(r.Context(), u.ID, body.Password); err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sess := httpx.Session(r); sess != nil {
		_, _ = h.Sessions.RevokeOthers(r.Context(), u.ID, sess.ID)
	}
	httpx.WriteMessage(w, http.StatusOK, "Password updated.")
}

func (h Handler) authLogoutOthers(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	sess := httpx.Session(r)
	revoked := 0
	if sess != nil {
		revoked, _ = h.Sessions.RevokeOthers(r.Context(), u.ID, sess.ID)
	}
	httpx.WriteJSON(w, http.StatusOK, dto.SignedOutOthers{Message: "Signed out of other devices.", Revoked: revoked})
}

func (h Handler) authLogout(w http.ResponseWriter, r *http.Request) {
	if sess := httpx.Session(r); sess != nil {
		_ = h.Sessions.Revoke(r.Context(), sess)
	}
	h.Security.ClearSessionCookies(w, h.Guard.Secure(r))
	httpx.WriteMessage(w, http.StatusOK, "Logged out.")
}

func (h Handler) passwordPreview(w http.ResponseWriter, r *http.Request) {
	row, err := h.Tokens.Preview(r.Context(), chi.URLParam(r, "token"))
	if err != nil || row == nil || row.User == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "This link is invalid or has expired.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dto.PasswordLink{
		Name: row.User.Name, Email: row.User.Email, IsInactive: row.User.IsInactive(), ExpiresAt: row.ExpiresAt.UTC(),
	})
}

func (h Handler) passwordAccept(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password             string `json:"password"`
		PasswordConfirmation string `json:"password_confirmation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || password.TooShort(body.Password) || body.Password != body.PasswordConfirmation {
		httpx.WriteValidation(w, map[string][]string{"password": {password.TooShortMessage}})
		return
	}
	row, err := h.Tokens.Consume(r.Context(), chi.URLParam(r, "token"))
	if err != nil || row == nil || row.User == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "This link is invalid or has expired.")
		return
	}
	if err := h.Users.UpdatePassword(r.Context(), row.User.ID, body.Password); err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	u, _ := h.Users.ByID(r.Context(), row.User.ID)
	// A reset is for taking an account back, so nothing from before it may stay: neither
	// sessions nor API tokens. Failing here must not hand out a fresh session.
	if err := h.Sessions.RevokeAll(r.Context(), u.ID); err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.APITokens.RevokeAll(r.Context(), u.ID); err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if u.HasTwoFactor() {
		tok, err := h.Challenges.Issue(r.Context(), u)
		if err != nil {
			httpx.WriteChallengeError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, dto.NewTwoFactorRequired(tok))
		return
	}
	h.Login.Issue(w, r, u, http.StatusOK, true)
}

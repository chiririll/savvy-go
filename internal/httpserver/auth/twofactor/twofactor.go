// Package twofactor serves two-factor authentication.
package twofactor

import (
	"encoding/json"
	"net/http"
	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/login"
)

// Handler holds what the two-factor routes need.
type Handler struct {
	TwoFactor  auth.TwoFactor
	Users      auth.Users
	Challenges auth.Challenges
	Login      login.Service
}

func (h Handler) twoFactorEnable(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	secret, uri, err := h.TwoFactor.Enable(r.Context(), u)
	if err != nil {
		if err.Error() == "already enabled" {
			httpx.WriteMessage(w, http.StatusBadRequest, "Two-factor authentication is already enabled.")
			return
		}
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"secret": secret, "qr_code_url": uri})
}

func (h Handler) twoFactorConfirm(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		httpx.WriteValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	u, _ := h.Users.ByID(r.Context(), httpx.User(r).ID)
	codes, err := h.TwoFactor.Confirm(r.Context(), u, body.Code)
	if err != nil {
		switch err.Error() {
		case "not pending":
			httpx.WriteMessage(w, http.StatusBadRequest, "Two-factor authentication is not pending confirmation.")
		case "invalid code":
			httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		default:
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"message":        "Two-factor authentication has been enabled.",
		"recovery_codes": codes,
	})
}

func (h Handler) twoFactorDisable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		httpx.WriteValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	u, _ := h.Users.ByID(r.Context(), httpx.User(r).ID)
	if err := h.TwoFactor.Disable(r.Context(), u, body.Code); err != nil {
		switch err.Error() {
		case "not enabled":
			httpx.WriteMessage(w, http.StatusBadRequest, "Two-factor authentication is not enabled.")
		case "invalid code":
			httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		default:
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httpx.WriteMessage(w, http.StatusOK, "Two-factor authentication has been disabled.")
}

func (h Handler) twoFactorVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token      string `json:"two_factor_token"`
		Code       string `json:"code"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || body.Code == "" {
		httpx.WriteValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	u, err := h.Challenges.Resolve(r.Context(), body.Token)
	if err != nil || u == nil {
		httpx.WriteMessage(w, http.StatusUnauthorized, "Invalid or expired token.")
		return
	}
	if !h.TwoFactor.VerifyAny(r.Context(), u, body.Code) {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		return
	}
	if _, err := h.Challenges.Consume(r.Context(), body.Token); err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.Login.Issue(w, r, u, http.StatusOK, body.RememberMe)
}

func (h Handler) twoFactorRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	if !u.HasTwoFactor() {
		httpx.WriteMessage(w, http.StatusBadRequest, "Two-factor authentication is not enabled.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"remaining_count": h.TwoFactor.Remaining(r.Context(), u.ID)})
}

func (h Handler) twoFactorRegenerate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		httpx.WriteValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	u, _ := h.Users.ByID(r.Context(), httpx.User(r).ID)
	codes, err := h.TwoFactor.Regenerate(r.Context(), u, body.Code)
	if err != nil {
		switch err.Error() {
		case "not enabled":
			httpx.WriteMessage(w, http.StatusBadRequest, "Two-factor authentication is not enabled.")
		case "invalid code":
			httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		default:
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"message":        "Recovery codes have been regenerated.",
		"recovery_codes": codes,
	})
}

func (h Handler) twoFactorStatus(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	var remaining any
	if u.HasTwoFactor() {
		remaining = h.TwoFactor.Remaining(r.Context(), u.ID)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"enabled":                  u.HasTwoFactor(),
		"pending_confirmation":     u.TwoFactorEnabled && !u.TwoFactorConfirmed,
		"recovery_codes_remaining": remaining,
	})
}

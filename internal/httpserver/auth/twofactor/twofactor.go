// Package twofactor serves two-factor authentication.
package twofactor

import (
	"encoding/json"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
)

func (h Handler) twoFactorEnable(w http.ResponseWriter, r *http.Request) {
	if !h.Confirm.RequireBody(w, r) {
		return
	}
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
	httpx.WriteJSON(w, http.StatusOK, dto.TwoFactorSetup{Secret: secret, QRCodeURL: uri})
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
	httpx.WriteJSON(w, http.StatusOK, dto.RecoveryCodes{Message: "Two-factor authentication has been enabled.", Codes: codes})
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
		remaining, err := h.Challenges.Fail(r.Context(), body.Token)
		if err != nil {
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		if remaining == 0 {
			// Used up: the SPA goes back to the login form.
			httpx.WriteMessage(w, http.StatusUnauthorized, "Invalid or expired token.")
			return
		}
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		return
	}
	// Claiming the challenge is atomic: of two requests racing on one challenge, one wins.
	claimed, err := h.Challenges.Consume(r.Context(), body.Token)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if claimed == nil {
		httpx.WriteMessage(w, http.StatusUnauthorized, "Invalid or expired token.")
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
	httpx.WriteJSON(w, http.StatusOK, dto.RecoveryCodesRemaining{Remaining: h.TwoFactor.Remaining(r.Context(), u.ID)})
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
	httpx.WriteJSON(w, http.StatusOK, dto.RecoveryCodes{Message: "Recovery codes have been regenerated.", Codes: codes})
}

func (h Handler) twoFactorStatus(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	var remaining *int
	if u.HasTwoFactor() {
		n := h.TwoFactor.Remaining(r.Context(), u.ID)
		remaining = &n
	}
	httpx.WriteJSON(w, http.StatusOK, dto.TwoFactorStatus{
		Enabled:                u.HasTwoFactor(),
		PendingConfirmation:    u.TwoFactorEnabled && !u.TwoFactorConfirmed,
		RecoveryCodesRemaining: remaining,
	})
}

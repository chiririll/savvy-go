package httpserver

import (
	"encoding/json"
	"net/http"
)

func (s *Server) twoFactorEnable(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	secret, uri, err := s.twoFactor.Enable(r.Context(), u)
	if err != nil {
		if err.Error() == "already enabled" {
			writeMessage(w, http.StatusBadRequest, "Two-factor authentication is already enabled.")
			return
		}
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"secret": secret, "qr_code_url": uri})
}

func (s *Server) twoFactorConfirm(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	u, _ := s.users.ByID(r.Context(), userFrom(r).ID)
	codes, err := s.twoFactor.Confirm(r.Context(), u, body.Code)
	if err != nil {
		switch err.Error() {
		case "not pending":
			writeMessage(w, http.StatusBadRequest, "Two-factor authentication is not pending confirmation.")
		case "invalid code":
			writeMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		default:
			writeMessage(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message":        "Two-factor authentication has been enabled.",
		"recovery_codes": codes,
	})
}

func (s *Server) twoFactorDisable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	u, _ := s.users.ByID(r.Context(), userFrom(r).ID)
	if err := s.twoFactor.Disable(r.Context(), u, body.Code); err != nil {
		switch err.Error() {
		case "not enabled":
			writeMessage(w, http.StatusBadRequest, "Two-factor authentication is not enabled.")
		case "invalid code":
			writeMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		default:
			writeMessage(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Two-factor authentication has been disabled."})
}

func (s *Server) twoFactorVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token      string `json:"two_factor_token"`
		Code       string `json:"code"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || body.Code == "" {
		writeValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	u, err := s.challenges.Resolve(r.Context(), body.Token)
	if err != nil || u == nil {
		writeMessage(w, http.StatusUnauthorized, "Invalid or expired token.")
		return
	}
	if !s.twoFactor.VerifyAny(r.Context(), u, body.Code) {
		writeMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		return
	}
	if _, err := s.challenges.Consume(r.Context(), body.Token); err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.issueSession(w, r, u, http.StatusOK, body.RememberMe)
}

func (s *Server) twoFactorRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if !u.HasTwoFactor() {
		writeMessage(w, http.StatusBadRequest, "Two-factor authentication is not enabled.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"remaining_count": s.twoFactor.Remaining(r.Context(), u.ID)})
}

func (s *Server) twoFactorRegenerate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	u, _ := s.users.ByID(r.Context(), userFrom(r).ID)
	codes, err := s.twoFactor.Regenerate(r.Context(), u, body.Code)
	if err != nil {
		switch err.Error() {
		case "not enabled":
			writeMessage(w, http.StatusBadRequest, "Two-factor authentication is not enabled.")
		case "invalid code":
			writeMessage(w, http.StatusUnprocessableEntity, "Invalid verification code.")
		default:
			writeMessage(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message":        "Recovery codes have been regenerated.",
		"recovery_codes": codes,
	})
}

func (s *Server) twoFactorStatus(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var remaining any
	if u.HasTwoFactor() {
		remaining = s.twoFactor.Remaining(r.Context(), u.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":                  u.HasTwoFactor(),
		"pending_confirmation":     u.TwoFactorEnabled && !u.TwoFactorConfirmed,
		"recovery_codes_remaining": remaining,
	})
}

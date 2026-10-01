package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

func (s *Server) settingsIndex(w http.ResponseWriter, r *http.Request) {
	all, err := s.settings.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (s *Server) settingsUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"settings": {"The given data was invalid."}})
		return
	}
	allowed := map[string]bool{
		"auto_update_currencies":     true,
		"sso_allow_signup":           true,
		"password_login_enabled":     true,
		"sso_require_verified_email": true,
	}
	if v, ok := body["password_login_enabled"]; ok && !asBool(v) && !s.enabledSSOExists(r) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Enable at least one SSO provider before turning off password sign-in.",
			"error":   "sso_required",
		})
		return
	}
	for k, v := range body {
		if !allowed[k] {
			continue
		}
		if err := s.settings.Set(r.Context(), k, v); err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if v, ok := body["auto_update_currencies"]; ok && asBool(v) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if _, _, err := s.currencies.UpdateRates(ctx); err != nil {
				slog.Warn("refresh currency rates", "err", err)
			}
		}()
	}
	all, err := s.settings.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "1" || t == "true"
	default:
		return false
	}
}

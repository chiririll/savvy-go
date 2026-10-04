package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"savvy-go/internal/settings"
	"time"
)

// settingsIndex lists the instance settings.
func (s *Server) settingsIndex(w http.ResponseWriter, r *http.Request) {
	all, err := s.settings.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, all)
}

// settingsUpdate changes instance settings (server admins only, by route).
func (s *Server) settingsUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"settings": {"The given data was invalid."}})
		return
	}
	if v, ok := body["password_login_enabled"]; ok && !asBool(v) && !s.enabledSSOExists(r) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Enable at least one SSO provider before turning off password sign-in.",
			"error":   "sso_required",
		})
		return
	}
	for k, v := range body {
		if !settings.IsServerKey(k) {
			continue
		}
		if err := s.settings.Set(r.Context(), k, v); err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	s.settingsIndex(w, r)
}

// spaceSettingsUpdate changes the settings of a space (its admins, by route).
func (s *Server) spaceSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"settings": {"The given data was invalid."}})
		return
	}
	scope := sp(r)
	for k, v := range body {
		if !settings.IsSpaceKey(k) {
			continue
		}
		if err := scope.settings.Set(r.Context(), k, v); err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if v, ok := body["auto_update_currencies"]; ok && asBool(v) {
		currencies := scope.currencies
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if _, _, err := currencies.UpdateRates(ctx); err != nil {
				slog.Warn("refresh currency rates", "err", err)
			}
		}()
	}
	s.spaceSettingsIndex(w, r)
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

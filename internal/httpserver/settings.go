package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"savvy-go/internal/domain"
	"savvy-go/internal/settings"
	"time"
)

// allSettings merges the instance settings with those of the request's space.
// Instance settings move to their own admin endpoint with the space routes.
func (s *Server) allSettings(r *http.Request) (map[string]any, error) {
	all, err := s.settings.All(r.Context())
	if err != nil {
		return nil, err
	}
	if sp(r) == nil {
		return all, nil
	}
	own, err := sp(r).settings.All(r.Context())
	if err != nil {
		return nil, err
	}
	for k, v := range own {
		all[k] = v
	}
	return all, nil
}

func (s *Server) settingsIndex(w http.ResponseWriter, r *http.Request) {
	all, err := s.allSettings(r)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, all)
}

// settingsUpdate changes instance settings (server admins only) and settings
// of the request's space (its admins only).
func (s *Server) settingsUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"settings": {"The given data was invalid."}})
		return
	}
	u := userFrom(r)
	for k := range body {
		switch {
		case settings.IsServerKey(k) && !u.IsAdmin():
			writeMessage(w, http.StatusForbidden, "Only server administrators can change this setting.")
			return
		case settings.IsSpaceKey(k) && sp(r) == nil:
			writeMessage(w, http.StatusNotFound, "Space not found.")
			return
		case settings.IsSpaceKey(k) && sp(r).role != domain.SpaceAdmin:
			writeMessage(w, http.StatusForbidden, "Only space administrators can change this setting.")
			return
		}
	}
	if v, ok := body["password_login_enabled"]; ok && !asBool(v) && !s.enabledSSOExists(r) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Enable at least one SSO provider before turning off password sign-in.",
			"error":   "sso_required",
		})
		return
	}
	for k, v := range body {
		var err error
		switch {
		case settings.IsServerKey(k):
			err = s.settings.Set(r.Context(), k, v)
		case settings.IsSpaceKey(k):
			err = sp(r).settings.Set(r.Context(), k, v)
		}
		if err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if v, ok := body["auto_update_currencies"]; ok && asBool(v) {
		currencies := sp(r).currencies
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if _, _, err := currencies.UpdateRates(ctx); err != nil {
				slog.Warn("refresh currency rates", "err", err)
			}
		}()
	}
	all, err := s.allSettings(r)
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

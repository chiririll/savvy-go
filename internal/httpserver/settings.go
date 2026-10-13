package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"savvy-go/internal/settings"
	"strings"
	"time"
)

// settingsIndex lists the instance settings.
func (s *Server) settingsIndex(w http.ResponseWriter, r *http.Request) {
	all, err := s.settings.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The public URL lives in the config file, not the database.
	all["app_url"] = s.cfg.URL()
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
	if v, ok := body["app_url"]; ok {
		clean, err := cleanAppURL(v)
		if err != nil {
			writeValidation(w, map[string][]string{"app_url": {"Enter a full URL such as https://savvy.example.com."}})
			return
		}
		if s.cfg.Live == nil {
			writeMessage(w, http.StatusInternalServerError, "The config file is not available.")
			return
		}
		if err := s.cfg.Live.SetAppURL(clean); err != nil {
			writeValidation(w, map[string][]string{"app_url": {"Could not save the config file: " + err.Error()}})
			return
		}
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

// cleanAppURL validates the app_url setting: empty (unset) or an
// absolute http(s) URL, stored without a trailing slash.
func cleanAppURL(v any) (string, error) {
	raw, ok := v.(string)
	if !ok {
		return "", errors.New("not a string")
	}
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("invalid url")
	}
	return raw, nil
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

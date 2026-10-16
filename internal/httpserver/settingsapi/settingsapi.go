// Package settingsapi serves the settings of the instance and of a space.
package settingsapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/emailverify"
	"savvy-go/internal/httpserver/signin"
	"savvy-go/internal/settings"
	"savvy-go/internal/sso"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handler holds what the settings routes need.
type Handler struct {
	Cfg      config.Config
	Settings settings.Store
	SSO      *sso.Service
	Verify   emailverify.Handler
}

// settingsIndex lists the instance settings.
func (h Handler) settingsIndex(w http.ResponseWriter, r *http.Request) {
	all, err := h.Settings.All(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The public URL lives in the config file, not the database.
	all["app_url"] = h.Cfg.URL()
	// False when the config file is read-only: the fields it holds are locked.
	all["config_writable"] = h.Cfg.Live.Writable()
	api.WriteJSON(w, http.StatusOK, all)
}

// settingsUpdate changes instance settings (server admins only, by route).
func (h Handler) settingsUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"settings": {"The given data was invalid."}})
		return
	}
	if v, ok := body["password_login_enabled"]; ok && !api.AsBool(v) && !h.SSO.AnyEnabled(r.Context()) {
		api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Enable at least one SSO provider before turning off password sign-in.",
			"error":   "sso_required",
		})
		return
	}
	if v, ok := body[signin.SettingRequireVerification]; ok && api.AsBool(v) && !h.Verify.CanRequire(w, r) {
		return
	}
	if v, ok := body["app_url"]; ok {
		clean, err := cleanAppURL(v)
		if err != nil {
			api.WriteValidation(w, map[string][]string{"app_url": {"Enter a full URL such as https://savvy.example.com."}})
			return
		}
		if !h.Cfg.Live.Writable() {
			api.WriteValidation(w, map[string][]string{"app_url": {"The config file is read-only. Set app_url in config.toml."}})
			return
		}
		if err := h.Cfg.Live.SetAppURL(clean); err != nil {
			api.WriteValidation(w, map[string][]string{"app_url": {"Could not save the config file: " + err.Error()}})
			return
		}
	}
	for k, v := range body {
		if !settings.IsServerKey(k) {
			continue
		}
		if err := h.Settings.Set(r.Context(), k, v); err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	h.settingsIndex(w, r)
}

// spaceSettingsUpdate changes the settings of a space (its admins, by route).
func (h Handler) spaceSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"settings": {"The given data was invalid."}})
		return
	}
	scope := api.Scope(r)
	for k, v := range body {
		if !settings.IsSpaceKey(k) {
			continue
		}
		if err := scope.Settings.Set(r.Context(), k, v); err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if v, ok := body["auto_update_currencies"]; ok && api.AsBool(v) {
		currencies := scope.Currencies
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if _, _, err := currencies.UpdateRates(ctx); err != nil {
				slog.Warn("refresh currency rates", "err", err)
			}
		}()
	}
	h.spaceSettingsIndex(w, r)
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

func (h Handler) spaceSettingsIndex(w http.ResponseWriter, r *http.Request) {
	all, err := api.Scope(r).Settings.All(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, all)
}

func (h Handler) AdminSessionRoutes(r chi.Router) {
	r.Patch("/settings", h.settingsUpdate)
}

func (h Handler) SpaceReadRoutes(r chi.Router) {
	r.Get("/settings", h.spaceSettingsIndex)
}

func (h Handler) SpaceAdminRoutes(r chi.Router) {
	r.Patch("/settings", h.spaceSettingsUpdate)
}

func (h Handler) AccountRoutes(r chi.Router) {
	// Instance settings everyone may read (sign-in options).
	r.Get("/settings", h.settingsIndex)
}

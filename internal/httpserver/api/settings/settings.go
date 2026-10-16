// Package settings serves the settings of the instance and of a space.
package settings

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"savvy-go/internal/app/signin"
	"savvy-go/internal/config/shared"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/mail"
	"savvy-go/internal/settings"
	"savvy-go/internal/sso"
	"strings"
	"time"
)

// Handler holds what the settings routes need.
type Handler struct {
	Cfg      shared.Config
	Writable func() bool // whether the config file can be changed
	Settings settings.Store
	SSO      *sso.Service
	Verify   mail.Verifier
}

// settingsIndex lists the instance settings.
func (h Handler) settingsIndex(w http.ResponseWriter, r *http.Request) {
	all, err := h.Settings.All(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The public URL lives in the config file, not the database.
	all["app_url"] = h.Cfg.AppURL()
	// False when the config file cannot be changed from the application: the fields it holds
	// are locked.
	all["config_writable"] = h.Writable()
	httpx.WriteJSON(w, http.StatusOK, all)
}

// settingsUpdate changes instance settings (server admins only, by route).
func (h Handler) settingsUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"settings": {"The given data was invalid."}})
		return
	}
	if v, ok := body["password_login_enabled"]; ok && !httpx.AsBool(v) && !h.SSO.AnyEnabled(r.Context()) {
		httpx.WriteCoded(w, http.StatusUnprocessableEntity, "Enable at least one SSO provider before turning off password sign-in.", "sso_required")
		return
	}
	if v, ok := body[signin.SettingRequireVerification]; ok && httpx.AsBool(v) && !h.canRequireVerification(w, r) {
		return
	}
	if v, ok := body["app_url"]; ok {
		clean, err := cleanAppURL(v)
		if err != nil {
			httpx.WriteValidation(w, map[string][]string{"app_url": {"Enter a full URL such as https://savvy.example.com."}})
			return
		}
		// The application does not write the config file for now: edit it and restart.
		if clean != h.Cfg.AppURL() {
			httpx.WriteValidation(w, map[string][]string{"app_url": {"Set app_url in config.toml and restart."}})
			return
		}
	}
	for k, v := range body {
		if !settings.IsServerKey(k) {
			continue
		}
		if err := h.Settings.Set(r.Context(), k, v); err != nil {
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	h.settingsIndex(w, r)
}

// canRequireVerification answers 422 and returns false when the requirement cannot be
// switched on without locking the admin out.
func (h Handler) canRequireVerification(w http.ResponseWriter, r *http.Request) bool {
	err := h.Verify.CanRequire(httpx.User(r))
	switch {
	case err == nil:
		return true
	case errors.Is(err, mail.ErrOwnAddressUnverified):
		httpx.WriteCoded(w, http.StatusUnprocessableEntity, "Confirm your own email address first, or you would lock yourself out.", "email_unverified")
	default:
		httpx.WriteCoded(w, http.StatusUnprocessableEntity, err.Error(), "mail_required")
	}
	return false
}

// spaceSettingsUpdate changes the settings of a space (its admins, by route).
func (h Handler) spaceSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"settings": {"The given data was invalid."}})
		return
	}
	scope := httpx.Scope(r)
	for k, v := range body {
		if !settings.IsSpaceKey(k) {
			continue
		}
		if err := scope.Settings.Set(r.Context(), k, v); err != nil {
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if v, ok := body["auto_update_currencies"]; ok && httpx.AsBool(v) {
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
	all, err := httpx.Scope(r).Settings.All(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, all)
}

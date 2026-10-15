package settings

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	repo "savvy-go/internal/db/repo/settings"
	"savvy-go/internal/store"
)

// serverDefaults are the instance settings, kept in the server database.
var serverDefaults = map[string]any{
	"sso_allow_signup":           true,
	"password_login_enabled":     true,
	"sso_require_verified_email": false,
	// Size limit of a space in MB; 0 means unlimited.
	"space_quota_mb": 0,
	// Backups a space keeps; older ones are removed. 0 keeps all.
	"space_backups_max": 10,
	// How many spaces a user may administer; null is unlimited, 0 none.
	"max_spaces_per_user": nil,
	// Whether an invitation link may create an account.
	"space_invites_can_register": true,
	// Server setup after onboarding: false while it is pending (set when the
	// first admin is created), true once finished or skipped. Unset (null) on
	// a server that never went through onboarding: nothing to show.
	"onboarding_server_done": nil,
}

// spaceDefaults are the settings of one space, kept in its own database.
var spaceDefaults = map[string]any{
	"auto_update_currencies": true,
}

// IsSpaceKey reports whether key is a per-space setting.
func IsSpaceKey(key string) bool {
	_, ok := spaceDefaults[key]
	return ok
}

// IsServerKey reports whether key is an instance setting.
func IsServerKey(key string) bool {
	_, ok := serverDefaults[key]
	return ok
}

// Store reads and writes settings: the instance settings of the server
// database, or with Space set the settings of the space database DB.
type Store struct {
	DB    store.DB
	Space bool
}

func (s Store) defaults() map[string]any {
	if s.Space {
		return spaceDefaults
	}
	return serverDefaults
}

func (s Store) repo() repo.Repo { return repo.Repo{DB: s.DB, Space: s.Space} }

func (s Store) Get(ctx context.Context, key string, fallback any) any {
	raw, ok, err := s.repo().Get(ctx, key)
	if err != nil || !ok {
		if fallback != nil {
			return fallback
		}
		if v, ok := s.defaults()[key]; ok {
			return v
		}
		return nil
	}
	return decode(raw)
}

func (s Store) Bool(ctx context.Context, key string, fallback bool) bool {
	v := s.Get(ctx, key, fallback)
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		b, err := strconv.ParseBool(t)
		if err == nil {
			return b
		}
	}
	return fallback
}

func (s Store) Set(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.repo().Set(ctx, key, string(raw))
}

func (s Store) All(ctx context.Context) (map[string]any, error) {
	defaults := s.defaults()
	out := make(map[string]any, len(defaults))
	for k, v := range defaults {
		out[k] = v
	}
	stored, err := s.repo().All(ctx)
	if err != nil {
		return nil, err
	}
	for key, raw := range stored {
		// Only known keys: internal markers (legacy_*, space_uuid) stay private.
		if _, known := defaults[key]; !known || strings.HasPrefix(key, "legacy_") {
			continue
		}
		out[key] = decode(raw)
	}
	return out, nil
}

func decode(raw string) any {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err == nil {
		return v
	}
	return raw
}

// Int reads a numeric setting.
func (s Store) Int(ctx context.Context, key string, fallback int64) int64 {
	switch t := s.Get(ctx, key, nil).(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case string:
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

// SpaceQuota is the size limit of a space in bytes, 0 when unlimited.
func SpaceQuota(ctx context.Context, server store.DB) int64 {
	var mb float64
	switch t := (Store{DB: server}).Get(ctx, "space_quota_mb", nil).(type) {
	case float64:
		mb = t
	case string:
		mb, _ = strconv.ParseFloat(t, 64)
	}
	if mb <= 0 {
		return 0
	}
	return int64(mb * (1 << 20))
}

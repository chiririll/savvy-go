package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// serverDefaults are the instance settings, kept in the server database.
var serverDefaults = map[string]any{
	"sso_allow_signup":           true,
	"password_login_enabled":     true,
	"sso_require_verified_email": false,
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

func (s Store) get(ctx context.Context, key string) (sql.NullString, error) {
	if s.Space {
		return db.Q(s.DB).GetSpaceSetting(ctx, key)
	}
	return db.Q(s.DB).GetSetting(ctx, key)
}

func (s Store) Get(ctx context.Context, key string, fallback any) any {
	var raw sql.NullString
	raw, err := s.get(ctx, key)
	if err != nil || !raw.Valid {
		if fallback != nil {
			return fallback
		}
		if v, ok := s.defaults()[key]; ok {
			return v
		}
		return nil
	}
	return decode(raw.String)
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
	if s.Space {
		return db.Q(s.DB).UpsertSpaceSetting(ctx, sqlc.UpsertSpaceSettingParams{Key: key, Value: db.NS(string(raw))})
	}
	return db.Q(s.DB).UpsertSetting(ctx, sqlc.UpsertSettingParams{Key: key, Value: db.NS(string(raw))})
}

func (s Store) All(ctx context.Context) (map[string]any, error) {
	defaults := s.defaults()
	out := make(map[string]any, len(defaults))
	for k, v := range defaults {
		out[k] = v
	}
	type kv struct {
		key   string
		value sql.NullString
	}
	var rows []kv
	if s.Space {
		list, err := db.Q(s.DB).ListSpaceSettings(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range list {
			rows = append(rows, kv{r.Key, r.Value})
		}
	} else {
		list, err := db.Q(s.DB).ListSettings(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range list {
			rows = append(rows, kv{r.Key, r.Value})
		}
	}
	for _, r := range rows {
		// Only known keys: internal markers (legacy_*, space_uuid) stay private.
		if _, known := defaults[r.key]; !known || strings.HasPrefix(r.key, "legacy_") {
			continue
		}
		if r.value.Valid {
			out[r.key] = decode(r.value.String)
		}
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

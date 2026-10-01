package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

var defaults = map[string]any{
	"auto_update_currencies":     true,
	"sso_allow_signup":           true,
	"password_login_enabled":     true,
	"sso_require_verified_email": false,
}

type Store struct {
	DB *sql.DB
}

func (s Store) Get(ctx context.Context, key string, fallback any) any {
	var raw sql.NullString
	raw, err := db.Q(s.DB).GetSetting(ctx, key)
	if err != nil || !raw.Valid {
		if fallback != nil {
			return fallback
		}
		if v, ok := defaults[key]; ok {
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
	return db.Q(s.DB).UpsertSetting(ctx, sqlc.UpsertSettingParams{Key: key, Value: db.NS(string(raw))})
}

func (s Store) All(ctx context.Context) (map[string]any, error) {
	out := make(map[string]any, len(defaults))
	for k, v := range defaults {
		out[k] = v
	}
	rows, err := db.Q(s.DB).ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if strings.HasPrefix(r.Key, "legacy_") {
			continue
		}
		if r.Value.Valid {
			out[r.Key] = decode(r.Value.String)
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

package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
)

var defaults = map[string]any{
	"auto_update_currencies":    true,
	"sso_allow_signup":          true,
	"password_login_enabled":    true,
	"sso_require_verified_email": false,
}

type Store struct {
	DB *sql.DB
}

func (s Store) Get(ctx context.Context, key string, fallback any) any {
	var raw sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&raw)
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
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, string(raw))
	return err
}

func (s Store) All(ctx context.Context) (map[string]any, error) {
	out := make(map[string]any, len(defaults))
	for k, v := range defaults {
		out[k] = v
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var raw sql.NullString
		if err := rows.Scan(&k, &raw); err != nil {
			return nil, err
		}
		if strings.HasPrefix(k, "legacy_") {
			continue
		}
		if raw.Valid {
			out[k] = decode(raw.String)
		}
	}
	return out, rows.Err()
}

func decode(raw string) any {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err == nil {
		return v
	}
	return raw
}

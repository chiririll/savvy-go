// Package settingsrepo stores settings as JSON text by key: the instance
// settings in the server database, or with Space set the settings of the
// space database. Package settings decides which keys exist and what their
// values mean.
package settings

import (
	"context"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Repo reads and writes the settings table of DB.
type Repo struct {
	DB    store.DB
	Space bool
}

// Get returns the stored text of key; ok is false when it is unset.
func (r Repo) Get(ctx context.Context, key string) (value string, ok bool, err error) {
	var raw db.NString
	if r.Space {
		raw, err = db.Q(r.DB).GetSpaceSetting(ctx, key)
	} else {
		raw, err = db.Q(r.DB).GetSetting(ctx, key)
	}
	if err != nil {
		if err == db.ErrNotFound {
			return "", false, nil
		}
		return "", false, err
	}
	return raw.String, raw.Valid, nil
}

func (r Repo) Set(ctx context.Context, key, value string) error {
	if r.Space {
		return db.Q(r.DB).UpsertSpaceSetting(ctx, sqlc.UpsertSpaceSettingParams{Key: key, Value: db.NS(value)})
	}
	return db.Q(r.DB).UpsertSetting(ctx, sqlc.UpsertSettingParams{Key: key, Value: db.NS(value)})
}

// All returns every stored setting that has a value.
func (r Repo) All(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	if r.Space {
		rows, err := db.Q(r.DB).ListSpaceSettings(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row.Value.Valid {
				out[row.Key] = row.Value.String
			}
		}
		return out, nil
	}
	rows, err := db.Q(r.DB).ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Value.Valid {
			out[row.Key] = row.Value.String
		}
	}
	return out, nil
}

// Delete removes a space setting; a key that is not set is not an error.
func (r Repo) Delete(ctx context.Context, key string) error {
	return db.Q(r.DB).DeleteSpaceSetting(ctx, key)
}

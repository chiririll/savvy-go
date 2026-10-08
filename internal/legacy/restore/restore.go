// Package restore brings Laravel-era databases into a running server: as a
// whole-server replacement, over one space, or as a new space. The file is
// converted for this one restoration and never kept as a backup.
//
// The conversion itself is internal/legacy, run by the store when it prepares
// the file. This package only drives the store's interface and imports none
// of its SQLite internals, so HTTP handlers may use it where they may not
// import internal/legacy.
package restore

import (
	"context"
	"os"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// IsDatabaseFile tells a bare database file, in practice a Laravel database,
// from a backup archive by its name.
func IsDatabaseFile(name string) bool { return strings.HasSuffix(strings.ToLower(name), ".sqlite") }

// Server replaces the server and every space with the Laravel database at
// path. appKey is the Laravel APP_KEY, "" if the user has none: two-factor
// secrets that cannot be decrypted are then reset (see
// legacy.UpgradeLegacyTOTPSecrets).
func Server(ctx context.Context, st store.Store, path, appKey string) error {
	p, err := st.PrepareServer(ctx, path, store.WithLegacyKey(appKey))
	if err != nil {
		return err
	}
	return st.ReplaceServer(ctx, p)
}

// Space replaces a space's data with the finances of the Laravel database at
// path; its users and other server data are left out.
func Space(ctx context.Context, backups backup.Service, sp spaces.Space, path string, quota int64) error {
	if exceeds(path, quota) {
		return backup.ErrTooLarge
	}
	p, err := backups.Store.PrepareSpace(ctx, path)
	if err != nil {
		return err
	}
	p.Settings = map[string]string{"space_uuid": sp.UUID}
	return backups.Replace(ctx, sp, p, quota)
}

// ImportSpace creates a new space, owned by owner, from the finances of the
// Laravel database at path. An empty name becomes "Imported".
func ImportSpace(ctx context.Context, spaces spaces.Service, path, name string, owner *auth.User, quota int64) (*spaces.Space, error) {
	if exceeds(path, quota) {
		return nil, backup.ErrTooLarge
	}
	p, err := spaces.Store.PrepareSpace(ctx, path)
	if err != nil {
		return nil, err
	}
	if quota > 0 && p.Size > quota {
		spaces.Store.Discard(p.Artifact)
		return nil, backup.ErrTooLarge
	}
	if name == "" {
		name = "Imported"
	}
	return spaces.CreateFrom(ctx, name, owner, p)
}

func exceeds(path string, limit int64) bool {
	if limit <= 0 {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() > limit
}

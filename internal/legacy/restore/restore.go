// Package restore brings Laravel-era databases into a running server. A
// database for the whole server or for one space is converted and kept as an
// ordinary backup, which the user then restores like any other; the Laravel
// file itself is never kept. A new space is created straight away, as it
// replaces nothing.
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

const note = "Upgraded from the Laravel version"

// IsDatabaseFile tells a bare database file, in practice a Laravel database,
// from a backup archive by its name.
func IsDatabaseFile(name string) bool { return strings.HasSuffix(strings.ToLower(name), ".sqlite") }

// Server converts the Laravel database at path and keeps it as a server
// backup. appKey is the Laravel APP_KEY, "" if the user has none: two-factor
// secrets that cannot be decrypted are then reset (see
// legacy.UpgradeLegacyTOTPSecrets). Nothing live is touched.
func Server(ctx context.Context, backups backup.Service, path, appKey string) (*backup.Entry, error) {
	p, err := backups.Store.PrepareServer(ctx, path, store.WithLegacyKey(appKey))
	if err != nil {
		return nil, err
	}
	return backups.SaveServerFrom(p, note)
}

// Space converts the finances of the Laravel database at path and keeps them
// as a backup of sp, whose users and other server data are left out.
func Space(
	ctx context.Context,
	backups backup.Service,
	sp spaces.Space,
	path string,
	quota int64,
) (*backup.Entry, error) {
	if exceeds(path, quota) {
		return nil, backup.ErrTooLarge
	}
	p, err := backups.Store.PrepareSpace(ctx, path, store.WithSpaceUUID(sp.UUID))
	if err != nil {
		return nil, err
	}
	if quota > 0 && p.Size > quota {
		backups.Store.Discard(p.Artifact)
		return nil, backup.ErrTooLarge
	}
	return backups.SaveSpaceFrom(sp, p, note)
}

// ImportSpace creates a new space, owned by owner, from the finances of the
// Laravel database at path. An empty name becomes "Imported".
func ImportSpace(
	ctx context.Context,
	spaces spaces.Service,
	path, name string,
	owner *auth.User,
	quota int64,
) (*spaces.Space, error) {
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

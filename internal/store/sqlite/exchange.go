package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"

	"savvy-go/internal/db"
	"savvy-go/internal/legacy"
	"savvy-go/internal/migrate"
	"savvy-go/internal/store"
)

const stagingDir = "staging"

// ErrWrongKind is returned when a file is not the kind of database asked for
// (a server backup given as a space backup, or the other way round).
var ErrWrongKind = errors.New("the file is not this kind of backup")

// newStaging creates a private directory under the data directory for a
// prepared artifact; it is on the same volume, so swapping is a rename.
func (s *Store) newStaging() (string, error) {
	root := filepath.Join(s.opts.Dir, stagingDir)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(root, "prep-")
}

// Discard removes a prepared artifact.
func (s *Store) Discard(artifact string) {
	if artifact != "" && filepath.Dir(artifact) == filepath.Join(s.opts.Dir, stagingDir) {
		_ = os.RemoveAll(artifact)
	}
}

func (s *Store) ExportSpace(ctx context.Context, id int64, dest string) error {
	sp, err := s.lookup(id)
	if err != nil {
		return err
	}
	return vacuumInto(ctx, handle{sp}, dest)
}

func vacuumInto(ctx context.Context, conn store.DB, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o775); err != nil {
		return err
	}
	removeFiles(dest)
	_, err := conn.ExecContext(ctx, `VACUUM INTO ?`, dest)
	return err
}

func (s *Store) ExportServer(ctx context.Context, dir string) error {
	s.layout.Lock()
	defer s.layout.Unlock()
	if err := vacuumInto(ctx, s.Server(), filepath.Join(dir, serverFile)); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	ids, _ := s.Spaces(ctx)
	for _, id := range ids {
		dest := filepath.Join(dir, spacesDir, strconv.FormatInt(id, 10)+spaceExt)
		sp, err := s.lookup(id)
		if err != nil {
			// An unavailable space is kept as it is on disk.
			if err := copyPlain(s.spacePath(id), dest); err != nil {
				return fmt.Errorf("space %d: %w", id, err)
			}
			continue
		}
		if err := vacuumInto(ctx, handle{sp}, dest); err != nil {
			return fmt.Errorf("space %d: %w", id, err)
		}
	}
	return nil
}

func copyPlain(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o775); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func (s *Store) PrepareSpace(ctx context.Context, src string) (*store.PreparedSpace, error) {
	kind, err := inspect(ctx, src)
	if err != nil {
		return nil, err
	}
	staging, err := s.newStaging()
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*store.PreparedSpace, error) {
		_ = os.RemoveAll(staging)
		return nil, err
	}
	switch kind {
	case kindSpace, kindCombined:
	case kindLaravel:
		if src, err = s.convertLaravel(ctx, src, staging); err != nil {
			return fail(err)
		}
	default:
		return fail(ErrWrongKind)
	}
	out := filepath.Join(staging, "space"+spaceExt)
	if err := sanitize(ctx, src, out, copyPlan{set: migrate.Space}); err != nil {
		return fail(err)
	}
	id, err := readSpaceUUID(ctx, out)
	if err != nil {
		return fail(err)
	}
	return &store.PreparedSpace{Artifact: staging, UUID: id, Size: fileSize(out)}, nil
}

// convertLaravel converts a copy of a Laravel-era database to the Go schema
// (everything in one file) and returns its path. The copy is opened as a
// live database, so its triggers and views are dropped first: the conversion
// writes to it and must not run anything the file brought along.
func (s *Store) convertLaravel(ctx context.Context, src, staging string) (string, error) {
	dst := filepath.Join(staging, "laravel"+spaceExt)
	if err := copyPlain(src, dst); err != nil {
		return "", err
	}
	d, err := db.Open(dst)
	if err != nil {
		return "", err
	}
	defer d.Close()
	if _, err := d.ExecContext(ctx, `PRAGMA trusted_schema = OFF`); err != nil {
		return "", err
	}
	rows, err := d.QueryContext(ctx, `SELECT type, name FROM sqlite_master WHERE type IN ('trigger', 'view')`)
	if err != nil {
		return "", errNotSQLite
	}
	var drops []string
	for rows.Next() {
		var typ, name string
		if err := rows.Scan(&typ, &name); err != nil {
			rows.Close()
			return "", err
		}
		drops = append(drops, fmt.Sprintf("DROP %s IF EXISTS %s", map[string]string{"trigger": "TRIGGER", "view": "VIEW"}[typ], quoteIdent(name)))
	}
	rows.Close()
	for _, q := range drops {
		if _, err := d.ExecContext(ctx, q); err != nil {
			return "", err
		}
	}
	if err := legacy.Upgrade(ctx, d, s.opts.AppKey); err != nil {
		return "", err
	}
	if _, err := d.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return "", err
	}
	return dst, nil
}

func (s *Store) PrepareServer(ctx context.Context, src string) (*store.PreparedServer, error) {
	staging, err := s.newStaging()
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*store.PreparedServer, error) {
		_ = os.RemoveAll(staging)
		return nil, err
	}
	var ids []int64
	if info, err := os.Stat(src); err == nil && info.IsDir() {
		ids, err = s.prepareServerDir(ctx, src, staging)
		if err != nil {
			return fail(err)
		}
	} else {
		kind, err := inspect(ctx, src)
		if err != nil {
			return fail(err)
		}
		switch kind {
		case kindCombined:
		case kindLaravel:
			if src, err = s.convertLaravel(ctx, src, staging); err != nil {
				return fail(err)
			}
		default:
			return fail(ErrWrongKind)
		}
		if err := splitCombined(ctx, src, staging); err != nil {
			return fail(err)
		}
		ids = []int64{1}
	}
	return &store.PreparedServer{Artifact: staging, Spaces: ids}, nil
}

// prepareServerDir sanitizes a directory written by ExportServer. Every space
// in the server's registry must have a database and every database a row.
func (s *Store) prepareServerDir(ctx context.Context, dir, staging string) ([]int64, error) {
	src := filepath.Join(dir, serverFile)
	if kind, err := inspect(ctx, src); err != nil {
		return nil, err
	} else if kind != kindServer {
		return nil, ErrWrongKind
	}
	out := filepath.Join(staging, serverFile)
	if err := sanitize(ctx, src, out, copyPlan{set: migrate.Server}); err != nil {
		return nil, fmt.Errorf("server: %w", err)
	}
	registered, err := registeredSpaces(ctx, out)
	if err != nil {
		return nil, err
	}
	files, err := spaceIDs(filepath.Join(dir, spacesDir))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if !slices.Equal(registered, files) {
		return nil, fmt.Errorf("the backup's spaces %v do not match its space files %v", registered, files)
	}
	for _, id := range files {
		name := strconv.FormatInt(id, 10) + spaceExt
		if kind, err := inspect(ctx, filepath.Join(dir, spacesDir, name)); err != nil {
			return nil, fmt.Errorf("space %d: %w", id, err)
		} else if kind != kindSpace {
			return nil, fmt.Errorf("space %d: %w", id, ErrWrongKind)
		}
		if err := sanitize(ctx, filepath.Join(dir, spacesDir, name), filepath.Join(staging, spacesDir, name), copyPlan{set: migrate.Space}); err != nil {
			return nil, fmt.Errorf("space %d: %w", id, err)
		}
	}
	return files, nil
}

func spaceIDs(dir string) ([]int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, e := range entries {
		var id int64
		if _, err := fmt.Sscanf(e.Name(), "%d"+spaceExt, &id); err == nil && id > 0 && e.Name() == strconv.FormatInt(id, 10)+spaceExt {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

func registeredSpaces(ctx context.Context, serverPath string) ([]int64, error) {
	d, err := db.Open(serverPath)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	rows, err := d.QueryContext(ctx, `SELECT id FROM spaces ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// retiredRole maps the read-write/read-only roles of a single-file database
// to the server role user.
const retiredRole = `CASE %s WHEN 'read-write' THEN 'user' WHEN 'read-only' THEN 'user' ELSE %s END`

// splitCombined turns a database that holds everything (the single-file Go
// layout or a converted Laravel database) into a server database and space 1.
// Every user joins space 1 with the space role of their old role: admins
// administer it, read-write users edit and read-only users view it.
func splitCombined(ctx context.Context, src, staging string) error {
	server := filepath.Join(staging, serverFile)
	plan := copyPlan{set: migrate.Server, transforms: map[string]map[string]string{
		"users":              {"role": fmt.Sprintf(retiredRole, "role", "role")},
		"identity_providers": {"default_role": fmt.Sprintf(retiredRole, "default_role", "default_role")},
	}}
	if err := sanitize(ctx, src, server, plan); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	space := filepath.Join(staging, spacesDir, "1"+spaceExt)
	if err := os.MkdirAll(filepath.Dir(space), 0o775); err != nil {
		return err
	}
	if err := sanitize(ctx, src, space, copyPlan{set: migrate.Space}); err != nil {
		return fmt.Errorf("space: %w", err)
	}
	roles, err := oldRoles(ctx, src)
	if err != nil {
		return err
	}
	id7, err := uuid.NewV7()
	if err != nil {
		return err
	}
	d, err := db.Open(server)
	if err != nil {
		return err
	}
	defer d.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	err = store.Tx(ctx, d, func(tx store.DB) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO spaces (id, uuid, name, created_at, updated_at) VALUES (1, ?, 'Default', ?, ?)`,
			id7.String(), now, now); err != nil {
			return err
		}
		for userID, role := range roles {
			spaceRole := map[string]string{"admin": "admin", "read-write": "editor"}[role]
			if spaceRole == "" {
				spaceRole = "viewer"
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO space_members (space_id, user_id, role, created_at) VALUES (1, ?, ?, ?)`,
				userID, spaceRole, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if _, err := d.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return err
	}
	return writeSpaceUUID(ctx, space, id7.String())
}

// oldRoles reads each user's role from a single-file database.
func oldRoles(ctx context.Context, src string) (map[int64]string, error) {
	scratch, err := openScratch(ctx)
	if err != nil {
		return nil, err
	}
	defer scratch.Close()
	if err := attachSource(ctx, scratch, src); err != nil {
		return nil, err
	}
	rows, err := scratch.QueryContext(ctx, `SELECT id, role FROM src.users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var role sql.NullString
		if err := rows.Scan(&id, &role); err != nil {
			return nil, err
		}
		out[id] = role.String
	}
	return out, rows.Err()
}

func (s *Store) ReplaceSpace(ctx context.Context, id int64, p *store.PreparedSpace) error {
	defer s.Discard(p.Artifact)
	sp, err := s.lookup(id)
	if err != nil {
		return err
	}
	unlock := s.exclusive(sp)
	defer unlock()
	if err := sp.db.Close(); err != nil {
		return err
	}
	prepared := filepath.Join(p.Artifact, "space"+spaceExt)
	if err := writeSettings(ctx, prepared, p.Settings); err != nil {
		return s.reopenOrFail(ctx, sp, err)
	}
	if err := swapIn(prepared, s.spacePath(id)); err != nil {
		return s.reopenOrFail(ctx, sp, err)
	}
	return s.reopenOrFail(ctx, sp, nil)
}

// reopenOrFail reopens a space after its file changed; a space that cannot be
// opened becomes unavailable.
func (s *Store) reopenOrFail(ctx context.Context, sp *space, cause error) error {
	sdb, err := s.openSpace(ctx, sp.id)
	if err != nil {
		sp.gone = true
		s.mu.Lock()
		delete(s.spaces, sp.id)
		s.unavailable[sp.id] = err.Error()
		s.mu.Unlock()
		return errors.Join(cause, err)
	}
	sp.db = sdb
	return cause
}

// writeSettings stores space settings in a prepared space database.
func writeSettings(ctx context.Context, path string, kv map[string]string) error {
	if len(kv) == 0 {
		return nil
	}
	d, err := db.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	for k, v := range kv {
		if _, err := d.ExecContext(ctx, `INSERT INTO space_settings (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return err
		}
	}
	_, err = d.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

// swapIn replaces dst (and its side files) with src.
func swapIn(src, dst string) error {
	removeFiles(dst)
	if err := os.MkdirAll(filepath.Dir(dst), 0o775); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

func (s *Store) ImportSpace(ctx context.Context, id int64, p *store.PreparedSpace) error {
	defer s.Discard(p.Artifact)
	s.layout.RLock()
	defer s.layout.RUnlock()
	s.mu.Lock()
	_, exists := s.spaces[id]
	_, broken := s.unavailable[id]
	s.mu.Unlock()
	if exists || broken {
		return fmt.Errorf("space %d already exists", id)
	}
	if _, err := os.Stat(s.spacePath(id)); err == nil {
		return fmt.Errorf("space %d already exists", id)
	}
	if err := swapIn(filepath.Join(p.Artifact, "space"+spaceExt), s.spacePath(id)); err != nil {
		return err
	}
	sdb, err := s.openSpace(ctx, id)
	if err != nil {
		removeFiles(s.spacePath(id))
		return err
	}
	s.mu.Lock()
	s.spaces[id] = &space{id: id, db: sdb}
	s.mu.Unlock()
	return nil
}

// ReplaceServer swaps every database for the prepared ones. All spaces are
// locked in id order, closed, replaced and reopened; statements already
// holding a space that is gone afterwards fail with store.ErrNotFound.
func (s *Store) ReplaceServer(ctx context.Context, p *store.PreparedServer) error {
	defer s.Discard(p.Artifact)
	s.layout.Lock()
	defer s.layout.Unlock()

	s.mu.Lock()
	old := make([]*space, 0, len(s.spaces))
	for _, sp := range s.spaces {
		old = append(old, sp)
	}
	s.mu.Unlock()
	slices.SortFunc(old, func(a, b *space) int { return int(a.id - b.id) })
	for _, sp := range old {
		sp.write.Lock()
		sp.gate.Lock()
	}
	defer func() {
		for i := len(old) - 1; i >= 0; i-- {
			old[i].gate.Unlock()
			old[i].write.Unlock()
		}
	}()
	s.serverGate.Lock()
	defer s.serverGate.Unlock()

	for _, sp := range old {
		sp.gone = true
		_ = sp.db.Close()
	}
	_ = s.server.Close()

	removeFiles(filepath.Join(s.opts.Dir, serverFile))
	spaces := filepath.Join(s.opts.Dir, spacesDir)
	if ids, err := spaceIDs(spaces); err == nil {
		for _, id := range ids {
			removeFiles(s.spacePath(id))
		}
	}
	if err := os.Rename(filepath.Join(p.Artifact, serverFile), filepath.Join(s.opts.Dir, serverFile)); err != nil {
		return err
	}
	for _, id := range p.Spaces {
		name := strconv.FormatInt(id, 10) + spaceExt
		if err := swapIn(filepath.Join(p.Artifact, spacesDir, name), s.spacePath(id)); err != nil {
			return err
		}
	}
	return s.openAll(ctx)
}

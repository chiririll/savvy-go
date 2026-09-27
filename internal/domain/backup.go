package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/migrate"
	"savvy-go/internal/version"
)

type Backup struct {
	ID         int64
	Filename   string
	Size       int64
	Note       *string
	AppVersion *string
	Migrations *string
	CreatedAt  *time.Time
}

type Backups struct {
	DB       *sql.DB
	Dir      string
	Database string
}

func (s Backups) All(ctx context.Context) ([]Backup, error) {
	rows, err := db.Q(s.DB).ListBackups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Backup, 0, len(rows))
	for _, r := range rows {
		out = append(out, backupFrom(r.ID, r.Filename, r.Size, r.Note, r.AppVersion, r.SchemaMigrations, r.CreatedAt))
	}
	return out, nil
}

func (s Backups) ByID(ctx context.Context, id int64) (*Backup, error) {
	r, err := db.Q(s.DB).GetBackup(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b := backupFrom(r.ID, r.Filename, r.Size, r.Note, r.AppVersion, r.SchemaMigrations, r.CreatedAt)
	return &b, nil
}

func (s Backups) Create(ctx context.Context, note *string) (*Backup, error) {
	if err := os.MkdirAll(s.Dir, 0o775); err != nil {
		return nil, err
	}
	if err := db.Checkpoint(ctx, s.DB); err != nil {
		return nil, err
	}
	name := time.Now().UTC().Format("20060102-150405") + ".sqlite"
	dest := filepath.Join(s.Dir, name)
	if err := copyFile(s.Database, dest); err != nil {
		return nil, err
	}
	info, err := os.Stat(dest)
	if err != nil {
		return nil, err
	}
	ver := version.Value
	migsJSON := migrationsJSON(ctx, s.DB)
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertBackup(ctx, sqlc.InsertBackupParams{
		Filename: name, Size: info.Size(), Note: db.NullString(note), AppVersion: db.NS(ver),
		SchemaMigrations: db.NS(migsJSON), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Backups) Ingest(ctx context.Context, srcPath string, note *string) (*Backup, error) {
	if err := os.MkdirAll(s.Dir, 0o775); err != nil {
		return nil, err
	}
	name := time.Now().UTC().Format("20060102-150405") + "-upload.sqlite"
	dest := filepath.Join(s.Dir, name)
	if err := copyFile(srcPath, dest); err != nil {
		return nil, err
	}
	_ = os.Remove(srcPath)
	info, err := os.Stat(dest)
	if err != nil {
		return nil, err
	}
	ver := version.Value
	migsJSON := migrationsJSON(ctx, s.DB)
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertBackup(ctx, sqlc.InsertBackupParams{
		Filename: name, Size: info.Size(), Note: db.NullString(note), AppVersion: db.NS(ver),
		SchemaMigrations: db.NS(migsJSON), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Backups) Delete(ctx context.Context, b Backup) error {
	_ = os.Remove(filepath.Join(s.Dir, b.Filename))
	return db.Q(s.DB).DeleteBackup(ctx, b.ID)
}

func (s Backups) Path(b Backup) string {
	return filepath.Join(s.Dir, b.Filename)
}

func (s Backups) Inspect(ctx context.Context, b Backup) (map[string]any, error) {
	src, err := sql.Open("sqlite", "file:"+filepath.ToSlash(s.Path(b))+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer src.Close()
	ran, _ := sqlc.New(src).ListSchemaMigrations(ctx)
	available, _ := db.Q(s.DB).ListSchemaMigrations(ctx)
	pending := diffStrings(available, ran)
	unknown := diffStrings(ran, available)
	return map[string]any{
		"valid":             true,
		"compatible":        len(unknown) == 0,
		"pendingCount":      len(pending),
		"pendingMigrations": pending,
		"unknownCount":      len(unknown),
		"unknownMigrations": unknown,
	}, nil
}

func migrationsJSON(ctx context.Context, sqlDB *sql.DB) string {
	migs, _ := db.Q(sqlDB).ListSchemaMigrations(ctx)
	data, _ := json.Marshal(migs)
	return string(data)
}

func (s Backups) Restore(ctx context.Context, b Backup) error {
	src := s.Path(b)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("backup file missing")
	}

	// Copy backup to a temp file so the original is never modified.
	tmp, err := os.CreateTemp(s.Dir, "restore-*.sqlite")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmp.Close()
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
		_ = os.Remove(tmpPath + "-wal")
		_ = os.Remove(tmpPath + "-shm")
	}
	defer cleanup()

	if err := copyFile(src, tmpPath); err != nil {
		return fmt.Errorf("copy backup: %w", err)
	}

	// Apply any pending migrations to the temp copy so schema matches current.
	tmpDB, err := db.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	if err := migrate.Up(ctx, tmpDB); err != nil {
		_ = tmpDB.Close()
		return fmt.Errorf("migrate backup: %w", err)
	}
	_ = db.Checkpoint(ctx, tmpDB)
	if err := tmpDB.Close(); err != nil {
		return fmt.Errorf("close backup: %w", err)
	}

	// Build a file URI for the temp file (modernc.org/sqlite requires forward slashes).
	tmpSlash := filepath.ToSlash(tmpPath)
	if len(tmpSlash) > 1 && tmpSlash[1] == ':' {
		tmpSlash = "/" + tmpSlash
	}
	attachURI := "file:" + tmpSlash + "?mode=ro"

	// Turn off FK checks before any writes (must be outside a transaction in SQLite).
	if _, err := s.DB.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return fmt.Errorf("fk off: %w", err)
	}
	defer s.DB.ExecContext(ctx, `PRAGMA foreign_keys=ON`) //nolint:errcheck

	// Attach the migrated backup as a read-only secondary database.
	if _, err := s.DB.ExecContext(ctx, `ATTACH DATABASE ? AS _restore`, attachURI); err != nil {
		return fmt.Errorf("attach: %w", err)
	}
	defer s.DB.ExecContext(ctx, `DETACH DATABASE _restore`) //nolint:errcheck

	// Enumerate user tables defined in the backup (creation order keeps FK deps right).
	rows, err := s.DB.QueryContext(ctx,
		`SELECT name FROM _restore.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY rowid`)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	var tables []string
	for rows.Next() {
		var t string
		_ = rows.Scan(&t)
		tables = append(tables, t)
	}
	_ = rows.Close()

	// Replace each table's content inside a single transaction.
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	for _, table := range tables {
		safe := strings.ReplaceAll(table, `"`, `""`)
		if _, err := tx.ExecContext(ctx, `DELETE FROM main."`+safe+`"`); err != nil {
			continue // table absent in current schema — skip
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO main."`+safe+`" SELECT * FROM _restore."`+safe+`"`); err != nil {
			return fmt.Errorf("copy %s: %w", table, err)
		}
	}
	return tx.Commit()
}

func backupFrom(id int64, filename string, size int64, note, ver, mig, created sql.NullString) Backup {
	b := Backup{ID: id, Filename: filename, Size: size}
	if note.Valid {
		b.Note = &note.String
	}
	if ver.Valid {
		b.AppVersion = &ver.String
	}
	if mig.Valid {
		b.Migrations = &mig.String
	}
	if tm, ok := parseNullTime(created); ok {
		b.CreatedAt = &tm
	}
	return b
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func diffStrings(a, b []string) []string {
	have := map[string]bool{}
	for _, x := range b {
		have[x] = true
	}
	var out []string
	for _, x := range a {
		if !have[x] {
			out = append(out, x)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

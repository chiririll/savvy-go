package domain

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/version"
)

const backupExt = ".sqlite"

// Backup metadata lives inside the backup file itself (table backup_meta), so
// the directory is the single source of truth and metadata survives
// download/upload. The live database never has this table.
const (
	metaNote       = "note"
	metaAppVersion = "app_version"
	metaCreatedAt  = "created_at"
)

type Backup struct {
	Filename   string
	Size       int64
	Note       *string
	AppVersion *string
	CreatedAt  *time.Time
}

type Backups struct {
	DB       *sql.DB
	Dir      string
	Database string
}

// All lists *.sqlite files in the backup directory, newest first.
func (s Backups) All(ctx context.Context) ([]Backup, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Backup{}, nil
		}
		return nil, err
	}
	out := make([]Backup, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), backupExt) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		out = append(out, s.describe(ctx, info))
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.CreatedAt != nil && b.CreatedAt != nil && !a.CreatedAt.Equal(*b.CreatedAt) {
			return a.CreatedAt.After(*b.CreatedAt)
		}
		return a.Filename > b.Filename
	})
	return out, nil
}

// ByName returns the backup with the given file name, or nil if the name is
// invalid or the file does not exist.
func (s Backups) ByName(ctx context.Context, name string) (*Backup, error) {
	if name == "" || filepath.Base(name) != name || !strings.HasSuffix(name, backupExt) {
		return nil, nil
	}
	info, err := os.Stat(filepath.Join(s.Dir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, nil
	}
	b := s.describe(ctx, info)
	return &b, nil
}

func (s Backups) describe(ctx context.Context, info os.FileInfo) Backup {
	b := Backup{Filename: info.Name(), Size: info.Size()}
	meta := db.ReadBackupMeta(ctx, filepath.Join(s.Dir, info.Name()))
	if v, ok := meta[metaNote]; ok && v != "" {
		b.Note = &v
	}
	if v, ok := meta[metaAppVersion]; ok && v != "" {
		b.AppVersion = &v
	}
	if tm, ok := parseNullTime(sql.NullString{String: meta[metaCreatedAt], Valid: meta[metaCreatedAt] != ""}); ok {
		b.CreatedAt = &tm
	} else {
		tm := info.ModTime().UTC()
		b.CreatedAt = &tm
	}
	return b
}

func (s Backups) Create(ctx context.Context, note *string) (*Backup, error) {
	if err := os.MkdirAll(s.Dir, 0o775); err != nil {
		return nil, err
	}
	if err := db.Checkpoint(ctx, s.DB); err != nil {
		return nil, err
	}
	name := time.Now().UTC().Format("20060102-150405") + backupExt
	dest := filepath.Join(s.Dir, name)
	if err := copyFile(s.Database, dest); err != nil {
		return nil, err
	}
	meta := map[string]string{
		metaAppVersion: version.Value,
		metaCreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if note != nil && *note != "" {
		meta[metaNote] = *note
	}
	if err := db.WriteBackupMeta(ctx, dest, meta, true); err != nil {
		_ = os.Remove(dest)
		return nil, err
	}
	return s.ByName(ctx, name)
}

// Ingest moves an uploaded file into the backup directory. Metadata already
// present in the file (e.g. from a download of one of our backups) is kept.
func (s Backups) Ingest(ctx context.Context, srcPath string, note *string) (*Backup, error) {
	if err := os.MkdirAll(s.Dir, 0o775); err != nil {
		return nil, err
	}
	name := time.Now().UTC().Format("20060102-150405") + "-upload" + backupExt
	dest := filepath.Join(s.Dir, name)
	if err := copyFile(srcPath, dest); err != nil {
		return nil, err
	}
	_ = os.Remove(srcPath)
	if note != nil && *note != "" {
		if err := db.WriteBackupMeta(ctx, dest, map[string]string{metaNote: *note}, true); err != nil {
			_ = os.Remove(dest)
			return nil, err
		}
	}
	err := db.WriteBackupMeta(ctx, dest, map[string]string{metaCreatedAt: time.Now().UTC().Format(time.RFC3339)}, false)
	if err != nil {
		_ = os.Remove(dest)
		return nil, err
	}
	return s.ByName(ctx, name)
}

func (s Backups) Delete(_ context.Context, b Backup) error {
	return os.Remove(filepath.Join(s.Dir, b.Filename))
}

func (s Backups) Path(b Backup) string {
	return filepath.Join(s.Dir, b.Filename)
}

// Migrations compares the Go schema migrations recorded in b with the ones
// this app knows: pending ones restore will apply, unknown ones come from a
// newer app. It fails for a file without a Go schema (not SQLite, or Laravel).
func (s Backups) Migrations(ctx context.Context, b Backup) (pending, unknown []string, err error) {
	src, err := db.OpenReadOnly(s.Path(b))
	if err != nil {
		return nil, nil, err
	}
	defer src.Close()
	ran, err := sqlc.New(src).ListSchemaMigrations(ctx)
	if err != nil {
		return nil, nil, err
	}
	available, err := db.Q(s.DB).ListSchemaMigrations(ctx)
	if err != nil {
		return nil, nil, err
	}
	return diffStrings(available, ran), diffStrings(ran, available), nil
}

// Restore replaces the live database with the given backup and returns a new
// *sql.DB connected to the restored file. The caller must call Server.reconnect
// with the returned DB so all domain objects switch to the new connection.
// Restore replaces the live database with the backup. The backup is first
// copied aside and brought up to date by prepare (schema migrations, legacy
// upgrade); only if that succeeds is the live database closed and replaced, so
// a bad or unsupported backup leaves the current database and connection
// untouched.
func (s Backups) Restore(ctx context.Context, b Backup, prepare func(context.Context, *sql.DB) error) (*sql.DB, error) {
	src := s.Path(b)
	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("backup file missing")
	}

	staged := s.Database + ".restore"
	defer removeSQLiteFiles(staged)
	removeSQLiteFiles(staged)
	if err := copyFile(src, staged); err != nil {
		return nil, fmt.Errorf("copy backup: %w", err)
	}
	if err := prepareStaged(ctx, staged, prepare); err != nil {
		return nil, err
	}

	// Close current connection before replacing the file.
	if err := s.DB.Close(); err != nil {
		return nil, fmt.Errorf("close db: %w", err)
	}
	removeSQLiteFiles(s.Database)
	if err := copyFile(staged, s.Database); err != nil {
		return nil, fmt.Errorf("copy backup: %w", err)
	}

	// Open a fresh connection to the restored file.
	newDB, err := db.Open(s.Database)
	if err != nil {
		return nil, fmt.Errorf("reopen db: %w", err)
	}
	return newDB, nil
}

// prepareStaged opens the staged copy, runs prepare and drops backup_meta
// (it only describes the backup file, not the live database), then closes it so
// the file is self-contained again.
func prepareStaged(ctx context.Context, path string, prepare func(context.Context, *sql.DB) error) error {
	staged, err := db.Open(path)
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	defer staged.Close()
	if prepare != nil {
		if err := prepare(ctx, staged); err != nil {
			return err
		}
	}
	if err := db.DropBackupMeta(ctx, staged); err != nil {
		return fmt.Errorf("drop backup meta: %w", err)
	}
	if err := db.Checkpoint(ctx, staged); err != nil {
		return fmt.Errorf("checkpoint: %w", err)
	}
	return nil
}

// removeSQLiteFiles deletes a database file and its WAL/SHM side files.
func removeSQLiteFiles(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}
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

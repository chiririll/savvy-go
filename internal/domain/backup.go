package domain

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/db/sqlc"
	"github.com/chiririll/savvy-plus/internal/version"
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

func (b Backup) JSON() map[string]any {
	status := "current"
	return map[string]any{
		"id": b.ID, "filename": b.Filename, "size": b.Size, "note": b.Note,
		"schemaVersion": b.AppVersion, "schemaStatus": status,
		"createdAt": func() any {
			if b.CreatedAt == nil {
				return nil
			}
			return b.CreatedAt.UTC().Format(time.RFC3339Nano)
		}(),
	}
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
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertBackup(ctx, sqlc.InsertBackupParams{
		Filename: name, Size: info.Size(), Note: db.NullString(note), AppVersion: db.NS(ver),
		CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
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
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertBackup(ctx, sqlc.InsertBackupParams{
		Filename: name, Size: info.Size(), Note: db.NullString(note), AppVersion: db.NS(ver),
		CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
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
	return map[string]any{"pendingMigrations": pending, "unknownMigrations": unknown}, nil
}

func (s Backups) Restore(ctx context.Context, b Backup) error {
	src := s.Path(b)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("backup file missing")
	}
	if err := db.Checkpoint(ctx, s.DB); err != nil {
		return err
	}
	if err := copyFile(src, s.Database); err != nil {
		return err
	}
	_ = os.Remove(s.Database + "-wal")
	_ = os.Remove(s.Database + "-shm")
	return nil
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

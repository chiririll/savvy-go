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
	rows, err := s.DB.QueryContext(ctx, `SELECT id, filename, size, note, app_version, schema_migrations, created_at FROM backups ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Backup
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s Backups) ByID(ctx context.Context, id int64) (*Backup, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, filename, size, note, app_version, schema_migrations, created_at FROM backups WHERE id = ?`, id)
	b, err := scanBackup(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
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
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO backups (filename, size, note, app_version, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		name, info.Size(), note, ver, now, now)
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
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO backups (filename, size, note, app_version, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		name, info.Size(), note, ver, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Backups) Delete(ctx context.Context, b Backup) error {
	_ = os.Remove(filepath.Join(s.Dir, b.Filename))
	_, err := s.DB.ExecContext(ctx, `DELETE FROM backups WHERE id = ?`, b.ID)
	return err
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
	var ran []string
	rows, err := src.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v string
			if rows.Scan(&v) == nil {
				ran = append(ran, v)
			}
		}
	}
	var available []string
	arows, err := s.DB.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err == nil {
		defer arows.Close()
		for arows.Next() {
			var v string
			if arows.Scan(&v) == nil {
				available = append(available, v)
			}
		}
	}
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

func scanBackup(row interface{ Scan(...any) error }) (Backup, error) {
	var b Backup
	var note, ver, mig, created sql.NullString
	err := row.Scan(&b.ID, &b.Filename, &b.Size, &note, &ver, &mig, &created)
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
	return b, err
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

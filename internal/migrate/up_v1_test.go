package migrate

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"savvy-go/internal/db"
	v1 "savvy-go/internal/migrate/v1"
)

// A v1.0 file records space/0001_initial. The base is a different version
// whose CREATE TABLE IF NOT EXISTS would skip the old tables, so Up must refuse
// instead of leaving half a schema.
func TestUpRefusesAV1File(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "v1.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(tableSQL); err != nil {
		t.Fatal(err)
	}
	if err := execScript(ctx, sqlDB, v1.Schema); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO schema_migrations (version, applied_at) ` +
		`VALUES ('space/0001_initial', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}

	err = Space.Up(ctx, sqlDB)
	if !errors.Is(err, ErrV1Schema) {
		t.Fatalf("Up = %v, want ErrV1Schema", err)
	}
	if !strings.Contains(err.Error(), "v1.0") {
		t.Errorf("the message must say what to do: %v", err)
	}
	var n int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'payments'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("a refused Up must change nothing: %d, %v", n, err)
	}
}

// Server.Up must not trip over a v1.0 space version in a shared file.
func TestServerUpIgnoresAV1SpaceFile(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "v1.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(tableSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO schema_migrations (version, applied_at) ` +
		`VALUES ('space/0001_initial', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := Server.Up(ctx, sqlDB); err != nil {
		t.Fatalf("Server.Up = %v", err)
	}
}

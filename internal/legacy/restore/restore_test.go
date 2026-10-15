package restore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDatabaseFile(t *testing.T) {
	for name, want := range map[string]bool{
		"laravel.sqlite": true,
		"OLD.SQLITE":     true,
		"backup.zip":     false,
		"sqlite":         false,
		"":               false,
	} {
		if got := IsDatabaseFile(name); got != want {
			t.Errorf("IsDatabaseFile(%q) = %v", name, got)
		}
	}
}

func TestExceeds(t *testing.T) {
	p := filepath.Join(t.TempDir(), "db.sqlite")
	if err := os.WriteFile(p, make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}
	for limit, want := range map[int64]bool{0: false, -1: false, 99: true, 100: false, 101: false} {
		if got := exceeds(p, limit); got != want {
			t.Errorf("exceeds(100 bytes, %d) = %v", limit, got)
		}
	}
	if exceeds(filepath.Join(t.TempDir(), "missing"), 1) {
		t.Error("a missing file counts as too large")
	}
}

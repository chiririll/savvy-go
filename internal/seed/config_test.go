package seed

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSeedConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "seed.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigEnabledByDefault(t *testing.T) {
	for _, body := range []string{"", `date = "2026-06-05"`, "enabled = true"} {
		c, err := LoadConfig(writeSeedConfig(t, body))
		if err != nil {
			t.Fatalf("%q: %v", body, err)
		}
		if !c.Enabled {
			t.Fatalf("%q: seeding disabled, want enabled by default", body)
		}
	}
}

func TestLoadConfigCanDisable(t *testing.T) {
	c, err := LoadConfig(writeSeedConfig(t, "enabled = false"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Enabled {
		t.Fatal("enabled = false ignored")
	}
}

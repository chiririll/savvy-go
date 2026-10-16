package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/config"
)

func TestServerAndPathsSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	text := "[server]\napp_url = \" https://x.example/ \"\ntimezone = \"Nowhere/Land\"\n\n[paths]\ndata = \"" + filepath.ToSlash(dir) + "/d\"\n\n[unknown]\nkey = 1\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := config.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Load(f)
	if err := f.Finish(); err != nil {
		t.Fatal(err)
	}
	if cfg.TZ != "UTC" || cfg.Location != time.UTC {
		t.Errorf("an unknown time zone must fall back to UTC: %q", cfg.TZ)
	}
	if cfg.DataDir != filepath.ToSlash(dir)+"/d" || cfg.UploadsDir != filepath.Join(cfg.DataDir, "uploads") || cfg.ListenAddr != DefaultListen {
		t.Errorf("paths %+v", cfg)
	}
	if got := cfg.AppURL(); got != "https://x.example" {
		t.Errorf("AppURL %q", got)
	}
	out, _ := os.ReadFile(path)
	if strings.Contains(string(out), "unknown") || !strings.HasPrefix(string(out), "# Go Savvy configuration") {
		t.Errorf("file:\n%s", out)
	}
}

// AppURL follows the file: a change through the section shows at once.
func TestAppURLFollowsTheSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[server]\napp_url = \"https://x.example\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := config.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Load(f)
	if err := f.Finish(); err != nil {
		t.Fatal(err)
	}
	if err := ServerSection.Update(f, func(s *serverTable) { s.AppURL = "https://y.example///" }); err != nil {
		t.Fatal(err)
	}
	if got := cfg.AppURL(); got != "https://y.example" {
		t.Errorf("AppURL after a change %q", got)
	}
}

func TestAppURLFixedAndFollow(t *testing.T) {
	if got := (Config{}).AppURL(); got != "" {
		t.Errorf("zero Config AppURL %q", got)
	}
	if got := (Config{}).WithAppURL("https://y.example/").AppURL(); got != "https://y.example" {
		t.Errorf("fixed AppURL %q", got)
	}
	f, err := config.Parse("[server]\napp_url = \"https://z.example\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := (Config{}).Follow(f).AppURL(); got != "https://z.example" {
		t.Errorf("followed AppURL %q", got)
	}
}

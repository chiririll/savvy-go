package appconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"savvy-go/internal/config"
)

func TestLoadDecidesPathsFromTheFile(t *testing.T) {
	f, err := config.Parse("[paths]\ndata = \"/srv/d\"\nbackups = \"/srv/b\"\n")
	if err != nil {
		t.Fatal(err)
	}
	cfg := Load(f)
	if cfg.DataDir != "/srv/d" {
		t.Errorf("DataDir %q", cfg.DataDir)
	}
	if want := filepath.Join("/srv/d", "uploads"); cfg.UploadsDir != want {
		t.Errorf("an empty uploads path falls back to <data>/uploads: %q, want %q", cfg.UploadsDir, want)
	}
	if cfg.BackupsDir != "/srv/b" {
		t.Errorf("an explicit backups path wins: %q", cfg.BackupsDir)
	}
}

func TestLoadWithoutPathsUsesDetectedDir(t *testing.T) {
	f, err := config.Parse("")
	if err != nil {
		t.Fatal(err)
	}
	cfg := Load(f)
	if cfg.DataDir == "" || cfg.UploadsDir != filepath.Join(cfg.DataDir, "uploads") ||
		cfg.BackupsDir != filepath.Join(cfg.DataDir, "backups") {
		t.Errorf("defaults %+v", cfg)
	}
}

func TestPathsAreBlankSafe(t *testing.T) {
	f, err := config.Parse("[paths]\ndata = \"  \"\n")
	if err != nil {
		t.Fatal(err)
	}
	if cfg := Load(f); cfg.DataDir == "" || cfg.DataDir == "  " {
		t.Errorf("a blank path counts as unset: %q", cfg.DataDir)
	}
}

func TestZeroConfigHasNoPublicURL(t *testing.T) {
	if got := (Config{}).AppURL(); got != "" {
		t.Errorf("zero Config AppURL %q", got)
	}
}

func TestWithAppURLTrimsTrailingSlashes(t *testing.T) {
	if got := (Config{}).WithAppURL("https://y.example///").AppURL(); got != "https://y.example" {
		t.Errorf("fixed AppURL %q", got)
	}
}

// The URL is read on every call, so a change in the running process shows at once.
func TestWithServerReadsURLEachTime(t *testing.T) {
	url := "https://a.example"
	cfg := Config{}.WithServer("Europe/Paris", time.UTC, func() string { return url })
	if cfg.TZ != "Europe/Paris" || cfg.Location != time.UTC {
		t.Errorf("time zone %q %v", cfg.TZ, cfg.Location)
	}
	url = "https://b.example"
	if got := cfg.AppURL(); got != "https://b.example" {
		t.Errorf("AppURL after a change %q", got)
	}
}

func TestFilePathPrefersArgumentThenEnvironment(t *testing.T) {
	t.Setenv("CONFIG_FILE", " /env/config.toml ")
	if got := FilePath("/arg/config.toml"); got != "/arg/config.toml" {
		t.Errorf("an explicit path wins: %q", got)
	}
	if got := FilePath(""); got != "/env/config.toml" {
		t.Errorf("CONFIG_FILE is used and trimmed: %q", got)
	}
	t.Setenv("CONFIG_FILE", "")
	if got := FilePath(""); filepath.Base(got) != "config.toml" {
		t.Errorf("the default lives in the data directory: %q", got)
	}
}

func TestDetectDataDirExistingDirDefault(t *testing.T) {
	got := detectDataDir()
	if got == "./savvy-data" {
		return
	}
	if st, err := os.Stat(got); err != nil || !st.IsDir() {
		t.Errorf("detected %q is not a directory", got)
	}
}

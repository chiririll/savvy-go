package appconfig

import (
	"errors"
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

func TestSetAppURLUsesTheProcessSetter(t *testing.T) {
	url := "https://a.example"
	cfg := Config{}.
		WithServer("UTC", time.UTC, func() string { return url }).
		WithAppURLSetter(func(u string) error { url = u; return nil })
	if err := cfg.SetAppURL("https://b.example"); err != nil {
		t.Fatal(err)
	}
	if got := cfg.AppURL(); got != "https://b.example" {
		t.Errorf("AppURL after SetAppURL %q", got)
	}

	failing := cfg.WithAppURLSetter(func(string) error { return config.ErrReadOnly })
	if err := failing.SetAppURL("https://c.example"); !errors.Is(err, config.ErrReadOnly) {
		t.Errorf("the setter's error is lost: %v", err)
	}
}

func TestSetAppURLWithoutSetterIsReadOnly(t *testing.T) {
	if err := (Config{}).SetAppURL("https://a.example"); !errors.Is(err, config.ErrReadOnly) {
		t.Errorf("want ErrReadOnly, got %v", err)
	}
}

// inMemory is a Config whose public URL starts as url and is saved to the returned variable.
func inMemory(url string) (Config, *string) {
	cur := url
	cfg := Config{}.
		WithServer("UTC", time.UTC, func() string { return cur }).
		WithAppURLSetter(func(next string) error { cur = next; return nil })
	return cfg, &cur
}

func TestSetAppURLCleansAndChecksTheURL(t *testing.T) {
	ok := map[string]string{
		"":                           "",
		"  ":                         "",
		"https://savvy.example.com/": "https://savvy.example.com",
		"http://localhost:3000":      "http://localhost:3000",
	}
	for in, want := range ok {
		cfg, _ := inMemory("https://old.example")
		if err := cfg.SetAppURL(in); err != nil || cfg.AppURL() != want {
			t.Errorf("SetAppURL(%q): %v, now %q; want %q", in, err, cfg.AppURL(), want)
		}
	}
	for _, in := range []string{"savvy.example.com", "ftp://x", "https://", "https://x/?a=1", "https://x/#top"} {
		cfg, _ := inMemory("https://old.example")
		var field config.FieldError
		if err := cfg.SetAppURL(in); !errors.As(err, &field) || field.Field != "app_url" {
			t.Errorf("SetAppURL(%q) = %v; want a FieldError for app_url", in, err)
		}
		if cfg.AppURL() != "https://old.example" {
			t.Errorf("SetAppURL(%q) changed the URL to %q", in, cfg.AppURL())
		}
	}
}

// Sending the URL the instance already has is not a change, so it needs no write access.
func TestSetAppURLUnchangedNeedsNoWriteAccess(t *testing.T) {
	cfg := Config{}.WithServer("UTC", time.UTC, func() string { return "https://a.example" })
	if err := cfg.SetAppURL("https://a.example/"); err != nil {
		t.Errorf("an unchanged URL on a read-only file: %v", err)
	}
	if err := cfg.SetAppURL("https://b.example"); !errors.Is(err, config.ErrReadOnly) {
		t.Errorf("a changed URL on a read-only file: %v", err)
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

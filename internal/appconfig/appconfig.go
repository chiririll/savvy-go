// Package appconfig is the part of the config every package may need: the public URL and time
// zone, and where the data lives. A package takes a Config
// and asks it, instead of being handed a getter for each value.
package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"savvy-go/internal/config"
)

// pathsTable is the [paths] table.
type pathsTable struct {
	Data    string `toml:"data" comment:"Where the databases live. Empty: /data or /var/lib/savvy-go when they exist, otherwise ./savvy-data."`
	Uploads string `toml:"uploads" comment:"Empty: <data>/uploads."`
	Backups string `toml:"backups" comment:"Empty: <data>/backups."`
}

// Defaults are empty: the directories are decided at start. They are written so the keys show.
func (pathsTable) Defaults() pathsTable { return pathsTable{} }

// Paths is the [paths] section: the directories, decided.
type Paths struct{ Data, Uploads, Backups string }

func decodePaths(t pathsTable) (Paths, error) {
	data := firstNonEmpty(t.Data, detectDataDir())
	return Paths{
		Data:    data,
		Uploads: firstNonEmpty(t.Uploads, filepath.Join(data, "uploads")),
		Backups: firstNonEmpty(t.Backups, filepath.Join(data, "backups")),
	}, nil
}

// PathsSection is the [paths] section.
var PathsSection = config.Section[pathsTable, Paths]{Name: "paths", Decode: decodePaths}

// Build says which release is running. The binary sets it at link time; it is handed down
// so no package reads a global.
type Build struct{ Version, Env string }

// Config is what the process shares: the directories and time zone, fixed at start, and
// the public URL, which can change while the process runs. It is a small value: pass it
// around, do not wrap its values in getters. The zero value has no public URL.
//
// The [server] table belongs to the httpserver package, which cannot be imported from
// the packages that need these values; the process copies them in with WithServer.
type Config struct {
	DataDir    string
	UploadsDir string
	BackupsDir string
	TZ         string
	Location   *time.Location

	appURL func() string
}

// Load is the Config of the [paths] section of f. The time zone and public URL come from
// WithServer.
func Load(f *config.Config) Config {
	p := config.Of(f, PathsSection).Get()
	return Config{DataDir: p.Data, UploadsDir: p.Uploads, BackupsDir: p.Backups}
}

// WithServer sets the time zone, fixed for the run, and the source of the public URL, read
// each time so a change shows at once.
func (c Config) WithServer(tz string, loc *time.Location, appURL func() string) Config {
	c.TZ, c.Location, c.appURL = tz, loc, appURL
	return c
}

// WithAppURL is c with a fixed public URL, for tests.
func (c Config) WithAppURL(url string) Config {
	url = strings.TrimRight(url, "/")
	c.appURL = func() string { return url }
	return c
}

// AppURL is the public URL of the instance without a trailing slash, empty when unset.
func (c Config) AppURL() string {
	if c.appURL == nil {
		return ""
	}
	return c.appURL()
}

// FilePath is the config file to use: path when set, else the CONFIG_FILE environment
// variable, else config.toml in the data directory. CONFIG_FILE is the only environment the
// application reads; everything else is configured in the file.
func FilePath(path string) string {
	if path != "" {
		return path
	}
	if env := strings.TrimSpace(os.Getenv("CONFIG_FILE")); env != "" {
		return env
	}
	return filepath.Join(detectDataDir(), "config.toml")
}

// detectDataDir is /data or /var/lib/savvy-go when they exist, otherwise ./savvy-data.
func detectDataDir() string {
	for _, candidate := range []string{"/data", "/var/lib/savvy-go"} {
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
	}
	return "./savvy-data"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

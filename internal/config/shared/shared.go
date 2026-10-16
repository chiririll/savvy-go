// Package shared is the part of the config file every package may need: where the server
// listens, its public URL and time zone, and where the data lives. A package takes a Config
// and asks it, instead of being handed a getter for each value.
package shared

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"savvy-go/internal/config"
)

// DefaultListen is the listen address of a new config file. A build may set it with
// -ldflags "-X savvy-go/internal/config/shared.DefaultListen=:80" (the container image
// does); it is loopback only otherwise.
var DefaultListen = "localhost:8080"

// serverTable is the [server] table. An empty value means "decide at run time".
type serverTable struct {
	Listen   string `toml:"listen" comment:"Address to listen on. Loopback only by default; use \":8080\" to listen on every interface."`
	AppURL   string `toml:"app_url" comment:"Public URL of the instance, e.g. \"https://savvy.example.com\". SSO and passkeys stay disabled without it."`
	Timezone string `toml:"timezone" comment:"Time zone."`
}

func (serverTable) Defaults() serverTable {
	return serverTable{Listen: DefaultListen, Timezone: "UTC"}
}

// Server is the [server] section.
type Server struct {
	Listen string
	// AppURL is the public URL without a trailing slash, empty when unset.
	AppURL   string
	TZ       string
	Location *time.Location
}

// decodeServer fills what the file leaves empty. An unknown time zone is UTC.
func decodeServer(t serverTable) (Server, error) {
	tz := firstNonEmpty(t.Timezone, "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, tz = time.UTC, "UTC"
	}
	return Server{
		Listen:   firstNonEmpty(t.Listen, DefaultListen),
		AppURL:   strings.TrimRight(strings.TrimSpace(t.AppURL), "/"),
		TZ:       tz,
		Location: loc,
	}, nil
}

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

// The sections of the config file that this package reads.
var (
	ServerSection = config.Section[serverTable, Server]{Name: "server", Decode: decodeServer}
	PathsSection  = config.Section[pathsTable, Paths]{Name: "paths", Decode: decodePaths}
)

// Config is what the process shares: the directories and time zone, fixed at start, and
// the public URL, which can change while the process runs. It is a small value: pass it
// around, do not wrap its values in getters. The zero value has no public URL.
type Config struct {
	ListenAddr string
	DataDir    string
	UploadsDir string
	BackupsDir string
	TZ         string
	Location   *time.Location

	server *config.Ref[Server]
	appURL string // used when server is nil
}

// Load is the Config of the [server] and [paths] sections of f. Its AppURL follows the file.
func Load(f *config.Config) Config {
	server := config.Of(f, ServerSection)
	p := config.Of(f, PathsSection).Get()
	s := server.Get()
	return Config{
		ListenAddr: s.Listen,
		DataDir:    p.Data,
		UploadsDir: p.Uploads,
		BackupsDir: p.Backups,
		TZ:         s.TZ,
		Location:   s.Location,
		server:     server,
	}
}

// Follow makes AppURL follow the [server] section of f, when c does not follow a file yet.
func (c Config) Follow(f *config.Config) Config {
	if c.server == nil && c.appURL == "" {
		c.server = config.Of(f, ServerSection)
	}
	return c
}

// WithAppURL is c with a fixed public URL, for tests.
func (c Config) WithAppURL(url string) Config {
	c.server, c.appURL = nil, strings.TrimRight(url, "/")
	return c
}

// AppURL is the public URL of the instance without a trailing slash, empty when unset. It
// reads the section each time, so a change shows at once.
func (c Config) AppURL() string {
	if c.server != nil {
		return c.server.Get().AppURL
	}
	return c.appURL
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

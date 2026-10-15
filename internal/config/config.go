package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml/v2"

	"savvy-go/internal/mailer"
)

// Config is process-wide runtime configuration loaded from the environment.
type Config struct {
	// Live holds what the admin panel may change; prefer URL() over AppURL.
	Live          *Live
	AppURL        string
	ListenAddr    string
	DataDir       string
	UploadsDir    string
	BackupsDir    string
	TZ            string
	Location      *time.Location
	SessionTTL    time.Duration
	RememberTTL   time.Duration
	ChallengeTTL  time.Duration
	SessionCookie string
	CSRFCookie    string
	CSRFHeader    string
	// Mail is the SMTP server from [mailer] as loaded at start; the server keeps
	// the current one, which the admin panel can change. It is disabled when
	// MailProblem is set.
	Mail        mailer.Config
	MailProblem string
}

// file mirrors config.toml: it is decoded over defaults() and encoded back, so
// the struct is the single description of the keys, their defaults and their
// comments. An empty value means "decide at run time" (see resolve).
type file struct {
	Server struct {
		Listen   string `toml:"listen" comment:"Address to listen on. Loopback only by default; use \":8080\" to listen on every interface."`
		AppURL   string `toml:"app_url" comment:"Public URL of the instance, e.g. \"https://savvy.example.com\". SSO and passkeys stay disabled without it. Can also be set in the admin panel (System > Server)."`
		Timezone string `toml:"timezone" comment:"Time zone."`
	} `toml:"server"`
	Paths struct {
		Data    string `toml:"data" comment:"Where the databases live. Empty: /data or /var/lib/savvy-go when they exist, otherwise ./savvy-data."`
		Uploads string `toml:"uploads" comment:"Empty: <data>/uploads."`
		Backups string `toml:"backups" comment:"Empty: <data>/backups."`
	} `toml:"paths"`
	Security struct {
		SessionTTL   int    `toml:"session_ttl" comment:"Lifetimes in minutes."`
		RememberTTL  int    `toml:"remember_ttl"`
		ChallengeTTL int    `toml:"challenge_ttl"`
		Cookie       string `toml:"session_cookie" comment:"Cookie and header names."`
		CSRFCookie   string `toml:"csrf_cookie"`
		CSRFHeader   string `toml:"csrf_header"`
	} `toml:"security"`
	Mailer struct {
		Enabled    bool   `toml:"enabled"`
		Host       string `toml:"host"`
		Port       int    `toml:"port"`
		Protocol   string `toml:"protocol" comment:"Supported: \"starttls\", \"smtps\" or \"smtp\""`
		User       string `toml:"user"`
		Password   string `toml:"password"`
		From       string `toml:"from"`
		SkipVerify bool   `toml:"skip_verify"`
		Timeout    int    `toml:"timeout"`
	} `toml:"mailer"`
}

// defaults is the configuration of a new install.
func defaults() file {
	var f file
	f.Server.Listen = DefaultListen
	f.Server.Timezone = "UTC"
	f.Security.SessionTTL = 60 * 24
	f.Security.RememberTTL = 60 * 24 * 7
	f.Security.ChallengeTTL = 5
	f.Security.Cookie = "svy_session"
	f.Security.CSRFCookie = "svy_csrf"
	f.Security.CSRFHeader = "X-CSRF-Token"
	f.Mailer.Port = 587
	f.Mailer.Protocol = mailer.STARTTLS
	f.Mailer.Timeout = 10
	return f
}

// header opens every written config file.
const header = `# Go Savvy configuration, maintained by the application: it is rewritten on
# every start (new keys appear with their defaults) and when settings are changed
# in the admin panel. Your values are kept; comments and formatting are not.

`

// encode is the text of the config file.
func (f file) encode() ([]byte, error) {
	out, err := toml.Marshal(f)
	if err != nil {
		return nil, err
	}
	return append([]byte(header), out...), nil
}

// DefaultListen is the listen address of a new config file. A build may set it
// with -ldflags "-X savvy-go/internal/config.DefaultListen=:80" (the container
// image does); it is loopback only otherwise.
var DefaultListen = "localhost:8080"

// Load reads the TOML config file, adds the keys it lacks and writes it back,
// creating it on the first run. path is the file to use; when empty, the
// CONFIG_FILE environment variable, else config.toml in the data directory
// (/data or /var/lib/savvy-go when they exist, otherwise ./savvy-data). That variable
// is the only environment the application reads: everything else is configured
// in the file. Unknown keys are logged and dropped on the rewrite; invalid TOML
// is an error. A file that cannot be written (read-only mount) is logged and otherwise ignored.
func Load(path string) (Config, error) {
	if path == "" {
		path = strings.TrimSpace(os.Getenv("CONFIG_FILE"))
	}
	if path == "" {
		path = filepath.Join(detectDataDir(), "config.toml")
	}
	f := defaults()
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		dec := toml.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&f); err != nil {
			// Unknown keys come from a typo or a version that dropped one. The
			// rest of the file is decoded already, so do not stop the server.
			var unknown *toml.StrictMissingError
			if !errors.As(err, &unknown) {
				return Config{}, fmt.Errorf("parse %s: %w", path, err)
			}
			keys := make([]string, 0, len(unknown.Errors))
			for _, e := range unknown.Errors {
				keys = append(keys, strings.Join(e.Key(), "."))
			}
			slog.Warn("ignoring unknown keys in the config file; they are left out when it is rewritten", "path", path, "keys", keys)
		}
	case errors.Is(err, fs.ErrNotExist):
	default:
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	out, err := f.encode()
	if err != nil {
		return Config{}, fmt.Errorf("encode config: %w", err)
	}
	// Write even when nothing changed: it is the probe for "can this process
	// save settings later" (a read-only mount or directory fails here).
	writable := true
	if err := writeFile(path, out); err != nil {
		writable = false
		slog.Warn("could not write the config file; settings changed in the admin panel are disabled", "path", path, "err", err)
	}
	cfg := f.resolve()
	if cfg.AppURL == "" {
		slog.Warn("the public URL is not set; SSO and passkeys stay disabled until app_url is set in the config file or the admin panel")
	}
	if cfg.MailProblem != "" {
		slog.Warn("mail is disabled: invalid [mailer] section in the config file", "problem", cfg.MailProblem)
	}
	cfg.Live = &Live{path: path, f: f, appURL: cfg.AppURL, writable: writable}
	return cfg, nil
}

// writeFile replaces path atomically, creating its directory.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o775); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ErrReadOnly is returned when the config file cannot be written.
var ErrReadOnly = errors.New("the config file is read-only")

// Live is the part of the configuration the admin panel can change at run
// time, persisted to the config file.
type Live struct {
	mu     sync.RWMutex
	path   string
	f      file
	appURL string
	// writable is whether the config file could be written at start.
	writable bool
}

// Writable reports whether the admin panel can change the config file. False
// for a read-only mount; the settings it covers are then shown but locked.
func (l *Live) Writable() bool { return l != nil && l.writable }

// AppURL is the public URL of the instance, empty when unset.
func (l *Live) AppURL() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.appURL
}

// SetAppURL changes the public URL and writes the config file.
func (l *Live) SetAppURL(v string) error {
	return l.save(func(f *file) { f.Server.AppURL = v }, func() { l.appURL = v })
}

// SetMail replaces the [mailer] section and writes the config file. It does
// not validate: the caller does, and may save a disabled, half-filled section.
func (l *Live) SetMail(c mailer.Config) error {
	return l.save(func(f *file) {
		f.Mailer.Enabled = c.Enabled
		f.Mailer.Host = c.Host
		f.Mailer.Port = c.Port
		f.Mailer.Protocol = c.Protocol
		f.Mailer.User = c.User
		f.Mailer.Password = c.Password
		f.Mailer.From = c.From
		f.Mailer.SkipVerify = c.SkipVerify
		f.Mailer.Timeout = int(c.Timeout / time.Second)
	}, nil)
}

// save applies change to a copy of the file, writes it, and only then keeps it
// (and runs commit), so a failed write leaves memory and disk as they were.
func (l *Live) save(change func(*file), commit func()) error {
	if !l.Writable() {
		return ErrReadOnly
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	next := l.f
	change(&next)
	out, err := next.encode()
	if err == nil {
		err = writeFile(l.path, out)
	}
	if err != nil {
		return err
	}
	l.f = next
	if commit != nil {
		commit()
	}
	return nil
}

func (f file) resolve() Config {
	dataDir := firstNonEmpty(f.Paths.Data, detectDataDir())
	tz := firstNonEmpty(f.Server.Timezone, "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
		tz = "UTC"
	}

	cfg := Config{
		AppURL:        strings.TrimRight(strings.TrimSpace(f.Server.AppURL), "/"),
		ListenAddr:    firstNonEmpty(f.Server.Listen, DefaultListen),
		DataDir:       dataDir,
		UploadsDir:    firstNonEmpty(f.Paths.Uploads, filepath.Join(dataDir, "uploads")),
		BackupsDir:    firstNonEmpty(f.Paths.Backups, filepath.Join(dataDir, "backups")),
		TZ:            tz,
		Location:      loc,
		SessionTTL:    minutes(f.Security.SessionTTL, 60*24),
		RememberTTL:   minutes(f.Security.RememberTTL, 60*24*7),
		ChallengeTTL:  minutes(f.Security.ChallengeTTL, 5),
		SessionCookie: firstNonEmpty(f.Security.Cookie, "svy_session"),
		CSRFCookie:    firstNonEmpty(f.Security.CSRFCookie, "svy_csrf"),
		CSRFHeader:    firstNonEmpty(f.Security.CSRFHeader, "X-CSRF-Token"),
		Mail: mailer.Config{
			Enabled:    f.Mailer.Enabled,
			Host:       strings.TrimSpace(f.Mailer.Host),
			Port:       f.Mailer.Port,
			Protocol:   strings.ToLower(strings.TrimSpace(f.Mailer.Protocol)),
			User:       f.Mailer.User,
			Password:   f.Mailer.Password,
			From:       strings.TrimSpace(f.Mailer.From),
			SkipVerify: f.Mailer.SkipVerify,
			Timeout:    seconds(f.Mailer.Timeout, 10),
		},
	}
	// A mistake in [mailer] switches mail off instead of stopping the server;
	// the admin panel shows the reason.
	if err := cfg.Mail.Validate(); err != nil {
		cfg.MailProblem = err.Error()
		cfg.Mail.Enabled = false
	}
	return cfg
}

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

// minutes turns a TTL in minutes into a duration; non-positive values give fallback.
func minutes(fromFile, fallback int) time.Duration {
	if fromFile > 0 {
		return time.Duration(fromFile) * time.Minute
	}
	return time.Duration(fallback) * time.Minute
}

// seconds turns a timeout in seconds into a duration; non-positive values give fallback.
func seconds(fromFile, fallback int) time.Duration {
	if fromFile > 0 {
		return time.Duration(fromFile) * time.Second
	}
	return time.Duration(fallback) * time.Second
}

// URL is the current public URL of the instance, empty when unset.
func (c Config) URL() string {
	if c.Live != nil {
		return c.Live.AppURL()
	}
	return c.AppURL
}

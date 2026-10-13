package config

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadFileValues(t *testing.T) {
	path := write(t, `
[server]
listen = "127.0.0.1:9000"
app_url = "https://savvy.example.com/"
[paths]
data = "/srv/savvy"
[security]
session_ttl = 30
`)
	// The environment does not configure the application.
	t.Setenv("LISTEN_ADDR", ":7000")
	t.Setenv("SEED_DEMO", "false")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "127.0.0.1:9000" {
		t.Errorf("env must be ignored: %q", cfg.ListenAddr)
	}
	if cfg.AppURL != "https://savvy.example.com" || cfg.DataDir != "/srv/savvy" {
		t.Errorf("file values: %q %q", cfg.AppURL, cfg.DataDir)
	}
	if cfg.SessionTTL != 30*time.Minute {
		t.Errorf("ttl: %v", cfg.SessionTTL)
	}
}

// A key a version dropped (or a typo) must not stop the server: it is named in
// the log and left out when the file is rewritten.
func TestLoadIgnoresUnknownKeys(t *testing.T) {
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	path := write(t, "[server]\napp_url = \"https://a.example\"\nlisten_addr = \"x\"\n[paths]\npublic = \"\"\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppURL != "https://a.example" {
		t.Errorf("known key lost: %q", cfg.AppURL)
	}
	if !strings.Contains(logs.String(), "server.listen_addr") || !strings.Contains(logs.String(), "paths.public") {
		t.Errorf("unknown keys not named in the log: %s", logs.String())
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "listen_addr") || strings.Contains(string(raw), "public") {
		t.Errorf("unknown keys kept in the rewritten file:\n%s", raw)
	}
}

func TestLoadRejectsBrokenSyntax(t *testing.T) {
	if _, err := Load(write(t, "[server\nlisten = \n")); err == nil {
		t.Fatal("broken TOML accepted")
	}
}

func TestLoadWithoutFileUsesDefaults(t *testing.T) {
	t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "config.toml"))
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "localhost:8080" || cfg.AppURL != "" {
		t.Errorf("defaults: %q %q", cfg.ListenAddr, cfg.AppURL)
	}
}

func TestLoadWritesDefaultsAndKeepsValues(t *testing.T) {
	path := write(t, "[server]\napp_url = \"https://a.example\"\n")
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	text := string(raw)
	for _, want := range []string{`app_url = 'https://a.example'`, `listen = 'localhost:8080'`, "session_ttl = 1440", "[security]", "# Time zone."} {
		if !strings.Contains(text, want) {
			t.Errorf("rewritten file lacks %q:\n%s", want, text)
		}
	}
	// The rewritten file loads back and stays the same.
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(path)
	if string(again) != text {
		t.Error("file changed on the second load")
	}
}

func TestLoadCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config not created: %v", err)
	}
}

func TestSetAppURLPersists(t *testing.T) {
	path := write(t, "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Live.SetAppURL("https://b.example"); err != nil {
		t.Fatal(err)
	}
	if cfg.URL() != "https://b.example" {
		t.Errorf("live url %q", cfg.URL())
	}
	again, _ := Load(path)
	if again.AppURL != "https://b.example" {
		t.Errorf("not persisted: %q", again.AppURL)
	}
}

func TestWritableFlag(t *testing.T) {
	path := write(t, "[server]\napp_url = \"https://a.example\"\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Live.Writable() {
		t.Error("a normal file must be writable")
	}

	// A directory where the temp file goes makes every write fail, like a read-only mount.
	path = write(t, "[server]\napp_url = \"https://a.example\"\n")
	if err := os.Mkdir(path+".tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("a read-only config must not stop the server: %v", err)
	}
	if cfg.Live.Writable() {
		t.Error("write failed but the config is reported writable")
	}
	if err := cfg.Live.SetAppURL("https://b.example"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("SetAppURL on a read-only config: %v", err)
	}
	if cfg.URL() != "https://a.example" {
		t.Errorf("url changed despite the failure: %q", cfg.URL())
	}
}

func TestMailerDefaultsAreWrittenAndDisabled(t *testing.T) {
	path := write(t, "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mail.Enabled || cfg.MailProblem != "" {
		t.Errorf("mail must be off without a config: %+v %q", cfg.Mail, cfg.MailProblem)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{"[mailer]", "enabled = false", "protocol = 'starttls'", "port = 587", "timeout = 10"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("defaults lack %q:\n%s", want, raw)
		}
	}
}

func TestMailerLoadsAndSurvivesRewrite(t *testing.T) {
	path := write(t, `
[mailer]
enabled = true
host = "smtp.example.com"
port = 465
protocol = "smtps"
user = "savvy"
password = "s3cret"
from = "Go Savvy <savvy@example.com>"
skip_verify = true
timeout = 3
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	m := cfg.Mail
	if !m.Enabled || m.Host != "smtp.example.com" || m.Port != 465 || m.Protocol != "smtps" ||
		m.User != "savvy" || m.Password != "s3cret" || !m.SkipVerify || m.Timeout != 3*time.Second {
		t.Errorf("mail config: %+v", m)
	}
	if cfg.MailProblem != "" {
		t.Errorf("unexpected problem: %s", cfg.MailProblem)
	}
	// Saving another setting rewrites the file; the mail block must stay.
	if err := cfg.Live.SetAppURL("https://b.example"); err != nil {
		t.Fatal(err)
	}
	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Mail != m {
		t.Errorf("mail config changed by a rewrite: %+v", again.Mail)
	}
}

// A mistake in [mailer] must not stop the server: mail is switched off and the
// reason is kept for the admin panel.
func TestMailerProblemDisablesMailWithoutStoppingTheServer(t *testing.T) {
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	cfg, err := Load(write(t, "[mailer]\nenabled = true\nhost = \"\"\nfrom = \"savvy@example.com\"\n"))
	if err != nil {
		t.Fatalf("a bad mailer section stopped the server: %v", err)
	}
	if cfg.Mail.Enabled {
		t.Error("mail stays enabled with an invalid config")
	}
	if !strings.Contains(cfg.MailProblem, "mailer.host") {
		t.Errorf("problem not kept: %q", cfg.MailProblem)
	}
	if !strings.Contains(logs.String(), "mailer.host") {
		t.Errorf("problem not logged: %s", logs.String())
	}
}

func TestMailerTimeoutFallsBack(t *testing.T) {
	cfg, err := Load(write(t, "[mailer]\nenabled = true\nhost = \"h\"\nfrom = \"a@example.com\"\ntimeout = 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mail.Timeout != 10*time.Second {
		t.Errorf("timeout %v", cfg.Mail.Timeout)
	}
}

func TestLoadConfigFileEnv(t *testing.T) {
	path := write(t, "[server]\nlisten = \":4000\"\n")
	t.Setenv("CONFIG_FILE", path)
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != ":4000" {
		t.Errorf("CONFIG_FILE not used: %q", cfg.ListenAddr)
	}
}

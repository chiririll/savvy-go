package boot

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/locator"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
)

// configIn writes a config file that keeps all data under dir.
func configIn(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	body := "[paths]\ndata = \"" + filepath.ToSlash(filepath.Join(dir, "data")) + "\"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuildCreatesTheDataDirectoriesAndServices(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p, err := NewBuilder(appconfig.Build{Version: "test"}).ConfigFile(configIn(t, dir)).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close(ctx)

	data := filepath.Join(dir, "data")
	for _, sub := range []string{"", "uploads", "backups", "keys"} {
		if st, err := os.Stat(filepath.Join(data, sub)); err != nil || !st.IsDir() {
			t.Errorf("%q was not created: %v", sub, err)
		}
	}
	if _, err := os.Stat(signing.Path(data)); err != nil {
		t.Errorf("the signing key: %v", err)
	}
	if got := locator.Get[appconfig.Config](p.Loc).DataDir; got != filepath.ToSlash(data) && got != data {
		t.Errorf("config data dir %q", got)
	}
	if locator.Get[store.Store](p.Loc) == nil {
		t.Error("the store is not registered")
	}
	// A server with no users starts with a first-run password.
	if raw, err := os.ReadFile(filepath.Join(data, bootstrap.FileName)); err != nil || strings.TrimSpace(string(raw)) == "" {
		t.Errorf("bootstrap password file: %q, %v", raw, err)
	}
}

func TestRestartKeepsTheSigningKeyAndTheFirstRunPassword(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfgPath := configIn(t, dir)
	data := filepath.Join(dir, "data")

	first, err := NewBuilder(appconfig.Build{}).ConfigFile(cfgPath).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	key1, _ := os.ReadFile(signing.Path(data))
	pass1, _ := os.ReadFile(filepath.Join(data, bootstrap.FileName))
	if err := first.Close(ctx); err != nil {
		t.Fatal(err)
	}

	second, err := NewBuilder(appconfig.Build{}).ConfigFile(cfgPath).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close(ctx)
	key2, _ := os.ReadFile(signing.Path(data))
	pass2, _ := os.ReadFile(filepath.Join(data, bootstrap.FileName))
	if string(key1) != string(key2) {
		t.Error("a restart replaced the signing key")
	}
	if string(pass1) != string(pass2) {
		t.Error("a restart changed the first-run password while no user exists")
	}
}

func TestRestartRefusesAMissingSigningKey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfgPath := configIn(t, dir)
	first, err := NewBuilder(appconfig.Build{}).ConfigFile(cfgPath).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_ = first.Close(ctx)
	if err := os.Remove(signing.Path(filepath.Join(dir, "data"))); err != nil {
		t.Fatal(err)
	}
	if p, err := NewBuilder(appconfig.Build{}).ConfigFile(cfgPath).Build(ctx); err == nil {
		_ = p.Close(ctx)
		t.Error("the server started with a new key although it had signed with another")
	}
}

func TestBuildReportsAnUnreadableConfigAndSeedFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(bad, []byte("[paths\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewBuilder(appconfig.Build{}).ConfigFile(bad).Build(ctx); err == nil || !strings.Contains(err.Error(), "load config") {
		t.Errorf("broken config: %v", err)
	}
	if _, err := NewBuilder(appconfig.Build{}).ConfigFile(configIn(t, dir)).SeedFile(filepath.Join(dir, "missing.toml")).Build(ctx); err == nil || !strings.Contains(err.Error(), "seed") {
		t.Errorf("missing seed file: %v", err)
	}
}

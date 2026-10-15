package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/config"
	"savvy-go/internal/store/sqlite"
)

func openStore(t *testing.T) (*sqlite.Store, config.Config) {
	t.Helper()
	cfg := config.Config{DataDir: t.TempDir()}
	st, err := sqlite.OpenApp(context.Background(), cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, cfg
}

func TestStartBootstrapOnAnEmptyServer(t *testing.T) {
	st, cfg := openStore(t)
	g, err := startBootstrap(context.Background(), cfg, st)
	if err != nil || !g.Active() {
		t.Fatalf("guard %v, err %v", g, err)
	}
	if _, err := os.Stat(filepath.Join(cfg.DataDir, bootstrap.FileName)); err != nil {
		t.Fatal(err)
	}
	again, _ := startBootstrap(context.Background(), cfg, st)
	if again.Password() != g.Password() {
		t.Fatal("a restart must reuse the password while there are no users")
	}
}

func TestStartBootstrapSkipsAServerWithUsers(t *testing.T) {
	st, cfg := openStore(t)
	pass := "secret1"
	if _, err := (auth.Users{DB: st.Server()}).Create(context.Background(), "A", "a@a.com", &pass, auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(cfg.DataDir, bootstrap.FileName)
	if err := os.WriteFile(stale, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := startBootstrap(context.Background(), cfg, st)
	if err != nil || g != nil {
		t.Fatalf("guard %v, err %v", g, err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale file kept: %v", err)
	}
}

func TestOpenURL(t *testing.T) {
	for addr, want := range map[string]string{
		":8080":          "http://localhost:8080",
		"0.0.0.0:9000":   "http://localhost:9000",
		"127.0.0.1:8080": "http://127.0.0.1:8080",
		"garbage":        "http://localhost",
	} {
		if got := openURL(config.Config{ListenAddr: addr}); got != want {
			t.Errorf("%q: got %q, want %q", addr, got, want)
		}
	}
	if got := openURL(config.Config{AppURL: "https://savvy.example.com", ListenAddr: ":8080"}); got != "https://savvy.example.com" {
		t.Errorf("public URL ignored: %q", got)
	}
}

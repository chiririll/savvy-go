package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chiririll/savvy-plus/internal/config"
	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/migrate"
)

func testConfig(t *testing.T) (config.Config, string) {
	t.Helper()
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		AppURL:        "http://localhost:8080",
		PublicDir:     public,
		DataDir:       dir,
		Database:      filepath.Join(dir, "database.sqlite"),
		UploadsDir:    filepath.Join(dir, "uploads"),
		BackupsDir:    filepath.Join(dir, "backups"),
		SessionTTL:    24 * time.Hour,
		RememberTTL:   7 * 24 * time.Hour,
		ChallengeTTL:  5 * time.Minute,
		SessionCookie: "svy_session",
		CSRFCookie:    "svy_csrf",
		CSRFHeader:    "X-CSRF-Token",
	}
	return cfg, dir
}

func TestLivezPass(t *testing.T) {
	cfg, _ := testConfig(t)
	sqlDB, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	srv := httptest.NewServer(New(cfg, sqlDB).Handler())
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/livez")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if !strings.Contains(res.Header.Get("Content-Type"), "application/health+json") {
		t.Fatalf("content-type %q", res.Header.Get("Content-Type"))
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "pass" {
		t.Fatalf("status %v", body["status"])
	}
}

func TestReadyzFailsBeforeMigrations(t *testing.T) {
	cfg, _ := testConfig(t)
	sqlDB, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	srv := httptest.NewServer(New(cfg, sqlDB).Handler())
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d", res.StatusCode)
	}
	if !strings.Contains(res.Header.Get("Content-Type"), "application/health+json") {
		t.Fatalf("content-type %q", res.Header.Get("Content-Type"))
	}
	raw, _ := io.ReadAll(res.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "fail" {
		t.Fatalf("body %s", raw)
	}
	checks, _ := body["checks"].(map[string]any)
	mig, _ := checks["schema:migrations"].([]any)
	if len(mig) == 0 {
		t.Fatalf("missing schema:migrations: %s", raw)
	}
	entry, _ := mig[0].(map[string]any)
	if entry["status"] != "fail" {
		t.Fatalf("migration check %v", entry)
	}
}

func TestReadyzPassAfterMigrate(t *testing.T) {
	cfg, _ := testConfig(t)
	sqlDB, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := migrate.Up(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(New(cfg, sqlDB).Handler())
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d body %s", res.StatusCode, raw)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "pass" {
		t.Fatalf("status %v", body["status"])
	}
	checks, _ := body["checks"].(map[string]any)
	conn, _ := checks["sqlite:connectivity"].([]any)
	if len(conn) == 0 {
		t.Fatal("missing sqlite:connectivity")
	}
	if conn[0].(map[string]any)["status"] != "pass" {
		t.Fatalf("connectivity %v", conn[0])
	}
	mig, _ := checks["schema:migrations"].([]any)
	if mig[0].(map[string]any)["observedValue"].(float64) != 0 {
		t.Fatalf("pending %v", mig[0])
	}
}

func TestSPAServesIndexAndStatic(t *testing.T) {
	cfg, _ := testConfig(t)
	if err := os.WriteFile(filepath.Join(cfg.PublicDir, "favicon.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	srv := httptest.NewServer(New(cfg, sqlDB).Handler())
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/favicon.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("favicon %d", res.StatusCode)
	}

	htmlRes, err := http.Get(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	defer htmlRes.Body.Close()
	raw, _ := io.ReadAll(htmlRes.Body)
	if htmlRes.StatusCode != http.StatusOK {
		t.Fatalf("spa %d", htmlRes.StatusCode)
	}
	if !strings.Contains(string(raw), `id="app"`) {
		t.Fatalf("spa html: %s", raw)
	}
	if !strings.Contains(htmlRes.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("content-type %q", htmlRes.Header.Get("Content-Type"))
	}
}

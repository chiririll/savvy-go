package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/config"
	"savvy-go/internal/store"
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
	srv := httptest.NewServer(New(cfg, openTestStore(t, cfg), testKeys(t, cfg)).Handler())
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

// startingStore is a store still opening and migrating its databases.
type startingStore struct{ store.Store }

func (startingStore) Status() store.Status { return store.Status{} }

// brokenSpaceStore reports one space that failed to migrate.
type brokenSpaceStore struct{ store.Store }

func (brokenSpaceStore) Status() store.Status {
	return store.Status{Ready: true, Unavailable: map[int64]string{7: "broken migration"}}
}

func readyz(t *testing.T, st store.Store) (int, map[string]any) {
	t.Helper()
	cfg, _ := testConfig(t)
	srv := httptest.NewServer(New(cfg, st, testKeys(t, cfg)).Handler())
	t.Cleanup(srv.Close)
	res, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if !strings.Contains(res.Header.Get("Content-Type"), "application/health+json") {
		t.Fatalf("content-type %q", res.Header.Get("Content-Type"))
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, body
}

func migrationCheck(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	checks, _ := body["checks"].(map[string]any)
	mig, _ := checks["store:migrations"].([]any)
	if len(mig) == 0 {
		t.Fatalf("missing store:migrations: %v", body)
	}
	return mig[0].(map[string]any)
}

func TestReadyzFailsWhileOpening(t *testing.T) {
	cfg, _ := testConfig(t)
	status, body := readyz(t, startingStore{openTestStore(t, cfg)})
	if status != http.StatusServiceUnavailable || body["status"] != "fail" {
		t.Fatalf("status %d body %v", status, body)
	}
	if migrationCheck(t, body)["status"] != "fail" {
		t.Fatalf("migration check %v", body)
	}
}

func TestReadyzPassAfterOpen(t *testing.T) {
	cfg, _ := testConfig(t)
	status, body := readyz(t, openTestStore(t, cfg))
	if status != http.StatusOK || body["status"] != "pass" {
		t.Fatalf("status %d body %v", status, body)
	}
	checks, _ := body["checks"].(map[string]any)
	conn, _ := checks["store:connectivity"].([]any)
	if len(conn) == 0 || conn[0].(map[string]any)["status"] != "pass" {
		t.Fatalf("connectivity %v", checks)
	}
	if migrationCheck(t, body)["observedValue"].(float64) != 0 {
		t.Fatalf("unavailable spaces %v", body)
	}
}

// P15: a space that failed to migrate is a warning, not a failed probe.
func TestP15ReadyzWarnsAboutUnavailableSpace(t *testing.T) {
	cfg, _ := testConfig(t)
	status, body := readyz(t, brokenSpaceStore{openTestStore(t, cfg)})
	if status != http.StatusOK {
		t.Fatalf("status %d body %v", status, body)
	}
	if mig := migrationCheck(t, body); mig["status"] != "warn" || mig["observedValue"].(float64) != 1 {
		t.Fatalf("migration check %v", mig)
	}
}

func TestSPAServesIndexAndStatic(t *testing.T) {
	cfg, _ := testConfig(t)
	if err := os.WriteFile(filepath.Join(cfg.PublicDir, "favicon.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	index := `<html><head></head><body><div id="app"></div><i>{{.Version}}</i></body></html>`
	if err := os.WriteFile(filepath.Join(cfg.PublicDir, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(cfg, openTestStore(t, cfg), testKeys(t, cfg)).Handler())
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
	if !strings.Contains(string(raw), `id="app"`) || strings.Contains(string(raw), "{{") {
		t.Fatalf("spa html: %s", raw)
	}
	if !strings.Contains(htmlRes.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("content-type %q", htmlRes.Header.Get("Content-Type"))
	}
}

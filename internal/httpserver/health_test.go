//go:build integration

package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/config"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
)

// testURL is the public URL of the test servers.
const testURL = "http://localhost:8080"

func testConfig(t *testing.T) (appconfig.Config, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:    dir,
		UploadsDir: filepath.Join(dir, "uploads"),
		BackupsDir: filepath.Join(dir, "backups"),
		TZ:         "UTC",
		Location:   time.UTC,
	}
	return cfg, dir
}

// testFile is a config file with the given TOML.
func testFile(toml string) *config.Config {
	f, err := config.Parse(toml)
	if err != nil {
		panic(err)
	}
	return f
}

// newServer is New with the public URL set to testURL.
func newServer(cfg appconfig.Config, st store.Store, keys *signing.Holder, opts ...Option) *Server {
	return New(cfg.WithAppURL(testURL), st, keys, opts...)
}

func TestLivezPass(t *testing.T) {
	cfg, _ := testConfig(t)
	srv := httptest.NewServer(newServer(cfg, openTestStore(t, cfg), testKeys(t, cfg)).Handler())
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
	srv := httptest.NewServer(newServer(cfg, st, testKeys(t, cfg)).Handler())
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

// A space that failed to migrate is a warning, not a failed probe.
func TestReadyzWarnsAboutUnavailableSpace(t *testing.T) {
	cfg, _ := testConfig(t)
	status, body := readyz(t, brokenSpaceStore{openTestStore(t, cfg)})
	if status != http.StatusOK {
		t.Fatalf("status %d body %v", status, body)
	}
	if mig := migrationCheck(t, body); mig["status"] != "warn" || mig["observedValue"].(float64) != 1 {
		t.Fatalf("migration check %v", mig)
	}
}

//go:build integration

package httpserver

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"savvy-go/internal/config"
)

func TestSetAppURLWritesTheServerSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	file, err := config.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	server := config.Of(file, ServerSection)
	if err := file.Finish(); err != nil {
		t.Fatal(err)
	}

	if err := SetAppURL(file, "https://savvy.example.com/"); err != nil {
		t.Fatal(err)
	}
	if got := server.Get().AppURL; got != "https://savvy.example.com" {
		t.Errorf("the running value is %q", got)
	}
	saved, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(saved), `app_url = 'https://savvy.example.com'`) &&
		!strings.Contains(string(saved), `app_url = "https://savvy.example.com"`) {
		t.Errorf("not in the file: %v\n%s", err, saved)
	}
	if got := server.Get(); got.Listen != DefaultListen || got.TZ != "UTC" {
		t.Errorf("the other keys changed: %+v", got)
	}

	// Empty unsets it.
	if err := SetAppURL(file, ""); err != nil || server.Get().AppURL != "" {
		t.Errorf("unset: %v %q", err, server.Get().AppURL)
	}
}

func TestSetAppURLOnAReadOnlyFile(t *testing.T) {
	file := config.Empty()
	config.Of(file, ServerSection)
	if err := SetAppURL(file, "https://savvy.example.com"); !errors.Is(err, config.ErrReadOnly) {
		t.Fatalf("want ErrReadOnly, got %v", err)
	}
}

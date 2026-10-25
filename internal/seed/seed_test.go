package seed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPresetsAreCopies(t *testing.T) {
	DefaultCategories()[0].Name = "changed"
	DefaultTags()[0] = "changed"
	if DefaultCategories()[0].Name != "#HOUSING" || DefaultTags()[0] != "#ESSENTIAL" {
		t.Fatal("a caller changed the shared presets")
	}
}

func TestConfigRejectsBadDate(t *testing.T) {
	if _, err := (Config{Date: "15.03.2026"}).now(time.UTC); err == nil {
		t.Fatal("accepted a malformed date")
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "seed.toml")
	if err := os.WriteFile(good, []byte("date = \"2026-03-15\"\nmanifest = \"/m.json\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(good)
	if err != nil || !c.Enabled || c.Date != "2026-03-15" || c.Manifest != "/m.json" {
		t.Fatalf("got %+v, %v", c, err)
	}
	bad := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(bad, []byte("dat = \"x\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(bad); err == nil {
		t.Error("accepted an unknown key")
	}
}

func TestPresetsAreUniqueTokensOneDefaultEach(t *testing.T) {
	seen := map[string]bool{}
	defaults := map[string]int{}
	for _, c := range DefaultCategories() {
		if !strings.HasPrefix(c.Name, "#") {
			t.Fatalf("category %q is not a token", c.Name)
		}
		if seen[c.Type+c.Name] {
			t.Fatalf("category %s duplicated in %s", c.Name, c.Type)
		}
		seen[c.Type+c.Name] = true
		if c.IsDefault {
			defaults[c.Type]++
		}
	}
	if defaults["expense"] != 1 || defaults["income"] != 1 {
		t.Fatalf("defaults per type: %v", defaults)
	}
	tags := map[string]bool{}
	for _, tag := range DefaultTags() {
		if !strings.HasPrefix(tag, "#") || tags[tag] {
			t.Fatalf("tag %q is not a unique token", tag)
		}
		tags[tag] = true
	}
}

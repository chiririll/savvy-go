package seed

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"

	"savvy-go/internal/domain"
	"savvy-go/internal/store"
)

// Config says how to seed the demo data. It is read from the file named by
// the --seed-config flag (TOML); every key is optional.
type Config struct {
	// Enabled seeds on first boot; it is set when a file is given.
	Enabled bool `toml:"-"`
	// Date (YYYY-MM-DD) is the day the data is placed relative to, the current
	// day when empty. Pin it to seed the same data on any day.
	Date string `toml:"date"`
	// Manifest is a file to write what was seeded to, if any.
	Manifest string `toml:"manifest"`
}

// LoadConfig reads a seed config file; unknown keys are an error.
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read seed config: %w", err)
	}
	c := Config{Enabled: true}
	dec := toml.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return c, nil
}

// Run seeds the demo data as cfg says (see Demo) and writes the manifest.
func Run(ctx context.Context, st store.Store, keys domain.KeyRing, cfg Config, loc *time.Location) error {
	if !cfg.Enabled {
		return nil
	}
	now, err := cfg.now(loc)
	if err != nil {
		return err
	}
	manifest, err := Demo(ctx, st, keys, Options{Loc: loc, Now: now})
	if err != nil {
		return err
	}
	if manifest == nil || cfg.Manifest == "" {
		return nil
	}
	if err := manifest.WriteFile(cfg.Manifest); err != nil {
		return fmt.Errorf("write seed manifest: %w", err)
	}
	return nil
}

// now is the moment the data is seeded for: the current time when no date is
// set, otherwise noon of that day in loc.
func (c Config) now(loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	if c.Date == "" {
		return time.Now().In(loc), nil
	}
	d, err := time.ParseInLocation("2006-01-02", c.Date, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("seed date %q: want YYYY-MM-DD", c.Date)
	}
	return d.Add(12 * time.Hour), nil
}

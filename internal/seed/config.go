package seed

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"savvy-go/internal/domain"
	"savvy-go/internal/store"
)

// Config says whether and how to seed the demo data. It is read from the
// environment apart from the rest of the server's configuration.
type Config struct {
	// Enabled is SEED_DEMO: seed on first boot.
	Enabled bool
	// Date is SEED_DATE (YYYY-MM-DD): the day the data is placed relative to, the
	// current day when empty. Pin it to seed the same data on any day.
	Date string
	// Manifest is SEED_MANIFEST: a file to write what was seeded to, if any.
	Manifest string
}

// ConfigFromEnv reads SEED_DEMO, SEED_DATE and SEED_MANIFEST.
func ConfigFromEnv() Config {
	return Config{
		Enabled:  truthy(os.Getenv("SEED_DEMO")),
		Date:     strings.TrimSpace(os.Getenv("SEED_DATE")),
		Manifest: strings.TrimSpace(os.Getenv("SEED_MANIFEST")),
	}
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

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

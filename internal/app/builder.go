package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"savvy-go/internal/config"
	"savvy-go/internal/config/shared"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/runtime"
	"savvy-go/internal/seed"
	"savvy-go/internal/store/sqlite"
)

// Builder starts the application: it reads the config, opens the databases, brings them to
// a servable state and then calls New. It only holds inputs; Build runs the phases once,
// in a fixed order.
type Builder struct {
	build      Build
	configPath string
	seedPath   string
}

// NewBuilder starts a Builder for the given release.
func NewBuilder(b Build) *Builder { return &Builder{build: b} }

// ConfigFile sets the config file. Empty means $CONFIG_FILE, else config.toml in the data
// directory; it is created when missing.
func (b *Builder) ConfigFile(path string) *Builder { b.configPath = path; return b }

// SeedFile sets a TOML file (keys date, manifest, both optional) that makes the first
// boot seed the demo data.
func (b *Builder) SeedFile(path string) *Builder { b.seedPath = path; return b }

// Build runs the startup phases and returns the services. The caller closes them.
func (b *Builder) Build(ctx context.Context) (*Services, error) {
	file, err := config.Open(shared.FilePath(b.configPath))
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	cfg := shared.Load(file)
	var seedCfg seed.Config
	if b.seedPath != "" {
		if seedCfg, err = seed.LoadConfig(b.seedPath); err != nil {
			return nil, fmt.Errorf("load seed config: %w", err)
		}
	}
	for _, dir := range []string{cfg.DataDir, cfg.UploadsDir, cfg.BackupsDir} {
		if err := os.MkdirAll(dir, 0o775); err != nil {
			return nil, fmt.Errorf("create %s: %w", dir, err)
		}
	}
	st, err := sqlite.OpenApp(ctx, cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	built := false
	defer func() {
		if !built {
			_ = st.Close()
		}
	}()

	if cfg.AppURL() == "" {
		slog.Warn("the public URL is not set; SSO and passkeys stay disabled until app_url is set in the config file")
	}
	keys, ring, err := prepare(ctx, cfg, st)
	if err != nil {
		return nil, err
	}
	if err := seed.Run(ctx, st, ring, seedCfg, cfg.Location); err != nil {
		return nil, fmt.Errorf("seed demo: %w", err)
	}
	guard, err := startBootstrap(ctx, cfg, st)
	if err != nil {
		return nil, err
	}

	queue := runtime.NewQueue(2)
	rates := currencydata.New()
	svc := New(cfg, st, keys,
		WithConfig(file),
		WithBuild(b.build), WithRates(rates), WithBootstrap(guard), WithQueue(queue))
	// Every package has asked for its sections by now.
	if err := file.Finish(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	svc.rates = rates
	// Closed in this order: a queued job may still use the store.
	svc.closers = []func(context.Context) error{
		func(ctx context.Context) error { queue.Shutdown(ctx); return nil },
		func(context.Context) error { return st.Close() },
	}
	built = true
	return svc, nil
}

// Close releases what Build acquired: it drains the queue, then closes the store. Stop
// taking requests first, as a running request could still enqueue work.
func (s *Services) Close(ctx context.Context) error {
	var errs []error
	for _, c := range s.closers {
		errs = append(errs, c(ctx))
	}
	return errors.Join(errs...)
}

package boot

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"savvy-go/internal/app"
	"savvy-go/internal/appconfig"
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/httpserver"
	"savvy-go/internal/jobs"
	"savvy-go/internal/locator"
	"savvy-go/internal/seed"
	"savvy-go/internal/store/sqlite"
)

// Process is the running application: the locator that holds the services, and the rate
// source the scheduled update expires. The locator also closes what Build acquired.
type Process struct {
	Loc   *locator.Locator
	rates *currencydata.Source
}

// Builder starts the application: it reads the config, opens the databases, brings them to
// a servable state and then registers the services with app.New. It only holds inputs; Build runs the phases once,
// in a fixed order.
type Builder struct {
	build      appconfig.Build
	configPath string
	seedPath   string
}

// NewBuilder starts a Builder for the given release.
func NewBuilder(b appconfig.Build) *Builder { return &Builder{build: b} }

// ConfigFile sets the config file. Empty means $CONFIG_FILE, else config.toml in the data
// directory; it is created when missing.
func (b *Builder) ConfigFile(path string) *Builder { b.configPath = path; return b }

// SeedFile sets a TOML file (keys date, manifest, both optional) that makes the first
// boot seed the demo data.
func (b *Builder) SeedFile(path string) *Builder { b.seedPath = path; return b }

// Build runs the startup phases and returns the running process. The caller closes it.
func (b *Builder) Build(ctx context.Context) (*Process, error) {
	file, err := config.Open(appconfig.FilePath(b.configPath))
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	server := config.Of(file, httpserver.ServerSection)
	srv := server.Get()
	cfg := appconfig.Load(file).WithServer(srv.TZ, srv.Location, func() string { return server.Get().AppURL })
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
	guard, err := startBootstrap(ctx, cfg, srv.Listen, st)
	if err != nil {
		return nil, err
	}

	queue := jobs.NewQueue(2)
	rates := currencydata.New()
	loc, err := app.New(cfg, st, keys,
		app.WithConfig(file),
		app.WithBuild(b.build), app.WithRates(rates), app.WithBootstrap(guard), app.WithQueue(queue))
	if err != nil {
		return nil, fmt.Errorf("build services: %w", err)
	}
	// Closed in reverse order of registration: a queued job may still use the store.
	locator.OnClose(loc, func(context.Context) error { return st.Close() })
	locator.OnClose(loc, func(ctx context.Context) error { queue.Shutdown(ctx); return nil })
	// Every package has asked for its sections by now.
	if err := file.Finish(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	built = true
	return &Process{Loc: loc, rates: rates}, nil
}

// Close releases what Build acquired: it drains the queue, then closes the store. Stop
// taking requests first, as a running request could still enqueue work.
func (p *Process) Close(ctx context.Context) error { return p.Loc.Close(ctx) }

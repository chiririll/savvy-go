package boot

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"savvy-go/internal/app"
	"savvy-go/internal/appconfig"
	"savvy-go/internal/backup"
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/httpserver"
	"savvy-go/internal/jobs"
	"savvy-go/internal/locator"
	"savvy-go/internal/seed"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
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
	cfg := appconfig.Load(file).
		WithServer(srv.TZ, srv.Location, func() string { return server.Get().AppURL }).
		WithAppURLSetter(func(url string) error { return httpserver.SetAppURL(file, url) })
	seedCfg, err := b.loadSeed()
	if err != nil {
		return nil, err
	}
	if err := makeDirs(cfg); err != nil {
		return nil, err
	}
	st, err := openMigrated(ctx, cfg, b.build, sqlite.AppOptions(ctx, cfg.DataDir))
	if err != nil {
		return nil, err
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

// openMigrated opens the databases and migrates them. If existing data has
// migrations pending, a normal server backup is taken first.
//
// The backup needs the signing key from the data dir and an open store, so
// the files are first opened without migrators (the schema stays as on disk),
// backed up, closed, and opened again with the real migrators.
//
// A failed backup stops the start: migrating without the promised backup
// could lose data with no way back, so the schema is left untouched.
func openMigrated(
	ctx context.Context,
	cfg appconfig.Config,
	build appconfig.Build,
	opts sqlite.Options,
) (*sqlite.Store, error) {
	probe := opts
	probe.MigrateServer, probe.MigrateSpace = nil, nil
	st, err := sqlite.Open(ctx, probe)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	pending, err := st.Pending(ctx)
	if err == nil && pending != "" {
		err = backupBeforeMigration(ctx, cfg, build, st, pending)
	}
	if cerr := st.Close(); err == nil && cerr != nil {
		err = cerr
	}
	if err != nil {
		return nil, fmt.Errorf("before migrating: %w", err)
	}
	st, err = sqlite.Open(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	return st, nil
}

// backupBeforeMigration takes the server backup for the migrations named by
// pending. The note carries pending, so a start that finds its backup again
// (the migration failed and the process is restarted in a loop) reuses it
// instead of filling the disk.
func backupBeforeMigration(
	ctx context.Context,
	cfg appconfig.Config,
	build appconfig.Build,
	st *sqlite.Store,
	pending string,
) error {
	note := "Before migration " + pending
	// The settings may be of an older schema; an unreadable one counts as no key used yet.
	kid, _ := (settings.Store{DB: st.Server()}).Get(ctx, signingKeyUsedKey, "").(string)
	key, err := signing.Load(cfg.DataDir, kid != "")
	if err != nil {
		return fmt.Errorf("signing key: %w", err)
	}
	svc := backup.Service{
		Store: st, Keys: signing.KeyRing{Holder: signing.NewHolder(key)},
		Dir: cfg.BackupsDir, DataDir: cfg.DataDir, AppVersion: build.Version,
	}
	// The migration failed before and the process was restarted: the backup of
	// that attempt still holds all the data unless a file was written since.
	if prev := svc.ServerBackupWithNote(note, st.LastWrite()); prev != nil {
		slog.Info("migrations pending; reusing the backup from an earlier attempt", "file", prev.Filename)
		return nil
	}
	slog.Info("migrations pending; backing up the server first")
	entry, err := svc.CreateServer(ctx, &note)
	if err != nil {
		return fmt.Errorf("backup failed, the data is not migrated: %w", err)
	}
	slog.Info("backup before migration written", "file", entry.Filename)
	return nil
}

// loadSeed reads the seed file; without one the demo data is not seeded.
func (b *Builder) loadSeed() (seed.Config, error) {
	if b.seedPath == "" {
		return seed.Config{}, nil
	}
	cfg, err := seed.LoadConfig(b.seedPath)
	if err != nil {
		return seed.Config{}, fmt.Errorf("load seed config: %w", err)
	}
	return cfg, nil
}

func makeDirs(cfg appconfig.Config) error {
	for _, dir := range []string{cfg.DataDir, cfg.UploadsDir, cfg.BackupsDir} {
		if err := os.MkdirAll(dir, 0o775); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

// Close releases what Build acquired: it drains the queue, then closes the store. Stop
// taking requests first, as a running request could still enqueue work.
func (p *Process) Close(ctx context.Context) error { return p.Loc.Close(ctx) }

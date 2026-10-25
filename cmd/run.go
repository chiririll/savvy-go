// Package cmd is the process: it reads the flags and the config, wires the services,
// and serves HTTP and the scheduled jobs until interrupted.
package cmd

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/boot"
	"savvy-go/internal/httpserver"
)

// Main runs the application and exits non-zero when it fails. build says which release
// this binary is; the root package sets it at link time.
func Main(build appconfig.Build) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	configPath := flag.String(
		"config",
		"",
		"path to config.toml (default: $CONFIG_FILE, else config.toml in the data directory; created if missing)",
	)
	seedConfig := flag.String(
		"seed-config",
		"",
		"path to a TOML file (keys date, manifest, both optional); seeds the demo data on first boot when given",
	)
	flag.Parse()

	if err := run(build, *configPath, *seedConfig); err != nil {
		slog.Error("savvy-go", "err", err)
		os.Exit(1)
	}
}

// run serves the application until it is interrupted.
func run(build appconfig.Build, configPath, seedPath string) error {
	// Cancelled by the first interrupt: it stops the jobs and then the server.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	proc, err := boot.NewBuilder(build).ConfigFile(configPath).SeedFile(seedPath).Build(ctx)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := proc.Close(closeCtx); err != nil {
			slog.Warn("close", "err", err)
		}
	}()
	proc.Start(ctx)
	return httpserver.Serve(ctx, proc.Loc)
}

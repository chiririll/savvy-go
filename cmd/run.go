// Package cmd is the process: it reads the flags and the config, wires the services,
// and serves HTTP and the scheduled jobs until interrupted.
package cmd

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"savvy-go/internal/app"
	"savvy-go/internal/httpserver"
)

// Main runs the application and exits non-zero when it fails. build says which release
// this binary is; the root package sets it at link time.
func Main(build app.Build) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	configPath := flag.String("config", "", "path to config.toml (default: $CONFIG_FILE, else config.toml in the data directory; created if missing)")
	seedConfig := flag.String("seed-config", "", "path to a TOML file (keys date, manifest, both optional); seeds the demo data on first boot when given")
	flag.Parse()

	if err := run(build, *configPath, *seedConfig); err != nil {
		slog.Error("savvy-go", "err", err)
		os.Exit(1)
	}
}

// run serves the application until it is interrupted.
func run(build app.Build, configPath, seedPath string) error {
	// Cancelled by the first interrupt: it stops the jobs and then the server.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svc, err := app.NewBuilder(build).ConfigFile(configPath).SeedFile(seedPath).Build(ctx)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := svc.Close(closeCtx); err != nil {
			slog.Warn("close", "err", err)
		}
	}()
	svc.Start(ctx)
	return serve(ctx, svc)
}

// serve answers requests until ctx is cancelled, then drains them.
func serve(ctx context.Context, svc *app.Services) error {
	cfg := svc.Cfg
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpserver.FromServices(svc).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	failed := make(chan error, 1)
	go func() {
		slog.Info("go savvy listening", "addr", cfg.ListenAddr, "data", cfg.DataDir, "version", svc.Build.Version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			failed <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	// The deferred Close in run drains the queue after this: stop taking requests first.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

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
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/httpserver"
	"savvy-go/internal/runtime"
	"savvy-go/internal/seed"
	"savvy-go/internal/store/sqlite"
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
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	var seedCfg seed.Config
	if seedPath != "" {
		if seedCfg, err = seed.LoadConfig(seedPath); err != nil {
			return fmt.Errorf("load seed config: %w", err)
		}
	}

	// Cancelled by the first interrupt: it stops the jobs and then the server.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for _, dir := range []string{cfg.DataDir, cfg.UploadsDir, cfg.BackupsDir} {
		if err := os.MkdirAll(dir, 0o775); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	st, err := sqlite.OpenApp(ctx, cfg.DataDir)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close() // last: serve returns once nothing uses the store

	keys, ring, err := prepare(ctx, cfg, st)
	if err != nil {
		return err
	}
	if err := seed.Run(ctx, st, ring, seedCfg, cfg.Location); err != nil {
		return fmt.Errorf("seed demo: %w", err)
	}
	guard, err := startBootstrap(ctx, cfg, st)
	if err != nil {
		return err
	}

	queue := runtime.NewQueue(2)
	rates := currencydata.New()
	svc := app.New(cfg, st, keys,
		app.WithBuild(build), app.WithRates(rates), app.WithBootstrap(guard), app.WithQueue(queue))
	startJobs(ctx, svc, rates)
	return serve(ctx, svc, queue)
}

// serve answers requests until ctx is cancelled, then drains them and the queue.
func serve(ctx context.Context, svc *app.Services, queue *runtime.Queue) error {
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
	// Stop taking requests before the queue closes: a request still running could enqueue.
	err := srv.Shutdown(shutdownCtx)
	queue.Shutdown(shutdownCtx)
	if err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

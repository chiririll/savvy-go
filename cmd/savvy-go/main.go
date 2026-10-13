package main

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

	"savvy-go/internal/config"
	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver"
	"savvy-go/internal/jobs"
	"savvy-go/internal/seed"
	"savvy-go/internal/store/sqlite"
	"savvy-go/internal/version"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	configPath := flag.String("config", "", "path to config.toml (default: $CONFIG_FILE, else config.toml in the data directory; created if missing)")
	seedConfig := flag.String("seed-config", "", "path to a TOML file (keys date, manifest, both optional); seeds the demo data on first boot when given")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	var seedCfg seed.Config
	if *seedConfig != "" {
		if seedCfg, err = seed.LoadConfig(*seedConfig); err != nil {
			slog.Error("load seed config", "err", err)
			os.Exit(1)
		}
	}
	if err := run(cfg, seedCfg); err != nil {
		slog.Error("savvy-go", "err", err)
		os.Exit(1)
	}
}

// run serves the application until it is interrupted.
func run(cfg config.Config, seedCfg seed.Config) error {
	for _, dir := range []string{cfg.DataDir, cfg.UploadsDir, cfg.BackupsDir} {
		if err := os.MkdirAll(dir, 0o775); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	ctx := context.Background()
	st, err := sqlite.OpenApp(ctx, cfg.DataDir)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()
	keys, transfers, err := prepare(ctx, cfg, st)
	if err != nil {
		return err
	}
	if err := seed.Run(ctx, st, transfers.Keys, seedCfg, cfg.Location); err != nil {
		return fmt.Errorf("seed demo: %w", err)
	}

	queue := jobs.New(2)
	jobsCtx, stopJobs := context.WithCancel(ctx)
	defer stopJobs()
	startJobs(jobsCtx, st, domain.Uploads{DB: st.Server(), Root: cfg.UploadsDir})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpserver.New(cfg, st, keys).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	failed := make(chan error, 1)
	go func() {
		slog.Info("go savvy listening", "addr", cfg.ListenAddr, "data", cfg.DataDir, "version", version.Value)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			failed <- fmt.Errorf("http server: %w", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-failed:
		return err
	case <-stop:
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	stopJobs()
	queue.Shutdown(shutdownCtx)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

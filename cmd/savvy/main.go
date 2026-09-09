package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chiririll/savvy-plus/internal/config"
	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/domain"
	"github.com/chiririll/savvy-plus/internal/httpserver"
	"github.com/chiririll/savvy-plus/internal/jobs"
	"github.com/chiririll/savvy-plus/internal/legacy"
	"github.com/chiririll/savvy-plus/internal/migrate"
	"github.com/chiririll/savvy-plus/internal/schedule"
	"github.com/chiririll/savvy-plus/internal/version"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.FromEnv()
	if v := os.Getenv("APP_VERSION"); v != "" {
		version.Value = v
	}

	if err := os.MkdirAll(cfg.DataDir, 0o775); err != nil {
		slog.Error("create data dir", "err", err)
		os.Exit(1)
	}
	_ = os.MkdirAll(cfg.UploadsDir, 0o775)
	_ = os.MkdirAll(cfg.BackupsDir, 0o775)

	sqlDB, err := db.Open(cfg.Database)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	if err := legacy.EnsureColumns(ctx, sqlDB); err != nil {
		slog.Error("legacy columns", "err", err)
		os.Exit(1)
	}
	if err := migrate.Up(ctx, sqlDB); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	if err := legacy.UpgradeInPlace(ctx, sqlDB); err != nil {
		slog.Error("legacy import", "err", err)
		os.Exit(1)
	}

	queue := jobs.New(2)
	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()
	recurring := domain.RecurringStore{DB: sqlDB, Txs: domain.Transactions{DB: sqlDB}}
	schedule.New(schedule.Job{
		Name:     "recurring:ensure-upcoming",
		Interval: time.Hour,
		Run:      recurring.EnsureUpcoming,
	}).Start(schedCtx)
	_ = queue

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpserver.New(cfg, sqlDB).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("savvy listening", "addr", cfg.ListenAddr, "data", cfg.DataDir, "version", version.Value)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	schedCancel()
	queue.Shutdown(shutdownCtx)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}

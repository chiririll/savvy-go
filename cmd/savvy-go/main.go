package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"savvy-go/internal/config"
	"savvy-go/internal/db"
	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver"
	"savvy-go/internal/jobs"
	"savvy-go/internal/migrate"
	"savvy-go/internal/schedule"
	"savvy-go/internal/seed"
	"savvy-go/internal/settings"
	"savvy-go/internal/version"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.FromEnv()

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
	if err := migrate.Up(ctx, sqlDB); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	if err := migrate.Conform(ctx, sqlDB); err != nil {
		slog.Error("conform schema", "err", err)
		os.Exit(1)
	}
	if err := seed.Demo(ctx, sqlDB, cfg.SeedDemo, cfg.Location); err != nil {
		slog.Error("seed demo", "err", err)
		os.Exit(1)
	}

	queue := jobs.New(2)
	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()
	recurring := domain.RecurringStore{DB: sqlDB, Txs: domain.Transactions{DB: sqlDB}}
	uploads := domain.Uploads{DB: sqlDB, Root: cfg.UploadsDir, AppURL: cfg.AppURL, SignSecret: cfg.AppURL + "|upload"}
	currencies := domain.Currencies{DB: sqlDB}
	appSettings := settings.Store{DB: sqlDB}
	schedule.New(
		schedule.Job{Name: "currencies:update", Interval: 24 * time.Hour, Run: func(ctx context.Context) error {
			if !appSettings.Bool(ctx, "auto_update_currencies", true) {
				return nil
			}
			updated, skipped, err := currencies.UpdateRates(ctx)
			if err == nil {
				slog.Info("currency rates updated", "updated", updated, "skipped", skipped)
			}
			return err
		}},
		schedule.Job{Name: "recurring:ensure-upcoming", Interval: time.Hour, Run: recurring.EnsureUpcoming},
		schedule.Job{Name: "uploads:prune", Interval: time.Hour, Run: uploads.PruneExpired},
	).Start(schedCtx)
	_ = queue

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpserver.New(cfg, sqlDB).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("go savvy listening", "addr", cfg.ListenAddr, "data", cfg.DataDir, "version", version.Value)
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

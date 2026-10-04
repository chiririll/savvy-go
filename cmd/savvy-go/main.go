package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"savvy-go/internal/config"
	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver"
	"savvy-go/internal/jobs"
	"savvy-go/internal/schedule"
	"savvy-go/internal/seed"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
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

	ctx := context.Background()
	if _, err := os.Stat(filepath.Join(cfg.DataDir, "database.sqlite")); err == nil {
		slog.Warn("database.sqlite is the single-file layout; it is not read anymore (data now lives in server.sqlite and spaces/)")
	}
	st, err := sqlite.OpenApp(ctx, cfg.DataDir)
	if err != nil {
		slog.Error("open store", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	for id, why := range st.Status().Unavailable {
		slog.Error("space unavailable", "space", id, "reason", why)
	}

	if err := seed.Demo(ctx, st, cfg.SeedDemo, cfg.Location); err != nil {
		slog.Error("seed demo", "err", err)
		os.Exit(1)
	}

	queue := jobs.New(2)
	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()
	uploads := domain.Uploads{DB: st.Server(), Root: cfg.UploadsDir, AppURL: cfg.AppURL, SignSecret: cfg.AppURL + "|upload"}
	// eachSpace runs a job in every available space; one failing space does
	// not stop the others.
	eachSpace := func(job func(context.Context, store.DB) error) func(context.Context) error {
		return func(ctx context.Context) error {
			ids, err := st.Spaces(ctx)
			if err != nil {
				return err
			}
			var errs []error
			for _, id := range ids {
				d, err := st.Space(ctx, id)
				if err != nil {
					continue // unavailable: reported at startup
				}
				if err := job(ctx, d); err != nil {
					errs = append(errs, fmt.Errorf("space %d: %w", id, err))
				}
			}
			return errors.Join(errs...)
		}
	}
	schedule.New(
		schedule.Job{Name: "currencies:update", Interval: 24 * time.Hour, Run: eachSpace(func(ctx context.Context, d store.DB) error {
			if !(settings.Store{DB: d, Space: true}).Bool(ctx, "auto_update_currencies", true) {
				return nil
			}
			updated, skipped, err := domain.Currencies{DB: d}.UpdateRates(ctx)
			if err == nil {
				slog.Info("currency rates updated", "updated", updated, "skipped", skipped)
			}
			return err
		})},
		schedule.Job{Name: "recurring:ensure-upcoming", Interval: time.Hour, Run: eachSpace(func(ctx context.Context, d store.DB) error {
			return domain.RecurringStore{DB: d, Txs: domain.Transactions{DB: d}}.EnsureUpcoming(ctx)
		})},
		schedule.Job{Name: "uploads:prune", Interval: time.Hour, Run: uploads.PruneExpired},
	).Start(schedCtx)
	_ = queue

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpserver.New(cfg, st).Handler(),
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

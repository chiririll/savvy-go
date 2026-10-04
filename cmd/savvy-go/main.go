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

	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver"
	"savvy-go/internal/jobs"
	"savvy-go/internal/schedule"
	"savvy-go/internal/seed"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
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
	st, err := sqlite.OpenApp(ctx, cfg.DataDir, cfg.AppKey)
	if err != nil {
		slog.Error("open store", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	if err := migrateSingleFile(ctx, cfg.DataDir, st); err != nil {
		slog.Error("migrate database.sqlite", "err", err)
		os.Exit(1)
	}
	keys, err := loadSigningKey(ctx, cfg.DataDir, st)
	if err != nil {
		slog.Error("signing key", "err", err)
		os.Exit(1)
	}
	if missing, orphans, err := (domain.Spaces{Store: st}).Reconcile(ctx); err == nil {
		for _, id := range missing {
			slog.Error("space has no database file", "space", id)
		}
		for _, id := range orphans {
			slog.Warn("space database file without a registered space; left untouched", "space", id)
		}
	}
	// Finish transfers a crash left half-written and review spaces restored
	// just before it; an unavailable space is merged on a later start.
	transfers := domain.Transfers{
		Spaces: domain.Spaces{Store: st},
		Keys:   domain.KeyRing{Holder: keys, Trusted: domain.TrustedKeys(st.Server())},
	}
	if err := transfers.SyncAll(ctx); err != nil {
		slog.Warn("merge transfers between spaces", "err", err)
	}
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
		Handler:           httpserver.New(cfg, st, keys).Handler(),
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

// migrateSingleFile moves a database.sqlite of the single-file layout (or a
// Laravel-era one) into server.sqlite and space 1, once: only while the server
// has no users, and the old file is kept renamed.
func migrateSingleFile(ctx context.Context, dataDir string, st *sqlite.Store) error {
	old := filepath.Join(dataDir, "database.sqlite")
	if _, err := os.Stat(old); err != nil {
		return nil
	}
	n, err := (auth.Users{DB: st.Server()}).Count(ctx)
	if err != nil || n > 0 {
		slog.Warn("database.sqlite is ignored: the server already has data", "file", old)
		return err
	}
	p, err := st.PrepareServer(ctx, old)
	if err != nil {
		return err
	}
	if err := st.ReplaceServer(ctx, p); err != nil {
		return err
	}
	slog.Info("database.sqlite split into server.sqlite and spaces/")
	return os.Rename(old, old+".migrated")
}

// signingKeyUsedKey records the key id once the server has a key, so a key
// file that disappears later is noticed instead of silently replaced.
const signingKeyUsedKey = "signing_kid"

func loadSigningKey(ctx context.Context, dataDir string, st *sqlite.Store) (*signing.Holder, error) {
	server := settings.Store{DB: st.Server()}
	kid, _ := server.Get(ctx, signingKeyUsedKey, "").(string)
	key, err := signing.Load(dataDir, kid != "")
	if err != nil {
		return nil, err
	}
	if kid == "" {
		if err := server.Set(ctx, signingKeyUsedKey, key.KID()); err != nil {
			return nil, err
		}
	} else if kid != key.KID() {
		slog.Warn("the signing key file is not the key this server used before", "recorded", kid, "file", key.KID())
	}
	return signing.NewHolder(key), nil
}

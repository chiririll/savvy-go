package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"savvy-go/internal/domain"
	"savvy-go/internal/schedule"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
)

// startJobs runs the periodic jobs until ctx is cancelled.
func startJobs(ctx context.Context, st *sqlite.Store, uploads domain.Uploads) {
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
	).Start(ctx)
}

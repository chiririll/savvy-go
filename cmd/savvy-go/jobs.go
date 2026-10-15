package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"savvy-go/internal/currencydata"
	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
	"savvy-go/internal/schedule"
	"savvy-go/internal/settings"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
	"savvy-go/internal/uploads"
)

// startJobs runs the periodic jobs until ctx is cancelled.
func startJobs(ctx context.Context, st *sqlite.Store, uploads uploads.Service, rates *currencydata.Source) {
	spaceSvc := spaces.Service{Store: st}
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
		schedule.Job{Name: "currencies:update", Interval: 24 * time.Hour, Run: func(ctx context.Context) error {
			// Rates are cached by base currency in the shared source, so spaces
			// with the same base share one request. Expiring them here makes
			// each base fetch once per run, and only if some space asks.
			rates.Expire()
			return eachSpace(func(ctx context.Context, d store.DB) error {
				if !(settings.Store{DB: d, Space: true}).Bool(ctx, "auto_update_currencies", true) {
					return nil
				}
				updated, skipped, err := ledger.Currencies{DB: d, Source: rates}.UpdateRates(ctx)
				if err == nil {
					slog.Info("currency rates updated", "updated", updated, "skipped", skipped)
				}
				return err
			})(ctx)
		}},
		schedule.Job{Name: "recurring:ensure-upcoming", Interval: time.Hour, Run: eachSpace(func(ctx context.Context, d store.DB) error {
			return recurring.Service{DB: d, Txs: ledger.Transactions{DB: d}}.EnsureUpcoming(ctx)
		})},
		schedule.Job{Name: "uploads:prune", Interval: time.Hour, Run: uploads.PruneExpired},
		// Expired invitations are kept an hour past their expiry: that is the
		// cooldown on inviting the same address again.
		schedule.Job{Name: "invitations:prune", Interval: time.Hour, Run: func(ctx context.Context) error {
			_, err := spaceSvc.PruneInvitations(ctx)
			return err
		}},
	).Start(ctx)
}

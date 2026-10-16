package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
	"savvy-go/internal/runtime"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

// Start runs the periodic jobs until ctx is cancelled.
func (svc *Services) Start(ctx context.Context) {
	st := svc.Store
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
	runtime.NewScheduler(
		runtime.Job{Name: "currencies:update", Interval: 24 * time.Hour, Run: func(ctx context.Context) error {
			// Rates are cached by base currency in the shared source, so spaces
			// with the same base share one request. Expiring them here makes
			// each base fetch once per run, and only if some space asks.
			if svc.rates != nil {
				svc.rates.Expire()
			}
			return eachSpace(func(ctx context.Context, d store.DB) error {
				if !(settings.Store{DB: d, Space: true}).Bool(ctx, "auto_update_currencies", true) {
					return nil
				}
				updated, skipped, err := ledger.Currencies{DB: d, Source: svc.Rates}.UpdateRates(ctx)
				if err == nil {
					slog.Info("currency rates updated", "updated", updated, "skipped", skipped)
				}
				return err
			})(ctx)
		}},
		runtime.Job{Name: "recurring:ensure-upcoming", Interval: time.Hour, Run: eachSpace(func(ctx context.Context, d store.DB) error {
			return recurring.Service{DB: d, Txs: ledger.Transactions{DB: d}}.EnsureUpcoming(ctx)
		})},
		runtime.Job{Name: "uploads:prune", Interval: time.Hour, Run: svc.Uploads.PruneExpired},
		// Expired invitations are kept an hour past their expiry: that is the
		// cooldown on inviting the same address again.
		runtime.Job{Name: "invitations:prune", Interval: time.Hour, Run: func(ctx context.Context) error {
			_, err := svc.Spaces.PruneInvitations(ctx)
			return err
		}},
	).Start(ctx)
}

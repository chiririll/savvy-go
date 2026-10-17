// Package boot is the composition root: it reads the config, opens the databases, brings
// them to a servable state, builds the services and runs the scheduled jobs. It is the one
// place that knows how the store keeps its files.
package boot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"savvy-go/internal/jobs"
	"savvy-go/internal/ledger"
	"savvy-go/internal/locator"
	"savvy-go/internal/recurring"
	"savvy-go/internal/settings"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/uploads"
)

// Start runs the periodic jobs until ctx is cancelled.
func (p *Process) Start(ctx context.Context) {
	l := p.Loc
	st := locator.Get[store.Store](l)
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
	jobs.NewScheduler(
		jobs.Job{Name: "currencies:update", Interval: 24 * time.Hour, Run: func(ctx context.Context) error {
			// Rates are cached by base currency in the shared source, so spaces
			// with the same base share one request. Expiring them here makes
			// each base fetch once per run, and only if some space asks.
			if p.rates != nil {
				p.rates.Expire()
			}
			return eachSpace(func(ctx context.Context, d store.DB) error {
				if !(settings.Store{DB: d, Space: true}).Bool(ctx, "auto_update_currencies", true) {
					return nil
				}
				updated, skipped, err := ledger.Currencies{DB: d, Source: locator.Get[ledger.RateSource](l)}.UpdateRates(ctx)
				if err == nil {
					slog.Info("currency rates updated", "updated", updated, "skipped", skipped)
				}
				return err
			})(ctx)
		}},
		jobs.Job{Name: "recurring:ensure-upcoming", Interval: time.Hour, Run: eachSpace(func(ctx context.Context, d store.DB) error {
			return recurring.Service{DB: d, Txs: ledger.Transactions{DB: d}}.EnsureUpcoming(ctx)
		})},
		jobs.Job{Name: "uploads:prune", Interval: time.Hour, Run: locator.Get[uploads.Service](l).PruneExpired},
		// Expired invitations are kept an hour past their expiry: that is the
		// cooldown on inviting the same address again.
		jobs.Job{Name: "invitations:prune", Interval: time.Hour, Run: func(ctx context.Context) error {
			_, err := locator.Get[spaces.Service](l).PruneInvitations(ctx)
			return err
		}},
	).Start(ctx)
}

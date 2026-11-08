package seed

import (
	"fmt"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

// reconcileKeys are the accounts that get reconciliation points, in the order
// of a batch. Debts, space accounts and the credit card are left out.
var reconcileKeys = []string{"checking", "savings", "cash", "eur"}

// reconcileGapText is how much short the checking account is at the second point.
const reconcileGapText = "46.50"

// reconcile pins the Demo space's accounts. It runs after the transfers across
// the links, which add payments on past days: a point made before them would
// show their amounts as a difference.
func (d *demo) reconcile() error {
	home := d.space(spaceDemo)
	db, err := d.st.Space(d.ctx, home.ID)
	if err != nil {
		return err
	}
	s := &seeder{
		ctx: d.ctx, db: db, now: d.now, end: d.now, accts: ledger.Accounts{DB: db},
		start: startOfMonth(d.now.AddDate(0, -d.months, 0)),
	}
	return s.seedReconciliations(home.accounts)
}

// seedReconciliations pins the main accounts on the last day of each of the
// three previous months, oldest first: the first month matches the ledger, the
// second leaves checking short by a round amount, the third matches again
// against the pinned balance, so the gap does not repeat. It uses no random
// numbers, so the rest of the seeded data is not touched.
func (s *seeder) seedReconciliations(accounts map[string]*ledger.Account) error {
	// Every queued transaction must be stored before a balance is read.
	if err := s.flush(); err != nil {
		return err
	}
	gap := decimal.RequireFromString(reconcileGapText)
	rec := ledger.Reconciliations{DB: s.db}
	for back := 3; back >= 1; back-- {
		day := startOfMonth(s.now).AddDate(0, -back+1, -1)
		if !s.inRange(day) || day.After(s.now) {
			continue
		}
		date := day.Format("2006-01-02")
		var items []ledger.ReconcileItem
		for _, key := range reconcileKeys {
			a := accounts[key]
			if a == nil {
				continue
			}
			bal, err := s.accts.Balance(s.ctx, *a, date)
			if err != nil {
				return fmt.Errorf("balance of %s on %s: %w", key, date, err)
			}
			item := ledger.ReconcileItem{AccountID: a.ID, Actual: bal.Decimal()}
			if back == 2 && key == "checking" {
				item.Actual = item.Actual.Sub(gap)
				item.Note = strPtr("Counted cash / bank statement")
			}
			items = append(items, item)
		}
		if len(items) == 0 {
			continue
		}
		if _, err := rec.Reconcile(s.ctx, date, items, s.now); err != nil {
			return fmt.Errorf("reconcile %s: %w", date, err)
		}
	}
	return nil
}

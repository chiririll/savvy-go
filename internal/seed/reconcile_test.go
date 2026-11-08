package seed

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
	"savvy-go/internal/signing"
	"savvy-go/internal/store/sqlite/sqlitetest"
)

// demoPoints seeds the demo on a pinned day and returns the reconciliation
// points of its Demo space, with its accounts by name.
func demoPoints(t *testing.T) ([]ledger.ReconciliationView, map[int64]ledger.Account, ledger.Accounts) {
	t.Helper()
	ctx := context.Background()
	st := sqlitetest.Open(t)
	key, err := signing.Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	manifest, err := Demo(ctx, st, signing.KeyRing{Holder: signing.NewHolder(key)}, Options{Now: now})
	if err != nil || manifest == nil {
		t.Fatalf("Demo = %v, %v", manifest, err)
	}
	for _, sp := range manifest.Spaces {
		if sp.Name != "Demo" {
			continue
		}
		db, err := st.Space(ctx, sp.ID)
		if err != nil {
			t.Fatal(err)
		}
		views, err := (ledger.Reconciliations{DB: db}).List(ctx, ledger.ReconciliationFilter{})
		if err != nil {
			t.Fatal(err)
		}
		accts := ledger.Accounts{DB: db}
		all, err := accts.All(ctx, false, false)
		if err != nil {
			t.Fatal(err)
		}
		byID := map[int64]ledger.Account{}
		for _, a := range all {
			byID[a.ID] = a
		}
		return views, byID, accts
	}
	t.Fatal("no Demo space")
	return nil, nil, ledger.Accounts{}
}

func TestDemoHasReconciliationPoints(t *testing.T) {
	views, byID, accts := demoPoints(t)
	if len(views) != 12 {
		t.Fatalf("points = %d, want 3 dates x 4 accounts", len(views))
	}
	today := "2026-03-15"
	gaps := 0
	for _, v := range views {
		if v.Date > today {
			t.Errorf("point in the future: %s", v.Date)
		}
		name := byID[v.AccountID].Name
		if v.Date == "2025-12-31" || v.Date == "2026-02-28" {
			if !v.Difference.Decimal().IsZero() {
				t.Errorf("%s on %s differs by %s", name, v.Date, v.Difference.Decimal())
			}
			continue
		}
		if v.Date != "2026-01-31" {
			t.Fatalf("unexpected date %s", v.Date)
		}
		if name == "Chase Checking" {
			gaps++
			if !v.Difference.Decimal().Equal(decimal.RequireFromString("-46.50")) || v.Note == nil {
				t.Errorf("gap point = %s, note %v", v.Difference.Decimal(), v.Note)
			}
			// The pin holds: the balance on that day is the counted one.
			bal, err := accts.Balance(context.Background(), byID[v.AccountID], v.Date)
			if err != nil || !bal.Decimal().Equal(v.Actual.Decimal()) {
				t.Errorf("balance on the gap day = %v (%v), want %s", bal, err, v.Actual.Decimal())
			}
		} else if !v.Difference.Decimal().IsZero() {
			t.Errorf("%s on %s differs by %s", name, v.Date, v.Difference.Decimal())
		}
	}
	if gaps != 1 {
		t.Errorf("gap points = %d, want 1", gaps)
	}
}

func TestReconciliationsAreDeterministic(t *testing.T) {
	a, _, _ := demoPoints(t)
	b, _, _ := demoPoints(t)
	if len(a) != len(b) {
		t.Fatalf("%d points, then %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Date != b[i].Date || a[i].Actual.Minor() != b[i].Actual.Minor() ||
			a[i].Difference.Minor() != b[i].Difference.Minor() {
			t.Errorf("point %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

package seed

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/filter"
	"savvy-go/internal/ledger"
	"savvy-go/internal/signing"
	"savvy-go/internal/store/sqlite/sqlitetest"
)

// The demo shows both kinds of split payment, and they keep every balance whole.
func TestDemoHasSplitPayments(t *testing.T) {
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
	var demoID int64
	for _, sp := range manifest.Spaces {
		if sp.Name == "Demo" {
			demoID = sp.ID
		}
	}
	db, err := st.Space(ctx, demoID)
	if err != nil {
		t.Fatal(err)
	}
	all, _, err := (ledger.Transactions{DB: db}).Filtered(ctx, filter.TxFilter{}, 1, 10000)
	if err != nil {
		t.Fatal(err)
	}
	legs := map[string][]ledger.Transaction{}
	for _, tx := range all {
		if tx.GroupUUID != nil {
			legs[*tx.GroupUUID] = append(legs[*tx.GroupUUID], tx)
		}
	}
	if len(legs) != 2 {
		t.Fatalf("%d split payments, want 2", len(legs))
	}
	for id, parts := range legs {
		if len(parts) != 2 {
			t.Errorf("payment %s has %d parts, want 2", id, len(parts))
		}
	}
}

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

// The demo shows an expense paid from two accounts, one of them with change in another currency.
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
	var split int
	for _, tx := range all {
		if tx.Type == "expense" && len(tx.Payments) == 2 && len(tx.Lines) == 1 {
			split++
		}
	}
	if split != 2 {
		t.Fatalf("%d single-line expenses paid from two accounts, want 2", split)
	}
}

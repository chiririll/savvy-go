package seed

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/ledger"
	"savvy-go/internal/signing"
	"savvy-go/internal/store/sqlite/sqlitetest"
)

// The demo's credit card is a card with terms and a status to show.
func TestDemoHasACreditCardWithAStatus(t *testing.T) {
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
	cards := ledger.CreditCards{Accounts: ledger.Accounts{DB: db}}
	list, err := cards.All(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "Amex Gold Card" || list[0].Type != "credit" {
		t.Fatalf("cards = %+v", list)
	}
	card := list[0]
	state, err := cards.State(ctx, card, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("balance %s limit %s statement %+v status %+v", card.Balance, card.Terms.Limit, state.Statement, state.Status)
	if state.Status.Level == ledger.StatusOK {
		t.Errorf("the demo card shows no status: %+v", state)
	}
}

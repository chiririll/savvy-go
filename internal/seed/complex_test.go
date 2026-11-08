package seed

import (
	"context"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/db/filter"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/signing"
	"savvy-go/internal/store/sqlite/sqlitetest"
)

// demoTransactions seeds the demo on a pinned day and returns every transaction
// of the space called name.
func demoTransactions(t *testing.T, name string) []ledger.Transaction {
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
		if sp.Name != name {
			continue
		}
		db, err := st.Space(ctx, sp.ID)
		if err != nil {
			t.Fatal(err)
		}
		all, _, err := (ledger.Transactions{DB: db}).Filtered(ctx, filter.TxFilter{}, 1, 10000)
		if err != nil {
			t.Fatal(err)
		}
		return all
	}
	t.Fatalf("no space %q", name)
	return nil
}

func categoriesOf(tx ledger.Transaction) int {
	seen := map[int64]bool{}
	for _, l := range tx.Lines {
		if l.CategoryID != nil {
			seen[*l.CategoryID] = true
		}
	}
	return len(seen)
}

func currenciesOf(tx ledger.Transaction) int {
	seen := map[money.Unit]bool{}
	for _, p := range tx.Payments {
		seen[p.Amount.Unit()] = true
	}
	return len(seen)
}

func hasNote(tx ledger.Transaction, part string) bool {
	for _, l := range tx.Lines {
		if l.Note != nil && strings.Contains(*l.Note, part) {
			return true
		}
	}
	return false
}

func TestDemoShowsTheNewTransactionModel(t *testing.T) {
	all := demoTransactions(t, "Demo")
	var basket, threePay, twoCur, bullets, tasks, link, pendingList, fee, splitIncome int
	for _, tx := range all {
		if categoriesOf(tx) >= 3 {
			basket++
		}
		if tx.Type == "expense" && len(tx.Payments) >= 3 {
			threePay++
		}
		if currenciesOf(tx) >= 2 && tx.Type == "expense" {
			twoCur++
		}
		if hasNote(tx, "\n- ") {
			bullets++
		}
		if hasNote(tx, "- [x]") && hasNote(tx, "- [ ]") {
			tasks++
		}
		if hasNote(tx, "](https://") {
			link++
		}
		if tx.Status == "pending" && hasNote(tx, "- [ ]") && len(tx.Lines) >= 2 {
			pendingList++
		}
		if tx.Type == "transfer" && len(tx.Lines) == 1 {
			fee++
		}
		if tx.Type == "income" && len(tx.Payments) >= 2 {
			splitIncome++
		}
	}
	for name, n := range map[string]int{
		"basket over three categories": basket,
		"expense paid from 3 accounts": threePay,
		"expense in two currencies":    twoCur,
		"markdown product list":        bullets,
		"task list with done and open": tasks,
		"markdown link":                link,
		"pending shopping list":        pendingList,
		"transfer with a fee":          fee,
		"income into several accounts": splitIncome,
	} {
		if n == 0 {
			t.Errorf("the demo has no %s", name)
		}
	}
}

func TestDemoIsTheSameOnEveryRun(t *testing.T) {
	fingerprint := func() []string {
		var out []string
		for _, tx := range demoTransactions(t, "Demo") {
			d, date := "", ""
			if tx.Description != nil {
				d = *tx.Description
			}
			if tx.Date != nil {
				date = *tx.Date
			}
			out = append(out, date+"|"+d)
		}
		return out
	}
	a, b := fingerprint(), fingerprint()
	if len(a) != len(b) {
		t.Fatalf("%d vs %d transactions", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("row %d: %q vs %q", i, a[i], b[i])
		}
	}
}

func TestSmallSpacesUseLinesAndPayments(t *testing.T) {
	family := demoTransactions(t, "Family Budget")
	var lines, list int
	for _, tx := range family {
		if len(tx.Lines) >= 2 {
			lines++
		}
		if tx.Status == "pending" && hasNote(tx, "- [ ]") {
			list++
		}
	}
	if lines == 0 || list == 0 {
		t.Errorf("Family Budget: %d transactions with lines, %d shopping lists", lines, list)
	}
	var multiLine, multiPay int
	for _, tx := range demoTransactions(t, "Studio GmbH") {
		if len(tx.Lines) >= 2 {
			multiLine++
		}
		if tx.Type == "expense" && currenciesOf(tx) >= 2 {
			multiPay++
		}
	}
	if multiLine == 0 || multiPay == 0 {
		t.Errorf("Studio GmbH: %d with several lines, %d paid in two currencies", multiLine, multiPay)
	}
}

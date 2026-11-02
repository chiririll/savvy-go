package ledger_test

import (
	"slices"
	"testing"
	"time"

	"savvy-go/internal/db/filter"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

// described stores a confirmed expense with a description and a line for each
// of the given ones, priced in the account's currency. Without lines the
// expense has a single uncategorised one for the whole amount.
func (f *fixture) described(
	t *testing.T,
	account, minor int64,
	desc string,
	lines ...ledgerrepo.LineFields,
) int64 {
	t.Helper()
	acc, _ := f.repo.Account(ctx, account)
	unit := acc.Currency.Unit()
	date := "2026-01-02"
	now := time.Now().UTC()
	id, err := f.repo.InsertTransaction(ctx, ledgerrepo.TxFields{
		Type: "expense", Status: "confirmed", Date: &date, Description: &desc,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.ReplacePayments(ctx, id, []ledgerrepo.PaymentFields{
		{AccountID: account, Amount: money.New(-minor, unit)},
	}); err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 {
		lines = []ledgerrepo.LineFields{{CategoryID: &f.food, Amount: money.New(minor, unit)}}
	}
	if err := f.repo.ReplaceLines(ctx, id, lines); err != nil {
		t.Fatal(err)
	}
	return id
}

// line is a line of a described expense with a note, priced in the account's
// currency.
func (f *fixture) line(t *testing.T, account int64, note string, minor int64) ledgerrepo.LineFields {
	t.Helper()
	acc, _ := f.repo.Account(ctx, account)
	return ledgerrepo.LineFields{CategoryID: &f.food, Amount: money.New(minor, acc.Currency.Unit()), Note: &note}
}

// found is the ids of the transactions a search returns, sorted, and checks
// the count agrees with the page.
func (f *fixture) found(t *testing.T, q string) []int64 {
	t.Helper()
	rows, total, err := f.repo.Filtered(ctx, filter.TxFilter{Search: q}, 50, 0)
	if err != nil {
		t.Fatalf("search %q: %v", q, err)
	}
	if total != len(rows) {
		t.Fatalf("search %q: total %d, page %d", q, total, len(rows))
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	slices.Sort(ids)
	return ids
}

func want(ids ...int64) []int64 {
	slices.Sort(ids)
	if ids == nil {
		return []int64{}
	}
	return ids
}

func TestSearchMatchesDescriptionAndLineNotesIgnoringCase(t *testing.T) {
	f := newFixture(t)
	coffee := f.described(t, f.cash, 350, "Morning Coffee")
	chair := f.described(t, f.cash, 90000, "Кресло в гостиную", f.line(t, f.cash, "Armchair", 90000))
	bread := f.described(t, f.cash, 100, "Bread")

	for q, ids := range map[string][]int64{
		"coffee":    want(coffee),
		"COFFEE":    want(coffee),
		"armchair":  want(chair),
		"ARMCHAIR":  want(chair),
		"кресло":    want(chair),
		"КРЕСЛО":    want(chair),
		"гостин":    want(chair),
		"nothing":   want(),
		"":          want(coffee, chair, bread),
		"   ":       want(coffee, chair, bread),
		"coffee ar": want(),
	} {
		if got := f.found(t, q); !slices.Equal(got, ids) {
			t.Errorf("search %q = %v, want %v", q, got, ids)
		}
	}
}

func TestSearchTermsAreAllRequired(t *testing.T) {
	f := newFixture(t)
	both := f.described(t, f.cash, 500, "Cafe latte", f.line(t, f.cash, "Croissant", 200))
	f.described(t, f.cash, 300, "Cafe espresso")
	if got := f.found(t, "cafe croissant"); !slices.Equal(got, want(both)) {
		t.Fatalf("got %v, want %v", got, want(both))
	}
	if got := f.found(t, "croissant CAFE"); !slices.Equal(got, want(both)) {
		t.Fatalf("order must not matter: got %v", got)
	}
}

func TestSearchTreatsWildcardsAsText(t *testing.T) {
	f := newFixture(t)
	pct := f.described(t, f.cash, 100, "Sale 50% off")
	f.described(t, f.cash, 100, "Plain")
	for q, ids := range map[string][]int64{"%": want(pct), "50%": want(pct), "_": want(), "'; DROP TABLE transactions; --": want()} {
		if got := f.found(t, q); !slices.Equal(got, ids) {
			t.Errorf("search %q = %v, want %v", q, got, ids)
		}
	}
}

// An amount matches in the currency of its account: 1879 is 187900 minor
// units of a two-decimal currency but 1879 of a zero-decimal one.
func TestSearchAmountFollowsTheAccountCurrency(t *testing.T) {
	f := newFixture(t)
	// Descriptions carry no digits, or they would match as text.
	usd := f.described(t, f.cash, 187900, "dollars")
	yen := f.described(t, f.yen, 1879, "yen")
	cents := f.described(t, f.cash, 1879, "cents")
	f.described(t, f.yen, 18790, "ten times yen")
	f.described(t, f.cash, 187950, "dollars and change")
	half := f.found(t, "1879.50")
	if len(half) != 1 {
		t.Fatalf("1879.50 = %v", half)
	}

	for q, ids := range map[string][]int64{
		"1879":    want(usd, yen),
		"1 879":   want(usd, yen),
		"1879.00": want(usd, yen),
		"1,879":   want(usd, yen),
		"18.79":   want(cents),
		"18,79":   want(cents),
		"1879.5":  half,
		"1879.55": want(),
	} {
		if got := f.found(t, q); !slices.Equal(got, ids) {
			t.Errorf("search %q = %v, want %v", q, got, ids)
		}
	}
}

func TestSearchAmountGroupsOnlyWholeThousands(t *testing.T) {
	f := newFixture(t)
	f.described(t, f.yen, 1, "a")
	f.described(t, f.yen, 879, "b")
	both := f.described(t, f.yen, 1879, "c")
	// "1 879" is the number 1879, not the two terms "1" and "879", which no
	// single transaction satisfies.
	if got := f.found(t, "1 879"); !slices.Equal(got, want(both)) {
		t.Fatalf("1 879 = %v, want %v", got, want(both))
	}
	// A group shorter than three digits is a separate term; nothing is 87.
	if got := f.found(t, "1 87"); !slices.Equal(got, want()) {
		t.Fatalf("1 87 = %v", got)
	}
}

func TestSearchAmountMatchesLineAmountsAndTransferAmounts(t *testing.T) {
	f := newFixture(t)
	receipt := f.described(t, f.cash, 500000, "Furniture store", f.line(t, f.cash, "Sofa", 187900), f.line(t, f.cash, "Lamp", 12100))
	f.described(t, f.cash, 4200, "Other", f.line(t, f.cash, "Pen", 4200))

	if got := f.found(t, "1879"); !slices.Equal(got, want(receipt)) {
		t.Fatalf("line amount: got %v, want %v", got, want(receipt))
	}
	if got := f.found(t, "sofa 1879"); !slices.Equal(got, want(receipt)) {
		t.Fatalf("name and price: got %v", got)
	}
	if got := f.found(t, "lamp 1879"); !slices.Equal(got, want(receipt)) {
		t.Fatalf("different lines of one receipt: got %v", got)
	}

	// A transfer's received amount is in the other account's currency.
	tr := f.transfer(t, 12, 1879, "2026-01-03")
	if got := f.found(t, "1879"); !slices.Contains(got, tr) {
		t.Fatalf("received amount: got %v, want it to include %d", got, tr)
	}
}

func TestFilterByCurrencyAndAccounts(t *testing.T) {
	f := newFixture(t)
	usd := f.described(t, f.cash, 100, "usd only")
	yen := f.described(t, f.yen, 100, "yen only")
	tr := f.transfer(t, 100, 150, "2026-01-03")

	list := func(fl filter.TxFilter) []int64 {
		t.Helper()
		rows, total, err := f.repo.Filtered(ctx, fl, 50, 0)
		if err != nil || total != len(rows) {
			t.Fatalf("%+v: total %d, rows %d, %v", fl, total, len(rows), err)
		}
		ids := make([]int64, len(rows))
		for i, r := range rows {
			ids[i] = r.ID
		}
		slices.Sort(ids)
		return ids
	}
	for name, c := range map[string]struct {
		f    filter.TxFilter
		want []int64
	}{
		"usd":           {filter.TxFilter{Currencies: []string{"USD"}}, want(usd, tr)},
		"jpy":           {filter.TxFilter{Currencies: []string{"JPY"}}, want(yen, tr)},
		"lower case":    {filter.TxFilter{Currencies: []string{"jpy"}}, want(yen, tr)},
		"either":        {filter.TxFilter{Currencies: []string{"USD", "JPY"}}, want(usd, yen, tr)},
		"unknown":       {filter.TxFilter{Currencies: []string{"EUR"}}, want()},
		"account ids":   {filter.TxFilter{AccountIDs: []int64{f.yen}}, want(yen, tr)},
		"two accounts":  {filter.TxFilter{AccountIDs: []int64{f.cash, f.yen}}, want(usd, yen, tr)},
		"both filters":  {filter.TxFilter{AccountIDs: []int64{f.cash}, Currencies: []string{"JPY"}}, want(tr)},
		"with a search": {filter.TxFilter{Currencies: []string{"USD"}, Search: "only"}, want(usd)},
	} {
		if got := list(c.f); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", name, got, c.want)
		}
	}
}

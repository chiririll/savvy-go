package ledger

import (
	"slices"
	"testing"
)

func dated(in TxInput, date string) TxInput {
	in.Date = &date
	return in
}

func (e *moneyEnv) addOn(t *testing.T, in TxInput) int64 {
	t.Helper()
	tx, err := e.txs.Create(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	return tx.ID
}

func ids(ms []SimilarMatch) []int64 {
	out := make([]int64, len(ms))
	for i, m := range ms {
		out[i] = m.Transaction.ID
	}
	return out
}

func (e *moneyEnv) similar(t *testing.T, q SimilarQuery, opt SimilarOptions) []SimilarMatch {
	t.Helper()
	out, err := e.txs.FindSimilar(e.ctx, []SimilarQuery{q}, opt)
	if err != nil || len(out) != 1 {
		t.Fatalf("FindSimilar = %v, %v", out, err)
	}
	return out[0]
}

// Amount and date decide, in the currency asked for, whatever the sign.
func TestFindSimilarMatchesAmountAndDay(t *testing.T) {
	e := newMoneyEnv(t)
	usd := e.addOn(t, dated(simple("expense", e.usdAcc.ID, "10"), "2024-02-01"))
	e.addOn(t, dated(simple("expense", e.jpyAcc.ID, "10"), "2024-02-01")) // 10 JPY is not 10 USD
	e.addOn(t, dated(simple("expense", e.usdAcc.ID, "10.01"), "2024-02-01"))
	e.addOn(t, dated(simple("expense", e.usdAcc.ID, "10"), "2024-02-09"))

	q := SimilarQuery{Date: "2024-02-02", Amount: dec("10"), Currency: "usd"}
	got := e.similar(t, q, SimilarOptions{ToleranceDays: 1})
	if !slices.Equal(ids(got), []int64{usd}) {
		t.Fatalf("matches = %v, want only transaction %d", ids(got), usd)
	}
	if !slices.Contains(got[0].Reasons, MatchAmount) || !slices.Contains(got[0].Reasons, MatchDate) {
		t.Fatalf("reasons = %v", got[0].Reasons)
	}
	if got = e.similar(t, q, SimilarOptions{ToleranceDays: 0}); len(got) != 0 {
		t.Fatalf("a day apart matched with no tolerance: %v", ids(got))
	}
	// An income of 10 is the same money: the sign of the payment does not matter.
	if got = e.similar(t, SimilarQuery{Date: "2024-02-01", Amount: dec("-10"), Currency: "USD"}, SimilarOptions{}); len(got) != 1 {
		t.Fatalf("negative amount: %v", ids(got))
	}
}

// An amount finer than the currency keeps cannot equal a stored one.
func TestFindSimilarDoesNotRoundAmounts(t *testing.T) {
	e := newMoneyEnv(t)
	e.addOn(t, dated(simple("expense", e.usdAcc.ID, "10.01"), "2024-02-01"))
	got := e.similar(t, SimilarQuery{Date: "2024-02-01", Amount: dec("10.005"), Currency: "USD"}, SimilarOptions{})
	if len(got) != 0 {
		t.Fatalf("10.005 USD matched 10.01: %v", ids(got))
	}
}

func TestFindSimilarCanBeLimitedToAnAccount(t *testing.T) {
	e := newMoneyEnv(t)
	other, err := (Accounts{DB: e.db}).Create(e.ctx, AccountInput{
		Name: "Second", Type: "cash", CurrencyID: e.usd.ID, InitialBalance: dec("0"), IsActive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	mine := e.addOn(t, dated(simple("expense", e.usdAcc.ID, "5"), "2024-02-01"))
	e.addOn(t, dated(simple("expense", other.ID, "5"), "2024-02-01"))
	got := e.similar(t, SimilarQuery{Date: "2024-02-01", Amount: dec("5"), AccountID: e.usdAcc.ID}, SimilarOptions{})
	if !slices.Equal(ids(got), []int64{mine}) {
		t.Fatalf("matches = %v, want %d; the account gives the currency", ids(got), mine)
	}
	if len(e.similar(t, SimilarQuery{Date: "2024-02-01", Amount: dec("5"), Currency: "USD"}, SimilarOptions{})) != 2 {
		t.Fatal("without an account both match")
	}
}

// The external id finds the transaction whatever its amount and date, and ranks first.
func TestFindSimilarExternalIDComesFirst(t *testing.T) {
	e := newMoneyEnv(t)
	byAmount := e.addOn(t, dated(simple("expense", e.usdAcc.ID, "10"), "2024-02-01"))
	byKey := e.addOn(t, withExt(dated(simple("expense", e.usdAcc.ID, "99"), "2024-03-15"), "R-1"))
	id := "R-1"
	got := e.similar(t, SimilarQuery{Date: "2024-02-01", Amount: dec("10"), Currency: "USD", ExternalID: &id}, SimilarOptions{})
	if !slices.Equal(ids(got), []int64{byKey, byAmount}) {
		t.Fatalf("matches = %v, want %d then %d", ids(got), byKey, byAmount)
	}
	if !slices.Contains(got[0].Reasons, MatchExternalID) {
		t.Fatalf("reasons = %v", got[0].Reasons)
	}
}

// A currency is needed to read the amount: no account, no code, nothing to compare.
func TestFindSimilarNeedsACurrency(t *testing.T) {
	e := newMoneyEnv(t)
	_, err := e.txs.FindSimilar(e.ctx, []SimilarQuery{{Date: "2024-02-01", Amount: dec("1")}}, SimilarOptions{})
	if err == nil {
		t.Fatal("a query with no currency was accepted")
	}
	_, err = e.txs.FindSimilar(e.ctx, []SimilarQuery{{Date: "2024-02-01", Amount: dec("1"), Currency: "XXX"}}, SimilarOptions{})
	if err == nil {
		t.Fatal("an unknown currency was accepted")
	}
}

// The closer the date and the same description, the higher; the cap applies per item.
func TestFindSimilarOrdersAndCaps(t *testing.T) {
	e := newMoneyEnv(t)
	desc := "Coffee   Shop"
	far := e.addOn(t, dated(simple("expense", e.usdAcc.ID, "4"), "2024-02-02"))
	near := e.addOn(t, dated(simple("expense", e.usdAcc.ID, "4"), "2024-02-01"))
	same := dated(simple("expense", e.usdAcc.ID, "4"), "2024-02-02")
	same.Description = &desc
	named := e.addOn(t, same)
	want := "coffee shop"
	q := SimilarQuery{Date: "2024-02-01", Amount: dec("4"), Currency: "USD", Description: &want}
	got := e.similar(t, q, SimilarOptions{ToleranceDays: 1})
	if !slices.Equal(ids(got), []int64{named, near, far}) {
		t.Fatalf("order = %v, want %d, %d, %d", ids(got), named, near, far)
	}
	if got = e.similar(t, q, SimilarOptions{ToleranceDays: 1, MaxPerItem: 2}); len(got) != 2 {
		t.Fatalf("cap: %d matches", len(got))
	}
}

// Many items are answered in order, each on its own.
func TestFindSimilarAnswersEachItemInOrder(t *testing.T) {
	e := newMoneyEnv(t)
	a := e.addOn(t, dated(simple("expense", e.usdAcc.ID, "1"), "2024-02-01"))
	out, err := e.txs.FindSimilar(e.ctx, []SimilarQuery{
		{Key: "x", Date: "2024-02-05", Amount: dec("1"), Currency: "USD"},
		{Key: "y", Date: "2024-02-01", Amount: dec("1"), Currency: "USD"},
	}, SimilarOptions{})
	if err != nil || len(out) != 2 || len(out[0]) != 0 || !slices.Equal(ids(out[1]), []int64{a}) {
		t.Fatalf("out = %v, %v", out, err)
	}
}

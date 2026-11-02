package ledger

import (
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/filter"
)

// A page loads the payments, lines and tags of all its transactions in one
// query each; every row must still land on its own transaction, in its currency.
func TestPageHydrationKeepsPaymentsLinesTags(t *testing.T) {
	e := newMoneyEnv(t)
	tags := Tags{DB: e.db}
	food, err := tags.Create(e.ctx, "food")
	if err != nil {
		t.Fatal(err)
	}
	trip, err := tags.Create(e.ctx, "trip")
	if err != nil {
		t.Fatal(err)
	}
	date := "2024-01-15"
	mk := func(acct int64, amount string, notes []string, tagIDs []int64) int64 {
		in := simple("expense", acct, "0")
		paid := dec(amount).Neg()
		in.Payments[0].Amount = &paid
		in.Lines = nil
		for _, n := range notes {
			in.Lines = append(in.Lines, LineInput{Amount: dec(amount).Div(decimal.NewFromInt(int64(len(notes)))), Note: &n})
		}
		if len(notes) == 0 {
			in.Lines = []LineInput{{Amount: dec(amount)}}
		}
		in.Date, in.TagIDs = &date, tagIDs
		tx, err := e.txs.Create(e.ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		return tx.ID
	}
	usdTx := mk(e.usdAcc.ID, "4", []string{"a", "b"}, []int64{food.ID, trip.ID})
	jpyTx := mk(e.jpyAcc.ID, "500", []string{"c"}, []int64{trip.ID})
	bare := mk(e.usdAcc.ID, "1", nil, nil)

	page, total, err := e.txs.Filtered(e.ctx, filter.TxFilter{}, 1, 25)
	if err != nil || total != 3 {
		t.Fatalf("page: %d rows, %v", total, err)
	}
	byID := map[int64]Transaction{}
	for _, tx := range page {
		byID[tx.ID] = tx
	}

	if got := byID[usdTx]; len(got.Lines) != 2 || *got.Lines[0].Note != "a" || *got.Lines[1].Note != "b" ||
		got.Lines[1].Seq != 1 || len(got.Tags) != 2 || len(got.Payments) != 1 {
		t.Fatalf("usd tx lines %+v tags %+v", got.Lines, got.Tags)
	}
	if got := byID[jpyTx]; len(got.Lines) != 1 || *got.Lines[0].Note != "c" || len(got.Tags) != 1 ||
		got.Tags[0].ID != trip.ID {
		t.Fatalf("jpy tx lines %+v tags %+v", got.Lines, got.Tags)
	}
	if got := byID[jpyTx].Lines[0].Amount; got.Unit() != e.jpy.Unit() || !got.Decimal().Equal(dec("500")) {
		t.Fatalf("jpy line = %s %+v, want 500 JPY", got, got.Unit())
	}
	if got := byID[jpyTx].Payments[0]; got.Account == nil || got.Account.ID != e.jpyAcc.ID || !got.Amount.Decimal().Equal(dec("-500")) {
		t.Fatalf("jpy payment = %+v, want -500 on the yen account", got)
	}
	if got := byID[bare]; got.Lines == nil || got.Tags == nil || len(got.Tags) != 0 || got.Reasons != nil {
		t.Fatalf("bare tx must have empty, non-nil tags and no reasons: %+v", got)
	}
	// Transactions of the same account share one loaded account.
	if byID[usdTx].Payments[0].Account != byID[bare].Payments[0].Account {
		t.Fatal("expected the account to be loaded once and shared")
	}
}

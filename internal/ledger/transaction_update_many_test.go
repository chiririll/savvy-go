package ledger

import (
	"errors"
	"strings"
	"testing"

	"savvy-go/internal/audit"
)

func desc(s string) TxInput { return TxInput{Description: &s} }

// Each update changes only what it sends; the rest of the transaction, the
// exact amounts included, stays as it was.
func TestUpdateManyKeepsWhatIsLeftOut(t *testing.T) {
	e := newMoneyEnv(t)
	a, _ := e.txs.Create(e.ctx, e.dated("expense", e.usdAcc.ID, "12.34"))
	b, _ := e.txs.Create(e.ctx, e.dated("expense", e.usdAcc.ID, "5"))
	newAmount := e.dated("expense", e.usdAcc.ID, "7.50")

	out, err := e.txs.UpdateMany(e.ctx, []TxUpdate{{a.ID, desc("coffee")}, {b.ID, TxInput{Payments: newAmount.Payments, Lines: newAmount.Lines}}})
	if err != nil || len(out) != 2 {
		t.Fatalf("UpdateMany = %v, %v", out, err)
	}
	if out[0].ID != a.ID || out[1].ID != b.ID {
		t.Fatalf("answer out of order: %d, %d", out[0].ID, out[1].ID)
	}
	if out[0].Description == nil || *out[0].Description != "coffee" || !out[0].Payments[0].Amount.Equal(a.Payments[0].Amount) {
		t.Fatalf("first = %+v, want a new description and the old money", out[0])
	}
	if !out[1].Payments[0].Amount.Decimal().Equal(dec("-7.50")) || out[1].Date == nil || *out[1].Date != *b.Date {
		t.Fatalf("second = %+v, want -7.50 and the old date", out[1])
	}
}

// One bad update stops the call with its position, and nothing changes.
func TestUpdateManyIsAllOrNone(t *testing.T) {
	e := newMoneyEnv(t)
	a, _ := e.txs.Create(e.ctx, e.dated("expense", e.usdAcc.ID, "1"))
	b, _ := e.txs.Create(e.ctx, e.dated("expense", e.usdAcc.ID, "2"))
	amt := b.Payments[0].Amount.Decimal()
	bad := TxInput{Payments: []PaymentInput{{AccountID: 99999, Amount: &amt}}}

	_, err := e.txs.UpdateMany(e.ctx, []TxUpdate{{a.ID, desc("changed")}, {b.ID, bad}})
	if err == nil || !strings.Contains(err.Error(), "transaction 2") {
		t.Fatalf("err = %v, want one naming transaction 2", err)
	}
	if got, _ := e.txs.ByID(e.ctx, a.ID); got.Description != nil && *got.Description == "changed" {
		t.Fatal("the valid update was stored although another failed")
	}
}

func TestUpdateManyRefusesUnknownAndRepeatedIDs(t *testing.T) {
	e := newMoneyEnv(t)
	a, _ := e.txs.Create(e.ctx, e.dated("expense", e.usdAcc.ID, "1"))

	if _, err := e.txs.UpdateMany(e.ctx, []TxUpdate{{a.ID, desc("x")}, {424242, desc("y")}}); !errors.Is(err, ErrUnknownTransaction) {
		t.Errorf("unknown id: err = %v, want ErrUnknownTransaction", err)
	}
	if _, err := e.txs.UpdateMany(e.ctx, []TxUpdate{{a.ID, desc("x")}, {a.ID, desc("y")}}); err == nil ||
		!strings.Contains(err.Error(), "twice") {
		t.Errorf("repeated id: err = %v, want a refusal", err)
	}
}

// The audit rows of one call share a batch, and one rollback puts all back.
func TestUpdateManyIsOneBatchInTheLog(t *testing.T) {
	e := newMoneyEnv(t)
	a, _ := e.txs.Create(e.ctx, e.dated("expense", e.usdAcc.ID, "1"))
	b, _ := e.txs.Create(e.ctx, e.dated("expense", e.usdAcc.ID, "2"))

	if _, err := e.txs.UpdateMany(e.agent(7), []TxUpdate{{a.ID, desc("one")}, {b.ID, desc("two")}}); err != nil {
		t.Fatal(err)
	}
	rows := e.entries(t, audit.Filter{TokenID: 7})
	if len(rows) != 2 || rows[0].Op != audit.OpUpdate || rows[0].BatchID == "" || rows[0].BatchID != rows[1].BatchID {
		t.Fatalf("rows = %+v, want two updates sharing a batch", rows)
	}
	if _, err := e.rollbacks().UndoBatch(e.agent(7), rows[0].BatchID, 7); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{a.ID, b.ID} {
		if got, _ := e.txs.ByID(e.ctx, id); got.Description != nil && *got.Description != "" {
			t.Errorf("transaction %d kept %q after the rollback", id, *got.Description)
		}
	}
}

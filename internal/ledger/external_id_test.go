package ledger

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func withExt(in TxInput, ext string) TxInput { return withSrc(in, SourceAPI, ext) }

func withSrc(in TxInput, source, ext string) TxInput {
	date := "2024-02-01"
	in.Date = &date
	in.ExternalSource = &source
	in.ExternalID = &ext
	return in
}

// A receipt number is stored with the transaction and comes back with it.
func TestExternalIDRoundTrips(t *testing.T) {
	e := newMoneyEnv(t)
	tx, err := e.txs.Create(e.ctx, withExt(simple("expense", e.usdAcc.ID, "10"), "  F6HDVMNQ-5105 "))
	if err != nil {
		t.Fatal(err)
	}
	if tx.ExternalID == nil || *tx.ExternalID != "F6HDVMNQ-5105" || tx.ExternalSource == nil || *tx.ExternalSource != SourceAPI {
		t.Fatalf("external key = %v %v, want api and a trimmed id", tx.ExternalSource, tx.ExternalID)
	}
}

// The same id from two sources is two keys: the second is stored, and the
// answer says where else the id is.
func TestOtherSourceSameIDStoredWithWarning(t *testing.T) {
	e := newMoneyEnv(t)
	first, err := e.txs.Create(e.ctx, withSrc(simple("expense", e.usdAcc.ID, "1"), SourceAPI, "42"))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.ExternalMatches) != 0 {
		t.Fatalf("matches = %+v for an id nothing else has", first.ExternalMatches)
	}
	second, err := e.txs.Create(e.ctx, withSrc(simple("expense", e.usdAcc.ID, "1"), SourceMCP, "42"))
	if err != nil {
		t.Fatalf("another source with the same id was refused: %v", err)
	}
	want := []ExternalMatch{{Source: SourceAPI, TransactionID: first.ID}}
	if !slices.Equal(second.ExternalMatches, want) {
		t.Fatalf("matches = %+v, want %+v", second.ExternalMatches, want)
	}
	if got, _ := e.txs.ByID(e.ctx, second.ID); len(got.ExternalMatches) != 0 {
		t.Fatal("matches are the answer to a create, not part of the stored transaction")
	}
}

// The bulk answer carries the same warnings, each for its own item.
func TestOtherSourceSameIDInBulk(t *testing.T) {
	e := newMoneyEnv(t)
	first, _ := e.txs.Create(e.ctx, withSrc(simple("expense", e.usdAcc.ID, "1"), SourceAPI, "42"))
	out, err := e.txs.CreateMany(e.ctx, []TxInput{
		withSrc(simple("expense", e.usdAcc.ID, "1"), SourceMCP, "7"),
		withSrc(simple("expense", e.usdAcc.ID, "1"), SourceMCP, "42"),
	})
	if err != nil || len(out) != 2 {
		t.Fatalf("CreateMany = %v, %v", out, err)
	}
	if len(out[0].ExternalMatches) != 0 || len(out[1].ExternalMatches) != 1 || out[1].ExternalMatches[0].TransactionID != first.ID {
		t.Fatalf("matches = %+v, %+v", out[0].ExternalMatches, out[1].ExternalMatches)
	}
}

func TestExternalIDIsValidated(t *testing.T) {
	e := newMoneyEnv(t)
	for name, c := range map[string]struct{ source, id string }{
		"empty id":          {SourceAPI, "   "},
		"id too long":       {SourceAPI, strings.Repeat("x", 256)},
		"importer's source": {"csv", "abc"},
		"made up source":    {"receipts", "abc"},
		"no source":         {"", "abc"},
	} {
		if _, err := e.txs.Create(e.ctx, withSrc(simple("expense", e.usdAcc.ID, "1"), c.source, c.id)); err == nil {
			t.Errorf("%s: accepted %q %q", name, c.source, c.id)
		}
	}
	// An id without a source is the caller's mistake: the server always sets it.
	id := "42"
	in := simple("expense", e.usdAcc.ID, "1")
	in.ExternalID = &id
	if _, err := e.txs.Create(e.ctx, in); err == nil {
		t.Error("an id without a source was accepted")
	}
}

// The key is written at creation and never changed, so an update that sends
// one is refused rather than ignored.
func TestExternalKeyCannotBeSentOnUpdate(t *testing.T) {
	e := newMoneyEnv(t)
	tx, err := e.txs.Create(e.ctx, withExt(simple("expense", e.usdAcc.ID, "10"), "R-1"))
	if err != nil {
		t.Fatal(err)
	}
	desc := "renamed"
	if _, err := e.txs.Update(e.ctx, tx.ID, withExt(TxInput{Type: "expense", Description: &desc}, "R-2")); err == nil {
		t.Fatal("update accepted an external id")
	}
	if got, _ := e.txs.ByID(e.ctx, tx.ID); got.ExternalID == nil || *got.ExternalID != "R-1" {
		t.Fatalf("external id = %v after a refused update", got.ExternalID)
	}
}

// A template makes many transactions, so it cannot carry a key every one of
// them would collide on.
func TestTemplateRefusesAnExternalKey(t *testing.T) {
	e := newMoneyEnv(t)
	ok := fmt.Sprintf(transferTemplate, e.usdAcc.ID, e.jpyAcc.ID)
	if err := (Templates{DB: e.db}).Validate(e.ctx, []byte(ok)); err != nil {
		t.Fatalf("control template refused: %v", err)
	}
	withKey := ok[:len(ok)-1] + `,"external_id":"1"}`
	if err := (Templates{DB: e.db}).Validate(e.ctx, []byte(withKey)); !errors.Is(err, ErrTemplate) {
		t.Fatalf("template with an external id: err = %v, want ErrTemplate", err)
	}
}

package transfers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/filter"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	repo "savvy-go/internal/db/repo/transfers"
)

// mirror is the transaction of one side of a transfer in a space, with the
// ids of its real and its space account; nil when the side has none.
func (e *transferEnv) mirror(t *testing.T, spaceID int64, uuid string) *ledgerrepo.Tx {
	t.Helper()
	d, _ := e.st.Space(context.Background(), spaceID)
	list, _, err := (ledgerrepo.Repo{DB: d}).Filtered(context.Background(), filter.TxFilter{}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range list {
		for _, p := range x.Payments {
			if p.SpaceTransferUUID != nil && *p.SpaceTransferUUID == uuid {
				return &x
			}
		}
	}
	return nil
}

func amountOn(tx *ledgerrepo.Tx, account int64) (int64, bool) {
	for _, p := range tx.Payments {
		if p.AccountID == account {
			return p.Amount.Minor(), true
		}
	}
	return 0, false
}

// Each space gets a transaction with the money on its real account and the
// opposite on the space account of its currency; the space payment carries the
// uuid and stays on the same payment when the transfer changes.
func TestEachSideHasSpacePayment(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")

	out, in := e.mirror(t, e.A.ID, tr.UUID), e.mirror(t, e.B.ID, tr.UUID)
	if out == nil || in == nil || out.Type != "transfer_out" || in.Type != "transfer_in" ||
		len(out.Payments) != 2 || len(in.Payments) != 2 {
		t.Fatalf("mirrors = %+v / %+v", out, in)
	}
	if n, _ := amountOn(out, e.accA); n != -1000 {
		t.Fatalf("source account moved %d, want -1000", n)
	}
	if n, _ := amountOn(in, e.accB); n != 1000 {
		t.Fatalf("destination account moved %d, want 1000", n)
	}
	dA, _ := e.st.Space(ctx, e.A.ID)
	space, _ := (ledgerrepo.Repo{DB: dA}).SpaceAccountID(ctx, 1)
	if space == nil {
		t.Fatal("no space account in the source space")
	}
	if n, ok := amountOn(out, *space); !ok || n != 1000 {
		t.Fatalf("space account moved %d (%v), want the opposite, 1000", n, ok)
	}

	if _, err := e.tr.Update(ctx, e.owner, e.A.ID, tr.UUID,
		Update{FromAmount: ptr(decimal.NewFromInt(25)), ToAmount: ptr(decimal.NewFromInt(25))}); err != nil {
		t.Fatal(err)
	}
	after := e.mirror(t, e.A.ID, tr.UUID)
	if after == nil || after.ID != out.ID {
		t.Fatalf("the transaction was replaced: %+v", after)
	}
	if n, _ := amountOn(after, e.accA); n != -2500 {
		t.Fatalf("source account moved %d after the update, want -2500", n)
	}

	// Deleting leaves a tombstone and no transaction.
	if err := e.tr.Delete(ctx, e.owner, e.A.ID, tr.UUID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{e.A.ID, e.B.ID} {
		if e.mirror(t, id, tr.UUID) != nil {
			t.Fatalf("space %d kept the transaction of a deleted transfer", id)
		}
		if e.records(t, id)[tr.UUID].DeletedAt == nil {
			t.Fatalf("space %d lost the tombstone", id)
		}
	}
}

// What is signed is what is stored, byte for byte: an edit to the stored
// payload fails verification, and a record rebuilt from its fields is not
// what is checked.
func TestSignatureIsCheckedOverTheStoredBytes(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")
	dA, _ := e.st.Space(ctx, e.A.ID)

	rec, _ := (repo.Space{DB: dA}).Record(ctx, tr.UUID)
	if rec.Payload == nil || *rec.Payload != string(tr.signed) {
		t.Fatalf("stored payload %v is not the signed bytes %s", rec.Payload, tr.signed)
	}
	if !e.ok(e.records(t, e.A.ID)[tr.UUID]) {
		t.Fatal("a stored record does not verify")
	}

	// The same transfer with one amount changed in the stored JSON, the signature untouched.
	if _, err := dA.ExecContext(ctx,
		`UPDATE space_transfers SET payload = replace(payload, '"amount":1000', '"amount":9000') WHERE uuid = ?`,
		tr.UUID); err != nil {
		t.Fatal(err)
	}
	if got := e.records(t, e.A.ID)[tr.UUID]; got.From.Amount != 9000 || e.ok(got) {
		t.Fatalf("an edited payload verifies: %+v", got)
	}
}

func TestServiceAccountIsNotTransferSide(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	e.send(t, "10")
	dA, _ := e.st.Space(ctx, e.A.ID)
	space, _ := (ledgerrepo.Repo{DB: dA}).SpaceAccountID(ctx, 1)

	_, err := e.tr.Create(ctx, e.owner, Input{
		FromSpace: e.A.ID, FromAccount: *space, FromAmount: decimal.NewFromInt(1),
		ToSpace: e.B.ID, ToAccount: e.accB, ToAmount: decimal.NewFromInt(1), Date: "2026-10-02",
	})
	if !errors.Is(err, ErrServiceAccount) {
		t.Fatalf("a transfer from a space account = %v, want ErrServiceAccount", err)
	}
}

// A confirmed transfer whose account is gone has no transaction; the list says
// it needs attention.
func TestTransferWithoutItsAccountNeedsAttention(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")
	dA, _ := e.st.Space(ctx, e.A.ID)

	missing := *tr
	missing.From.AccountID = 999
	missing.Version = 2
	e.sign(t, &missing)
	if err := writeRecord(ctx, dA, e.A.UUID, missing, "confirmed", nil, nil); err != nil {
		t.Fatal(err)
	}
	if e.mirror(t, e.A.ID, tr.UUID) != nil {
		t.Fatal("a transaction is left for a transfer with no account")
	}
	views, err := e.tr.List(ctx, e.A.ID)
	if err != nil || len(views) != 1 || !views[0].NeedsAttention || views[0].Status != transferConfirmed {
		t.Fatalf("view = %+v, %v", views, err)
	}
}

// markRestored flags a space as restored from a backup, so the next merge
// reviews it instead of copying into it.
func (e *transferEnv) markRestored(t *testing.T, spaceID int64) {
	t.Helper()
	d, _ := e.st.Space(context.Background(), spaceID)
	if _, err := d.ExecContext(context.Background(),
		`INSERT INTO space_settings (key, value) VALUES (?, ?)`, repo.ReviewKey, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
}

// A restored space lacking a transfer holds the other space's record as the
// remote alone, with its transaction pending; accepting moves the remote into
// the record and confirms the transaction.
func TestReviewOfRemoteOnlyTransfer(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")
	dB, _ := e.st.Space(ctx, e.B.ID)
	if err := (repo.Space{DB: dB}).Delete(ctx, tr.UUID); err != nil { // B as it was in an older backup
		t.Fatal(err)
	}
	e.markRestored(t, e.B.ID)
	if err := e.tr.SyncAll(ctx); err != nil {
		t.Fatal(err)
	}

	rec, _ := (repo.Space{DB: dB}).Record(ctx, tr.UUID)
	if rec == nil || rec.Payload != nil || rec.Remote == nil || rec.Status != transferInReview ||
		rec.Source != e.A.UUID || rec.Destination != e.B.UUID || rec.Version != 1 {
		t.Fatalf("record = %+v", rec)
	}
	got := e.records(t, e.B.ID)[tr.UUID]
	if got.Review == nil || *got.Review != reviewCreatedRemote || got.From != tr.From || got.Remote != nil ||
		!e.ok(got) {
		t.Fatalf("transfer = %+v", got)
	}
	if m := e.mirror(t, e.B.ID, tr.UUID); m == nil || m.Status != "pending" {
		t.Fatalf("mirror = %+v, want it pending", m)
	}

	if err := e.tr.Resolve(ctx, e.owner, e.B.ID, tr.UUID, true); err != nil {
		t.Fatal(err)
	}
	rec, _ = (repo.Space{DB: dB}).Record(ctx, tr.UUID)
	if rec.Payload == nil || rec.Remote != nil || rec.Status != transferConfirmed || rec.SourceSig == nil {
		t.Fatalf("record after accepting = %+v", rec)
	}
	if m := e.mirror(t, e.B.ID, tr.UUID); m == nil || m.Status != "confirmed" {
		t.Fatalf("mirror after accepting = %+v", m)
	}
}

// The other space is ahead: the restored one keeps its record, holds the other
// version as the remote, and its kind is a change. Accepting takes it.
func TestReviewOfAChangedTransfer(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")
	dA, _ := e.st.Space(ctx, e.A.ID)

	newer := *tr
	newer.Version, newer.From.Amount, newer.To.Amount = 2, 4000, 4000
	e.sign(t, &newer)
	if err := writeRecord(ctx, dA, e.A.UUID, newer, "confirmed", nil, nil); err != nil {
		t.Fatal(err)
	}
	e.markRestored(t, e.B.ID)
	if err := e.tr.SyncAll(ctx); err != nil {
		t.Fatal(err)
	}

	got := e.records(t, e.B.ID)[tr.UUID]
	if got.Review == nil || *got.Review != reviewChangedRemote || got.Version != 1 || got.Remote == nil ||
		got.Remote.Version != 2 {
		t.Fatalf("transfer = %+v", got)
	}
	if n, _ := amountOn(e.mirror(t, e.B.ID, tr.UUID), e.accB); n != 1000 {
		t.Fatalf("the mirror moved %d, want it untouched until decided", n)
	}

	if err := e.tr.Resolve(ctx, e.owner, e.B.ID, tr.UUID, true); err != nil {
		t.Fatal(err)
	}
	if got := e.records(t, e.B.ID)[tr.UUID]; got.Version != 2 || got.Review != nil || got.Status != transferConfirmed {
		t.Fatalf("transfer after accepting = %+v", got)
	}
	if n, _ := amountOn(e.mirror(t, e.B.ID, tr.UUID), e.accB); n != 4000 {
		t.Fatalf("the mirror moved %d after accepting, want 4000", n)
	}
}

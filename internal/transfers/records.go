package transfers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	repo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/store"
)

// transferFromRecord reads a stored record. Its signed bytes are kept as they
// are, so a signature is checked over what was signed, not over a rebuild.
//
// A record in review is read as this space's version, and the kind of review
// follows from which versions exist: no payload means only the other space has
// the transfer (the remote becomes the record itself), no remote means only
// this space has it, a live payload against a deleted remote is a deletion
// there, and anything else is a change.
func transferFromRecord(r repo.Record) Transfer {
	var t Transfer
	var remote *Transfer
	if r.Remote != nil {
		var x Transfer
		if json.Unmarshal([]byte(*r.Remote), &x) == nil {
			remote = &x
		}
	}
	if r.Payload != nil {
		var p transferPayloadV1
		if json.Unmarshal([]byte(*r.Payload), &p) == nil {
			t = Transfer{
				UUID: p.UUID, From: p.From, To: p.To, Date: p.Date, Description: p.Description,
				CreatedBy: p.CreatedBy, Version: p.Version, DeletedAt: p.DeletedAt,
			}
		}
		t.signed = []byte(*r.Payload)
	} else if remote != nil {
		t, remote = *remote, nil
	}
	t.UUID = r.UUID
	t.SigV = r.SigV
	if r.SourceKID != nil {
		t.SignerKID = *r.SourceKID
	}
	if r.SourceSig != nil {
		t.Signature = *r.SourceSig
	}
	if r.DestinationKID != nil {
		t.DestKID = *r.DestinationKID
	}
	if r.DestSig != nil {
		t.DestSignature = *r.DestSig
	}
	t.Status = r.Status
	if r.Status != transferInReview {
		return t
	}
	kind := reviewChangedRemote
	switch {
	case r.Payload == nil:
		kind = reviewCreatedRemote
	case r.Remote == nil:
		kind = reviewMissingRemote
	case remote != nil && remote.DeletedAt != nil && t.DeletedAt == nil:
		kind = reviewDeletedRemote
	}
	t.Review, t.Remote = &kind, remote
	return t
}

// recordOf is the stored form of t. A transfer that only the other space has
// (review created_remote) is stored as the remote alone.
func recordOf(t Transfer, review *string, remote *Transfer) repo.Record {
	rec := repo.Record{UUID: t.UUID, SigV: t.SigV, Status: transferConfirmed}
	if review != nil {
		rec.Status = transferInReview
	}
	if review != nil && *review == reviewCreatedRemote {
		remote = &t
	} else {
		p, _ := t.payload(t.SigV)
		rec.Payload = ptr(string(p))
		rec.SourceKID, rec.SourceSig = optional(t.SignerKID), optional(t.Signature)
		rec.DestinationKID, rec.DestSig = optional(t.DestKID), optional(t.DestSignature)
	}
	if remote != nil {
		b, _ := json.Marshal(remote)
		rec.Remote = ptr(string(b))
	}
	return rec
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func loadTransfers(ctx context.Context, d store.DB) (map[string]Transfer, error) {
	rows, err := repo.Space{DB: d}.Records(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Transfer, len(rows))
	for _, r := range rows {
		out[r.UUID] = transferFromRecord(r)
	}
	return out, nil
}

func loadTransfer(ctx context.Context, d store.DB, id string) (*Transfer, error) {
	r, err := repo.Space{DB: d}.Record(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrGone
	}
	t := transferFromRecord(*r)
	return &t, nil
}

// writeRecord stores t in the space whose uuid is own: the record and the
// transaction of own's side, with localStatus ("confirmed" or "pending"). A
// tombstone removes the transaction; an account missing here, or one that
// cannot be a side of a transfer, leaves the record without a transaction (it
// needs attention). d must be the handle of the transaction the caller runs in.
func writeRecord(
	ctx context.Context,
	d store.DB,
	own string,
	t Transfer,
	localStatus string,
	review *string,
	remote *Transfer,
) error {
	sp := repo.Space{DB: d}
	side, sender, ok := t.side(own)
	if !ok {
		return fmt.Errorf("transfer %s does not involve this space", t.UUID)
	}
	hasAccount := false
	if t.DeletedAt == nil {
		acc, err := sp.AccountCurrency(ctx, side.AccountID)
		if err != nil {
			return err
		}
		hasAccount = acc != nil && !acc.IsService
	}
	at := time.Now().UTC()
	if err := sp.Save(ctx, recordOf(t, review, remote), at); err != nil {
		return err
	}
	if t.DeletedAt != nil || !hasAccount {
		return sp.DeleteMirror(ctx, t.UUID)
	}
	m := repo.Mirror{
		UUID: t.UUID, Type: "transfer_in", Date: t.Date, Status: localStatus, AccountID: side.AccountID,
		Amount: side.Amount, Description: t.Description, CreatedBy: t.CreatedBy,
	}
	if sender {
		m.Type, m.Amount = "transfer_out", -side.Amount
	}
	return sp.SaveMirror(ctx, m, at)
}

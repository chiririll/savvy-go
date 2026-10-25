package transfers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	repo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/store"
)

func sideFromRecord(r repo.Side) Side { return Side(r) }

func transferFromRecord(r repo.Record) Transfer {
	t := Transfer{
		UUID: r.UUID, From: sideFromRecord(r.From), To: sideFromRecord(r.To),
		Date: r.Date, Description: r.Description, CreatedBy: r.CreatedBy, Version: r.Version, DeletedAt: r.DeletedAt,
		SigV: r.SigV, SignerKID: r.SignerKID, Signature: r.Signature, Status: r.Status, Review: r.Review,
	}
	if r.Remote != "" {
		var remote Transfer
		if json.Unmarshal([]byte(r.Remote), &remote) == nil {
			t.Remote = &remote
		}
	}
	return t
}

func recordOf(t Transfer, status string, review *string, remote *Transfer) repo.Record {
	rec := repo.Record{
		UUID: t.UUID, From: repo.Side(t.From), To: repo.Side(t.To),
		Date: t.Date, Description: t.Description, CreatedBy: t.CreatedBy, Version: t.Version, DeletedAt: t.DeletedAt,
		SigV: t.SigV, SignerKID: t.SignerKID, Signature: t.Signature, Status: status, Review: review,
	}
	if remote != nil {
		b, _ := json.Marshal(remote)
		rec.Remote = string(b)
	}
	return rec
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
// tombstone removes the transaction; an account missing here leaves the
// record needing attention without a transaction. d must be the handle of
// the transaction the caller runs in.
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
	status := transferOK
	if review != nil {
		status = transferPending
	}
	hasAccount := false
	if t.DeletedAt == nil {
		acc, err := sp.AccountCurrency(ctx, side.AccountID)
		if err != nil {
			return err
		}
		if acc != nil {
			hasAccount = true
		} else if review == nil {
			status = transferNeedsAttention
		}
	}
	at := time.Now().UTC()
	if err := sp.Save(ctx, recordOf(t, status, review, remote), at); err != nil {
		return err
	}
	if t.DeletedAt != nil || !hasAccount {
		return sp.DeleteTransaction(ctx, t.UUID)
	}
	typ := "transfer_in"
	if sender {
		typ = "transfer_out"
	}
	tx := repo.Transaction{
		UUID: t.UUID, Type: typ, Date: t.Date, Status: localStatus, AccountID: side.AccountID, Amount: side.Amount,
		Description: t.Description, CreatedBy: t.CreatedBy,
	}
	exists, err := sp.HasTransaction(ctx, t.UUID)
	if err != nil {
		return err
	}
	if !exists {
		return sp.InsertTransaction(ctx, tx, at)
	}
	return sp.UpdateTransaction(ctx, tx, at)
}

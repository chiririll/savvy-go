// Package transfersrepo stores what linked spaces share: on the server the
// links between spaces and the trusted signing keys; in each space the
// transfer records and the transaction of its own side. Package transfers
// signs, verifies and merges them.
package transfers

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Space reads and writes the transfer tables of one space database.
type Space struct{ DB store.DB }

func (r Space) q() *sqlc.Queries { return db.Q(r.DB) }

// Record is a transfer as stored. Payload is the exact bytes that were signed
// (nil only for a record the other space has and this one lacks); Remote is
// the other space's whole record as JSON while the transfer is in review. The
// source space, destination space and version are generated from Payload (or
// Remote) and are only read.
type Record struct {
	UUID                    string
	Payload                 *string
	SigV                    int64
	SourceKID, SourceSig    *string
	DestinationKID, DestSig *string
	Source, Destination     string
	Version                 int64
	Status                  string
	Remote                  *string
}

func recordFrom(x sqlc.SpaceTransfer) Record {
	return Record{
		UUID: x.Uuid, Payload: optString(x.Payload), SigV: x.SigV,
		SourceKID: optString(x.SourceKid), SourceSig: optString(x.SourceSig),
		DestinationKID: optString(x.DestinationKid), DestSig: optString(x.DestinationSig),
		Source: x.Source.String, Destination: x.Destination.String, Version: x.Version.Int64,
		Status: x.Status, Remote: optString(x.Remote),
	}
}

func optString(v db.NString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

// Records lists every transfer record of the space, newest first.
func (r Space) Records(ctx context.Context) ([]Record, error) {
	rows, err := r.q().ListSpaceTransfers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Record, len(rows))
	for i, row := range rows {
		out[i] = recordFrom(sqlc.SpaceTransfer(row))
	}
	return out, nil
}

// Record returns nil for an unknown uuid.
func (r Space) Record(ctx context.Context, uuid string) (*Record, error) {
	row, err := r.q().GetSpaceTransfer(ctx, uuid)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec := recordFrom(row)
	return &rec, nil
}

// Save inserts or replaces a record.
func (r Space) Save(ctx context.Context, t Record, now time.Time) error {
	return r.q().UpsertSpaceTransfer(ctx, sqlc.UpsertSpaceTransferParams{
		Uuid: t.UUID, Payload: db.NullString(t.Payload), SigV: t.SigV,
		SourceKid: db.NullString(t.SourceKID), SourceSig: db.NullString(t.SourceSig),
		DestinationKid: db.NullString(t.DestinationKID), DestinationSig: db.NullString(t.DestSig),
		Status: t.Status, Remote: db.NullString(t.Remote), UpdatedAt: db.TimeNS(now),
	})
}

// Delete removes a record and the transaction of this space's side; the
// transaction goes first because its payment restricts the record's delete.
func (r Space) Delete(ctx context.Context, uuid string) error {
	if err := r.DeleteMirror(ctx, uuid); err != nil {
		return err
	}
	return r.q().DeleteSpaceTransfer(ctx, uuid)
}

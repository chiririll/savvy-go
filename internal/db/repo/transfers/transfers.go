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

// Server reads and writes the transfer tables of the server database.
type Server struct{ DB store.DB }

func (r Server) q() *sqlc.Queries { return db.Q(r.DB) }

// Link is a pair of linked spaces, lower id first.
type Link struct{ A, B int64 }

func (r Server) Linked(ctx context.Context, a, b int64) (bool, error) {
	n, err := r.q().CountSpaceLink(ctx, sqlc.CountSpaceLinkParams{SpaceAID: a, SpaceBID: b})
	return n > 0, err
}

// Links lists the links a space is in.
func (r Server) Links(ctx context.Context, spaceID int64) ([]Link, error) {
	rows, err := r.q().ListSpaceLinks(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]Link, len(rows))
	for i, row := range rows {
		out[i] = Link{A: row.SpaceAID, B: row.SpaceBID}
	}
	return out, nil
}

func (r Server) InsertLink(ctx context.Context, a, b, actor int64, now time.Time) error {
	return r.q().InsertSpaceLink(ctx, sqlc.InsertSpaceLinkParams{SpaceAID: a, SpaceBID: b, CreatedBy: db.NI(actor), CreatedAt: db.TimeNS(now)})
}

// DeleteLink reports whether the spaces were linked.
func (r Server) DeleteLink(ctx context.Context, a, b int64) (bool, error) {
	res, err := r.q().DeleteSpaceLink(ctx, sqlc.DeleteSpaceLinkParams{SpaceAID: a, SpaceBID: b})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// TrustedKey is a public key (base64) whose signatures this server accepts.
type TrustedKey struct {
	KID, PublicKey, Name string
	CreatedAt            *time.Time
}

// PublicKey returns the base64 key with the id; ok is false when it is not trusted.
func (r Server) PublicKey(ctx context.Context, kid string) (string, bool) {
	raw, err := r.q().GetTrustedKey(ctx, kid)
	return raw, err == nil
}

func (r Server) TrustedKeys(ctx context.Context) ([]TrustedKey, error) {
	rows, err := r.q().ListTrustedKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TrustedKey, len(rows))
	for i, row := range rows {
		out[i] = TrustedKey{KID: row.Kid, PublicKey: row.PublicKey, Name: row.Name}
		if t, ok := db.ParseTime(row.CreatedAt); ok {
			out[i].CreatedAt = &t
		}
	}
	return out, nil
}

func (r Server) InsertTrustedKey(ctx context.Context, kid, publicKey, name string, actor int64, now time.Time) error {
	return r.q().InsertTrustedKey(ctx, sqlc.InsertTrustedKeyParams{
		Kid: kid, PublicKey: publicKey, Name: name, AddedBy: db.NI(actor), CreatedAt: db.TimeNS(now),
	})
}

// DeleteTrustedKey reports whether the key was trusted.
func (r Server) DeleteTrustedKey(ctx context.Context, kid string) (bool, error) {
	res, err := r.q().DeleteTrustedKey(ctx, kid)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// Space reads and writes the transfer tables of one space database.
type Space struct{ DB store.DB }

func (r Space) q() *sqlc.Queries { return db.Q(r.DB) }

// Side is one space's side of a transfer.
type Side struct {
	SpaceUUID string
	AccountID int64
	Amount    int64
	Currency  string
	Decimals  int64
}

// Record is a transfer record as stored. Remote is the other space's version
// as JSON, empty when none.
type Record struct {
	UUID                 string
	From, To             Side
	Date                 string
	Description          *string
	CreatedBy            *int64
	Version, SigV        int64
	DeletedAt            *string
	SignerKID, Signature string
	Status               string
	Review               *string
	Remote               string
}

func recordFrom(r sqlc.SpaceTransfer) Record {
	rec := Record{
		UUID: r.Uuid,
		From: Side{r.FromSpaceUuid, r.FromAccountID, r.FromAmount, r.FromCurrency, r.FromDecimals},
		To:   Side{r.ToSpaceUuid, r.ToAccountID, r.ToAmount, r.ToCurrency, r.ToDecimals},
		Date: r.Date, Version: r.Version, SigV: r.SigV, SignerKID: r.SignerKid.String, Signature: r.Signature.String,
		Status: r.Status, Remote: r.Remote.String,
	}
	if r.Description.Valid {
		rec.Description = &r.Description.String
	}
	if r.CreatedBy.Valid {
		rec.CreatedBy = &r.CreatedBy.Int64
	}
	if r.DeletedAt.Valid {
		rec.DeletedAt = &r.DeletedAt.String
	}
	if r.Review.Valid {
		rec.Review = &r.Review.String
	}
	return rec
}

// Records lists every transfer record of the space.
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
	rec := recordFrom(sqlc.SpaceTransfer(row))
	return &rec, nil
}

// Save inserts or replaces a record.
func (r Space) Save(ctx context.Context, t Record, now time.Time) error {
	var remote db.NString
	if t.Remote != "" {
		remote = db.NS(t.Remote)
	}
	return r.q().UpsertSpaceTransfer(ctx, sqlc.UpsertSpaceTransferParams{
		Uuid: t.UUID, FromSpaceUuid: t.From.SpaceUUID, FromAccountID: t.From.AccountID, FromAmount: t.From.Amount,
		FromCurrency: t.From.Currency, FromDecimals: t.From.Decimals,
		ToSpaceUuid: t.To.SpaceUUID, ToAccountID: t.To.AccountID, ToAmount: t.To.Amount,
		ToCurrency: t.To.Currency, ToDecimals: t.To.Decimals,
		Date: t.Date, Description: db.NullString(t.Description), CreatedBy: db.NullInt64(t.CreatedBy),
		Version: t.Version, DeletedAt: db.NullString(t.DeletedAt), SigV: t.SigV,
		SignerKid: db.NullStringVal(t.SignerKID), Signature: db.NullStringVal(t.Signature),
		Status: t.Status, Review: db.NullString(t.Review), Remote: remote, UpdatedAt: db.TimeNS(now),
	})
}

// Delete removes a record and the transaction of this space's side.
func (r Space) Delete(ctx context.Context, uuid string) error {
	if err := r.DeleteTransaction(ctx, uuid); err != nil {
		return err
	}
	return r.q().DeleteSpaceTransfer(ctx, uuid)
}

// The transaction of this space's side of a transfer.

func (r Space) DeleteTransaction(ctx context.Context, uuid string) error {
	return r.q().DeleteTransferTransaction(ctx, db.NS(uuid))
}

// HasTransaction reports whether this side's transaction exists.
func (r Space) HasTransaction(ctx context.Context, uuid string) (bool, error) {
	_, err := r.q().GetTransferTransaction(ctx, db.NS(uuid))
	if err == db.ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

// Transaction is this space's side of a transfer, as it appears in the books.
type Transaction struct {
	UUID, Type, Date, Status string
	AccountID, Amount        int64
	Description              *string
	CreatedBy                *int64
}

func (r Space) InsertTransaction(ctx context.Context, t Transaction, now time.Time) error {
	return r.q().InsertTransferTransaction(ctx, sqlc.InsertTransferTransactionParams{
		Type: t.Type, AccountID: t.AccountID, Amount: t.Amount, Description: db.NullString(t.Description),
		Date: db.NS(t.Date), Status: t.Status, SpaceTransferUuid: db.NS(t.UUID), CreatedBy: db.NullInt64(t.CreatedBy),
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

func (r Space) UpdateTransaction(ctx context.Context, t Transaction, now time.Time) error {
	return r.q().UpdateTransferTransaction(ctx, sqlc.UpdateTransferTransactionParams{
		Type: t.Type, AccountID: t.AccountID, Amount: t.Amount, Description: db.NullString(t.Description),
		Date: db.NS(t.Date), Status: t.Status, UpdatedAt: db.TimeNS(now), SpaceTransferUuid: db.NS(t.UUID),
	})
}

// AccountCurrency is the currency code and decimals of an account; nil when
// the account does not exist.
type AccountCurrency struct {
	Code     string
	Decimals int64
}

func (r Space) AccountCurrency(ctx context.Context, account int64) (*AccountCurrency, error) {
	row, err := r.q().GetAccountCurrency(ctx, account)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &AccountCurrency{Code: row.Code, Decimals: row.Decimals}, nil
}

// AllLinks lists every link, ordered.
func (r Server) AllLinks(ctx context.Context) ([]Link, error) {
	rows, err := r.q().ListAllSpaceLinks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Link, len(rows))
	for i, row := range rows {
		out[i] = Link{A: row.SpaceAID, B: row.SpaceBID}
	}
	return out, nil
}

// ReviewKey marks a restored space whose transfers wait for a review merge;
// it is written before the restored file goes live.
const ReviewKey = "transfer_review_pending"

// ReviewPending reports whether the space still waits for its review merge.
func (r Space) ReviewPending(ctx context.Context) (bool, error) {
	v, err := r.q().GetSpaceSetting(ctx, ReviewKey)
	if err != nil {
		if err == db.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return v.Valid && v.String != "", nil
}

// ClearReview removes the review mark; a missing mark is not an error.
func (r Space) ClearReview(ctx context.Context) error {
	return r.q().DeleteSpaceSetting(ctx, ReviewKey)
}

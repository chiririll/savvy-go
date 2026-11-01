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

func (r Server) InsertLink(ctx context.Context, a, b, actor int64, now time.Time) error {
	return r.q().
		InsertSpaceLink(ctx,
			sqlc.InsertSpaceLinkParams{SpaceAID: a, SpaceBID: b, CreatedBy: db.NI(actor), CreatedAt: db.TimeNS(now)})
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

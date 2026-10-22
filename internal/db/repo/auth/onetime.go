package auth

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// OneTime is a stored one-time token found by its hash.
type OneTime struct {
	ID, UserID int64
	ExpiresAt  time.Time
}

func oneTimeFrom(id, userID int64, expires string) OneTime {
	t := OneTime{ID: id, UserID: userID}
	t.ExpiresAt, _ = db.ParseTimeString(expires)
	return t
}

func (r Repo) InsertPasswordToken(ctx context.Context, userID int64, hash string, expires, now time.Time) error {
	return r.q().InsertPasswordToken(ctx, sqlc.InsertPasswordTokenParams{
		UserID: userID, TokenHash: hash, ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now),
		UpdatedAt: db.TimeNS(now),
	})
}

// ConsumePasswordToken marks an unexpired, unused token used and reports
// whether it did.
func (r Repo) ConsumePasswordToken(ctx context.Context, hash string, now time.Time) (bool, error) {
	res, err := r.q().ConsumePasswordToken(ctx, sqlc.ConsumePasswordTokenParams{
		ConsumedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), TokenHash: hash, ExpiresAt: db.FormatTime(now),
	})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r Repo) RevokeActivePasswordTokens(ctx context.Context, userID int64, now time.Time) error {
	return r.q().RevokeActivePasswordTokens(ctx, sqlc.RevokeActivePasswordTokensParams{
		ConsumedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), UserID: userID,
	})
}

// ValidPasswordToken finds an unused token that has not expired at now.
func (r Repo) ValidPasswordToken(ctx context.Context, hash string, now time.Time) (*OneTime, error) {
	row, err := r.q().
		GetValidPasswordToken(ctx, sqlc.GetValidPasswordTokenParams{TokenHash: hash, ExpiresAt: db.FormatTime(now)})
	return mapFound(
		row,
		err,
		func(row sqlc.GetValidPasswordTokenRow) OneTime { return oneTimeFrom(row.ID, row.UserID, row.ExpiresAt) },
	)
}

// PasswordTokenByHash finds a token whether or not it is used or expired.
func (r Repo) PasswordTokenByHash(ctx context.Context, hash string) (*OneTime, error) {
	row, err := r.q().GetPasswordTokenByHash(ctx, hash)
	return mapFound(row, err, func(row sqlc.GetPasswordTokenByHashRow) OneTime {
		return oneTimeFrom(row.ID, row.UserID, row.ExpiresAt)
	})
}

// EmailVerification is the stored link that confirms an address.
type EmailVerification struct {
	UserID    int64
	Email     string
	ExpiresAt time.Time
}

// ReplaceEmailVerificationToken stores the user's confirmation link for an
// address, ending any earlier one.
func (r Repo) ReplaceEmailVerificationToken(
	ctx context.Context,
	userID int64,
	email, hash string,
	expires, now time.Time,
) error {
	return store.Tx(ctx, r.DB, func(tx store.DB) error {
		q := db.Q(tx)
		if err := q.DeleteUserEmailVerificationTokens(ctx, userID); err != nil {
			return err
		}
		return q.InsertEmailVerificationToken(ctx, sqlc.InsertEmailVerificationTokenParams{
			UserID: userID, Email: email, TokenHash: hash, ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now),
		})
	})
}

// EmailVerificationTokenByHash finds a confirmation link, expired or not.
func (r Repo) EmailVerificationTokenByHash(ctx context.Context, hash string) (*EmailVerification, error) {
	row, err := r.q().GetEmailVerificationToken(ctx, hash)
	return mapFound(row, err, func(row sqlc.GetEmailVerificationTokenRow) EmailVerification {
		v := EmailVerification{UserID: row.UserID, Email: row.Email}
		v.ExpiresAt, _ = db.ParseTimeString(row.ExpiresAt)
		return v
	})
}

// ConsumeEmailVerificationToken deletes a confirmation link and reports
// whether there was one, so a link works once.
func (r Repo) ConsumeEmailVerificationToken(ctx context.Context, hash string) (bool, error) {
	res, err := r.q().ConsumeEmailVerificationToken(ctx, hash)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

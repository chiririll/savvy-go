package auth

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

const passwordTokenTTL = 7 * 24 * time.Hour

type PasswordToken struct {
	ID        int64
	UserID    int64
	ExpiresAt time.Time
	User      *User
}

type PasswordTokens struct {
	DB store.DB
}

func (p PasswordTokens) Issue(ctx context.Context, user *User) (token string, expires time.Time, err error) {
	if err = p.RevokeActive(ctx, user.ID); err != nil {
		return "", time.Time{}, err
	}
	token = RandomString(64)
	now := time.Now().UTC()
	expires = now.Add(passwordTokenTTL)
	err = db.Q(p.DB).InsertPasswordToken(ctx, sqlc.InsertPasswordTokenParams{
		UserID: user.ID, TokenHash: HashToken(token), ExpiresAt: db.FormatTime(expires),
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	return token, expires, err
}

func (p PasswordTokens) Preview(ctx context.Context, token string) (*PasswordToken, error) {
	return p.findValid(ctx, token)
}

func (p PasswordTokens) Consume(ctx context.Context, token string) (*PasswordToken, error) {
	now := db.FormatTime(time.Now().UTC())
	res, err := db.Q(p.DB).ConsumePasswordToken(ctx, sqlc.ConsumePasswordTokenParams{
		ConsumedAt: db.NS(now), UpdatedAt: db.NS(now), TokenHash: HashToken(token), ExpiresAt: now,
	})
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, nil
	}
	return p.findHash(ctx, HashToken(token))
}

func (p PasswordTokens) RevokeActive(ctx context.Context, userID int64) error {
	now := db.FormatTime(time.Now().UTC())
	return db.Q(p.DB).RevokeActivePasswordTokens(ctx, sqlc.RevokeActivePasswordTokensParams{
		ConsumedAt: db.NS(now), UpdatedAt: db.NS(now), UserID: userID,
	})
}

func (p PasswordTokens) findValid(ctx context.Context, token string) (*PasswordToken, error) {
	row, err := db.Q(p.DB).GetValidPasswordToken(ctx, sqlc.GetValidPasswordTokenParams{
		TokenHash: HashToken(token), ExpiresAt: db.FormatTime(time.Now().UTC()),
	})
	return p.fromValid(ctx, row.ID, row.UserID, row.ExpiresAt, err)
}

func (p PasswordTokens) findHash(ctx context.Context, hash string) (*PasswordToken, error) {
	row, err := db.Q(p.DB).GetPasswordTokenByHash(ctx, hash)
	return p.fromValid(ctx, row.ID, row.UserID, row.ExpiresAt, err)
}

func (p PasswordTokens) fromValid(ctx context.Context, id, userID int64, exp string, err error) (*PasswordToken, error) {
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := &PasswordToken{ID: id, UserID: userID}
	out.ExpiresAt, _ = db.ParseTimeString(exp)
	u, err := Users{DB: p.DB}.ByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out.User = u
	return out, nil
}

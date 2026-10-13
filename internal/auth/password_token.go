package auth

import (
	"context"
	"time"

	repo "savvy-go/internal/db/repo/auth"
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

func (p PasswordTokens) repo() repo.Repo { return repo.Repo{DB: p.DB} }

func (p PasswordTokens) Issue(ctx context.Context, user *User) (token string, expires time.Time, err error) {
	if err = p.RevokeActive(ctx, user.ID); err != nil {
		return "", time.Time{}, err
	}
	token = RandomString(64)
	now := time.Now().UTC()
	expires = now.Add(passwordTokenTTL)
	err = p.repo().InsertPasswordToken(ctx, user.ID, HashToken(token), expires, now)
	return token, expires, err
}

func (p PasswordTokens) Preview(ctx context.Context, token string) (*PasswordToken, error) {
	row, err := p.repo().ValidPasswordToken(ctx, HashToken(token), time.Now().UTC())
	return p.withUser(ctx, row, err)
}

func (p PasswordTokens) Consume(ctx context.Context, token string) (*PasswordToken, error) {
	hash := HashToken(token)
	ok, err := p.repo().ConsumePasswordToken(ctx, hash, time.Now().UTC())
	if err != nil || !ok {
		return nil, err
	}
	row, err := p.repo().PasswordTokenByHash(ctx, hash)
	return p.withUser(ctx, row, err)
}

func (p PasswordTokens) RevokeActive(ctx context.Context, userID int64) error {
	return p.repo().RevokeActivePasswordTokens(ctx, userID, time.Now().UTC())
}

func (p PasswordTokens) withUser(ctx context.Context, row *repo.OneTime, err error) (*PasswordToken, error) {
	if row == nil || err != nil {
		return nil, err
	}
	u, err := Users{DB: p.DB}.ByID(ctx, row.UserID)
	if err != nil {
		return nil, err
	}
	return &PasswordToken{ID: row.ID, UserID: row.UserID, ExpiresAt: row.ExpiresAt, User: u}, nil
}

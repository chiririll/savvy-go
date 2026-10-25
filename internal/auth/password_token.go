package auth

import (
	"context"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"
)

const (
	// inviteTokenTTL is how long the link of an account without a password works: its owner
	// has not signed in yet and may take days to see the invitation.
	inviteTokenTTL = 7 * 24 * time.Hour
	// defaultResetTTL applies when PasswordTokens.ResetTTL is unset.
	defaultResetTTL = time.Hour
)

type PasswordToken struct {
	ID        int64
	UserID    int64
	ExpiresAt time.Time
	User      *User
}

type PasswordTokens struct {
	DB store.DB
	// ResetTTL is the lifetime of a link that replaces the password of an active account.
	// Zero is an hour.
	ResetTTL time.Duration
}

func (p PasswordTokens) repo() repo.Repo { return repo.Repo{DB: p.DB} }

// ttl is the lifetime of a link for user: an account that has no password yet is being
// invited, any other is being reset.
func (p PasswordTokens) ttl(user *User) time.Duration {
	switch {
	case user.IsInactive():
		return inviteTokenTTL
	case p.ResetTTL != 0:
		return p.ResetTTL
	}
	return defaultResetTTL
}

// Issue replaces the user's pending link with a new one.
func (p PasswordTokens) Issue(ctx context.Context, user *User) (token string, expires time.Time, err error) {
	if err = p.RevokeActive(ctx, user.ID); err != nil {
		return "", time.Time{}, err
	}
	token = RandomString(64)
	now := time.Now().UTC()
	expires = now.Add(p.ttl(user))
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

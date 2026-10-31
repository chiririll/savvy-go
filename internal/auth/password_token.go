package auth

import (
	"context"
	"strings"
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
	// Emailed is true when the link was issued by IssueEmailed: its holder read the mail.
	Emailed bool
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

// emailedPrefix marks a link that is sent by email. RandomString never yields "_", so a
// link that was handed over cannot carry it, and adding it to one changes its hash. The
// mark therefore needs no column: it lives in the token the mailbox owner received.
const emailedPrefix = "m_"

// Issue replaces the user's pending link with a new one that the caller hands over as is.
// Whoever holds it has not proven the mailbox.
func (p PasswordTokens) Issue(ctx context.Context, user *User) (token string, expires time.Time, err error) {
	return p.issue(ctx, user, "")
}

// IssueEmailed is Issue for a link that is only sent to the user's address, so using it
// proves the mailbox (see PasswordToken.Emailed). The caller must not reveal the token.
func (p PasswordTokens) IssueEmailed(ctx context.Context, user *User) (token string, expires time.Time, err error) {
	return p.issue(ctx, user, emailedPrefix)
}

func (p PasswordTokens) issue(ctx context.Context, user *User, prefix string) (token string, expires time.Time, err error) {
	if err = p.RevokeActive(ctx, user.ID); err != nil {
		return "", time.Time{}, err
	}
	token = prefix + RandomString(64)
	now := time.Now().UTC()
	expires = now.Add(p.ttl(user))
	err = p.repo().InsertPasswordToken(ctx, user.ID, HashToken(token), expires, now)
	return token, expires, err
}

func (p PasswordTokens) Preview(ctx context.Context, token string) (*PasswordToken, error) {
	row, err := p.repo().ValidPasswordToken(ctx, HashToken(token), time.Now().UTC())
	return p.withUser(ctx, row, err, token)
}

func (p PasswordTokens) Consume(ctx context.Context, token string) (*PasswordToken, error) {
	hash := HashToken(token)
	ok, err := p.repo().ConsumePasswordToken(ctx, hash, time.Now().UTC())
	if err != nil || !ok {
		return nil, err
	}
	row, err := p.repo().PasswordTokenByHash(ctx, hash)
	return p.withUser(ctx, row, err, token)
}

func (p PasswordTokens) RevokeActive(ctx context.Context, userID int64) error {
	return p.repo().RevokeActivePasswordTokens(ctx, userID, time.Now().UTC())
}

func (p PasswordTokens) withUser(ctx context.Context, row *repo.OneTime, err error, token string) (*PasswordToken, error) {
	if row == nil || err != nil {
		return nil, err
	}
	u, err := Users{DB: p.DB}.ByID(ctx, row.UserID)
	if err != nil {
		return nil, err
	}
	return &PasswordToken{ID: row.ID, UserID: row.UserID, ExpiresAt: row.ExpiresAt, User: u,
		Emailed: strings.HasPrefix(token, emailedPrefix)}, nil
}

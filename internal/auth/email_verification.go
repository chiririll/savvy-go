package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"
)

// EmailVerificationTTL is how long a confirmation link works.
const EmailVerificationTTL = 7 * 24 * time.Hour

// EmailVerifications issues the links that confirm an address. A link is one
// random token, stored hashed, with the address it was sent to. There is one
// per user: a new one replaces the old. It works once, and not after the
// address changes.
type EmailVerifications struct {
	DB store.DB
}

func (e EmailVerifications) repo() repo.Repo { return repo.Repo{DB: e.DB} }

// Issue returns a new token for the user's current address and when it expires.
func (e EmailVerifications) Issue(ctx context.Context, user *User) (token string, expires time.Time, err error) {
	token = RandomString(40)
	now := time.Now().UTC()
	expires = now.Add(EmailVerificationTTL)
	err = e.repo().ReplaceEmailVerificationToken(ctx, user.ID, user.Email, HashToken(token), expires, now)
	return token, expires, err
}

// ErrOtherAccount is returned when a confirmation link is opened in the session of
// another user than the one it was sent to.
var ErrOtherAccount = errors.New("this link belongs to another account")

// Consume uses up a token and returns the user whose address it confirms. It
// returns nil for an unknown, used or expired token, for a user who is gone,
// and for one whose address is no longer the one the link was sent to.
// viewerID is the user signed in where the link was opened, 0 for none; a link
// that is not theirs is refused with ErrOtherAccount and stays valid.
func (e EmailVerifications) Consume(ctx context.Context, token string, viewerID int64) (*User, error) {
	hash := HashToken(token)
	row, err := e.repo().EmailVerificationTokenByHash(ctx, hash)
	if row == nil || err != nil {
		return nil, err
	}
	if viewerID != 0 && row.UserID != viewerID {
		return nil, ErrOtherAccount
	}
	ok, err := e.repo().ConsumeEmailVerificationToken(ctx, hash)
	if err != nil || !ok || !time.Now().Before(row.ExpiresAt) {
		return nil, err
	}
	u, err := Users{DB: e.DB}.ByID(ctx, row.UserID)
	if err != nil || u == nil || u.Deleted || !strings.EqualFold(u.Email, row.Email) {
		return nil, err
	}
	return u, nil
}

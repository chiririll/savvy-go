package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"
)

const (
	APIScopeRead      = "read"
	APIScopeReadWrite = "read-write"

	apiTokenPrefix    = "svy_"
	apiTokenTouchEach = time.Minute
	maxAPITokens      = 50
)

var ErrTokenLimit = errors.New("api token limit reached")

// ErrNotFound is returned when the user has no token or passkey with that id.
var ErrNotFound = errors.New("not found")

// APIToken is a long-lived bearer credential issued to an external app.
type APIToken struct {
	ID         int64
	UserID     int64
	Name       string
	Prefix     string
	Scope      string
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  *time.Time
	User       *User
}

func ValidAPIScope(scope string) bool {
	return scope == APIScopeRead || scope == APIScopeReadWrite
}

// CanWrite reports whether the token itself allows mutating requests.
func (t *APIToken) CanWrite() bool { return t.Scope == APIScopeReadWrite }

func (t *APIToken) Expired(now time.Time) bool {
	return t.ExpiresAt != nil && !now.Before(*t.ExpiresAt)
}

type APITokens struct {
	DB store.DB
}

func (s APITokens) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// Issue creates a token and returns the raw value, which is never stored.
func (s APITokens) Issue(ctx context.Context, user *User, name, scope string, expiresAt *time.Time) (string, *APIToken, error) {
	existing, err := s.repo().APITokens(ctx, user.ID)
	if err != nil {
		return "", nil, err
	}
	if len(existing) >= maxAPITokens {
		return "", nil, ErrTokenLimit
	}
	raw := apiTokenPrefix + RandomString(48)
	prefix := raw[:len(apiTokenPrefix)+4]
	now := time.Now().UTC()
	id, err := s.repo().InsertAPIToken(ctx, repo.NewAPIToken{
		UserID: user.ID, Name: name, Hash: HashToken(raw), Prefix: prefix, Scope: scope, ExpiresAt: expiresAt, Now: now,
	})
	if err != nil {
		return "", nil, err
	}
	return raw, &APIToken{
		ID: id, UserID: user.ID, Name: name, Prefix: prefix, Scope: scope, ExpiresAt: expiresAt, CreatedAt: &now,
	}, nil
}

func (s APITokens) List(ctx context.Context, userID int64) ([]APIToken, error) {
	rows, err := s.repo().APITokens(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]APIToken, len(rows))
	for i, r := range rows {
		out[i] = toAPIToken(r)
	}
	return out, nil
}

// Resolve returns the active token and its owner, or nil for unknown/expired values.
func (s APITokens) Resolve(ctx context.Context, raw string) (*APIToken, error) {
	if !strings.HasPrefix(raw, apiTokenPrefix) {
		return nil, nil
	}
	row, err := s.repo().APITokenByHash(ctx, HashToken(raw))
	if row == nil || err != nil {
		return nil, err
	}
	t := toAPIToken(*row)
	now := time.Now().UTC()
	if t.Expired(now) {
		return nil, nil
	}
	u, err := Users{DB: s.DB}.ByID(ctx, t.UserID)
	if err != nil || u == nil {
		return nil, err
	}
	t.User = u
	if t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) > apiTokenTouchEach {
		_ = s.repo().TouchAPIToken(ctx, t.ID, now)
	}
	return &t, nil
}

// Revoke deletes the user's token; ErrNotFound if they have no such one.
func (s APITokens) Revoke(ctx context.Context, userID, id int64) error {
	ok, err := s.repo().DeleteAPIToken(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func toAPIToken(r repo.APIToken) APIToken {
	return APIToken{
		ID: r.ID, UserID: r.UserID, Name: r.Name, Prefix: r.Prefix, Scope: r.Scope,
		ExpiresAt: r.ExpiresAt, LastUsedAt: r.LastUsedAt, CreatedAt: r.CreatedAt,
	}
}

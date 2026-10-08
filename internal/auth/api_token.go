package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
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

// Issue creates a token and returns the raw value, which is never stored.
func (s APITokens) Issue(ctx context.Context, user *User, name, scope string, expiresAt *time.Time) (string, *APIToken, error) {
	n, err := s.count(ctx, user.ID)
	if err != nil {
		return "", nil, err
	}
	if n >= maxAPITokens {
		return "", nil, ErrTokenLimit
	}
	raw := apiTokenPrefix + RandomString(48)
	now := time.Now().UTC()
	res, err := db.Q(s.DB).InsertAPIToken(ctx, sqlc.InsertAPITokenParams{
		UserID: user.ID, Name: name, TokenHash: HashToken(raw), Prefix: raw[:len(apiTokenPrefix)+4],
		Scope: scope, ExpiresAt: db.NullTime(expiresAt), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return "", nil, err
	}
	id, _ := res.LastInsertId()
	return raw, &APIToken{
		ID: id, UserID: user.ID, Name: name, Prefix: raw[:len(apiTokenPrefix)+4], Scope: scope,
		ExpiresAt: expiresAt, CreatedAt: &now,
	}, nil
}

func (s APITokens) count(ctx context.Context, userID int64) (int, error) {
	rows, err := db.Q(s.DB).ListAPITokensByUser(ctx, userID)
	return len(rows), err
}

func (s APITokens) List(ctx context.Context, userID int64) ([]APIToken, error) {
	rows, err := db.Q(s.DB).ListAPITokensByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]APIToken, 0, len(rows))
	for _, r := range rows {
		out = append(out, tokenFromRow(r))
	}
	return out, nil
}

// Resolve returns the active token and its owner, or nil for unknown/expired values.
func (s APITokens) Resolve(ctx context.Context, raw string) (*APIToken, error) {
	if !strings.HasPrefix(raw, apiTokenPrefix) {
		return nil, nil
	}
	row, err := db.Q(s.DB).GetAPITokenByHash(ctx, HashToken(raw))
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t := tokenFromRow(row)
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
		_ = db.Q(s.DB).TouchAPIToken(ctx, sqlc.TouchAPITokenParams{LastUsedAt: db.TimeNS(now), ID: t.ID})
	}
	return &t, nil
}

func (s APITokens) Revoke(ctx context.Context, userID, id int64) error {
	res, err := db.Q(s.DB).DeleteAPIToken(ctx, sqlc.DeleteAPITokenParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return db.ErrNotFound
	}
	return nil
}

func tokenFromRow(r sqlc.ApiToken) APIToken {
	t := APIToken{ID: r.ID, UserID: r.UserID, Name: r.Name, Prefix: r.Prefix, Scope: r.Scope}
	if v, ok := db.ParseTime(r.ExpiresAt); ok {
		t.ExpiresAt = &v
	}
	if v, ok := db.ParseTime(r.LastUsedAt); ok {
		t.LastUsedAt = &v
	}
	if v, ok := db.ParseTime(r.CreatedAt); ok {
		t.CreatedAt = &v
	}
	return t
}

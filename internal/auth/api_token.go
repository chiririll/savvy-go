package auth

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"

	"savvy-go/internal/auth/permissions"
)

const (
	apiTokenPrefix    = "svy_"
	apiTokenTouchEach = time.Minute
	maxAPITokens      = 50
)

var ErrTokenLimit = errors.New("api token limit reached")

// ErrNotFound is returned when the user has no token or passkey with that id.
var ErrNotFound = errors.New("not found")

// Grant is what a token may do: which resources, and in which spaces.
type Grant struct {
	// Kind is permissions.KindPersonal (the default) or permissions.KindMCP. Fixed at issue.
	Kind        string
	Permissions []string
	// AllSpaces reaches every space the owner belongs to, now and later.
	// Otherwise the token reaches only SpaceIDs.
	AllSpaces bool
	SpaceIDs  []int64
}

// APIToken is a long-lived bearer credential issued to an external app.
type APIToken struct {
	ID          int64
	UserID      int64
	Name        string
	Prefix      string
	Kind        string
	Permissions []string
	AllSpaces   bool
	SpaceIDs    []int64
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	CreatedAt   *time.Time
	User        *User
}

// Allows reports whether the token holds a grant for resource that covers
// level (permissions.Read, permissions.Write or permissions.Delete): a higher grant covers the lower.
func (t *APIToken) Allows(resource, level string) bool {
	need := permissions.Rank(level)
	for _, p := range t.Permissions {
		res, lv, _ := strings.Cut(p, ":")
		if res == resource && need > 0 && permissions.Rank(lv) >= need {
			return true
		}
	}
	return false
}

// AllowsSpace reports whether the token reaches the space. It says nothing
// about the owner's role there, which is checked as well.
func (t *APIToken) AllowsSpace(id int64) bool {
	return t.AllSpaces || slices.Contains(t.SpaceIDs, id)
}

func (t *APIToken) Expired(now time.Time) bool {
	return t.ExpiresAt != nil && !now.Before(*t.ExpiresAt)
}

type APITokens struct {
	DB store.DB
}

func (s APITokens) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// checkGrant validates a grant for a token of this kind and returns the
// normalized permissions and the sorted, distinct space ids (none for all
// spaces). The error is an permissions.ErrBadGrant.
func checkGrant(kind string, grant Grant) (perms []string, spaceIDs []int64, err error) {
	if perms, err = permissions.Normalize(grant.Permissions); err != nil {
		return nil, nil, permissions.ErrBadGrant
	}
	if !grant.AllSpaces && len(grant.SpaceIDs) == 0 {
		return nil, nil, permissions.ErrBadGrant
	}
	if err = permissions.CheckKind(kind, perms); err != nil {
		return nil, nil, err
	}
	spaceIDs = []int64{}
	if !grant.AllSpaces {
		spaceIDs = slices.Compact(slices.Sorted(slices.Values(grant.SpaceIDs)))
	}
	return perms, spaceIDs, nil
}

// Issue creates a token and returns the raw value, which is never stored. The
// grant must name at least one space unless it is for all of them; the caller
// has checked that the owner belongs to each.
func (s APITokens) Issue(
	ctx context.Context,
	user *User,
	name string,
	grant Grant,
	expiresAt *time.Time,
) (string, *APIToken, error) {
	if grant.Kind == "" {
		grant.Kind = permissions.KindPersonal
	}
	perms, spaceIDs, err := checkGrant(grant.Kind, grant)
	if err != nil {
		return "", nil, err
	}
	raw := apiTokenPrefix + RandomString(48)
	prefix := raw[:len(apiTokenPrefix)+4]
	now := time.Now().UTC()
	var id int64
	err = store.Tx(ctx, s.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		existing, err := r.APITokens(ctx, user.ID)
		if err != nil {
			return err
		}
		if len(existing) >= maxAPITokens {
			return ErrTokenLimit
		}
		id, err = r.InsertAPIToken(ctx, repo.NewAPIToken{
			UserID: user.ID, Name: name, Hash: HashToken(raw), Prefix: prefix, Kind: grant.Kind,
			Permissions: perms, AllSpaces: grant.AllSpaces, SpaceIDs: spaceIDs, ExpiresAt: expiresAt, Now: now,
		})
		return err
	})
	if err != nil {
		return "", nil, err
	}
	return raw, &APIToken{
		ID: id, UserID: user.ID, Name: name, Prefix: prefix, Kind: grant.Kind, Permissions: perms,
		AllSpaces: grant.AllSpaces, SpaceIDs: spaceIDs, ExpiresAt: expiresAt, CreatedAt: &now,
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

// UpdateGrant changes what the user's token may do and which spaces it reaches,
// keeping the token itself: the app that holds it needs no new one. It takes
// effect on the next request. For a connected AI app it also decides what the
// next refresh carries over. The token keeps its kind, and a connected app's
// grant is held to the parts its tools use. ErrNotFound if they have no such
// token.
func (s APITokens) UpdateGrant(ctx context.Context, userID, id int64, grant Grant) (*APIToken, error) {
	cur, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	perms, spaceIDs, err := checkGrant(cur.Kind, grant)
	if err != nil {
		return nil, err
	}
	var found bool
	err = store.Tx(ctx, s.DB, func(tx store.DB) error {
		var e error
		found, e = repo.Repo{DB: tx}.UpdateAPITokenGrant(ctx, userID, id, perms, grant.AllSpaces, spaceIDs, time.Now().UTC())
		return e
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return s.Get(ctx, userID, id)
}

// Get returns the user's token with that id, or ErrNotFound.
func (s APITokens) Get(ctx context.Context, userID, id int64) (*APIToken, error) {
	list, err := s.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == id {
			return &list[i], nil
		}
	}
	return nil, ErrNotFound
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

// RevokeAll deletes every token the user has; none is not an error.
func (s APITokens) RevokeAll(ctx context.Context, userID int64) error {
	return s.repo().DeleteAPITokens(ctx, userID)
}

func toAPIToken(r repo.APIToken) APIToken {
	return APIToken{
		ID: r.ID, UserID: r.UserID, Name: r.Name, Prefix: r.Prefix, Kind: r.Kind,
		Permissions: r.Permissions, AllSpaces: r.AllSpaces, SpaceIDs: r.SpaceIDs,
		ExpiresAt: r.ExpiresAt, LastUsedAt: r.LastUsedAt, CreatedAt: r.CreatedAt,
	}
}

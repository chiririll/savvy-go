package auth

import (
	"context"
	"strings"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

type APIToken struct {
	ID, UserID                       int64
	Name, Prefix, Kind               string
	Permissions                      []string
	AllSpaces                        bool
	SpaceIDs                         []int64 // only the token's own list; empty when AllSpaces
	ExpiresAt, LastUsedAt, CreatedAt *time.Time
}

type NewAPIToken struct {
	UserID             int64
	Name, Hash, Prefix string
	Kind               string
	Permissions        []string
	AllSpaces          bool
	SpaceIDs           []int64
	ExpiresAt          *time.Time
	Now                time.Time
}

// InsertAPIToken stores the token and its spaces; call it inside a transaction.
func (r Repo) InsertAPIToken(ctx context.Context, t NewAPIToken) (int64, error) {
	if t.Kind == "" {
		t.Kind = "personal"
	}
	res, err := r.q().InsertAPIToken(ctx, sqlc.InsertAPITokenParams{
		UserID: t.UserID, Name: t.Name, TokenHash: t.Hash, Prefix: t.Prefix, Kind: t.Kind,
		Permissions: strings.Join(t.Permissions, ","), AllSpaces: boolInt(t.AllSpaces),
		ExpiresAt: db.NullTime(t.ExpiresAt), CreatedAt: db.TimeNS(t.Now), UpdatedAt: db.TimeNS(t.Now),
	})
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if !t.AllSpaces {
		for _, sid := range t.SpaceIDs {
			if err := r.q().InsertAPITokenSpace(ctx, sqlc.InsertAPITokenSpaceParams{ApiTokenID: id, SpaceID: sid}); err != nil {
				return 0, err
			}
		}
	}
	return id, nil
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func apiTokenFrom(row sqlc.ApiToken, spaceIDs []int64) APIToken {
	perms := []string{}
	if row.Permissions != "" {
		perms = strings.Split(row.Permissions, ",")
	}
	if spaceIDs == nil {
		spaceIDs = []int64{}
	}
	return APIToken{
		ID: row.ID, UserID: row.UserID, Name: row.Name, Prefix: row.Prefix, Kind: row.Kind,
		Permissions: perms, AllSpaces: row.AllSpaces == 1, SpaceIDs: spaceIDs,
		ExpiresAt: optTime(row.ExpiresAt), LastUsedAt: optTime(row.LastUsedAt), CreatedAt: optTime(row.CreatedAt),
	}
}

func (r Repo) APITokens(ctx context.Context, userID int64) ([]APIToken, error) {
	rows, err := r.q().ListAPITokensByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	links, err := r.q().ListAPITokenSpacesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	spaces := map[int64][]int64{}
	for _, l := range links {
		spaces[l.ApiTokenID] = append(spaces[l.ApiTokenID], l.SpaceID)
	}
	out := make([]APIToken, len(rows))
	for i, row := range rows {
		out[i] = apiTokenFrom(row, spaces[row.ID])
	}
	return out, nil
}

func (r Repo) APITokenByHash(ctx context.Context, hash string) (*APIToken, error) {
	row, err := r.q().GetAPITokenByHash(ctx, hash)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ids, err := r.q().ListAPITokenSpaces(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	t := apiTokenFrom(row, ids)
	return &t, nil
}

func (r Repo) TouchAPIToken(ctx context.Context, id int64, now time.Time) error {
	return r.q().TouchAPIToken(ctx, sqlc.TouchAPITokenParams{LastUsedAt: db.TimeNS(now), ID: id})
}

// UpdateAPITokenGrant replaces what the user's token may do and which spaces it
// reaches, and reports whether they had such a token. Call it inside a
// transaction: the spaces are rewritten too.
func (r Repo) UpdateAPITokenGrant(
	ctx context.Context,
	userID, id int64,
	perms []string,
	allSpaces bool,
	spaceIDs []int64,
	now time.Time,
) (bool, error) {
	res, err := r.q().UpdateAPITokenGrant(ctx, sqlc.UpdateAPITokenGrantParams{
		Permissions: strings.Join(
			perms,
			",",
		), AllSpaces: boolInt(allSpaces), UpdatedAt: db.TimeNS(now), ID: id, UserID: userID,
	})
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}
	if err := r.q().DeleteAPITokenSpaces(ctx, id); err != nil {
		return false, err
	}
	if !allSpaces {
		for _, sid := range spaceIDs {
			if err := r.q().InsertAPITokenSpace(ctx, sqlc.InsertAPITokenSpaceParams{ApiTokenID: id, SpaceID: sid}); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// DeleteAPITokens deletes every token of the user, with their space links and OAuth refresh tokens.
func (r Repo) DeleteAPITokens(ctx context.Context, userID int64) error {
	return r.q().DeleteUserAPITokens(ctx, userID)
}

// DeleteAPIToken reports whether the user had such a token.
func (r Repo) DeleteAPIToken(ctx context.Context, userID, id int64) (bool, error) {
	res, err := r.q().DeleteAPIToken(ctx, sqlc.DeleteAPITokenParams{ID: id, UserID: userID})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// Package oauth is where OAuth clients and refresh tokens are read from and
// written to the server database. Package internal/oauth knows the rules;
// this one speaks plain Go values: times as time.Time, redirect URIs as a
// slice, "no such row" as nil.
package oauth

import (
	"context"
	"encoding/json"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Repo reads and writes the OAuth tables of one server database.
type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

func found[T any](v T, err error) (*T, error) {
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// RequestInfo is what the server saw of the registration request, for the
// admin who reviews the registered apps.
type RequestInfo struct{ IP, UserAgent, Origin string }

// Client is a registered OAuth client.
type Client struct {
	ID, Name     string
	RedirectURIs []string
	Request      RequestInfo
	CreatedAt    time.Time
}

func (r Repo) InsertClient(ctx context.Context, id, name string, redirectURIs []string, info RequestInfo, now time.Time) error {
	raw, err := json.Marshal(redirectURIs)
	if err != nil {
		return err
	}
	return r.q().InsertOAuthClient(ctx, sqlc.InsertOAuthClientParams{
		ClientID: id, Name: name, RedirectUris: string(raw),
		Ip: info.IP, UserAgent: info.UserAgent, Origin: info.Origin, CreatedAt: db.TimeNS(now),
	})
}

func toClient(row sqlc.OauthClient) (*Client, error) {
	c := Client{ID: row.ClientID, Name: row.Name, Request: RequestInfo{IP: row.Ip, UserAgent: row.UserAgent, Origin: row.Origin}}
	if err := json.Unmarshal([]byte(row.RedirectUris), &c.RedirectURIs); err != nil {
		return nil, err
	}
	c.CreatedAt, _ = db.ParseTime(row.CreatedAt)
	return &c, nil
}

func (r Repo) Client(ctx context.Context, id string) (*Client, error) {
	row, err := found(r.q().GetOAuthClient(ctx, id))
	if row == nil || err != nil {
		return nil, err
	}
	return toClient(*row)
}

// Clients lists every client, newest first.
func (r Repo) Clients(ctx context.Context) ([]Client, error) {
	rows, err := r.q().ListOAuthClients(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Client, 0, len(rows))
	for _, row := range rows {
		c, err := toClient(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

// DeleteClient removes the client, its refresh tokens and the access tokens
// they carry; false when it is unknown.
func (r Repo) DeleteClient(ctx context.Context, id string) (bool, error) {
	if err := r.q().DeleteAPITokensOfOAuthClient(ctx, id); err != nil {
		return false, err
	}
	res, err := r.q().DeleteOAuthClient(ctx, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r Repo) CountClients(ctx context.Context) (int, error) {
	n, err := r.q().CountOAuthClients(ctx)
	return int(n), err
}

// PruneClients deletes clients registered before cutoff that never got a refresh token.
func (r Repo) PruneClients(ctx context.Context, cutoff time.Time) error {
	_, err := r.q().DeleteIdleOAuthClients(ctx, db.TimeNS(cutoff))
	return err
}

// Refresh is a stored refresh token, found by its hash.
type Refresh struct {
	ID, UserID, APITokenID int64
	ClientID               string
	ExpiresAt              time.Time
}

type NewRefresh struct {
	Hash, ClientID     string
	UserID, APITokenID int64
	ExpiresAt, Now     time.Time
}

func (r Repo) InsertRefresh(ctx context.Context, t NewRefresh) error {
	return r.q().InsertOAuthRefresh(ctx, sqlc.InsertOAuthRefreshParams{
		TokenHash: t.Hash, ClientID: t.ClientID, UserID: t.UserID, ApiTokenID: t.APITokenID,
		ExpiresAt: db.FormatTime(t.ExpiresAt), CreatedAt: db.TimeNS(t.Now),
	})
}

func (r Repo) RefreshByHash(ctx context.Context, hash string) (*Refresh, error) {
	row, err := found(r.q().GetOAuthRefreshByHash(ctx, hash))
	if row == nil || err != nil {
		return nil, err
	}
	out := Refresh{ID: row.ID, UserID: row.UserID, APITokenID: row.ApiTokenID, ClientID: row.ClientID}
	out.ExpiresAt, _ = db.ParseTimeString(row.ExpiresAt)
	return &out, nil
}

// RotateRefresh swaps the token's hash and access token, but only while it
// still holds oldHash, so a replayed token cannot rotate twice. It reports
// whether it changed the row.
func (r Repo) RotateRefresh(ctx context.Context, id int64, oldHash, newHash string, apiTokenID int64, expires time.Time) (bool, error) {
	res, err := r.q().RotateOAuthRefresh(ctx, sqlc.RotateOAuthRefreshParams{
		TokenHash: newHash, ApiTokenID: apiTokenID, ExpiresAt: db.FormatTime(expires), ID: id, TokenHash_2: oldHash,
	})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r Repo) DeleteRefresh(ctx context.Context, id int64) error {
	return r.q().DeleteOAuthRefresh(ctx, id)
}

func (r Repo) DeleteExpiredRefresh(ctx context.Context, now time.Time) error {
	return r.q().DeleteExpiredOAuthRefresh(ctx, db.FormatTime(now))
}

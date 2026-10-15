// Package ssorepo stores identity providers, the links between them and
// users, and the short-lived state and tickets of a sign-in. Package sso
// decides what the stored settings mean.
package sso

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

// Provider is an identity provider row. Config, Secrets, ClaimMappings and
// RoleMapping are JSON text, empty when unset.
type Provider struct {
	ID                                              int64
	Name, Slug, Protocol, Preset, DefaultRole       string
	Enabled, AllowJIT, SyncRoleOnLogin, LinkByEmail bool
	SortOrder                                       int
	Config, Secrets, ClaimMappings, RoleMapping     string
	CreatedAt, UpdatedAt                            *time.Time
}

func providerFrom(row sqlc.IdentityProvider) Provider {
	p := Provider{
		ID: row.ID, Name: row.Name, Slug: row.Slug, Protocol: row.Protocol, Preset: row.Preset,
		Enabled: row.Enabled != 0, SortOrder: int(row.SortOrder), DefaultRole: row.DefaultRole,
		AllowJIT: row.AllowJit != 0, SyncRoleOnLogin: row.SyncRoleOnLogin != 0, LinkByEmail: row.LinkByEmail != 0,
		Config: row.Config.String, Secrets: row.Secrets.String,
		ClaimMappings: row.ClaimMappings.String, RoleMapping: row.RoleMapping.String,
	}
	if t, ok := db.ParseTime(row.CreatedAt); ok {
		p.CreatedAt = &t
	}
	if t, ok := db.ParseTime(row.UpdatedAt); ok {
		p.UpdatedAt = &t
	}
	return p
}

// Filter narrows List; the zero value lists every provider.
type Filter struct {
	ID          int64 // 0: any
	Slug        string
	EnabledOnly bool
}

func (r Repo) List(ctx context.Context, f Filter) ([]Provider, error) {
	arg := sqlc.ListIdentityProvidersParams{ID: db.NInt{}, Slug: db.NullStringVal(f.Slug)}
	if f.ID != 0 {
		arg.ID = db.NI(f.ID)
	}
	if f.EnabledOnly {
		arg.EnabledOnly = db.Flag(true)
	}
	rows, err := r.q().ListIdentityProviders(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Provider, len(rows))
	for i, row := range rows {
		out[i] = providerFrom(row)
	}
	return out, nil
}

func (r Repo) CountEnabled(ctx context.Context) (int, error) {
	n, err := r.q().CountEnabledIdentityProviders(ctx)
	return int(n), err
}

// Save is the writable part of a provider.
type Save struct {
	Name, Slug, Protocol, Preset, DefaultRole       string
	Enabled, AllowJIT, SyncRoleOnLogin, LinkByEmail bool
	SortOrder                                       int
	Config, Secrets, ClaimMappings, RoleMapping     string
}

func (r Repo) Insert(ctx context.Context, p Save, now time.Time) (int64, error) {
	res, err := r.q().InsertIdentityProvider(ctx, sqlc.InsertIdentityProviderParams{
		Name: p.Name, Slug: p.Slug, Protocol: p.Protocol, Preset: p.Preset,
		Enabled: db.BoolInt(p.Enabled), SortOrder: int64(p.SortOrder),
		Config: db.NS(p.Config), Secrets: db.NS(p.Secrets), ClaimMappings: db.NS(p.ClaimMappings), RoleMapping: db.NS(p.RoleMapping),
		DefaultRole: p.DefaultRole, AllowJit: db.BoolInt(p.AllowJIT), SyncRoleOnLogin: db.BoolInt(p.SyncRoleOnLogin),
		LinkByEmail: db.BoolInt(p.LinkByEmail), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) Update(ctx context.Context, id int64, p Save, now time.Time) error {
	return r.q().UpdateIdentityProvider(ctx, sqlc.UpdateIdentityProviderParams{
		Name: p.Name, Slug: p.Slug, Protocol: p.Protocol, Preset: p.Preset,
		Enabled: db.BoolInt(p.Enabled), SortOrder: int64(p.SortOrder),
		Config: db.NS(p.Config), Secrets: db.NS(p.Secrets), ClaimMappings: db.NS(p.ClaimMappings), RoleMapping: db.NS(p.RoleMapping),
		DefaultRole: p.DefaultRole, AllowJit: db.BoolInt(p.AllowJIT),
		SyncRoleOnLogin: db.BoolInt(p.SyncRoleOnLogin), LinkByEmail: db.BoolInt(p.LinkByEmail),
		UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) Delete(ctx context.Context, id int64) error {
	return r.q().DeleteIdentityProvider(ctx, id)
}

// SSOOnlyOrphans counts users who can sign in only through this provider.
func (r Repo) SSOOnlyOrphans(ctx context.Context, id int64) (int, error) {
	n, err := r.q().CountSSOOnlyOrphans(ctx, id)
	return int(n), err
}

// Identity links

// Link is the tie between a provider's subject and a user.
type Link struct{ ID, UserID int64 }

// LinkBySubject returns nil when the subject has not signed in before.
func (r Repo) LinkBySubject(ctx context.Context, providerID int64, subject string) (*Link, error) {
	row, err := r.q().GetUserIdentity(ctx, sqlc.GetUserIdentityParams{IdentityProviderID: providerID, Subject: subject})
	if err != nil {
		if err == db.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &Link{ID: row.ID, UserID: row.UserID}, nil
}

func (r Repo) InsertLink(ctx context.Context, userID, providerID int64, subject, claims string, now time.Time) error {
	return r.q().InsertUserIdentity(ctx, sqlc.InsertUserIdentityParams{
		UserID: userID, IdentityProviderID: providerID, Subject: subject,
		LastLoginAt: db.TimeNS(now), Claims: db.NS(claims), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

func (r Repo) TouchLink(ctx context.Context, id int64, claims string, now time.Time) error {
	return r.q().TouchUserIdentity(ctx, sqlc.TouchUserIdentityParams{
		LastLoginAt: db.TimeNS(now), Claims: db.NS(claims), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// Sign-in tickets and state

func (r Repo) InsertTicket(ctx context.Context, ticket string, userID int64, requires2FA bool, expires, now time.Time) error {
	return r.q().InsertSSOTicket(ctx, sqlc.InsertSSOTicketParams{
		Ticket: ticket, UserID: userID, Requires2fa: db.BoolInt(requires2FA),
		ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

// TakeTicket uses up an unexpired ticket. ok is false for one that is
// unknown, used or expired.
func (r Repo) TakeTicket(ctx context.Context, ticket string, now time.Time) (userID int64, requires2FA, ok bool, err error) {
	res, err := r.q().ConsumeSSOTicket(ctx, sqlc.ConsumeSSOTicketParams{
		ConsumedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), Ticket: ticket, ExpiresAt: db.FormatTime(now),
	})
	if err != nil {
		return 0, false, false, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return 0, false, false, nil
	}
	row, err := r.q().GetSSOTicket(ctx, ticket)
	if err != nil {
		return 0, false, false, err
	}
	return row.UserID, row.Requires2fa != 0, true, nil
}

// State is what a sign-in remembers between redirecting out and coming back.
type State struct {
	ProviderID                                        int64
	Nonce, CodeVerifier, SAMLRequestID, RedirectAfter string
}

func (r Repo) InsertState(ctx context.Context, state string, st State, expires, now time.Time) error {
	return r.q().InsertSSOState(ctx, sqlc.InsertSSOStateParams{
		State: state, IdentityProviderID: st.ProviderID,
		Nonce: db.NullStringVal(st.Nonce), CodeVerifier: db.NullStringVal(st.CodeVerifier),
		SamlRequestID: db.NullStringVal(st.SAMLRequestID), RedirectAfter: db.NullStringVal(st.RedirectAfter),
		ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

func (r Repo) AttachSAMLRequestID(ctx context.Context, state, requestID string) error {
	return r.q().AttachSAMLRequestID(ctx, sqlc.AttachSAMLRequestIDParams{SamlRequestID: db.NS(requestID), State: state})
}

// TakeState removes the state and returns it; nil when it is unknown or has
// expired by now.
func (r Repo) TakeState(ctx context.Context, state string, now time.Time) (*State, error) {
	row, err := r.q().GetSSOState(ctx, state)
	if err != nil {
		return nil, nil
	}
	_ = r.q().DeleteSSOState(ctx, state)
	if expires, ok := db.ParseTimeString(row.ExpiresAt); !ok || !now.Before(expires) {
		return nil, nil
	}
	return &State{
		ProviderID: row.IdentityProviderID, Nonce: row.Nonce.String, CodeVerifier: row.CodeVerifier.String,
		SAMLRequestID: row.SamlRequestID.String, RedirectAfter: row.RedirectAfter.String,
	}, nil
}

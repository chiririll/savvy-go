package domain

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// An invitation is a single-use link a space admin hands to someone. It may
// be bound to an email; only a user with that address can accept it. Only its
// hash is stored. Whoever registers through an invitation from someone other
// than a server admin becomes a guest: they cannot create spaces until a
// server admin promotes them.

const invitationTTL = 7 * 24 * time.Hour

var (
	// ErrInvitationInvalid covers unknown, expired and used invitations alike,
	// so the answer does not tell them apart.
	ErrInvitationInvalid = errors.New("this invitation is not valid")
	// ErrInvitationEmail is returned when an email-bound invitation is used
	// with another address.
	ErrInvitationEmail = errors.New("this invitation is for another email address")
	// ErrRegistrationClosed is returned when invitations may not create accounts.
	ErrRegistrationClosed = errors.New("invitations cannot create accounts on this server; ask the server administrator")
)

// Invitation is an invitation as its space's admins list it.
type Invitation struct {
	ID         int64
	Email      *string
	Role       string
	Inviter    string
	ExpiresAt  *time.Time
	AcceptedAt *time.Time
	CreatedAt  *time.Time
}

// InvitationPreview is what the holder of a link sees before accepting.
type InvitationPreview struct {
	SpaceName string
	Role      string
	Inviter   string
	Email     *string
	ExpiresAt time.Time
	// CanRegister reports whether the link may create an account.
	CanRegister bool
}

type invitation struct {
	id, spaceID        int64
	email              *string
	role               string
	byServerAdmin      bool
	expires            time.Time
	accepted           bool
	spaceName, inviter string
}

// Invite creates an invitation and returns its raw token, shown once.
func (s Spaces) Invite(ctx context.Context, spaceID int64, inviter *auth.User, email, role string) (string, error) {
	if !ValidSpaceRole(role) {
		return "", errors.New("invalid space role")
	}
	token := auth.RandomString(40)
	var mail sql.NullString
	if e := strings.TrimSpace(email); e != "" {
		mail = db.NS(e)
	}
	t := time.Now().UTC()
	_, err := s.server().InsertInvitation(ctx, sqlc.InsertInvitationParams{
		SpaceID: spaceID, Email: mail, Role: role, TokenHash: auth.HashToken(token),
		InvitedBy: db.NI(inviter.ID), InvitedByServerAdmin: db.BoolInt(inviter.IsAdmin()),
		ExpiresAt: t.Add(invitationTTL).Format(time.RFC3339), CreatedAt: db.NS(t.Format(time.RFC3339)),
	})
	return token, err
}

// Invitations lists a space's invitations, newest first.
func (s Spaces) Invitations(ctx context.Context, spaceID int64) ([]Invitation, error) {
	rows, err := s.server().ListInvitations(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]Invitation, len(rows))
	for i, r := range rows {
		inv := Invitation{ID: r.ID, Role: r.Role, Inviter: r.Inviter}
		if r.Email.Valid {
			e := r.Email.String
			inv.Email = &e
		}
		if t, err := time.Parse(time.RFC3339, r.ExpiresAt); err == nil {
			inv.ExpiresAt = &t
		}
		if t, ok := parseNullTime(r.AcceptedAt); ok {
			inv.AcceptedAt = &t
		}
		if t, ok := parseNullTime(r.CreatedAt); ok {
			inv.CreatedAt = &t
		}
		out[i] = inv
	}
	return out, nil
}

// Revoke deletes an invitation of a space.
func (s Spaces) Revoke(ctx context.Context, spaceID, id int64) error {
	res, err := s.server().DeleteInvitation(ctx, sqlc.DeleteInvitationParams{ID: id, SpaceID: spaceID})
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrInvitationInvalid
	}
	return nil
}

func (s Spaces) lookupInvitation(ctx context.Context, q *sqlc.Queries, token string) (*invitation, error) {
	r, err := q.GetInvitationByHash(ctx, auth.HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvitationInvalid
	}
	if err != nil {
		return nil, err
	}
	inv := &invitation{id: r.ID, spaceID: r.SpaceID, role: r.Role, byServerAdmin: r.InvitedByServerAdmin == 1,
		accepted: r.AcceptedAt.Valid, spaceName: r.SpaceName, inviter: r.Inviter}
	if r.Email.Valid {
		e := r.Email.String
		inv.email = &e
	}
	inv.expires, _ = time.Parse(time.RFC3339, r.ExpiresAt)
	if inv.accepted || !time.Now().Before(inv.expires) {
		return nil, ErrInvitationInvalid
	}
	return inv, nil
}

// Preview describes a valid invitation to whoever holds its link.
func (s Spaces) Preview(ctx context.Context, token string) (*InvitationPreview, error) {
	inv, err := s.lookupInvitation(ctx, s.server(), token)
	if err != nil {
		return nil, err
	}
	return &InvitationPreview{
		SpaceName: inv.spaceName, Role: inv.role, Inviter: inv.inviter, Email: inv.email, ExpiresAt: inv.expires,
		CanRegister: s.settings().Bool(ctx, "space_invites_can_register", true),
	}, nil
}

// Accept makes u a member with the invitation's role and uses it up. An
// email-bound invitation needs u's email; a guest cannot take an admin role
// (P8) and a new admin stays within the limit (P10).
func (s Spaces) Accept(ctx context.Context, token string, u *auth.User) (*Space, error) {
	// Checks first: inside the transaction only its handle may be used (the
	// database has one connection, which the transaction holds).
	inv, err := s.lookupInvitation(ctx, s.server(), token)
	if err != nil {
		return nil, err
	}
	if inv.email != nil && !strings.EqualFold(*inv.email, u.Email) {
		return nil, ErrInvitationEmail
	}
	if inv.role == SpaceAdmin {
		if u.IsGuest() {
			return nil, ErrGuestCannotAdmin
		}
		if current, _ := s.Role(ctx, inv.spaceID, u.ID); current != SpaceAdmin {
			if err := s.checkAdminLimit(ctx, u); err != nil {
				return nil, err
			}
		}
	}
	err = store.Tx(ctx, s.Store.Server(), func(tx store.DB) error {
		q := db.Q(tx)
		res, err := q.AcceptInvitation(ctx, sqlc.AcceptInvitationParams{AcceptedAt: db.NS(now()), AcceptedBy: db.NI(u.ID), ID: inv.id})
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrInvitationInvalid // used meanwhile
		}
		return q.UpsertSpaceMember(ctx, sqlc.UpsertSpaceMemberParams{SpaceID: inv.spaceID, UserID: u.ID, Role: inv.role, CreatedAt: db.NS(now())})
	})
	if err != nil {
		return nil, err
	}
	sp, err := s.Get(ctx, inv.spaceID)
	if sp != nil {
		sp.Role, _ = s.Role(ctx, inv.spaceID, u.ID)
	}
	return sp, err
}

// Register creates an account through an invitation and accepts it. The new
// user is a guest unless a server admin sent the invitation; they get no
// personal space (P7).
func (s Spaces) Register(ctx context.Context, users auth.Users, token, name, email, password string) (*auth.User, *Space, error) {
	if !s.settings().Bool(ctx, "space_invites_can_register", true) {
		return nil, nil, ErrRegistrationClosed
	}
	inv, err := s.lookupInvitation(ctx, s.server(), token)
	if err != nil {
		return nil, nil, err
	}
	if inv.email != nil && !strings.EqualFold(*inv.email, email) {
		return nil, nil, ErrInvitationEmail
	}
	role := auth.RoleGuest
	if inv.byServerAdmin {
		role = auth.RoleUser
	}
	if inv.role == SpaceAdmin && role == auth.RoleGuest {
		return nil, nil, ErrGuestCannotAdmin
	}
	u, err := users.Create(ctx, name, email, &password, role)
	if err != nil {
		return nil, nil, err
	}
	sp, err := s.Accept(ctx, token, u)
	if err != nil {
		_ = users.Delete(ctx, u.ID)
		return nil, nil, err
	}
	return u, sp, nil
}

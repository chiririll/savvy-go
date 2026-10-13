package spaces

import (
	"context"
	"errors"
	"strings"
	"time"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/spaces"
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
func (s Service) Invite(ctx context.Context, spaceID int64, inviter *auth.User, email, role string) (string, error) {
	if !ValidRole(role) {
		return "", errors.New("invalid space role")
	}
	token := auth.RandomString(40)
	now := time.Now().UTC()
	return token, s.registry().InsertInvitation(ctx, repo.NewInvitation{
		SpaceID: spaceID, Email: strings.TrimSpace(email), Role: role, TokenHash: auth.HashToken(token),
		InvitedBy: inviter.ID, ByServerAdmin: inviter.IsAdmin(), Expires: now.Add(invitationTTL), Now: now,
	})
}

// Invitations lists a space's invitations, newest first.
func (s Service) Invitations(ctx context.Context, spaceID int64) ([]Invitation, error) {
	rows, err := s.registry().Invitations(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]Invitation, len(rows))
	for i, r := range rows {
		out[i] = Invitation{ID: r.ID, Email: r.Email, Role: r.Role, Inviter: r.Inviter,
			ExpiresAt: r.ExpiresAt, AcceptedAt: r.AcceptedAt, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// Revoke deletes an invitation of a space.
func (s Service) Revoke(ctx context.Context, spaceID, id int64) error {
	ok, err := s.registry().DeleteInvitation(ctx, spaceID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvitationInvalid
	}
	return nil
}

func (s Service) lookupInvitation(ctx context.Context, reg repo.Repo, token string) (*invitation, error) {
	r, err := reg.InvitationByHash(ctx, auth.HashToken(token))
	if err != nil {
		return nil, err
	}
	if r == nil || r.Used || !time.Now().Before(r.Expires) {
		return nil, ErrInvitationInvalid
	}
	return &invitation{id: r.ID, spaceID: r.SpaceID, email: r.Email, role: r.Role, byServerAdmin: r.ByServerAdmin,
		expires: r.Expires, accepted: r.Used, spaceName: r.SpaceName, inviter: r.Inviter}, nil
}

// Preview describes a valid invitation to whoever holds its link.
func (s Service) Preview(ctx context.Context, token string) (*InvitationPreview, error) {
	inv, err := s.lookupInvitation(ctx, s.registry(), token)
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
func (s Service) Accept(ctx context.Context, token string, u *auth.User) (*Space, error) {
	// Checks first: inside the transaction only its handle may be used (the
	// database has one connection, which the transaction holds).
	inv, err := s.lookupInvitation(ctx, s.registry(), token)
	if err != nil {
		return nil, err
	}
	if inv.email != nil && !strings.EqualFold(*inv.email, u.Email) {
		return nil, ErrInvitationEmail
	}
	if inv.role == RoleAdmin {
		if u.IsGuest() {
			return nil, ErrGuestCannotAdmin
		}
		if current, _ := s.Role(ctx, inv.spaceID, u.ID); current != RoleAdmin {
			if err := s.checkAdminLimit(ctx, u); err != nil {
				return nil, err
			}
		}
	}
	err = store.Tx(ctx, s.Store.Server(), func(tx store.DB) error {
		reg := repo.Repo{DB: tx}
		now := time.Now().UTC()
		ok, err := reg.AcceptInvitation(ctx, inv.id, u.ID, now)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvitationInvalid // used meanwhile
		}
		return reg.UpsertMember(ctx, inv.spaceID, u.ID, inv.role, now)
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
func (s Service) Register(ctx context.Context, users auth.Users, token, name, email, password string) (*auth.User, *Space, error) {
	if !s.settings().Bool(ctx, "space_invites_can_register", true) {
		return nil, nil, ErrRegistrationClosed
	}
	inv, err := s.lookupInvitation(ctx, s.registry(), token)
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
	if inv.role == RoleAdmin && role == auth.RoleGuest {
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

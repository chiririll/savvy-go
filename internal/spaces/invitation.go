package spaces

import (
	"context"
	"errors"
	"fmt"
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
//
// A bound invitation is meant to be emailed and nothing else: the caller must
// never show its token to the inviter, so holding the token proves the
// mailbox and registering through it confirms the address.
//
// Revoking expires an invitation and keeps the row. An unaccepted row for an
// address stops a new email to it: a pending one in the same space says "already
// invited", an expired one anywhere is the cooldown that keeps an admin from
// flooding a mailbox with invite, revoke, invite. PruneInvitations removes the
// expired rows, which ends the cooldown.

const (
	invitationTTL = 7 * 24 * time.Hour
	// A row must stay at least this long past its expiry before it is pruned.
	invitationCooldown = time.Hour
)

var (
	// ErrInvitationInvalid covers unknown, expired and used invitations alike,
	// so the answer does not tell them apart.
	ErrInvitationInvalid = errors.New("this invitation is not valid")
	// ErrInvitationEmail is returned when an email-bound invitation is used
	// with another address.
	ErrInvitationEmail = errors.New("this invitation is for another email address")
	// ErrInvitationExists is returned when the address already has a pending
	// invitation to the space.
	ErrInvitationExists = errors.New("this address is already invited to the space")
	// ErrInvitationCooldown is returned when an invitation to the address
	// expired or was revoked a moment ago.
	ErrInvitationCooldown = errors.New("this address was invited a moment ago; try again in a couple of hours")
	// ErrAlreadyMember is returned when the address belongs to a member.
	ErrAlreadyMember = errors.New("this address already belongs to a member of the space")
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

// Invite creates an invitation and returns its raw token, shown once. An
// invitation bound to an address is refused when the address is a member, is
// already invited to the space, or is in its cooldown.
func (s Service) Invite(ctx context.Context, spaceID int64, inviter *auth.User, email, role string) (string, int64, error) {
	if !ValidRole(role) {
		return "", 0, errors.New("invalid space role")
	}
	email = strings.TrimSpace(email)
	token := auth.RandomString(40)
	now := time.Now().UTC()
	var id int64
	err := store.Tx(ctx, s.Store.Server(), func(tx store.DB) error {
		reg := repo.Repo{DB: tx}
		if email != "" {
			if err := checkInvitable(ctx, reg, spaceID, email, now); err != nil {
				return err
			}
		}
		var err error
		id, err = reg.InsertInvitation(ctx, repo.NewInvitation{
			SpaceID: spaceID, Email: email, Role: role, TokenHash: auth.HashToken(token),
			InvitedBy: inviter.ID, ByServerAdmin: inviter.IsAdmin(), Expires: now.Add(invitationTTL), Now: now,
		})
		return err
	})
	if err != nil {
		return "", 0, err
	}
	return token, id, nil
}

func checkInvitable(ctx context.Context, reg repo.Repo, spaceID int64, email string, now time.Time) error {
	if member, err := reg.IsMemberByEmail(ctx, spaceID, email); err != nil {
		return err
	} else if member {
		return ErrAlreadyMember
	}
	held, err := reg.UnacceptedFor(ctx, email)
	if err != nil {
		return err
	}
	cooling := false
	for _, h := range held {
		switch {
		case now.Before(h.Expires) && h.SpaceID == spaceID:
			return ErrInvitationExists
		case !now.Before(h.Expires):
			cooling = true
		}
	}
	if cooling {
		return ErrInvitationCooldown
	}
	return nil
}

// DiscardInvitation deletes an unaccepted invitation outright, as when its email could
// not be sent: nobody received it, so it should not hold the address.
func (s Service) DiscardInvitation(ctx context.Context, id int64) error {
	return s.registry().DeletePending(ctx, id)
}

// Invitations lists a space's invitations, newest first: accepted ones and
// those still open.
func (s Service) Invitations(ctx context.Context, spaceID int64) ([]Invitation, error) {
	rows, err := s.registry().Invitations(ctx, spaceID, time.Now().UTC())
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

// Revoke ends a pending invitation of a space, or removes an accepted one from
// its history. A revoked row stays until pruned, so its address is in the
// cooldown.
func (s Service) Revoke(ctx context.Context, spaceID, id int64) error {
	reg := s.registry()
	ok, err := reg.ExpireInvitation(ctx, spaceID, id, time.Now().UTC())
	if err == nil && !ok {
		ok, err = reg.DeleteAcceptedInvitation(ctx, spaceID, id)
	}
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvitationInvalid
	}
	return nil
}

// PruneInvitations deletes the invitations that expired more than the cooldown
// ago and returns how many. Run it often: the cooldown ends when the row goes.
func (s Service) PruneInvitations(ctx context.Context) (int, error) {
	return s.registry().PruneInvitations(ctx, time.Now().UTC().Add(-invitationCooldown))
}

// ClearInvitationCooldowns deletes the expired invitations of one address, or
// of all when email is empty, and records it. It is for a server admin: a
// space admin who could clear the cooldown would defeat it.
func (s Service) ClearInvitationCooldowns(ctx context.Context, actor *auth.User, email string) (int, error) {
	email = strings.TrimSpace(email)
	n, err := s.registry().ClearExpiredInvitations(ctx, email, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	details := fmt.Sprintf("%d cleared", n)
	if email != "" {
		details = fmt.Sprintf("%s: %d cleared", email, n)
	}
	return n, s.Audit(ctx, actor, "invitations.cleared", nil, nil, details)
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
	// The token of a bound invitation only ever went to that mailbox.
	if inv.email != nil {
		if err := users.SetEmailVerified(ctx, u.ID, true); err != nil {
			return nil, nil, err
		}
		u.EmailVerified = true
	}
	return u, sp, nil
}

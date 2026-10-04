package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Roles inside a space. They decide what a member may do with its finances;
// the server role (auth.Role*) decides what they may do with the server.
const (
	SpaceAdmin  = "admin"
	SpaceEditor = "editor"
	SpaceViewer = "viewer"
)

// ValidSpaceRole reports whether role is a space role.
func ValidSpaceRole(role string) bool {
	return role == SpaceAdmin || role == SpaceEditor || role == SpaceViewer
}

// CanWriteSpace reports whether a space role may change the space's data.
func CanWriteSpace(role string) bool { return role == SpaceAdmin || role == SpaceEditor }

// spaceUUIDKey is the space_settings key that ties a space database to its
// registry row, so a file can be recognised when it is restored or imported.
const spaceUUIDKey = "space_uuid"

// Space is a space as one user sees it: Role is their role in it.
type Space struct {
	ID   int64
	UUID string
	Name string
	Role string
}

// ErrGuestCannotAdmin is returned when a guest would become a space admin.
var ErrGuestCannotAdmin = errors.New("a guest cannot be a space admin")

// Spaces keeps the registry of spaces and their members in the server
// database and creates their databases through the store.
type Spaces struct{ Store store.Store }

func (s Spaces) server() *sqlc.Queries { return db.Q(s.Store.Server()) }

// Create registers a space, creates its database and, when owner is set,
// makes them its admin.
func (s Spaces) Create(ctx context.Context, name string, owner *auth.User) (*Space, error) {
	if owner != nil && owner.IsGuest() {
		return nil, ErrGuestCannotAdmin
	}
	id7, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var createdBy sql.NullInt64
	if owner != nil {
		createdBy = db.NI(owner.ID)
	}
	id, err := s.server().InsertSpace(ctx, sqlc.InsertSpaceParams{
		Uuid: id7.String(), Name: name, CreatedBy: createdBy, CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	if err := s.Store.CreateSpace(ctx, id); err != nil {
		_ = s.server().DeleteSpaceRow(ctx, id)
		return nil, fmt.Errorf("create space database: %w", err)
	}
	spaceDB, err := s.Store.Space(ctx, id)
	if err == nil {
		err = db.Q(spaceDB).UpsertSpaceSetting(ctx, sqlc.UpsertSpaceSettingParams{Key: spaceUUIDKey, Value: db.NS(id7.String())})
	}
	if err != nil {
		_ = s.Store.DeleteSpace(ctx, id)
		_ = s.server().DeleteSpaceRow(ctx, id)
		return nil, err
	}
	sp := &Space{ID: id, UUID: id7.String(), Name: name}
	if owner != nil {
		if err := s.SetMember(ctx, id, owner, SpaceAdmin); err != nil {
			return nil, err
		}
		sp.Role = SpaceAdmin
	}
	return sp, nil
}

// Provision gives a new user their personal space. Guests get none: they
// only take part in spaces they were invited to.
func (s Spaces) Provision(ctx context.Context, u *auth.User) (*Space, error) {
	if u == nil || u.IsGuest() || s.Store == nil {
		return nil, nil
	}
	return s.Create(ctx, u.Name, u)
}

// SetMember adds u to a space or changes their role.
func (s Spaces) SetMember(ctx context.Context, spaceID int64, u *auth.User, role string) error {
	if !ValidSpaceRole(role) {
		return fmt.Errorf("invalid space role %q", role)
	}
	if role == SpaceAdmin && u.IsGuest() {
		return ErrGuestCannotAdmin
	}
	return s.server().UpsertSpaceMember(ctx, sqlc.UpsertSpaceMemberParams{
		SpaceID: spaceID, UserID: u.ID, Role: role, CreatedAt: db.NS(time.Now().UTC().Format(time.RFC3339)),
	})
}

// Role is userID's role in a space, "" when they are not a member.
func (s Spaces) Role(ctx context.Context, spaceID, userID int64) (string, error) {
	role, err := s.server().GetMemberRole(ctx, sqlc.GetMemberRoleParams{SpaceID: spaceID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return role, err
}

// ForUser lists the spaces a user is a member of, oldest first.
func (s Spaces) ForUser(ctx context.Context, userID int64) ([]Space, error) {
	rows, err := s.server().ListUserSpaces(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Space, len(rows))
	for i, r := range rows {
		out[i] = Space{ID: r.ID, UUID: r.Uuid, Name: r.Name, Role: r.Role}
	}
	return out, nil
}

// IDs lists every registered space.
func (s Spaces) IDs(ctx context.Context) ([]int64, error) {
	return s.server().ListSpaceIDs(ctx)
}

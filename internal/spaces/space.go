package spaces

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/spaces"
	"savvy-go/internal/store"
)

// Roles inside a space. They decide what a member may do with its finances;
// the server role (auth.Role*) decides what they may do with the server.
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// ValidRole reports whether role is a space role.
func ValidRole(role string) bool {
	return role == RoleAdmin || role == RoleEditor || role == RoleViewer
}

// CanWrite reports whether a space role may change the space's data.
func CanWrite(role string) bool { return role == RoleAdmin || role == RoleEditor }

// Space is a space as one user sees it: Role is their role in it.
type Space struct {
	ID   int64
	UUID string
	Name string
	Role string
}

// ErrGuestCannotAdmin is returned when a guest would become a space admin.
var ErrGuestCannotAdmin = errors.New("a guest cannot be a space admin")

// Service keeps the registry of spaces and their members in the server
// database and creates their databases through the store.
type Service struct{ Store store.Store }

func (s Service) registry() repo.Repo { return repo.Repo{DB: s.Store.Server()} }

// stampUUID writes a space's uuid into its own database.
func (s Service) stampUUID(ctx context.Context, id int64, uuid string) error {
	spaceDB, err := s.Store.Space(ctx, id)
	if err != nil {
		return err
	}
	return repo.Stamp{DB: spaceDB}.SetUUID(ctx, uuid)
}

func ownerID(owner *auth.User) *int64 {
	if owner == nil {
		return nil
	}
	return &owner.ID
}

// Create registers a space, creates its database and, when owner is set,
// makes them its admin.
func (s Service) Create(ctx context.Context, name string, owner *auth.User) (*Space, error) {
	if owner != nil && owner.IsGuest() {
		return nil, ErrGuestCannotAdmin
	}
	id7, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	id, err := s.registry().Insert(ctx, id7.String(), name, ownerID(owner), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if err := s.Store.CreateSpace(ctx, id); err != nil {
		_ = s.registry().Delete(ctx, id)
		return nil, fmt.Errorf("create space database: %w", err)
	}
	if err := s.stampUUID(ctx, id, id7.String()); err != nil {
		_ = s.Store.DeleteSpace(ctx, id)
		_ = s.registry().Delete(ctx, id)
		return nil, err
	}
	sp := &Space{ID: id, UUID: id7.String(), Name: name}
	if owner != nil {
		if err := s.SetMember(ctx, id, owner, RoleAdmin); err != nil {
			return nil, err
		}
		sp.Role = RoleAdmin
	}
	return sp, nil
}

// CreateFrom registers a space whose database is a prepared backup. It keeps
// the backup's uuid unless another space here has it.
func (s Service) CreateFrom(ctx context.Context, name string, owner *auth.User, p *store.PreparedSpace) (*Space, error) {
	if owner != nil && owner.IsGuest() {
		s.Store.Discard(p.Artifact)
		return nil, ErrGuestCannotAdmin
	}
	id7 := p.UUID
	taken, err := s.registry().HasUUID(ctx, id7)
	if id7 == "" || taken || err != nil {
		fresh, err := uuid.NewV7()
		if err != nil {
			s.Store.Discard(p.Artifact)
			return nil, err
		}
		id7 = fresh.String()
	}
	id, err := s.registry().Insert(ctx, id7, name, ownerID(owner), time.Now().UTC())
	if err != nil {
		s.Store.Discard(p.Artifact)
		return nil, err
	}
	if err := s.Store.ImportSpace(ctx, id, p); err != nil {
		_ = s.registry().Delete(ctx, id)
		return nil, err
	}
	if id7 != p.UUID {
		if err := s.stampUUID(ctx, id, id7); err != nil {
			return nil, err
		}
	}
	sp := &Space{ID: id, UUID: id7, Name: name}
	if owner != nil {
		if err := s.SetMember(ctx, id, owner, RoleAdmin); err != nil {
			return nil, err
		}
		sp.Role = RoleAdmin
	}
	return sp, nil
}

// Get is a space by id, without a role.
func (s Service) Get(ctx context.Context, id int64) (*Space, error) {
	row, err := s.registry().ByID(ctx, id)
	if row == nil || err != nil {
		return nil, err
	}
	return &Space{ID: row.ID, UUID: row.UUID, Name: row.Name}, nil
}

// Provision gives a new user their personal space. Guests get none: they
// only take part in spaces they were invited to.
func (s Service) Provision(ctx context.Context, u *auth.User) (*Space, error) {
	if u == nil || u.IsGuest() || s.Store == nil {
		return nil, nil
	}
	return s.Create(ctx, u.Name, u)
}

// SetMember adds u to a space or changes their role.
func (s Service) SetMember(ctx context.Context, spaceID int64, u *auth.User, role string) error {
	if !ValidRole(role) {
		return fmt.Errorf("invalid space role %q", role)
	}
	if role == RoleAdmin && u.IsGuest() {
		return ErrGuestCannotAdmin
	}
	return s.registry().UpsertMember(ctx, spaceID, u.ID, role, time.Now().UTC())
}

// Role is userID's role in a space, "" when they are not a member.
func (s Service) Role(ctx context.Context, spaceID, userID int64) (string, error) {
	return s.registry().MemberRole(ctx, spaceID, userID)
}

// ForUser lists the spaces a user is a member of, oldest first.
func (s Service) ForUser(ctx context.Context, userID int64) ([]Space, error) {
	rows, err := s.registry().ForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Space, len(rows))
	for i, r := range rows {
		out[i] = Space{ID: r.ID, UUID: r.UUID, Name: r.Name, Role: r.Role}
	}
	return out, nil
}

// All lists every registered space with its uuid.
func (s Service) All(ctx context.Context) ([]Space, error) {
	rows, err := s.registry().All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Space, len(rows))
	for i, r := range rows {
		out[i] = Space{ID: r.ID, UUID: r.UUID, Name: r.Name}
	}
	return out, nil
}

// IDs lists every registered space.
func (s Service) IDs(ctx context.Context) ([]int64, error) {
	return s.registry().IDs(ctx)
}

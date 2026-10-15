package auth

import (
	"context"
	"strings"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"
)

// Server roles. What a user may do with the finances of a space is decided by
// their role in that space, not here.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
	// RoleGuest joined through an invitation from someone other than a server
	// admin: they cannot create spaces or be a space admin until promoted.
	RoleGuest = "guest"
)

// ValidRole reports whether role is a server role.
func ValidRole(role string) bool {
	return role == RoleAdmin || role == RoleUser || role == RoleGuest
}

// NormalizeRole maps the retired read-write/read-only roles (Laravel data,
// old SSO role mappings) to user.
func NormalizeRole(role string) string {
	switch role {
	case "read-write", "read-only":
		return RoleUser
	}
	return role
}

type User struct {
	ID                 int64
	Name               string
	Email              string
	Password           *string
	Role               string
	IsSSOOnly          bool
	EmailVerified      bool
	TwoFactorSecret    *string
	TwoFactorEnabled   bool
	TwoFactorConfirmed bool
	Deleted            bool
	CreatedAt          *time.Time
	UpdatedAt          *time.Time
}

func (u User) IsInactive() bool {
	return u.Password == nil || *u.Password == ""
}

func (u User) HasTwoFactor() bool {
	return u.TwoFactorEnabled && u.TwoFactorConfirmed
}

func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

func (u User) IsGuest() bool { return u.Role == RoleGuest }

func (u User) SessionJSON() map[string]any {
	return map[string]any{
		"id":            u.ID,
		"name":          u.Name,
		"email":         u.Email,
		"role":          u.Role,
		"isSsoOnly":     u.IsSSOOnly,
		"emailVerified": u.EmailVerified,
	}
}

func (u User) ResourceJSON() map[string]any {
	var created any
	if u.CreatedAt != nil {
		created = u.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"id":            u.ID,
		"name":          u.Name,
		"email":         u.Email,
		"role":          u.Role,
		"isInactive":    u.IsInactive(),
		"isSsoOnly":     u.IsSSOOnly,
		"emailVerified": u.EmailVerified,
		"createdAt":     created,
	}
}

type Users struct {
	DB store.DB
}

func (s Users) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Users) Count(ctx context.Context) (int, error) { return s.repo().CountUsers(ctx) }

func (s Users) ByID(ctx context.Context, id int64) (*User, error) {
	row, err := s.repo().UserByID(ctx, id)
	return userFrom(row, err)
}

func (s Users) ByEmail(ctx context.Context, email string) (*User, error) {
	row, err := s.repo().UserByEmail(ctx, strings.ToLower(email))
	return userFrom(row, err)
}

func (s Users) All(ctx context.Context) ([]User, error) {
	rows, err := s.repo().ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]User, len(rows))
	for i := range rows {
		out[i] = *toUser(&rows[i])
	}
	return out, nil
}

func (s Users) Create(ctx context.Context, name, email string, password *string, role string) (*User, error) {
	if role == "" {
		role = RoleUser
	}
	var hash string // empty: an account with no password (SSO only)
	if password != nil && *password != "" {
		h, err := HashPassword(*password)
		if err != nil {
			return nil, err
		}
		hash = h
	}
	id, err := s.repo().InsertUser(ctx, name, strings.ToLower(email), hash, role, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Users) UpdatePassword(ctx context.Context, id int64, plain string) error {
	h, err := HashPassword(plain)
	if err != nil {
		return err
	}
	return s.repo().SetPassword(ctx, id, h, time.Now().UTC())
}

// A new address clears EmailVerified, unless verified says otherwise (an admin's word).
func (s Users) Update(ctx context.Context, id int64, name, email, role *string, password *string, verified *bool) (*User, error) {
	u, err := s.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		u.Name = *name
	}
	if email != nil {
		// A new address has not been proven yet.
		if e := strings.ToLower(*email); e != u.Email {
			u.Email = e
			u.EmailVerified = false
		}
	}
	if role != nil {
		u.Role = *role
	}
	if verified != nil {
		u.EmailVerified = *verified
	}
	var hash string // empty keeps the current password
	if password != nil && *password != "" {
		h, err := HashPassword(*password)
		if err != nil {
			return nil, err
		}
		hash = h
	}
	if err := s.repo().UpdateUser(ctx, id, u.Name, u.Email, u.Role, hash, u.EmailVerified, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// SetEmailVerified records whether the user has proven they own their address.
func (s Users) SetEmailVerified(ctx context.Context, id int64, verified bool) error {
	return s.repo().SetEmailVerified(ctx, id, verified, time.Now().UTC())
}

// VerifiedAdminCount is the number of admins whose address is verified.
func (s Users) VerifiedAdminCount(ctx context.Context) (int, error) {
	return s.repo().CountVerifiedAdmins(ctx, RoleAdmin)
}

func (s Users) MarkSSOOnly(ctx context.Context, id int64) error {
	return s.repo().MarkSSOOnly(ctx, id, time.Now().UTC())
}

func (s Users) SetRole(ctx context.Context, id int64, role string) error {
	return s.repo().SetRole(ctx, id, role, time.Now().UTC())
}

func (s Users) SetTwoFactor(ctx context.Context, id int64, secret *string, enabled, confirmed bool) error {
	return s.repo().SetTwoFactor(ctx, id, secret, enabled, confirmed, time.Now().UTC())
}

// Delete tombstones a user: the row stays (ids in space databases still
// resolve to "Deleted user") but loses its email, password, second factors,
// sessions, tokens, passkeys, SSO links and space memberships.
func (s Users) Delete(ctx context.Context, id int64) error {
	return s.repo().Tombstone(ctx, id, time.Now().UTC())
}

func (s Users) AdminCount(ctx context.Context) (int, error) {
	return s.repo().CountAdmins(ctx, RoleAdmin)
}

func userFrom(r *repo.User, err error) (*User, error) {
	if r == nil || err != nil {
		return nil, err
	}
	return toUser(r), nil
}

func toUser(r *repo.User) *User {
	return &User{
		ID: r.ID, Name: r.Name, Email: r.Email, Password: r.Password, Role: r.Role,
		IsSSOOnly: r.SSOOnly, EmailVerified: r.EmailVerified, TwoFactorSecret: r.TwoFactorSecret,
		TwoFactorEnabled: r.TwoFactorEnabled, TwoFactorConfirmed: r.TwoFactorConfirmed,
		Deleted: r.Deleted, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

package auth

import (
	"context"
	"strings"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
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
		"id":        u.ID,
		"name":      u.Name,
		"email":     u.Email,
		"role":      u.Role,
		"isSsoOnly": u.IsSSOOnly,
	}
}

func (u User) ResourceJSON() map[string]any {
	var created any
	if u.CreatedAt != nil {
		created = u.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"id":         u.ID,
		"name":       u.Name,
		"email":      u.Email,
		"role":       u.Role,
		"isInactive": u.IsInactive(),
		"isSsoOnly":  u.IsSSOOnly,
		"createdAt":  created,
	}
}

type Users struct {
	DB store.DB
}

func (s Users) Count(ctx context.Context) (int, error) {
	n, err := db.Q(s.DB).CountUsers(ctx)
	return int(n), err
}

func (s Users) ByID(ctx context.Context, id int64) (*User, error) {
	row, err := db.Q(s.DB).GetUser(ctx, id)
	return userFromRow(row, err)
}

func (s Users) ByEmail(ctx context.Context, email string) (*User, error) {
	row, err := db.Q(s.DB).GetUserByEmail(ctx, strings.ToLower(email))
	return userFromRow(row, err)
}

func (s Users) All(ctx context.Context) ([]User, error) {
	rows, err := db.Q(s.DB).ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]User, 0, len(rows))
	for _, r := range rows {
		u, err := userFromRow(r, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, nil
}

func (s Users) Create(ctx context.Context, name, email string, password *string, role string) (*User, error) {
	if role == "" {
		role = RoleUser
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var hash string // empty: an account with no password (SSO only)
	if password != nil && *password != "" {
		h, err := HashPassword(*password)
		if err != nil {
			return nil, err
		}
		hash = h
	}
	res, err := db.Q(s.DB).InsertUser(ctx, sqlc.InsertUserParams{
		Name: name, Email: strings.ToLower(email), Password: db.NullStringVal(hash), Role: role,
		CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
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
	return db.Q(s.DB).UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		Password: db.NS(h), UpdatedAt: db.NS(time.Now().UTC().Format(time.RFC3339)), ID: id,
	})
}

func (s Users) Update(ctx context.Context, id int64, name, email, role *string, password *string) (*User, error) {
	u, err := s.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		u.Name = *name
	}
	if email != nil {
		u.Email = strings.ToLower(*email)
	}
	if role != nil {
		u.Role = *role
	}
	var hash string
	if password != nil && *password != "" {
		h, err := HashPassword(*password)
		if err != nil {
			return nil, err
		}
		hash = h
	}
	now := time.Now().UTC().Format(time.RFC3339)
	q := db.Q(s.DB)
	if hash != "" {
		err = q.UpdateUserWithPassword(ctx, sqlc.UpdateUserWithPasswordParams{
			Name: u.Name, Email: u.Email, Role: u.Role, Password: db.NS(hash), UpdatedAt: db.NS(now), ID: id,
		})
	} else {
		err = q.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
			Name: u.Name, Email: u.Email, Role: u.Role, UpdatedAt: db.NS(now), ID: id,
		})
	}
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Users) MarkSSOOnly(ctx context.Context, id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return db.Q(s.DB).MarkUserSSOOnly(ctx, sqlc.MarkUserSSOOnlyParams{UpdatedAt: db.NS(now), ID: id})
}

func (s Users) SetRole(ctx context.Context, id int64, role string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return db.Q(s.DB).SetUserRole(ctx, sqlc.SetUserRoleParams{Role: role, UpdatedAt: db.NS(now), ID: id})
}

func (s Users) SetTwoFactor(ctx context.Context, id int64, secret *string, enabled, confirmed bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return db.Q(s.DB).SetUserTwoFactor(ctx, sqlc.SetUserTwoFactorParams{
		TwoFactorSecret: db.NullString(secret), TwoFactorEnabled: db.BoolInt(enabled),
		TwoFactorConfirmed: db.BoolInt(confirmed), UpdatedAt: db.NS(now), ID: id,
	})
}

// Delete tombstones a user: the row stays (ids in space databases still
// resolve to "Deleted user") but loses its email, password, second factors,
// sessions, tokens, passkeys, SSO links and space memberships.
func (s Users) Delete(ctx context.Context, id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.Tx(ctx, s.DB, func(tx store.DB) error {
		q := db.Q(tx)
		for _, del := range []func(context.Context, int64) error{
			q.DeleteUserCredentials, q.DeleteUserAPITokens, q.DeleteUserPasskeys,
			q.DeleteUserIdentities, q.DeleteUserPasswordTokens, q.DeleteUserRecoveryCodes,
			q.DeleteUserMemberships,
		} {
			if err := del(ctx, id); err != nil {
				return err
			}
		}
		return q.TombstoneUser(ctx, sqlc.TombstoneUserParams{Now: db.NS(now), ID: id})
	})
}

func (s Users) AdminCount(ctx context.Context) (int, error) {
	n, err := db.Q(s.DB).CountAdmins(ctx, RoleAdmin)
	return int(n), err
}

func userFromRow(r sqlc.User, err error) (*User, error) {
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u := &User{ID: r.ID, Name: r.Name, Email: r.Email, Role: r.Role}
	if r.Password.Valid {
		u.Password = &r.Password.String
	}
	if r.TwoFactorSecret.Valid {
		u.TwoFactorSecret = &r.TwoFactorSecret.String
	}
	u.IsSSOOnly = r.IsSsoOnly != 0
	u.TwoFactorEnabled = r.TwoFactorEnabled != 0
	u.TwoFactorConfirmed = r.TwoFactorConfirmed != 0
	u.Deleted = r.DeletedAt.Valid
	if t, ok := db.ParseTime(r.CreatedAt); ok {
		u.CreatedAt = &t
	}
	if t, ok := db.ParseTime(r.UpdatedAt); ok {
		u.UpdatedAt = &t
	}
	return u, nil
}

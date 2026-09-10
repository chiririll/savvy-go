package auth

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

const (
	RoleAdmin     = "admin"
	RoleReadWrite = "read-write"
	RoleReadOnly  = "read-only"
)

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

func (u User) IsReadOnly() bool { return u.Role == RoleReadOnly }

func (u User) CanWrite() bool { return u.Role == RoleAdmin || u.Role == RoleReadWrite }

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
	DB *sql.DB
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
		role = RoleReadOnly
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var hash any
	if password != nil && *password != "" {
		h, err := HashPassword(*password)
		if err != nil {
			return nil, err
		}
		hash = h
	}
	var pass sql.NullString
	if h, ok := hash.(string); ok {
		pass = db.NS(h)
	}
	res, err := db.Q(s.DB).InsertUser(ctx, sqlc.InsertUserParams{
		Name: name, Email: strings.ToLower(email), Password: pass, Role: role,
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
	var pass any = nil
	setPass := false
	if password != nil && *password != "" {
		h, err := HashPassword(*password)
		if err != nil {
			return nil, err
		}
		pass = h
		setPass = true
	}
	now := time.Now().UTC().Format(time.RFC3339)
	q := db.Q(s.DB)
	if setPass {
		var p sql.NullString
		if h, ok := pass.(string); ok {
			p = db.NS(h)
		}
		err = q.UpdateUserWithPassword(ctx, sqlc.UpdateUserWithPasswordParams{
			Name: u.Name, Email: u.Email, Role: u.Role, Password: p, UpdatedAt: db.NS(now), ID: id,
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
		TwoFactorSecret: db.NullString(secret), TwoFactorEnabled: int64(boolToInt(enabled)),
		TwoFactorConfirmed: int64(boolToInt(confirmed)), UpdatedAt: db.NS(now), ID: id,
	})
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (s Users) Delete(ctx context.Context, id int64) error {
	return db.Q(s.DB).DeleteUser(ctx, id)
}

func (s Users) AdminCount(ctx context.Context) (int, error) {
	n, err := db.Q(s.DB).CountAdmins(ctx, RoleAdmin)
	return int(n), err
}

func userFromRow(r sqlc.User, err error) (*User, error) {
	if err == sql.ErrNoRows {
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
	if t, ok := parseTime(r.CreatedAt); ok {
		u.CreatedAt = &t
	}
	if t, ok := parseTime(r.UpdatedAt); ok {
		u.UpdatedAt = &t
	}
	return u, nil
}

func parseTime(raw sql.NullString) (time.Time, bool) {
	if !raw.Valid || raw.String == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw.String, time.UTC); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

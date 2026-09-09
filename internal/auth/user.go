package auth

import (
	"context"
	"database/sql"
	"strings"
	"time"
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
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s Users) ByID(ctx context.Context, id int64) (*User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, userSelect+` WHERE id = ?`, id))
}

func (s Users) ByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, userSelect+` WHERE lower(email) = ?`, strings.ToLower(email)))
}

func (s Users) All(ctx context.Context) ([]User, error) {
	rows, err := s.DB.QueryContext(ctx, userSelect+` ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
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
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO users (name, email, password, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		name, strings.ToLower(email), hash, role, now, now,
	)
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
	_, err = s.DB.ExecContext(ctx,
		`UPDATE users SET password = ?, updated_at = ? WHERE id = ?`,
		h, time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
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
	if setPass {
		_, err = s.DB.ExecContext(ctx, `
			UPDATE users SET name=?, email=?, role=?, password=?, updated_at=? WHERE id=?`,
			u.Name, u.Email, u.Role, pass, now, id,
		)
	} else {
		_, err = s.DB.ExecContext(ctx, `
			UPDATE users SET name=?, email=?, role=?, updated_at=? WHERE id=?`,
			u.Name, u.Email, u.Role, now, id,
		)
	}
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Users) MarkSSOOnly(ctx context.Context, id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET is_sso_only=1, updated_at=? WHERE id=?`, now, id)
	return err
}

func (s Users) SetRole(ctx context.Context, id int64, role string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET role=?, updated_at=? WHERE id=?`, role, now, id)
	return err
}

func (s Users) SetTwoFactor(ctx context.Context, id int64, secret *string, enabled, confirmed bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx, `
		UPDATE users SET two_factor_secret=?, two_factor_enabled=?, two_factor_confirmed=?, updated_at=? WHERE id=?`,
		secret, boolToInt(enabled), boolToInt(confirmed), now, id)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (s Users) Delete(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s Users) AdminCount(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&n)
	return n, err
}

const userSelect = `SELECT id, name, email, password, role, is_sso_only,
	two_factor_secret, two_factor_enabled, two_factor_confirmed, created_at, updated_at
	FROM users`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*User, error) {
	u, err := scanUserRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func scanUserRow(row rowScanner) (*User, error) {
	var (
		u          User
		password   sql.NullString
		secret     sql.NullString
		sso        int
		tfEnabled  int
		tfConfirm  int
		createdRaw sql.NullString
		updatedRaw sql.NullString
	)
	err := row.Scan(&u.ID, &u.Name, &u.Email, &password, &u.Role, &sso,
		&secret, &tfEnabled, &tfConfirm, &createdRaw, &updatedRaw)
	if err != nil {
		return nil, err
	}
	if password.Valid {
		u.Password = &password.String
	}
	if secret.Valid {
		u.TwoFactorSecret = &secret.String
	}
	u.IsSSOOnly = sso != 0
	u.TwoFactorEnabled = tfEnabled != 0
	u.TwoFactorConfirmed = tfConfirm != 0
	if t, ok := parseTime(createdRaw); ok {
		u.CreatedAt = &t
	}
	if t, ok := parseTime(updatedRaw); ok {
		u.UpdatedAt = &t
	}
	return &u, nil
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

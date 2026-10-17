// Package authrepo is where users, sessions, tokens, second factors and
// passkeys are read from and written to the server database. It knows the
// tables and sqlc; package auth knows the rules and sees plain Go values:
// times as time.Time, optional values as pointers, "no such row" as nil.
package auth

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Repo reads and writes the auth tables of one server database.
type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

// found maps a single-row lookup's "no rows" to a nil result.
func found[T any](v T, err error) (*T, error) {
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func optTime(raw db.NString) *time.Time {
	if t, ok := db.ParseTime(raw); ok {
		return &t
	}
	return nil
}

func optString(raw db.NString) *string {
	if !raw.Valid {
		return nil
	}
	return &raw.String
}

// Users

type User struct {
	ID                 int64
	Name, Email        string
	Password           *string
	Role               string
	SSOOnly            bool
	EmailVerified      bool
	TwoFactorSecret    *string
	TwoFactorEnabled   bool
	TwoFactorConfirmed bool
	Deleted            bool
	CreatedAt          *time.Time
	UpdatedAt          *time.Time
}

func userFrom(r sqlc.User) User {
	return User{
		ID: r.ID, Name: r.Name, Email: r.Email, Role: r.Role,
		Password: optString(r.Password), TwoFactorSecret: optString(r.TwoFactorSecret),
		SSOOnly: r.IsSsoOnly != 0, EmailVerified: r.EmailVerified != 0, TwoFactorEnabled: r.TwoFactorEnabled != 0,
		TwoFactorConfirmed: r.TwoFactorConfirmed != 0, Deleted: r.DeletedAt.Valid,
		CreatedAt: optTime(r.CreatedAt), UpdatedAt: optTime(r.UpdatedAt),
	}
}

func (r Repo) CountUsers(ctx context.Context) (int, error) {
	n, err := r.q().CountUsers(ctx)
	return int(n), err
}

func (r Repo) CountAdmins(ctx context.Context, adminRole string) (int, error) {
	n, err := r.q().CountAdmins(ctx, adminRole)
	return int(n), err
}

func (r Repo) UserByID(ctx context.Context, id int64) (*User, error) {
	row, err := r.q().GetUser(ctx, id)
	return mapFound(row, err, userFrom)
}

func (r Repo) UserByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.q().GetUserByEmail(ctx, email)
	return mapFound(row, err, userFrom)
}

func mapFound[S, T any](row S, err error, conv func(S) T) (*T, error) {
	p, err := found(row, err)
	if p == nil || err != nil {
		return nil, err
	}
	out := conv(*p)
	return &out, nil
}

func (r Repo) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.q().ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]User, len(rows))
	for i, row := range rows {
		out[i] = userFrom(row)
	}
	return out, nil
}

// InsertUser adds a user; an empty hash means no password.
func (r Repo) InsertUser(ctx context.Context, name, email, hash, role string, now time.Time) (int64, error) {
	res, err := r.q().InsertUser(ctx, sqlc.InsertUserParams{
		Name: name, Email: email, Password: db.NullStringVal(hash), Role: role,
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) SetPassword(ctx context.Context, id int64, hash string, now time.Time) error {
	return r.q().
		UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{Password: db.NS(hash), UpdatedAt: db.TimeNS(now), ID: id})
}

// UpdateUser saves the profile; a non-empty hash replaces the password too.
func (r Repo) UpdateUser(
	ctx context.Context,
	id int64,
	name, email, role, hash string,
	verified bool,
	now time.Time,
) error {
	if hash != "" {
		return r.q().UpdateUserWithPassword(ctx, sqlc.UpdateUserWithPasswordParams{
			Name: name, Email: email, Role: role, EmailVerified: db.BoolInt(verified),
			Password: db.NS(hash), UpdatedAt: db.TimeNS(now), ID: id,
		})
	}
	return r.q().UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		Name: name, Email: email, Role: role, EmailVerified: db.BoolInt(verified), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) SetEmailVerified(ctx context.Context, id int64, verified bool, now time.Time) error {
	return r.q().SetUserEmailVerified(ctx, sqlc.SetUserEmailVerifiedParams{
		EmailVerified: db.BoolInt(verified), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// CountVerifiedAdmins counts the live users of adminRole whose email is verified.
func (r Repo) CountVerifiedAdmins(ctx context.Context, adminRole string) (int, error) {
	n, err := r.q().CountVerifiedAdmins(ctx, adminRole)
	return int(n), err
}

func (r Repo) MarkSSOOnly(ctx context.Context, id int64, now time.Time) error {
	return r.q().MarkUserSSOOnly(ctx, sqlc.MarkUserSSOOnlyParams{UpdatedAt: db.TimeNS(now), ID: id})
}

func (r Repo) SetRole(ctx context.Context, id int64, role string, now time.Time) error {
	return r.q().SetUserRole(ctx, sqlc.SetUserRoleParams{Role: role, UpdatedAt: db.TimeNS(now), ID: id})
}

func (r Repo) SetTwoFactor(
	ctx context.Context,
	id int64,
	secret *string,
	enabled, confirmed bool,
	now time.Time,
) error {
	return r.q().SetUserTwoFactor(ctx, sqlc.SetUserTwoFactorParams{
		TwoFactorSecret: db.NullString(secret), TwoFactorEnabled: db.BoolInt(enabled),
		TwoFactorConfirmed: db.BoolInt(confirmed), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// AdvanceTwoFactorStep records step as the last authenticator step used and
// reports whether it was newer than the one recorded. Of two calls with the
// same step, one gets true.
func (r Repo) AdvanceTwoFactorStep(ctx context.Context, id, step int64) (bool, error) {
	res, err := r.q().AdvanceTwoFactorStep(ctx, sqlc.AdvanceTwoFactorStepParams{
		TwoFactorLastStep: db.NullInt64(&step), ID: id, TwoFactorLastStep_2: db.NullInt64(&step),
	})
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ClearTwoFactorStep forgets the last authenticator step, for a new secret.
func (r Repo) ClearTwoFactorStep(ctx context.Context, id int64) error {
	return r.q().ClearTwoFactorStep(ctx, id)
}

// Tombstone removes everything that identifies the user and everything that
// grants access (credentials, tokens, passkeys, SSO links, memberships), and
// keeps the row so ids stored in space databases still resolve.
func (r Repo) Tombstone(ctx context.Context, id int64, now time.Time) error {
	return store.Tx(ctx, r.DB, func(tx store.DB) error {
		q := db.Q(tx)
		for _, del := range []func(context.Context, int64) error{
			q.DeleteUserCredentials, q.DeleteUserAPITokens, q.DeleteUserPasskeys,
			q.DeleteUserIdentities, q.DeleteUserPasswordTokens, q.DeleteUserEmailVerificationTokens, q.DeleteUserRecoveryCodes,
			q.DeleteUserMemberships,
		} {
			if err := del(ctx, id); err != nil {
				return err
			}
		}
		return q.TombstoneUser(ctx, sqlc.TombstoneUserParams{Now: db.TimeNS(now), ID: id})
	})
}

// Sessions

// API tokens

// One-time tokens: password setup and two-factor challenges

// Recovery codes

// Passkeys

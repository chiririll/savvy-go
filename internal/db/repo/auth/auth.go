// Package authrepo is where users, sessions, tokens, second factors and
// passkeys are read from and written to the server database. It knows the
// tables and sqlc; package auth knows the rules and sees plain Go values:
// times as time.Time, optional values as pointers, "no such row" as nil.
package auth

import (
	"context"
	"strings"
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

type Session struct {
	ID, UserID                                   int64
	TokenHash, CSRF                              string
	RememberMe                                   bool
	LastUsedAt, IdleExpiresAt, AbsoluteExpiresAt time.Time
	RefreshedAt, RevokedAt                       *time.Time
	CreatedAt                                    time.Time
}

type NewSession struct {
	UserID          int64
	TokenHash, CSRF string
	IP, UserAgent   string
	RememberMe      bool
	Expires         time.Time
	Now             time.Time
}

func (r Repo) InsertSession(ctx context.Context, s NewSession) (int64, error) {
	res, err := r.q().InsertSession(ctx, sqlc.InsertSessionParams{
		UserID: s.UserID, TokenHash: s.TokenHash, Csrf: s.CSRF,
		Ip: db.NullStringVal(s.IP), UserAgent: db.NullStringVal(s.UserAgent), RememberMe: db.BoolInt(s.RememberMe),
		LastUsedAt: db.FormatTime(s.Now), RefreshedAt: db.TimeNS(s.Now),
		IdleExpiresAt: db.FormatTime(s.Expires), AbsoluteExpiresAt: db.FormatTime(s.Expires),
		CreatedAt: db.TimeNS(s.Now), UpdatedAt: db.TimeNS(s.Now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) SessionByTokenHash(ctx context.Context, hash string) (*Session, error) {
	row0, err := r.q().GetSessionByTokenHash(ctx, hash)
	return mapFound(row0, err, func(row sqlc.GetSessionByTokenHashRow) Session {
		s := Session{
			ID: row.ID, UserID: row.UserID, TokenHash: row.TokenHash, CSRF: row.Csrf, RememberMe: row.RememberMe != 0,
			RefreshedAt: optTime(row.RefreshedAt), RevokedAt: optTime(row.RevokedAt),
		}
		s.LastUsedAt, _ = db.ParseTimeString(row.LastUsedAt)
		s.IdleExpiresAt, _ = db.ParseTimeString(row.IdleExpiresAt)
		s.AbsoluteExpiresAt, _ = db.ParseTimeString(row.AbsoluteExpiresAt)
		s.CreatedAt, _ = db.ParseTime(row.CreatedAt)
		return s
	})
}

func (r Repo) TouchSession(ctx context.Context, id int64, lastUsed, idleExpires time.Time) error {
	return r.q().TouchSession(ctx, sqlc.TouchSessionParams{
		LastUsedAt: db.FormatTime(
			lastUsed,
		), IdleExpiresAt: db.FormatTime(idleExpires), UpdatedAt: db.TimeNS(lastUsed), ID: id,
	})
}

// RotateSession gives a session a new token and a new lifetime.
func (r Repo) RotateSession(ctx context.Context, id int64, tokenHash, csrf string, now, expires time.Time) error {
	return r.q().RefreshSession(ctx, sqlc.RefreshSessionParams{
		TokenHash: tokenHash, Csrf: csrf, LastUsedAt: db.FormatTime(now), RefreshedAt: db.TimeNS(now),
		IdleExpiresAt: db.FormatTime(expires), AbsoluteExpiresAt: db.FormatTime(expires), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) RevokeSession(ctx context.Context, id int64, now time.Time) error {
	return r.q().RevokeSession(ctx, sqlc.RevokeSessionParams{RevokedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), ID: id})
}

func (r Repo) RevokeUserSessions(ctx context.Context, userID int64, now time.Time) error {
	return r.q().
		RevokeUserSessions(ctx, sqlc.RevokeUserSessionsParams{RevokedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), UserID: userID})
}

// RevokeOtherSessions revokes every session of the user but keepID and
// returns how many it revoked.
func (r Repo) RevokeOtherSessions(ctx context.Context, userID, keepID int64, now time.Time) (int, error) {
	res, err := r.q().RevokeOtherSessions(ctx, sqlc.RevokeOtherSessionsParams{
		RevokedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), UserID: userID, ID: keepID,
	})
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// API tokens

type APIToken struct {
	ID, UserID                       int64
	Name, Prefix, Kind               string
	Permissions                      []string
	AllSpaces                        bool
	SpaceIDs                         []int64 // only the token's own list; empty when AllSpaces
	ExpiresAt, LastUsedAt, CreatedAt *time.Time
}

type NewAPIToken struct {
	UserID             int64
	Name, Hash, Prefix string
	Kind               string
	Permissions        []string
	AllSpaces          bool
	SpaceIDs           []int64
	ExpiresAt          *time.Time
	Now                time.Time
}

// InsertAPIToken stores the token and its spaces; call it inside a transaction.
func (r Repo) InsertAPIToken(ctx context.Context, t NewAPIToken) (int64, error) {
	if t.Kind == "" {
		t.Kind = "personal"
	}
	res, err := r.q().InsertAPIToken(ctx, sqlc.InsertAPITokenParams{
		UserID: t.UserID, Name: t.Name, TokenHash: t.Hash, Prefix: t.Prefix, Kind: t.Kind,
		Permissions: strings.Join(t.Permissions, ","), AllSpaces: boolInt(t.AllSpaces),
		ExpiresAt: db.NullTime(t.ExpiresAt), CreatedAt: db.TimeNS(t.Now), UpdatedAt: db.TimeNS(t.Now),
	})
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if !t.AllSpaces {
		for _, sid := range t.SpaceIDs {
			if err := r.q().InsertAPITokenSpace(ctx, sqlc.InsertAPITokenSpaceParams{ApiTokenID: id, SpaceID: sid}); err != nil {
				return 0, err
			}
		}
	}
	return id, nil
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func apiTokenFrom(row sqlc.ApiToken, spaceIDs []int64) APIToken {
	perms := []string{}
	if row.Permissions != "" {
		perms = strings.Split(row.Permissions, ",")
	}
	if spaceIDs == nil {
		spaceIDs = []int64{}
	}
	return APIToken{
		ID: row.ID, UserID: row.UserID, Name: row.Name, Prefix: row.Prefix, Kind: row.Kind,
		Permissions: perms, AllSpaces: row.AllSpaces == 1, SpaceIDs: spaceIDs,
		ExpiresAt: optTime(row.ExpiresAt), LastUsedAt: optTime(row.LastUsedAt), CreatedAt: optTime(row.CreatedAt),
	}
}

func (r Repo) APITokens(ctx context.Context, userID int64) ([]APIToken, error) {
	rows, err := r.q().ListAPITokensByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	links, err := r.q().ListAPITokenSpacesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	spaces := map[int64][]int64{}
	for _, l := range links {
		spaces[l.ApiTokenID] = append(spaces[l.ApiTokenID], l.SpaceID)
	}
	out := make([]APIToken, len(rows))
	for i, row := range rows {
		out[i] = apiTokenFrom(row, spaces[row.ID])
	}
	return out, nil
}

func (r Repo) APITokenByHash(ctx context.Context, hash string) (*APIToken, error) {
	row, err := r.q().GetAPITokenByHash(ctx, hash)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ids, err := r.q().ListAPITokenSpaces(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	t := apiTokenFrom(row, ids)
	return &t, nil
}

func (r Repo) TouchAPIToken(ctx context.Context, id int64, now time.Time) error {
	return r.q().TouchAPIToken(ctx, sqlc.TouchAPITokenParams{LastUsedAt: db.TimeNS(now), ID: id})
}

// UpdateAPITokenGrant replaces what the user's token may do and which spaces it
// reaches, and reports whether they had such a token. Call it inside a
// transaction: the spaces are rewritten too.
func (r Repo) UpdateAPITokenGrant(
	ctx context.Context,
	userID, id int64,
	perms []string,
	allSpaces bool,
	spaceIDs []int64,
	now time.Time,
) (bool, error) {
	res, err := r.q().UpdateAPITokenGrant(ctx, sqlc.UpdateAPITokenGrantParams{
		Permissions: strings.Join(
			perms,
			",",
		), AllSpaces: boolInt(allSpaces), UpdatedAt: db.TimeNS(now), ID: id, UserID: userID,
	})
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}
	if err := r.q().DeleteAPITokenSpaces(ctx, id); err != nil {
		return false, err
	}
	if !allSpaces {
		for _, sid := range spaceIDs {
			if err := r.q().InsertAPITokenSpace(ctx, sqlc.InsertAPITokenSpaceParams{ApiTokenID: id, SpaceID: sid}); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// DeleteAPITokens deletes every token of the user, with their space links and OAuth refresh tokens.
func (r Repo) DeleteAPITokens(ctx context.Context, userID int64) error {
	return r.q().DeleteUserAPITokens(ctx, userID)
}

// DeleteAPIToken reports whether the user had such a token.
func (r Repo) DeleteAPIToken(ctx context.Context, userID, id int64) (bool, error) {
	res, err := r.q().DeleteAPIToken(ctx, sqlc.DeleteAPITokenParams{ID: id, UserID: userID})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// One-time tokens: password setup and two-factor challenges

// OneTime is a stored one-time token found by its hash.
type OneTime struct {
	ID, UserID int64
	ExpiresAt  time.Time
}

func oneTimeFrom(id, userID int64, expires string) OneTime {
	t := OneTime{ID: id, UserID: userID}
	t.ExpiresAt, _ = db.ParseTimeString(expires)
	return t
}

func (r Repo) InsertPasswordToken(ctx context.Context, userID int64, hash string, expires, now time.Time) error {
	return r.q().InsertPasswordToken(ctx, sqlc.InsertPasswordTokenParams{
		UserID: userID, TokenHash: hash, ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

// ConsumePasswordToken marks an unexpired, unused token used and reports
// whether it did.
func (r Repo) ConsumePasswordToken(ctx context.Context, hash string, now time.Time) (bool, error) {
	res, err := r.q().ConsumePasswordToken(ctx, sqlc.ConsumePasswordTokenParams{
		ConsumedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), TokenHash: hash, ExpiresAt: db.FormatTime(now),
	})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r Repo) RevokeActivePasswordTokens(ctx context.Context, userID int64, now time.Time) error {
	return r.q().RevokeActivePasswordTokens(ctx, sqlc.RevokeActivePasswordTokensParams{
		ConsumedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), UserID: userID,
	})
}

// ValidPasswordToken finds an unused token that has not expired at now.
func (r Repo) ValidPasswordToken(ctx context.Context, hash string, now time.Time) (*OneTime, error) {
	row, err := r.q().
		GetValidPasswordToken(ctx, sqlc.GetValidPasswordTokenParams{TokenHash: hash, ExpiresAt: db.FormatTime(now)})
	return mapFound(
		row,
		err,
		func(row sqlc.GetValidPasswordTokenRow) OneTime { return oneTimeFrom(row.ID, row.UserID, row.ExpiresAt) },
	)
}

// PasswordTokenByHash finds a token whether or not it is used or expired.
func (r Repo) PasswordTokenByHash(ctx context.Context, hash string) (*OneTime, error) {
	row, err := r.q().GetPasswordTokenByHash(ctx, hash)
	return mapFound(row, err, func(row sqlc.GetPasswordTokenByHashRow) OneTime {
		return oneTimeFrom(row.ID, row.UserID, row.ExpiresAt)
	})
}

// EmailVerification is the stored link that confirms an address.
type EmailVerification struct {
	UserID    int64
	Email     string
	ExpiresAt time.Time
}

// ReplaceEmailVerificationToken stores the user's confirmation link for an
// address, ending any earlier one.
func (r Repo) ReplaceEmailVerificationToken(
	ctx context.Context,
	userID int64,
	email, hash string,
	expires, now time.Time,
) error {
	return store.Tx(ctx, r.DB, func(tx store.DB) error {
		q := db.Q(tx)
		if err := q.DeleteUserEmailVerificationTokens(ctx, userID); err != nil {
			return err
		}
		return q.InsertEmailVerificationToken(ctx, sqlc.InsertEmailVerificationTokenParams{
			UserID: userID, Email: email, TokenHash: hash, ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now),
		})
	})
}

// EmailVerificationTokenByHash finds a confirmation link, expired or not.
func (r Repo) EmailVerificationTokenByHash(ctx context.Context, hash string) (*EmailVerification, error) {
	row, err := r.q().GetEmailVerificationToken(ctx, hash)
	return mapFound(row, err, func(row sqlc.GetEmailVerificationTokenRow) EmailVerification {
		v := EmailVerification{UserID: row.UserID, Email: row.Email}
		v.ExpiresAt, _ = db.ParseTimeString(row.ExpiresAt)
		return v
	})
}

// ConsumeEmailVerificationToken deletes a confirmation link and reports
// whether there was one, so a link works once.
func (r Repo) ConsumeEmailVerificationToken(ctx context.Context, hash string) (bool, error) {
	res, err := r.q().ConsumeEmailVerificationToken(ctx, hash)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r Repo) InsertChallenge(ctx context.Context, userID int64, hash string, expires, now time.Time) error {
	return r.q().InsertTwoFactorChallenge(ctx, sqlc.InsertTwoFactorChallengeParams{
		UserID: userID, TokenHash: hash, ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

// ChallengeUserID is the user a two-factor challenge was issued to, if it is
// unused and has not expired at now.
func (r Repo) ChallengeUserID(ctx context.Context, hash string, now time.Time) (*int64, error) {
	return found(
		r.q().
			GetValidChallengeUserID(ctx, sqlc.GetValidChallengeUserIDParams{TokenHash: hash, ExpiresAt: db.FormatTime(now)}),
	)
}

// ConsumeChallenge marks an unexpired, unused challenge used and reports
// whether it did.
func (r Repo) ConsumeChallenge(ctx context.Context, hash string, now time.Time) (bool, error) {
	res, err := r.q().ConsumeTwoFactorChallenge(ctx, sqlc.ConsumeTwoFactorChallengeParams{
		ConsumedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), TokenHash: hash, ExpiresAt: db.FormatTime(now),
	})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// FailChallenge counts a wrong code against an open challenge and uses the
// challenge up once it has had max of them. It returns the owner and the wrong
// codes so far; open is false when the challenge is not open.
func (r Repo) FailChallenge(
	ctx context.Context,
	hash string,
	max int,
	now time.Time,
) (userID, attempts int64, open bool, err error) {
	row, err := r.q().FailTwoFactorChallenge(ctx, sqlc.FailTwoFactorChallengeParams{
		MaxAttempts: int64(
			max,
		), ConsumedNow: db.TimeNS(now), UpdatedNow: db.TimeNS(now), TokenHash: hash, ValidAt: db.FormatTime(now),
	})
	if err == db.ErrNotFound {
		return 0, 0, false, nil
	}
	return row.UserID, row.Attempts, err == nil, err
}

// TwoFactorLockedUntil is when the user may get a new second-factor challenge
// again, or nil when no delay is running.
func (r Repo) TwoFactorLockedUntil(ctx context.Context, userID int64) (*time.Time, error) {
	raw, err := r.q().GetTwoFactorLock(ctx, userID)
	if err != nil && err != db.ErrNotFound {
		return nil, err
	}
	return optTime(raw), nil
}

// AddTwoFactorFailure counts a wrong code for the user across challenges and
// returns the count.
func (r Repo) AddTwoFactorFailure(ctx context.Context, userID int64) (int, error) {
	n, err := r.q().AddTwoFactorFailure(ctx, userID)
	return int(n), err
}

func (r Repo) LockTwoFactor(ctx context.Context, userID int64, until time.Time) error {
	return r.q().LockTwoFactor(ctx, sqlc.LockTwoFactorParams{LockedUntil: db.TimeNS(until), UserID: userID})
}

// ResetTwoFactorFailures clears the user's count and any delay after a success.
func (r Repo) ResetTwoFactorFailures(ctx context.Context, userID int64) error {
	return r.q().ResetTwoFactorFailures(ctx, userID)
}

// AnyChallengeUserID is the owner of a challenge whether or not it is used.
func (r Repo) AnyChallengeUserID(ctx context.Context, hash string) (int64, error) {
	return r.q().GetChallengeUserID(ctx, hash)
}

func (r Repo) CountChallenges(ctx context.Context) (int, error) {
	n, err := r.q().CountTwoFactorChallenges(ctx)
	return int(n), err
}

// Recovery codes

func (r Repo) CountUnusedRecoveryCodes(ctx context.Context, userID int64) (int, error) {
	n, err := r.q().CountUnusedRecoveryCodes(ctx, userID)
	return int(n), err
}

// UseRecoveryCode marks the unused code with this hash as used and reports whether there was
// one. Finding and marking are one statement, so two requests cannot both use a code.
func (r Repo) UseRecoveryCode(ctx context.Context, userID int64, hash string, now time.Time) (bool, error) {
	res, err := r.q().UseRecoveryCode(ctx, sqlc.UseRecoveryCodeParams{UsedAt: db.TimeNS(now), UserID: userID, Code: hash})
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r Repo) InvalidateUnusedRecoveryCodes(ctx context.Context, userID int64, now time.Time) error {
	return r.q().
		InvalidateUnusedRecoveryCodes(ctx, sqlc.InvalidateUnusedRecoveryCodesParams{UsedAt: db.TimeNS(now), UserID: userID})
}

func (r Repo) InsertRecoveryCode(ctx context.Context, userID int64, hash string, now time.Time) error {
	return r.q().
		InsertRecoveryCode(ctx, sqlc.InsertRecoveryCodeParams{UserID: userID, Code: hash, CreatedAt: db.TimeNS(now)})
}

func (r Repo) DeleteRecoveryCodes(ctx context.Context, userID int64) error {
	return r.q().DeleteRecoveryCodes(ctx, userID)
}

// Passkeys

// Passkey is a registered credential as the settings page lists it.
type Passkey struct {
	ID                    int64
	Name, AAGUID          *string
	LastUsedAt, CreatedAt *time.Time
}

func (r Repo) Passkeys(ctx context.Context, userID int64) ([]Passkey, error) {
	rows, err := r.q().ListWebAuthnCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Passkey, len(rows))
	for i, row := range rows {
		out[i] = Passkey{ID: row.ID, Name: optString(row.Name), AAGUID: optString(row.Aaguid),
			LastUsedAt: optTime(row.LastUsedAt), CreatedAt: optTime(row.CreatedAt)}
	}
	return out, nil
}

func (r Repo) CountPasskeys(ctx context.Context, userID int64) (int, error) {
	n, err := r.q().CountWebAuthnCredentials(ctx, userID)
	return int(n), err
}

// PasskeyRecords are the serialized credentials the WebAuthn library needs.
func (r Repo) PasskeyRecords(ctx context.Context, userID int64) ([]string, error) {
	return r.q().ListWebAuthnRecords(ctx, userID)
}

type NewPasskey struct {
	UserID                             int64
	CredentialID, Name, AAGUID, Record string
	Counter                            int64
	Now                                time.Time
}

func (r Repo) InsertPasskey(ctx context.Context, p NewPasskey) (int64, error) {
	res, err := r.q().InsertWebAuthnCredential(ctx, sqlc.InsertWebAuthnCredentialParams{
		UserID: p.UserID, CredentialID: p.CredentialID, Name: db.NS(p.Name), Aaguid: db.NS(p.AAGUID),
		Record: p.Record, Counter: p.Counter, CreatedAt: db.TimeNS(p.Now), UpdatedAt: db.TimeNS(p.Now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// RenamePasskey and DeletePasskey report whether the user had such a passkey.
func (r Repo) RenamePasskey(ctx context.Context, userID, id int64, name string, now time.Time) (bool, error) {
	res, err := r.q().RenameWebAuthnCredential(ctx, sqlc.RenameWebAuthnCredentialParams{
		Name: db.NS(name), UpdatedAt: db.TimeNS(now), ID: id, UserID: userID,
	})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r Repo) DeletePasskey(ctx context.Context, userID, id int64) (bool, error) {
	res, err := r.q().DeleteWebAuthnCredential(ctx, sqlc.DeleteWebAuthnCredentialParams{ID: id, UserID: userID})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// UpdatePasskeyAfterLogin stores the credential record and sign count the
// authenticator reported, and marks the passkey as used.
func (r Repo) UpdatePasskeyAfterLogin(
	ctx context.Context,
	credentialID, record string,
	counter int64,
	now time.Time,
) error {
	return r.q().UpdateWebAuthnCredentialAfterLogin(ctx, sqlc.UpdateWebAuthnCredentialAfterLoginParams{
		Record: record, Counter: counter, LastUsedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), CredentialID: credentialID,
	})
}

// PasskeyUserID is the owner of a credential, or nil if it is unknown.
func (r Repo) PasskeyUserID(ctx context.Context, credentialID string) (*int64, error) {
	return found(r.q().GetWebAuthnUserIDByCredential(ctx, credentialID))
}

// PasskeyChallenge is a registration or login ceremony in progress.
type PasskeyChallenge struct {
	UserID  int64 // 0 for a login that does not name the user
	Options string
}

func (r Repo) InsertPasskeyChallenge(
	ctx context.Context,
	userID *int64,
	hash, typ, options string,
	expires, now time.Time,
) error {
	return r.q().InsertWebAuthnChallenge(ctx, sqlc.InsertWebAuthnChallengeParams{
		UserID: db.NullInt64(userID), TokenHash: hash, Type: typ, Options: options,
		ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

// TakePasskeyChallenge returns an open challenge of the type and marks it
// used in one statement, so of two racing calls only one gets it. A challenge
// that is unknown, used or expired is nil; a database error is returned.
func (r Repo) TakePasskeyChallenge(ctx context.Context, hash, typ string, now time.Time) (*PasskeyChallenge, error) {
	row, err := r.q().TakeWebAuthnChallenge(ctx, sqlc.TakeWebAuthnChallengeParams{
		ConsumedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
		TokenHash: hash, Type: typ, ExpiresAt: db.FormatTime(now),
	})
	if err != nil {
		if err == db.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &PasskeyChallenge{UserID: row.UserID.Int64, Options: row.Options}, nil
}

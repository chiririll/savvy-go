package auth

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

func (r Repo) InsertChallenge(ctx context.Context, userID int64, hash string, expires, now time.Time) error {
	return r.q().InsertTwoFactorChallenge(ctx, sqlc.InsertTwoFactorChallengeParams{
		UserID: userID, TokenHash: hash, ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now),
		UpdatedAt: db.TimeNS(now),
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

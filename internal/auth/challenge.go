package auth

import (
	"context"
	"errors"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"
)

const (
	// maxChallengeAttempts is how many wrong codes one challenge takes before it
	// is used up and the caller has to sign in again.
	maxChallengeAttempts = 5
	// lockAfterFailures is how many wrong codes in a row, across challenges, make
	// the account wait. The wait starts at lockBase and doubles with each further
	// wrong code up to lockMax. A right code resets the count.
	lockAfterFailures = 10
	lockBase          = time.Minute
	lockMax           = time.Hour
)

// ErrLocked means too many wrong second-factor codes: no new challenge until
// the delay is over.
var ErrLocked = errors.New("two-factor locked")

type Challenges struct {
	DB store.DB
	// TTL is how long a challenge can be answered.
	TTL time.Duration
}

func (c Challenges) repo() repo.Repo { return repo.Repo{DB: c.DB} }

func (c Challenges) Issue(ctx context.Context, user *User) (string, error) {
	if locked, err := c.locked(ctx, user.ID); err != nil {
		return "", err
	} else if locked {
		return "", ErrLocked
	}
	token := RandomString(48)
	now := time.Now().UTC()
	err := c.repo().InsertChallenge(ctx, user.ID, HashToken(token), now.Add(c.TTL), now)
	return token, err
}

func (c Challenges) Resolve(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, nil
	}
	userID, err := c.repo().ChallengeUserID(ctx, HashToken(token), time.Now().UTC())
	if userID == nil || err != nil {
		return nil, err
	}
	// A challenge issued before the delay began cannot be answered during it.
	if locked, err := c.locked(ctx, *userID); err != nil || locked {
		return nil, err
	}
	return Users{DB: c.DB}.ByID(ctx, *userID)
}

// Fail counts a wrong code against the challenge and returns how many more it
// takes; at 0 the challenge is used up. It also counts toward the user's
// delay. A challenge that is not open has none left.
func (c Challenges) Fail(ctx context.Context, token string) (remaining int, err error) {
	now := time.Now().UTC()
	userID, attempts, open, err := c.repo().FailChallenge(ctx, HashToken(token), maxChallengeAttempts, now)
	if err != nil || !open {
		return 0, err
	}
	failures, err := c.repo().AddTwoFactorFailure(ctx, userID)
	if err != nil {
		return 0, err
	}
	if failures >= lockAfterFailures {
		wait := lockMax
		if shift := failures - lockAfterFailures; shift < 6 {
			wait = min(lockBase<<shift, lockMax)
		}
		if err := c.repo().LockTwoFactor(ctx, userID, now.Add(wait)); err != nil {
			return 0, err
		}
	}
	return max(maxChallengeAttempts-int(attempts), 0), nil
}

func (c Challenges) locked(ctx context.Context, userID int64) (bool, error) {
	until, err := c.repo().TwoFactorLockedUntil(ctx, userID)
	return until != nil && until.After(time.Now()), err
}

func (c Challenges) Consume(ctx context.Context, token string) (*User, error) {
	hash := HashToken(token)
	ok, err := c.repo().ConsumeChallenge(ctx, hash, time.Now().UTC())
	if err != nil || !ok {
		return nil, err
	}
	userID, err := c.repo().AnyChallengeUserID(ctx, hash)
	if err != nil {
		return nil, err
	}
	if err := c.repo().ResetTwoFactorFailures(ctx, userID); err != nil {
		return nil, err
	}
	return Users{DB: c.DB}.ByID(ctx, userID)
}

func (c Challenges) Count(ctx context.Context) (int, error) { return c.repo().CountChallenges(ctx) }

package auth

import (
	"context"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"
)

type Challenges struct {
	DB store.DB
	// TTL is how long a challenge can be answered.
	TTL time.Duration
}

func (c Challenges) repo() repo.Repo { return repo.Repo{DB: c.DB} }

func (c Challenges) Issue(ctx context.Context, user *User) (string, error) {
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
	return Users{DB: c.DB}.ByID(ctx, *userID)
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
	return Users{DB: c.DB}.ByID(ctx, userID)
}

func (c Challenges) Count(ctx context.Context) (int, error) { return c.repo().CountChallenges(ctx) }

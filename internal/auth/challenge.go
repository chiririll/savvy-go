package auth

import (
	"context"
	"database/sql"
	"time"

	"savvy-go/internal/config"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

type Challenges struct {
	DB  *sql.DB
	Cfg config.Config
}

func (c Challenges) Issue(ctx context.Context, user *User) (string, error) {
	token := RandomString(48)
	now := time.Now().UTC()
	exp := now.Add(c.Cfg.ChallengeTTL)
	err := db.Q(c.DB).InsertTwoFactorChallenge(ctx, sqlc.InsertTwoFactorChallengeParams{
		UserID: user.ID, TokenHash: HashToken(token), ExpiresAt: fmtTime(exp),
		CreatedAt: db.NS(fmtTime(now)), UpdatedAt: db.NS(fmtTime(now)),
	})
	return token, err
}

func (c Challenges) Resolve(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, nil
	}
	userID, err := db.Q(c.DB).GetValidChallengeUserID(ctx, sqlc.GetValidChallengeUserIDParams{
		TokenHash: HashToken(token), ExpiresAt: fmtTime(time.Now().UTC()),
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return Users{DB: c.DB}.ByID(ctx, userID)
}

func (c Challenges) Consume(ctx context.Context, token string) (*User, error) {
	now := fmtTime(time.Now().UTC())
	res, err := db.Q(c.DB).ConsumeTwoFactorChallenge(ctx, sqlc.ConsumeTwoFactorChallengeParams{
		ConsumedAt: db.NS(now), UpdatedAt: db.NS(now), TokenHash: HashToken(token), ExpiresAt: now,
	})
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, nil
	}
	userID, err := db.Q(c.DB).GetChallengeUserID(ctx, HashToken(token))
	if err != nil {
		return nil, err
	}
	return Users{DB: c.DB}.ByID(ctx, userID)
}

func (c Challenges) Count(ctx context.Context) (int, error) {
	n, err := db.Q(c.DB).CountTwoFactorChallenges(ctx)
	return int(n), err
}

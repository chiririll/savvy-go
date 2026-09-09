package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/chiririll/savvy-plus/internal/config"
)

type Challenges struct {
	DB  *sql.DB
	Cfg config.Config
}

func (c Challenges) Issue(ctx context.Context, user *User) (string, error) {
	token := RandomString(48)
	now := time.Now().UTC()
	exp := now.Add(c.Cfg.ChallengeTTL)
	_, err := c.DB.ExecContext(ctx, `
		INSERT INTO two_factor_challenges (user_id, token_hash, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		user.ID, HashToken(token), fmtTime(exp), fmtTime(now), fmtTime(now),
	)
	return token, err
}

func (c Challenges) Resolve(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, nil
	}
	var userID int64
	err := c.DB.QueryRowContext(ctx, `
		SELECT user_id FROM two_factor_challenges
		WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?`,
		HashToken(token), fmtTime(time.Now().UTC()),
	).Scan(&userID)
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
	res, err := c.DB.ExecContext(ctx, `
		UPDATE two_factor_challenges SET consumed_at = ?, updated_at = ?
		WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?`,
		now, now, HashToken(token), now,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, nil
	}
	var userID int64
	if err := c.DB.QueryRowContext(ctx,
		`SELECT user_id FROM two_factor_challenges WHERE token_hash = ?`, HashToken(token),
	).Scan(&userID); err != nil {
		return nil, err
	}
	return Users{DB: c.DB}.ByID(ctx, userID)
}

func (c Challenges) Count(ctx context.Context) (int, error) {
	var n int
	err := c.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM two_factor_challenges`).Scan(&n)
	return n, err
}

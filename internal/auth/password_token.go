package auth

import (
	"context"
	"database/sql"
	"time"
)

const passwordTokenTTL = 7 * 24 * time.Hour

type PasswordToken struct {
	ID        int64
	UserID    int64
	ExpiresAt time.Time
	User      *User
}

type PasswordTokens struct {
	DB *sql.DB
}

func (p PasswordTokens) Issue(ctx context.Context, user *User) (token string, expires time.Time, err error) {
	if err = p.RevokeActive(ctx, user.ID); err != nil {
		return "", time.Time{}, err
	}
	token = RandomString(64)
	now := time.Now().UTC()
	expires = now.Add(passwordTokenTTL)
	_, err = p.DB.ExecContext(ctx, `
		INSERT INTO password_tokens (user_id, token_hash, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		user.ID, HashToken(token), fmtTime(expires), fmtTime(now), fmtTime(now),
	)
	return token, expires, err
}

func (p PasswordTokens) Preview(ctx context.Context, token string) (*PasswordToken, error) {
	return p.findValid(ctx, token)
}

func (p PasswordTokens) Consume(ctx context.Context, token string) (*PasswordToken, error) {
	now := fmtTime(time.Now().UTC())
	res, err := p.DB.ExecContext(ctx, `
		UPDATE password_tokens SET consumed_at = ?, updated_at = ?
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
	return p.findHash(ctx, HashToken(token))
}

func (p PasswordTokens) RevokeActive(ctx context.Context, userID int64) error {
	now := fmtTime(time.Now().UTC())
	_, err := p.DB.ExecContext(ctx, `
		UPDATE password_tokens SET consumed_at = ?, updated_at = ?
		WHERE user_id = ? AND consumed_at IS NULL`, now, now, userID)
	return err
}

func (p PasswordTokens) findValid(ctx context.Context, token string) (*PasswordToken, error) {
	return p.scan(ctx, `
		SELECT id, user_id, expires_at FROM password_tokens
		WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?`,
		HashToken(token), fmtTime(time.Now().UTC()),
	)
}

func (p PasswordTokens) findHash(ctx context.Context, hash string) (*PasswordToken, error) {
	return p.scan(ctx, `SELECT id, user_id, expires_at FROM password_tokens WHERE token_hash = ?`, hash)
}

func (p PasswordTokens) scan(ctx context.Context, q string, args ...any) (*PasswordToken, error) {
	var row PasswordToken
	var exp string
	err := p.DB.QueryRowContext(ctx, q, args...).Scan(&row.ID, &row.UserID, &exp)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.ExpiresAt, _ = parseTime(sql.NullString{String: exp, Valid: true})
	u, err := Users{DB: p.DB}.ByID(ctx, row.UserID)
	if err != nil {
		return nil, err
	}
	row.User = u
	return &row, nil
}

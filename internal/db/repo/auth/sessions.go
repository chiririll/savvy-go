package auth

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

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
		RevokeUserSessions(ctx,
			sqlc.RevokeUserSessionsParams{RevokedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), UserID: userID})
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

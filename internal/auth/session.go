package auth

import (
	"context"
	"database/sql"
	"time"

	"savvy-go/internal/config"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

type Session struct {
	ID                int64
	UserID            int64
	TokenHash         string
	CSRF              string
	RememberMe        bool
	LastUsedAt        time.Time
	RefreshedAt       *time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	RevokedAt         *time.Time
	CreatedAt         time.Time
	User              *User
}

func (s *Session) IsActive(now time.Time) bool {
	if s == nil || s.RevokedAt != nil {
		return false
	}
	if now.After(s.IdleExpiresAt) || now.After(s.AbsoluteExpiresAt) {
		return false
	}
	return true
}

type Sessions struct {
	DB  *sql.DB
	Cfg config.Config
}

type Issued struct {
	Token   string
	CSRF    string
	Session *Session
}

func (s Sessions) Issue(ctx context.Context, user *User, ip, ua string, remember bool) (*Issued, error) {
	token := RandomString(48)
	csrf := RandomString(40)
	ttl := s.ttl(remember)
	now := time.Now().UTC()
	exp := now.Add(ttl)
	res, err := db.Q(s.DB).InsertSession(ctx, sqlc.InsertSessionParams{
		UserID: user.ID, TokenHash: HashToken(token), Csrf: csrf,
		Ip: db.NullStringVal(ip), UserAgent: db.NS(truncate(ua, 500)), RememberMe: int64(boolInt(remember)),
		LastUsedAt: fmtTime(now), RefreshedAt: db.NS(fmtTime(now)),
		IdleExpiresAt: fmtTime(exp), AbsoluteExpiresAt: fmtTime(exp),
		CreatedAt: db.NS(fmtTime(now)), UpdatedAt: db.NS(fmtTime(now)),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	sess := &Session{
		ID: id, UserID: user.ID, TokenHash: HashToken(token), CSRF: csrf,
		RememberMe: remember, LastUsedAt: now, IdleExpiresAt: exp, AbsoluteExpiresAt: exp,
		CreatedAt: now, User: user,
	}
	ref := now
	sess.RefreshedAt = &ref
	return &Issued{Token: token, CSRF: csrf, Session: sess}, nil
}

func (s Sessions) Resolve(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, nil
	}
	row, err := db.Q(s.DB).GetSessionByTokenHash(ctx, HashToken(token))
	sess, err := sessionFromRow(row, err)
	if err != nil || sess == nil {
		return sess, err
	}
	if !sess.IsActive(time.Now().UTC()) {
		return nil, nil
	}
	u, err := Users{DB: s.DB}.ByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	sess.User = u
	return sess, nil
}

func (s Sessions) Touch(ctx context.Context, sess *Session) error {
	now := time.Now().UTC()
	idle := now.Add(s.ttl(sess.RememberMe))
	if idle.After(sess.AbsoluteExpiresAt) {
		idle = sess.AbsoluteExpiresAt
	}
	sess.LastUsedAt = now
	sess.IdleExpiresAt = idle
	return db.Q(s.DB).TouchSession(ctx, sqlc.TouchSessionParams{
		LastUsedAt: fmtTime(now), IdleExpiresAt: fmtTime(idle), UpdatedAt: db.NS(fmtTime(now)), ID: sess.ID,
	})
}

// MaybeRefresh rotates remember-me tokens once a day. Returns new raw token+csrf or nil.
func (s Sessions) MaybeRefresh(ctx context.Context, sess *Session) (*Issued, error) {
	if !sess.RememberMe {
		return nil, s.Touch(ctx, sess)
	}
	from := sess.CreatedAt
	if sess.RefreshedAt != nil {
		from = *sess.RefreshedAt
	}
	if time.Now().UTC().Before(from.Add(24 * time.Hour)) {
		return nil, s.Touch(ctx, sess)
	}
	return s.Refresh(ctx, sess)
}

func (s Sessions) Refresh(ctx context.Context, sess *Session) (*Issued, error) {
	token := RandomString(48)
	csrf := RandomString(40)
	now := time.Now().UTC()
	ttl := s.ttl(true)
	exp := now.Add(ttl)
	err := db.Q(s.DB).RefreshSession(ctx, sqlc.RefreshSessionParams{
		TokenHash: HashToken(token), Csrf: csrf, LastUsedAt: fmtTime(now),
		RefreshedAt: db.NS(fmtTime(now)), IdleExpiresAt: fmtTime(exp),
		AbsoluteExpiresAt: fmtTime(exp), UpdatedAt: db.NS(fmtTime(now)), ID: sess.ID,
	})
	if err != nil {
		return nil, err
	}
	sess.TokenHash = HashToken(token)
	sess.CSRF = csrf
	sess.LastUsedAt = now
	sess.RefreshedAt = &now
	sess.IdleExpiresAt = exp
	sess.AbsoluteExpiresAt = exp
	return &Issued{Token: token, CSRF: csrf, Session: sess}, nil
}

func (s Sessions) ClientTimes(sess *Session) (expiresAt string, refreshAt *string) {
	expiresAt = sess.AbsoluteExpiresAt.UTC().Format(time.RFC3339)
	if sess.RememberMe {
		from := sess.CreatedAt
		if sess.RefreshedAt != nil {
			from = *sess.RefreshedAt
		}
		at := from.Add(24 * time.Hour)
		if at.After(sess.AbsoluteExpiresAt) {
			at = sess.AbsoluteExpiresAt
		}
		iso := at.UTC().Format(time.RFC3339)
		refreshAt = &iso
	}
	return expiresAt, refreshAt
}

func (s Sessions) Revoke(ctx context.Context, sess *Session) error {
	now := time.Now().UTC()
	return db.Q(s.DB).RevokeSession(ctx, sqlc.RevokeSessionParams{
		RevokedAt: db.NS(fmtTime(now)), UpdatedAt: db.NS(fmtTime(now)), ID: sess.ID,
	})
}

func (s Sessions) RevokeAll(ctx context.Context, userID int64) error {
	now := time.Now().UTC()
	return db.Q(s.DB).RevokeUserSessions(ctx, sqlc.RevokeUserSessionsParams{
		RevokedAt: db.NS(fmtTime(now)), UpdatedAt: db.NS(fmtTime(now)), UserID: userID,
	})
}

func (s Sessions) RevokeOthers(ctx context.Context, userID, keepID int64) (int, error) {
	now := time.Now().UTC()
	res, err := db.Q(s.DB).RevokeOtherSessions(ctx, sqlc.RevokeOtherSessionsParams{
		RevokedAt: db.NS(fmtTime(now)), UpdatedAt: db.NS(fmtTime(now)), UserID: userID, ID: keepID,
	})
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s Sessions) ttl(remember bool) time.Duration {
	if remember {
		return s.Cfg.RememberTTL
	}
	return s.Cfg.SessionTTL
}

func sessionFromRow(r sqlc.GetSessionByTokenHashRow, err error) (*Session, error) {
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess := &Session{
		ID: r.ID, UserID: r.UserID, TokenHash: r.TokenHash, CSRF: r.Csrf,
		RememberMe: r.RememberMe != 0,
	}
	sess.LastUsedAt, _ = parseTime(sql.NullString{String: r.LastUsedAt, Valid: true})
	if t, ok := parseTime(r.RefreshedAt); ok {
		sess.RefreshedAt = &t
	}
	sess.IdleExpiresAt, _ = parseTime(sql.NullString{String: r.IdleExpiresAt, Valid: true})
	sess.AbsoluteExpiresAt, _ = parseTime(sql.NullString{String: r.AbsoluteExpiresAt, Valid: true})
	if t, ok := parseTime(r.RevokedAt); ok {
		sess.RevokedAt = &t
	}
	if t, ok := parseTime(r.CreatedAt); ok {
		sess.CreatedAt = t
	}
	return sess, nil
}

func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

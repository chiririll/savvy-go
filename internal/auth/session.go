package auth

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
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
	DB store.DB
	// SessionTTL and RememberTTL are how long a session lasts without and with
	// "remember me".
	SessionTTL, RememberTTL time.Duration
}

// refreshEvery is how often a remember-me session rotates its token.
const refreshEvery = 24 * time.Hour

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
		Ip: db.NullStringVal(ip), UserAgent: db.NS(truncate(ua, 500)), RememberMe: db.BoolInt(remember),
		LastUsedAt: db.FormatTime(now), RefreshedAt: db.TimeNS(now),
		IdleExpiresAt: db.FormatTime(exp), AbsoluteExpiresAt: db.FormatTime(exp),
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
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
		LastUsedAt: db.FormatTime(now), IdleExpiresAt: db.FormatTime(idle), UpdatedAt: db.TimeNS(now), ID: sess.ID,
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
	if time.Now().UTC().Before(from.Add(refreshEvery)) {
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
		TokenHash: HashToken(token), Csrf: csrf, LastUsedAt: db.FormatTime(now),
		RefreshedAt: db.TimeNS(now), IdleExpiresAt: db.FormatTime(exp),
		AbsoluteExpiresAt: db.FormatTime(exp), UpdatedAt: db.TimeNS(now), ID: sess.ID,
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

// RefreshDueAt is when a remember-me session should rotate its token, capped
// by its absolute expiry; nil for sessions that do not rotate.
func (s *Session) RefreshDueAt() *time.Time {
	if !s.RememberMe {
		return nil
	}
	from := s.CreatedAt
	if s.RefreshedAt != nil {
		from = *s.RefreshedAt
	}
	at := from.Add(refreshEvery)
	if at.After(s.AbsoluteExpiresAt) {
		at = s.AbsoluteExpiresAt
	}
	return &at
}

func (s Sessions) Revoke(ctx context.Context, sess *Session) error {
	now := time.Now().UTC()
	return db.Q(s.DB).RevokeSession(ctx, sqlc.RevokeSessionParams{
		RevokedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), ID: sess.ID,
	})
}

func (s Sessions) RevokeAll(ctx context.Context, userID int64) error {
	now := time.Now().UTC()
	return db.Q(s.DB).RevokeUserSessions(ctx, sqlc.RevokeUserSessionsParams{
		RevokedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), UserID: userID,
	})
}

func (s Sessions) RevokeOthers(ctx context.Context, userID, keepID int64) (int, error) {
	now := time.Now().UTC()
	res, err := db.Q(s.DB).RevokeOtherSessions(ctx, sqlc.RevokeOtherSessionsParams{
		RevokedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), UserID: userID, ID: keepID,
	})
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s Sessions) ttl(remember bool) time.Duration {
	if remember {
		return s.RememberTTL
	}
	return s.SessionTTL
}

func sessionFromRow(r sqlc.GetSessionByTokenHashRow, err error) (*Session, error) {
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess := &Session{
		ID: r.ID, UserID: r.UserID, TokenHash: r.TokenHash, CSRF: r.Csrf,
		RememberMe: r.RememberMe != 0,
	}
	sess.LastUsedAt, _ = db.ParseTimeString(r.LastUsedAt)
	if t, ok := db.ParseTime(r.RefreshedAt); ok {
		sess.RefreshedAt = &t
	}
	sess.IdleExpiresAt, _ = db.ParseTimeString(r.IdleExpiresAt)
	sess.AbsoluteExpiresAt, _ = db.ParseTimeString(r.AbsoluteExpiresAt)
	if t, ok := db.ParseTime(r.RevokedAt); ok {
		sess.RevokedAt = &t
	}
	if t, ok := db.ParseTime(r.CreatedAt); ok {
		sess.CreatedAt = t
	}
	return sess, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

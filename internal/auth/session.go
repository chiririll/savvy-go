package auth

import (
	"context"
	"time"

	repo "savvy-go/internal/db/repo/auth"
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

func (s Sessions) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Sessions) Issue(ctx context.Context, user *User, ip, ua string, remember bool) (*Issued, error) {
	token := RandomString(48)
	csrf := RandomString(40)
	now := time.Now().UTC()
	exp := now.Add(s.ttl(remember))
	id, err := s.repo().InsertSession(ctx, repo.NewSession{
		UserID: user.ID, TokenHash: HashToken(token), CSRF: csrf, IP: ip, UserAgent: truncate(ua, 500),
		RememberMe: remember, Expires: exp, Now: now,
	})
	if err != nil {
		return nil, err
	}
	ref := now
	sess := &Session{
		ID: id, UserID: user.ID, TokenHash: HashToken(token), CSRF: csrf,
		RememberMe: remember, LastUsedAt: now, RefreshedAt: &ref, IdleExpiresAt: exp, AbsoluteExpiresAt: exp,
		CreatedAt: now, User: user,
	}
	return &Issued{Token: token, CSRF: csrf, Session: sess}, nil
}

func (s Sessions) Resolve(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, nil
	}
	row, err := s.repo().SessionByTokenHash(ctx, HashToken(token))
	if row == nil || err != nil {
		return nil, err
	}
	sess := &Session{
		ID: row.ID, UserID: row.UserID, TokenHash: row.TokenHash, CSRF: row.CSRF, RememberMe: row.RememberMe,
		LastUsedAt: row.LastUsedAt, RefreshedAt: row.RefreshedAt, IdleExpiresAt: row.IdleExpiresAt,
		AbsoluteExpiresAt: row.AbsoluteExpiresAt, RevokedAt: row.RevokedAt, CreatedAt: row.CreatedAt,
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
	return s.repo().TouchSession(ctx, sess.ID, now, idle)
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
	exp := now.Add(s.ttl(true))
	if err := s.repo().RotateSession(ctx, sess.ID, HashToken(token), csrf, now, exp); err != nil {
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
	return s.repo().RevokeSession(ctx, sess.ID, time.Now().UTC())
}

func (s Sessions) RevokeAll(ctx context.Context, userID int64) error {
	return s.repo().RevokeUserSessions(ctx, userID, time.Now().UTC())
}

func (s Sessions) RevokeOthers(ctx context.Context, userID, keepID int64) (int, error) {
	return s.repo().RevokeOtherSessions(ctx, userID, keepID, time.Now().UTC())
}

func (s Sessions) ttl(remember bool) time.Duration {
	if remember {
		return s.RememberTTL
	}
	return s.SessionTTL
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

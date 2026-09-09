package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/chiririll/savvy-plus/internal/config"
)

type Session struct {
	ID                 int64
	UserID             int64
	TokenHash          string
	CSRF               string
	RememberMe         bool
	LastUsedAt         time.Time
	RefreshedAt        *time.Time
	IdleExpiresAt      time.Time
	AbsoluteExpiresAt  time.Time
	RevokedAt          *time.Time
	CreatedAt          time.Time
	User               *User
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
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO auth_sessions (
			user_id, token_hash, csrf, ip, user_agent, remember_me,
			last_used_at, refreshed_at, idle_expires_at, absolute_expires_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, HashToken(token), csrf, nullStr(ip), truncate(ua, 500), boolInt(remember),
		fmtTime(now), fmtTime(now), fmtTime(exp), fmtTime(exp), fmtTime(now), fmtTime(now),
	)
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
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, csrf, remember_me, last_used_at, refreshed_at,
		       idle_expires_at, absolute_expires_at, revoked_at, created_at
		FROM auth_sessions WHERE token_hash = ?`, HashToken(token))
	sess, err := scanSession(row)
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
	_, err := s.DB.ExecContext(ctx, `
		UPDATE auth_sessions SET last_used_at = ?, idle_expires_at = ?, updated_at = ?
		WHERE id = ?`, fmtTime(now), fmtTime(idle), fmtTime(now), sess.ID)
	return err
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
	_, err := s.DB.ExecContext(ctx, `
		UPDATE auth_sessions SET token_hash=?, csrf=?, last_used_at=?, refreshed_at=?,
			idle_expires_at=?, absolute_expires_at=?, updated_at=?
		WHERE id=?`,
		HashToken(token), csrf, fmtTime(now), fmtTime(now), fmtTime(exp), fmtTime(exp), fmtTime(now), sess.ID,
	)
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
	_, err := s.DB.ExecContext(ctx,
		`UPDATE auth_sessions SET revoked_at = ?, updated_at = ? WHERE id = ?`,
		fmtTime(now), fmtTime(now), sess.ID)
	return err
}

func (s Sessions) RevokeAll(ctx context.Context, userID int64) error {
	now := time.Now().UTC()
	_, err := s.DB.ExecContext(ctx,
		`UPDATE auth_sessions SET revoked_at = ?, updated_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		fmtTime(now), fmtTime(now), userID)
	return err
}

func (s Sessions) RevokeOthers(ctx context.Context, userID, keepID int64) (int, error) {
	now := time.Now().UTC()
	res, err := s.DB.ExecContext(ctx, `
		UPDATE auth_sessions SET revoked_at = ?, updated_at = ?
		WHERE user_id = ? AND id != ? AND revoked_at IS NULL`,
		fmtTime(now), fmtTime(now), userID, keepID)
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

func scanSession(row rowScanner) (*Session, error) {
	var (
		sess       Session
		remember   int
		lastUsed   string
		refreshed  sql.NullString
		idle       string
		absolute   string
		revoked    sql.NullString
		created    string
	)
	err := row.Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.CSRF, &remember,
		&lastUsed, &refreshed, &idle, &absolute, &revoked, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess.RememberMe = remember != 0
	sess.LastUsedAt, _ = parseTime(sql.NullString{String: lastUsed, Valid: true})
	if t, ok := parseTime(refreshed); ok {
		sess.RefreshedAt = &t
	}
	sess.IdleExpiresAt, _ = parseTime(sql.NullString{String: idle, Valid: true})
	sess.AbsoluteExpiresAt, _ = parseTime(sql.NullString{String: absolute, Valid: true})
	if t, ok := parseTime(revoked); ok {
		sess.RevokedAt = &t
	}
	sess.CreatedAt, _ = parseTime(sql.NullString{String: created, Valid: true})
	return &sess, nil
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

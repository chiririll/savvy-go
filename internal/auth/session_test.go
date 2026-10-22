package auth

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSessionLifecycle(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	s := Sessions{DB: d, SessionTTL: time.Hour, RememberTTL: 24 * time.Hour * 7}

	issued, err := s.Issue(ctx, u, "1.2.3.4", strings.Repeat("u", 600), false)
	if err != nil || len(issued.Token) != 48 || issued.CSRF == "" {
		t.Fatalf("issue: %+v, %v", issued, err)
	}
	got, err := s.Resolve(ctx, issued.Token)
	if err != nil || got == nil || got.UserID != u.ID || got.User == nil || got.CSRF != issued.CSRF {
		t.Fatalf("resolve: %+v, %v", got, err)
	}
	for _, bad := range []string{"", "nope", issued.Token + "x"} {
		if sess, _ := s.Resolve(ctx, bad); sess != nil {
			t.Errorf("%q resolved", bad)
		}
	}
	if sess, _ := s.Resolve(ctx, issued.CSRF); sess != nil {
		t.Error("the CSRF value is not a session token")
	}

	if err := s.Revoke(ctx, got); err != nil {
		t.Fatal(err)
	}
	if sess, _ := s.Resolve(ctx, issued.Token); sess != nil {
		t.Error("a revoked session resolved")
	}
}

func TestExpiredSessionsDoNotResolve(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	issued, err := (Sessions{DB: d, SessionTTL: -time.Minute}).Issue(ctx, u, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if sess, _ := (Sessions{DB: d}).Resolve(ctx, issued.Token); sess != nil {
		t.Error("an expired session resolved")
	}
}

func TestSessionIsActive(t *testing.T) {
	now := time.Now()
	live := &Session{IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(time.Hour)}
	if !live.IsActive(now) {
		t.Error("live session")
	}
	revoked := *live
	revoked.RevokedAt = &now
	idle := &Session{IdleExpiresAt: now.Add(-time.Second), AbsoluteExpiresAt: now.Add(time.Hour)}
	absolute := &Session{IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(-time.Second)}
	for name, s := range map[string]*Session{"revoked": &revoked, "idle": idle, "absolute": absolute, "nil": nil} {
		if s.IsActive(now) {
			t.Errorf("%s session is active", name)
		}
	}
}

func TestTouchSlidesIdleDeadlineWithinAbsolute(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	s := Sessions{DB: d, SessionTTL: time.Hour, RememberTTL: time.Hour}
	issued, _ := s.Issue(ctx, u, "", "", false)
	sess := issued.Session

	sess.AbsoluteExpiresAt = time.Now().Add(10 * time.Minute)
	if err := s.Touch(ctx, sess); err != nil {
		t.Fatal(err)
	}
	if sess.IdleExpiresAt.After(sess.AbsoluteExpiresAt) {
		t.Errorf("idle %v runs past the absolute %v", sess.IdleExpiresAt, sess.AbsoluteExpiresAt)
	}
}

func TestRememberMeSessionsRotateTokenDaily(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	s := Sessions{DB: d, SessionTTL: time.Hour, RememberTTL: 7 * 24 * time.Hour}

	plain, _ := s.Issue(ctx, u, "", "", false)
	if rotated, err := s.MaybeRefresh(ctx, plain.Session); err != nil || rotated != nil {
		t.Errorf("a plain session never rotates: %v, %v", rotated, err)
	}

	issued, _ := s.Issue(ctx, u, "", "", true)
	if due := issued.Session.RefreshDueAt(); due == nil || due.Before(time.Now().Add(23*time.Hour)) {
		t.Errorf("due at %v", due)
	}
	if rotated, err := s.MaybeRefresh(ctx, issued.Session); err != nil || rotated != nil {
		t.Fatalf("a fresh remember-me session is not due: %v, %v", rotated, err)
	}

	old := time.Now().Add(-25 * time.Hour)
	issued.Session.RefreshedAt = &old
	rotated, err := s.MaybeRefresh(ctx, issued.Session)
	if err != nil || rotated == nil || rotated.Token == issued.Token {
		t.Fatalf("due session did not rotate: %v, %v", rotated, err)
	}
	if sess, _ := s.Resolve(ctx, issued.Token); sess != nil {
		t.Error("the old token still works after rotation")
	}
	if sess, _ := s.Resolve(ctx, rotated.Token); sess == nil || sess.CSRF != rotated.CSRF {
		t.Error("the new token does not resolve")
	}
	if (&Session{}).RefreshDueAt() != nil {
		t.Error("only remember-me sessions have a refresh time")
	}
}

func TestRevokeAllOthers(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	other := newUser(t, d, "b@test.com", RoleUser)
	s := Sessions{DB: d, SessionTTL: time.Hour, RememberTTL: time.Hour}
	keep, _ := s.Issue(ctx, u, "", "", false)
	drop1, _ := s.Issue(ctx, u, "", "", false)
	drop2, _ := s.Issue(ctx, u, "", "", false)
	theirs, _ := s.Issue(ctx, other, "", "", false)

	n, err := s.RevokeOthers(ctx, u.ID, keep.Session.ID)
	if err != nil || n != 2 {
		t.Fatalf("revoked %d, %v", n, err)
	}
	for name, tok := range map[string]string{"keep": keep.Token, "theirs": theirs.Token} {
		if sess, _ := s.Resolve(ctx, tok); sess == nil {
			t.Errorf("%s was revoked", name)
		}
	}
	for _, tok := range []string{drop1.Token, drop2.Token} {
		if sess, _ := s.Resolve(ctx, tok); sess != nil {
			t.Error("another session survived")
		}
	}
	if err := s.RevokeAll(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if sess, _ := s.Resolve(ctx, keep.Token); sess != nil {
		t.Error("RevokeAll left a session")
	}
	if sess, _ := s.Resolve(ctx, theirs.Token); sess == nil {
		t.Error("RevokeAll reached another user")
	}
}

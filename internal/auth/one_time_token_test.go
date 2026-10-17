package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPasswordTokensWorkOnceAndReplaceEachOther(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	pt := PasswordTokens{DB: d}

	first, expires, err := pt.Issue(ctx, u)
	if err != nil || len(first) != 64 || !expires.After(time.Now().Add(6*24*time.Hour)) {
		t.Fatalf("issue: %q %v %v", first, expires, err)
	}
	second, _, _ := pt.Issue(ctx, u)
	if got, _ := pt.Preview(ctx, first); got != nil {
		t.Error("a new link must replace the old one")
	}
	preview, err := pt.Preview(ctx, second)
	if err != nil || preview == nil || preview.User == nil || preview.User.ID != u.ID {
		t.Fatalf("preview: %+v, %v", preview, err)
	}
	if again, _ := pt.Preview(ctx, second); again == nil {
		t.Error("looking at a link must not use it up")
	}
	used, err := pt.Consume(ctx, second)
	if err != nil || used == nil || used.UserID != u.ID {
		t.Fatalf("consume: %+v, %v", used, err)
	}
	if again, _ := pt.Consume(ctx, second); again != nil {
		t.Error("a link worked twice")
	}
	if got, _ := pt.Consume(ctx, "unknown"); got != nil {
		t.Error("an unknown token consumed")
	}

	third, _, _ := pt.Issue(ctx, u)
	if err := pt.RevokeActive(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := pt.Preview(ctx, third); got != nil {
		t.Error("a revoked link works")
	}
}

func TestEmailVerificationLinks(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	u := newUser(t, d, "a@test.com", RoleUser)
	ev := EmailVerifications{DB: d}

	old, _, _ := ev.Issue(ctx, u)
	token, expires, err := ev.Issue(ctx, u)
	if err != nil || len(token) != 40 || !expires.After(time.Now()) {
		t.Fatalf("issue: %v", err)
	}
	if got, _ := ev.Consume(ctx, old); got != nil {
		t.Error("a new link replaces the old one")
	}
	got, err := ev.Consume(ctx, token)
	if err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("consume: %+v, %v", got, err)
	}
	if again, _ := ev.Consume(ctx, token); again != nil {
		t.Error("a link worked twice")
	}

	// A link for an address the user has since left proves nothing.
	stale, _, _ := ev.Issue(ctx, u)
	if _, err := users.Update(ctx, u.ID, nil, ptr("new@test.com"), nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := ev.Consume(ctx, stale); got != nil {
		t.Error("a link for the old address verified the new one")
	}
}

func TestChallengesAreSingleUseAndExpire(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	ch := Challenges{DB: d, TTL: time.Minute}

	token, err := ch.Issue(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := ch.Resolve(ctx, token); got == nil || got.ID != u.ID {
		t.Error("resolve")
	}
	if got, _ := ch.Resolve(ctx, token); got == nil {
		t.Error("resolving does not use the challenge up (a wrong code can be retried)")
	}
	if got, _ := ch.Resolve(ctx, ""); got != nil {
		t.Error("empty token resolved")
	}
	if got, _ := ch.Consume(ctx, token); got == nil || got.ID != u.ID {
		t.Fatal("consume")
	}
	if got, _ := ch.Consume(ctx, token); got != nil {
		t.Error("a challenge was consumed twice")
	}
	if got, _ := ch.Resolve(ctx, token); got != nil {
		t.Error("a consumed challenge still resolves")
	}

	expired, _ := Challenges{DB: d, TTL: -time.Second}.Issue(ctx, u)
	if got, _ := ch.Resolve(ctx, expired); got != nil {
		t.Error("an expired challenge resolved")
	}
	if got, _ := ch.Consume(ctx, expired); got != nil {
		t.Error("an expired challenge was consumed")
	}
}

func TestChallengeDiesAfterFiveWrongCodes(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	ch := Challenges{DB: d, TTL: time.Minute}
	token, _ := ch.Issue(ctx, u)

	for i := 1; i <= 5; i++ {
		if got, _ := ch.Resolve(ctx, token); got == nil {
			t.Fatalf("challenge died before failure %d", i)
		}
		left, err := ch.Fail(ctx, token)
		if err != nil || left != 5-i {
			t.Fatalf("failure %d: remaining %d, %v", i, left, err)
		}
	}
	if got, _ := ch.Resolve(ctx, token); got != nil {
		t.Error("a challenge with five wrong codes still resolves")
	}
	if got, _ := ch.Consume(ctx, token); got != nil {
		t.Error("a used-up challenge was consumed")
	}
	if left, _ := ch.Fail(ctx, token); left != 0 {
		t.Errorf("failing a dead challenge left %d", left)
	}
}

func TestFailuresAcrossChallengesLockTheUser(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	ch := Challenges{DB: d, TTL: time.Minute}

	var last string
	for range 2 {
		last, _ = ch.Issue(ctx, u)
		for range 5 {
			_, _ = ch.Fail(ctx, last)
		}
	}
	if _, err := ch.Issue(ctx, u); !errors.Is(err, ErrLocked) {
		t.Fatalf("issue after 10 failures: %v", err)
	}

	// Another user is not affected.
	other := newUser(t, d, "b@test.com", RoleUser)
	if _, err := ch.Issue(ctx, other); err != nil {
		t.Fatalf("other user locked: %v", err)
	}

	// A challenge issued before the lock cannot be answered while it lasts.
	open, _ := ch.Issue(ctx, other)
	_, _ = d.ExecContext(ctx, `INSERT INTO two_factor_failures (user_id, failures, locked_until) VALUES (?, 10, ?)`,
		other.ID, time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano))
	if got, _ := ch.Resolve(ctx, open); got != nil {
		t.Error("a locked user resolved a challenge")
	}

	// Once the delay is over, a challenge can be issued again.
	_, _ = d.ExecContext(ctx, `UPDATE two_factor_failures SET locked_until = ?`, time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano))
	token, err := ch.Issue(ctx, u)
	if err != nil {
		t.Fatalf("issue after the delay: %v", err)
	}
	// The count was kept, so the next wrong code locks again for longer.
	_, _ = ch.Fail(ctx, token)
	if _, err := ch.Issue(ctx, u); !errors.Is(err, ErrLocked) {
		t.Fatalf("issue after another failure: %v", err)
	}
}

func TestSuccessResetsTheFailureCount(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	ch := Challenges{DB: d, TTL: time.Minute}

	token, _ := ch.Issue(ctx, u)
	for range 4 {
		_, _ = ch.Fail(ctx, token)
	}
	if got, _ := ch.Consume(ctx, token); got == nil {
		t.Fatal("consume")
	}
	for range 2 {
		tok, err := ch.Issue(ctx, u)
		if err != nil {
			t.Fatalf("issue: %v", err)
		}
		for range 4 {
			_, _ = ch.Fail(ctx, tok)
		}
	}
	if _, err := ch.Issue(ctx, u); err != nil {
		t.Fatalf("8 failures after a success locked the user: %v", err)
	}
}

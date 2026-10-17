package auth

import (
	"context"
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

package auth

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"savvy-go/internal/auth/totp"
)

func TestTwoFactorLifecycle(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	tf := TwoFactor{DB: d, Users: users}
	reload := func(id int64) *User { u, _ := users.ByID(ctx, id); return u }
	u := newUser(t, d, "a@test.com", RoleUser)

	if _, err := tf.Confirm(ctx, u, "123456"); err == nil {
		t.Error("nothing to confirm before enabling")
	}
	secret, uri, err := tf.Enable(ctx, u)
	if err != nil || secret == "" || !strings.Contains(uri, secret) {
		t.Fatalf("enable: %q %q %v", secret, uri, err)
	}
	u = reload(u.ID)
	if u.HasTwoFactor() {
		t.Fatal("two-factor counts only once confirmed")
	}
	if _, err := tf.Confirm(ctx, u, "000000"); err == nil {
		t.Error("a wrong code confirmed")
	}
	// Each authenticator code works once, so the steps below use the previous,
	// the current and the next code, which the 90 second window accepts.
	codes, err := tf.Confirm(ctx, u, totp.At(secret, time.Now().Add(-30*time.Second)))
	if err != nil || len(codes) != 8 {
		t.Fatalf("confirm: %v, %v", codes, err)
	}
	u = reload(u.ID)
	if !u.HasTwoFactor() {
		t.Fatal("confirmed")
	}
	if _, _, err := tf.Enable(ctx, u); err == nil {
		t.Error("enabling twice")
	}
	if n := tf.Remaining(ctx, u.ID); n != 8 {
		t.Errorf("remaining %d", n)
	}

	// A recovery code works once; case and spaces do not matter.
	if !tf.VerifyAny(ctx, u, " "+strings.ToUpper(codes[0])+" ") {
		t.Error("a recovery code did not verify")
	}
	if tf.VerifyAny(ctx, u, codes[0]) {
		t.Error("a recovery code verified twice")
	}
	if tf.ConsumeRecovery(ctx, u.ID, "aaaa-bbbb") {
		t.Error("an invented code verified")
	}
	if n := tf.Remaining(ctx, u.ID); n != 7 {
		t.Errorf("remaining after one use: %d", n)
	}
	if !tf.VerifyAny(ctx, u, totp.Now(secret)) {
		t.Error("the authenticator code verifies too")
	}

	// New codes replace the unused ones, and need the authenticator itself.
	if _, err := tf.Regenerate(ctx, u, codes[1]); err == nil {
		t.Error("a recovery code cannot mint more recovery codes")
	}
	fresh, err := tf.Regenerate(ctx, u, totp.At(secret, time.Now().Add(30*time.Second)))
	if err != nil || len(fresh) != 8 {
		t.Fatalf("regenerate: %v, %v", fresh, err)
	}
	if tf.ConsumeRecovery(ctx, u.ID, codes[1]) {
		t.Error("an old code works after regenerating")
	}
	if n := tf.Remaining(ctx, u.ID); n != 8 {
		t.Errorf("remaining after regenerating: %d", n)
	}
	if !tf.ConsumeRecovery(ctx, u.ID, fresh[0]) {
		t.Error("a new code does not work")
	}

	if err := tf.Disable(ctx, u, "000000"); err == nil {
		t.Error("disabled with a wrong code")
	}
	if err := tf.Disable(ctx, u, fresh[1]); err != nil {
		t.Fatalf("a recovery code may disable two-factor: %v", err)
	}
	u = reload(u.ID)
	if u.HasTwoFactor() || u.TwoFactorSecret != nil {
		t.Errorf("after disabling %+v", u)
	}
	if n := tf.Remaining(ctx, u.ID); n != 0 {
		t.Errorf("recovery codes remain after disabling: %d", n)
	}
	if _, err := tf.Regenerate(ctx, u, "123456"); err == nil {
		t.Error("regenerating without two-factor")
	}
	if err := tf.Disable(ctx, u, "123456"); err == nil {
		t.Error("disabling what is not enabled")
	}
}

func TestRecoveryCodesLookRight(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		c := recoveryCode()
		if len(c) != 9 || c[4] != '-' {
			t.Fatalf("code %q", c)
		}
		if strings.ContainsAny(c, "ilo01") {
			t.Fatalf("%q has characters that are easy to misread", c)
		}
		seen[c] = true
	}
	if len(seen) < 190 {
		t.Errorf("only %d distinct codes in 200", len(seen))
	}
}

func TestRandomCharactersAreUniform(t *testing.T) {
	for _, alphabet := range []string{recoveryChars, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"} {
		const n = 200000
		counts := map[rune]int{}
		for _, c := range randomFrom(alphabet, n) {
			counts[c]++
		}
		want := float64(n) / float64(len(alphabet))
		var chi float64
		for _, c := range alphabet {
			d := float64(counts[c]) - want
			chi += d * d / want
		}
		// Degrees of freedom are len-1 (30 or 61); 120 is far above the 99.99th
		// percentile for both, so a fair generator never trips it.
		if len(counts) != len(alphabet) || chi > 120 {
			t.Errorf("alphabet of %d: chi-square %.1f, %d distinct", len(alphabet), chi, len(counts))
		}
	}
}

func TestEnableDropsRecoveryCodesOfEarlierSetup(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	tf := TwoFactor{DB: d, Users: Users{DB: d}}
	u := newUser(t, d, "a@test.com", RoleUser)
	if _, err := generateCodes(ctx, tf.repo(), u.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := tf.Enable(ctx, u); err != nil {
		t.Fatal(err)
	}
	if n := tf.Remaining(ctx, u.ID); n != 0 {
		t.Errorf("%d stale recovery codes survive a new setup", n)
	}
}

func TestRecoveryCodesAreStoredAsHashes(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	tf := TwoFactor{DB: d, Users: users}
	a := newUser(t, d, "a@test.com", RoleUser)
	b := newUser(t, d, "b@test.com", RoleUser)
	codes, err := generateCodes(ctx, tf.repo(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := d.QueryRowContext(ctx, `SELECT code FROM two_factor_recovery_codes WHERE user_id = ? LIMIT 1`, a.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == codes[0] || strings.Contains(stored, "-") || len(stored) != 64 {
		t.Errorf("stored %q: want a SHA-256 hex digest, not the code", stored)
	}
	if tf.ConsumeRecovery(ctx, b.ID, codes[1]) {
		t.Error("a code opened another user's account")
	}
	if !tf.ConsumeRecovery(ctx, a.ID, codes[1]) || tf.ConsumeRecovery(ctx, a.ID, codes[1]) {
		t.Error("a code must verify once, then fail")
	}
}

func TestRecoveryCodeCanBeUsedOnceWhenRequestsRace(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	tf := TwoFactor{DB: d, Users: users}
	u := newUser(t, d, "race@test.com", RoleUser)
	secret, _, err := tf.Enable(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	u, _ = users.ByID(ctx, u.ID)
	codes, err := tf.Confirm(ctx, u, totp.Now(secret))
	if err != nil {
		t.Fatal(err)
	}

	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tf.ConsumeRecovery(ctx, u.ID, codes[0]) {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("one recovery code succeeded %d times", wins.Load())
	}
}

func TestChallengeConsumeSucceedsOnce(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "chal@test.com", RoleUser)
	ch := Challenges{DB: d, TTL: time.Minute}
	tok, err := ch.Issue(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, err := ch.Consume(ctx, tok); err == nil && got != nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("one challenge was consumed %d times", wins.Load())
	}
}

func TestAuthenticatorCodeWorksOnce(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	tf := TwoFactor{DB: d, Users: users}
	u := newUser(t, d, "replay@test.com", RoleUser)
	secret, _, err := tf.Enable(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	u, _ = users.ByID(ctx, u.ID)
	now := time.Now()
	prev, cur, next := totp.At(
		secret,
		now.Add(-30*time.Second),
	), totp.At(
		secret,
		now,
	), totp.At(
		secret,
		now.Add(30*time.Second),
	)

	if _, err := tf.Confirm(ctx, u, cur); err != nil {
		t.Fatal(err)
	}
	u, _ = users.ByID(ctx, u.ID)
	if tf.VerifyAny(ctx, u, cur) {
		t.Error("the code that confirmed two-factor worked again")
	}
	if tf.VerifyAny(ctx, u, prev) {
		t.Error("a code from an older step than the last accepted one worked")
	}
	if !tf.VerifyAny(ctx, u, next) {
		t.Error("a code from a newer step did not work")
	}
	if tf.VerifyAny(ctx, u, next) {
		t.Error("a code worked twice")
	}
}

func TestAuthenticatorCodeWorksOnceWhenRequestsRace(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	tf := TwoFactor{DB: d, Users: users}
	u := newUser(t, d, "raceotp@test.com", RoleUser)
	secret, _, err := tf.Enable(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	u, _ = users.ByID(ctx, u.ID)
	if _, err := tf.Confirm(ctx, u, totp.At(secret, time.Now().Add(-30*time.Second))); err != nil {
		t.Fatal(err)
	}
	u, _ = users.ByID(ctx, u.ID)
	code := totp.Now(secret)

	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tf.VerifyAny(ctx, u, code) {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("one code was accepted %d times", wins.Load())
	}
}

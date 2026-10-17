package auth

import (
	"context"
	"strings"
	"testing"
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
	codes, err := tf.Confirm(ctx, u, TOTPNow(secret))
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
	if !tf.VerifyAny(ctx, u, TOTPNow(secret)) {
		t.Error("the authenticator code verifies too")
	}

	// New codes replace the unused ones, and need the authenticator itself.
	if _, err := tf.Regenerate(ctx, u, codes[1]); err == nil {
		t.Error("a recovery code cannot mint more recovery codes")
	}
	fresh, err := tf.Regenerate(ctx, u, TOTPNow(secret))
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

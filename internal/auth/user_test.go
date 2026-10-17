package auth

import (
	"context"
	"testing"
	"time"
)

func TestUsersCreateAndFind(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	u, err := users.Create(ctx, "Ann", "Ann@Test.COM", ptr("pw-12345"), "")
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "ann@test.com" || u.Role != RoleUser || u.EmailVerified || u.IsInactive() {
		t.Errorf("created %+v", u)
	}
	if u.Password == nil || !CheckPassword(*u.Password, "pw-12345") {
		t.Error("the password is stored hashed")
	}
	byMail, err := users.ByEmail(ctx, "ANN@test.com")
	if err != nil || byMail == nil || byMail.ID != u.ID {
		t.Errorf("lookup by email ignores case: %v, %v", byMail, err)
	}
	if got, err := users.ByEmail(ctx, "nobody@test.com"); got != nil || err != nil {
		t.Errorf("unknown email: %v, %v", got, err)
	}
	if got, err := users.ByID(ctx, 9999); got != nil || err != nil {
		t.Errorf("unknown id: %v, %v", got, err)
	}
	if _, err := users.Create(ctx, "Dup", "ann@test.com", nil, ""); err == nil {
		t.Error("a second account with the same address was created")
	}
	noPass, err := users.Create(ctx, "SSO", "sso@test.com", nil, RoleGuest)
	if err != nil || !noPass.IsInactive() || !noPass.IsGuest() {
		t.Errorf("an account without a password is inactive: %+v, %v", noPass, err)
	}
	if n, _ := users.Count(ctx); n != 2 {
		t.Errorf("count %d", n)
	}
	all, err := users.All(ctx)
	if err != nil || len(all) != 2 {
		t.Errorf("All: %d, %v", len(all), err)
	}
}

func TestUpdateReverifiesAChangedAddress(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	u := newUser(t, d, "a@test.com", RoleUser)
	if err := users.SetEmailVerified(ctx, u.ID, true); err != nil {
		t.Fatal(err)
	}

	same, err := users.Update(ctx, u.ID, ptr("New Name"), ptr("A@Test.com"), nil, nil, nil)
	if err != nil || !same.EmailVerified || same.Name != "New Name" {
		t.Fatalf("the same address in another case stays verified: %+v, %v", same, err)
	}
	moved, _ := users.Update(ctx, u.ID, nil, ptr("b@test.com"), nil, nil, nil)
	if moved.EmailVerified || moved.Email != "b@test.com" {
		t.Errorf("a new address has not been proven: %+v", moved)
	}
	vouched, _ := users.Update(ctx, u.ID, nil, ptr("c@test.com"), nil, nil, ptr(true))
	if !vouched.EmailVerified {
		t.Error("an admin may vouch for the new address")
	}
}

func TestUpdateKeepsThePasswordUnlessGivenOne(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	u := newUser(t, d, "a@test.com", RoleUser)

	kept, _ := users.Update(ctx, u.ID, ptr("x"), nil, nil, ptr(""), nil)
	if !CheckPassword(*kept.Password, "correct horse") {
		t.Error("an empty password means no change")
	}
	changed, _ := users.Update(ctx, u.ID, nil, nil, ptr(RoleAdmin), ptr("brand new"), nil)
	if !CheckPassword(*changed.Password, "brand new") || changed.Role != RoleAdmin {
		t.Errorf("changed %+v", changed)
	}
	if err := users.UpdatePassword(ctx, u.ID, "third"); err != nil {
		t.Fatal(err)
	}
	if again, _ := users.ByID(ctx, u.ID); !CheckPassword(*again.Password, "third") {
		t.Error("UpdatePassword")
	}
}

func TestAdminCountsAndRoles(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	a := newUser(t, d, "a@test.com", RoleAdmin)
	b := newUser(t, d, "b@test.com", RoleAdmin)
	newUser(t, d, "c@test.com", RoleUser)

	if n, _ := users.AdminCount(ctx); n != 2 {
		t.Errorf("admins %d", n)
	}
	if n, _ := users.VerifiedAdminCount(ctx); n != 0 {
		t.Errorf("verified admins %d", n)
	}
	if err := users.SetEmailVerified(ctx, a.ID, true); err != nil {
		t.Fatal(err)
	}
	if n, _ := users.VerifiedAdminCount(ctx); n != 1 {
		t.Errorf("verified admins %d", n)
	}
	if err := users.SetRole(ctx, b.ID, RoleUser); err != nil {
		t.Fatal(err)
	}
	if n, _ := users.AdminCount(ctx); n != 1 {
		t.Errorf("admins after demoting one: %d", n)
	}
	if got, _ := users.ByID(ctx, b.ID); got.IsAdmin() {
		t.Error("role not changed")
	}
}

func TestDeleteTombstonesTheUser(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	users := Users{DB: d}
	u := newUser(t, d, "gone@test.com", RoleUser)
	sessions := Sessions{DB: d, SessionTTL: time.Hour, RememberTTL: time.Hour}
	issued, _ := sessions.Issue(ctx, u, "1.2.3.4", "ua", false)
	raw, _, _ := APITokens{DB: d}.Issue(ctx, u, "t", Grant{Permissions: []string{"reports:read"}, AllSpaces: true}, nil)

	if err := users.Delete(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	got, err := users.ByID(ctx, u.ID)
	if err != nil || got == nil || !got.Deleted || got.Email == "gone@test.com" || !got.IsInactive() {
		t.Errorf("the row stays so old ids resolve, without credentials: %+v, %v", got, err)
	}
	if byMail, _ := users.ByEmail(ctx, "gone@test.com"); byMail != nil {
		t.Error("the address is free again")
	}
	if tok, _ := (APITokens{DB: d}).Resolve(ctx, raw); tok != nil {
		t.Error("the API tokens of a deleted user are gone")
	}
	if sess, _ := sessions.Resolve(ctx, issued.Token); sess != nil && sess.User != nil && !sess.User.Deleted {
		t.Error("a deleted user's session still resolves to a live account")
	}
	if _, err := users.Create(ctx, "New", "gone@test.com", nil, ""); err != nil {
		t.Errorf("the address can be reused: %v", err)
	}
}

func TestRoles(t *testing.T) {
	for role, ok := range map[string]bool{RoleAdmin: true, RoleUser: true, RoleGuest: true, "read-only": false, "": false, "root": false} {
		if ValidRole(role) != ok {
			t.Errorf("ValidRole(%q) = %v", role, !ok)
		}
	}
	for in, want := range map[string]string{"read-write": RoleUser, "read-only": RoleUser, RoleAdmin: RoleAdmin, RoleGuest: RoleGuest} {
		if got := NormalizeRole(in); got != want {
			t.Errorf("NormalizeRole(%q) = %q", in, got)
		}
	}
}

package sso

import (
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/totp"
)

func (e *ssoEnv) admin(t *testing.T) *auth.User {
	t.Helper()
	u, err := e.svc.Users.Create(e.ctx, "Admin", "root@test.com", ptr("pw-pw-pw"), auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func ident(sub, email string, verified bool) NormalizedIdentity {
	return NormalizedIdentity{
		Subject:       sub,
		Email:         email,
		EmailVerified: verified,
		Name:          "Ann",
		Raw:           map[string]any{"sub": sub},
	}
}

func TestProvisionNeedsSubjectAdminToExist(t *testing.T) {
	e := newSSOEnv(t, nil)
	if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("", "a@test.com", true)); ssoCode(err) != "missing_subject" {
		t.Errorf("no subject: %q", ssoCode(err))
	}
	if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("s", "a@test.com", true)); ssoCode(err) != "no_admin" {
		t.Errorf("an empty server must be set up locally first: %q", ssoCode(err))
	}
}

func TestProvisionCreatesSSOUserFirstLogin(t *testing.T) {
	e := newSSOEnv(t, nil)
	e.admin(t)
	u, needs2FA, err := e.svc.Provision(e.ctx, *e.p, ident("s1", "New@Test.com", true))
	if err != nil || u == nil {
		t.Fatal(err)
	}
	if needs2FA || u.Email != "new@test.com" || !u.IsSSOOnly || !u.EmailVerified || u.Role != auth.RoleUser {
		t.Errorf("user %+v", u)
	}
	// "read-only" is a retired role name; it means user.
	again, _, err := e.svc.Provision(e.ctx, *e.p, ident("s1", "changed@test.com", true))
	if err != nil || again.ID != u.ID {
		t.Fatalf("the second login finds the same user by subject, whatever the address now says: %v, %v", again, err)
	}
	if n, _ := e.svc.Users.Count(e.ctx); n != 2 {
		t.Errorf("%d users", n)
	}
}

func TestProvisionLeavesAnUnverifiedAddressUnverified(t *testing.T) {
	e := newSSOEnv(t, nil)
	e.admin(t)
	u, _, err := e.svc.Provision(e.ctx, *e.p, ident("s1", "new@test.com", false))
	if err != nil || u.EmailVerified {
		t.Errorf("%+v, %v", u, err)
	}
}

// An unverified address must never take over an existing account.
func TestProvisionLinksEmailOnlyWhenVerified(t *testing.T) {
	e := newSSOEnv(t, nil)
	victim := e.admin(t)

	if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("attacker", "root@test.com", false)); ssoCode(
		err,
	) != "email_in_use" {
		t.Fatalf("an unverified claim to root@test.com: %q", ssoCode(err))
	}

	u, _, err := e.svc.Provision(e.ctx, *e.p, ident("owner", "ROOT@test.com", true))
	if err != nil || u.ID != victim.ID {
		t.Fatalf("a verified address links to the existing account: %+v, %v", u, err)
	}
	if !u.EmailVerified {
		t.Error("the provider's word makes the local address verified")
	}
}

func TestProvisionWithoutEmailLinkKeepsApart(t *testing.T) {
	e := newSSOEnv(t, func(in *IdPWrite) { in.LinkByEmail = ptr(false) })
	e.admin(t)
	if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("s", "root@test.com", true)); ssoCode(err) != "email_in_use" {
		t.Errorf("error %q", ssoCode(err))
	}
}

func TestProvisionSignupRules(t *testing.T) {
	t.Run("the provider forbids signup", func(t *testing.T) {
		e := newSSOEnv(t, func(in *IdPWrite) { in.AllowJIT = ptr(false) })
		e.admin(t)
		if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("s", "a@test.com", true)); ssoCode(err) != "signup_disabled" {
			t.Errorf("%q", ssoCode(err))
		}
	})
	t.Run("the server forbids signup", func(t *testing.T) {
		e := newSSOEnv(t, nil)
		e.admin(t)
		if err := e.svc.Settings.Set(e.ctx, "sso_allow_signup", false); err != nil {
			t.Fatal(err)
		}
		if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("s", "a@test.com", true)); ssoCode(err) != "signup_disabled" {
			t.Errorf("%q", ssoCode(err))
		}
	})
	t.Run("the server wants a verified address", func(t *testing.T) {
		e := newSSOEnv(t, nil)
		e.admin(t)
		if err := e.svc.Settings.Set(e.ctx, "sso_require_verified_email", true); err != nil {
			t.Fatal(err)
		}
		if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("s", "a@test.com", false)); ssoCode(err) != "email_unverified" {
			t.Errorf("%q", ssoCode(err))
		}
		if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("s", "a@test.com", true)); err != nil {
			t.Errorf("a verified address is fine: %v", err)
		}
	})
	t.Run("no address", func(t *testing.T) {
		e := newSSOEnv(t, nil)
		e.admin(t)
		if _, _, err := e.svc.Provision(e.ctx, *e.p, ident("s", "", false)); ssoCode(err) != "no_email" {
			t.Errorf("%q", ssoCode(err))
		}
	})
	t.Run("a name is optional", func(t *testing.T) {
		e := newSSOEnv(t, nil)
		e.admin(t)
		id := ident("s", "a@test.com", true)
		id.Name = ""
		if u, _, err := e.svc.Provision(e.ctx, *e.p, id); err != nil || u.Name != "a@test.com" {
			t.Errorf("%+v, %v", u, err)
		}
	})
}

func TestProvisionGivesTheMappedDefaultRole(t *testing.T) {
	mapping := []any{
		map[string]any{"claim": "groups", "operator": "contains", "value": "admins", "role": "admin"},
		map[string]any{"claim": "groups", "operator": "contains", "value": "guests", "role": "guest"},
	}
	tests := []struct {
		name   string
		role   string
		groups []string
		want   string
	}{
		{"a mapped group", "user", []string{"admins"}, auth.RoleAdmin},
		{"the first matching rule wins", "user", []string{"guests", "admins"}, auth.RoleAdmin},
		{"no group matches: the default", "guest", []string{"other"}, auth.RoleGuest},
		{"a bad default is a plain user", "superuser", nil, auth.RoleUser},
	}
	for _, tc := range tests {
		e := newSSOEnv(t, func(in *IdPWrite) { in.RoleMapping, in.DefaultRole = mapping, tc.role })
		e.admin(t)
		id := ident("s", "a@test.com", true)
		id.Groups = tc.groups
		u, _, err := e.svc.Provision(e.ctx, *e.p, id)
		if err != nil || u.Role != tc.want {
			t.Errorf("%s: role %v (%v), want %s", tc.name, u, err, tc.want)
		}
	}
}

func TestProvisionSyncsRolesKeepsLastAdmin(t *testing.T) {
	mapping := []any{map[string]any{"claim": "groups", "operator": "contains", "value": "admins", "role": "admin"}}
	e := newSSOEnv(t, func(in *IdPWrite) {
		in.RoleMapping, in.SyncRoleOnLogin, in.DefaultRole = mapping, ptr(true), "user"
	})
	root := e.admin(t)

	// The only admin signs in through SSO without the admin group: stays admin.
	id := ident("root-sub", "root@test.com", true)
	u, _, err := e.svc.Provision(e.ctx, *e.p, id)
	if err != nil || u.ID != root.ID || !u.IsAdmin() {
		t.Fatalf("linked login: %+v, %v", u, err)
	}
	// Nothing maps them to a role, so nothing changes either.
	u, _, _ = e.svc.Provision(e.ctx, *e.p, id)
	if !u.IsAdmin() {
		t.Error("an unmapped identity must not change the role")
	}

	// A normal user gets promoted by the group at their next login.
	member := ident("m-sub", "m@test.com", true)
	m, _, err := e.svc.Provision(e.ctx, *e.p, member)
	if err != nil || m.IsAdmin() {
		t.Fatalf("member %+v, %v", m, err)
	}
	member.Groups = []string{"admins"}
	m, _, _ = e.svc.Provision(e.ctx, *e.p, member)
	if !m.IsAdmin() {
		t.Error("the group promotes at login")
	}

	// With two admins, one can lose the role...
	e.p.RoleMapping = []any{map[string]any{"claim": "groups", "operator": "contains", "value": "staff", "role": "user"}}
	demote := ident("m-sub", "m@test.com", true)
	demote.Groups = []string{"staff"}
	m, _, _ = e.svc.Provision(e.ctx, *e.p, demote)
	if m.IsAdmin() {
		t.Error("an admin who is not the last may be demoted by the mapping")
	}
	// ...but the last one never does.
	rootID := ident("root-sub", "root@test.com", true)
	rootID.Groups = []string{"staff"}
	r, _, _ := e.svc.Provision(e.ctx, *e.p, rootID)
	if !r.IsAdmin() {
		t.Error("the last admin was demoted")
	}
}

func TestProvisionLeavesRolesAloneWithoutSync(t *testing.T) {
	mapping := []any{map[string]any{"claim": "groups", "operator": "contains", "value": "admins", "role": "admin"}}
	e := newSSOEnv(t, func(in *IdPWrite) { in.RoleMapping = mapping })
	e.admin(t)
	member := ident("m", "m@test.com", true)
	m, _, _ := e.svc.Provision(e.ctx, *e.p, member)
	member.Groups = []string{"admins"}
	m, _, _ = e.svc.Provision(e.ctx, *e.p, member)
	if m.IsAdmin() {
		t.Error("roles change at login only when the provider syncs them")
	}
}

func TestProvisionReportsTwoFactorLinkedUsers(t *testing.T) {
	e := newSSOEnv(t, nil)
	root := e.admin(t)
	tf := auth.TwoFactor{DB: e.svc.DB, Users: e.svc.Users}
	secret, _, err := tf.Enable(e.ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	root, _ = e.svc.Users.ByID(e.ctx, root.ID)
	if _, err := tf.Confirm(e.ctx, root, totp.Now(secret)); err != nil {
		t.Fatal(err)
	}
	_, needs, err := e.svc.Provision(e.ctx, *e.p, ident("root-sub", "root@test.com", true))
	if err != nil || !needs {
		t.Errorf("a user with two-factor still owes the code after SSO: %v, %v", needs, err)
	}
}

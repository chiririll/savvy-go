package httpserver

import (
	"context"
	"net/http"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/password"
)

type passwordChange map[string]string

func changePassword(current, next string) passwordChange {
	return passwordChange{"current_password": current, "password": next, "password_confirmation": next}
}

func TestChangePasswordKeepsThisSessionAndEndsTheOthers(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", roleEditor)
	current, other := a.issue(u, false), a.issue(u, false)

	status(t, a.do("PUT", "/api/auth/password", changePassword("secret1", "newsecret1"), current.Token, current.CSRF), 200, "change")
	status(t, a.do("GET", "/api/auth/2fa/status", nil, current.Token, ""), 200, "this session")
	status(t, a.do("GET", "/api/auth/2fa/status", nil, other.Token, ""), 401, "other session")

	u, _ = a.svc.Users.ByID(context.Background(), u.ID)
	if !password.Check(*u.Password, "newsecret1") {
		t.Fatal("password not updated")
	}
}

func TestChangePasswordRefusals(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", roleEditor)
	current, other := a.issue(u, false), a.issue(u, false)
	status(t, a.do("PUT", "/api/auth/password", changePassword("wrong-password", "newsecret1"), current.Token, current.CSRF), 422, "wrong current password")
	status(t, a.do("GET", "/api/auth/2fa/status", nil, other.Token, ""), 200, "a refused change ends no session")

	sso := a.createUser("sso@test.com", "secret1", roleEditor)
	_, _ = a.db.Exec(`UPDATE users SET is_sso_only = 1 WHERE id = ?`, sso.ID)
	sso, _ = a.svc.Users.ByID(context.Background(), sso.ID)
	iss := a.issue(sso, false)
	status(t, a.do("PUT", "/api/auth/password", changePassword("secret1", "newsecret1"), iss.Token, iss.CSRF), 422, "SSO-only account")
}

func TestLogoutOthersKeepsThisSession(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("u@test.com", "secret1", roleEditor)
	current, other := a.issue(u, false), a.issue(u, false)

	res := a.do("POST", "/api/auth/logout-others", map[string]any{}, current.Token, current.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 200 || body["revoked"].(float64) != 1 {
		t.Fatalf("logout-others %d %v", res.StatusCode, body)
	}
	status(t, a.do("GET", "/api/auth/2fa/status", nil, current.Token, ""), 200, "this session")
	status(t, a.do("GET", "/api/auth/2fa/status", nil, other.Token, ""), 401, "other session")
}

func TestAdminCreatesActiveAndInvitedUsers(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)

	res := a.do("POST", "/api/users", map[string]any{"name": "Active", "email": "active@test.com", "password": "password1", "role": "user"}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	data, _ := body["data"].(map[string]any)
	if res.StatusCode != 201 || data["isInactive"] != false || data["token"] != nil {
		t.Fatalf("user with a password: %d %v", res.StatusCode, body)
	}

	// Without a password the account waits for its owner to choose one through a link.
	res = a.do("POST", "/api/users", map[string]any{"name": "Invited", "email": "invited@test.com", "role": "user"}, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	data, _ = body["data"].(map[string]any)
	if res.StatusCode != 201 || data["isInactive"] != true || data["token"] == nil {
		t.Fatalf("user without a password: %d %v", res.StatusCode, body)
	}
	res = a.do("POST", "/api/auth/login", map[string]string{"email": "invited@test.com", "password": "anything1"}, "", "")
	status(t, res, 422, "inactive user signs in")
}

func TestSetPasswordLinkActivatesAnInvitedUser(t *testing.T) {
	a := newTestApp(t)
	invited, err := a.svc.Users.Create(context.Background(), "Invited", "invited@test.com", nil, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := a.svc.PasswordTokens.Issue(context.Background(), invited)
	if err != nil {
		t.Fatal(err)
	}

	res := a.do("GET", "/api/auth/password/"+token, nil, "", "")
	if preview := decodeJSON(t, res); res.StatusCode != 200 || preview["email"] != "invited@test.com" || preview["isInactive"] != true {
		t.Fatalf("preview %d %v", res.StatusCode, preview)
	}
	status(t, a.do("GET", "/api/auth/password/not-a-real-token", nil, "", ""), 404, "unknown token")

	set := map[string]string{"password": "newpass12", "password_confirmation": "newpass12"}
	res = a.do("POST", "/api/auth/password/"+token, set, "", "")
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["user"].(map[string]any)["email"] != "invited@test.com" {
		t.Fatalf("accept %d %v", res.StatusCode, body)
	}
	status(t, a.do("POST", "/api/auth/password/"+token, set, "", ""), 404, "link used twice")
	status(t, a.do("POST", "/api/auth/login", map[string]string{"email": "invited@test.com", "password": "newpass12"}, "", ""), 200, "sign in with the new password")
}

func TestAdminResetLinkReplacesTheOldOne(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	sess := a.issue(admin, false)
	target, _ := a.svc.Users.Create(context.Background(), "Target", "target@test.com", ptr("oldpass12"), auth.RoleUser)
	first, _, _ := a.svc.PasswordTokens.Issue(context.Background(), target)

	res := a.do("POST", "/api/users/"+itoa(target.ID)+"/password-token", map[string]any{}, sess.Token, sess.CSRF)
	next := decodeJSON(t, res)["data"].(map[string]any)["token"].(string)
	if next == first {
		t.Fatal("token not rotated")
	}
	status(t, a.do("GET", "/api/auth/password/"+first, nil, "", ""), 404, "old link")
	res = a.do("GET", "/api/auth/password/"+next, nil, "", "")
	if preview := decodeJSON(t, res); res.StatusCode != 200 || preview["isInactive"] != false {
		t.Fatalf("a reset link is for an active account: %d %v", res.StatusCode, preview)
	}
}

func TestAdminResetLinkRefusals(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	sess := a.issue(admin, false)
	sso, _ := a.svc.Users.Create(context.Background(), "Sso", "sso@test.com", ptr("x"), auth.RoleUser)
	_, _ = a.db.Exec(`UPDATE users SET is_sso_only = 1 WHERE id = ?`, sso.ID)

	status(t, a.do("POST", "/api/users/"+itoa(admin.ID)+"/password-token", map[string]any{}, sess.Token, sess.CSRF), http.StatusUnprocessableEntity, "own account")
	status(t, a.do("POST", "/api/users/"+itoa(sso.ID)+"/password-token", map[string]any{}, sess.Token, sess.CSRF), http.StatusUnprocessableEntity, "SSO-only account")
}

func TestAdminSettingAPasswordRevokesThePendingLink(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	invited, _ := a.svc.Users.Create(context.Background(), "Inv", "inv@test.com", nil, auth.RoleUser)
	token, _, _ := a.svc.PasswordTokens.Issue(context.Background(), invited)

	res := a.do("PATCH", "/api/users/"+itoa(invited.ID), map[string]string{"password": "setbyadmin"}, sess.Token, sess.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["data"].(map[string]any)["isInactive"] != false {
		t.Fatalf("admin sets a password %d %v", res.StatusCode, body)
	}
	status(t, a.do("GET", "/api/auth/password/"+token, nil, "", ""), 404, "link after the admin chose a password")
}

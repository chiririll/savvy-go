package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/domain"
	"savvy-go/internal/store"
)

func (a *testApp) invite(spaceID int64, sess *auth.Issued, email, role string) string {
	a.t.Helper()
	res := a.do("POST", spacePath(spaceID, "/invitations"), map[string]any{"email": email, "role": role}, sess.Token, sess.CSRF)
	body := decodeJSON(a.t, res)
	if res.StatusCode != http.StatusCreated {
		a.t.Fatalf("invite %d %v", res.StatusCode, body)
	}
	return body["data"].(map[string]any)["token"].(string)
}

func memberRole(t *testing.T, a *testApp, spaceID, userID int64) string {
	t.Helper()
	role, err := a.s.spaces.Role(context.Background(), spaceID, userID)
	if err != nil {
		t.Fatal(err)
	}
	return role
}

func userPath(id int64) string { return "/" + strconv.FormatInt(id, 10) }

// P6: an invitation is single use, bound to its email when it has one, and
// can be revoked; its preview does not need an account.
func TestP6InvitationLifecycle(t *testing.T) {
	a := newTestApp(t)
	admin := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	alice := a.createUser("alice@test.com", "secret1", auth.RoleUser)
	bob := a.createUser("bob@test.com", "secret1", auth.RoleUser)
	as, bs := a.issue(alice, false), a.issue(bob, false)

	token := a.invite(a.space.ID, admin, "alice@test.com", roleEditor)
	preview := decodeJSON(t, a.do("GET", "/api/invitations/"+token, nil, "", ""))["data"].(map[string]any)
	if preview["spaceName"] != "Test" || preview["role"] != roleEditor {
		t.Fatalf("preview %v", preview)
	}
	status(t, a.do("POST", "/api/invitations/"+token+"/accept", nil, bs.Token, bs.CSRF), 422, "wrong email")
	status(t, a.do("POST", "/api/invitations/"+token+"/accept", nil, as.Token, as.CSRF), 200, "accept")
	if memberRole(t, a, a.space.ID, alice.ID) != roleEditor {
		t.Fatal("alice did not join as editor")
	}
	status(t, a.do("POST", "/api/invitations/"+token+"/accept", nil, as.Token, as.CSRF), 404, "reuse")
	status(t, a.do("GET", "/api/invitations/"+token, nil, "", ""), 404, "used preview")
	status(t, a.do("GET", "/api/invitations/garbage", nil, "", ""), 404, "unknown token")

	open := a.invite(a.space.ID, admin, "", roleViewer)
	list := decodeJSON(t, a.do("GET", spacePath(a.space.ID, "/invitations"), nil, admin.Token, ""))["data"].([]any)
	id := int64(list[0].(map[string]any)["id"].(float64))
	status(t, a.do("DELETE", spacePath(a.space.ID, "/invitations/"+strconv.FormatInt(id, 10)), nil, admin.Token, admin.CSRF), 204, "revoke")
	status(t, a.do("POST", "/api/invitations/"+open+"/accept", nil, bs.Token, bs.CSRF), 404, "revoked")

	expired := a.invite(a.space.ID, admin, "", roleViewer)
	if _, err := a.db.Exec(`UPDATE space_invitations SET expires_at = '2000-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	status(t, a.do("POST", "/api/invitations/"+expired+"/accept", nil, bs.Token, bs.CSRF), 404, "expired")

	status(t, a.do("POST", spacePath(a.space.ID, "/invitations"), map[string]any{"role": roleViewer}, as.Token, as.CSRF), 403, "editor invites")
}

// P7: registering through an invitation from a space admin makes a guest,
// from a server admin a user; neither gets a personal space; the server can
// switch it off.
func TestP7RegisterThroughInvitation(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	serverAdmin := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	if err := a.s.spaces.SetMember(ctx, a.space.ID, owner, domain.SpaceAdmin); err != nil {
		t.Fatal(err)
	}
	spaceAdmin := a.issue(owner, false)

	register := func(token, email string) (int, map[string]any) {
		res := a.do("POST", "/api/invitations/"+token+"/register", map[string]any{"name": "New", "email": email, "password": "password1"}, "", "")
		return res.StatusCode, decodeJSON(t, res)
	}
	if code, body := register(a.invite(a.space.ID, spaceAdmin, "", roleEditor), "guest@test.com"); code != http.StatusCreated {
		t.Fatalf("register %d %v", code, body)
	}
	guest, _ := a.s.users.ByEmail(ctx, "guest@test.com")
	if guest.Role != auth.RoleGuest || memberRole(t, a, a.space.ID, guest.ID) != roleEditor {
		t.Fatalf("guest %+v", guest)
	}
	if code, _ := register(a.invite(a.space.ID, serverAdmin, "", roleViewer), "user@test.com"); code != http.StatusCreated {
		t.Fatal("register through a server admin's invitation")
	}
	user, _ := a.s.users.ByEmail(ctx, "user@test.com")
	if user.Role != auth.RoleUser {
		t.Fatalf("role %s", user.Role)
	}
	for _, u := range []*auth.User{guest, user} {
		if mine, _ := a.s.spaces.ForUser(ctx, u.ID); len(mine) != 1 {
			t.Fatalf("%s is in %d spaces, want only the invited one", u.Email, len(mine))
		}
	}
	if code, _ := register(a.invite(a.space.ID, spaceAdmin, "", roleViewer), "owner@test.com"); code != http.StatusUnprocessableEntity {
		t.Fatalf("register with a taken email: %d", code)
	}

	_ = a.s.settings.Set(ctx, "space_invites_can_register", false)
	if code, _ := register(a.invite(a.space.ID, spaceAdmin, "", roleViewer), "late@test.com"); code != http.StatusForbidden {
		t.Fatalf("registration closed: %d", code)
	}
}

// P8: a guest is never a space admin: not through an invitation, a role
// change or the last-admin hand-over; guests cannot create spaces.
func TestP8GuestNeverSpaceAdmin(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	as := a.issue(admin, false)
	guest := a.createUser("guest@test.com", "secret1", auth.RoleGuest)
	gs := a.issue(guest, false)

	status(t, a.do("POST", "/api/invitations/"+a.invite(a.space.ID, as, "", domain.SpaceAdmin)+"/accept", nil, gs.Token, gs.CSRF), 422, "admin invitation")
	status(t, a.do("POST", "/api/invitations/"+a.invite(a.space.ID, as, "", roleEditor)+"/accept", nil, gs.Token, gs.CSRF), 200, "editor invitation")
	status(t, a.do("PATCH", spacePath(a.space.ID, "/members"+userPath(guest.ID)), map[string]any{"role": domain.SpaceAdmin}, as.Token, as.CSRF), 422, "promote")
	status(t, a.do("POST", "/api/spaces", map[string]any{"name": "Mine"}, gs.Token, gs.CSRF), 422, "create space")
	status(t, a.do("POST", "/api/admin/spaces/"+strconv.FormatInt(a.space.ID, 10)+"/admins", map[string]any{"user_id": guest.ID}, as.Token, as.CSRF), 422, "assign admin")
	// Registering through an admin invitation from a space admin would make
	// a guest admin.
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	_ = a.s.spaces.SetMember(context.Background(), a.space.ID, owner, domain.SpaceAdmin)
	res := a.do("POST", "/api/invitations/"+a.invite(a.space.ID, a.issue(owner, false), "", domain.SpaceAdmin)+"/register",
		map[string]any{"name": "N", "email": "n@test.com", "password": "password1"}, "", "")
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("guest admin by registration %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
}

// P9: a server admin cannot demote to guest someone who administers a space.
func TestP9NoGuestWhileSpaceAdmin(t *testing.T) {
	a := newTestApp(t)
	as := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	if _, err := a.s.spaces.Create(context.Background(), "Owned", owner); err != nil {
		t.Fatal(err)
	}
	status(t, a.do("PATCH", "/api/users"+userPath(owner.ID), map[string]any{"role": auth.RoleGuest}, as.Token, as.CSRF), 422, "demote space admin")
	plain := a.createUser("plain@test.com", "secret1", auth.RoleUser)
	status(t, a.do("PATCH", "/api/users"+userPath(plain.ID), map[string]any{"role": auth.RoleGuest}, as.Token, as.CSRF), 200, "demote plain user")
}

// P10: max_spaces_per_user counts admin memberships: creating, accepting an
// admin invitation and promotion are refused beyond it; server admins and
// their assignments are exempt.
func TestP10SpaceLimit(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	as := a.issue(admin, false)
	user := a.createUser("user@test.com", "secret1", auth.RoleUser)
	us := a.issue(user, false)
	_ = a.s.settings.Set(ctx, "max_spaces_per_user", 1)

	status(t, a.do("POST", "/api/spaces", map[string]any{"name": "First"}, us.Token, us.CSRF), 201, "first space")
	status(t, a.do("POST", "/api/spaces", map[string]any{"name": "Second"}, us.Token, us.CSRF), 422, "second space")
	status(t, a.do("POST", "/api/invitations/"+a.invite(a.space.ID, as, "", domain.SpaceAdmin)+"/accept", nil, us.Token, us.CSRF), 422, "admin invitation")
	status(t, a.do("POST", "/api/invitations/"+a.invite(a.space.ID, as, "", roleEditor)+"/accept", nil, us.Token, us.CSRF), 200, "editor invitation")
	status(t, a.do("PATCH", spacePath(a.space.ID, "/members"+userPath(user.ID)), map[string]any{"role": domain.SpaceAdmin}, as.Token, as.CSRF), 422, "promotion")
	status(t, a.do("POST", "/api/admin/spaces/"+strconv.FormatInt(a.space.ID, 10)+"/admins", map[string]any{"user_id": user.ID}, as.Token, as.CSRF), 204, "server admin assigns")
	status(t, a.do("POST", "/api/spaces", map[string]any{"name": "Admin's"}, as.Token, as.CSRF), 201, "server admin has no limit")

	_ = a.s.settings.Set(ctx, "max_spaces_per_user", 0)
	other := a.issue(a.createUser("other@test.com", "secret1", auth.RoleUser), false)
	status(t, a.do("POST", "/api/spaces", map[string]any{"name": "None"}, other.Token, other.CSRF), 422, "limit zero")
}

// P11: a space keeps an admin while it has other members.
func TestP11LastAdmin(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	os := a.issue(owner, false)
	space, err := a.s.spaces.Create(ctx, "Family", owner)
	if err != nil {
		t.Fatal(err)
	}
	member := a.createUser("member@test.com", "secret1", auth.RoleUser)
	_ = a.s.spaces.SetMember(ctx, space.ID, member, roleEditor)

	status(t, a.do("POST", spacePath(space.ID, "/leave"), nil, os.Token, os.CSRF), 422, "last admin leaves")
	status(t, a.do("PATCH", spacePath(space.ID, "/members"+userPath(owner.ID)), map[string]any{"role": roleEditor}, os.Token, os.CSRF), 422, "last admin demotes")
	admin := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	status(t, a.do("DELETE", "/api/users"+userPath(owner.ID), nil, admin.Token, admin.CSRF), 422, "delete last admin")

	status(t, a.do("PATCH", spacePath(space.ID, "/members"+userPath(member.ID)), map[string]any{"role": domain.SpaceAdmin}, os.Token, os.CSRF), 200, "hand over")
	status(t, a.do("POST", spacePath(space.ID, "/leave"), nil, os.Token, os.CSRF), 204, "leave after hand-over")

	ms := a.issue(member, false)
	status(t, a.do("POST", spacePath(space.ID, "/leave"), nil, ms.Token, ms.CSRF), 422, "only member leaves")
}

// P12, P41: deleting a user tombstones them; spaces they were alone in go
// with them, leaving a final backup only server admins see.
func TestP12P41DeleteUserAndTheirSpaces(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	admin := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	gone := a.createUser("gone@test.com", "secret1", auth.RoleUser)
	gs := a.issue(gone, false)
	alone, err := a.s.spaces.Create(ctx, "Alone", gone)
	if err != nil {
		t.Fatal(err)
	}
	status(t, a.inSpace(alone.ID, "POST", "/api/tags", map[string]any{"name": "keepsake"}, gs), 201, "tag")

	status(t, a.do("DELETE", "/api/users"+userPath(gone.ID), nil, admin.Token, admin.CSRF), 204, "delete")
	u, _ := a.s.users.ByID(ctx, gone.ID)
	if u == nil || !u.Deleted || u.Email == "gone@test.com" || u.Password != nil {
		t.Fatalf("tombstone %+v", u)
	}
	status(t, a.do("GET", "/api/spaces", nil, gs.Token, ""), 401, "session revoked")
	if sp, _ := a.s.spaces.Get(ctx, alone.ID); sp != nil {
		t.Fatal("the space the user was alone in survived")
	}
	status(t, a.do("GET", "/api/users"+userPath(gone.ID), nil, admin.Token, ""), 404, "deleted user hidden")

	list := decodeJSON(t, a.do("GET", "/api/admin/deleted-spaces", nil, admin.Token, ""))["data"].([]any)
	if len(list) != 1 {
		t.Fatalf("deleted backups %v", list)
	}
	editor := a.issue(a.createUser("e@test.com", "secret1", roleEditor), false)
	status(t, a.do("GET", "/api/admin/deleted-spaces", nil, editor.Token, ""), 403, "non-admin")
	name := list[0].(map[string]any)["filename"].(string)
	res := a.do("POST", "/api/admin/deleted-spaces/"+name+"/restore", nil, admin.Token, admin.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("restore deleted %d %v", res.StatusCode, body)
	}
	restored := int64(body["data"].(map[string]any)["id"].(float64))
	if names := tagNames(t, a.inSpace(restored, "GET", "/api/tags", nil, admin)); len(names) != 1 || names[0] != "keepsake" {
		t.Fatalf("restored tags %v", names)
	}
}

// P5: the overview shows every space's metadata, never its data; a server
// admin outside a space cannot read it.
func TestP5AdminOverviewIsMetadataOnly(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	as := a.issue(admin, false)
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	private, _ := a.s.spaces.Create(ctx, "Private", owner)

	status(t, a.inSpace(private.ID, "GET", "/api/accounts", nil, as), 404, "server admin reads data")
	list := decodeJSON(t, a.do("GET", "/api/admin/spaces", nil, as.Token, ""))["data"].([]any)
	var found map[string]any
	for _, x := range list {
		if row := x.(map[string]any); int64(row["id"].(float64)) == private.ID {
			found = row
		}
	}
	if found == nil || found["name"] != "Private" || found["members"].(float64) != 1 || found["size"].(float64) <= 0 {
		t.Fatalf("overview %v", found)
	}
	for _, k := range []string{"accounts", "balance", "transactions"} {
		if _, ok := found[k]; ok {
			t.Fatalf("overview leaks %s", k)
		}
	}
	user := a.issue(owner, false)
	status(t, a.do("GET", "/api/admin/spaces", nil, user.Token, ""), 403, "non-admin overview")
}

// P53: what a server admin does to a space is in the space's audit log.
func TestP53AdminActionsAreAudited(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	as := a.issue(admin, false)
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	space, _ := a.s.spaces.Create(ctx, "Watched", owner)
	os := a.issue(owner, false)

	id := strconv.FormatInt(space.ID, 10)
	status(t, a.do("POST", "/api/admin/spaces/"+id+"/admins", map[string]any{"user_id": admin.ID}, as.Token, as.CSRF), 204, "assign self")
	status(t, a.do("PATCH", "/api/admin/spaces/"+id, map[string]any{"quota_bytes": 10 << 20}, as.Token, as.CSRF), 200, "quota")
	log := decodeJSON(t, a.do("GET", spacePath(space.ID, "/audit"), nil, os.Token, ""))["data"].([]any)
	actions := map[string]bool{}
	for _, x := range log {
		actions[x.(map[string]any)["action"].(string)] = true
	}
	if !actions["assign_admin"] || !actions["set_quota"] {
		t.Fatalf("audit %v", log)
	}
	if q := a.s.spaces.Quota(ctx, space.ID); q != 10<<20 {
		t.Fatalf("quota %d", q)
	}
}

// P42: a broken space is a 503 for its members and a 404 for everyone else.
func TestP42UnavailableSpaceIsHiddenFromOutsiders(t *testing.T) {
	a := newTestApp(t)
	member := a.issue(a.createUser("m@test.com", "secret1", roleViewer), false)
	outsider := a.issue(a.createUser("o@test.com", "secret1", auth.RoleUser), false)
	a.s.store = unavailableStore{a.store, a.space.ID}
	status(t, a.do("GET", "/api/tags", nil, member.Token, ""), 503, "member")
	status(t, a.do("GET", "/api/tags", nil, outsider.Token, ""), 404, "outsider")
}

func TestSpaceRenameAndDelete(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	os := a.issue(owner, false)
	space, _ := a.s.spaces.Create(ctx, "Old", owner)
	viewer := a.createUser("v@test.com", "secret1", auth.RoleUser)
	_ = a.s.spaces.SetMember(ctx, space.ID, viewer, roleViewer)
	vs := a.issue(viewer, false)

	status(t, a.do("PATCH", spacePath(space.ID, ""), map[string]any{"name": "New"}, vs.Token, vs.CSRF), 403, "viewer renames")
	res := a.do("PATCH", spacePath(space.ID, ""), map[string]any{"name": "New"}, os.Token, os.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["data"].(map[string]any)["name"] != "New" {
		t.Fatalf("rename %d %v", res.StatusCode, body)
	}
	members := decodeJSON(t, a.do("GET", spacePath(space.ID, "/members"), nil, vs.Token, ""))["data"].([]any)
	if len(members) != 2 {
		t.Fatalf("members %v", members)
	}
	status(t, a.do("DELETE", spacePath(space.ID, ""), nil, vs.Token, vs.CSRF), 403, "viewer deletes")
	status(t, a.do("DELETE", spacePath(space.ID, ""), nil, os.Token, os.CSRF), 204, "admin deletes")
	status(t, a.do("GET", spacePath(space.ID, "/members"), nil, os.Token, ""), 404, "gone")
}

// unavailableStore reports one space as unavailable, as after a failed
// migration.
type unavailableStore struct {
	store.Store
	broken int64
}

func (u unavailableStore) Space(ctx context.Context, id int64) (store.DB, error) {
	if id == u.broken {
		return nil, store.ErrUnavailable
	}
	return u.Store.Space(ctx, id)
}

// P16: a registered space without a database and a database without a
// registered space are both reported, never silently dropped.
func TestP16ReconcileRegistryAndFiles(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	admin := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	if _, err := a.db.Exec(`INSERT INTO spaces (id, uuid, name) VALUES (900, 'no-file', 'Ghost')`); err != nil {
		t.Fatal(err)
	}
	if err := a.store.CreateSpace(ctx, 901); err != nil {
		t.Fatal(err)
	}
	missing, orphans, err := a.s.spaces.Reconcile(ctx)
	if err != nil || len(missing) != 1 || missing[0] != 900 || len(orphans) != 1 || orphans[0] != 901 {
		t.Fatalf("reconcile %v %v %v", missing, orphans, err)
	}
	for _, x := range decodeJSON(t, a.do("GET", "/api/admin/spaces", nil, admin.Token, ""))["data"].([]any) {
		row := x.(map[string]any)
		if row["id"].(float64) == 900 && row["unavailable"] == "" {
			t.Fatalf("a space without a file looks healthy: %v", row)
		}
	}
}

// The space limits and invitation registration are instance settings a
// server admin changes through the API.
func TestSpaceSettingsAreAdminEditable(t *testing.T) {
	a := newTestApp(t)
	admin := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	res := a.do("PATCH", "/api/settings", map[string]any{"max_spaces_per_user": 0, "space_invites_can_register": false}, admin.Token, admin.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 200 || body["max_spaces_per_user"] != float64(0) || body["space_invites_can_register"] != false {
		t.Fatalf("update %d %v", res.StatusCode, body)
	}
	user := a.issue(a.createUser("u@test.com", "secret1", auth.RoleUser), false)
	status(t, a.do("POST", "/api/spaces", map[string]any{"name": "No"}, user.Token, user.CSRF), 422, "limit 0 through the API")
	res = a.do("PATCH", "/api/settings", map[string]any{"max_spaces_per_user": nil}, admin.Token, admin.CSRF)
	res.Body.Close()
	status(t, a.do("POST", "/api/spaces", map[string]any{"name": "Yes"}, user.Token, user.CSRF), 201, "unlimited again")
}

package httpserver

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/domain"
)

// inSpace sends a data request to a given space.
func (a *testApp) inSpace(spaceID int64, method, path string, body any, sess *auth.Issued) *http.Response {
	a.t.Helper()
	req := a.request(method, spacedPath(a.s, spaceID, path), body, sess.Token, sess.CSRF)
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	return res
}

func tagNames(t *testing.T, res *http.Response) []string {
	t.Helper()
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list tags %d %v", res.StatusCode, body)
	}
	var out []string
	for _, x := range body["data"].([]any) {
		out = append(out, x.(map[string]any)["name"].(string))
	}
	return out
}

// P3: each space has its own data; a space the caller does not belong to is
// a 404, never a 403.
func TestP3SpacesAreIsolated(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	alice := a.createUser("alice@test.com", "secret1", auth.RoleUser)
	bob := a.createUser("bob@test.com", "secret1", auth.RoleUser)
	aliceSpace, err := a.s.spaces.Create(ctx, "Alice", alice)
	if err != nil {
		t.Fatal(err)
	}
	bobSpace, err := a.s.spaces.Create(ctx, "Bob", bob)
	if err != nil {
		t.Fatal(err)
	}
	as, bs := a.issue(alice, false), a.issue(bob, false)

	status(t, a.inSpace(aliceSpace.ID, "POST", "/api/tags", map[string]any{"name": "alice-only"}, as), 201, "alice tag")
	status(t, a.inSpace(bobSpace.ID, "POST", "/api/tags", map[string]any{"name": "bob-only"}, bs), 201, "bob tag")

	if names := tagNames(t, a.inSpace(aliceSpace.ID, "GET", "/api/tags", nil, as)); len(names) != 1 || names[0] != "alice-only" {
		t.Fatalf("alice sees %v", names)
	}
	if names := tagNames(t, a.inSpace(bobSpace.ID, "GET", "/api/tags", nil, bs)); len(names) != 1 || names[0] != "bob-only" {
		t.Fatalf("bob's space shows %v", names)
	}
	status(t, a.inSpace(aliceSpace.ID, "GET", "/api/tags", nil, bs), 404, "bob reads alice's space")
	status(t, a.inSpace(aliceSpace.ID, "POST", "/api/tags", map[string]any{"name": "x"}, bs), 404, "bob writes alice's space")
	status(t, a.inSpace(999, "GET", "/api/tags", nil, bs), 404, "missing space")

	// Joining as a viewer makes the space readable but not writable.
	if err := a.s.spaces.SetMember(ctx, aliceSpace.ID, bob, domain.SpaceViewer); err != nil {
		t.Fatal(err)
	}
	if names := tagNames(t, a.inSpace(aliceSpace.ID, "GET", "/api/tags", nil, bs)); len(names) != 1 {
		t.Fatalf("viewer sees %v", names)
	}
	status(t, a.inSpace(aliceSpace.ID, "POST", "/api/tags", map[string]any{"name": "x"}, bs), 403, "viewer writes")
}

func TestUserWithoutSpaceGets404ForData(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("lonely@test.com", "secret1", auth.RoleUser)
	sess := a.issue(u, false)
	status(t, a.do("GET", "/api/accounts", nil, sess.Token, ""), 404, "not in the space")
	status(t, a.do("GET", "/api/settings", nil, sess.Token, ""), 200, "instance settings need no space")
}

func TestRegistrationProvisionsPersonalSpace(t *testing.T) {
	a := newTestApp(t)
	res := a.do("POST", "/api/auth/register", map[string]string{"name": "Owner", "email": "o@test.com", "password": "secret1"}, "", "")
	session := cookieNamed(res, "svy_session")
	res.Body.Close()
	if res.StatusCode != 201 || session == nil {
		t.Fatalf("register %d", res.StatusCode)
	}
	body := decodeJSON(t, a.do("GET", "/api/spaces", nil, session.Value, ""))
	list := body["data"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["role"] != domain.SpaceAdmin || list[0].(map[string]any)["name"] != "Owner" {
		t.Fatalf("spaces %v", list)
	}
	id := int64(list[0].(map[string]any)["id"].(float64))
	status(t, a.do("GET", spacedPath(a.s, id, "/api/accounts"), nil, session.Value, ""), 200, "personal space works")
}

// P8: a guest cannot administer a space and gets no personal space.
func TestP8GuestCannotBeSpaceAdmin(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	guest := a.createUser("guest@test.com", "secret1", auth.RoleGuest)
	if sp, err := a.s.spaces.Provision(ctx, guest); err != nil || sp != nil {
		t.Fatalf("guest got a personal space %v %v", sp, err)
	}
	if _, err := a.s.spaces.Create(ctx, "Mine", guest); !errors.Is(err, domain.ErrGuestCannotAdmin) {
		t.Fatalf("guest created a space: %v", err)
	}
	if err := a.s.spaces.SetMember(ctx, a.space.ID, guest, domain.SpaceAdmin); !errors.Is(err, domain.ErrGuestCannotAdmin) {
		t.Fatalf("guest made admin: %v", err)
	}
	if err := a.s.spaces.SetMember(ctx, a.space.ID, guest, domain.SpaceEditor); err != nil {
		t.Fatalf("guest as editor: %v", err)
	}
}

// Only server admins change instance settings; only space admins change the
// space's own settings, which live in the space.
func TestSettingsUpdateRoles(t *testing.T) {
	a := newTestApp(t)
	editor := a.issue(a.createUser("e@test.com", "secret1", roleEditor), false)
	admin := a.issue(a.createUser("a@test.com", "secret1", auth.RoleAdmin), false)
	spaceSettings := spacePath(a.space.ID, "/settings")

	status(t, a.do("PATCH", "/api/settings", map[string]any{"sso_allow_signup": false}, editor.Token, editor.CSRF), 403, "editor instance key")
	status(t, a.do("PATCH", spaceSettings, map[string]any{"auto_update_currencies": false}, editor.Token, editor.CSRF), 403, "editor space key")
	status(t, a.do("GET", "/api/settings", nil, editor.Token, ""), 200, "instance settings are readable")

	res := a.do("PATCH", "/api/settings", map[string]any{"sso_allow_signup": false}, admin.Token, admin.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["sso_allow_signup"] != false {
		t.Fatalf("admin instance update %d %v", res.StatusCode, body)
	}
	res = a.do("PATCH", spaceSettings, map[string]any{"auto_update_currencies": false}, admin.Token, admin.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["auto_update_currencies"] != false {
		t.Fatalf("admin space update %d %v", res.StatusCode, body)
	}
	if v, _ := sp0(t, a).settings.All(context.Background()); v["auto_update_currencies"] != false {
		t.Fatalf("space setting not stored in the space: %v", v)
	}
}

// sp0 is the scope of the shared test space.
func sp0(t *testing.T, a *testApp) *spaceScope {
	t.Helper()
	return a.s.newScope(a.space.ID, domain.SpaceAdmin, a.spaceDB())
}

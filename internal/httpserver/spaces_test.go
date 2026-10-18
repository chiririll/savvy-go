package httpserver

import (
	"context"
	"net/http"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/spaces"
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

// Each space has its own data; a space the caller does not belong to is
// a 404, never a 403.
func TestSpacesAreIsolated(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	alice := a.createUser("alice@test.com", "secret1", auth.RoleUser)
	bob := a.createUser("bob@test.com", "secret1", auth.RoleUser)
	aliceSpace, err := a.svc.Spaces.Create(ctx, "Alice", alice)
	if err != nil {
		t.Fatal(err)
	}
	bobSpace, err := a.svc.Spaces.Create(ctx, "Bob", bob)
	if err != nil {
		t.Fatal(err)
	}
	as, bs := a.issue(alice, false), a.issue(bob, false)

	status(t, a.inSpace(aliceSpace.ID, "POST", "/api/tags", map[string]any{"name": "alice-only"}, as), 201, "alice tag")
	status(t, a.inSpace(bobSpace.ID, "POST", "/api/tags", map[string]any{"name": "bob-only"}, bs), 201, "bob tag")

	if names := tagNames(t, a.inSpace(aliceSpace.ID, "GET", "/api/tags", nil, as)); len(names) != 1 ||
		names[0] != "alice-only" {
		t.Fatalf("alice sees %v", names)
	}
	if names := tagNames(t, a.inSpace(bobSpace.ID, "GET", "/api/tags", nil, bs)); len(names) != 1 ||
		names[0] != "bob-only" {
		t.Fatalf("bob's space shows %v", names)
	}
	status(t, a.inSpace(aliceSpace.ID, "GET", "/api/tags", nil, bs), 404, "bob reads alice's space")
	status(
		t,
		a.inSpace(aliceSpace.ID, "POST", "/api/tags", map[string]any{"name": "x"}, bs),
		404,
		"bob writes alice's space",
	)
	status(t, a.inSpace(999, "GET", "/api/tags", nil, bs), 404, "missing space")

	// Joining as a viewer makes the space readable but not writable.
	if err := a.svc.Spaces.SetMember(ctx, aliceSpace.ID, bob, spaces.RoleViewer); err != nil {
		t.Fatal(err)
	}
	if names := tagNames(t, a.inSpace(aliceSpace.ID, "GET", "/api/tags", nil, bs)); len(names) != 1 {
		t.Fatalf("viewer sees %v", names)
	}
	status(t, a.inSpace(aliceSpace.ID, "POST", "/api/tags", map[string]any{"name": "x"}, bs), 403, "viewer writes")
}

// Only server admins change instance settings; only space admins change the
// space's own settings, which live in the space.
func TestSettingsUpdateRoles(t *testing.T) {
	a := newTestApp(t)
	editor := a.issue(a.createUser("e@test.com", "secret1", roleEditor), false)
	admin := a.issue(a.createUser("a@test.com", "secret1", auth.RoleAdmin), false)
	spaceSettings := spacePath(a.space.ID, "/settings")

	status(
		t,
		a.do("PATCH", "/api/settings", map[string]any{"sso_allow_signup": false}, editor.Token, editor.CSRF),
		403,
		"editor instance key",
	)
	status(
		t,
		a.do("PATCH", spaceSettings, map[string]any{"auto_update_currencies": false}, editor.Token, editor.CSRF),
		403,
		"editor space key",
	)
	status(t, a.do("GET", "/api/settings", nil, editor.Token, ""), 200, "instance settings are readable")

	res := a.do("PATCH", "/api/settings", map[string]any{"sso_allow_signup": false}, admin.Token, admin.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["sso_allow_signup"] != false {
		t.Fatalf("admin instance update %d %v", res.StatusCode, body)
	}
	res = a.do("PATCH", spaceSettings, map[string]any{"auto_update_currencies": false}, admin.Token, admin.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != 200 || body["auto_update_currencies"] != false {
		t.Fatalf("admin space update %d %v", res.StatusCode, body)
	}
	if v, _ := sp0(t, a).Settings.All(context.Background()); v["auto_update_currencies"] != false {
		t.Fatalf("space setting not stored in the space: %v", v)
	}
}

// sp0 is the scope of the shared test space.
func sp0(t *testing.T, a *testApp) *httpx.SpaceScope {
	t.Helper()
	return a.s.newScope(a.space.ID, spaces.RoleAdmin, a.spaceDB())
}

// A new account belongs to no space, however it was made, and cannot read space data until
// it joins or creates one. Instance settings need no space.
func TestNewAccountsHaveNoSpace(t *testing.T) {
	t.Run("registration", func(t *testing.T) {
		a := newTestApp(t)
		res := a.register(a.bootstrap.Password(), nil)
		session := cookieNamed(res, "svy_session")
		status(t, res, 201, "register")
		list := decodeJSON(t, a.do("GET", "/api/spaces", nil, session.Value, ""))["data"].([]any)
		if len(list) != 0 {
			t.Fatalf("spaces %v", list)
		}
		status(t, a.do("GET", "/api/auth/me", nil, session.Value, ""), 200, "signed in without a space")
	})

	t.Run("created by an admin", func(t *testing.T) {
		a := newTestApp(t)
		admin := a.issue(a.createUser("a@test.com", "secret1", auth.RoleAdmin), false)
		res := a.do(
			"POST",
			"/api/users",
			map[string]any{"name": "New", "email": "n@test.com", "password": "secret12", "role": auth.RoleUser},
			admin.Token,
			admin.CSRF,
		)
		status(t, res, 201, "create user")
		u, err := a.svc.Users.ByEmail(context.Background(), "n@test.com")
		if err != nil || u == nil {
			t.Fatalf("user %v %v", u, err)
		}
		if list, err := a.svc.Spaces.ForUser(context.Background(), u.ID); err != nil || len(list) != 0 {
			t.Fatalf("spaces %v %v", list, err)
		}
	})

	t.Run("data routes", func(t *testing.T) {
		a := newTestApp(t)
		sess := a.issue(a.createUser("lonely@test.com", "secret1", auth.RoleUser), false)
		status(t, a.do("GET", "/api/accounts", nil, sess.Token, ""), 404, "data without a space")
		status(t, a.do("GET", "/api/settings", nil, sess.Token, ""), 200, "instance settings")
	})
}

func TestSpaceEmojiAndColor(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("look@test.com", "secret1", auth.RoleUser)
	s := a.issue(u, false)

	res := a.do("POST", "/api/spaces", map[string]any{"name": "Trip", "emoji": "🏖️", "color": "#FF8800"}, s.Token, s.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	data := body["data"].(map[string]any)
	if data["emoji"] != "🏖️" || data["color"] != "#ff8800" {
		t.Fatalf("created = %v", data)
	}
	id := int64(data["id"].(float64))

	for field, payload := range map[string]map[string]any{
		"emoji": {"name": "X", "emoji": "🏠🏠"},
		"color": {"name": "X", "color": "red"},
	} {
		res := a.do("POST", "/api/spaces", payload, s.Token, s.CSRF)
		b := decodeJSON(t, res)
		if res.StatusCode != http.StatusUnprocessableEntity || b["errors"].(map[string]any)[field] == nil {
			t.Fatalf("create with bad %s: %d %v", field, res.StatusCode, b)
		}
		res = a.do("PATCH", spacePath(id, ""), payload, s.Token, s.CSRF)
		b = decodeJSON(t, res)
		if res.StatusCode != http.StatusUnprocessableEntity || b["errors"].(map[string]any)[field] == nil {
			t.Fatalf("update with bad %s: %d %v", field, res.StatusCode, b)
		}
	}

	res = a.do("PATCH", spacePath(id, ""), map[string]any{"name": "Trip", "color": "#001122"}, s.Token, s.CSRF)
	b := decodeJSON(t, res)
	d := b["data"].(map[string]any)
	if res.StatusCode != http.StatusOK || d["emoji"] != nil || d["color"] != "#001122" {
		t.Fatalf("update %d %v", res.StatusCode, b)
	}

	list := decodeJSON(t, a.do("GET", "/api/spaces", nil, s.Token, s.CSRF))
	var seen bool
	for _, x := range list["data"].([]any) {
		if m := x.(map[string]any); m["name"] == "Trip" {
			seen = m["color"] == "#001122"
		}
	}
	if !seen {
		t.Fatalf("list lost the look: %v", list)
	}
}

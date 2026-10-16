package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/spaces"
)

// syncPair is two spaces and the user who administers both.
type syncPair struct {
	a     *testApp
	owner *auth.User
	sess  *auth.Issued
	A, B  *spaces.Space
}

func newSyncPair(t *testing.T) *syncPair {
	t.Helper()
	a := newTestApp(t)
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	A, err := a.s.Spaces.Create(context.Background(), "Personal", owner)
	if err != nil {
		t.Fatal(err)
	}
	B, err := a.s.Spaces.Create(context.Background(), "Family", owner)
	if err != nil {
		t.Fatal(err)
	}
	return &syncPair{a: a, owner: owner, sess: a.issue(owner, false), A: A, B: B}
}

func (p *syncPair) call(sess *auth.Issued, method, path string, body any) (int, map[string]any) {
	p.a.t.Helper()
	res := p.a.do(method, path, body, sess.Token, sess.CSRF)
	code := res.StatusCode
	return code, decodeJSON(p.a.t, res)
}

func (p *syncPair) group(extra map[string]any) map[string]any {
	p.a.t.Helper()
	body := map[string]any{"add_space_ids": []int64{p.A.ID, p.B.ID}, "categories": true, "tags": true}
	for k, v := range extra {
		body[k] = v
	}
	code, out := p.call(p.sess, "POST", "/api/sync-groups/apply", body)
	if code != http.StatusOK {
		p.a.t.Fatalf("apply %d %v", code, out)
	}
	return out
}

func (p *syncPair) names(spaceID int64, kind string) []string {
	p.a.t.Helper()
	code, out := p.call(p.sess, "GET", spacePath(spaceID, "/"+kind), nil)
	if code != http.StatusOK {
		p.a.t.Fatalf("%s %d %v", kind, code, out)
	}
	var names []string
	for _, x := range out["data"].([]any) {
		names = append(names, x.(map[string]any)["name"].(string))
	}
	return names
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestSyncGroupMirrorsCategoriesAndTagsOverHTTP(t *testing.T) {
	p := newSyncPair(t)
	p.group(nil)

	code, out := p.call(p.sess, "GET", spacePath(p.A.ID, "/sync-group"), nil)
	g, _ := out["data"].(map[string]any)
	if code != 200 || g == nil || g["categories"] != true || len(g["members"].([]any)) != 2 || g["hidden"] != float64(0) {
		t.Fatalf("group %d %v", code, out)
	}

	for _, name := range []string{"Food", "Rent"} { // the last category of a type cannot go
		if code, out := p.call(p.sess, "POST", spacePath(p.A.ID, "/categories"), map[string]any{"name": name, "type": "expense"}); code != 201 {
			t.Fatalf("create %d %v", code, out)
		}
	}
	if !contains(p.names(p.B.ID, "categories"), "Rent") {
		t.Fatal("the category did not reach space B")
	}
	if code, out := p.call(p.sess, "POST", spacePath(p.B.ID, "/tags"), map[string]any{"name": "trip"}); code != 201 {
		t.Fatalf("tag %d %v", code, out)
	}
	if !contains(p.names(p.A.ID, "tags"), "trip") {
		t.Fatal("the tag did not reach space A")
	}

	_, out = p.call(p.sess, "GET", spacePath(p.A.ID, "/categories"), nil)
	var rent float64
	for _, x := range out["data"].([]any) {
		if c := x.(map[string]any); c["name"] == "Rent" {
			rent = c["id"].(float64)
		}
	}
	if code, _ := p.call(p.sess, "DELETE", spacePath(p.A.ID, "/categories/"+strconv.Itoa(int(rent))), nil); code != http.StatusNoContent {
		t.Fatalf("delete %d", code)
	}
	if contains(p.names(p.B.ID, "categories"), "Rent") {
		t.Fatal("the delete did not reach space B")
	}
}

func TestSyncGroupRefusesAnEditorWhoOnlyViewsAnotherSpace(t *testing.T) {
	p := newSyncPair(t)
	p.group(nil)
	bob := p.a.createUser("bob@test.com", "secret1", auth.RoleUser)
	ctx := context.Background()
	if err := p.a.s.Spaces.SetMember(ctx, p.A.ID, bob, spaces.RoleEditor); err != nil {
		t.Fatal(err)
	}
	if err := p.a.s.Spaces.SetMember(ctx, p.B.ID, bob, spaces.RoleViewer); err != nil {
		t.Fatal(err)
	}
	bobSess := p.a.issue(bob, false)

	code, out := p.call(bobSess, "POST", spacePath(p.A.ID, "/categories"), map[string]any{"name": "Gifts", "type": "expense"})
	if code != http.StatusForbidden {
		t.Fatalf("create %d %v", code, out)
	}
	if contains(p.names(p.A.ID, "categories"), "Gifts") {
		t.Fatal("the refused category was created")
	}

	// Bob sees his space in the group and only counts the other.
	_, out = p.call(bobSess, "GET", spacePath(p.A.ID, "/sync-group"), nil)
	g := out["data"].(map[string]any)
	if len(g["members"].([]any)) != 2 { // he belongs to both, as editor and viewer
		t.Fatalf("members %v", g["members"])
	}
	carol := p.a.createUser("carol@test.com", "secret1", auth.RoleUser)
	_ = p.a.s.Spaces.SetMember(ctx, p.A.ID, carol, spaces.RoleViewer)
	_, out = p.call(p.a.issue(carol, false), "GET", spacePath(p.A.ID, "/sync-group"), nil)
	if g := out["data"].(map[string]any); len(g["members"].([]any)) != 1 || g["hidden"] != float64(1) {
		t.Fatalf("carol sees %v", g)
	}
}

func TestSyncGroupDifferencesNeedDecisions(t *testing.T) {
	p := newSyncPair(t)
	if code, _ := p.call(p.sess, "POST", spacePath(p.A.ID, "/categories"), map[string]any{"name": "Rent", "type": "expense", "is_default": true}); code != 201 {
		t.Fatal("create A")
	}
	if code, _ := p.call(p.sess, "POST", spacePath(p.B.ID, "/categories"), map[string]any{"name": "Rent", "type": "expense", "is_default": true}); code != 201 {
		t.Fatal("create B")
	}
	if code, _ := p.call(p.sess, "POST", spacePath(p.B.ID, "/tags"), map[string]any{"name": "extra"}); code != 201 {
		t.Fatal("tag B")
	}
	body := map[string]any{"add_space_ids": []int64{p.A.ID, p.B.ID}, "categories": true, "tags": true}

	code, out := p.call(p.sess, "POST", "/api/sync-groups/diff", body)
	if code != 200 {
		t.Fatalf("diff %d %v", code, out)
	}
	d := out["data"].(map[string]any)
	tags := d["tags"].([]any)
	if len(tags) != 1 || tags[0].(map[string]any)["name"] != "extra" || len(d["spaces"].([]any)) != 2 {
		t.Fatalf("diff = %v", d)
	}

	if code, out := p.call(p.sess, "POST", "/api/sync-groups/apply", body); code != http.StatusUnprocessableEntity {
		t.Fatalf("apply without decisions %d %v", code, out)
	}
	body["decisions"] = []map[string]any{{"kind": "tags", "name": "extra", "keep": true}}
	if code, out := p.call(p.sess, "POST", "/api/sync-groups/apply", body); code != 200 {
		t.Fatalf("apply %d %v", code, out)
	}
	if !contains(p.names(p.A.ID, "tags"), "extra") {
		t.Fatal("the kept tag did not reach space A")
	}
}

func TestSyncGroupDoesNotLetSpacesBeProbed(t *testing.T) {
	p := newSyncPair(t)
	stranger := p.a.createUser("stranger@test.com", "secret1", auth.RoleUser)
	other, err := p.a.s.Spaces.Create(context.Background(), "Private", stranger)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"add_space_ids": []int64{p.A.ID, other.ID}, "categories": true}
	if code, _ := p.call(p.sess, "POST", "/api/sync-groups/apply", body); code != http.StatusNotFound {
		t.Fatalf("adding a space the caller does not belong to: %d", code)
	}
	// Not an admin of one's own space either: refused, not probed.
	bob := p.a.createUser("bob@test.com", "secret1", auth.RoleUser)
	_ = p.a.s.Spaces.SetMember(context.Background(), p.A.ID, bob, spaces.RoleEditor)
	_ = p.a.s.Spaces.SetMember(context.Background(), p.B.ID, bob, spaces.RoleEditor)
	body = map[string]any{"add_space_ids": []int64{p.A.ID, p.B.ID}, "categories": true}
	if code, _ := p.call(p.a.issue(bob, false), "POST", "/api/sync-groups/apply", body); code != http.StatusForbidden {
		t.Fatalf("an editor created a group: %d", code)
	}
}

package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/spaces"
)

// presetToken creates a token the way the Settings screen does and returns its secret.
func presetToken(a *testApp, sess *auth.Issued, body map[string]any) string {
	a.t.Helper()
	res := a.do("POST", "/api/auth/api-tokens", body, sess.Token, sess.CSRF)
	if res.StatusCode != http.StatusCreated {
		a.t.Fatalf("create token %v = %d", body, res.StatusCode)
	}
	return decodeJSON(a.t, res)["data"].(map[string]any)["token"].(string)
}

func tokenBody(extra map[string]any) map[string]any {
	body := confirmed(map[string]any{"name": "p", "all_spaces": true})
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func TestPresetsViewerEditorAdmin(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("admin@a.com", "secret1", "admin")
	sess := a.issue(u, false)
	res := a.do(
		"POST",
		"/api/currencies",
		map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "is_base": true, "rate": 1},
		sess.Token,
		sess.CSRF,
	)
	usd := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	res = a.do(
		"POST",
		"/api/accounts",
		map[string]any{"name": "Cash", "type": "cash", "currency_id": usd, "initial_balance": 100},
		sess.Token,
		sess.CSRF,
	)
	acc := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	res = a.do("POST", "/api/categories", map[string]any{"name": "Food", "type": "expense"}, sess.Token, sess.CSRF)
	cat := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	tx := map[string]any{
		"type":        "expense",
		"account_id":  acc,
		"category_id": cat,
		"amount":      5,
		"date":        time.Now().UTC().Format("2006-01-02"),
	}

	// Viewer: reads everything, changes nothing.
	viewer := presetToken(a, sess, tokenBody(map[string]any{"preset": "viewer"}))
	for _, p := range []string{"/api/accounts", "/api/transactions", "/api/reports/overview", fmt.Sprintf("/api/spaces/%d/members", a.space.ID)} {
		status(t, a.bearer("GET", p, nil, viewer), 200, "viewer reads "+p)
	}
	status(t, a.bearer("POST", "/api/transactions", tx, viewer), 403, "viewer cannot add")

	// Editor: adds and changes the data of a space (transactions, accounts, tags, ...), but
	// not categories, currencies, reports, transfers or the space itself, and cannot delete.
	editor := presetToken(a, sess, tokenBody(map[string]any{"preset": "editor"}))
	status(t, a.bearer("GET", "/api/accounts", nil, editor), 200, "editor reads")
	r := a.bearer("POST", "/api/transactions", tx, editor)
	if r.StatusCode != 201 {
		t.Fatalf("editor adds = %d", r.StatusCode)
	}
	id := int64(decodeJSON(t, r)["data"].(map[string]any)["id"].(float64))
	status(
		t,
		a.bearer(
			"POST",
			"/api/accounts",
			map[string]any{"name": "Card", "type": "cash", "currency_id": usd, "initial_balance": 0},
			editor,
		),
		201,
		"editor adds accounts",
	)
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "ok"}, editor), 201, "editor adds tags")
	status(
		t,
		a.bearer("POST", "/api/categories", map[string]any{"name": "No", "type": "expense"}, editor),
		403,
		"editor cannot add categories",
	)
	status(
		t,
		a.bearer(
			"POST",
			"/api/currencies",
			map[string]any{"code": "EUR", "name": "Euro", "symbol": "€", "decimals": 2, "rate": 1},
			editor,
		),
		403,
		"editor cannot add currencies",
	)
	status(
		t,
		a.bearer("PATCH", fmt.Sprintf("/api/spaces/%d", a.space.ID), map[string]any{"name": "Mine"}, editor),
		403,
		"editor cannot change the space",
	)
	res = a.bearer("DELETE", "/api/transactions/"+itoa(id), nil, editor)
	if res.StatusCode != 403 || decodeJSON(t, res)["message"] != "This API token has no transactions:delete permission." {
		t.Fatalf("editor deletes = %d", res.StatusCode)
	}

	// Editor with deleting switched on: transactions only.
	deleter := presetToken(a, sess, tokenBody(map[string]any{"preset": "editor", "allow_delete": true}))
	status(t, a.bearer("DELETE", "/api/transactions/"+itoa(id), nil, deleter), 204, "editor with delete")
	status(t, a.bearer("DELETE", "/api/accounts/"+itoa(acc), nil, deleter), 403, "delete is only for transactions")

	// Admin: everything in the space, but not the space's administration, and
	// never the sync groups.
	admin := presetToken(a, sess, tokenBody(map[string]any{"preset": "admin"}))
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "ok2"}, admin), 201, "admin writes")
	status(
		t,
		a.bearer("GET", fmt.Sprintf("/api/spaces/%d/backups", a.space.ID), nil, admin),
		403,
		"no space administration",
	)
	status(t, a.bearer("POST", "/api/sync-groups/apply", map[string]any{}, admin), 403, "tokens never change sync groups")

	for label, body := range map[string]map[string]any{
		"unknown preset":     tokenBody(map[string]any{"preset": "owner"}),
		"preset and list":    tokenBody(map[string]any{"preset": "viewer", "permissions": []string{"tags:read"}}),
		"unknown permission": tokenBody(map[string]any{"permissions": []string{"tags:erase"}}),
	} {
		status(t, a.do("POST", "/api/auth/api-tokens", body, sess.Token, sess.CSRF), 422, label)
	}
}

// A token is never more powerful than the person it belongs to, however high
// its grants: their role in each space is checked on every request, so a
// demotion reaches their tokens at once.
func TestTokenNeverExceedsItsOwner(t *testing.T) {
	a := newTestApp(t)
	owner := a.createUser("ed@a.com", "secret1", roleEditor)
	sess := a.issue(owner, false)
	admin := presetToken(a, sess, tokenBody(map[string]any{"preset": "admin"}))

	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "t"}, admin), 201, "editor owner writes")
	status(
		t,
		a.bearer("PATCH", fmt.Sprintf("/api/spaces/%d", a.space.ID), map[string]any{"name": "Mine"}, admin),
		403,
		"no space administration",
	)

	if err := a.svc.Spaces.SetMember(context.Background(), a.space.ID, owner, spaces.RoleViewer); err != nil {
		t.Fatal(err)
	}
	status(
		t,
		a.bearer("POST", "/api/tags", map[string]any{"name": "t2"}, admin),
		403,
		"demoted owner: the token cannot write",
	)
	status(t, a.bearer("DELETE", "/api/tags/1", nil, admin), 403, "nor delete")
	status(t, a.bearer("GET", "/api/tags", nil, admin), 200, "but still reads")
}

// A sync group mirrors categories and tags into the other spaces, so a token
// that changes them in one space must reach all of them, and its owner must be
// allowed to edit each (checked like for a session).
func TestTokenChangingSyncedDataNeedsEverySpaceOfTheGroup(t *testing.T) {
	p := newSyncPair(t)
	p.group(nil)
	inA := spacePath(p.A.ID, "/categories")
	cat := map[string]any{"name": "Synced", "type": "expense"}

	onlyA, _ := p.a.issueGrant(
		p.owner,
		auth.Grant{Permissions: []string{"categories:write"}, SpaceIDs: []int64{p.A.ID}},
		nil,
	)
	res := p.a.bearer("POST", inA, cat, onlyA)
	if res.StatusCode != 403 || !strings.Contains(fmt.Sprint(decodeJSON(t, res)["message"]), "syncs its categories") {
		t.Fatalf("token for one space of a group = %d", res.StatusCode)
	}
	if names := p.names(p.A.ID, "categories"); slices.Contains(names, "Synced") {
		t.Fatal("the refused change was applied")
	}

	both, _ := p.a.issueGrant(
		p.owner,
		auth.Grant{Permissions: []string{"categories:write"}, SpaceIDs: []int64{p.A.ID, p.B.ID}},
		nil,
	)
	status(t, p.a.bearer("POST", inA, cat, both), 201, "token for both spaces")
	for _, id := range []int64{p.A.ID, p.B.ID} {
		if names := p.names(id, "categories"); !slices.Contains(names, "Synced") {
			t.Fatalf("space %d has %v, want the mirrored category", id, names)
		}
	}

	// Reaching both is not enough when the owner may not edit one of them.
	other := p.a.createUser("other@test.com", "secret1", auth.RoleUser)
	ctx := context.Background()
	if err := p.a.svc.Spaces.SetMember(ctx, p.B.ID, other, spaces.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := p.a.svc.Spaces.SetMember(ctx, p.B.ID, p.owner, spaces.RoleViewer); err != nil {
		t.Fatal(err)
	}
	res = p.a.bearer("POST", inA, map[string]any{"name": "Second", "type": "expense"}, both)
	if res.StatusCode != 403 {
		t.Fatalf("owner is a viewer in B: got %d, want 403", res.StatusCode)
	}
	if names := p.names(p.A.ID, "categories"); slices.Contains(names, "Second") {
		t.Fatal("the change was applied to A although B refused it")
	}
}

//go:build integration

package integration

import (
	"strings"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/permissions"
)

// The case of issue #102: an assistant that may add and change data but not
// delete it makes a wrong transaction and takes it back itself. It cannot
// undo what a person changed after it, nor another connection's entries.
func TestMCPRollbackOfItsOwnMistakes(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("agent@a.com", "secret1", "admin")
	sp := a.space.ID
	perms, err := permissions.FromPreset(permissions.PresetEditor, false, permissions.MCPResources) // no delete grant
	if err != nil {
		t.Fatal(err)
	}
	grant := auth.Grant{Kind: permissions.KindMCP, Permissions: perms, AllSpaces: true}
	tok, _ := a.issueGrant(u, grant, nil)
	s := a.mcpSession(tok)
	acc := idOf(t, a.call(s, "create_account", map[string]any{
		"space_id": sp, "name": "Wallet", "type": "cash", "currency_code": "EUR", "initial_balance": "100.00",
	}))

	wrong := idOf(t, a.call(s, "create_transaction", mcpTx(sp, "expense", acc, "99.99", "oops", "2026-03-01")))
	if res := a.call(s, "delete_transaction", map[string]any{"space_id": sp, "id": wrong}); !res.IsError ||
		!strings.Contains(resultText(res), "transactions:delete") {
		t.Fatalf("delete without the grant = %s, want a refusal naming the permission", resultText(res))
	}

	actions := toolData(t, a.call(s, "list_my_actions", map[string]any{"space_id": sp, "entity_type": "transaction"})).([]any)
	if len(actions) != 1 || actions[0].(map[string]any)["rollbackable"] != true || actions[0].(map[string]any)["source"] != "mcp" {
		t.Fatalf("actions = %v, want one rollbackable mcp create", actions)
	}
	entry := int64(actions[0].(map[string]any)["id"].(float64))
	if res := a.call(s, "rollback_actions", map[string]any{"space_id": sp, "ids": []int64{entry}}); res.IsError {
		t.Fatalf("rollback: %s", resultText(res))
	}
	left := toolData(t, a.call(s, "list_transactions", map[string]any{"space_id": sp})).([]any)
	if len(left) != 0 {
		t.Fatalf("transactions after rollback = %v", left)
	}
	if res := a.call(s, "rollback_actions", map[string]any{"space_id": sp, "ids": []int64{entry}}); !res.IsError ||
		!strings.Contains(resultText(res), "already rolled back") {
		t.Fatalf("second rollback = %s", resultText(res))
	}

	// A person edits what the assistant made: the assistant may no longer undo it.
	mine := idOf(t, a.call(s, "create_transaction", mcpTx(sp, "expense", acc, "5.00", "coffee", "2026-03-02")))
	sess := a.issue(u, false)
	status(t, a.inSpace(sp, "PATCH", "/api/transactions/"+itoa(mine), map[string]any{"description": "Coffee at Joe's"}, sess), 200, "edit")
	mineEntry := int64(toolData(t, a.call(s, "list_my_actions", map[string]any{"space_id": sp, "entity_id": mine})).([]any)[0].(map[string]any)["id"].(float64))
	res := a.call(s, "rollback_actions", map[string]any{"space_id": sp, "ids": []int64{mineEntry}})
	if !res.IsError || !strings.Contains(resultText(res), "changed again") {
		t.Fatalf("rollback after a person's edit = %s, want a refusal", resultText(res))
	}
	if got := toolData(t, a.call(s, "get_transaction", map[string]any{"space_id": sp, "id": mine})).(map[string]any); got["description"] != "Coffee at Joe's" {
		t.Fatalf("the person's edit was lost: %v", got)
	}

	// Another connection cannot see or undo these entries.
	tok2, _ := a.issueGrant(u, grant, nil)
	s2 := a.mcpSession(tok2)
	if other := toolData(t, a.call(s2, "list_my_actions", map[string]any{"space_id": sp})).([]any); len(other) != 0 {
		t.Fatalf("another token sees %v", other)
	}
	if res := a.call(s2, "rollback_actions", map[string]any{"space_id": sp, "ids": []int64{mineEntry}}); !res.IsError {
		t.Fatalf("another token rolled back %s", resultText(res))
	}

	// The space admin sees everything and clears it by hand, never a token.
	all := decodeJSON(t, a.inSpace(sp, "GET", "/api/audit-log", nil, sess))["data"].([]any)
	sources := map[string]bool{}
	for _, e := range all {
		sources[e.(map[string]any)["source"].(string)] = true
	}
	if !sources["mcp"] || !sources["ui"] {
		t.Fatalf("the admin's view has sources %v, want mcp and ui", sources)
	}
	for _, c := range []struct{ query, field, want string }{{"source=mcp", "source", "mcp"}, {"source=ui", "source", "ui"}, {"op=update", "op", "update"}} {
		got := decodeJSON(t, a.inSpace(sp, "GET", "/api/audit-log?"+c.query, nil, sess))["data"].([]any)
		if len(got) == 0 {
			t.Fatalf("?%s found nothing", c.query)
		}
		for _, e := range got {
			if e.(map[string]any)[c.field] != c.want {
				t.Fatalf("?%s returned %v", c.query, e)
			}
		}
	}
	if code := a.bearer("DELETE", "/api/spaces/"+itoa(sp)+"/audit-log", nil, tok).StatusCode; code != 403 {
		t.Fatalf("a token cleared the log: %d", code)
	}
	cleared := decodeJSON(t, a.inSpace(sp, "DELETE", "/api/audit-log", nil, sess))["data"].(map[string]any)
	if cleared["removed"].(float64) < 3 {
		t.Fatalf("cleared = %v", cleared)
	}
	left2 := decodeJSON(t, a.inSpace(sp, "GET", "/api/audit-log", nil, sess))["data"].([]any)
	if len(left2) != 1 || left2[0].(map[string]any)["op"] != "cleanup" || left2[0].(map[string]any)["userId"] == nil {
		t.Fatalf("log after clearing = %v, want one cleanup entry naming the user", left2)
	}
}

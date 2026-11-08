//go:build integration

package integration

import (
	"strconv"
	"strings"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/permissions"
)

// An external id makes a retry safe: the second create fails and names the
// first transaction. The server records the source as mcp, so a client cannot
// claim to be the CSV import, and find_similar_transactions sees the same line.
func TestMCPExternalIDAndFindSimilar(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	perms, err := permissions.FromPreset(permissions.PresetAdmin, true, permissions.MCPResources)
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := a.issueGrant(u, auth.Grant{Kind: permissions.KindMCP, Permissions: perms, AllSpaces: true}, nil)
	s := a.mcpSession(tok)
	acc := idOf(t, a.call(s, "create_account", map[string]any{
		"space_id": sp, "name": "Wallet", "type": "cash", "currency_code": "EUR", "initial_balance": "100.00",
	}))
	args := mcpTx(sp, "expense", acc, "10.50", "bread", "2026-03-01")
	args["external_id"] = "F6HDVMNQ-5105"
	args["external_source"] = "csv" // not a field: the server decides
	if res := a.call(s, "create_transaction", args); !res.IsError {
		t.Fatalf("an unknown argument was accepted: %s", resultText(res))
	}
	delete(args, "external_source")
	first := idOf(t, a.call(s, "create_transaction", args))
	again := a.call(s, "create_transaction", args)
	if !again.IsError || !strings.Contains(resultText(again), "id "+strconv.FormatInt(first, 10)) {
		t.Fatalf("retry = %v %s, want an error naming transaction %d", again.IsError, resultText(again), first)
	}
	got := toolData(t, a.call(s, "list_transactions", map[string]any{"space_id": sp, "external_id": "F6HDVMNQ-5105"})).([]any)
	if len(got) != 1 || got[0].(map[string]any)["externalSource"] != "mcp" {
		t.Fatalf("lookup by key = %v", got)
	}

	res := a.call(s, "find_similar_transactions", map[string]any{"space_id": sp, "items": []map[string]any{
		{"key": "line 1", "date": "2026-03-02", "amount": "10.50", "currency": "EUR"},
		{"key": "line 2", "date": "2026-03-02", "amount": "10.51", "currency": "EUR"},
	}})
	if res.IsError {
		t.Fatalf("find_similar_transactions: %s", resultText(res))
	}
	items := toolData(t, res).([]any)
	if len(items[0].(map[string]any)["matches"].([]any)) != 1 || len(items[1].(map[string]any)["matches"].([]any)) != 0 {
		t.Fatalf("matches = %s", resultText(res))
	}
}

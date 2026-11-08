//go:build integration

package integration

import (
	"strings"
	"testing"
)

// update_transactions changes a list in one save, or nothing when one of them
// is refused, and says which one. Only the fields sent change.
func TestMCPUpdateTransactionsIsAllOrNone(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	s := a.mcpFull(u)
	acc := idOf(t, a.call(s, "create_account", map[string]any{
		"space_id": sp, "name": "Wallet", "type": "cash", "currency_code": "EUR", "initial_balance": "100.00",
	}))
	var ids []int64
	for _, d := range []string{"bread", "milk"} {
		ids = append(ids, idOf(t, a.call(s, "create_transaction", mcpTx(sp, "expense", acc, "10.50", d, "2026-03-01"))))
	}
	desc := func(i int) any {
		items := toolData(t, a.call(s, "list_transactions", map[string]any{"space_id": sp})).([]any)
		for _, it := range items {
			if m := it.(map[string]any); int64(m["id"].(float64)) == ids[i] {
				return m["description"]
			}
		}
		return nil
	}

	res := a.call(s, "update_transactions", map[string]any{"space_id": sp, "transactions": []map[string]any{
		{"id": ids[0], "description": "rye bread"}, {"id": ids[1], "date": "2026-03-02"},
	}})
	if res.IsError {
		t.Fatalf("update_transactions: %s", resultText(res))
	}
	out := toolData(t, res).([]any)
	if len(out) != 2 || out[0].(map[string]any)["description"] != "rye bread" || out[1].(map[string]any)["description"] != "milk" {
		t.Fatalf("answer = %s, want both in order with the unsent fields kept", resultText(res))
	}

	bad := a.call(s, "update_transactions", map[string]any{"space_id": sp, "transactions": []map[string]any{
		{"id": ids[0], "description": "changed"}, {"id": 999999, "description": "ghost"},
	}})
	if !bad.IsError || !strings.Contains(resultText(bad), "transaction 2") {
		t.Fatalf("an unknown id = %v %s", bad.IsError, resultText(bad))
	}
	if desc(0) != "rye bread" {
		t.Fatalf("a refused list changed a transaction: %v", desc(0))
	}
	if empty := a.call(s, "update_transactions", map[string]any{"space_id": sp, "transactions": []map[string]any{}}); !empty.IsError {
		t.Fatal("an empty list must be refused")
	}
}

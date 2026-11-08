//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestMCPReconcileAccounts(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	s := a.mcpFull(u)

	acc := idOf(t, a.call(s, "create_account", map[string]any{
		"space_id": sp, "name": "Wallet", "type": "cash", "currency_code": "EUR", "initial_balance": "100.00",
	}))
	if res := a.call(s, "create_transaction", mcpTx(sp, "expense", acc, "12.34", "lunch", "2026-03-01")); res.IsError {
		t.Fatal(resultText(res))
	}

	// Exact strings in; the answer has actual, expected and difference.
	res := a.call(s, "reconcile_account", map[string]any{
		"space_id": sp, "date": "2026-03-05",
		"accounts": []map[string]any{{"account_id": acc, "actual_balance": "0.10", "note": "statement"}},
	})
	if res.IsError {
		t.Fatalf("reconcile_account: %s", resultText(res))
	}
	text := resultText(res)
	for _, want := range []string{`"actualBalance":0.1`, `"expectedBalance":87.66`, `"difference":-87.56`, `"note":"statement"`} {
		if !strings.Contains(text, want) {
			t.Errorf("answer lacks %s: %s", want, text)
		}
	}
	// It creates no transaction.
	if got := resultText(a.call(s, "list_transactions", map[string]any{"space_id": sp})); strings.Contains(got, "econcil") {
		t.Errorf("a transaction was created: %s", got)
	}

	// A 15+ digit value stays exact; the same account and date is replaced.
	res = a.call(s, "reconcile_account", map[string]any{
		"space_id": sp, "date": "2026-03-05",
		"accounts": []map[string]any{{"account_id": acc, "actual_balance": "1234567890123.45"}},
	})
	if res.IsError || !strings.Contains(resultText(res), "1234567890123.45") {
		t.Fatalf("large value: %v %s", res.IsError, resultText(res))
	}
	list := a.call(s, "list_reconciliations", map[string]any{
		"space_id": sp, "account_ids": []int64{acc}, "from": "2026-03-01", "to": "2026-03-31",
	})
	points := toolData(t, list).([]any)
	if len(points) != 1 {
		t.Fatalf("points = %v", points)
	}

	// All or none; the future is refused.
	bad := a.call(s, "reconcile_account", map[string]any{
		"space_id": sp, "date": "2999-01-01",
		"accounts": []map[string]any{{"account_id": acc, "actual_balance": "1"}},
	})
	if !bad.IsError {
		t.Errorf("a future date was accepted: %s", resultText(bad))
	}

	id := int64(points[0].(map[string]any)["id"].(float64))
	if res := a.call(s, "delete_reconciliation", map[string]any{"space_id": sp, "id": id}); res.IsError {
		t.Fatalf("delete_reconciliation: %s", resultText(res))
	}
	if left := toolData(t, a.call(s, "list_reconciliations", map[string]any{"space_id": sp})).([]any); len(left) != 0 {
		t.Errorf("the point is still there: %v", left)
	}
}

func TestMCPReconcileNeedsItsPermissions(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	body := map[string]any{"space_id": sp, "date": "2026-03-05",
		"accounts": []map[string]any{{"account_id": 1, "actual_balance": "1"}}}

	res := a.call(a.mcpGrant(u, "accounts:read"), "reconcile_account", body)
	if !res.IsError || !strings.Contains(resultText(res), "accounts:write") {
		t.Errorf("reconcile_account with accounts:read = %v %s", res.IsError, resultText(res))
	}
	res = a.call(a.mcpGrant(u, "accounts:write"), "delete_reconciliation", map[string]any{"space_id": sp, "id": 1})
	if !res.IsError || !strings.Contains(resultText(res), "accounts:delete") {
		t.Errorf("delete_reconciliation with accounts:write = %v %s", res.IsError, resultText(res))
	}
	res = a.call(a.mcpGrant(u, "transactions:read"), "list_reconciliations", map[string]any{"space_id": sp})
	if !res.IsError || !strings.Contains(resultText(res), "accounts:read") {
		t.Errorf("list_reconciliations without accounts:read = %v %s", res.IsError, resultText(res))
	}
}

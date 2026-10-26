//go:build integration

package httpserver

import (
	"fmt"
	"net/http"
	"testing"
)

func groupLegs(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	var legs []map[string]any
	for _, l := range out["data"].(map[string]any)["legs"].([]any) {
		legs = append(legs, l.(map[string]any))
	}
	return legs
}

// Rent of 45 USD paid with 50 USD, the change given back in dinars: an expense
// and an exchange that belong together.
func TestTransactionGroupLifecycle(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("group@test.com", "secret1", roleEditor), false)
	usdAcc, catID := seedMoney(t, a, sess)
	rsd := apiID(t, a, sess, "/api/currencies", map[string]any{
		"code": "RSD", "name": "Dinar", "symbol": "din", "decimals": 2, "rate": 0.0085,
	})
	rsdAcc := apiID(t, a, sess, "/api/accounts", map[string]any{
		"name": "Dinars", "type": "cash", "currency_id": rsd, "initial_balance": 1000,
	})
	body := func(rent, exchanged, change float64) map[string]any {
		return map[string]any{"legs": []map[string]any{
			{"type": "expense", "account_id": usdAcc, "category_id": catID, "amount": rent, "date": "2024-01-15"},
			{
				"type": "transfer", "account_id": usdAcc, "to_account_id": rsdAcc,
				"amount": exchanged, "to_amount": change, "date": "2024-01-15",
			},
		}}
	}

	res := a.do("POST", "/api/transaction-groups", body(45, 5, 500), sess.Token, sess.CSRF)
	out := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d %v", res.StatusCode, out)
	}
	uuid := out["data"].(map[string]any)["uuid"].(string)
	legs := groupLegs(t, out)
	if len(legs) != 2 {
		t.Fatalf("%d legs, want 2", len(legs))
	}
	for _, leg := range legs {
		if leg["groupUuid"] != uuid {
			t.Errorf("leg groupUuid = %v, want %s", leg["groupUuid"], uuid)
		}
	}

	// The list shows each leg with its group.
	listed := decodeJSON(t, a.do("GET", "/api/transactions", nil, sess.Token, ""))["data"].([]any)
	if len(listed) != 2 || listed[0].(map[string]any)["groupUuid"] != uuid {
		t.Errorf("list = %v", listed)
	}

	// One leg changes only with its group.
	legID := int64(legs[0]["id"].(float64))
	status(t, a.do("PUT", fmt.Sprintf("/api/transactions/%d", legID),
		map[string]any{"amount": 1}, sess.Token, sess.CSRF), http.StatusUnprocessableEntity, "update one leg")
	status(t, a.do("DELETE", fmt.Sprintf("/api/transactions/%d", legID), nil, sess.Token, sess.CSRF),
		http.StatusUnprocessableEntity, "delete one leg")
	status(t, a.do("POST", fmt.Sprintf("/api/transactions/%d/duplicate", legID), nil, sess.Token, sess.CSRF),
		http.StatusUnprocessableEntity, "duplicate one leg")

	// Not a payment: one leg, or legs on different days.
	status(t, a.do("POST", "/api/transaction-groups", map[string]any{"legs": []map[string]any{
		{"type": "expense", "account_id": usdAcc, "amount": 1, "date": "2024-01-15"},
	}}, sess.Token, sess.CSRF), http.StatusUnprocessableEntity, "one leg")

	// The whole group is replaced in one step.
	res = a.do("PUT", "/api/transaction-groups/"+uuid, body(40, 10, 1000), sess.Token, sess.CSRF)
	out = decodeJSON(t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update = %d %v", res.StatusCode, out)
	}
	if got := out["data"].(map[string]any)["uuid"]; got != uuid {
		t.Errorf("uuid = %v after update, want %s", got, uuid)
	}
	got := decodeJSON(t, a.do("GET", "/api/transaction-groups/"+uuid, nil, sess.Token, ""))
	if legs = groupLegs(t, got); len(legs) != 2 || legs[0]["amount"] != float64(40) {
		t.Errorf("legs = %v", legs)
	}
	status(t, a.do("PUT", "/api/transaction-groups/nope", body(1, 1, 1), sess.Token, sess.CSRF),
		http.StatusNotFound, "update unknown group")

	status(t, a.do("DELETE", "/api/transaction-groups/"+uuid, nil, sess.Token, sess.CSRF),
		http.StatusNoContent, "delete group")
	status(t, a.do("GET", "/api/transaction-groups/"+uuid, nil, sess.Token, ""), http.StatusNotFound, "get deleted")
	rest := decodeJSON(t, a.do("GET", "/api/transactions", nil, sess.Token, ""))["data"].([]any)
	if len(rest) != 0 {
		t.Errorf("%d transactions left", len(rest))
	}
}

// A payment can be recorded, replaced and removed through MCP, with exact
// decimal strings.
func TestMCPTransactionGroup(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	s := a.mcpFull(u)
	account := func(name string) int64 {
		return idOf(t, a.call(s, "create_account", map[string]any{
			"space_id": sp, "name": name, "type": "cash", "currency_code": "EUR", "initial_balance": "100.00",
		}))
	}
	cash, card := account("Cash"), account("Card")
	balances := func() map[string]float64 {
		t.Helper()
		out := map[string]float64{}
		for _, row := range toolData(t, a.call(s, "list_accounts", map[string]any{"space_id": sp})).([]any) {
			acc := row.(map[string]any)
			out[acc["name"].(string)] = acc["currentBalance"].(float64)
		}
		return out
	}
	legs := func(cashAmount, cardAmount string) []map[string]any {
		return []map[string]any{
			{"type": "expense", "account_id": cash, "amount": cashAmount, "date": "2026-03-01"},
			{"type": "expense", "account_id": card, "amount": cardAmount, "date": "2026-03-01"},
		}
	}

	res := a.call(s, "create_transaction_group", map[string]any{"space_id": sp, "legs": legs("15.50", "24.50")})
	if res.IsError {
		t.Fatalf("create_transaction_group: %s", resultText(res))
	}
	uuid := toolData(t, res).(map[string]any)["uuid"].(string)
	if got := balances(); got["Cash"] != 84.5 || got["Card"] != 75.5 {
		t.Fatalf("balances = %v, want Cash 84.5 and Card 75.5", got)
	}

	res = a.call(s, "update_transaction_group", map[string]any{"space_id": sp, "uuid": uuid, "legs": legs("10", "10")})
	if res.IsError {
		t.Fatalf("update_transaction_group: %s", resultText(res))
	}
	if got := balances(); got["Cash"] != 90 || got["Card"] != 90 {
		t.Fatalf("balances = %v after update, want 90 and 90", got)
	}

	if bad := a.call(s, "create_transaction_group", map[string]any{
		"space_id": sp, "legs": legs("1", "1")[:1],
	}); !bad.IsError {
		t.Fatalf("one leg is not a payment: %s", resultText(bad))
	}

	res = a.call(s, "delete_transaction_group", map[string]any{"space_id": sp, "uuid": uuid})
	if res.IsError {
		t.Fatalf("delete_transaction_group: %s", resultText(res))
	}
	if got := balances(); got["Cash"] != 100 || got["Card"] != 100 {
		t.Fatalf("balances = %v after delete, want 100 and 100", got)
	}
}

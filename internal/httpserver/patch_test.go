package httpserver

import (
	"testing"
)

// patchCase creates a resource, then checks that a PATCH sending one field
// keeps the others and that an explicit null clears a nullable field.
type patchCase struct {
	name     string
	create   func(accID, catID int64) (path string, body map[string]any)
	change   map[string]any // one field, the rest must survive
	keep     map[string]any // response fields that must not change
	clearKey string         // request field set to null
	clearOut string         // response field that must become null
}

func TestPatchKeepsOmittedFieldsAndNullClears(t *testing.T) {
	cases := []patchCase{
		{
			name: "transaction",
			create: func(accID, catID int64) (string, map[string]any) {
				return "/api/transactions", map[string]any{
					"type": "expense", "account_id": accID, "category_id": catID, "amount": 25,
					"description": "Lunch", "date": "2026-09-01",
				}
			},
			change:   map[string]any{"amount": 30},
			keep:     map[string]any{"description": "Lunch", "date": "2026-09-01", "amount": float64(30)},
			clearKey: "description", clearOut: "description",
		},
		{
			name: "budget",
			create: func(_, catID int64) (string, map[string]any) {
				return "/api/budgets", map[string]any{
					"name": "Food", "amount": 100, "period": "monthly", "category_ids": []int64{catID},
					"start_date": "2026-01-01", "end_date": "2026-12-31", "notify_at_percent": 80, "is_active": true,
				}
			},
			change:   map[string]any{"is_active": false},
			keep:     map[string]any{"name": "Food", "endDate": "2026-12-31", "notifyAtPercent": float64(80), "isActive": false},
			clearKey: "end_date", clearOut: "endDate",
		},
		{
			name: "automation rule",
			create: func(_, catID int64) (string, map[string]any) {
				return "/api/automation-rules", map[string]any{
					"name": "Groceries", "description": "Big shops", "trigger_type": "on_transaction_create", "priority": 7,
					"conditions": map[string]any{"match": "all", "conditions": []map[string]any{
						{"field": "amount", "op": "gte", "value": 10},
					}},
					"actions":   []map[string]any{{"type": "set_category", "category_id": catID}},
					"is_active": true,
				}
			},
			change:   map[string]any{"is_active": false},
			keep:     map[string]any{"name": "Groceries", "description": "Big shops", "priority": float64(7), "isActive": false},
			clearKey: "description", clearOut: "description",
		},
		{
			name: "debt",
			create: func(accID, _ int64) (string, map[string]any) {
				return "/api/debts", map[string]any{
					"origin": "new", "name": "Loan", "debt_type": "owed_to_me", "account_id": accID, "amount": 200,
					"date": "2026-09-01", "due_date": "2027-01-01", "counterparty": "Ivan", "description": "Car",
				}
			},
			change:   map[string]any{"name": "Loan to Ivan"},
			keep:     map[string]any{"name": "Loan to Ivan", "dueDate": "2027-01-01", "counterparty": "Ivan", "description": "Car"},
			clearKey: "due_date", clearOut: "dueDate",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t)
			u := a.createUser("patch@test.com", "secret1", roleEditor)
			sess := a.issue(u, false)
			accID, catID := seedMoney(t, a, sess)

			path, body := tc.create(accID, catID)
			res := a.do("POST", path, body, sess.Token, sess.CSRF)
			created := decodeJSON(t, res)
			if res.StatusCode != 201 {
				t.Fatalf("create %d %v", res.StatusCode, created)
			}
			item := path + "/" + itoa(int64(created["data"].(map[string]any)["id"].(float64)))

			res = a.do("PATCH", item, tc.change, sess.Token, sess.CSRF)
			got := decodeJSON(t, res)
			if res.StatusCode != 200 {
				t.Fatalf("patch %d %v", res.StatusCode, got)
			}
			data := got["data"].(map[string]any)
			for k, want := range tc.keep {
				if data[k] != want {
					t.Errorf("%s = %v, want %v", k, data[k], want)
				}
			}

			res = a.do("PATCH", item, map[string]any{tc.clearKey: nil}, sess.Token, sess.CSRF)
			got = decodeJSON(t, res)
			if res.StatusCode != 200 {
				t.Fatalf("clear %d %v", res.StatusCode, got)
			}
			if v := got["data"].(map[string]any)[tc.clearOut]; v != nil {
				t.Errorf("%s = %v after null, want null", tc.clearOut, v)
			}
		})
	}
}

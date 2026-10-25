//go:build integration

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

func TestPatchKeepsOmittedFieldsNullClears(t *testing.T) {
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
			change: map[string]any{"name": "Loan to Ivan"},
			keep: map[string]any{
				"name":         "Loan to Ivan",
				"dueDate":      "2027-01-01",
				"counterparty": "Ivan",
				"description":  "Car",
			},
			clearKey: "due_date", clearOut: "dueDate",
		},
		{
			name: "recurring transaction",
			create: func(accID, catID int64) (string, map[string]any) {
				return "/api/recurring", map[string]any{
					"type": "expense", "account_id": accID, "category_id": catID, "amount": 50,
					"description": "Gym", "frequency": "monthly", "interval": 1, "day_of_month": 5,
					"start_date": "2030-01-05", "end_date": "2031-01-05", "is_active": true,
				}
			},
			change: map[string]any{"is_active": false},
			keep: map[string]any{
				"description": "Gym",
				"endDate":     "2031-01-05",
				"dayOfMonth":  float64(5),
				"amount":      float64(50),
				"isActive":    false,
			},
			clearKey: "end_date", clearOut: "endDate",
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

// A PATCH replaces category_ids / tag_ids only when the body sends them.
func TestPatchReplacesLinkListsOnlyWhenSent(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("patchlists@test.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	accID, catID := seedMoney(t, a, sess)

	res := a.do("POST", "/api/tags", map[string]any{"name": "Trip"}, sess.Token, sess.CSRF)
	tagID := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))

	count := func(data map[string]any, key string) int { return len(data[key].([]any)) }
	patch := func(path string, body map[string]any) map[string]any {
		t.Helper()
		res := a.do("PATCH", path, body, sess.Token, sess.CSRF)
		got := decodeJSON(t, res)
		if res.StatusCode != 200 {
			t.Fatalf("patch %s %d %v", path, res.StatusCode, got)
		}
		return got["data"].(map[string]any)
	}

	res = a.do("POST", "/api/budgets", map[string]any{
		"name": "Food", "amount": 100, "period": "monthly",
		"category_ids": []int64{catID}, "tag_ids": []int64{tagID},
	}, sess.Token, sess.CSRF)
	created := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("create budget %d %v", res.StatusCode, created)
	}
	budget := "/api/budgets/" + itoa(int64(created["data"].(map[string]any)["id"].(float64)))

	data := patch(budget, map[string]any{"name": "Groceries"})
	if count(data, "categories") != 1 || count(data, "tags") != 1 {
		t.Errorf("budget omitted lists changed: %v", data)
	}
	data = patch(budget, map[string]any{"tag_ids": []int64{}})
	if count(data, "categories") != 1 || count(data, "tags") != 0 {
		t.Errorf("budget tag_ids [] should clear tags only: %v", data)
	}

	res = a.do("POST", "/api/recurring", map[string]any{
		"type": "expense", "account_id": accID, "category_id": catID, "amount": 50,
		"frequency": "monthly", "interval": 1, "day_of_month": 5, "start_date": "2030-01-05",
		"tag_ids": []int64{tagID},
	}, sess.Token, sess.CSRF)
	created = decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("create recurring %d %v", res.StatusCode, created)
	}
	rec := "/api/recurring/" + itoa(int64(created["data"].(map[string]any)["id"].(float64)))

	data = patch(rec, map[string]any{"amount": 60})
	if count(data, "tags") != 1 {
		t.Errorf("recurring omitted tag_ids changed: %v", data)
	}
	data = patch(rec, map[string]any{"tag_ids": []int64{}})
	if count(data, "tags") != 0 {
		t.Errorf("recurring tag_ids [] should clear tags: %v", data)
	}
}

func TestResourceLookupUnknownIDIs404(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("lookup@test.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	for _, path := range []string{"/api/budgets/999", "/api/recurring/999", "/api/automation-rules/999", "/api/tags/abc"} {
		if res := a.do("GET", path, nil, sess.Token, sess.CSRF); res.StatusCode != 404 {
			t.Errorf("GET %s = %d, want 404", path, res.StatusCode)
		}
	}
}

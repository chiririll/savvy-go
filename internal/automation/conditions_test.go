package automation

import (
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

func testTx() *ledger.Transaction {
	desc, cat := "Coffee at Blue Bottle", int64(7)
	return &ledger.Transaction{
		Type:        "expense",
		Amount:      money.FromDecimal(decimal.RequireFromString("12.50"), money.Unit{ID: 1, Decimals: 2}),
		Description: &desc,
		CategoryID:  &cat,
		Account:     &ledger.Account{ID: 3, Name: "Wallet"},
		Tags:        []ledger.Tag{{ID: 1, Name: "work"}, {ID: 2, Name: "cafe"}},
	}
}

func TestEvaluateCondition(t *testing.T) {
	tx := testTx()
	for _, c := range []struct {
		name  string
		field string
		op    string
		value any
		want  bool
	}{
		{"equals", "type", "equals", "expense", true},
		{"equals other", "type", "equals", "income", false},
		{"default operator is equals", "type", "", "expense", true},
		{"not_equals", "type", "not_equals", "income", true},
		{"in", "type", "in", []any{"income", "expense"}, true},
		{"not_in", "type", "not_in", []any{"income"}, true},
		{"gt exact", "amount", "gt", "12.49", true},
		{"gt equal", "amount", "gt", "12.50", false},
		{"gte equal", "amount", "gte", "12.50", true},
		{"lt", "amount", "lt", "12.51", true},
		{"lte equal", "amount", "lte", "12.50", true},
		{"between inclusive", "amount", "between", []any{"12.50", "20"}, true},
		{"between outside", "amount", "between", []any{"1", "12.49"}, false},
		{"between needs two values", "amount", "between", []any{"1"}, false},
		{"gt with a non-number", "amount", "gt", "lots", false},
		{"contains ignores case", "description", "contains", "BLUE", true},
		{"not_contains", "description", "not_contains", "tesla", true},
		{"starts_with", "description", "starts_with", "coffee", true},
		{"ends_with", "description", "ends_with", "bottle", true},
		{"matches is case-insensitive", "description", "matches", `^coffee .* bottle$`, true},
		{"matches with a bad pattern", "description", "matches", `(`, false},
		{"is_null on a set field", "description", "is_null", nil, false},
		{"is_null on an unset field", "date", "is_null", nil, true},
		{"is_not_null", "category_id", "is_not_null", nil, true},
		{"nested field", "account.name", "equals", "Wallet", true},
		{"unknown field", "nope", "equals", "x", false},
		{"unknown operator", "type", "resembles", "expense", false},
		{"has_any", "tags", "has_any", []any{2.0, 9.0}, true},
		{"has_any none", "tags", "has_any", []any{9.0}, false},
		{"has_all", "tags", "has_all", []any{1.0, 2.0}, true},
		{"has_all missing one", "tags", "has_all", []any{1.0, 9.0}, false},
		{"has_none", "tags", "has_none", []any{9.0}, true},
		{"has_none present", "tags", "has_none", []any{1.0}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			cond := map[string]any{"field": c.field, "op": c.op, "value": c.value}
			if got := evaluateCondition(cond, tx); got != c.want {
				t.Fatalf("got %v", got)
			}
		})
	}
}

func TestEvaluateConditionsGroup(t *testing.T) {
	tx := testTx()
	yes := map[string]any{"field": "type", "op": "equals", "value": "expense"}
	no := map[string]any{"field": "type", "op": "equals", "value": "income"}
	group := func(match string, conds ...map[string]any) map[string]any {
		list := make([]any, len(conds))
		for i, c := range conds {
			list[i] = c
		}
		return map[string]any{"match": match, "conditions": list}
	}
	for _, c := range []struct {
		name  string
		group map[string]any
		want  bool
	}{
		{"no group matches everything", nil, true},
		{"empty conditions match everything", group("all"), true},
		{"all of true and false", group("all", yes, no), false},
		{"all of true", group("all", yes, yes), true},
		{"any of true and false", group("any", no, yes), true},
		{"any of false", group("any", no, no), false},
		{"match defaults to all", map[string]any{"conditions": []any{yes, no}}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := evaluateConditions(c.group, tx); got != c.want {
				t.Fatalf("got %v", got)
			}
		})
	}
}

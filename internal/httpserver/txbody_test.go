//go:build integration

package httpserver

import "fmt"

// flowBody is the body of an income or expense of amount on an account: one
// payment (signed by the type) and one line without a category.
func flowBody(typ string, account int64, amount any, date string) map[string]any {
	return flowIn(typ, account, 0, amount, date)
}

// flowIn is flowBody with a category (none when 0).
func flowIn(typ string, account, category int64, amount any, date string) map[string]any {
	line := fmt.Sprint(amount)
	paid := line
	if typ == "expense" {
		paid = "-" + line
	}
	l := map[string]any{"amount": line}
	if category != 0 {
		l["category_id"] = category
	}
	return map[string]any{
		"type": typ, "date": date,
		"payments": []map[string]any{{"account_id": account, "amount": paid}},
		"lines":    []map[string]any{l},
	}
}

package importer

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestProcessImportRow(t *testing.T) {
	mapping := map[string]any{"date": float64(0), "amount": float64(1), "description": float64(2), "category": float64(3)}
	options := map[string]any{"date_format": "ISO", "amount_format": "US"}
	cases := []struct {
		name    string
		row     []string
		mapping map[string]any
		want    importRow // compared on err, typ, amount, desc, category
	}{
		{"expense from sign", []string{"2024-01-05", "-12.50", " Coffee ", " Food "}, mapping,
			importRow{typ: "expense", amount: decimal.RequireFromString("12.5"), desc: "Coffee", category: "Food"}},
		{"income from sign", []string{"2024-01-05", "100", "Pay", ""}, mapping,
			importRow{typ: "income", amount: decimal.RequireFromString("100"), desc: "Pay"}},
		{"bad date", []string{"05 Jan", "1", "", ""}, mapping, importRow{typ: "expense", err: "Invalid date: 05 Jan"}},
		{"future date", []string{"2999-01-01", "1", "", ""}, mapping, importRow{typ: "expense", err: "Date is in the future."}},
		{"bad amount", []string{"2024-01-05", "lots", "", ""}, mapping, importRow{typ: "expense", err: "Invalid amount: lots"}},
		{"short row", []string{"2024-01-05"}, mapping, importRow{typ: "expense", err: "Row is too short."}},
		{"amount not mapped", []string{"2024-01-05", "1"}, map[string]any{"date": float64(0)}, importRow{typ: "expense", err: "Amount column is not mapped."}},
		{"date not mapped", []string{"2024-01-05", "1"}, map[string]any{"amount": float64(1)}, importRow{typ: "expense", err: "Date column is not mapped."}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := processImportRow(c.row, c.mapping, options, 1)
			if got.err != c.want.err || got.typ != c.want.typ || got.desc != c.want.desc || got.category != c.want.category || !got.amount.Equal(c.want.amount) {
				t.Fatalf("got %+v want %+v", got, c.want)
			}
		})
	}
}

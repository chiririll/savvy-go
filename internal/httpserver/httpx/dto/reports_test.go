package dto_test

import (
	"github.com/shopspring/decimal"

	"savvy-go/internal/importer"
	"testing"

	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/money"
	"savvy-go/internal/reports"
)

func TestReportShapes(t *testing.T) {
	unit := money.Unit{ID: 1, Decimals: 2}
	usd := money.Zero(unit)
	ten := money.New(1000, unit)

	// Nothing to compare with: optionals are null, lists are [], the currency is
	// null while there is no base.
	got := decode(t, dto.NewOverview(reports.OverviewReport{
		Income: reports.Metric[money.Money]{Value: ten},
	}))
	income := got["income"].(map[string]any)
	if income["value"] != 10.0 || income["previous"] != nil {
		t.Errorf("income = %v", income)
	}
	if s, ok := income["sparkline"].([]any); !ok || len(s) != 0 {
		t.Errorf("sparkline = %v, want []", income["sparkline"])
	}
	if got["currency"] != nil {
		t.Errorf("currency = %v, want null", got["currency"])
	}

	// A flow node keeps the colour under itemStyle, where the chart reads it.
	flow := decode(t, dto.NewMoneyFlow(reports.MoneyFlowReport{
		Nodes: []reports.FlowNode{{Name: "Salary", Color: "#22c55e"}}, Currency: "USD",
	}))
	node := flow["nodes"].([]any)[0].(map[string]any)
	if node["itemStyle"].(map[string]any)["color"] != "#22c55e" || flow["currency"] != "USD" {
		t.Errorf("flow = %v", flow)
	}
	if links, ok := flow["links"].([]any); !ok || len(links) != 0 {
		t.Errorf("links = %v, want []", flow["links"])
	}

	// The comparison fields of a cash-flow point are left out, not null.
	cash := decode(t, dto.NewCashFlow(reports.CashFlowReport{Items: []reports.CashFlowPoint{
		{Date: "2024-01-01", Income: usd, Expenses: usd, Balance: usd},
		{Date: "2024-01-02", Income: usd, Expenses: usd, Balance: usd, PrevIncome: &ten, PrevExpenses: &ten, PrevBalance: &ten},
	}}))
	items := cash["items"].([]any)
	if _, has := items[0].(map[string]any)["prevIncome"]; has {
		t.Errorf("prevIncome present without a comparison: %v", items[0])
	}
	if items[1].(map[string]any)["prevBalance"] != 10.0 {
		t.Errorf("prevBalance = %v", items[1])
	}

	// An uncategorised top transaction has a null category and description.
	top := decode(t, dto.NewTxTop(reports.TxTopReport{Items: []reports.TopTransaction{
		{ID: 1, Amount: ten, Date: "2024-01-01", Account: reports.TopAccount{ID: 2, Name: "Cash"}},
	}}))
	tx := top["items"].([]any)[0].(map[string]any)
	if tx["category"] != nil || tx["description"] != nil || tx["account"].(map[string]any)["name"] != "Cash" {
		t.Errorf("top = %v", tx)
	}
}

func TestImportPreviewShape(t *testing.T) {
	id := int64(3)
	total := 40
	got := decode(t, dto.NewImportPreview(importer.PreviewResult{
		Transactions: []importer.PreviewRow{{Row: 1, Date: "2024-01-02", Type: "expense", Amount: decimal.RequireFromString("12.50"), Status: "new"}},
		Summary: importer.PreviewSummary{
			WillCreate: 1, TotalRows: &total, Sampled: 1,
			Categories: []importer.PreviewCategory{{Name: "Food", Type: "expense", Count: 1, MatchID: &id}},
		},
	}))
	row := got["previewTransactions"].([]any)[0].(map[string]any)
	if row["amount"] != 12.5 || row["error"] != nil || row["category"] != nil || row["duplicateOf"] != nil {
		t.Errorf("row = %v", row)
	}
	for _, key := range []string{"tags", "warnings"} {
		if l, ok := row[key].([]any); !ok || len(l) != 0 {
			t.Errorf("%s = %v, want []", key, row[key])
		}
	}
	sum := got["summary"].(map[string]any)
	if sum["totalRows"] != 40.0 || sum["categoriesToCreate"] == nil || sum["currenciesToCreate"] == nil {
		t.Errorf("summary = %v", sum)
	}
	if sum["categories"].([]any)[0].(map[string]any)["matchId"] != 3.0 {
		t.Errorf("categories = %v", sum["categories"])
	}
}

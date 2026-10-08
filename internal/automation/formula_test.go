package automation

import (
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

func TestFormulaIsExact(t *testing.T) {
	dec := decimal.RequireFromString
	if got := evaluateFormula("0.1 + 0.2", nil); !got.Equal(dec("0.3")) {
		t.Fatalf("0.1+0.2 = %s", got)
	}
	tx := &ledger.Transaction{Amount: money.FromDecimal(dec("0.1"), money.Unit{ID: 1, Decimals: 2})}
	if got := evaluateFormula("{{transaction.amount}} * 3", tx); !got.Equal(dec("0.3")) {
		t.Fatalf("amount*3 = %s", got)
	}
	if got := evaluateFormula("{{transaction.amount}} / 0", tx); !got.IsZero() {
		t.Fatalf("division by zero should yield 0, got %s", got)
	}
}

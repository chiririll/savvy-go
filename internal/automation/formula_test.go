package automation

import (
	"encoding/json"
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

func TestFormulaArithmetic(t *testing.T) {
	for in, want := range map[string]string{
		"1 + 2 * 3":        "7",
		"(1 + 2) * 3":      "9",
		"10 - 4 - 3":       "3",
		"-5 + 8":           "3",
		"2 * -3":           "-6",
		"100 / 8":          "12.5",
		".5 + .25":         "0.75",
		"  7  ":            "7",
		"(2 + 3":           "5", // an unclosed bracket is closed at the end
		"0.1 * 3":          "0.3",
		"19.99 * 100":      "1999",
		"1 / 0":            "0",
		"":                 "0",
		"1 +":              "0",
		"2 3":              "0",
		"abc":              "0",
		"2 ** 3":           "0",
		"1e3":              "1000", // a plain number, read as one
		"system('rm') + 1": "0",    // text is never evaluated
	} {
		if got := evaluateFormula(in, nil); !got.Equal(decimal.RequireFromString(want)) {
			t.Errorf("%q = %s, want %s", in, got, want)
		}
	}
}

func TestFormulaTakesNumbersPlaceholders(t *testing.T) {
	dec := decimal.RequireFromString
	tx := &ledger.Transaction{Amount: money.FromDecimal(dec("20"), money.Unit{ID: 1, Decimals: 2})}
	for name, tc := range map[string]struct {
		formula any
		want    string
	}{
		"a number":               {12.5, "12.5"},
		"a numeric string":       {"7", "7"},
		"a placeholder":          {"{{transaction.amount}} * 0.15", "3"},
		"entity is an alias":     {"{{entity.amount}} / 4", "5"},
		"an unknown placeholder": {"{{transaction.nope}} + 1", "1"},
		"a foreign placeholder":  {"{{user.id}} + 1", "1"},
		"not a formula":          {[]any{1}, "0"},
		"nil":                    {nil, "0"},
	} {
		if got := evaluateFormula(tc.formula, tx); !got.Equal(dec(tc.want)) {
			t.Errorf("%s: got %s, want %s", name, got, tc.want)
		}
	}
}

func TestTemplatesFillOnlyTransactionFields(t *testing.T) {
	tx := testTx()
	for in, want := range map[string]string{
		"{{transaction.description}}":                 "Coffee at Blue Bottle",
		"{{ entity.type }} of {{transaction.amount}}": "expense of 12.5",
		"{{transaction.account.name}}":                "Wallet",
		"{{transaction.date}}":                        "", // unset: empty, not "<nil>"
		"{{user.password}}":                           "{{user.password}}",
		"no placeholders":                             "no placeholders",
	} {
		if got := parseTemplate(in, tx); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestResolveValue(t *testing.T) {
	tx := testTx()
	if got := resolveValue("{{transaction.account.id}}", tx); got != int64(3) {
		t.Errorf("placeholder resolved to %v", got)
	}
	if got := resolveValue("{{ entity.type }}", tx); got != "expense" {
		t.Errorf("entity alias resolved to %v", got)
	}
	if got := resolveValue("plain", tx); got != "plain" {
		t.Errorf("plain string %v", got)
	}
	if got := resolveValue(float64(4), tx); got != float64(4) {
		t.Errorf("non-string %v", got)
	}
}

func TestCoercions(t *testing.T) {
	dec := decimal.RequireFromString
	one := dec("1.5")
	var nilDec *decimal.Decimal
	var nilMoney *money.Money
	for name, tc := range map[string]struct {
		in   any
		want string
		ok   bool
	}{
		"float":       {1.5, "1.5", true},
		"int":         {2, "2", true},
		"int64":       {int64(3), "3", true},
		"string":      {"4.25", "4.25", true},
		"decimal":     {dec("5"), "5", true},
		"decimal ptr": {&one, "1.5", true},
		"json number": {json.Number("6.1"), "6.1", true},
		"a word":      {"many", "0", false},
		"nil":         {nil, "0", false},
		"nil decimal": {nilDec, "0", false},
		"nil money":   {nilMoney, "0", false},
		"a bool":      {true, "0", false},
	} {
		got, ok := asDecimal(tc.in)
		if ok != tc.ok || !got.Equal(dec(tc.want)) {
			t.Errorf("asDecimal(%s) = %s, %v", name, got, ok)
		}
	}
	if id, ok := asInt64(float64(7)); !ok || id != 7 {
		t.Errorf("asInt64(7.0) = %d, %v", id, ok)
	}
	if id, ok := asInt64("12"); !ok || id != 12 {
		t.Errorf("asInt64(\"12\") = %d, %v", id, ok)
	}
	if _, ok := asInt64("x"); ok {
		t.Error("asInt64 of a word")
	}
	if got := asInt64Slice([]any{1.0, "2", "x", nil}); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("asInt64Slice = %v", got)
	}
	if got := asInt64Slice([]int64{4, 5}); len(got) != 2 {
		t.Errorf("asInt64Slice of ids = %v", got)
	}
	if got := asInt64Slice("1,2"); got != nil {
		t.Errorf("asInt64Slice of a string = %v", got)
	}
	s := "hi"
	var nilStr *string
	if asString(&s) != "hi" || asString(nilStr) != "" || asString(nil) != "" || asString(12) != "12" {
		t.Error("asString")
	}
	if firstString(map[string]any{"a": "", "b": "x", "c": "y"}, "a", "b", "c") != "x" || firstString(nil, "a") != "" {
		t.Error("firstString takes the first non-empty string")
	}
}

func TestEqualityNumericForNumbersTextOtherwise(t *testing.T) {
	if !valuesEqual("12.50", float64(12.5)) || !valuesEqual(int64(3), "3") {
		t.Error("numbers are compared by value")
	}
	if valuesEqual("12.5", "12.6") || valuesEqual("a", "b") {
		t.Error("different values are not equal")
	}
	if !valuesEqual(nil, nil) || !valuesEqual(nil, "") {
		t.Error("nothing equals nothing")
	}
	if inSlice("a", "a") {
		t.Error("in needs a list")
	}
}

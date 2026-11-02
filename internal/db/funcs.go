package db

import (
	"database/sql/driver"
	"strings"

	"modernc.org/sqlite"
)

// SQL functions every connection offers, for the dynamic queries in
// internal/db/filter.
const (
	// FoldFunc is savvy_fold(text): the text in lower case. SQLite's own
	// lower() and LIKE fold ASCII only, so "Кресло" would never match "кресло".
	// NULL folds to the empty string.
	FoldFunc = "savvy_fold"
	// Pow10Func is savvy_pow10(n): 10^n as an exact integer, NULL outside
	// 0..18. SQLite's pow() is a float, and money never is.
	Pow10Func = "savvy_pow10"
)

func init() {
	sqlite.MustRegisterDeterministicScalarFunction(FoldFunc, 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			switch v := args[0].(type) {
			case string:
				return strings.ToLower(v), nil
			case []byte:
				return strings.ToLower(string(v)), nil
			default:
				return "", nil
			}
		})
	sqlite.MustRegisterDeterministicScalarFunction(Pow10Func, 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			n, ok := args[0].(int64)
			if !ok || n < 0 || n > 18 {
				return nil, nil
			}
			p := int64(1)
			for range n {
				p *= 10
			}
			return p, nil
		})
}

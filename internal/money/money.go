// Package money converts between exact decimal amounts and the integer
// minor units stored in SQLite (scale = the owning currency's decimals).
package money

import (
	"database/sql"
	"encoding/json"

	"github.com/shopspring/decimal"
)

func FromMinor(v int64, decimals int) decimal.Decimal {
	return decimal.New(v, -int32(decimals))
}

// ToMinor rounds half away from zero to the currency's scale.
func ToMinor(d decimal.Decimal, decimals int) int64 {
	return d.Round(int32(decimals)).Shift(int32(decimals)).IntPart()
}

func FromNullMinor(v sql.NullInt64, decimals int) *decimal.Decimal {
	if !v.Valid {
		return nil
	}
	d := FromMinor(v.Int64, decimals)
	return &d
}

func ToNullMinor(d *decimal.Decimal, decimals int) sql.NullInt64 {
	if d == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: ToMinor(*d, decimals), Valid: true}
}

// Number renders d as a JSON number with exactly `decimals` fraction digits.
func Number(d decimal.Decimal, decimals int) json.Number {
	return json.Number(d.StringFixed(int32(decimals)))
}

// Plain renders d as a JSON number at full precision (rates, quantities).
func Plain(d decimal.Decimal) json.Number {
	return json.Number(d.String())
}

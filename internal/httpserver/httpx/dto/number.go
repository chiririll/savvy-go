package dto

import "github.com/shopspring/decimal"

// Number is a decimal that is not an amount of money (a quantity, a rate). It
// goes over the wire as a JSON number, not as the string shopspring/decimal
// writes by default, without changing how decimals marshal anywhere else.
type Number decimal.Decimal

func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(n).String()), nil
}

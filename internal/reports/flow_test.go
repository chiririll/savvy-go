package reports

import (
	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

// flowTx is an income or an expense of amount on an account: one payment (signed
// by the type) and one line in the category, none when nil.
func flowTx(typ string, account int64, category *int64, amount decimal.Decimal) ledger.TxInput {
	paid := amount
	if typ == "expense" {
		paid = amount.Neg()
	}
	return ledger.TxInput{
		Type:     typ,
		Payments: []ledger.PaymentInput{{AccountID: account, Amount: &paid}},
		Lines:    []ledger.LineInput{{CategoryID: category, Amount: amount}},
	}
}

// at dates a transaction input.
func at(in ledger.TxInput, date string) ledger.TxInput {
	in.Date = &date
	return in
}

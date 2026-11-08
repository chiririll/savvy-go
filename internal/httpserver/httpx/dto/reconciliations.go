package dto

import (
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

// Reconciliation is a point that pins an account's balance, with what the
// ledger expected. Amounts but DifferenceBase are in the account's currency;
// DifferenceBase is in the base currency, null when the currency has no rate.
type Reconciliation struct {
	ID              int64        `json:"id"`
	AccountID       int64        `json:"accountId"`
	Date            string       `json:"date"`
	ActualBalance   money.Money  `json:"actualBalance"`
	ExpectedBalance money.Money  `json:"expectedBalance"`
	Difference      money.Money  `json:"difference"`
	Note            *string      `json:"note"`
	DifferenceBase  *money.Money `json:"differenceBase"`
}

func NewReconciliation(v ledger.ReconciliationView) Reconciliation {
	return Reconciliation{
		ID: v.ID, AccountID: v.AccountID, Date: v.Date, ActualBalance: v.Actual, ExpectedBalance: v.Expected,
		Difference: v.Difference, Note: v.Note, DifferenceBase: v.DifferenceBase,
	}
}

// ReconciliationDay is the points of one date with the sum of their
// differences in the base currency. Unconverted is true when a point had no
// rate and is left out of the sum.
type ReconciliationDay struct {
	Date           string           `json:"date"`
	DifferenceBase money.Money      `json:"differenceBase"`
	Unconverted    bool             `json:"unconverted"`
	Points         []Reconciliation `json:"points"`
}

func NewReconciliationDay(d ledger.ReconciliationDay) ReconciliationDay {
	return ReconciliationDay{
		Date: d.Date, DifferenceBase: d.DifferenceBase, Unconverted: d.Unconverted,
		Points: Map(d.Points, NewReconciliation),
	}
}

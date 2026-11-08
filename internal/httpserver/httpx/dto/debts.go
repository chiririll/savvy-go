package dto

import (
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

// Debt is a debt account. CurrentBalance is the amount still owed.
type Debt struct {
	Account
	DebtType        *string     `json:"debtType"`
	TargetAmount    money.Money `json:"targetAmount"`
	PaymentProgress float64     `json:"paymentProgress"`
	DueDate         *string     `json:"dueDate"`
	Counterparty    *string     `json:"counterparty"`
	Description     *string     `json:"description"`
	IsPaidOff       bool        `json:"isPaidOff"`
}

func NewDebt(d ledger.Debt) Debt {
	acc := NewAccount(d.Account)
	acc.CurrentBalance = d.Owed() // the account balance is signed; the API shows what is owed
	direction := d.Direction
	return Debt{
		Account: acc, DebtType: &direction, TargetAmount: d.Principal, PaymentProgress: d.PaymentProgress(),
		DueDate: d.DueDate, Counterparty: d.Counterparty, Description: d.Note, IsPaidOff: d.IsPaidOff(),
	}
}

type DebtSummary struct {
	TotalIOwe     money.Money `json:"totalIOwe"`
	TotalOwedToMe money.Money `json:"totalOwedToMe"`
	NetDebt       money.Money `json:"netDebt"`
	DebtsCount    int         `json:"debtsCount"`
	Currency      *string     `json:"currency"`
	Decimals      int         `json:"decimals"`
}

func NewDebtSummary(s ledger.DebtSummary) DebtSummary {
	return DebtSummary{
		TotalIOwe: s.IOwe, TotalOwedToMe: s.OwedToMe, NetDebt: s.Net(), DebtsCount: s.Count,
		Currency: currencyCode(s.Currency), Decimals: currencyDecimals(s.Currency),
	}
}

// DebtList is debts with the summary the caller asked for.
type DebtList struct {
	Data    []Debt       `json:"data"`
	Summary *DebtSummary `json:"summary,omitempty"`
}

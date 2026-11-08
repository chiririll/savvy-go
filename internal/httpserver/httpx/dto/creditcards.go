package dto

import (
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

// CreditCard is a credit card: its account, its terms and what they say about
// today. CurrentBalance is signed, negative when money is owed.
type CreditCard struct {
	Account
	CardInfo
}

// CardInfo is what a card adds to its account: the terms and what they say
// about today. It is the creditCard of an account of type credit in /accounts,
// and flat in a CreditCard.
type CardInfo struct {
	Limit        money.Money `json:"limit"`
	Cycle        string      `json:"cycle"`
	StatementDay *int        `json:"statementDay"`
	GraceDays    int         `json:"graceDays"`
	// The rest is derived from the card's payments as of today.
	Used        money.Money   `json:"used"`
	Available   money.Money   `json:"available"`
	PeriodStart string        `json:"periodStart"`
	PeriodEnd   string        `json:"periodEnd"`
	Spent       money.Money   `json:"spent"`
	Statement   CardStatement `json:"statement"`
	Status      CardStatus    `json:"status"`
}

// CardStatement is the last closed statement.
type CardStatement struct {
	Date      string      `json:"date"`
	Balance   money.Money `json:"balance"`
	DueDate   string      `json:"dueDate"`
	Paid      money.Money `json:"paid"`
	AmountDue money.Money `json:"amountDue"`
}

// CardStatus is the triangle: level ok, warning or danger, why, and the amount
// and date it is about (no date for over_limit).
type CardStatus struct {
	Level  string      `json:"level"`
	Reason string      `json:"reason,omitempty"`
	Amount money.Money `json:"amount"`
	Date   string      `json:"date,omitempty"`
}

func NewCreditCard(c ledger.Card, st ledger.CardState) CreditCard {
	return CreditCard{Account: NewAccount(c.Account), CardInfo: NewCardInfo(c, st)}
}

func NewCardInfo(c ledger.Card, st ledger.CardState) CardInfo {
	return CardInfo{
		Limit: c.Terms.Limit, Cycle: c.Terms.Cycle, StatementDay: c.Terms.StatementDay, GraceDays: c.Terms.GraceDays,
		Used: st.Used, Available: st.Available, PeriodStart: st.PeriodStart, PeriodEnd: st.PeriodEnd, Spent: st.Spent,
		Statement: CardStatement{
			Date: st.Statement.Date, Balance: st.Statement.Balance, DueDate: st.Statement.DueDate,
			Paid: st.Statement.Paid, AmountDue: st.Statement.AmountDue,
		},
		Status: CardStatus{Level: st.Status.Level, Reason: st.Status.Reason, Amount: st.Status.Amount, Date: st.Status.Date},
	}
}

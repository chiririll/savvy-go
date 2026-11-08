package dto

import (
	"time"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

type TxActions struct {
	Edit        bool `json:"edit"`
	EditDetails bool `json:"editDetails"`
	Delete      bool `json:"delete"`
	Duplicate   bool `json:"duplicate"`
	Confirm     bool `json:"confirm"`
	Skip        bool `json:"skip"`
}

// Payment is money that moved on an account, signed, in the account's currency.
type Payment struct {
	AccountID int64       `json:"accountId"`
	Account   *Account    `json:"account"`
	Amount    money.Money `json:"amount"`
}

// Line is an amount for a category in its own currency.
type Line struct {
	Seq        int         `json:"seq"`
	CategoryID *int64      `json:"categoryId"`
	Category   *Category   `json:"category"`
	CurrencyID int64       `json:"currencyId"`
	Amount     money.Money `json:"amount"`
	Note       *string     `json:"note"`
}

// Restriction says why a transaction cannot be changed freely.
type Restriction struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// CurrencyAmount is an amount with the currency it is in.
type CurrencyAmount struct {
	Currency Currency    `json:"currency"`
	Amount   money.Money `json:"amount"`
}

func newCurrencyAmount(c ledger.CurrencyAmount) CurrencyAmount {
	return CurrencyAmount{Currency: NewCurrency(c.Currency), Amount: c.Amount}
}

// RowAmount is the one figure a list row shows, ready to draw. Main is in the
// first currency of Totals; when Converted, it adds the other totals at the
// current rates (an estimate) and Totals hold the exact amounts.
type RowAmount struct {
	Main      CurrencyAmount   `json:"main"`
	Totals    []CurrencyAmount `json:"totals"`
	Converted bool             `json:"converted"`
}

func newRowAmount(r *ledger.RowAmount) *RowAmount {
	if r == nil {
		return nil
	}
	return &RowAmount{Main: newCurrencyAmount(r.Main), Totals: Map(r.Totals, newCurrencyAmount), Converted: r.Converted}
}

// DayTotal is what one day netted, per currency, largest first.
type DayTotal struct {
	Date   string           `json:"date"`
	Totals []CurrencyAmount `json:"totals"`
}

func NewDayTotals(days []ledger.DayTotal) []DayTotal {
	return Map(days, func(d ledger.DayTotal) DayTotal {
		return DayTotal{Date: d.Date, Totals: Map(d.Totals, newCurrencyAmount)}
	})
}

// Transaction is a transaction with its payments and lines. Amount is the one
// figure for a list (nil when the payments are in several currencies);
// RowAmount is what a row shows instead, converted for several currencies,
// and ReceivedAmount the amount a transfer arrived with;
// Difference is set only on the answer to a save, when the payments and the
// lines disagree, in the base currency.
type Transaction struct {
	ID                     int64         `json:"id"`
	Type                   string        `json:"type"`
	Description            *string       `json:"description"`
	Date                   *string       `json:"date"`
	Status                 string        `json:"status"`
	RecurringTransactionID *int64        `json:"recurringTransactionId"`
	Amount                 *money.Money  `json:"amount"`
	RowAmount              *RowAmount    `json:"rowAmount"`
	ReceivedAmount         *RowAmount    `json:"receivedAmount"`
	Difference             *money.Money  `json:"difference"`
	Actions                TxActions     `json:"actions"`
	Restrictions           []Restriction `json:"restrictions"`
	Payments               []Payment     `json:"payments"`
	Lines                  []Line        `json:"lines"`
	Tags                   []Tag         `json:"tags"`
	CreatedAt              *time.Time    `json:"createdAt"`
}

func newActions(a ledger.TxActions) TxActions {
	return TxActions{
		Edit: a.Edit, EditDetails: a.EditDetails, Delete: a.Delete, Duplicate: a.Duplicate,
		Confirm: a.Confirm, Skip: a.Skip,
	}
}

func NewPayment(p ledger.Payment) Payment {
	return Payment{AccountID: p.AccountID, Account: ptr(p.Account, NewAccount), Amount: p.Amount}
}

func NewLine(l ledger.Line) Line {
	return Line{
		Seq: l.Seq, CategoryID: l.CategoryID, Category: ptr(l.Category, NewCategory),
		CurrencyID: l.Amount.Unit().ID, Amount: l.Amount, Note: l.Note,
	}
}

func NewTransaction(t ledger.Transaction) Transaction {
	out := Transaction{
		ID: t.ID, Type: t.Type, Description: t.Description, Date: t.Date, Status: t.Status,
		RecurringTransactionID: t.RecurringID, Difference: t.Difference,
		Actions:      newActions(t.Allowed()),
		Restrictions: []Restriction{},
		Payments:     Map(t.Payments, NewPayment),
		Lines:        Map(t.Lines, NewLine),
		Tags:         Map(t.Tags, NewTag),
		CreatedAt:    utc(t.CreatedAt),
	}
	main, received := t.ListAmounts()
	out.RowAmount, out.ReceivedAmount = newRowAmount(main), newRowAmount(received)
	if m, ok := t.Amount(); ok {
		out.Amount = &m
	}
	for _, r := range t.Reasons {
		out.Restrictions = append(out.Restrictions, Restriction{Reason: string(r), Message: r.Message()})
	}
	return out
}

type TransactionSummary struct {
	Income            money.Money `json:"income"`
	Expense           money.Money `json:"expense"`
	Balance           money.Money `json:"balance"`
	TransactionsCount int         `json:"transactionsCount"`
	Currency          *string     `json:"currency"`
}

func NewTransactionSummary(s ledger.TransactionSummary) TransactionSummary {
	return TransactionSummary{
		Income: s.Income, Expense: s.Expense, Balance: s.Balance(),
		TransactionsCount: s.Count, Currency: currencyCode(s.Currency),
	}
}

// PageMeta describes one page of a paginated list.
type PageMeta struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	// HasMore is set instead of an exact Total when the request asked not to
	// count (count=0): Total is then only what this page holds.
	HasMore *bool `json:"has_more,omitempty"`
}

// TransactionPage is a page of transactions, with the summary when asked for.
type TransactionPage struct {
	Data    []Transaction       `json:"data"`
	Meta    PageMeta            `json:"meta"`
	Summary *TransactionSummary `json:"summary,omitempty"`
}

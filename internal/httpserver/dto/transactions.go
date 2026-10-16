package dto

import (
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

type TxItem struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Quantity     decimal.Decimal `json:"quantity"`
	PricePerUnit money.Money     `json:"pricePerUnit"`
	TotalPrice   money.Money     `json:"totalPrice"`
}

func NewTxItem(i ledger.TxItem) TxItem {
	return TxItem{ID: i.ID, Name: i.Name, Quantity: i.Quantity, PricePerUnit: i.PricePerUnit, TotalPrice: i.TotalPrice}
}

type TxActions struct {
	Edit      bool `json:"edit"`
	Delete    bool `json:"delete"`
	Duplicate bool `json:"duplicate"`
	Confirm   bool `json:"confirm"`
	Skip      bool `json:"skip"`
}

type Transaction struct {
	ID                     int64        `json:"id"`
	Type                   string       `json:"type"`
	Amount                 money.Money  `json:"amount"`
	ToAmount               *money.Money `json:"toAmount"`
	IsEstimated            bool         `json:"isEstimated"`
	Description            *string      `json:"description"`
	Date                   *string      `json:"date"`
	Status                 string       `json:"status"`
	RecurringTransactionID *int64       `json:"recurringTransactionId"`
	Actions                TxActions    `json:"actions"`
	Account                *Account     `json:"account"`
	ToAccount              *Account     `json:"toAccount"`
	Category               *Category    `json:"category"`
	Items                  []TxItem     `json:"items"`
	Tags                   []Tag        `json:"tags"`
	CreatedAt              *time.Time   `json:"createdAt"`
}

func NewTransaction(t ledger.Transaction) Transaction {
	return Transaction{
		ID: t.ID, Type: t.Type, Amount: t.Amount, ToAmount: t.ToAmount, IsEstimated: t.IsEstimated,
		Description: t.Description, Date: t.Date, Status: t.Status, RecurringTransactionID: t.RecurringID,
		Actions:   transactionActions(t),
		Account:   ptr(t.Account, NewAccount),
		ToAccount: ptr(t.ToAccount, NewAccount),
		Category:  ptr(t.Category, NewCategory),
		Items:     Map(t.Items, NewTxItem),
		Tags:      Map(t.Tags, NewTag),
		CreatedAt: utc(t.CreatedAt),
	}
}

func transactionActions(t ledger.Transaction) TxActions {
	// One side of a transfer between spaces changes only through the
	// transfer (in the space settings), never as a transaction.
	if t.Type == "transfer_out" || t.Type == "transfer_in" {
		return TxActions{}
	}
	pending := t.Status == "pending"
	skipped := t.Status == "skipped"
	recurring := t.RecurringID != nil
	return TxActions{
		Edit:      !skipped && !recurring,
		Delete:    !recurring,
		Duplicate: !recurring && !skipped,
		Confirm:   pending,
		Skip:      pending && recurring,
	}
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
		Income: s.Income, Expense: s.Expense, Balance: s.Income.Sub(s.Expense),
		TransactionsCount: s.Count, Currency: currencyCode(s.Currency),
	}
}

package dto

import (
	"encoding/json"

	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
)

// Recurring is a rule. Template is what it creates, in the shape the API takes
// to create a transaction (snake_case keys), plus a payment without an amount
// that each occurrence fills in; Schedule is when.
type Recurring struct {
	ID          int64              `json:"id"`
	Template    json.RawMessage    `json:"template"`
	Schedule    recurring.Schedule `json:"schedule"`
	IsActive    bool               `json:"isActive"`
	Error       *string            `json:"error"`
	NextRunDate *string            `json:"nextRunDate"`
	LastRunDate *string            `json:"lastRunDate"`
	// RowAmount and ReceivedAmount are what the rule's transaction will show in
	// a list, at today's rates (see Transaction). Null when the template cannot
	// be read.
	RowAmount      *RowAmount `json:"rowAmount"`
	ReceivedAmount *RowAmount `json:"receivedAmount"`
}

// NewRecurring maps a rule; preview is the transaction its template makes, if it can be read.
func NewRecurring(r recurring.Rule, preview *ledger.Transaction) Recurring {
	out := Recurring{
		ID: r.ID, Template: r.Template, Schedule: r.Schedule, IsActive: r.IsActive, Error: r.Error,
		NextRunDate: r.NextRunDate, LastRunDate: r.LastRunDate,
	}
	if preview != nil {
		main, received := preview.ListAmounts()
		out.RowAmount, out.ReceivedAmount = newRowAmount(main), newRowAmount(received)
	}
	return out
}

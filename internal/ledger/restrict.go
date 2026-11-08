package ledger

import (
	"errors"
	"slices"
)

// Reason is a fact about a transaction that restricts what may be done with
// it. Reasons are never stored: each follows from data already in the database
// (see Transactions.hydrate), so none can get out of step with it.
type Reason string

const (
	// ReasonDebt: a payment on a debt account. The debts module owns it.
	ReasonDebt Reason = "debt"
	// ReasonSpaceTransfer: a payment on a space account carries a transfer
	// between spaces. internal/transfers owns that payment and the date.
	ReasonSpaceTransfer Reason = "space_transfer"
	// ReasonTransferReview: that transfer is in review, so nothing changes
	// until the review is resolved.
	ReasonTransferReview Reason = "transfer_review"
	// ReasonRecurringPending: the open occurrence of a recurring rule, which
	// the rule replans.
	ReasonRecurringPending Reason = "recurring_pending"
)

// TxActions is what may be done to a transaction. The Transactions service
// refuses what is not allowed, and the API reports the same flags to clients.
// EditDetails is the part of Edit that leaves the money alone: description,
// tags, and the categories and notes of lines.
type TxActions struct {
	Edit, Delete, Duplicate, Confirm, Skip, EditDetails bool
}

func (a TxActions) and(b TxActions) TxActions {
	return TxActions{
		Edit: a.Edit && b.Edit, Delete: a.Delete && b.Delete, Duplicate: a.Duplicate && b.Duplicate,
		Confirm: a.Confirm && b.Confirm, Skip: a.Skip && b.Skip, EditDetails: a.EditDetails && b.EditDetails,
	}
}

// has reports whether the action named by one of the Action* constants is on.
func (a TxActions) has(action string) bool {
	switch action {
	case ActionEdit:
		return a.Edit
	case ActionEditDetails:
		return a.EditDetails
	case ActionDelete:
		return a.Delete
	case ActionDuplicate:
		return a.Duplicate
	case ActionConfirm:
		return a.Confirm
	case ActionSkip:
		return a.Skip
	}
	return false
}

// The actions refuse and the service name.
const (
	ActionEdit        = "edit"
	ActionEditDetails = "edit_details"
	ActionDelete      = "delete"
	ActionDuplicate   = "duplicate"
	ActionConfirm     = "confirm"
	ActionSkip        = "skip"
	// ActionSpacePayment is a change to the payment on a space account, to what
	// it pairs with, or to the date: only the transfers module makes it.
	ActionSpacePayment = "edit_space_payment"
)

// reasonRule is what one reason allows and the message shown to the user when
// it refuses. A reason missing from reasonRules allows nothing.
type reasonRule struct {
	Allows  TxActions
	Message string
}

// reasonRules is the only place that says what a reason allows. Changing it is
// a code change, never a migration.
var reasonRules = map[Reason]reasonRule{
	ReasonDebt: {
		Allows:  TxActions{EditDetails: true},
		Message: "this transaction belongs to a debt; change the debt instead",
	},
	ReasonSpaceTransfer: {
		Allows: TxActions{Edit: true, EditDetails: true},
		Message: "this transaction is one side of a transfer between spaces; " +
			"its space payment and date change through the transfer",
	},
	ReasonTransferReview: {
		Message: "this transfer between spaces is in review; resolve the review first",
	},
	ReasonRecurringPending: {
		Allows:  TxActions{Confirm: true, Skip: true},
		Message: "this is the next occurrence of a recurring transaction; confirm or skip it",
	},
}

// Message is what to tell the user about a reason: why it restricts a
// transaction. Empty for a reason nobody mapped.
func (r Reason) Message() string { return reasonRules[r].Message }

// statusActions is what the status alone allows.
func statusActions(status string) TxActions {
	switch status {
	case "skipped":
		return TxActions{Delete: true}
	case "pending":
		return TxActions{Edit: true, Delete: true, Duplicate: true, Confirm: true, EditDetails: true}
	default:
		return TxActions{Edit: true, Delete: true, Duplicate: true, EditDetails: true}
	}
}

// Allowed is the single source of truth for TxActions: what the status allows,
// narrowed by every reason that applies. Only an open occurrence can be
// skipped; a transaction entered by hand is deleted instead.
func (t Transaction) Allowed() TxActions {
	a := statusActions(t.Status)
	for _, r := range t.Reasons {
		a = a.and(reasonRules[r].Allows)
	}
	a.Skip = t.Status == "pending" && slices.Contains(t.Reasons, ReasonRecurringPending)
	return a
}

// ErrRestricted is the error of every refusal because of a reason.
var ErrRestricted = errors.New("this transaction cannot be changed that way")

// RestrictedError says which reason refused an action.
type RestrictedError struct {
	Reason  Reason
	Action  string
	Message string
}

func (e *RestrictedError) Error() string   { return e.Message }
func (e *RestrictedError) Is(t error) bool { return t == ErrRestricted }

// refuse returns nil when action is allowed, otherwise the error naming the
// first reason that denies it, or a plain refusal when only the status does.
func (t Transaction) refuse(action string) error {
	if t.Allowed().has(action) {
		return nil
	}
	for _, r := range t.Reasons {
		if !reasonRules[r].Allows.has(action) {
			return &RestrictedError{Reason: r, Action: action, Message: reasonRules[r].Message}
		}
	}
	return &RestrictedError{Action: action, Message: "a " + t.Status + " transaction cannot be changed that way"}
}

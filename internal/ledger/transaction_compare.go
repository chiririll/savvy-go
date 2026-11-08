package ledger

import (
	"errors"
	"slices"

	repo "savvy-go/internal/db/repo/ledger"
)

// Comparing a stored transaction with a changed one.

// ErrDetailsOnly is returned when a change to a transaction that allows only
// its details touches the money, the date or the type.
var ErrDetailsOnly = errors.New("only the description, tags and the categories and notes of lines can change")

// frozenParts keeps what only the owner of a transfer between spaces changes:
// the payment on the space account, the date and the type, and for a transfer
// the real payment it pairs with.
func frozenParts(cur Transaction, in TxInput, b built) error {
	if !slices.Contains(cur.Reasons, ReasonSpaceTransfer) {
		return nil
	}
	if in.Type != cur.Type || !sameDate(cur.Date, in.Date) {
		return cur.refuse(ActionSpacePayment)
	}
	paired := cur.Type == "transfer_out" || cur.Type == "transfer_in"
	for _, p := range cur.Payments {
		if p.SpaceTransferUUID == nil && !paired {
			continue
		}
		if !hasPayment(b.payments, p) {
			return cur.refuse(ActionSpacePayment)
		}
	}
	return nil
}

func hasPayment(have []repo.PaymentFields, p Payment) bool {
	return slices.ContainsFunc(have, func(h repo.PaymentFields) bool {
		return h.AccountID == p.AccountID && h.Amount.Equal(p.Amount)
	})
}

func samePayments(cur []Payment, next []repo.PaymentFields) bool {
	if len(cur) != len(next) {
		return false
	}
	for _, p := range cur {
		if !hasPayment(next, p) {
			return false
		}
	}
	return true
}

// sameLines compares lines in order: by amount and currency, and, unless
// moneyOnly, by category and note too.
func sameLines(cur []Line, next []repo.LineFields, moneyOnly bool) bool {
	if len(cur) != len(next) {
		return false
	}
	for i, l := range cur {
		n := next[i]
		if !l.Amount.Equal(n.Amount) {
			return false
		}
		if !moneyOnly && (!sameInt(l.CategoryID, n.CategoryID) || !sameStr(l.Note, n.Note)) {
			return false
		}
	}
	return true
}

func sameInt(a, b *int64) bool   { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
func sameStr(a, b *string) bool  { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
func sameDate(a, b *string) bool { return sameStr(a, b) }

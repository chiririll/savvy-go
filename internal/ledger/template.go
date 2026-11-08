package ledger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

// Templates makes and checks the transactions of recurring rules. A template
// is the JSON of TxInput, so a rule can create whatever can be entered by
// hand; it adds only a payment without an amount, which each occurrence fills
// in at that day's rates. It implements recurring.Occurrences.
type Templates struct{ DB store.DB }

func (s Templates) txs() Transactions { return Transactions{DB: s.DB}.As(ReasonRecurringPending) }

// ErrTemplate is returned for a template that cannot be read.
var ErrTemplate = errors.New("invalid template")

// decode reads a template strictly: a key the ledger does not know is an
// error, so a typo is caught when the rule is saved, not at the occurrence.
func decode(template json.RawMessage) (TxInput, error) {
	var in TxInput
	dec := json.NewDecoder(bytes.NewReader(template))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return TxInput{}, fmt.Errorf("%w: %v", ErrTemplate, err)
	}
	return in, nil
}

// Validate checks a template as the ledger would accept it as a new
// transaction, with the amount of one payment left out.
func (s Templates) Validate(ctx context.Context, template json.RawMessage) error {
	in, err := decode(template)
	if err != nil {
		return err
	}
	_, err = s.txs().build(ctx, in, true)
	return err
}

// fill turns a template into a transaction input dated due: the payment
// without an amount gets what balances the others.
func (s Templates) fill(ctx context.Context, template json.RawMessage, due string) (TxInput, error) {
	in, err := decode(template)
	if err != nil {
		return TxInput{}, err
	}
	b, err := s.txs().build(ctx, in, true)
	if err != nil {
		return TxInput{}, err
	}
	if at, amount, ok := b.shape.balancing(); ok {
		unit := b.shape.Payments[at].Account.Currency.Unit()
		rounded, err := money.FromInput(amount, unit)
		if err != nil {
			return TxInput{}, err
		}
		d := rounded.Decimal()
		for i := range in.Payments {
			if in.Payments[i].Amount == nil {
				in.Payments[i].Amount = &d
			}
			in.Payments[i].Estimated = false
		}
	}
	in.Date, in.Status = &due, nil
	if in.TagIDs == nil {
		in.TagIDs = []int64{}
	}
	return in, nil
}

// Create makes the pending transaction of a template dated due and returns
// its id.
func (s Templates) Create(ctx context.Context, template json.RawMessage, due string) (int64, error) {
	in, err := s.fill(ctx, template, due)
	if err != nil {
		return 0, err
	}
	pending := "pending"
	in.Status = &pending
	tx, err := s.txs().Create(ctx, in)
	if err != nil {
		return 0, err
	}
	return tx.ID, nil
}

// Replan rewrites a pending occurrence from a changed template. The rule owns
// the occurrence's content, so the edit rules of a hand-entered one do not
// apply (see ReasonRecurringPending).
func (s Templates) Replan(ctx context.Context, txID int64, template json.RawMessage, due string) error {
	in, err := s.fill(ctx, template, due)
	if err != nil {
		return err
	}
	_, err = s.txs().Update(ctx, txID, in)
	return err
}

// Preview is the transaction a template makes, worked out and not saved: the
// payment without an amount takes what balances the others at today's rates.
// It is for showing a rule; the occurrence itself is made by Create.
func (s Templates) Preview(ctx context.Context, template json.RawMessage) (Transaction, error) {
	in, err := decode(template)
	if err != nil {
		return Transaction{}, err
	}
	b, err := s.txs().build(ctx, in, true)
	if err != nil {
		return Transaction{}, err
	}
	sh := b.shape
	if at, amount, ok := sh.balancing(); ok {
		rounded, err := money.FromInput(amount, sh.Payments[at].Account.Currency.Unit())
		if err != nil {
			return Transaction{}, err
		}
		sh.Payments[at].Amount = rounded
	}
	tx := Transaction{Type: sh.Type, Status: "pending"}
	for _, p := range sh.Payments {
		account := p.Account
		tx.Payments = append(tx.Payments, Payment{AccountID: account.ID, Account: &account, Amount: p.Amount})
	}
	for _, l := range sh.Lines {
		tx.Lines = append(tx.Lines, Line{Amount: l.Amount})
	}
	return tx, nil
}

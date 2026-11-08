package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

// Resolving an input into the rows to store.

// built is a transaction input resolved against the database: the rows to
// store and the shape the rules check.
type built struct {
	shape    shape
	payments []repo.PaymentFields
	lines    []repo.LineFields
}

// build resolves an input: loads its accounts, categories and currencies,
// rounds amounts into their currencies, and checks the shape.
func (s Transactions) build(ctx context.Context, in TxInput, template bool) (built, error) {
	b := built{shape: shape{Type: in.Type, Owner: s.as, Template: template}}
	accts := Accounts{DB: s.DB}
	currencies := map[int64]Currency{}
	for _, p := range in.Payments {
		a, err := accts.Bare(ctx, p.AccountID)
		if err != nil {
			return built{}, err
		}
		if a == nil || a.Currency == nil {
			return built{}, ErrUnknownAccount
		}
		currencies[a.Currency.ID] = *a.Currency
		sp := shapePayment{Account: *a}
		if p.SpaceTransferUUID != nil && (s.as != ReasonSpaceTransfer || a.Type != "space") {
			return built{}, fmt.Errorf("%w: only a transfer between spaces sets this", ErrSpaceShape)
		}
		if p.Amount == nil {
			sp.Balancing = true
		} else {
			if sp.Amount, err = money.FromInput(*p.Amount, a.Currency.Unit()); err != nil {
				return built{}, err
			}
			b.payments = append(b.payments, repo.PaymentFields{
				AccountID: a.ID, Amount: sp.Amount, SpaceTransferUUID: p.SpaceTransferUUID,
			})
		}
		b.shape.Payments = append(b.shape.Payments, sp)
	}
	curs, cats := Currencies{DB: s.DB}, Categories{DB: s.DB}
	for _, l := range in.Lines {
		cur, ok := currencies[l.CurrencyID]
		if l.CurrencyID == 0 && len(currencies) == 1 {
			for _, only := range currencies {
				cur, ok = only, true
			}
		}
		if !ok { // a currency no payment account has: let the rules say so
			c, err := curs.ByID(ctx, l.CurrencyID)
			if err != nil {
				return built{}, err
			}
			if c == nil {
				return built{}, ErrUnknownCurrency
			}
			cur = *c
		}
		amount, err := money.FromInput(l.Amount, cur.Unit())
		if err != nil {
			return built{}, err
		}
		sl := shapeLine{Currency: cur, Amount: amount}
		if l.CategoryID != nil {
			if sl.Category, err = cats.Bare(ctx, *l.CategoryID); err != nil {
				return built{}, err
			}
			if sl.Category == nil {
				return built{}, ErrUnknownCategory
			}
		}
		b.shape.Lines = append(b.shape.Lines, sl)
		b.lines = append(b.lines, repo.LineFields{CategoryID: l.CategoryID, Amount: amount, Note: l.Note})
	}
	return b, b.shape.check()
}

// ConvertedTransfer is the input of a transfer of amount, in the currency of
// from, that delivers the same value on to at the current rates.
func (s Transactions) ConvertedTransfer(ctx context.Context, from, to int64, amount decimal.Decimal) (TxInput, error) {
	accts := Accounts{DB: s.DB}
	src, err := accts.ByID(ctx, from)
	if err != nil || src == nil || src.Currency == nil {
		return TxInput{}, ErrUnknownAccount
	}
	dst, err := accts.ByID(ctx, to)
	if err != nil || dst == nil || dst.Currency == nil {
		return TxInput{}, ErrUnknownAccount
	}
	out, err := money.FromInput(amount, src.Currency.Unit())
	if err != nil {
		return TxInput{}, err
	}
	in, err := TryConvert(out, *src.Currency, *dst.Currency)
	if err != nil {
		return TxInput{}, err
	}
	taken, delivered := out.Neg().Decimal(), in.Decimal()
	return TxInput{
		Type:     "transfer",
		Payments: []PaymentInput{{AccountID: from, Amount: &taken}, {AccountID: to, Amount: &delivered}},
	}, nil
}

// ErrUnknownCategory is returned when a category referenced by id does not exist.
var ErrUnknownCategory = errors.New("unknown category")

// difference is the soft check on a built transaction: nil when payments and
// lines agree, or when there is no base currency to compare in.
func (s Transactions) difference(ctx context.Context, b built) *money.Money {
	base, err := (Currencies{DB: s.DB}).Base(ctx)
	if err != nil || base == nil || len(b.shape.Payments) == 0 {
		return nil
	}
	d, ok := b.shape.difference(*base)
	if !ok || d.IsZero() {
		return nil
	}
	return &d
}

// store writes the payments, lines and tags of a transaction.
func (s Transactions) store(ctx context.Context, d store.DB, id int64, b built, tagIDs []int64) error {
	r := repo.Repo{DB: d}
	if err := r.ReplacePayments(ctx, id, b.payments); err != nil {
		return err
	}
	if err := r.ReplaceLines(ctx, id, b.lines); err != nil {
		return err
	}
	if tagIDs != nil {
		return r.SetTags(ctx, id, tagIDs)
	}
	return nil
}

// blankToNil stores an empty description as none.
func blankToNil(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

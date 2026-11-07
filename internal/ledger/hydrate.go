package ledger

import (
	"context"
	"slices"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/filter"
	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

// TransactionSummary totals transactions in the base currency (nil when there
// is none), rounded to its decimals.
type TransactionSummary struct {
	Income   money.Money
	Expense  money.Money
	Count    int
	Currency *Currency
}

// Balance is income less expense.
func (s TransactionSummary) Balance() money.Money { return s.Income.Sub(s.Expense) }

// Summary totals the confirmed (or pending) income and expense matching f, in
// the base currency.
func (s Transactions) Summary(ctx context.Context, f filter.TxFilter, pendingOnly bool) TransactionSummary {
	status := "confirmed"
	if pendingOnly {
		status = "pending"
	}
	base, _ := (Currencies{DB: s.DB}).Base(ctx)
	out := TransactionSummary{Currency: base}
	var unit money.Unit // no base currency means no transactions either
	if base != nil {
		unit = base.Unit()
	}
	out.Income, out.Expense = money.Zero(unit), money.Zero(unit)
	rows, err := s.repo().SummaryRows(ctx, f, status)
	if err != nil {
		return out
	}
	// Sum in base major units and round once: rounding each row would drift.
	income, expense := decimal.Zero, decimal.Zero
	for _, r := range rows {
		amount := decimal.New(r.Minor, -int32(r.Decimals))
		if !r.IsBase && !r.Rate.IsZero() {
			amount = amount.Mul(r.Rate)
		}
		if r.Type == "income" {
			income = income.Add(amount)
		} else {
			expense = expense.Add(amount)
		}
		out.Count++
	}
	out.Income, out.Expense = money.FromDecimal(income, unit), money.FromDecimal(expense, unit)
	return out
}

// hydrate turns rows into transactions with their accounts and categories (one
// lookup per distinct one) and the reasons that restrict them. The payments,
// lines and tags came with the rows, in a few queries for the whole page.
func (s Transactions) hydrate(ctx context.Context, rows []repo.Tx) ([]Transaction, error) {
	return s.hydrateWith(ctx, rows, NewRelated(ctx, s.DB))
}

func (s Transactions) hydrateWith(ctx context.Context, rows []repo.Tx, rel *Related) ([]Transaction, error) {
	out := make([]Transaction, 0, len(rows))
	for _, r := range rows {
		t := Transaction{
			ID: r.ID, Type: r.Type, Description: r.Description, Date: r.Date, Status: r.Status,
			ExternalID: r.ExternalID, CreatedAt: r.CreatedAt, RecurringID: r.RecurringID,
			Payments: make([]Payment, len(r.Payments)), Lines: make([]Line, len(r.Lines)),
			Tags: make([]Tag, len(r.Tags)), Reasons: reasonsOf(r),
		}
		for i, p := range r.Payments {
			t.Payments[i] = Payment{
				AccountID: p.AccountID, Account: rel.Account(p.AccountID), Amount: p.Amount,
				SpaceTransferUUID: p.SpaceTransferUUID,
			}
		}
		for i, l := range r.Lines {
			t.Lines[i] = Line{
				Seq: l.Seq, CategoryID: l.CategoryID, Category: rel.Category(l.CategoryID),
				Amount: l.Amount, Note: l.Note,
			}
		}
		for i, g := range r.Tags {
			t.Tags[i] = tagFrom(g)
		}
		out = append(out, t)
	}
	return out, nil
}

// reasonsOf derives, from facts already stored, why a transaction is
// restricted. Nothing here is saved, so nothing can drift from the data.
func reasonsOf(r repo.Tx) []Reason {
	var out []Reason
	for _, p := range r.Payments {
		if p.AccountType == "debt" {
			out = append(out, ReasonDebt)
			break
		}
	}
	if slices.ContainsFunc(r.Payments, func(p repo.Payment) bool { return p.SpaceTransferUUID != nil }) {
		out = append(out, ReasonSpaceTransfer)
		if r.TransferStatus == "in_review" {
			out = append(out, ReasonTransferReview)
		}
	}
	if r.Status == "pending" && r.RecurringID != nil {
		out = append(out, ReasonRecurringPending)
	}
	return out
}

package ledger

import (
	"context"

	"github.com/shopspring/decimal"

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

func (s Transactions) Summary(ctx context.Context, pendingOnly bool) TransactionSummary {
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
	rows, err := s.repo().SummaryRows(ctx, status)
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

// hydrate turns rows into transactions with their accounts, category, items
// and tags: one lookup per distinct account and category, and one query each
// for the items and tags of the whole page.
func (s Transactions) hydrate(ctx context.Context, rows []repo.Tx) ([]Transaction, error) {
	out := make([]Transaction, 0, len(rows))
	ids := make([]int64, 0, len(rows))
	at := make(map[int64]int, len(rows))
	rel := NewRelated(ctx, s.DB)
	for _, r := range rows {
		t := txFrom(r)
		t.Account = rel.Account(t.AccountID)
		t.ToAccount = rel.OptionalAccount(t.ToAccountID)
		t.Category = rel.Category(t.CategoryID)
		t.Items, t.Tags = []TxItem{}, []Tag{}
		at[t.ID] = len(out)
		ids = append(ids, t.ID)
		out = append(out, t)
	}
	if len(ids) == 0 {
		return out, nil
	}
	items, err := s.repo().ItemsOf(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, r := range items {
		t := &out[at[r.TransactionID]]
		unit := t.Amount.Unit() // items are priced in the account currency
		t.Items = append(t.Items, TxItem{
			ID: r.ID, Name: r.Name, Quantity: r.Quantity,
			PricePerUnit: money.New(r.PricePerUnit, unit), TotalPrice: money.New(r.TotalPrice, unit),
		})
	}
	tags, err := s.repo().TagsOf(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, r := range tags {
		t := &out[at[r.TransactionID]]
		t.Tags = append(t.Tags, tagFrom(r.Tag))
	}
	return out, nil
}

// itemRow is a transaction line ready to store.
type itemRow = repo.NewTxItem

// itemRows checks the lines entered for a transaction against the range and
// rounds them into unit, so nothing is written for a transaction with a bad
// line. A line without a total costs quantity times price.
func itemRows(items []TxItemInput, unit money.Unit) ([]itemRow, error) {
	rows := make([]itemRow, 0, len(items))
	for _, it := range items {
		if it.TotalPrice.IsZero() {
			it.TotalPrice = it.Quantity.Mul(it.PricePerUnit)
		}
		price, err := money.FromInput(it.PricePerUnit, unit)
		if err != nil {
			return nil, err
		}
		line, err := money.FromInput(it.TotalPrice, unit)
		if err != nil {
			return nil, err
		}
		rows = append(rows, itemRow{Name: it.Name, Quantity: it.Quantity, PricePerUnit: price, TotalPrice: line})
	}
	return rows, nil
}

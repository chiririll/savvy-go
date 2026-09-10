package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/filter"
	"savvy-go/internal/db/sqlc"
)

type TxItem struct {
	ID           int64
	Name         string
	Quantity     float64
	PricePerUnit float64
	TotalPrice   float64
}

func (i TxItem) JSON() map[string]any {
	return map[string]any{
		"id": i.ID, "name": i.Name, "quantity": i.Quantity,
		"pricePerUnit": i.PricePerUnit, "totalPrice": i.TotalPrice,
	}
}

type Transaction struct {
	ID           int64
	Type         string
	AccountID    int64
	ToAccountID  *int64
	CategoryID   *int64
	Amount       float64
	ToAmount     *float64
	ExchangeRate *float64
	Description  *string
	Date         *string
	Status       string
	RecurringID  *int64
	CreatedAt    *time.Time
	Account      *Account
	ToAccount    *Account
	Category     *Category
	Items        []TxItem
	Tags         []Tag
}

func (t Transaction) JSON() map[string]any {
	m := map[string]any{
		"id": t.ID, "type": t.Type, "amount": t.Amount,
		"description": t.Description, "date": t.Date, "status": t.Status,
		"recurringTransactionId": t.RecurringID,
		"actions":                t.actions(),
		"items":                  mapSliceVal(t.Items, TxItem.JSON),
		"itemsCount":             len(t.Items),
		"tags":                   mapSliceVal(t.Tags, Tag.JSON),
	}
	if t.ToAmount != nil {
		m["toAmount"] = *t.ToAmount
	}
	if t.ExchangeRate != nil {
		m["exchangeRate"] = *t.ExchangeRate
	}
	if t.Account != nil {
		m["account"] = t.Account.JSON()
	}
	if t.ToAccount != nil {
		m["toAccount"] = t.ToAccount.JSON()
	}
	if t.Category != nil {
		m["category"] = t.Category.JSON()
	}
	if t.CreatedAt != nil {
		m["createdAt"] = t.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func (t Transaction) actions() map[string]bool {
	pending := t.Status == "pending"
	skipped := t.Status == "skipped"
	recurring := t.RecurringID != nil
	return map[string]bool{
		"edit":      !skipped && !recurring,
		"delete":    !recurring,
		"duplicate": !recurring && !skipped,
		"confirm":   pending,
		"skip":      pending && recurring,
	}
}

type TxInput struct {
	Type         string
	AccountID    int64
	ToAccountID  *int64
	CategoryID   *int64
	Amount       float64
	ToAmount     *float64
	ExchangeRate *float64
	Description  *string
	Date         *string
	Status       *string
	RecurringID  *int64
	TagIDs       []int64
	Items        []TxItem
}

type Transactions struct{ DB *sql.DB }

func (s Transactions) Create(ctx context.Context, in TxInput) (*Transaction, error) {
	status := "confirmed"
	if in.Status != nil {
		status = *in.Status
	} else if in.Date == nil || isFuture(*in.Date) {
		status = "pending"
	}
	if status == "confirmed" {
		if in.Date == nil {
			return nil, fmt.Errorf("date required")
		}
		if isFuture(*in.Date) {
			return nil, fmt.Errorf("future")
		}
	}
	if in.Type == "transfer" && in.ToAccountID != nil && in.ToAmount == nil {
		amt := in.Amount
		in.ToAmount = &amt
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertTransaction(ctx, sqlc.InsertTransactionParams{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: db.NullInt64(in.ToAccountID),
		CategoryID: db.NullInt64(in.CategoryID), Amount: in.Amount, ToAmount: db.NullFloat64(in.ToAmount),
		ExchangeRate: db.NullFloat64(in.ExchangeRate), Description: db.NullString(in.Description),
		Date: db.NullString(in.Date), Status: status, RecurringTransactionID: db.NullInt64(in.RecurringID),
		CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if err := s.saveItems(ctx, id, in.Items); err != nil {
		return nil, err
	}
	if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Update(ctx context.Context, id int64, in TxInput) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.Status == "skipped" || cur.RecurringID != nil {
		return nil, fmt.Errorf("cannot edit")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: db.NullInt64(in.ToAccountID),
		CategoryID: db.NullInt64(in.CategoryID), Amount: in.Amount, ToAmount: db.NullFloat64(in.ToAmount),
		ExchangeRate: db.NullFloat64(in.ExchangeRate), Description: db.NullString(in.Description),
		Date: db.NullString(in.Date), UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	if in.Items != nil {
		_ = db.Q(s.DB).DeleteTransactionItems(ctx, id)
		if err := s.saveItems(ctx, id, in.Items); err != nil {
			return nil, err
		}
	}
	if in.TagIDs != nil {
		if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
			return nil, err
		}
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Delete(ctx context.Context, id int64) error {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return err
	}
	if cur.RecurringID != nil {
		return fmt.Errorf("cannot delete recurring")
	}
	return db.Q(s.DB).DeleteTransaction(ctx, id)
}

func (s Transactions) Confirm(ctx context.Context, id int64, date string) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.Status != "pending" {
		return nil, fmt.Errorf("not pending")
	}
	if date == "" && cur.Date != nil {
		date = *cur.Date
	}
	if date == "" {
		return nil, fmt.Errorf("date required")
	}
	if isFuture(date) {
		return nil, fmt.Errorf("future")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).ConfirmTransaction(ctx, sqlc.ConfirmTransactionParams{Date: db.NS(date), UpdatedAt: db.NS(now), ID: id})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Skip(ctx context.Context, id int64) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.Status != "pending" || cur.RecurringID == nil {
		return nil, fmt.Errorf("cannot skip")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).SkipTransaction(ctx, sqlc.SkipTransactionParams{UpdatedAt: db.NS(now), ID: id})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Duplicate(ctx context.Context, id int64) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.RecurringID != nil {
		return nil, fmt.Errorf("cannot duplicate")
	}
	today := time.Now().UTC().Format("2006-01-02")
	in := TxInput{
		Type: cur.Type, AccountID: cur.AccountID, ToAccountID: cur.ToAccountID,
		CategoryID: cur.CategoryID, Amount: cur.Amount, ToAmount: cur.ToAmount,
		ExchangeRate: cur.ExchangeRate, Description: cur.Description, Date: &today,
		Items: cur.Items,
	}
	for _, t := range cur.Tags {
		in.TagIDs = append(in.TagIDs, t.ID)
	}
	st := "confirmed"
	in.Status = &st
	return s.Create(ctx, in)
}

func (s Transactions) ByID(ctx context.Context, id int64) (*Transaction, error) {
	list, err := s.list(ctx, sqlc.ListTransactionsParams{ID: db.NI(id), Limit: 1, Offset: 0})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Transactions) Filtered(ctx context.Context, f filter.TxFilter, page, perPage int) ([]Transaction, int, error) {
	if perPage <= 0 {
		perPage = 25
	}
	if page <= 0 {
		page = 1
	}
	arg := sqlc.CountTransactionsParams{
		Type: db.Narg(f.Type), AccountID: db.NullInt64If(f.AccountID), CategoryID: db.NullInt64If(f.CategoryID),
		Status: db.Narg(f.Status), StartDate: db.Narg(f.StartDate), EndDate: db.Narg(f.EndDate),
	}
	total, err := db.Q(s.DB).CountTransactions(ctx, arg)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.list(ctx, sqlc.ListTransactionsParams{
		Type: arg.Type, AccountID: arg.AccountID, CategoryID: arg.CategoryID,
		Status: arg.Status, StartDate: arg.StartDate, EndDate: arg.EndDate,
		Limit: int64(perPage), Offset: int64((page - 1) * perPage),
	})
	return list, int(total), err
}

func (s Transactions) Summary(ctx context.Context, pendingOnly bool) map[string]any {
	status := "confirmed"
	if pendingOnly {
		status = "pending"
	}
	rows, err := db.Q(s.DB).ListTransactionSummaryRows(ctx, status)
	if err != nil {
		return map[string]any{"income": 0, "expense": 0, "balance": 0, "transactions_count": 0, "currency": nil}
	}
	var income, expense float64
	n := 0
	for _, r := range rows {
		amount := r.Amount
		if r.IsBase == 0 && r.Rate != 0 {
			amount *= r.Rate
		}
		if r.Type == "income" {
			income += amount
		} else {
			expense += amount
		}
		n++
	}
	code, _ := db.Q(s.DB).GetBaseCurrencyCode(ctx)
	return map[string]any{
		"income": income, "expense": expense, "balance": income - expense,
		"transactions_count": n, "currency": nilOr(code),
	}
}

func (s Transactions) list(ctx context.Context, arg sqlc.ListTransactionsParams) ([]Transaction, error) {
	rows, err := db.Q(s.DB).ListTransactions(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Transaction, 0, len(rows))
	for _, r := range rows {
		out = append(out, txFromRow(r))
	}
	accts := Accounts{DB: s.DB}
	cats := Categories{DB: s.DB}
	for i := range out {
		if a, _ := accts.ByID(ctx, out[i].AccountID); a != nil {
			out[i].Account = a
		}
		if out[i].ToAccountID != nil {
			if a, _ := accts.ByID(ctx, *out[i].ToAccountID); a != nil {
				out[i].ToAccount = a
			}
		}
		if out[i].CategoryID != nil {
			if c, _ := cats.ByID(ctx, *out[i].CategoryID); c != nil {
				out[i].Category = c
			}
		}
		out[i].Items, _ = s.items(ctx, out[i].ID)
		out[i].Tags, _ = s.tags(ctx, out[i].ID)
	}
	return out, nil
}

func (s Transactions) items(ctx context.Context, id int64) ([]TxItem, error) {
	rows, err := db.Q(s.DB).ListTransactionItems(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]TxItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, TxItem{ID: r.ID, Name: r.Name, Quantity: r.Quantity, PricePerUnit: r.PricePerUnit, TotalPrice: r.TotalPrice})
	}
	return out, nil
}

func (s Transactions) tags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := db.Q(s.DB).ListTransactionTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, tagFromList(r.ID, r.Name, r.CreatedAt, r.TransactionsCount))
	}
	return out, nil
}

func (s Transactions) saveItems(ctx context.Context, txID int64, items []TxItem) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, it := range items {
		if it.TotalPrice == 0 {
			it.TotalPrice = it.Quantity * it.PricePerUnit
		}
		if err := db.Q(s.DB).InsertTransactionItem(ctx, sqlc.InsertTransactionItemParams{
			TransactionID: txID, Name: it.Name, Quantity: it.Quantity, PricePerUnit: it.PricePerUnit,
			TotalPrice: it.TotalPrice, CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s Transactions) saveTags(ctx context.Context, txID int64, ids []int64) error {
	_ = db.Q(s.DB).DeleteTransactionTags(ctx, txID)
	for _, id := range ids {
		if err := db.Q(s.DB).InsertTransactionTag(ctx, sqlc.InsertTransactionTagParams{TransactionID: txID, TagID: id}); err != nil {
			return err
		}
	}
	return nil
}

func txFromRow(r sqlc.ListTransactionsRow) Transaction {
	t := Transaction{ID: r.ID, Type: r.Type, AccountID: r.AccountID, Amount: r.Amount, Status: r.Status}
	if r.ToAccountID.Valid {
		t.ToAccountID = &r.ToAccountID.Int64
	}
	if r.CategoryID.Valid {
		t.CategoryID = &r.CategoryID.Int64
	}
	if r.ToAmount.Valid {
		t.ToAmount = &r.ToAmount.Float64
	}
	if r.ExchangeRate.Valid {
		t.ExchangeRate = &r.ExchangeRate.Float64
	}
	if r.Description.Valid {
		t.Description = &r.Description.String
	}
	if r.Date.Valid {
		t.Date = &r.Date.String
	}
	if r.RecurringTransactionID.Valid {
		t.RecurringID = &r.RecurringTransactionID.Int64
	}
	if tm, ok := parseNullTime(r.CreatedAt); ok {
		t.CreatedAt = &tm
	}
	return t
}

func isFuture(date string) bool {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return d.After(today)
}

func mapSliceVal[T any](in []T, fn func(T) map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		out = append(out, fn(v))
	}
	return out
}

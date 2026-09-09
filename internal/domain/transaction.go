package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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
	ID                     int64
	Type                   string
	AccountID              int64
	ToAccountID            *int64
	CategoryID             *int64
	Amount                 float64
	ToAmount               *float64
	ExchangeRate           *float64
	Description            *string
	Date                   *string
	Status                 string
	RecurringID            *int64
	CreatedAt              *time.Time
	Account                *Account
	ToAccount              *Account
	Category               *Category
	Items                  []TxItem
	Tags                   []Tag
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
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      float64
	ToAmount    *float64
	ExchangeRate *float64
	Description *string
	Date        *string
	Status      *string
	TagIDs      []int64
	Items       []TxItem
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
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO transactions (type, account_id, to_account_id, category_id, amount, to_amount, exchange_rate,
			description, date, status, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		in.Type, in.AccountID, in.ToAccountID, in.CategoryID, in.Amount, in.ToAmount, in.ExchangeRate,
		in.Description, in.Date, status, now, now)
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
	_, err = s.DB.ExecContext(ctx, `
		UPDATE transactions SET type=?, account_id=?, to_account_id=?, category_id=?, amount=?, to_amount=?,
			exchange_rate=?, description=?, date=?, updated_at=? WHERE id=?`,
		in.Type, in.AccountID, in.ToAccountID, in.CategoryID, in.Amount, in.ToAmount,
		in.ExchangeRate, in.Description, in.Date, now, id)
	if err != nil {
		return nil, err
	}
	if in.Items != nil {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM transaction_items WHERE transaction_id = ?`, id)
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
	_, err = s.DB.ExecContext(ctx, `DELETE FROM transactions WHERE id = ?`, id)
	return err
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
	_, err = s.DB.ExecContext(ctx, `UPDATE transactions SET status='confirmed', date=?, updated_at=? WHERE id=?`, date, now, id)
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
	_, err = s.DB.ExecContext(ctx, `UPDATE transactions SET status='skipped', updated_at=? WHERE id=?`, now, id)
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
	list, err := s.list(ctx, `WHERE t.id = ?`, id)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Transactions) Filtered(ctx context.Context, q string, args []any, page, perPage int) ([]Transaction, int, error) {
	if perPage <= 0 {
		perPage = 25
	}
	if page <= 0 {
		page = 1
	}
	var total int
	countQ := `SELECT COUNT(*) FROM transactions t ` + q
	if err := s.DB.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	list, err := s.list(ctx, q+` ORDER BY t.date DESC, t.id DESC LIMIT ? OFFSET ?`,
		append(args, perPage, (page-1)*perPage)...)
	return list, total, err
}

func (s Transactions) Summary(ctx context.Context, pendingOnly bool) map[string]any {
	status := "confirmed"
	if pendingOnly {
		status = "pending"
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT t.type, t.amount, c.rate, c.is_base
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		WHERE t.status = ? AND t.type IN ('income','expense')`, status)
	if err != nil {
		return map[string]any{"income": 0, "expense": 0, "balance": 0, "transactions_count": 0, "currency": nil}
	}
	defer rows.Close()
	var income, expense float64
	n := 0
	for rows.Next() {
		var typ string
		var amount, rate float64
		var base int
		_ = rows.Scan(&typ, &amount, &rate, &base)
		if base == 0 && rate != 0 {
			amount *= rate
		}
		if typ == "income" {
			income += amount
		} else {
			expense += amount
		}
		n++
	}
	var code sql.NullString
	_ = s.DB.QueryRowContext(ctx, `SELECT code FROM currencies WHERE is_base = 1`).Scan(&code)
	return map[string]any{
		"income": income, "expense": expense, "balance": income - expense,
		"transactions_count": n, "currency": nilOr(code.String),
	}
}

func (s Transactions) list(ctx context.Context, where string, args ...any) ([]Transaction, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT t.id, t.type, t.account_id, t.to_account_id, t.category_id, t.amount, t.to_amount, t.exchange_rate,
			t.description, t.date, t.status, t.recurring_transaction_id, t.created_at
		FROM transactions t `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		t, err := scanTx(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
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
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, quantity, price_per_unit, total_price FROM transaction_items WHERE transaction_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TxItem
	for rows.Next() {
		var i TxItem
		if err := rows.Scan(&i.ID, &i.Name, &i.Quantity, &i.PricePerUnit, &i.TotalPrice); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s Transactions) tags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT tags.id, tags.name, tags.created_at, 0 FROM tags
		JOIN transaction_tag tt ON tt.tag_id = tags.id WHERE tt.transaction_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s Transactions) saveItems(ctx context.Context, txID int64, items []TxItem) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, it := range items {
		if it.TotalPrice == 0 {
			it.TotalPrice = it.Quantity * it.PricePerUnit
		}
		if _, err := s.DB.ExecContext(ctx, `
			INSERT INTO transaction_items (transaction_id, name, quantity, price_per_unit, total_price, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?)`, txID, it.Name, it.Quantity, it.PricePerUnit, it.TotalPrice, now, now); err != nil {
			return err
		}
	}
	return nil
}

func (s Transactions) saveTags(ctx context.Context, txID int64, ids []int64) error {
	_, _ = s.DB.ExecContext(ctx, `DELETE FROM transaction_tag WHERE transaction_id = ?`, txID)
	for _, id := range ids {
		if _, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO transaction_tag (transaction_id, tag_id) VALUES (?,?)`, txID, id); err != nil {
			return err
		}
	}
	return nil
}

func scanTx(row interface{ Scan(...any) error }) (Transaction, error) {
	var t Transaction
	var toAcc, cat, rec sql.NullInt64
	var toAmt, rate sql.NullFloat64
	var desc, date, created sql.NullString
	err := row.Scan(&t.ID, &t.Type, &t.AccountID, &toAcc, &cat, &t.Amount, &toAmt, &rate, &desc, &date, &t.Status, &rec, &created)
	if toAcc.Valid {
		t.ToAccountID = &toAcc.Int64
	}
	if cat.Valid {
		t.CategoryID = &cat.Int64
	}
	if toAmt.Valid {
		t.ToAmount = &toAmt.Float64
	}
	if rate.Valid {
		t.ExchangeRate = &rate.Float64
	}
	if desc.Valid {
		t.Description = &desc.String
	}
	if date.Valid {
		t.Date = &date.String
	}
	if rec.Valid {
		t.RecurringID = &rec.Int64
	}
	if tm, ok := parseNullTime(created); ok {
		t.CreatedAt = &tm
	}
	return t, err
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


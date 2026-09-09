package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Account struct {
	ID             int64
	Name           string
	Type           string
	CurrencyID     int64
	InitialBalance float64
	IsActive       bool
	SortOrder      int
	DebtType       *string
	TargetAmount   *float64
	DueDate        *string
	IsPaidOff      bool
	Counterparty   *string
	DebtDesc       *string
	CreatedAt      *time.Time
	Currency       *Currency
	Balance        float64
}

func (a Account) JSON() map[string]any {
	var created any
	if a.CreatedAt != nil {
		created = a.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	var cur any
	if a.Currency != nil {
		cur = a.Currency.JSON()
	}
	return map[string]any{
		"id":             a.ID,
		"name":           a.Name,
		"type":           a.Type,
		"currencyId":     a.CurrencyID,
		"initialBalance": a.InitialBalance,
		"currentBalance": a.Balance,
		"isActive":       a.IsActive,
		"sortOrder":      a.SortOrder,
		"currency":       cur,
		"createdAt":      created,
	}
}

type Accounts struct{ DB *sql.DB }

func (s Accounts) All(ctx context.Context, onlyActive, excludeDebts bool) ([]Account, error) {
	q := accountSelect + ` WHERE 1=1`
	var args []any
	if onlyActive {
		q += ` AND a.is_active = 1`
	}
	if excludeDebts {
		q += ` AND a.type IN ('bank','crypto','cash')`
	}
	q += ` ORDER BY CASE WHEN a.type = 'debt' THEN 1 ELSE 0 END, a.sort_order, a.id`
	return s.list(ctx, q, args...)
}

func (s Accounts) ByID(ctx context.Context, id int64) (*Account, error) {
	list, err := s.list(ctx, accountSelect+` WHERE a.id = ?`, id)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Accounts) Create(ctx context.Context, a Account) (*Account, error) {
	if a.Type == "" {
		a.Type = "cash"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var max sql.NullInt64
	scope := `type IN ('bank','crypto','cash')`
	if a.Type == "debt" {
		scope = `type = 'debt'`
	}
	_ = s.DB.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM accounts WHERE `+scope).Scan(&max)
	a.SortOrder = int(max.Int64) + 1
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO accounts (name, type, debt_type, currency_id, initial_balance, target_amount, due_date,
			is_paid_off, counterparty, debt_description, is_active, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Name, a.Type, a.DebtType, a.CurrencyID, a.InitialBalance, a.TargetAmount, a.DueDate,
		boolInt(a.IsPaidOff), a.Counterparty, a.DebtDesc, boolInt(a.IsActive), a.SortOrder, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Accounts) Update(ctx context.Context, id int64, a Account) (*Account, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx, `
		UPDATE accounts SET name=?, type=?, currency_id=?, initial_balance=?, is_active=?,
			debt_type=?, target_amount=?, due_date=?, is_paid_off=?, counterparty=?, debt_description=?, updated_at=?
		WHERE id=?`,
		a.Name, a.Type, a.CurrencyID, a.InitialBalance, boolInt(a.IsActive),
		a.DebtType, a.TargetAmount, a.DueDate, boolInt(a.IsPaidOff), a.Counterparty, a.DebtDesc, now, id,
	)
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Accounts) Delete(ctx context.Context, id int64) error {
	var n int
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions WHERE account_id = ? OR to_account_id = ?`, id, id).Scan(&n)
	if n > 0 {
		return fmt.Errorf("has transactions")
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	return err
}

func (s Accounts) Reorder(ctx context.Context, ids []int64) error {
	for i, id := range ids {
		if _, err := s.DB.ExecContext(ctx, `UPDATE accounts SET sort_order = ? WHERE id = ?`, i, id); err != nil {
			return err
		}
	}
	return nil
}

func (s Accounts) Summary(ctx context.Context, base *Currency) map[string]any {
	accts, _ := s.All(ctx, true, true)
	total := 0.0
	code := ""
	decimals := 2
	if base != nil {
		code = base.Code
		decimals = base.Decimals
		for _, a := range accts {
			if a.Currency == nil {
				continue
			}
			total += Convert(a.Balance, *a.Currency, *base)
		}
	}
	return map[string]any{
		"total_balance":  roundTo(total, decimals),
		"currency":       nilOr(code),
		"currency_code":  code,
		"decimals":       decimals,
		"accounts_count": len(accts),
	}
}

func (s Accounts) list(ctx context.Context, q string, args ...any) ([]Account, error) {
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range out {
		out[i].Balance, _ = s.balance(ctx, out[i])
	}
	return out, nil
}

func (s Accounts) balance(ctx context.Context, a Account) (float64, error) {
	if a.Type == "debt" {
		var paid sql.NullFloat64
		_ = s.DB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(to_amount),0) FROM transactions
			WHERE to_account_id = ? AND status = 'confirmed' AND type IN ('debt_payment','debt_collection')`, a.ID).Scan(&paid)
		target := 0.0
		if a.TargetAmount != nil {
			target = *a.TargetAmount
		}
		return target - paid.Float64, nil
	}
	var income, expense, tout, tin, dIn, dOut sql.NullFloat64
	_ = s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM transactions WHERE account_id=? AND status='confirmed' AND type='income'`, a.ID).Scan(&income)
	_ = s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM transactions WHERE account_id=? AND status='confirmed' AND type='expense'`, a.ID).Scan(&expense)
	_ = s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM transactions WHERE account_id=? AND status='confirmed' AND type='transfer'`, a.ID).Scan(&tout)
	_ = s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(to_amount),0) FROM transactions WHERE to_account_id=? AND status='confirmed'`, a.ID).Scan(&tin)
	_ = s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM transactions WHERE account_id=? AND status='confirmed' AND type IN ('debt_collection','debt_borrow')`, a.ID).Scan(&dIn)
	_ = s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM transactions WHERE account_id=? AND status='confirmed' AND type IN ('debt_payment','debt_lend')`, a.ID).Scan(&dOut)
	return a.InitialBalance + income.Float64 - expense.Float64 - tout.Float64 + tin.Float64 + dIn.Float64 - dOut.Float64, nil
}

const accountSelect = `SELECT a.id, a.name, a.type, a.currency_id, a.initial_balance, a.is_active, a.sort_order,
	a.debt_type, a.target_amount, a.due_date, a.is_paid_off, a.counterparty, a.debt_description, a.created_at,
	c.id, c.code, c.name, c.symbol, c.decimals, c.is_base, c.rate
	FROM accounts a JOIN currencies c ON c.id = a.currency_id`

func scanAccount(row interface{ Scan(...any) error }) (Account, error) {
	var a Account
	var active, paid, base int
	var debtType, due, counter, desc, created sql.NullString
	var target sql.NullFloat64
	var cur Currency
	err := row.Scan(&a.ID, &a.Name, &a.Type, &a.CurrencyID, &a.InitialBalance, &active, &a.SortOrder,
		&debtType, &target, &due, &paid, &counter, &desc, &created,
		&cur.ID, &cur.Code, &cur.Name, &cur.Symbol, &cur.Decimals, &base, &cur.Rate)
	if err != nil {
		return a, err
	}
	a.IsActive = active != 0
	a.IsPaidOff = paid != 0
	cur.IsBase = base != 0
	a.Currency = &cur
	if debtType.Valid {
		a.DebtType = &debtType.String
	}
	if target.Valid {
		a.TargetAmount = &target.Float64
	}
	if due.Valid {
		a.DueDate = &due.String
	}
	if counter.Valid {
		a.Counterparty = &counter.String
	}
	if desc.Valid {
		a.DebtDesc = &desc.String
	}
	if t, ok := parseNullTime(created); ok {
		a.CreatedAt = &t
	}
	return a, nil
}

func parseNullTime(raw sql.NullString) (time.Time, bool) {
	if !raw.Valid || raw.String == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw.String, time.UTC); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func nilOr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

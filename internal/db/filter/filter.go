// Package filter confines dynamic SQL that cannot be expressed as a single
// sqlc query: variable-length IN lists and report GROUP BY period expressions.
package filter

import (
	"context"
	"database/sql"
	"strings"
)

// TxFilter is the HTTP/list filter for transactions. Optional fields are
// empty when unset; IDs use 0 as unset. Domain code maps this to sqlc nargs.
type TxFilter struct {
	Type       string
	AccountID  int64
	CategoryID int64
	Status     string
	StartDate  string
	EndDate    string
}

// ReportWhere is the shared report/budget aggregation filter.
type ReportWhere struct {
	Type        string
	Start       string
	End         string
	CategoryID  int64
	AccountIDs  []int64
	CategoryIDs []int64
	TagIDs      []int64
}

// Clause builds `WHERE …` plus args for confirmed transactions in a range.
func Clause(w ReportWhere) (string, []any) {
	q := `WHERE t.status = 'confirmed' AND t.type = ? AND t.date >= ? AND t.date <= ?`
	args := []any{w.Type, w.Start, w.End}
	if w.CategoryID > 0 {
		q += ` AND t.category_id = ?`
		args = append(args, w.CategoryID)
	}
	if len(w.AccountIDs) > 0 {
		q += ` AND t.account_id IN (` + Placeholders(len(w.AccountIDs)) + `)`
		for _, id := range w.AccountIDs {
			args = append(args, id)
		}
	}
	if len(w.CategoryIDs) > 0 {
		q += ` AND t.category_id IN (` + Placeholders(len(w.CategoryIDs)) + `)`
		for _, id := range w.CategoryIDs {
			args = append(args, id)
		}
	}
	if len(w.TagIDs) > 0 {
		q += ` AND EXISTS (SELECT 1 FROM transaction_tag tt WHERE tt.transaction_id = t.id AND tt.tag_id IN (` + Placeholders(len(w.TagIDs)) + `))`
		for _, id := range w.TagIDs {
			args = append(args, id)
		}
	}
	return q, args
}

// Placeholders returns n comma-separated `?` markers.
func Placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// PeriodExpr is the SQLite date grouping expression for report series.
func PeriodExpr(groupBy string) string {
	switch groupBy {
	case "week":
		return `DATE(t.date, '-' || strftime('%w', t.date) || ' days')`
	case "month":
		return `DATE(t.date, 'start of month')`
	default:
		return `DATE(t.date)`
	}
}

func SumByType(ctx context.Context, sqlDB *sql.DB, w ReportWhere) float64 {
	where, args := Clause(w)
	var total sql.NullFloat64
	_ = sqlDB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(t.amount * c.rate), 0)
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		`+where, args...).Scan(&total)
	return total.Float64
}

type CatTotal struct {
	ID    int64
	Name  string
	Icon  sql.NullString
	Color sql.NullString
	Total float64
}

func SumGroupedByCategory(ctx context.Context, sqlDB *sql.DB, w ReportWhere) []CatTotal {
	where, args := Clause(w)
	where += ` AND t.category_id IS NOT NULL`
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT cat.id, cat.name, cat.icon, cat.color, SUM(t.amount * c.rate) as total
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		JOIN categories cat ON cat.id = t.category_id
		`+where+`
		GROUP BY cat.id, cat.name, cat.icon, cat.color
		ORDER BY total DESC`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []CatTotal
	for rows.Next() {
		var x CatTotal
		if err := rows.Scan(&x.ID, &x.Name, &x.Icon, &x.Color, &x.Total); err != nil {
			return nil
		}
		out = append(out, x)
	}
	return out
}

type DayTotal struct {
	Day   string
	Total float64
	Count int
}

func DailyTotals(ctx context.Context, sqlDB *sql.DB, w ReportWhere) []DayTotal {
	where, args := Clause(w)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT DATE(t.date), SUM(t.amount * c.rate), COUNT(*)
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		`+where+` GROUP BY DATE(t.date)`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []DayTotal
	for rows.Next() {
		var x DayTotal
		if err := rows.Scan(&x.Day, &x.Total, &x.Count); err != nil {
			return nil
		}
		out = append(out, x)
	}
	return out
}

type PeriodTotal struct {
	Key   string
	Total float64
}

func GroupedByPeriod(ctx context.Context, sqlDB *sql.DB, w ReportWhere, groupBy string) []PeriodTotal {
	where, args := Clause(w)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT `+PeriodExpr(groupBy)+` as period_date, SUM(t.amount * c.rate)
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		`+where+` GROUP BY period_date`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []PeriodTotal
	for rows.Next() {
		var x PeriodTotal
		if err := rows.Scan(&x.Key, &x.Total); err != nil {
			return nil
		}
		out = append(out, x)
	}
	return out
}

type TopRow struct {
	ID          int64
	Description sql.NullString
	Date        sql.NullString
	Amount      float64
	CatID       sql.NullInt64
	CatName     sql.NullString
	Icon        sql.NullString
	Color       sql.NullString
	AccID       int64
	AccName     sql.NullString
}

func TopTransactions(ctx context.Context, sqlDB *sql.DB, w ReportWhere, limit int) []TopRow {
	where, args := Clause(w)
	args = append(args, limit)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT t.id, t.description, t.date, t.amount * c.rate,
			cat.id, cat.name, cat.icon, cat.color, a.id, a.name
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		LEFT JOIN categories cat ON cat.id = t.category_id
		`+where+` ORDER BY t.amount * c.rate DESC LIMIT ?`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []TopRow
	for rows.Next() {
		var x TopRow
		if err := rows.Scan(&x.ID, &x.Description, &x.Date, &x.Amount, &x.CatID, &x.CatName, &x.Icon, &x.Color, &x.AccID, &x.AccName); err != nil {
			continue
		}
		out = append(out, x)
	}
	return out
}

type BudgetAmount struct {
	Amount float64
	Rate   float64
	IsBase bool
}

func ScopedMonthlyBudgets(ctx context.Context, sqlDB *sql.DB, categoryIDs, tagIDs []int64) []BudgetAmount {
	q := `SELECT b.amount, c.rate, c.is_base FROM budgets b
		LEFT JOIN currencies c ON c.id = b.currency_id
		WHERE b.is_active = 1 AND b.period = 'monthly' AND b.is_global = 0 AND (`
	var args []any
	parts := []string{}
	if len(categoryIDs) > 0 {
		parts = append(parts, `EXISTS (SELECT 1 FROM budget_category bc WHERE bc.budget_id = b.id AND bc.category_id IN (`+Placeholders(len(categoryIDs))+`))`)
		for _, id := range categoryIDs {
			args = append(args, id)
		}
	}
	if len(tagIDs) > 0 {
		parts = append(parts, `EXISTS (SELECT 1 FROM budget_tag bt WHERE bt.budget_id = b.id AND bt.tag_id IN (`+Placeholders(len(tagIDs))+`))`)
		for _, id := range tagIDs {
			args = append(args, id)
		}
	}
	q += strings.Join(parts, " OR ") + `)`
	return scanBudgetAmounts(ctx, sqlDB, q, args)
}

func scanBudgetAmounts(ctx context.Context, sqlDB *sql.DB, q string, args []any) []BudgetAmount {
	rows, err := sqlDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []BudgetAmount
	for rows.Next() {
		var amount, rate sql.NullFloat64
		var base sql.NullInt64
		if err := rows.Scan(&amount, &rate, &base); err != nil {
			return nil
		}
		out = append(out, BudgetAmount{Amount: amount.Float64, Rate: rate.Float64, IsBase: base.Valid && base.Int64 != 0})
	}
	return out
}

// BudgetSpent sums confirmed expenses in [start,end], optionally scoped by
// category/tag IN lists. Variable-length IN cannot be a single sqlc query.
func BudgetSpent(ctx context.Context, sqlDB *sql.DB, start, end string, categoryIDs, tagIDs []int64) (float64, error) {
	q := `
		SELECT COALESCE(SUM(t.amount * c.rate), 0)
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		WHERE t.status = 'confirmed' AND t.type = 'expense'
			AND t.date >= ? AND t.date <= ?`
	args := []any{start, end}
	if categoryIDs != nil {
		if len(categoryIDs) == 0 {
			return 0, nil
		}
		q += ` AND t.category_id IN (` + Placeholders(len(categoryIDs)) + `)`
		for _, id := range categoryIDs {
			args = append(args, id)
		}
	}
	if len(tagIDs) > 0 {
		q += ` AND EXISTS (
			SELECT 1 FROM transaction_tag tt
			WHERE tt.transaction_id = t.id AND tt.tag_id IN (` + Placeholders(len(tagIDs)) + `))`
		for _, id := range tagIDs {
			args = append(args, id)
		}
	}
	var total sql.NullFloat64
	if err := sqlDB.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total.Float64, nil
}

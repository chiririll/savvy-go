// Package filter confines dynamic SQL that cannot be expressed as a single
// sqlc query: variable-length IN lists and report GROUP BY period expressions.
package filter

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

// The aggregates below fold amounts of several currencies into one currency.
// Each row is converted at its currency's rate and the converted values are
// added unrounded, so a total rounds once, into a money.Money, instead of once
// per currency group.

// major is minor units of a currency with the given decimals in major units.
func major(minor, decimals int64) decimal.Decimal {
	return decimal.New(minor, -int32(decimals))
}

// toBase converts minor units of a currency (given its decimals and rate to
// the base currency) into base-currency major units.
func toBase(minor, decimals int64, rate decimal.Decimal) decimal.Decimal {
	return major(minor, decimals).Mul(rate)
}

// TxFilter is the HTTP/list filter for transactions. Optional fields are
// empty when unset; IDs use 0 as unset. Domain code maps this to sqlc nargs.
type TxFilter struct {
	Type        string
	AccountID   int64
	CategoryID  int64
	CategoryIDs []int64
	TagIDs      []int64
	Status      string
	StartDate   string
	EndDate     string
	SortBy      string
	SortDir     string
}

func txWhere(f TxFilter) (string, []any) {
	q := `WHERE 1=1`
	var args []any
	if f.Type != "" {
		q += ` AND t.type = ?`
		args = append(args, f.Type)
	}
	if f.AccountID > 0 {
		q += ` AND t.account_id = ?`
		args = append(args, f.AccountID)
	}
	if f.CategoryID > 0 {
		q += ` AND t.category_id = ?`
		args = append(args, f.CategoryID)
	}
	if len(f.CategoryIDs) > 0 {
		q += ` AND t.category_id IN (` + Placeholders(len(f.CategoryIDs)) + `)`
		for _, id := range f.CategoryIDs {
			args = append(args, id)
		}
	}
	if len(f.TagIDs) > 0 {
		q += ` AND EXISTS (SELECT 1 FROM transaction_tag tt WHERE tt.transaction_id = t.id AND tt.tag_id IN (` + Placeholders(len(f.TagIDs)) + `))`
		for _, id := range f.TagIDs {
			args = append(args, id)
		}
	}
	if f.Status != "" {
		q += ` AND t.status = ?`
		args = append(args, f.Status)
	}
	if f.StartDate != "" {
		q += ` AND (t.date IS NULL OR t.date >= ?)`
		args = append(args, f.StartDate)
	}
	if f.EndDate != "" {
		q += ` AND (t.date IS NULL OR t.date <= ?)`
		args = append(args, f.EndDate)
	}
	return q, args
}

// txOrderBy maps whitelisted sort keys to SQL; request strings never reach the query.
func txOrderBy(sortBy, sortDir string) string {
	dir := "DESC"
	if sortDir == "asc" {
		dir = "ASC"
	}
	switch sortBy {
	case "amount":
		return `t.amount * COALESCE(NULLIF(CAST(c.rate AS REAL), 0), 1) / pow(10, c.decimals) ` + dir + `, t.id ` + dir
	case "created_at":
		return `t.created_at ` + dir + `, t.id ` + dir
	default:
		return `CASE WHEN t.date IS NULL THEN 1 ELSE 0 END, t.date ` + dir + `, t.id ` + dir
	}
}

func CountTransactions(ctx context.Context, sqlDB *sql.DB, f TxFilter) (int64, error) {
	where, args := txWhere(f)
	var n int64
	err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions t `+where, args...).Scan(&n)
	return n, err
}

func ListTransactions(ctx context.Context, sqlDB *sql.DB, f TxFilter, limit, offset int) ([]sqlc.GetTransactionRow, error) {
	where, args := txWhere(f)
	args = append(args, limit, offset)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT t.id, t.type, t.account_id, t.to_account_id, t.category_id, t.amount, t.to_amount,
			t.is_estimated, t.description, t.date, t.status, t.recurring_transaction_id, t.created_at,
			c.id, c.decimals, COALESCE(cb.id, c.id), COALESCE(cb.decimals, c.decimals)
		FROM transactions t
		LEFT JOIN accounts a ON a.id = t.account_id
		LEFT JOIN currencies c ON c.id = a.currency_id
		LEFT JOIN accounts ta ON ta.id = t.to_account_id
		LEFT JOIN currencies cb ON cb.id = ta.currency_id
		`+where+`
		ORDER BY `+txOrderBy(f.SortBy, f.SortDir)+`
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []sqlc.GetTransactionRow{}
	for rows.Next() {
		var r sqlc.GetTransactionRow
		if err := rows.Scan(&r.ID, &r.Type, &r.AccountID, &r.ToAccountID, &r.CategoryID, &r.Amount, &r.ToAmount,
			&r.IsEstimated, &r.Description, &r.Date, &r.Status, &r.RecurringTransactionID, &r.CreatedAt,
			&r.CurrencyID, &r.Decimals, &r.ToCurrencyID, &r.ToDecimals); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
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

// SumByType is the total of the matching transactions in the base currency.
func SumByType(ctx context.Context, sqlDB *sql.DB, w ReportWhere, base money.Unit) money.Money {
	where, args := Clause(w)
	total := decimal.Zero
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT COALESCE(SUM(t.amount), 0), c.decimals, c.rate
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		`+where+` GROUP BY c.id`, args...)
	if err != nil {
		return money.Zero(base)
	}
	defer rows.Close()
	for rows.Next() {
		var minor, decimals int64
		var rate decimal.Decimal
		if err := rows.Scan(&minor, &decimals, &rate); err != nil {
			return money.Zero(base)
		}
		total = total.Add(toBase(minor, decimals, rate))
	}
	return money.FromDecimal(total, base)
}

type CatTotal struct {
	ID    int64
	Name  string
	Icon  sql.NullString
	Color sql.NullString
	Total money.Money
}

// SumGroupedByCategory is the total per category in the base currency, largest
// first.
func SumGroupedByCategory(ctx context.Context, sqlDB *sql.DB, w ReportWhere, base money.Unit) []CatTotal {
	where, args := Clause(w)
	where += ` AND t.category_id IS NOT NULL`
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT cat.id, cat.name, cat.icon, cat.color, SUM(t.amount), c.decimals, c.rate
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		JOIN categories cat ON cat.id = t.category_id
		`+where+`
		GROUP BY cat.id, cat.name, cat.icon, cat.color, c.id`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []CatTotal
	sums := map[int64]decimal.Decimal{}
	idx := map[int64]int{}
	for rows.Next() {
		var x CatTotal
		var minor, decimals int64
		var rate decimal.Decimal
		if err := rows.Scan(&x.ID, &x.Name, &x.Icon, &x.Color, &minor, &decimals, &rate); err != nil {
			return nil
		}
		if _, ok := idx[x.ID]; !ok {
			idx[x.ID] = len(out)
			out = append(out, x)
		}
		sums[x.ID] = sums[x.ID].Add(toBase(minor, decimals, rate))
	}
	for i := range out {
		out[i].Total = money.FromDecimal(sums[out[i].ID], base)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Total.Cmp(out[j].Total) > 0 })
	return out
}

type DayTotal struct {
	Day   string
	Total money.Money
	Count int
}

// DailyTotals is the total and the number of transactions per day in the base
// currency.
func DailyTotals(ctx context.Context, sqlDB *sql.DB, w ReportWhere, base money.Unit) []DayTotal {
	where, args := Clause(w)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT DATE(t.date), SUM(t.amount), COUNT(*), c.decimals, c.rate
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		`+where+` GROUP BY DATE(t.date), c.id`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []DayTotal
	sums := map[string]decimal.Decimal{}
	idx := map[string]int{}
	for rows.Next() {
		var day string
		var minor, decimals int64
		var count int
		var rate decimal.Decimal
		if err := rows.Scan(&day, &minor, &count, &decimals, &rate); err != nil {
			return nil
		}
		i, ok := idx[day]
		if !ok {
			i = len(out)
			idx[day] = i
			out = append(out, DayTotal{Day: day})
		}
		out[i].Count += count
		sums[day] = sums[day].Add(toBase(minor, decimals, rate))
	}
	for i := range out {
		out[i].Total = money.FromDecimal(sums[out[i].Day], base)
	}
	return out
}

type PeriodTotal struct {
	Key   string
	Total money.Money
}

// GroupedByPeriod is the total per period (see PeriodExpr) in the base
// currency.
func GroupedByPeriod(ctx context.Context, sqlDB *sql.DB, w ReportWhere, groupBy string, base money.Unit) []PeriodTotal {
	where, args := Clause(w)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT `+PeriodExpr(groupBy)+` as period_date, SUM(t.amount), c.decimals, c.rate
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		`+where+` GROUP BY period_date, c.id`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []PeriodTotal
	sums := map[string]decimal.Decimal{}
	idx := map[string]int{}
	for rows.Next() {
		var key string
		var minor, decimals int64
		var rate decimal.Decimal
		if err := rows.Scan(&key, &minor, &decimals, &rate); err != nil {
			return nil
		}
		if _, ok := idx[key]; !ok {
			idx[key] = len(out)
			out = append(out, PeriodTotal{Key: key})
		}
		sums[key] = sums[key].Add(toBase(minor, decimals, rate))
	}
	for i := range out {
		out[i].Total = money.FromDecimal(sums[out[i].Key], base)
	}
	return out
}

// TopRow is one transaction; Amount is converted to the base currency.
type TopRow struct {
	ID          int64
	Description sql.NullString
	Date        sql.NullString
	Amount      money.Money
	CatID       sql.NullInt64
	CatName     sql.NullString
	Icon        sql.NullString
	Color       sql.NullString
	AccID       int64
	AccName     sql.NullString
}

// TopTransactions are the largest transactions by value in the base currency.
func TopTransactions(ctx context.Context, sqlDB *sql.DB, w ReportWhere, limit int, base money.Unit) []TopRow {
	where, args := Clause(w)
	args = append(args, limit)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT t.id, t.description, t.date, t.amount, c.decimals, c.rate,
			cat.id, cat.name, cat.icon, cat.color, a.id, a.name
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		LEFT JOIN categories cat ON cat.id = t.category_id
		`+where+` ORDER BY t.amount * CAST(c.rate AS REAL) / pow(10, c.decimals) DESC LIMIT ?`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []TopRow
	for rows.Next() {
		var x TopRow
		var minor, decimals int64
		var rate decimal.Decimal
		if err := rows.Scan(&x.ID, &x.Description, &x.Date, &minor, &decimals, &rate, &x.CatID, &x.CatName, &x.Icon, &x.Color, &x.AccID, &x.AccName); err != nil {
			continue
		}
		x.Amount = money.FromDecimal(toBase(minor, decimals, rate), base)
		out = append(out, x)
	}
	return out
}

// BudgetAmount is a budget limit in the budget's own currency. Rate is that
// currency's rate to the base currency, for callers that fold budgets together.
type BudgetAmount struct {
	Amount money.Money
	Rate   decimal.Decimal
	IsBase bool
}

func ScopedMonthlyBudgets(ctx context.Context, sqlDB *sql.DB, categoryIDs, tagIDs []int64) []BudgetAmount {
	q := `SELECT b.amount, c.rate, c.is_base, c.id, c.decimals FROM budgets b
		JOIN currencies c ON c.id = b.currency_id
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
		var minor, base, currencyID, decimals int64
		var rate decimal.Decimal
		if err := rows.Scan(&minor, &rate, &base, &currencyID, &decimals); err != nil {
			return nil
		}
		unit := money.Unit{ID: currencyID, Decimals: int(decimals)}
		out = append(out, BudgetAmount{Amount: money.New(minor, unit), Rate: rate, IsBase: base != 0})
	}
	return out
}

// BudgetSpent sums confirmed expenses in [start,end], optionally scoped by
// category/tag IN lists, expressed in the currency `in` whose rate to the base
// currency is inRate (1 when it is the base). Variable-length IN cannot be a
// single sqlc query.
func BudgetSpent(ctx context.Context, sqlDB *sql.DB, start, end string, categoryIDs, tagIDs []int64, in money.Unit, inRate decimal.Decimal) (money.Money, error) {
	zero := money.Zero(in)
	q := `
		SELECT COALESCE(SUM(t.amount), 0), c.decimals, c.rate
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		WHERE t.status = 'confirmed' AND t.type = 'expense'
			AND t.date >= ? AND t.date <= ?`
	args := []any{start, end}
	if categoryIDs != nil {
		if len(categoryIDs) == 0 {
			return zero, nil
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
	q += ` GROUP BY c.id`
	rows, err := sqlDB.QueryContext(ctx, q, args...)
	if err != nil {
		return zero, err
	}
	defer rows.Close()
	total := decimal.Zero
	for rows.Next() {
		var minor, decimals int64
		var rate decimal.Decimal
		if err := rows.Scan(&minor, &decimals, &rate); err != nil {
			return zero, err
		}
		total = total.Add(toBase(minor, decimals, rate))
	}
	if err := rows.Err(); err != nil {
		return zero, err
	}
	return money.FromDecimal(total.DivRound(inRate, money.RateDivPrecision), in), nil
}

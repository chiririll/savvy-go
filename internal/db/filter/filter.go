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
	"savvy-go/internal/store"
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
	AccountIDs  []int64
	Currencies  []string // codes; a transfer matches when either side is one
	CategoryID  int64
	CategoryIDs []int64
	TagIDs      []int64
	Status      string
	StartDate   string
	EndDate     string
	// Search is free text: space-separated terms, all of which must match
	// (see searchWhere).
	Search  string
	SortBy  string
	SortDir string
}

func txWhere(f TxFilter) (string, []any) {
	q := `WHERE 1=1`
	var args []any
	if f.Type != "" {
		q += ` AND t.type = ?`
		args = append(args, f.Type)
	}
	if f.AccountID > 0 {
		q += ` AND EXISTS (SELECT 1 FROM payments p WHERE p.transaction_id = t.id AND p.account_id = ?)`
		args = append(args, f.AccountID)
	}
	if len(f.AccountIDs) > 0 {
		q += ` AND EXISTS (SELECT 1 FROM payments p WHERE p.transaction_id = t.id AND p.account_id IN (` +
			Placeholders(len(f.AccountIDs)) + `))`
		for _, id := range f.AccountIDs {
			args = append(args, id)
		}
	}
	if len(f.Currencies) > 0 {
		ph := Placeholders(len(f.Currencies))
		// currencies.code compares without regard to case. A transaction is
		// in a currency when it moved money in it or has a line in it.
		q += ` AND (EXISTS (SELECT 1 FROM payments p
				JOIN accounts a ON a.id = p.account_id
				JOIN currencies c ON c.id = a.currency_id
				WHERE p.transaction_id = t.id AND c.code IN (` + ph + `))
			OR EXISTS (SELECT 1 FROM transaction_lines l
				JOIN currencies c ON c.id = l.currency_id
				WHERE l.transaction_id = t.id AND c.code IN (` + ph + `)))`
		for range 2 {
			for _, code := range f.Currencies {
				args = append(args, code)
			}
		}
	}
	if f.CategoryID > 0 {
		q += ` AND EXISTS (SELECT 1 FROM transaction_lines l WHERE l.transaction_id = t.id AND l.category_id = ?)`
		args = append(args, f.CategoryID)
	}
	if len(f.CategoryIDs) > 0 {
		q += ` AND EXISTS (SELECT 1 FROM transaction_lines l WHERE l.transaction_id = t.id AND l.category_id IN (` +
			Placeholders(len(f.CategoryIDs)) + `))`
		for _, id := range f.CategoryIDs {
			args = append(args, id)
		}
	}
	if len(f.TagIDs) > 0 {
		q += ` AND EXISTS (SELECT 1 FROM transaction_tag tt WHERE tt.transaction_id = t.id AND tt.tag_id IN (` + Placeholders(
			len(f.TagIDs),
		) + `))`
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
	sq, sargs := searchWhere(f.Search)
	return q + sq, append(args, sargs...)
}

// txSize is how much money a transaction moved, in the base currency: the
// larger of what came in and what went out, so a transfer counts once and a
// transfer fee is not added to it.
const txSize = `(SELECT MAX(
		COALESCE(SUM(CASE WHEN p.amount > 0 THEN p.amount * COALESCE(NULLIF(CAST(c.rate AS REAL), 0), 1) / pow(10, c.decimals) END), 0),
		COALESCE(-SUM(CASE WHEN p.amount < 0 THEN p.amount * COALESCE(NULLIF(CAST(c.rate AS REAL), 0), 1) / pow(10, c.decimals) END), 0))
	FROM payments p
	JOIN accounts a ON a.id = p.account_id
	JOIN currencies c ON c.id = a.currency_id
	WHERE p.transaction_id = t.id)`

// txOrderBy maps whitelisted sort keys to SQL; request strings never reach the query.
func txOrderBy(sortBy, sortDir string) string {
	dir := "DESC"
	if sortDir == "asc" {
		dir = "ASC"
	}
	switch sortBy {
	case "amount":
		return txSize + ` ` + dir + `, t.id ` + dir
	case "created_at":
		return `t.created_at ` + dir + `, t.id ` + dir
	default:
		// Undated last in both directions, in the shape of the indexes in
		// migrate/sql/space/0001_base: DESC puts NULL last by itself, ASC needs the
		// (date IS NULL) key first.
		if dir == "DESC" {
			return `t.date DESC NULLS LAST, COALESCE(t.day_order, t.id) DESC, t.id DESC`
		}
		return `(t.date IS NULL), t.date ASC, COALESCE(t.day_order, t.id) ASC, t.id ASC`
	}
}

func CountTransactions(ctx context.Context, sqlDB store.DB, f TxFilter) (int64, error) {
	where, args := txWhere(f)
	var n int64
	err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions t `+where, args...).Scan(&n)
	return n, err
}

// SummaryRow is one income or expense with what folds it into the base currency.
type SummaryRow struct {
	Type     string
	Minor    int64
	Decimals int64
	Rate     decimal.Decimal
	IsBase   bool
}

// SummaryRows lists the lines of the income and expenses in a status that
// match f. The Status of f is replaced by status, since a summary covers one
// status; a Type of f still narrows it, so "transfer" leaves nothing. Lines,
// never payments: a transfer fee is a line of an expense, a change is not income.
func SummaryRows(ctx context.Context, sqlDB store.DB, f TxFilter, status string) ([]SummaryRow, error) {
	f.Status = status
	where, args := txWhere(f)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT t.type, l.amount, c.rate, c.is_base, c.decimals
		FROM transactions t
		JOIN transaction_lines l ON l.transaction_id = t.id
		JOIN currencies c ON c.id = l.currency_id
		`+where+` AND t.type IN ('income', 'expense')`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SummaryRow{}
	for rows.Next() {
		var r SummaryRow
		var isBase int64
		if err := rows.Scan(&r.Type, &r.Minor, &r.Rate, &isBase, &r.Decimals); err != nil {
			return nil, err
		}
		r.IsBase = isBase != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

func ListTransactions(
	ctx context.Context,
	sqlDB store.DB,
	f TxFilter,
	limit, offset int,
) ([]sqlc.GetTransactionRow, error) {
	where, args := txWhere(f)
	args = append(args, limit, offset)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT t.id, t.type, t.status, t.description, t.date, t.day_order, t.external_id, t.created_by,
			t.created_at, t.updated_at
		FROM transactions t
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
		if err := rows.Scan(&r.ID, &r.Type, &r.Status, &r.Description, &r.Date, &r.DayOrder, &r.ExternalID,
			&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// A report reads lines, never payments: the money that moved says nothing
// about what it was for, so change is not income and a transfer fee is
// spending. Every line is in its own currency, and is converted at that
// currency's rate, not the account's.
const lineFrom = `FROM transaction_lines l
		JOIN transactions t ON t.id = l.transaction_id
		JOIN currencies c ON c.id = l.currency_id
		LEFT JOIN categories cat ON cat.id = l.category_id`

// lineKind is whether a line is income or expense: its category's type, and
// for an uncategorised line the type of its transaction. A debt transaction
// has no lines, so it never gets here.
const lineKind = `COALESCE(cat.type, CASE WHEN t.type IN ('income', 'expense') THEN t.type END)`

// ReportWhere is the shared report/budget aggregation filter. AccountIDs match
// a transaction with a payment on one of the accounts.
type ReportWhere struct {
	Type        string
	Start       string
	End         string
	CategoryID  int64
	AccountIDs  []int64
	CategoryIDs []int64
	TagIDs      []int64
}

// Clause builds `WHERE …` plus args for the lines of confirmed transactions in
// a range, over lineFrom.
func Clause(w ReportWhere) (string, []any) {
	q := `WHERE t.status = 'confirmed' AND ` + lineKind + ` = ? AND t.date >= ? AND t.date <= ?`
	args := []any{w.Type, w.Start, w.End}
	if w.CategoryID > 0 {
		q += ` AND l.category_id = ?`
		args = append(args, w.CategoryID)
	}
	if len(w.AccountIDs) > 0 {
		q += ` AND EXISTS (SELECT 1 FROM payments p WHERE p.transaction_id = t.id AND p.account_id IN (` +
			Placeholders(len(w.AccountIDs)) + `))`
		for _, id := range w.AccountIDs {
			args = append(args, id)
		}
	}
	if len(w.CategoryIDs) > 0 {
		q += ` AND l.category_id IN (` + Placeholders(len(w.CategoryIDs)) + `)`
		for _, id := range w.CategoryIDs {
			args = append(args, id)
		}
	}
	if len(w.TagIDs) > 0 {
		q += ` AND EXISTS (SELECT 1 FROM transaction_tag tt WHERE tt.transaction_id = t.id AND tt.tag_id IN (` + Placeholders(
			len(w.TagIDs),
		) + `))`
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

// SumByType is the total of the matching lines in the base currency.
func SumByType(ctx context.Context, sqlDB store.DB, w ReportWhere, base money.Unit) money.Money {
	where, args := Clause(w)
	total := decimal.Zero
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT COALESCE(SUM(l.amount), 0), c.decimals, c.rate
		`+lineFrom+`
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
// first. Uncategorised lines are left out.
func SumGroupedByCategory(ctx context.Context, sqlDB store.DB, w ReportWhere, base money.Unit) []CatTotal {
	where, args := Clause(w)
	where += ` AND l.category_id IS NOT NULL`
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT cat.id, cat.name, cat.icon, cat.color, SUM(l.amount), c.decimals, c.rate
		`+lineFrom+`
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
// currency. A transaction counts once however many lines or currencies it has,
// so the count is a query of its own.
func DailyTotals(ctx context.Context, sqlDB store.DB, w ReportWhere, base money.Unit) []DayTotal {
	where, args := Clause(w)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT DATE(t.date), SUM(l.amount), c.decimals, c.rate
		`+lineFrom+`
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
		var rate decimal.Decimal
		if err := rows.Scan(&day, &minor, &decimals, &rate); err != nil {
			return nil
		}
		if _, ok := idx[day]; !ok {
			idx[day] = len(out)
			out = append(out, DayTotal{Day: day})
		}
		sums[day] = sums[day].Add(toBase(minor, decimals, rate))
	}
	if rows.Err() != nil {
		return nil
	}
	for i := range out {
		out[i].Total = money.FromDecimal(sums[out[i].Day], base)
	}
	counts, err := sqlDB.QueryContext(ctx, `
		SELECT DATE(t.date), COUNT(DISTINCT t.id)
		`+lineFrom+`
		`+where+` GROUP BY DATE(t.date)`, args...)
	if err != nil {
		return nil
	}
	defer counts.Close()
	for counts.Next() {
		var day string
		var n int
		if err := counts.Scan(&day, &n); err != nil {
			return nil
		}
		if i, ok := idx[day]; ok {
			out[i].Count = n
		}
	}
	return out
}

type PeriodTotal struct {
	Key   string
	Total money.Money
}

// GroupedByPeriod is the total per period (see PeriodExpr) in the base
// currency.
func GroupedByPeriod(
	ctx context.Context,
	sqlDB store.DB,
	w ReportWhere,
	groupBy string,
	base money.Unit,
) []PeriodTotal {
	where, args := Clause(w)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT `+PeriodExpr(groupBy)+` as period_date, SUM(l.amount), c.decimals, c.rate
		`+lineFrom+`
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

// TopRow is one transaction; Amount is what its matching lines add up to,
// converted to the base currency. The category is that of its largest line and
// the account that of its largest payment on the side of the money that moved
// (out for an expense, in for income); AccID is 0 when it has none.
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
	// CatCount and AccCount are how many distinct categories the lines and
	// how many accounts the moving payments touch. CatID and AccID are the
	// largest of each.
	CatCount int
	AccCount int
}

// TopTransactions are the largest transactions by value in the base currency.
// The transactions are picked by an approximate value in SQL, then their lines
// and payments are read to total them exactly.
func TopTransactions(ctx context.Context, sqlDB store.DB, w ReportWhere, limit int, base money.Unit) []TopRow {
	where, args := Clause(w)
	pickArgs := append(append([]any{}, args...), limit)
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT t.id, t.description, t.date
		`+lineFrom+`
		`+where+`
		GROUP BY t.id
		ORDER BY SUM(l.amount * COALESCE(NULLIF(CAST(c.rate AS REAL), 0), 1) / pow(10, c.decimals)) DESC, t.id DESC
		LIMIT ?`, pickArgs...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []TopRow
	at := map[int64]int{}
	var ids []any
	for rows.Next() {
		var x TopRow
		if err := rows.Scan(&x.ID, &x.Description, &x.Date); err != nil {
			return nil
		}
		at[x.ID] = len(out)
		out = append(out, x)
		ids = append(ids, x.ID)
	}
	if rows.Err() != nil || len(out) == 0 {
		return out
	}
	in := Placeholders(len(ids))

	lines, err := sqlDB.QueryContext(ctx, `
		SELECT t.id, l.amount, c.decimals, c.rate, cat.id, cat.name, cat.icon, cat.color
		`+lineFrom+`
		`+where+` AND t.id IN (`+in+`)
		ORDER BY t.id, l.seq`, append(append([]any{}, args...), ids...)...)
	if err != nil {
		return nil
	}
	defer lines.Close()
	totals := map[int64]decimal.Decimal{}
	largest := map[int64]decimal.Decimal{}
	cats := map[int64]map[int64]bool{}
	for lines.Next() {
		var id, minor, decimals int64
		var rate decimal.Decimal
		var catID sql.NullInt64
		var name, icon, color sql.NullString
		if err := lines.Scan(&id, &minor, &decimals, &rate, &catID, &name, &icon, &color); err != nil {
			return nil
		}
		v := toBase(minor, decimals, rate)
		totals[id] = totals[id].Add(v)
		x := &out[at[id]]
		if catID.Valid {
			if cats[id] == nil {
				cats[id] = map[int64]bool{}
			}
			cats[id][catID.Int64] = true
		}
		if best, seen := largest[id]; !seen || v.GreaterThan(best) {
			largest[id] = v
			x.CatID, x.CatName, x.Icon, x.Color = catID, name, icon, color
		}
	}
	if lines.Err() != nil {
		return nil
	}

	// An expense moved money out of its accounts, income into them.
	side := `p.amount < 0`
	if w.Type == "income" {
		side = `p.amount > 0`
	}
	pays, err := sqlDB.QueryContext(ctx, `
		SELECT p.transaction_id, p.amount, c.decimals, c.rate, a.id, a.name
		FROM payments p
		JOIN accounts a ON a.id = p.account_id
		JOIN currencies c ON c.id = a.currency_id
		WHERE `+side+` AND p.transaction_id IN (`+in+`)
		ORDER BY p.transaction_id, a.id`, ids...)
	if err != nil {
		return nil
	}
	defer pays.Close()
	moved := map[int64]decimal.Decimal{}
	accs := map[int64]map[int64]bool{}
	for pays.Next() {
		var id, minor, decimals, accID int64
		var rate decimal.Decimal
		var name sql.NullString
		if err := pays.Scan(&id, &minor, &decimals, &rate, &accID, &name); err != nil {
			return nil
		}
		v := toBase(minor, decimals, rate).Abs()
		x := &out[at[id]]
		if accs[id] == nil {
			accs[id] = map[int64]bool{}
		}
		accs[id][accID] = true
		if best, seen := moved[id]; !seen || v.GreaterThan(best) {
			moved[id] = v
			x.AccID, x.AccName = accID, name
		}
	}
	if pays.Err() != nil {
		return nil
	}

	for i := range out {
		out[i].Amount = money.FromDecimal(totals[out[i].ID], base)
		out[i].CatCount = len(cats[out[i].ID])
		out[i].AccCount = len(accs[out[i].ID])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Amount.Cmp(out[j].Amount) > 0 })
	return out
}

// BudgetAmount is a budget limit in the budget's own currency. Rate is that
// currency's rate to the base currency, for callers that fold budgets together.
type BudgetAmount struct {
	Amount money.Money
	Rate   decimal.Decimal
	IsBase bool
}

func ScopedMonthlyBudgets(ctx context.Context, sqlDB store.DB, categoryIDs, tagIDs []int64) []BudgetAmount {
	q := `SELECT b.amount, c.rate, c.is_base, c.id, c.decimals FROM budgets b
		JOIN currencies c ON c.id = b.currency_id
		WHERE b.is_active = 1 AND b.period = 'monthly' AND b.is_global = 0 AND (`
	var args []any
	parts := []string{}
	if len(categoryIDs) > 0 {
		parts = append(
			parts,
			`EXISTS (SELECT 1 FROM budget_category bc WHERE bc.budget_id = b.id AND bc.category_id IN (`+Placeholders(
				len(categoryIDs),
			)+`))`,
		)
		for _, id := range categoryIDs {
			args = append(args, id)
		}
	}
	if len(tagIDs) > 0 {
		parts = append(
			parts,
			`EXISTS (SELECT 1 FROM budget_tag bt WHERE bt.budget_id = b.id AND bt.tag_id IN (`+Placeholders(len(tagIDs))+`))`,
		)
		for _, id := range tagIDs {
			args = append(args, id)
		}
	}
	q += strings.Join(parts, " OR ") + `)`
	return scanBudgetAmounts(ctx, sqlDB, q, args)
}

func scanBudgetAmounts(ctx context.Context, sqlDB store.DB, q string, args []any) []BudgetAmount {
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

// BudgetSpent sums what the lines of confirmed expenses in [start,end] add up
// to (a transfer fee is one), optionally scoped by category/tag IN lists,
// expressed in the currency `in` whose rate to the base currency is inRate (1
// when it is the base). Variable-length IN cannot be a single sqlc query.
func BudgetSpent(
	ctx context.Context,
	sqlDB store.DB,
	start, end string,
	categoryIDs, tagIDs []int64,
	in money.Unit,
	inRate decimal.Decimal,
) (money.Money, error) {
	zero := money.Zero(in)
	q := `
		SELECT COALESCE(SUM(l.amount), 0), c.decimals, c.rate
		` + lineFrom + `
		WHERE t.status = 'confirmed' AND ` + lineKind + ` = 'expense'
			AND t.date >= ? AND t.date <= ?`
	args := []any{start, end}
	if categoryIDs != nil {
		if len(categoryIDs) == 0 {
			return zero, nil
		}
		q += ` AND l.category_id IN (` + Placeholders(len(categoryIDs)) + `)`
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

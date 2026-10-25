package legacy

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
)

// Laravel stored money as major-unit numbers; the Go schema stores integer
// minor units scaled by the owning currency decimals. scales holds the lookups
// needed to convert a row.
type scales struct {
	currencyDec map[int64]int
	accountCur  map[int64]int64
	txAccount   map[int64]int64
	base        int64
}

func loadScales(ctx context.Context, q querier) (*scales, error) {
	s := &scales{currencyDec: map[int64]int{}, accountCur: map[int64]int64{}, txAccount: map[int64]int64{}}
	pairs := []struct {
		query string
		each  func(a, b int64)
	}{
		{`SELECT id, decimals FROM currencies ORDER BY id`, func(id, dec int64) {
			s.currencyDec[id] = int(dec)
			if s.base == 0 {
				s.base = id
			}
		}},
		{`SELECT id, currency_id FROM accounts`, func(id, cur int64) { s.accountCur[id] = cur }},
		{`SELECT id, account_id FROM transactions`, func(id, acc int64) { s.txAccount[id] = acc }},
	}
	for _, p := range pairs {
		if err := scanPairs(ctx, q, p.query, p.each); err != nil {
			if strings.Contains(err.Error(), "no such table") {
				continue
			}
			return nil, err
		}
	}
	// is_base only exists on newer Laravel schemas; keep the lowest id otherwise.
	_ = scanPairs(
		ctx,
		q,
		`SELECT id, is_base FROM currencies WHERE is_base = 1 ORDER BY id LIMIT 1`,
		func(id, _ int64) { s.base = id },
	)
	return s, nil
}

func scanPairs(ctx context.Context, q querier, query string, each func(a, b int64)) error {
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var a, b int64
		if err := rows.Scan(&a, &b); err != nil {
			return err
		}
		each(a, b)
	}
	return rows.Err()
}

func (s *scales) decimals(currencyID int64) int {
	if d, ok := s.currencyDec[currencyID]; ok {
		return d
	}
	return 2
}

func (s *scales) accountDec(accountID int64) int {
	return s.decimals(s.accountCur[accountID])
}

func toDecimal(v any) (decimal.Decimal, error) {
	switch n := v.(type) {
	case int64:
		return decimal.NewFromInt(n), nil
	case float64:
		return decimal.NewFromFloat(n), nil
	case string:
		return decimal.NewFromString(strings.TrimSpace(n))
	case []byte:
		return decimal.NewFromString(strings.TrimSpace(string(n)))
	}
	return decimal.Zero, fmt.Errorf("unsupported value type %T", v)
}

func asInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case float64:
		return int64(n), true
	}
	return 0, false
}

// fixMoney converts the money columns of one row (cols and vals aligned) from
// major units to minor units in place. Budgets without a currency get the base
// currency, which the Go schema requires.
func (s *scales) fixMoney(table string, cols []string, vals []any) error {
	idx := make(map[string]int, len(cols))
	for i, c := range cols {
		idx[c] = i
	}
	row := moneyRow{table: table, idx: idx, vals: vals}
	switch table {
	case "accounts":
		dec := s.decimals(row.get("currency_id"))
		return row.convAll(dec, "initial_balance", "target_amount")
	case "transactions", "recurring_transactions":
		dec := s.accountDec(row.get("account_id"))
		toDec := dec
		if to := row.get("to_account_id"); to > 0 {
			toDec = s.accountDec(to)
		}
		if err := row.conv("amount", dec); err != nil {
			return err
		}
		return row.conv("to_amount", toDec)
	case "transaction_items":
		dec := s.accountDec(s.txAccount[row.get("transaction_id")])
		return row.convAll(dec, "price_per_unit", "total_price")
	case "budgets":
		cid := row.get("currency_id")
		if cid == 0 {
			cid = s.base
		}
		if i, ok := idx["currency_id"]; ok {
			vals[i] = cid
		}
		return row.conv("amount", s.decimals(cid))
	}
	return nil
}

// moneyRow is one row (columns by name in idx, values in vals) whose money
// columns are converted in place.
type moneyRow struct {
	table string
	idx   map[string]int
	vals  []any
}

func (r moneyRow) get(name string) int64 {
	i, ok := r.idx[name]
	if !ok {
		return 0
	}
	n, _ := asInt(r.vals[i])
	return n
}

// conv turns one column from major to minor units; a missing or NULL column
// is left alone.
func (r moneyRow) conv(name string, dec int) error {
	i, ok := r.idx[name]
	if !ok || r.vals[i] == nil {
		return nil
	}
	d, err := toDecimal(r.vals[i])
	if err != nil {
		return fmt.Errorf("%s.%s: %w", r.table, name, err)
	}
	m, err := money.FromInput(d, money.Unit{Decimals: dec}) // only the scale matters here
	if err != nil {
		return fmt.Errorf("%s.%s = %s: %w", r.table, name, d, err)
	}
	r.vals[i] = m.Minor()
	return nil
}

func (r moneyRow) convAll(dec int, names ...string) error {
	for _, name := range names {
		if err := r.conv(name, dec); err != nil {
			return err
		}
	}
	return nil
}

// moneyCol is a money column to swap for a real INTEGER column.
type moneyCol struct {
	name    string
	notNull bool
}

// inPlaceMoney lists, per table, the columns read to scale a row and the
// money columns converted. budgets.currency_id is also backfilled.
var inPlaceMoney = []struct {
	table string
	read  []string
	write []moneyCol
}{
	{"accounts", []string{"currency_id", "initial_balance", "target_amount"},
		[]moneyCol{{"initial_balance", true}, {"target_amount", false}}},
	{"transactions", []string{"account_id", "to_account_id", "amount", "to_amount"},
		[]moneyCol{{"amount", true}, {"to_amount", false}}},
	{"recurring_transactions", []string{"account_id", "to_account_id", "amount", "to_amount"},
		[]moneyCol{{"amount", true}, {"to_amount", false}}},
	{"transaction_items", []string{"transaction_id", "price_per_unit", "total_price"},
		[]moneyCol{{"price_per_unit", true}, {"total_price", true}}},
	{"budgets", []string{"currency_id", "amount"}, []moneyCol{{"amount", true}}},
}

// convertMoneyInPlace converts the money columns of a Laravel-era database that
// is upgraded without copying: values become integer minor units and each
// column is swapped for a real INTEGER column, so later code can rely on
// integer storage (Laravel columns are REAL/NUMERIC and read back as floats).
// It avoids a table rebuild, which would cascade through foreign keys. One
// transaction; upgradeInPlace calls it once; Upgrade skips stamped databases.
func convertMoneyInPlace(ctx context.Context, db *sql.DB) error {
	plans, err := planMoney(ctx, db)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	sc, err := loadScales(ctx, tx)
	if err != nil {
		return err
	}
	for _, p := range plans {
		if err := p.convert(ctx, tx, sc); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Indexes on the swapped columns were dropped; recreate every declared index.
	return ensureIndexes(ctx, db)
}

// moneyPlan is what convertMoneyInPlace does to one table that has money.
type moneyPlan struct {
	table string
	read  []string
	write []moneyCol
	fill  bool
}

// planMoney keeps, per table, only the columns that exist in this database.
func planMoney(ctx context.Context, db *sql.DB) ([]moneyPlan, error) {
	var plans []moneyPlan
	for _, spec := range inPlaceMoney {
		have, err := columns(ctx, db, spec.table)
		if err != nil {
			return nil, err
		}
		p := moneyPlan{
			table: spec.table,
			read:  intersect(spec.read, have),
			fill:  spec.table == "budgets" && slices.Contains(have, "currency_id"),
		}
		for _, c := range spec.write {
			if slices.Contains(have, c.name) {
				p.write = append(p.write, c)
			}
		}
		if len(p.write) > 0 {
			plans = append(plans, p)
		}
	}
	return plans, nil
}

func (p moneyPlan) convert(ctx context.Context, tx *sql.Tx, sc *scales) error {
	cols := append([]string{"id"}, p.read...)
	all, err := p.readRows(ctx, tx, cols)
	if err != nil {
		return err
	}
	if err := p.addMinorColumns(ctx, tx); err != nil {
		return err
	}
	if err := p.rewriteRows(ctx, tx, sc, cols, all); err != nil {
		return err
	}
	return p.swapColumns(ctx, tx)
}

func (p moneyPlan) readRows(ctx context.Context, tx *sql.Tx, cols []string) ([][]any, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s", strings.Join(quoteAll(cols), ", "), p.table))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var all [][]any
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		all = append(all, raw)
	}
	return all, rows.Err()
}

func (p moneyPlan) addMinorColumns(ctx context.Context, tx *sql.Tx) error {
	for _, c := range p.write {
		ddl := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN "%s__minor" INTEGER`, p.table, c.name)
		if c.notNull {
			ddl += " NOT NULL DEFAULT 0"
		}
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("%s: %w", ddl, err)
		}
	}
	return nil
}

func (p moneyPlan) rewriteRows(ctx context.Context, tx *sql.Tx, sc *scales, cols []string, all [][]any) error {
	sets := make([]string, 0, len(p.write)+1)
	for _, c := range p.write {
		sets = append(sets, fmt.Sprintf(`"%s__minor" = ?`, c.name))
	}
	if p.fill {
		sets = append(sets, `"currency_id" = ?`)
	}
	update := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", p.table, strings.Join(sets, ", "))
	for _, raw := range all {
		if err := sc.fixMoney(p.table, cols, raw); err != nil {
			return err
		}
		byName := make(map[string]any, len(cols))
		for i, c := range cols {
			byName[c] = raw[i]
		}
		args := make([]any, 0, len(sets)+1)
		for _, c := range p.write {
			v := byName[c.name]
			if v == nil && c.notNull {
				v = int64(0)
			}
			args = append(args, v)
		}
		if p.fill {
			args = append(args, byName["currency_id"])
		}
		args = append(args, byName["id"])
		if _, err := tx.ExecContext(ctx, update, args...); err != nil {
			return fmt.Errorf("%s: %w", p.table, err)
		}
	}
	return nil
}

// swapColumns drops each old column and renames its __minor twin into place.
func (p moneyPlan) swapColumns(ctx context.Context, tx *sql.Tx) error {
	old := make([]string, len(p.write))
	for i, c := range p.write {
		old[i] = c.name
	}
	if err := dropIndexesOn(ctx, tx, p.table, old); err != nil {
		return err
	}
	for _, c := range p.write {
		for _, ddl := range []string{
			fmt.Sprintf(`ALTER TABLE %s DROP COLUMN "%s"`, p.table, c.name),
			fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN "%s__minor" TO "%s"`, p.table, c.name, c.name),
		} {
			if _, err := tx.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("%s: %w", ddl, err)
			}
		}
	}
	return nil
}

// dropIndexesOn drops the CREATE INDEX indexes of table that include any of cols.
func dropIndexesOn(ctx context.Context, tx *sql.Tx, table string, cols []string) error {
	rows, err := tx.QueryContext(ctx, `PRAGMA index_list(`+table+`)`)
	if err != nil {
		return err
	}
	var names []string
	for rows.Next() {
		var seq, unique, partial int
		var name, origin string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			_ = rows.Close()
			return err
		}
		if origin == "c" {
			names = append(names, name)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_ = rows.Close()
	for _, name := range names {
		info, err := tx.QueryContext(ctx, `PRAGMA index_info(`+quoteIdent(name)+`)`)
		if err != nil {
			return err
		}
		hit := false
		for info.Next() {
			var seqno, cid int
			var col sql.NullString
			if err := info.Scan(&seqno, &cid, &col); err != nil {
				_ = info.Close()
				return err
			}
			if col.Valid && slices.Contains(cols, col.String) {
				hit = true
			}
		}
		if err := info.Err(); err != nil {
			return err
		}
		_ = info.Close()
		if hit {
			if _, err := tx.ExecContext(ctx, `DROP INDEX `+quoteIdent(name)); err != nil {
				return err
			}
		}
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

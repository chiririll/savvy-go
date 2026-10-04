package legacy

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/shopspring/decimal"

	"savvy-go/internal/migrate"
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

func loadScales(ctx context.Context, q migrate.Querier) (*scales, error) {
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
	_ = scanPairs(ctx, q, `SELECT id, is_base FROM currencies WHERE is_base = 1 ORDER BY id LIMIT 1`, func(id, _ int64) { s.base = id })
	return s, nil
}

func scanPairs(ctx context.Context, q migrate.Querier, query string, each func(a, b int64)) error {
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
	get := func(name string) int64 {
		i, ok := idx[name]
		if !ok {
			return 0
		}
		n, _ := asInt(vals[i])
		return n
	}
	conv := func(name string, dec int) error {
		i, ok := idx[name]
		if !ok || vals[i] == nil {
			return nil
		}
		d, err := toDecimal(vals[i])
		if err != nil {
			return fmt.Errorf("%s.%s: %w", table, name, err)
		}
		vals[i] = money.FromDecimal(d, money.Unit{Decimals: dec}).Minor() // only the scale matters here
		return nil
	}
	var errs [3]error
	switch table {
	case "accounts":
		dec := s.decimals(get("currency_id"))
		errs[0], errs[1] = conv("initial_balance", dec), conv("target_amount", dec)
	case "transactions", "recurring_transactions":
		dec := s.accountDec(get("account_id"))
		toDec := dec
		if to := get("to_account_id"); to > 0 {
			toDec = s.accountDec(to)
		}
		errs[0], errs[1] = conv("amount", dec), conv("to_amount", toDec)
	case "transaction_items":
		dec := s.accountDec(s.txAccount[get("transaction_id")])
		errs[0], errs[1] = conv("price_per_unit", dec), conv("total_price", dec)
	case "budgets":
		cid := get("currency_id")
		if cid == 0 {
			cid = s.base
		}
		if i, ok := idx["currency_id"]; ok {
			vals[i] = cid
		}
		errs[0] = conv("amount", s.decimals(cid))
	}
	for _, err := range errs {
		if err != nil {
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
	type plan struct {
		table string
		read  []string
		write []moneyCol
		fill  bool
	}
	var plans []plan
	for _, spec := range inPlaceMoney {
		have, err := migrate.Columns(ctx, db, spec.table)
		if err != nil {
			return err
		}
		p := plan{table: spec.table, read: intersect(spec.read, have), fill: spec.table == "budgets" && slices.Contains(have, "currency_id")}
		for _, c := range spec.write {
			if slices.Contains(have, c.name) {
				p.write = append(p.write, c)
			}
		}
		if len(p.write) > 0 {
			plans = append(plans, p)
		}
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
		cols := append([]string{"id"}, p.read...)
		rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s", strings.Join(quoteAll(cols), ", "), p.table))
		if err != nil {
			return err
		}
		var all [][]any
		for rows.Next() {
			raw := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range raw {
				ptrs[i] = &raw[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				_ = rows.Close()
				return err
			}
			all = append(all, raw)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		_ = rows.Close()

		for _, c := range p.write {
			ddl := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN "%s__minor" INTEGER`, p.table, c.name)
			if c.notNull {
				ddl += " NOT NULL DEFAULT 0"
			}
			if _, err := tx.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("%s: %w", ddl, err)
			}
		}
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
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Indexes on the swapped columns were dropped; recreate every declared index.
	return migrate.EnsureIndexes(ctx, db)
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

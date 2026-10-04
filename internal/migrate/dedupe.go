package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Deduper gives rows that repeat a (scope, name) pair a numbered name, "Food
// (1)", "Food (2)", ... Names compare the way SQLite's NOCASE collation does
// (ASCII letters only), which is what the unique indexes use.
type Deduper struct {
	used  map[string]bool // every key taken, original or generated
	first map[string]bool // keys already claimed by an unrenamed row
}

// NameKey is one existing row: its uniqueness scope (e.g. a category type, or
// "" when names are unique globally) and its name.
type NameKey struct{ Scope, Name string }

// NewDeduper reserves every existing name up front, so a generated name never
// collides with a row that is merely later in the table.
func NewDeduper(existing []NameKey) *Deduper {
	d := &Deduper{used: make(map[string]bool, len(existing)), first: map[string]bool{}}
	for _, e := range existing {
		d.used[nameKey(e.Scope, e.Name)] = true
	}
	return d
}

// Unique returns name unchanged for its first occurrence and a numbered
// variant for every repeat. Call it for each row in a stable (id) order.
func (d *Deduper) Unique(scope, name string) string {
	key := nameKey(scope, name)
	if !d.first[key] {
		d.first[key] = true
		return name
	}
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s (%d)", name, n)
		ck := nameKey(scope, candidate)
		if !d.used[ck] {
			d.used[ck] = true
			d.first[ck] = true
			return candidate
		}
	}
}

func nameKey(scope, name string) string {
	return scope + "\x00" + asciiLower(name)
}

func asciiLower(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, s)
}

// uniqueNames lists the tables whose names the schema keeps unique, with the
// column that scopes the uniqueness ("" for none).
var uniqueNames = []struct{ table, scope string }{
	{"categories", "type"},
	{"tags", ""},
}

// DedupeNames renames repeated category and tag names in place (the lowest id
// keeps its name) so the unique indexes and NOCASE columns can be applied to
// data written before they existed. Missing tables are skipped. It is cheap
// and a no-op on clean data, so every migration run starts with it.
func DedupeNames(ctx context.Context, db *sql.DB) error {
	for _, u := range uniqueNames {
		if !tableExists(ctx, db, u.table) {
			continue
		}
		if err := dedupeTable(ctx, db, u.table, u.scope); err != nil {
			return fmt.Errorf("dedupe %s: %w", u.table, err)
		}
	}
	return nil
}

func dedupeTable(ctx context.Context, db *sql.DB, table, scope string) error {
	scopeExpr := "''"
	if scope != "" {
		scopeExpr = scope
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`SELECT id, %s, name FROM %s ORDER BY id`, scopeExpr, table))
	if err != nil {
		return err
	}
	type row struct {
		id          int64
		scope, name string
	}
	var all []row
	var keys []NameKey
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.scope, &r.name); err != nil {
			_ = rows.Close()
			return err
		}
		all = append(all, r)
		keys = append(keys, NameKey{r.scope, r.name})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()

	d := NewDeduper(keys)
	for _, r := range all {
		if name := d.Unique(r.scope, r.name); name != r.name {
			if _, err := db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET name = ? WHERE id = ?`, table), name, r.id); err != nil {
				return err
			}
		}
	}
	return nil
}

func tableExists(ctx context.Context, db *sql.DB, name string) bool {
	var found string
	err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&found)
	return err == nil
}

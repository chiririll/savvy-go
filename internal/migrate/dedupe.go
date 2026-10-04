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

// nameScopes lists the tables whose names the schema keeps unique, with the
// column that scopes the uniqueness ("" for none).
var nameScopes = map[string]string{
	"categories": "type",
	"tags":       "",
}

// NameScope reports whether table keeps its names unique and which column
// scopes that uniqueness ("" for none).
func NameScope(table string) (scope string, ok bool) {
	scope, ok = nameScopes[table]
	return scope, ok
}

type nameRow struct {
	id int64
	NameKey
}

// nameRows reads every row's (scope, name) of a table from NameScope, in id
// order.
func nameRows(ctx context.Context, q Querier, table string) ([]nameRow, error) {
	scopeExpr := "''"
	if scope := nameScopes[table]; scope != "" {
		scopeExpr = scope
	}
	rows, err := q.QueryContext(ctx, fmt.Sprintf(`SELECT id, %s, name FROM %s ORDER BY id`, scopeExpr, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []nameRow
	for rows.Next() {
		var r nameRow
		if err := rows.Scan(&r.id, &r.Scope, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func keysOf(rows []nameRow) []NameKey {
	keys := make([]NameKey, len(rows))
	for i, r := range rows {
		keys[i] = r.NameKey
	}
	return keys
}

// LoadDeduper returns a Deduper that knows every name already in table.
func LoadDeduper(ctx context.Context, q Querier, table string) (*Deduper, error) {
	rows, err := nameRows(ctx, q, table)
	if err != nil {
		return nil, err
	}
	return NewDeduper(keysOf(rows)), nil
}

// DedupeNames renames repeated category and tag names in place (the lowest id
// keeps its name) so the unique indexes and NOCASE columns can be applied to
// data written before they existed. Missing tables are skipped. It is cheap
// and a no-op on clean data, so every migration run starts with it.
func DedupeNames(ctx context.Context, db *sql.DB) error {
	for table := range nameScopes {
		if !TableExists(ctx, db, table) {
			continue
		}
		if err := dedupeTable(ctx, db, table); err != nil {
			return fmt.Errorf("dedupe %s: %w", table, err)
		}
	}
	return nil
}

func dedupeTable(ctx context.Context, db *sql.DB, table string) error {
	rows, err := nameRows(ctx, db, table)
	if err != nil {
		return err
	}
	d := NewDeduper(keysOf(rows))
	for _, r := range rows {
		if name := d.Unique(r.Scope, r.Name); name != r.Name {
			if _, err := db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET name = ? WHERE id = ?`, table), name, r.id); err != nil {
				return err
			}
		}
	}
	return nil
}

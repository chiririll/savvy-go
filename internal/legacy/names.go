package legacy

import (
	"context"
	"fmt"
	"slices"

	"savvy-go/internal/migrate"
)

// nameFixer renames repeated category and tag names while rows are copied. The
// Go schema keeps them unique ignoring case, and INSERT OR IGNORE would drop a
// repeat while the transactions that point at its id stay behind.
type nameFixer struct {
	name, scope int // column positions in the copied row; scope is -1 for none
	dedupe      *migrate.Deduper
}

// newNameFixer returns nil for tables whose names need no fixing; a nil
// fixer's fix is a no-op.
func newNameFixer(ctx context.Context, src querier, table string, cols []string) (*nameFixer, error) {
	scopeCol := ""
	switch table {
	case "categories":
		scopeCol = "type"
	case "tags":
	default:
		return nil, nil
	}
	f := &nameFixer{name: slices.Index(cols, "name"), scope: -1}
	if scopeCol != "" {
		f.scope = slices.Index(cols, scopeCol)
	}
	if f.name < 0 || (scopeCol != "" && f.scope < 0) {
		return nil, nil
	}
	scopeExpr := "''"
	if scopeCol != "" {
		scopeExpr = scopeCol
	}
	rows, err := src.QueryContext(ctx, fmt.Sprintf("SELECT %s, name FROM %s", scopeExpr, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []migrate.NameKey
	for rows.Next() {
		var k migrate.NameKey
		if err := rows.Scan(&k.Scope, &k.Name); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	f.dedupe = migrate.NewDeduper(keys)
	return f, nil
}

// fix rewrites the name in a scanned row in place.
func (f *nameFixer) fix(raw []any) {
	if f == nil {
		return
	}
	name, ok := raw[f.name].(string)
	if !ok {
		return
	}
	scope := ""
	if f.scope >= 0 {
		scope, _ = raw[f.scope].(string)
	}
	raw[f.name] = f.dedupe.Unique(scope, name)
}

package legacy

import (
	"context"
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
func newNameFixer(ctx context.Context, src migrate.Querier, table string, cols []string) (*nameFixer, error) {
	scopeCol, ok := migrate.NameScope(table)
	if !ok {
		return nil, nil
	}
	f := &nameFixer{name: slices.Index(cols, "name"), scope: -1}
	if scopeCol != "" {
		if f.scope = slices.Index(cols, scopeCol); f.scope < 0 {
			return nil, nil
		}
	}
	if f.name < 0 {
		return nil, nil
	}
	d, err := migrate.LoadDeduper(ctx, src, table)
	if err != nil {
		return nil, err
	}
	f.dedupe = d
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

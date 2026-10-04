package db

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// P35: sqlc queries stay in SQL that Postgres also accepts, so a Postgres
// store needs new migrations, not rewritten queries. Dialect-specific SQL
// (date bucketing, rate maths) is confined to internal/db/filter.
var sqliteOnly = regexp.MustCompile(`(?i)\bINSERT\s+OR\b|\bREPLACE\s+INTO\b|\bIFNULL\s*\(|\bstrftime\s*\(|\bjulianday\s*\(|\bdatetime\s*\(|\bgroup_concat\s*\(|\bprintf\s*\(`)

func TestP35QueriesAvoidSQLiteOnlySyntax(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("queries", "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no query files: %v", err)
	}
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range sqliteOnly.FindAllIndex(body, -1) {
			t.Errorf("%s: SQLite-only syntax %q", f, body[m[0]:m[1]])
		}
	}
}

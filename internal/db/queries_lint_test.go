package db

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func queryFiles(t *testing.T, dir string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("queries", dir, "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no query files in %s: %v", dir, err)
	}
	return files
}

// P35: sqlc queries stay in SQL that Postgres also accepts, so a Postgres
// store needs new migrations, not rewritten queries. Dialect-specific SQL
// (date bucketing, rate maths) is confined to internal/db/filter.
var sqliteOnly = regexp.MustCompile(`(?i)\bINSERT\s+OR\b|\bREPLACE\s+INTO\b|\bIFNULL\s*\(|\bstrftime\s*\(|\bjulianday\s*\(|\bdatetime\s*\(|\bgroup_concat\s*\(|\bprintf\s*\(`)

func TestP35QueriesAvoidSQLiteOnlySyntax(t *testing.T) {
	for _, dir := range []string{"server", "space"} {
		for _, f := range queryFiles(t, dir) {
			body, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range sqliteOnly.FindAllIndex(body, -1) {
				t.Errorf("%s: SQLite-only syntax %q", f, body[m[0]:m[1]])
			}
		}
	}
}

var (
	createTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`)
	tableRef    = regexp.MustCompile(`(?i)\b(?:FROM|JOIN|INTO|UPDATE)\s+(\w+)`)
	sqlComment  = regexp.MustCompile(`--[^\n]*`)
)

func schemaTables(t *testing.T, set string) map[string]bool {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join("..", "migrate", "sql", set, "*.sql"))
	out := map[string]bool{}
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range createTable.FindAllStringSubmatch(string(body), -1) {
			out[strings.ToLower(m[1])] = true
		}
	}
	if len(out) == 0 {
		t.Fatalf("no tables in the %s schema", set)
	}
	return out
}

// Server and space tables live in different databases, so a query must not
// mix them: it would compile (one sqlc package) but fail at run time.
func TestQueriesTouchOnlyTheirDatabase(t *testing.T) {
	sets := map[string]map[string]bool{"server": schemaTables(t, "server"), "space": schemaTables(t, "space")}
	for dir, own := range sets {
		for _, f := range queryFiles(t, dir) {
			body, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			sql := sqlComment.ReplaceAllString(string(body), "")
			for _, m := range tableRef.FindAllStringSubmatch(sql, -1) {
				name := strings.ToLower(m[1])
				if name == "set" { // ON CONFLICT ... DO UPDATE SET
					continue
				}
				if !own[name] {
					t.Errorf("%s: %q is not a %s table", f, m[1], dir)
				}
			}
		}
	}
}

package sqlc

// Not generated: guards the query plans of the hot read paths.

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"savvy-go/internal/migrate"
)

var scanRE = regexp.MustCompile(`^SCAN (\w+)`)

// TestHotQueriesDoNotScanTables runs EXPLAIN QUERY PLAN on the queries that run
// once per row of a page or per request, and fails when one of the listed
// tables is read by a full scan instead of an index search.
func TestHotQueriesDoNotScanTables(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1) // an in-memory database lives per connection
	// Queries of both sets are planned against one scratch file.
	for _, set := range []migrate.Set{migrate.Server, migrate.Space} {
		if err := set.Up(ctx, sqlDB); err != nil {
			t.Fatal(err)
		}
	}

	for _, c := range []struct {
		name  string
		query string
		// tables that must not be scanned; a list query may scan its own table
		searched []string
	}{
		{"GetTransaction", getTransaction, []string{"t", "a", "ca", "ta", "cb"}},
		{"GetAccount", getAccount, []string{"a", "c"}},
		{"GetRecurring", getRecurring, []string{"r", "a", "ca", "ta", "cb"}},
		{"GetCategory", getCategory, []string{"categories", "t"}},
		{"GetTag", getTag, []string{"tags", "tt"}},
		{"ListCategories (count per row)", listCategories, []string{"t"}},
		{"ListTags (count per row)", listTags, []string{"tt"}},
		{"ListItemsOfTransactions", listItemsOfTransactions, []string{"transaction_items"}},
		{"ListTagsOfTransactions", listTagsOfTransactions, []string{"tt", "tags"}},
		{"AccountDailyDeltas", accountDailyDeltas, []string{"t"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			q := strings.ReplaceAll(c.query, "/*SLICE:ids*/?", "?")
			args := make([]any, strings.Count(q, "?"))
			rows, err := sqlDB.QueryContext(ctx, "EXPLAIN QUERY PLAN "+q, args...)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			var plan []string
			for rows.Next() {
				var id, parent, unused int
				var detail string
				if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
					t.Fatal(err)
				}
				plan = append(plan, detail)
			}
			for _, line := range plan {
				m := scanRE.FindStringSubmatch(line)
				if m == nil {
					continue
				}
				for _, table := range c.searched {
					if m[1] == table {
						t.Errorf("%s scans %s:\n  %s", c.name, table, strings.Join(plan, "\n  "))
					}
				}
			}
		})
	}
}

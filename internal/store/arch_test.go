package store_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// P34: business logic never learns how the store keeps its databases. The
// packages below may use the store only through internal/store; only the
// store's SQLite implementation (and the SQLite helpers it uses) know about
// files, PRAGMAs and ATTACH.
var businessPackages = []string{"ledger", "sso", "spaces", "transfers", "uploads", "currencydata", "automation", "recurring", "reports", "calendar", "backup", "importer", "legacy/restore", "httpserver", "auth", "settings", "seed", "signing", "schedule", "jobs"}

var sqliteOnlyImports = []string{
	"savvy-go/internal/store/sqlite",
	"savvy-go/internal/migrate",
	"savvy-go/internal/legacy",
	"modernc.org/sqlite",
	// Nullable columns, "no rows" and timestamps come from internal/db
	// (db.NInt, db.ErrNotFound, db.TimeNS), so only the database packages
	// import database/sql.
	"database/sql",
}

var sqliteOnlySQL = regexp.MustCompile(`(?i)\bPRAGMA\b|\bATTACH\s+DATABASE\b|\bsqlite_master\b|\bsqlite_sequence\b`)

func TestP34BusinessLogicDoesNotKnowTheFiles(t *testing.T) {
	for _, pkg := range businessPackages {
		dir := filepath.Join("..", pkg)
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range parsed.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				for _, banned := range sqliteOnlyImports {
					if path == banned {
						t.Errorf("%s imports %s", f, path)
					}
				}
			}
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if m := sqliteOnlySQL.Find(src); m != nil {
				t.Errorf("%s uses SQLite internals (%s)", f, m)
			}
		}
	}
}

// P57: Laravel is only ever converted by internal/legacy, and the migrations
// know nothing about it: they create and update Go databases and no more.
func TestP57MigrationsKnowNothingOfLaravel(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "migrate", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	laravel := regexp.MustCompile(`(?i)laravel|\blegacy\b`)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if loc := laravel.Find(body); loc != nil {
			t.Errorf("%s mentions %q: Laravel conversion belongs in internal/legacy", f, loc)
		}
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "migrate", "migrate.go"), nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range parsed.Imports {
		if path, _ := strconv.Unquote(imp.Path.Value); path == "savvy-go/internal/legacy" {
			t.Errorf("migrate imports %s", path)
		}
	}
}

// The layering in AGENTS.md ("Packages"): a package never imports one that
// sits above it, and the finance core never touches the network. Both keep
// the core testable without a server and the dependencies pointing one way.
var forbiddenImports = map[string][]string{
	"ledger": {
		"net/http", "savvy-go/internal/httpserver", "savvy-go/internal/backup", "savvy-go/internal/sso",
		"savvy-go/internal/importer", "savvy-go/internal/reports", "savvy-go/internal/recurring",
		"savvy-go/internal/automation", "savvy-go/internal/transfers", "savvy-go/internal/uploads",
		"savvy-go/internal/currencydata", "savvy-go/internal/spaces",
	},
	"spaces":     {"net/http", "savvy-go/internal/ledger", "savvy-go/internal/transfers", "savvy-go/internal/backup"},
	"transfers":  {"net/http", "savvy-go/internal/ledger", "savvy-go/internal/backup"},
	"reports":    {"net/http", "savvy-go/internal/recurring", "savvy-go/internal/automation"},
	"recurring":  {"net/http", "savvy-go/internal/reports", "savvy-go/internal/automation"},
	"automation": {"net/http", "savvy-go/internal/reports", "savvy-go/internal/recurring"},
	"uploads":    {"savvy-go/internal/ledger", "savvy-go/internal/importer"},
	"calendar":   {"savvy-go/internal/ledger", "savvy-go/internal/db"},
}

func TestPackagesRespectTheLayering(t *testing.T) {
	for pkg, banned := range forbiddenImports {
		files, err := filepath.Glob(filepath.Join("..", pkg, "*.go"))
		if err != nil || len(files) == 0 {
			t.Fatalf("%s: %v (%d files)", pkg, err, len(files))
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range parsed.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				for _, b := range banned {
					if path == b {
						t.Errorf("%s imports %s", f, path)
					}
				}
			}
		}
	}
}

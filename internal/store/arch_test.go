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
	// SQL lives in the database packages: internal/db, its generated sqlc
	// code and the repositories internal/db/repo/<area>. Business code asks a
	// repository and sees plain Go values; helpers for nullable columns, "no
	// rows" and timestamps are in internal/db (db.ErrNotFound, db.TimeNS).
	"database/sql",
	"savvy-go/internal/db/sqlc",
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

// A repository package (internal/db/repo/<area>) moves rows for one area and
// knows nothing of the business package above it: the rules sit there, and a
// repository importing it back would be a cycle in spirit if not in Go.
func TestRepositoriesDoNotImportBusinessCode(t *testing.T) {
	repos, err := filepath.Glob(filepath.Join("..", "db", "repo", "*"))
	if err != nil || len(repos) == 0 {
		t.Fatalf("no repositories found: %v", err)
	}
	business := map[string]bool{}
	for _, pkg := range businessPackages {
		business["savvy-go/internal/"+pkg] = true
	}
	for _, dir := range repos {
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range parsed.Imports {
				if path, _ := strconv.Unquote(imp.Path.Value); business[path] {
					t.Errorf("%s imports business package %s", f, path)
				}
			}
		}
	}
}

// A business package reaches rows of another area through that area's
// package, never through its repository. reports and seed read and write
// ledger and automation rows directly by design.
func TestBusinessCodeUsesOnlyItsOwnRepository(t *testing.T) {
	allowed := map[string]bool{"reports": true, "seed": true}
	const prefix = "savvy-go/internal/db/repo/"
	for _, pkg := range businessPackages {
		if allowed[pkg] {
			continue
		}
		files, _ := filepath.Glob(filepath.Join("..", filepath.FromSlash(pkg), "*.go"))
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
				if area, ok := strings.CutPrefix(path, prefix); ok && area != pkg {
					t.Errorf("%s imports the %s repository; use the %s package", f, area, area)
				}
			}
		}
	}
}

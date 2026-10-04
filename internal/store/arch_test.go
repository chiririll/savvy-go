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
var businessPackages = []string{"domain", "httpserver", "auth", "settings", "seed", "signing", "schedule", "jobs"}

var sqliteOnlyImports = []string{
	"savvy-go/internal/store/sqlite",
	"savvy-go/internal/migrate",
	"savvy-go/internal/legacy",
	"modernc.org/sqlite",
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

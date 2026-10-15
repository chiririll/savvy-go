package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// rule is one measured property of the code. A value above soft is worth a
// look in the audit report; a value above hard fails the build.
type rule struct {
	name string
	soft int
	hard int
	what string
}

const (
	ruleFuncLines   = "funcLines"
	ruleNesting     = "nesting"
	ruleComplexity  = "complexity"
	ruleParams      = "params"
	ruleFileLines   = "fileLines"
	ruleLineWidth   = "lineWidth"
	ruleImports     = "imports"
	ruleNameWords   = "nameWords"
	ruleNameJoins   = "nameJoins"
	ruleIgnoreNoWhy = "ignoreWithoutReason"
)

var rules = []rule{
	{ruleFuncLines, 60, 500, "lines in a function"},
	{ruleNesting, 4, 8, "nested blocks in a function"},
	{ruleComplexity, 15, 40, "branches in a function (cyclomatic complexity)"},
	{ruleParams, 5, 10, "parameters of a function"},
	{ruleFileLines, 600, 2000, "lines in a file"},
	{ruleLineWidth, 120, 300, "characters in a line"},
	// Many imports mean a file that knows too much about the rest of the code.
	{ruleImports, 15, 40, "imports in a file"},
	{ruleNameWords, 6, 10, "words in a name"},
	// A name that joins two ideas with And/Or usually does two jobs. Soft
	// only: a hard limit would punish legitimate names like ValidateOrFail.
	{ruleNameJoins, 0, 1 << 30, "And/Or in a name"},
	{ruleIgnoreNoWhy, 0, 0, "archtest:ignore without a reason"},
}

var rulesByName = func() map[string]rule {
	m := make(map[string]rule, len(rules))
	for _, r := range rules {
		m[r.name] = r
	}
	return m
}()

// finding is one measurement above its soft limit.
type finding struct {
	rule  string
	file  string // slash-separated, relative to the repository root
	line  int
	name  string
	value int
}

func (f finding) hard() bool { return f.value > rulesByName[f.rule].hard }

const repoRoot = "../.."

// scanRepo measures every hand-written Go file under internal and cmd.
func scanRepo(t *testing.T) []finding {
	t.Helper()
	var all []finding
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(repoRoot, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
			if err != nil {
				return err
			}
			if ast.IsGenerated(file) {
				return nil
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			all = append(all, analyzeFile(fset, file, filepath.ToSlash(rel), src)...)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].file != all[j].file {
			return all[i].file < all[j].file
		}
		return all[i].line < all[j].line
	})
	return all
}

// analyzeFile returns the findings of one file, with opt-outs applied.
func analyzeFile(fset *token.FileSet, file *ast.File, rel string, src []byte) []finding {
	isTest := strings.HasSuffix(rel, "_test.go")
	var out []finding
	add := func(ruleName string, pos token.Pos, name string, value int, ignored map[string]bool) {
		if value <= rulesByName[ruleName].soft || ignored[ruleName] {
			return
		}
		out = append(out, finding{ruleName, rel, fset.Position(pos).Line, name, value})
	}

	add(ruleFileLines, file.Package, filepath.Base(rel), fset.Position(file.End()).Line, nil)
	add(ruleImports, file.Package, filepath.Base(rel), len(file.Imports), nil)
	tf := fset.File(file.Pos())
	for i, line := range strings.Split(string(src), "\n") {
		if i >= tf.LineCount() {
			break
		}
		add(ruleLineWidth, tf.LineStart(i+1), fmt.Sprintf("line %d", i+1), utf8.RuneCountInString(strings.TrimRight(line, "\r")), nil)
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			ignored, bad := ignores(d.Doc)
			for range bad {
				add(ruleIgnoreNoWhy, d.Pos(), funcName(d), 1, nil)
			}
			name := funcName(d)
			checkName(add, d.Name.Pos(), name, d.Name.Name, isTest, ignored)
			add(ruleParams, d.Pos(), name, paramCount(d.Type), ignored)
			if d.Body != nil {
				add(ruleFuncLines, d.Pos(), name, fset.Position(d.End()).Line-fset.Position(d.Pos()).Line+1, ignored)
				add(ruleNesting, d.Pos(), name, nesting(d.Body), ignored)
				add(ruleComplexity, d.Pos(), name, complexity(d.Body), ignored)
			}
		case *ast.GenDecl:
			ignored, bad := ignores(d.Doc)
			for range bad {
				add(ruleIgnoreNoWhy, d.Pos(), d.Tok.String(), 1, nil)
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					checkName(add, s.Name.Pos(), s.Name.Name, s.Name.Name, false, ignored)
				case *ast.ValueSpec:
					for _, n := range s.Names {
						checkName(add, n.Pos(), n.Name, n.Name, false, ignored)
					}
				}
			}
		}
	}
	return out
}

type addFunc = func(ruleName string, pos token.Pos, name string, value int, ignored map[string]bool)

func checkName(add addFunc, pos token.Pos, label, name string, isTest bool, ignored map[string]bool) {
	words := splitWords(name)
	if isTest && len(words) > 0 && isTestPrefix(words[0]) {
		words = words[1:]
	}
	add(ruleNameWords, pos, label, len(words), ignored)
	joins := 0
	for _, w := range words {
		if w == "And" || w == "Or" {
			joins++
		}
	}
	add(ruleNameJoins, pos, label, joins, ignored)
}

func isTestPrefix(word string) bool {
	switch word {
	case "Test", "Benchmark", "Example", "Fuzz":
		return true
	}
	return false
}

func funcName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}
	return receiverName(d.Recv.List[0].Type) + "." + d.Name.Name
}

func receiverName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return receiverName(e.X)
	case *ast.IndexExpr:
		return receiverName(e.X)
	case *ast.IndexListExpr:
		return receiverName(e.X)
	case *ast.Ident:
		return e.Name
	}
	return "?"
}

func paramCount(ft *ast.FuncType) int {
	n := 0
	for _, f := range ft.Params.List {
		n += max(len(f.Names), 1)
	}
	return n
}

// ignores reads //archtest:ignore <rule> <reason> lines. bad has one entry
// per directive that gives no reason.
func ignores(doc *ast.CommentGroup) (ignored map[string]bool, bad []string) {
	if doc == nil {
		return nil, nil
	}
	ignored = map[string]bool{}
	for _, c := range doc.List {
		rest, ok := strings.CutPrefix(c.Text, "//archtest:ignore ")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			bad = append(bad, rest)
			continue
		}
		ignored[fields[0]] = true
	}
	return ignored, bad
}

// splitWords cuts a Go name into its words: "HTTPServerV2_ok" is HTTP,
// Server, V, 2, ok.
func splitWords(name string) []string {
	runes := []rune(name)
	var words []string
	start := 0
	flush := func(end int) {
		if end > start {
			words = append(words, string(runes[start:end]))
		}
		start = end
	}
	for i := 1; i < len(runes); i++ {
		prev, cur := runes[i-1], runes[i]
		switch {
		case cur == '_':
			flush(i)
			start = i + 1
		case prev == '_':
			// start already moved past the underscore
		case unicode.IsLower(prev) && unicode.IsUpper(cur):
			flush(i)
		case unicode.IsLetter(prev) != unicode.IsLetter(cur) && cur != '_':
			flush(i)
		case unicode.IsUpper(prev) && unicode.IsUpper(cur) && i+1 < len(runes) && unicode.IsLower(runes[i+1]):
			flush(i)
		}
	}
	flush(len(runes))
	return words
}

// nesting is the deepest chain of blocks (if, for, switch, select, closure)
// inside body. An else-if continues its chain and adds no level.
func nesting(body *ast.BlockStmt) int {
	deepest, depth := 0, 0
	var stack []ast.Node
	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if opensBlock(top, stack) {
				depth--
			}
			return true
		}
		if opensBlock(n, stack) {
			depth++
			deepest = max(deepest, depth)
		}
		stack = append(stack, n)
		return true
	})
	return deepest
}

func opensBlock(n ast.Node, parents []ast.Node) bool {
	switch n.(type) {
	case *ast.IfStmt:
		if len(parents) > 0 {
			if parent, ok := parents[len(parents)-1].(*ast.IfStmt); ok && parent.Else == n {
				return false
			}
		}
		return true
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, *ast.FuncLit:
		return true
	default:
		return false
	}
}

// complexity is cyclomatic complexity: one path, plus one per branch.
func complexity(body *ast.BlockStmt) int {
	c := 1
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			c++
		case *ast.CaseClause:
			if s.List != nil {
				c++
			}
		case *ast.CommClause:
			if s.Comm != nil {
				c++
			}
		case *ast.BinaryExpr:
			if s.Op == token.LAND || s.Op == token.LOR {
				c++
			}
		}
		return true
	})
	return c
}

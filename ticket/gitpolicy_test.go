package ticket

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// This file polices the whole module, not just this package, which is why it
// walks up from here rather than staying put. It lives in `ticket` because the
// read-only promise is this library's, and because a test that reads source
// text has no reason to sit anywhere in particular. TestCorpusCoversEveryPlanCode
// already reads ../docs/plan.md, so reaching up is the established shape.

// gitHelpers are the only functions allowed to execute git. One per package:
// runGit in ticket, readGit in cli. Adding a third means adding it here and
// saying why in plan 7.4, which is the review this test exists to force.
var gitHelpers = map[string]bool{"runGit": true, "readGit": true}

// planGitCommandPattern reads the command column of the table in plan 7.4.
// The character class allows the hyphen because `symbolic-ref` has one.
var planGitCommandPattern = regexp.MustCompile("(?m)^\\| `([a-z-]+)` \\|")

// planAllowedGitCommands reads the allowlist from plan 7.4 rather than
// duplicating it here, on the same argument as TestCorpusCoversEveryPlanCode:
// the plan is authoritative, so the test has to fail when the two disagree.
func planAllowedGitCommands(t *testing.T) map[string]bool {
	t.Helper()
	plan, err := os.ReadFile(filepath.Join("..", "docs", "plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(plan)
	start := strings.Index(text, "### 7.4 The Git commands this code runs")
	end := strings.Index(text, "## 8. Query surface")
	if start < 0 || end < 0 || end < start {
		t.Fatal("cannot find section 7.4 in docs/plan.md")
	}
	allowed := map[string]bool{}
	for _, m := range planGitCommandPattern.FindAllStringSubmatch(text[start:end], -1) {
		allowed[m[1]] = true
	}
	if len(allowed) == 0 {
		t.Fatal("section 7.4 lists no commands, which cannot be right")
	}
	return allowed
}

// moduleGoFiles returns every non-test Go file in the module. Test files are
// exempt: they run `git init` to build fixture repositories, and a fixture
// repository is not a user's repository.
func moduleGoFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// callName renders the function of a call as either "name" or "pkg.name".
func callName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		if pkg, ok := fn.X.(*ast.Ident); ok {
			return pkg.Name + "." + fn.Sel.Name
		}
	}
	return ""
}

// stringLit returns the value of an untyped string literal, and false for an
// expression that is anything else. A non-literal is a failure rather than a
// skip: a git command assembled at runtime is exactly what this test is for.
func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

// TestGitCommandsAreReadOnly enforces plan 7.4. AGENTS.md and acceptance
// criterion "no command performs a remote operation as a side effect" both
// promise this library runs no git command that writes, and before this test
// the only thing holding that promise was a doc comment. It matters more since
// v0.2.0, because a host embedding the cli package ships our git calls inside
// its own binary, running in the user's repository.
//
// Three assertions, in the order plan 7.4 states them.
func TestGitCommandsAreReadOnly(t *testing.T) {
	allowed := planAllowedGitCommands(t)

	var execCalls, helperCalls int
	fset := token.NewFileSet()

	for _, path := range moduleGoFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				where := fset.Position(call.Pos())
				name := callName(call)

				switch {
				case name == "exec.Command" || name == "exec.CommandContext":
					execCalls++
					// One: the binary is git and nothing else.
					bin := 0
					if name == "exec.CommandContext" {
						bin = 1
					}
					if len(call.Args) <= bin {
						t.Errorf("%s: %s has no binary argument", where, name)
						return true
					}
					switch got, ok := stringLit(call.Args[bin]); {
					case !ok:
						t.Errorf("%s: %s names a binary that is not a string literal, so nothing can check it", where, name)
					case got != "git":
						t.Errorf("%s: %s runs %q; this module executes git only", where, name, got)
					}
					// Two: it sits in one of the approved helpers.
					if !gitHelpers[fn.Name.Name] {
						t.Errorf("%s: %s executes git in %s, which is not one of the helpers in plan 7.4", where, name, fn.Name.Name)
					}

				case gitHelpers[name]:
					helperCalls++
					// Three: it names a command the plan allows. Argument
					// zero is the directory, so the command follows it.
					if len(call.Args) < 2 {
						t.Errorf("%s: %s is called with no git command", where, name)
						return true
					}
					switch got, ok := stringLit(call.Args[1]); {
					case !ok:
						t.Errorf("%s: %s is called with a computed git command, which plan 7.4 does not allow", where, name)
					case !allowed[got]:
						t.Errorf("%s: %s runs %q, which plan 7.4 does not list; add it there with its reason, or do not run it", where, name, got)
					}
				}
				return true
			})
		}
	}

	// A guard that silently matches nothing is worse than no guard, so fail
	// when the walk found no call of either kind.
	if execCalls == 0 {
		t.Error("found no exec.Command call in the module, so this test asserted nothing about the binary")
	}
	if helperCalls == 0 {
		t.Error("found no call to a git helper, so this test asserted nothing about the commands")
	}
}

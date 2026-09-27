// Command accplan prints, for every acceptance test, whether it would run
// with the current environment and why not: RUN, or SKIPPED with the reason
// (TF_ACC not set, capability gate not enabled, destructive gate not
// enabled, credentials unavailable, test input not set). It runs nothing and
// prints variable names only, never their values.
//
// It finds the tests by reading the accgate.Require* calls in the repository's test files, so a new
// acceptance test shows up without being registered anywhere.
//
//	go run ./cmd/accplan
//	SAP_INTEGRATION_SUITE_ACC_ALL=1 TF_ACC=1 go run ./cmd/accplan
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apidiscovery"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// planned is one acceptance test and what gates it.
type planned struct {
	Package     string
	Test        string
	Capability  accgate.Capability
	Destructive bool
	Extra       []string
	// Unresolved is set when an argument is not a constant; the test then
	// needs variables accplan cannot name statically.
	Unresolved bool
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	consts, err := stringConstants(root)
	if err != nil {
		fail(err)
	}
	tests, err := findTests(root, consts)
	if err != nil {
		fail(err)
	}

	counts := map[string]int{}
	fmt.Printf("%-78s %s\n", "TEST", "STATUS")
	for _, p := range tests {
		if p.Package == "apidiscovery" && p.Test == "TestAccMetadata" {
			// One subtest per service, each with its own credentials.
			for _, svc := range apidiscovery.Services {
				d := accgate.Check{Capability: p.Capability, Credentials: svc.Env}.Evaluate(os.Getenv)
				if svc.SpecificationOnly() {
					d = accgate.Decision{Status: "SKIPPED — official specification only (cmd/apidiscovery -spec-dir)"}
				}
				report(counts, fmt.Sprintf("%s/%s/%s", p.Package, p.Test, svc.ID), p.Capability, d)
			}
			continue
		}
		d := accgate.Evaluate(os.Getenv, p.Capability, p.Destructive, p.Extra...)
		if d.Run && p.Unresolved {
			d.Status = "RUN (may still skip on test-specific variables)"
		}
		report(counts, p.Package+"/"+p.Test, p.Capability, d)
	}
	fmt.Printf("\n%d run, %d skipped\n", counts["run"], counts["skip"])
}

func report(counts map[string]int, name string, c accgate.Capability, d accgate.Decision) {
	if d.Run {
		counts["run"]++
	} else {
		counts["skip"]++
	}
	fmt.Printf("%-78s %s\n", name+" ["+string(c)+"]", d.Status)
}

// stringConstants maps "pkg.Name" to the value of every string constant
// declared in the repository's non-test Go files, so that arguments such as
// accgate.CloudIntegration or samples.EnvLocalContent can be resolved.
func stringConstants(root string) (map[string]string, error) {
	out := map[string]string{}
	err := walkGo(root, false, func(pkg string, f *ast.File) {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, name := range vs.Names {
					if i < len(vs.Values) {
						if v, ok := literal(vs.Values[i], pkg, out); ok {
							out[pkg+"."+name.Name] = v
						}
					}
				}
			}
		}
	})
	return out, err
}

// literal evaluates a string literal, a known constant, or a sum of them.
func literal(e ast.Expr, pkg string, consts map[string]string) (string, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind == token.STRING {
			v, err := strconv.Unquote(x.Value)
			return v, err == nil
		}
	case *ast.Ident:
		v, ok := consts[pkg+"."+x.Name]
		return v, ok
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			v, ok := consts[id.Name+"."+x.Sel.Name]
			return v, ok
		}
	case *ast.BinaryExpr:
		if x.Op == token.ADD {
			a, ok1 := literal(x.X, pkg, consts)
			b, ok2 := literal(x.Y, pkg, consts)
			return a + b, ok1 && ok2
		}
	case *ast.CallExpr:
		// Capability("X") conversions.
		if len(x.Args) == 1 {
			return literal(x.Args[0], pkg, consts)
		}
	}
	return "", false
}

func findTests(root string, consts map[string]string) ([]planned, error) {
	var tests []planned
	err := walkGo(root, true, func(pkg string, f *ast.File) {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "accgate" ||
					!strings.HasPrefix(sel.Sel.Name, "Require") || len(call.Args) < 2 {
					return true
				}
				p := planned{Package: pkg, Test: fn.Name.Name, Destructive: sel.Sel.Name == "RequireDestructive"}
				c, ok := literal(call.Args[1], pkg, consts)
				if !ok {
					p.Unresolved = true
				}
				p.Capability = accgate.Capability(c)
				for _, a := range call.Args[2:] {
					if v, ok := literal(a, pkg, consts); ok && call.Ellipsis == token.NoPos {
						p.Extra = append(p.Extra, v)
					} else {
						p.Unresolved = true
					}
				}
				tests = append(tests, p)
				return false
			})
		}
	})
	sort.Slice(tests, func(i, j int) bool {
		if tests[i].Package != tests[j].Package {
			return tests[i].Package < tests[j].Package
		}
		return tests[i].Test < tests[j].Test
	})
	return tests, err
}

// walkGo parses the Go files below root (test files only, or non-test files
// only) and calls visit with each file's package name.
func walkGo(root string, testFiles bool, visit func(pkg string, f *ast.File)) error {
	fset := token.NewFileSet()
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error { //nolint:gosec // G703: walks the repository named on the command line
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor" || name == "tools") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") != testFiles {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		visit(strings.TrimSuffix(f.Name.Name, "_test"), f)
		return nil
	})
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

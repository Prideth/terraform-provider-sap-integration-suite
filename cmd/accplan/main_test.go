package main

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// TestEveryAcceptanceTestIsGated fails for a TestAcc function that does not
// call accgate.Require*, because such a test would run without a capability
// gate (and would be missing from the plan).
func TestEveryAcceptanceTestIsGated(t *testing.T) {
	root := "../.."
	consts, err := stringConstants(root)
	if err != nil {
		t.Fatal(err)
	}
	tests, err := findTests(root, consts)
	if err != nil {
		t.Fatal(err)
	}
	gated := map[string]bool{}
	for _, p := range tests {
		gated[p.Package+"/"+p.Test] = true
		if _, ok := accgate.Lookup(p.Capability); !ok {
			t.Errorf("%s/%s: capability %q is unknown or not a constant", p.Package, p.Test, p.Capability)
		}
	}
	var accTests []string
	err = walkGo(root, true, func(pkg string, f *ast.File) {
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil && isAcceptanceTest(fn) {
				accTests = append(accTests, pkg+"/"+fn.Name.Name)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(accTests) == 0 {
		t.Fatal("found no acceptance tests; is the root right?")
	}
	for _, name := range accTests {
		if !gated[name] {
			t.Errorf("%s does not call accgate.Require, accgate.RequireService or accgate.RequireDestructive", name)
		}
	}
}

// isAcceptanceTest: named TestAcc<Name> (TestAccess... is a unit test of
// access policies), or running Terraform through resource.Test.
func isAcceptanceTest(fn *ast.FuncDecl) bool {
	name := fn.Name.Name
	if strings.HasPrefix(name, "TestAcc") && !strings.HasPrefix(name, "TestAccess") &&
		len(name) > len("TestAcc") && name[len("TestAcc")] >= 'A' && name[len("TestAcc")] <= 'Z' {
		return true
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "resource" && (sel.Sel.Name == "Test" || sel.Sel.Name == "ParallelTest") {
					found = true
				}
			}
		}
		return !found
	})
	return found
}

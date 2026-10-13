package engine

import (
	"go/ast"
	"go/importer"
	"go/parser"
	gotoken "go/token"
	"go/types"
	"os"
	"strings"
	"testing"
)

// TestEventKindsUnique type-checks the package's own source and asserts
// that no two EventKind constants share a value. Kinds are chained as
// "previous + 1" across files, so two kinds chained off the same
// predecessor would compile and silently share a number.
func TestEventKindsUnique(t *testing.T) {
	fset := gotoken.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, n, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	pkg, err := conf.Check("engine", fset, files, nil)
	if err != nil {
		t.Fatal(err)
	}
	kind := pkg.Scope().Lookup("EventKind")
	if kind == nil {
		t.Fatal("no EventKind type")
	}
	seen := map[string]string{}
	n := 0
	for _, name := range pkg.Scope().Names() {
		c, ok := pkg.Scope().Lookup(name).(*types.Const)
		if !ok || !types.Identical(c.Type(), kind.Type()) {
			continue
		}
		n++
		v := c.Val().ExactString()
		if other, dup := seen[v]; dup {
			t.Errorf("EventKind %s: %s and %s", v, other, name)
		}
		seen[v] = name
	}
	if n < 80 {
		t.Errorf("found %d EventKind constants, want the package's whole list", n)
	}
}

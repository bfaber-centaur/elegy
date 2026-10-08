package engine

import (
	"go/ast"
	"go/importer"
	"go/parser"
	gotoken "go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"
)

// TestEventKindsDistinct type-checks the package source and fails if two
// EventKind constants share a value. Event kinds are chained across files,
// so a chain started from the wrong constant would make two messages
// indistinguishable.
func TestEventKindsDistinct(t *testing.T) {
	fset := gotoken.NewFileSet()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, n := range names {
		if strings.HasSuffix(n, "_test.go") {
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
	seen := map[string]string{}
	for _, n := range pkg.Scope().Names() {
		c, ok := pkg.Scope().Lookup(n).(*types.Const)
		if !ok || c.Type().String() != "engine.EventKind" {
			continue
		}
		v := c.Val().String()
		if other, dup := seen[v]; dup {
			t.Errorf("%s and %s are both event kind %s", other, n, v)
		}
		seen[v] = n
	}
}

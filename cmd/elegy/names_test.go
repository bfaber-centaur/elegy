package main

import (
	"go/ast"
	"go/importer"
	"go/parser"
	gotoken "go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// engineConstants type-checks the engine's source and returns the value
// of every constant of each named type, by constant name.
func engineConstants(t *testing.T, typeNames ...string) map[string]map[string]int {
	t.Helper()
	dir := filepath.Join("..", "..", "engine")
	fset := gotoken.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, n), nil, 0)
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
	out := map[string]map[string]int{}
	for _, tn := range typeNames {
		typ := pkg.Scope().Lookup(tn)
		if typ == nil {
			t.Fatalf("no engine type %s", tn)
		}
		out[tn] = map[string]int{}
		for _, name := range pkg.Scope().Names() {
			c, ok := pkg.Scope().Lookup(name).(*types.Const)
			if !ok || !types.Identical(c.Type(), typ.Type()) {
				continue
			}
			v, err := strconv.Atoi(c.Val().ExactString())
			if err != nil {
				t.Fatal(err)
			}
			out[tn][name] = v
		}
	}
	return out
}

// TestNames checks that show and order name every event kind, item kind,
// task kind and battle-plan setting the engine has, so a new one cannot be
// added without a name.
func TestNames(t *testing.T) {
	c := engineConstants(t, "EventKind", "ItemKind", "TaskKind", "Tactic", "TargetType", "AttackWho")
	if len(c["EventKind"]) < 100 || len(eventNames) != len(c["EventKind"]) {
		t.Errorf("%d event names for %d event kinds", len(eventNames), len(c["EventKind"]))
	}
	for name, v := range c["EventKind"] {
		if n, ok := eventNames[engine.EventKind(v)]; !ok || n == "" {
			t.Errorf("event kind %s (%d) has no name", name, v)
		}
	}
	for _, x := range []struct {
		typ   string
		names []string
	}{{"ItemKind", itemNames}, {"TaskKind", taskNames}, {"Tactic", tacticNames}, {"TargetType", targetNames}, {"AttackWho", attackNames}} {
		if len(x.names) != len(c[x.typ]) {
			t.Errorf("%d names for %d %s constants", len(x.names), len(c[x.typ]), x.typ)
		}
		for name, v := range c[x.typ] {
			if v < 0 || v >= len(x.names) || x.names[v] == "" {
				t.Errorf("%s %s (%d) has no name", x.typ, name, v)
			}
		}
	}
	seen := map[string]bool{}
	for _, n := range eventNames {
		if seen[n] {
			t.Errorf("event name %q twice", n)
		}
		seen[n] = true
	}
	for _, x := range []struct {
		got, want string
	}{
		{eventName(engine.EventPlanetCaptured), "planet captured"},
		{eventName(engine.EventWarp10FleetLost), "warp-10 fleet lost"},
		{eventName(-1), "event -1"},
		{fieldName(engine.Biotech), "biotechnology"},
		{fieldName(engine.NumFields), "field 6"},
		{fieldName(-1), "field -1"},
		{nextName(engine.NextSameField), "same field"},
		{nextName(engine.NextLowestField), "lowest field"},
		{nextName(engine.Energy), "energy"},
		{itemName(engine.ItemAutoMaxTerraform), "auto-max-terraform"},
		{itemName(engine.ItemAutoMaxTerraform + 1), "item 18"},
		{itemName(-1), "item -1"},
		{taskName(engine.TaskScrap), "scrap"},
		{taskName(engine.TaskScrap + 1), "task 10"},
		{taskName(-1), "task -1"},
	} {
		if x.got != x.want {
			t.Errorf("name %q, want %q", x.got, x.want)
		}
	}
	if i, err := lookup("item", "FACTORIES", itemNames); err != nil || i != int(engine.ItemFactory) {
		t.Errorf("lookup FACTORIES: %d %v", i, err)
	}
	if _, err := lookup("item", "widgets", itemNames); err == nil || !strings.Contains(err.Error(), "one of mines, factories") {
		t.Errorf("lookup widgets: %v", err)
	}
}

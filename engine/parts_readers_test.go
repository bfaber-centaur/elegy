package engine

import (
	"go/ast"
	"go/parser"
	gotoken "go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestPartsReaders: every reader of a design's parts outside slotsReaders
// uses Parts, so a misslotted part adds nothing to it. ai/ is left out:
// it is frozen, and its designs never misslot a part.
func TestPartsReaders(t *testing.T) {
	// slotsReaders are the functions outside tests that may read a design's
	// Slots rather than Parts: they compute the cost or the armor, which keep
	// a misslotted part in the year that reads the design (COMBAT.md "Parts
	// the game drops from a design"), build or compare designs, or work on
	// battleDesign's copy. Keys are "path:function".
	slotsReaders := map[string]bool{
		"engine/catalog.go:Hull":                   true, // a hull's slots
		"engine/catalog.go:newDesign":              true, // builds the design
		"engine/catalog.go:dropMisslottedParts":    true,
		"engine/catalog.go:misslotted":             true,
		"engine/catalog.go:working":                true,
		"engine/catalog.go:anyMisslotted":          true,
		"engine/catalog.go:Parts":                  true,
		"engine/ships.go:designCost":               true, // cost
		"engine/launch.go:designArmor":             true, // armor
		"engine/launch.go:StarbaseReplacementCost": true, // cost
		"engine/orders_design.go:fitFills":         true, // a hull's slots
		"engine/fleetops.go:readDesign":            true, // builds the design
		"engine/combat_token.go:tokenValues":       true, // battleDesign's copy; armor from the full design
		"objects/minefields.go:designArmor":        true, // armor
		"objects/trader.go:sameLoadout":            true, // the design's identity
		"internal/vectors/orders.go:orders":        true, // design orders
		"internal/vectors/vectors.go:loadVector":   true, // builds designs
		"internal/vectors/vectors.go:designEquals": true, // compares the stored loadout with the original's file
	}
	var found []string
	err := filepath.WalkDir("..", func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("..", path)
		rel = filepath.ToSlash(rel)
		if e.IsDir() {
			if rel == "ai" || strings.HasPrefix(e.Name(), ".") && rel != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(gotoken.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				s, ok := n.(*ast.SelectorExpr)
				if !ok || s.Sel.Name != "Slots" {
					return true
				}
				if x, ok := s.X.(*ast.SelectorExpr); ok && x.Sel.Name == "Hull" {
					return true // the hull's slots
				}
				found = append(found, rel+":"+fn.Name.Name)
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(found)
	seen := map[string]bool{}
	for _, k := range found {
		if !slotsReaders[k] && !seen[k] {
			t.Errorf("%s reads a design's Slots: use Parts, or add it to slotsReaders with the reason", k)
		}
		seen[k] = true
	}
	for k := range slotsReaders {
		if !seen[k] {
			t.Errorf("slotsReaders lists %s, which reads no Slots", k)
		}
	}
}

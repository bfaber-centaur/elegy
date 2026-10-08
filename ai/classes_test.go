package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// Every part named in the class table is a component of the table, so a
// typo cannot silently drop a part from a class.
func TestPartClassNamesExist(t *testing.T) {
	cat := engine.Components()
	for k, list := range partClasses {
		if len(list) == 0 {
			t.Errorf("class %d is empty", k)
		}
		for _, name := range list {
			if _, ok := cat.Lookup(name); !ok {
				t.Errorf("class %d: no component %q", k, name)
			}
		}
	}
}

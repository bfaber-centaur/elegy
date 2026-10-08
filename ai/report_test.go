package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// ViewOf keeps the other players' designs the report shows in full, by
// design index; a design seen only partially is not in View.Foreign.
func TestViewOfForeignDesigns(t *testing.T) {
	d := engine.Design{Name: "Their Fort"}
	r := game.Report{KnownDesigns: []game.KnownDesign{
		{Index: 4, Hull: "Space Station", Full: true, Design: &d},
		{Index: 7, Hull: "Scout"},
	}}
	v := ViewOf(r, Expert)
	if len(v.Foreign) != 1 || v.Foreign[4].Name != "Their Fort" {
		t.Errorf("Foreign %v, want design 4 only", v.Foreign)
	}
}

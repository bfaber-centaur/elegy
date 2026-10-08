package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// The planner's ideal warp is the engine's (PRODUCTION-LAUNCH.md "Ideal
// warp of the fleet", CONFIRMED for Long Hump 6 and Quick Jump 5): a Long
// Hump 6 scout alone gives 6, a Quick Jump 5 stack after it lowers the
// fleet to 5, and a stack whose design the view lacks gives 0.
func TestIdealWarpFromFleetShips(t *testing.T) {
	v := caView(t, 2420)
	qj, err := engine.Components().NewDesign("Quick", "Scout", []engine.SlotFill{fill(0, "Quick Jump 5", 1)})
	if err != nil {
		t.Fatal(err)
	}
	v.Ships = append(v.Ships, Design{Slot: 5, Index: 15, Design: qj})
	for _, tc := range []struct {
		name   string
		stacks []engine.Stack
		want   int
	}{
		{"long hump", []engine.Stack{{Design: 10, Count: 1}}, 6},
		{"then quick jump", []engine.Stack{{Design: 10, Count: 1}, {Design: 15, Count: 2}}, 5},
		{"unknown design", []engine.Stack{{Design: 99, Count: 1}}, 0},
	} {
		if got := v.idealWarp(&engine.Fleet{Stacks: tc.stacks}); got != tc.want {
			t.Errorf("%s: ideal warp %d, want %d", tc.name, got, tc.want)
		}
	}
}

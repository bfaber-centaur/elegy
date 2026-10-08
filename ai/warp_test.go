package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// repick runs the warp re-pick alone on the view's fleets.
func repick(v *View) *Result {
	res := &Result{}
	(&automation{v: v, res: res}).repickWarps()
	return res
}

// AI.md §11 "Warp choice", its example: a lone Long Hump 6 Scout 18 ly
// from its target at warp 6 is rewritten to warp 5, which also takes one
// year.
func TestWarpChoiceExample(t *testing.T) {
	v := caView(t, 2401)
	f := fleet(100, 1, engine.Point{X: 1000, Y: 1000}, 10, 1)
	f.Waypoints = []engine.Waypoint{{Pos: engine.Point{X: 1018, Y: 1000}, Warp: 6, Target: engine.TargetSpace}}
	v.Fleets = []engine.Fleet{f}
	res := repick(v)
	wps := ordersOf[engine.WaypointOrder](res.Orders)
	if len(wps) != 1 || wps[0].Fleet != 100 || wps[0].Waypoints[0].Warp != 5 || wps[0].Waypoints[0].Pos != f.Waypoints[0].Pos {
		t.Errorf("orders %+v, want warp 5 to the same point", wps)
	}
	// Once at warp 5 nothing changes, so no order.
	if res := repick(v); len(ordersOf[engine.WaypointOrder](res.Orders)) != 0 {
		t.Errorf("second re-pick wrote %+v", res.Orders)
	}
}

// The rule's steps: with fuel for every warp a colony ship (a
// colonization module lifts the raw cap) keeps warp 9 for a leg that
// needs it; without fuel the warp steps down to the ideal; a waypoint at
// a fleet keeps the warp the fuel and cap allow, with no slowing for the
// same years.
func TestWarpChoiceSteps(t *testing.T) {
	v := caView(t, 2401)
	far := engine.Point{X: 1300, Y: 1000} // 300 ly: 4 years at 9, 5 at 8
	for _, c := range []struct {
		name   string
		design int
		fuel   int
		target engine.TargetKind
		want   int
	}{
		{"colony ship, full tank", 11, 100000, engine.TargetSpace, 9},
		{"colony ship, empty tank", 11, 0, engine.TargetSpace, 6},
		{"scout, full tank, capped", 10, 100000, engine.TargetSpace, v.ships(&engine.Fleet{Stacks: []engine.Stack{{Design: 10, Count: 1}}}).RawWarp()},
	} {
		f := fleet(100, 1, engine.Point{X: 1000, Y: 1000}, c.design, 1)
		f.Fuel = c.fuel
		f.Waypoints = []engine.Waypoint{{Pos: far, Warp: 4, Target: c.target}}
		if got := v.warpChoice(&f); got != c.want {
			t.Errorf("%s: warp %d, want %d", c.name, got, c.want)
		}
	}
	// 18 ly to a fleet: warp 5 would take the same year, but a fleet
	// target is not slowed.
	f := fleet(100, 1, engine.Point{X: 1000, Y: 1000}, 10, 1)
	f.Waypoints = []engine.Waypoint{{Pos: engine.Point{X: 1018, Y: 1000}, Warp: 6, Target: engine.TargetFleet, ID: 7}}
	if got := v.warpChoice(&f); got != 6 {
		t.Errorf("fleet target: warp %d, want 6", got)
	}
}

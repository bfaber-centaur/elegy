package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

// repick runs the warp re-pick alone on the view's fleets, drawing from
// rng.
func repick(v *View, rng engine.Rand) *Result {
	res := &Result{}
	(&automation{v: v, res: res, rng: rng}).repickWarps()
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
	res := repick(v, &script{t: t})
	wps := ordersOf[engine.WaypointOrder](res.Orders)
	if len(wps) != 1 || wps[0].Fleet != 100 || wps[0].Waypoints[0].Warp != 5 || wps[0].Waypoints[0].Pos != f.Waypoints[0].Pos {
		t.Errorf("orders %+v, want warp 5 to the same point", wps)
	}
	// Once at warp 5 nothing changes, so no order.
	if res := repick(v, &script{t: t}); len(ordersOf[engine.WaypointOrder](res.Orders)) != 0 {
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

// §11's minefield rule: a field of 100 mines counts as ⌊10 + 10.5⌋² =
// 400 in squared radius, so a fleet 19 ly from its centre is inside and
// one 20 ly away is not. Inside another player's field the warp is 6 for
// heavy, 4 or 5 on Random(10) < 4 for standard, plus 1 for SS; the first
// field in object order decides; own and speed bump fields are ignored,
// with no draw. The fleet's leg (30 ly, scout) gives warp 6 otherwise.
func TestMinefieldWarp(t *testing.T) {
	field := func(owner int, x int, kind objects.MineKind) objects.MinefieldSighting {
		return objects.MinefieldSighting{Owner: owner, Pos: engine.Point{X: x, Y: 1000}, Mines: 100, Kind: kind}
	}
	for _, c := range []struct {
		name   string
		fields []objects.MinefieldSighting
		x      int // the fleet's position
		ss     bool
		draws  []int
		want   int
	}{
		{"standard, Random(10) = 3", []objects.MinefieldSighting{field(1, 1000, objects.Standard)}, 1019, false, []int{3}, 4},
		{"standard, Random(10) = 4", []objects.MinefieldSighting{field(1, 1000, objects.Standard)}, 1019, false, []int{4}, 5},
		{"heavy", []objects.MinefieldSighting{field(1, 1000, objects.Heavy)}, 1019, false, nil, 6},
		{"heavy, SS", []objects.MinefieldSighting{field(1, 1000, objects.Heavy)}, 1019, true, nil, 7},
		{"standard, SS", []objects.MinefieldSighting{field(1, 1000, objects.Standard)}, 1019, true, []int{0}, 5},
		{"20 ly: outside", []objects.MinefieldSighting{field(1, 1000, objects.Standard)}, 1020, false, nil, 6},
		{"own field", []objects.MinefieldSighting{field(0, 1000, objects.Heavy)}, 1019, false, nil, 6},
		{"speed bump", []objects.MinefieldSighting{field(1, 1000, objects.SpeedBump)}, 1019, false, nil, 6},
		{"first in object order", []objects.MinefieldSighting{field(1, 1000, objects.Heavy), field(2, 1010, objects.Standard)}, 1019, false, nil, 6},
	} {
		v := caView(t, 2401)
		if c.ss {
			v.Self.Race.PRT = engine.PRTSuperStealth
		}
		v.Minefields = c.fields
		f := fleet(100, 1, engine.Point{X: c.x, Y: 1000}, 10, 1)
		f.Waypoints = []engine.Waypoint{{Pos: engine.Point{X: c.x + 30, Y: 1000}, Warp: 9, Target: engine.TargetSpace}}
		v.Fleets = []engine.Fleet{f}
		rng := &script{t: t, draws: c.draws}
		res := repick(v, rng)
		wps := ordersOf[engine.WaypointOrder](res.Orders)
		if len(wps) != 1 || wps[0].Waypoints[0].Warp != c.want {
			t.Errorf("%s: orders %+v, want warp %d", c.name, wps, c.want)
		}
		if len(rng.draws) != 0 || len(rng.bounds) != len(c.draws) || len(rng.bounds) > 0 && rng.bounds[0] != 10 {
			t.Errorf("%s: draws %v, want Random(10) %d times", c.name, rng.bounds, len(c.draws))
		}
	}
}

// A58: the raw-cap exception reads an own planet's starbase design among
// the player's starbase designs only. A Space Station lifts the cap, an
// Orbital Fort does not, and an index that is a ship design (the planet
// without a starbase holding 10, the scout) counts as no design.
func TestWarpCapStarbaseSlot(t *testing.T) {
	for _, c := range []struct {
		name    string
		has     bool
		design  int
		uncaped bool
	}{
		{"Space Station", true, 20, true},
		{"Orbital Fort", true, 21, false},
		{"no starbase, last a Space Station", false, 22, true},
		{"no starbase, a ship design index", false, 10, false},
	} {
		v := caView(t, 2401)
		v.Planets = append(v.Planets, engine.Planet{ID: 3, Pos: v.Universe[2].Pos, Owner: 0, HasStarbase: c.has, StarbaseDesign: c.design})
		f := fleet(100, 1, v.Universe[0].Pos, 10, 1)
		wp := toPlanet(3, v.Universe[2].Pos, 6, engine.Task{})
		if got := v.uncapped(&f, wp); got != c.uncaped {
			t.Errorf("%s: uncapped %v, want %v", c.name, got, c.uncaped)
		}
	}
}

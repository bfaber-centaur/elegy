package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

func robotoidTest(t *testing.T, v *View, rng engine.Rand) *robotoidTurn {
	t.Helper()
	rt := &robotoidTurn{turn: newTurn(v, rng, &Result{}), d1415: -1, d1113: -1, d910: -1, d25: -1, d67: -1}
	rt.params()
	return rt
}

// §1 step 3: potency and armada size, and the age limit.
func TestRobotoidParams(t *testing.T) {
	for _, c := range []struct{ y, p, a, l int }{{50, 4, 6, 50}, {120, 4, 6, 70}, {150, 5, 8, 70}, {300, 13, 12, 100}} {
		rt := robotoidTest(t, caView(t, FirstYear+c.y), top{})
		if rt.potency != c.p || rt.size != c.a || rt.limit != c.l {
			t.Errorf("y %d: P %d A %d T %d, want %d %d %d", c.y, rt.potency, rt.size, rt.limit, c.p, c.a, c.l)
		}
	}
}

// §1 step 4: a Nubian in slot 14 is never marked obsolete.
func TestRobotoidAgeingNubian(t *testing.T) {
	v := caView(t, 2460)
	home := v.Universe[0].Pos
	nub := v.Ships[0]
	nub.Slot, nub.Index, nub.Design.Hull.Name = 14, 44, "Nubian"
	v.Ships = append(v.Ships, nub)
	v.Fleets = []engine.Fleet{fleet(100, 1, home, 44, 1)}
	rt := robotoidTest(t, v, top{})
	rt.ageing()
	if rt.obsolete[14] || rt.d1415 != 14 {
		t.Errorf("obsolete %v, D1415 %d; want 14 unmarked and current", rt.obsolete, rt.d1415)
	}
}

// AI.md §11 "Fleet classes": a warship is an attack fleet; an unarmed
// Frigate stops the walk even before a warship in a later slot.
func TestAttackFleet(t *testing.T) {
	v := caView(t, 2460)
	cat := engine.Components()
	dd, err := cat.NewDesign("D", "Destroyer", []engine.SlotFill{fill(0, "Long Hump 6", 1)})
	if err != nil {
		t.Fatal(err)
	}
	fr, err := cat.NewDesign("F", "Frigate", []engine.SlotFill{fill(0, "Long Hump 6", 1)})
	if err != nil {
		t.Fatal(err)
	}
	v.Ships = append(v.Ships, Design{Slot: 4, Index: 34, Design: dd}, Design{Slot: 3, Index: 33, Design: fr})
	rt := robotoidTest(t, v, top{})
	home := v.Universe[0].Pos
	war := fleet(100, 1, home, 34, 1)
	both := fleet(101, 2, home, 34, 1)
	both.Stacks = append(both.Stacks, engine.Stack{Design: 33, Count: 1})
	if !rt.attackFleet(&war) {
		t.Error("a Destroyer fleet is not an attack fleet")
	}
	if rt.attackFleet(&both) {
		t.Error("an unarmed Frigate in an earlier slot did not stop the walk")
	}
}

// §3 step 1: freighters when fewer than 0.8 F are alive.
func TestRobotoidFreighters(t *testing.T) {
	v := caView(t, 2430)
	v.Planets[0].Population = 3000
	v.Ships = append(v.Ships, Design{Slot: 11, Index: 41, Design: v.Ships[2].Design, Created: 2415})
	rt := robotoidTest(t, v, top{})
	rt.d1113, rt.hubs = 11, []int{1} // F = 4
	rt.produce(&v.Planets[0], 0)
	q := rt.q.get(&v.Planets[0])
	if len(q) == 0 || q[0].Kind != engine.ItemShip || q[0].Slot != 11 || q[0].Count != 1 {
		t.Errorf("queue %+v, want one ship of slot 11 first", q)
	}
}

// AI.md §6 steps 3 and 4: transports go to the nearest hub with room,
// then a hub with fewer than 4 takes one from a hub with 2 more.
func TestAssignHubs(t *testing.T) {
	v := caView(t, 2430)
	pv, err := engine.Components().NewDesign("Raider", "Privateer", []engine.SlotFill{fill(0, "Long Hump 6", 1)})
	if err != nil {
		t.Fatal(err)
	}
	v.Ships = append(v.Ships, Design{Slot: 11, Index: 41, Design: pv})
	home := v.Universe[0].Pos
	for i := range 3 {
		v.Fleets = append(v.Fleets, fleet(100+i, 1+i, home, 41, 1))
	}
	rt := robotoidTest(t, v, top{})
	rt.hubs = []int{1, 5}
	got := rt.assignHubs()
	n := map[int]int{}
	for _, h := range got {
		n[h]++
	}
	if n[1] != 2 || n[5] != 1 || got[102] != 5 {
		t.Errorf("assignment %v, want two at hub 1 and fleet 102 moved to hub 5", got)
	}
}

// AI.md §11 hub freighters' load task at an own planet: mode 0 loads all
// three minerals; mode 1 or 2 loads 66 % of the room in the scarce
// mineral and 33 % in the others, split between them (ASSUMPTION A56).
func TestHubLoad(t *testing.T) {
	all := hubLoad(0, engine.Boranium, 300)
	for m := range engine.NumMinerals {
		if all.Kind != engine.TaskTransport || all.Transport[m] != (engine.Transport{Action: engine.LoadAll}) {
			t.Errorf("mode 0 mineral %d: %+v, want load all", m, all.Transport[m])
		}
	}
	got := hubLoad(2, engine.Boranium, 300)
	want := [3]int{49, 198, 49}
	for m, n := range want {
		if got.Transport[m] != (engine.Transport{Action: engine.LoadExactly, Amount: n}) {
			t.Errorf("mode 2 mineral %d: %+v, want load exactly %d", m, got.Transport[m], n)
		}
	}
}

// AI.md §11 armada invasion test (ASSUMPTION A59: need in colonists and
// the cargo c in kT as plain numbers). A planet estimated at 4,000
// colonists with no defenses (defense % 4) needs ⌊4000/96⌋ = 41.
func TestArmadaDrop(t *testing.T) {
	rep := func(pop, def int) engine.PlanetReport {
		return engine.PlanetReport{Owner: 1, PopEstimate: pop, DefenseEstimate: def}
	}
	for _, c := range []struct {
		name string
		rep  engine.PlanetReport
		c    int
		want int
	}{
		{"41 < 250/5: drop c/2", rep(4000, 0), 250, 125},
		{"41 not < 200/5: none", rep(4000, 0), 200, 0},
		{"83 < 200 and c > 350", rep(8000, 0), 360, 180},
		{"195 < 200 and c > 350: drop 5·need/4", rep(18800, 0), 360, 243},
		{"need 200: none", rep(19200, 0), 360, 0},
		{"defense 15 (72 %): 4000/28 = 142 not < 340/5", rep(4000, 15), 340, 0},
		{"defense 0 instead: 41 < 340/5", rep(4000, 0), 340, 170},
		{"capped at 30,000", rep(400, 0), 100000, 30000},
	} {
		if got := armadaDrop(c.rep, c.c); got != c.want {
			t.Errorf("%s: drop %d, want %d", c.name, got, c.want)
		}
	}
}

// §11: a strong armada at another player's planet invades there, with a
// waypoint-0 unload-exactly task and no move; one whose troops do not
// suffice does nothing that turn.
func TestArmadaInvades(t *testing.T) {
	for _, c := range []struct {
		cargo, want int
	}{{250, 125}, {200, 0}} {
		v := caView(t, 2450)
		war, troop := v.Ships[0], v.Ships[0]
		war.Slot, war.Index = 4, 34
		troop.Slot, troop.Index = 9, 39
		v.Ships = append(v.Ships, war, troop)
		f := fleet(100, 1, v.Universe[2].Pos, 34, 2)
		f.Stacks = append(f.Stacks, engine.Stack{Design: 39, Count: 2})
		f.Cargo.Colonists = c.cargo
		v.Fleets = []engine.Fleet{f}
		v.Known[3] = engine.PlanetReport{Planet: 3, Level: engine.ReportNormal, Owner: 1, Pos: v.Universe[2].Pos, PopEstimate: 4000}
		rt := robotoidTest(t, v, top{})
		rt.armada(&v.Fleets[0])
		wps := ordersOf[engine.WaypointOrder](rt.res.Orders)
		if c.want == 0 {
			if len(rt.res.Orders) != 0 || len(rt.res.Unsupported) != 0 {
				t.Errorf("cargo %d: orders %+v, unsupported %v; want none", c.cargo, rt.res.Orders, rt.res.Unsupported)
			}
			continue
		}
		if len(wps) != 1 || len(wps[0].Waypoints) != 0 ||
			wps[0].Task.Kind != engine.TaskTransport || wps[0].Task.Transport[engine.CargoColonists] != (engine.Transport{Action: engine.UnloadExactly, Amount: c.want}) {
			t.Errorf("cargo %d: orders %+v, want unload exactly %d here", c.cargo, rt.res.Orders, c.want)
		}
	}
}

// robotoid.md §3's colonizer test at year index 40 with one owned planet
// known (e = 1) and no colony fleets (c = 0): 0 ≤ b − (e + c) ≤ 25 says
// yes, so b = 1 and b = 26 pass; b = 0 and b = 27 fall through to c < 5,
// where Random(2) draws 1 and says no. Ships built of other designs (the
// scout here) do not count.
func TestColonizerTestBuilt(t *testing.T) {
	for _, c := range []struct {
		built, other int
		want         bool
	}{{0, 0, false}, {1, 0, true}, {26, 0, true}, {27, 0, false}, {0, 1, false}} {
		v := caView(t, 2440)
		for i := range v.Ships {
			if colonyHull(v.Ships[i].Design.Hull.Name) {
				v.Ships[i].Built = c.built
			} else {
				v.Ships[i].Built = c.other
			}
		}
		rt := &robotoidTurn{turn: newTurn(v, top{}, &Result{})}
		if got := rt.colonizerTest(); got != c.want {
			t.Errorf("b = %d, other %d: colonizer test %v, want %v", c.built, c.other, got, c.want)
		}
	}
}

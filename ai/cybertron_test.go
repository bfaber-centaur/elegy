package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

func cyberTest(t *testing.T, v *View, rng engine.Rand) *cyberTurn {
	t.Helper()
	ct := &cyberTurn{turn: newTurn(v, rng, &Result{}), taken: map[int]bool{}, inbound: map[int]int{}}
	ct.params()
	return ct
}

// §1 step 4: the strength unit and the age limit.
func TestCybertronParams(t *testing.T) {
	for _, c := range []struct{ y, s, l int }{{40, 1, 50}, {60, 2, 50}, {110, 8, 50}, {130, 12, 70}, {250, 51, 100}} {
		v := caView(t, FirstYear+c.y)
		ct := cyberTest(t, v, top{})
		if ct.s != c.s || ct.limit != c.l || ct.potency != 0 || ct.size != 0 {
			t.Errorf("y %d: s %d, L %d, P %d, A %d; want s %d, L %d, P = A = 0", c.y, ct.s, ct.limit, ct.potency, ct.size, c.s, c.l)
		}
	}
}

// AI.md §10 "Merging": fleets of the slots at one place merge into the
// first; the fleet after each merged one is skipped (LEGACY BUG).
func TestCybertronMerge(t *testing.T) {
	v := caView(t, 2420)
	home := v.Universe[0].Pos
	v.Fleets = []engine.Fleet{fleet(100, 1, home, 10, 1), fleet(101, 2, home, 10, 1), fleet(102, 3, home, 10, 1),
		fleet(103, 4, home, 10, 1), fleet(104, 5, v.Universe[1].Pos, 10, 1), fleet(105, 6, home, 11, 1)}
	ct := cyberTest(t, v, top{})
	ct.merge([]int{0})
	got := ordersOf[engine.MergeOrder](ct.res.Orders)
	if len(got) != 2 || got[0].Into != 100 || got[0].From[0] != 101 || got[1].Into != 100 || got[1].From[0] != 103 {
		t.Fatalf("merges %+v, want 101 then 103 into 100 (102 skipped)", got)
	}
	if n := len(v.Fleets); n != 4 || v.Fleets[0].Stacks[0].Count != 3 {
		t.Errorf("picture: %d fleets, first has %d ships; want 4 and 3", n, v.Fleets[0].Stacks[0].Count)
	}
}

// §3: after the 14–15 check DD is unmarked again, so GG stays obsolete
// (LEGACY BUG); a warship group past the limit marks designs with ships
// and deletes the rest, keeping its first slot when a higher one is kept.
func TestCybertronAgeing(t *testing.T) {
	v := caView(t, 2460) // y = 60, L = 50
	home := v.Universe[0].Pos
	d := v.Ships[0].Design
	for _, k := range []int{4, 14, 15, 6, 7, 8} {
		v.Ships = append(v.Ships, Design{Slot: k, Index: 30 + k, Design: d, Created: 2400})
	}
	v.Fleets = []engine.Fleet{fleet(100, 1, home, 34, 1), fleet(101, 2, home, 44, 1), fleet(102, 3, home, 45, 1), fleet(103, 4, home, 37, 1)}
	ct := cyberTest(t, v, top{})
	ct.ageing()
	if ct.dd != 4 || ct.gg != 14 || ct.obsolete[4] || !ct.obsolete[14] || !ct.obsolete[15] {
		t.Errorf("DD %d GG %d obsolete %v; want DD 4 unmarked, GG 14 and 15 marked", ct.dd, ct.gg, ct.obsolete)
	}
	if !ct.aged6 || !ct.obsolete[7] || !ct.obsolete[6] || ct.sd.slots[8].present || !ct.sd.slots[6].present {
		t.Errorf("group 6: aged %v, obsolete %v, slot 6 %v, slot 8 %v; want 7 marked, 6 kept and marked, 8 deleted",
			ct.aged6, ct.obsolete, ct.sd.slots[6].present, ct.sd.slots[8].present)
	}
	// Slot 2 (the fixture's old miner, no ships) goes in the 2–3 check.
	if del := ordersOf[engine.DeleteDesignOrder](ct.res.Orders); len(del) != 2 || del[0].Slot != 2 || del[1].Slot != 8 {
		t.Errorf("deletes %+v, want slots 2 and 8", del)
	}
}

// Armada targeting under clean state: an idle armada at an own planet is
// never weak, so it targets the best threat-plus-distance score at warp 4.
func TestCybertronArmada(t *testing.T) {
	v := caView(t, 2460)
	home := v.Universe[0].Pos
	v.Ships = append(v.Ships, Design{Slot: 6, Index: 36, Design: v.Ships[0].Design, Created: 2450})
	v.Known[3] = engine.PlanetReport{Planet: 3, Level: engine.ReportNormal, Owner: 1, PopEstimate: 1000} // 60 ly: 5 + 5
	v.Known[5] = engine.PlanetReport{Planet: 5, Level: engine.ReportNormal, Owner: 1, Starbase: true}    // 120 ly: 2 + 4
	v.Fleets = []engine.Fleet{fleet(100, 1, home, 36, 2)}
	ct := cyberTest(t, v, &script{t: t})
	ct.notes()
	ct.passA()
	ct.passB()
	got := ordersOf[engine.WaypointOrder](ct.res.Orders)
	if len(got) != 1 || len(got[0].Waypoints) != 1 || got[0].Waypoints[0].ID != 3 || got[0].Waypoints[0].Warp != 4 {
		t.Fatalf("orders %+v, want a move to planet 3 at warp 4", got)
	}
	if !ct.targeted[3] {
		t.Error("planet 3 not marked targeted")
	}
}

// The freighter rule at a crowded own planet: load up to 1,000 kT
// (clamped to the hold) and go to the nearest own planet within 400 ly
// with fewer than 20,000 colonists, with no task, warp 4.
func TestCybertronFreighter(t *testing.T) {
	v := caView(t, 2430)
	pv, err := engine.Components().NewDesign("Raider", "Privateer", []engine.SlotFill{fill(0, "Long Hump 6", 1)})
	if err != nil {
		t.Fatal(err)
	}
	v.Ships = append(v.Ships, Design{Slot: 2, Index: 32, Design: pv, Created: 2421})
	v.Planets[0].Population = 3000
	v.Planets = append(v.Planets, engine.Planet{ID: 3, Pos: v.Universe[2].Pos, Owner: 0, Env: [3]int{50, 50, 50}, Population: 50})
	v.Fleets = []engine.Fleet{fleet(100, 1, v.Universe[0].Pos, 32, 1)}
	ct := cyberTest(t, v, top{})
	ct.passA()
	ct.passB()
	cargo := ordersOf[engine.CargoOrder](ct.res.Orders)
	move := ordersOf[engine.WaypointOrder](ct.res.Orders)
	if len(cargo) != 1 || cargo[0].Amounts[engine.CargoColonists] != pv.CargoCapacity {
		t.Errorf("cargo %+v, want %d kT of colonists", cargo, pv.CargoCapacity)
	}
	if len(move) != 1 || move[0].Waypoints[0].ID != 3 || move[0].Waypoints[0].Warp != 4 {
		t.Errorf("moves %+v, want planet 3 at warp 4", move)
	}
}

// §4.2 before year index 60: the colonizer test always passes, so a
// station planet queues a colony ship.
func TestCybertronColonyShipProduction(t *testing.T) {
	v := caView(t, 2410)
	ct := cyberTest(t, v, top{})
	ct.dd, ct.gg, ct.fr, ct.gr = -1, -1, -1, -1
	ct.produce(&v.Planets[0], 0)
	q := ct.q.get(&v.Planets[0])
	found := false
	for _, it := range q {
		if it.Kind == engine.ItemShip && it.Slot == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("queue %+v, want a colony ship", q)
	}
}

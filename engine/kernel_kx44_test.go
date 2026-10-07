package engine

import "testing"

// Movement rules KERNEL.md added in stars-elegy PR #44 (TK-117, FM round 2).

func TestConfirmedARColonistsInFlight(t *testing.T) {
	// TK-117 and TK-107: an AR fleet with more than 10 kT of colonists
	// loses trunc((C+11)·3/100) kT in a year it moves; a stationary fleet
	// loses nothing.
	for _, c := range []struct {
		colonists, warp, want int
	}{{10, 5, 10}, {11, 5, 11}, {25, 5, 24}, {40, 5, 39}, {100, 5, 97}, {200, 5, 194}, {200, 0, 200}} {
		g := scoutGame(Engine{}, 1, Fleet{ID: 1, Stacks: []Stack{{Design: 0, Count: 1}}, Pos: Point{100, 100}, Fuel: 300,
			Cargo:     Cargo{Colonists: c.colonists},
			Waypoints: []Waypoint{{Pos: Point{400, 100}, Warp: c.warp}}})
		g.Designs[0].CargoCapacity = 1000
		g.Players = []Player{{Race: Race{PRT: PRTAlternateReality}}}
		ev := moveFleets(&g)
		if got := g.Fleets[0].Cargo.Colonists; got != c.want {
			t.Errorf("%d kT at warp %d: %d left, want %d", c.colonists, c.warp, got, c.want)
		}
		lost := c.colonists - c.want
		if (lost > 0) != (len(ev) > 0 && ev[0].Kind == EventColonistsLostInFlight && ev[0].Count == lost) {
			t.Errorf("%d kT: events %v", c.colonists, ev)
		}
	}
	// Another race loses nothing.
	g := scoutGame(Engine{}, 1, Fleet{ID: 1, Stacks: []Stack{{Design: 0, Count: 1}}, Fuel: 300, Cargo: Cargo{Colonists: 200},
		Waypoints: []Waypoint{{Pos: Point{300, 0}, Warp: 5}}})
	g.Designs[0].CargoCapacity = 1000
	g.Players = []Player{{}}
	moveFleets(&g)
	if g.Fleets[0].Cargo.Colonists != 200 {
		t.Errorf("non-AR: %d", g.Fleets[0].Cargo.Colonists)
	}
}

func TestConfirmedUnderEnginedFuelWrap(t *testing.T) {
	// FM-105: a Large Freighter with one of its two Long Hump 6 engines
	// (134 kT) at warp 5 with 200, 50 and 500 mg moves 7, 1 and 19 ly and
	// ends with 0 mg: f = 99999 and the 32-bit range estimate wraps.
	if got := fuelTermWrap(underEngined, 1000, 134, true); got != 257482 {
		t.Errorf("wrapped term %d, want 257482", got)
	}
	if got := fuelTermWrap(underEngined, 1000, 134, false); got != 6699933 {
		t.Errorf("exact term %d, want 6699933", got)
	}
	for _, c := range []struct{ fuel, want int }{{200, 7}, {50, 1}, {500, 19}} {
		d := Design{Name: "Large Freighter", Hull: Hull{Name: "Large Freighter", Slots: []HullSlot{{Kinds: []PartKind{PartEngine}, Max: 2}}},
			Mass: 134, Engines: 1, FuelCapacity: 2600}
		d.Engine.Fuel = [11]int{0, 0, 0, 0, 0, 0, 100, 100, 100, 100, 100}
		g := Game{Designs: []Design{d}, Fleets: []Fleet{{ID: 1, Stacks: []Stack{{Design: 0, Count: 1}}, Fuel: c.fuel,
			Waypoints: []Waypoint{{Pos: Point{1000, 0}, Warp: 5}}}}}
		moveFleets(&g)
		if f := g.Fleets[0]; f.Pos != (Point{c.want, 0}) || f.Fuel != 0 {
			t.Errorf("fuel %d: at %v with %d mg, want (%d,0) and 0", c.fuel, f.Pos, f.Fuel, c.want)
		}
	}
}

func TestPredictionFuelTermForms(t *testing.T) {
	// Only the integer form wraps; a full set of engines uses its table.
	if got := fuelTermWrap(underEngined, 1000, 4000, true); got != underEngined*1000*4000/2000 {
		t.Errorf("float form: %d", got)
	}
	d := Design{Hull: Hull{Slots: []HullSlot{{Max: 2}}}, Engines: 2}
	d.Engine.Fuel[5] = 35
	if engineFactor(d, 5) != 35 {
		t.Error("full engines")
	}
	d.Engines = 0
	if engineFactor(d, 5) != underEngined {
		t.Error("empty engine slot")
	}
}

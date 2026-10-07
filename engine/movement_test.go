package engine

import "testing"

// The FM-001..004 corpus tests cover every CONFIRMED movement vector in
// KERNEL.md; these are the worked vectors restated, plus BINARY-ONLY rules
// the corpus does not exercise.

func scoutGame(engine Engine, engines int, fleets ...Fleet) Game {
	d := Design{Name: "scout", Mass: 18, Engine: engine, Engines: engines, FuelCapacity: 300}
	return Game{Designs: []Design{d}, Fleets: fleets}
}

func TestConfirmedFuelCostVectors(t *testing.T) {
	g := Game{Designs: fmDesigns()}
	tests := []struct {
		name string
		f    Fleet
		warp int
		dist int
		want int
	}{
		{"one QJ5 scout 25 ly w5 (FM-001 0)", Fleet{Stacks: []Stack{{Design: 0, Count: 1}}}, 5, 25, 3},
		{"seven QJ5 scouts 36 ly w6 (FM-001 42)", Fleet{Stacks: []Stack{{Design: 0, Count: 7}}}, 6, 36, 41},
		{"3 QJ5 + 1 AD8 49 ly w7 (FM-002 32)", Fleet{Stacks: []Stack{{Design: 0, Count: 3}, {Design: 5, Count: 1}}}, 7, 49, 74},
		{"warp-9 scout 55 ly (FM-001 67)", Fleet{Stacks: []Stack{{Design: 0, Count: 1}}}, 9, 55, 45},
		{"QJ5 + LH6 scouts 4 ly w2 (FM-004 MS)", Fleet{Stacks: []Stack{{Design: 0, Count: 1}, {Design: 3, Count: 1}}}, 2, 4, 0},
		{"QJ5 + FM scouts 25 ly w5 (FM-004 MS)", Fleet{Stacks: []Stack{{Design: 0, Count: 1}, {Design: 7, Count: 1}}}, 5, 25, 3},
		{"QJ5 + LH6 freighters 70 kT (FM-004 CA)", Fleet{Stacks: []Stack{{Design: 1, Count: 1}, {Design: 9, Count: 1}}, Cargo: Cargo{Minerals: Minerals{70, 0, 0}}}, 6, 36, 30},
		{"QJ5 + LH6 freighters 100 kT (FM-004 CA)", Fleet{Stacks: []Stack{{Design: 1, Count: 1}, {Design: 9, Count: 1}}, Cargo: Cargo{Minerals: Minerals{100, 0, 0}}}, 6, 36, 40},
	}
	for _, tt := range tests {
		if got := g.FuelCost(&tt.f, tt.warp, tt.dist); got != tt.want {
			t.Errorf("%s: %d mg, want %d", tt.name, got, tt.want)
		}
	}
}

func TestConfirmedFuelRangeVectors(t *testing.T) {
	g := Game{Designs: fmDesigns()}
	// FM-004 LR first row: LH6 scout, warp 6, fuel 3: C1000 120, R 25.
	f := Fleet{Stacks: []Stack{{Design: 3, Count: 1}}, Fuel: 3}
	if r, _ := g.fuelRange(&f, 6); r != 25 {
		t.Errorf("R = %d, want 25", r)
	}
}

func TestPredictionNoFreeWarpLeavesWarp(t *testing.T) {
	var costly Engine
	for w := 1; w <= 10; w++ {
		costly.Fuel[w] = 100
	}
	g := scoutGame(costly, 1, Fleet{ID: 1, Stacks: []Stack{{Design: 0, Count: 1}}, Pos: Point{0, 0}, Fuel: 1,
		Waypoints: []Waypoint{{Pos: Point{100, 0}, Warp: 6}}})
	moveFleets(&g)
	f := g.Fleets[0]
	if f.Fuel != 0 || f.Waypoints[0].Warp != 6 {
		t.Errorf("fuel %d warp %d, want 0 and unchanged 6", f.Fuel, f.Waypoints[0].Warp)
	}
}

func TestPredictionRamScoopEnginesPerShip(t *testing.T) {
	// Two free engines per ship at warp 1 (free through warp 4): 2·10·1.
	fm := fmEngines[2]
	g := scoutGame(fm, 2, Fleet{ID: 1, Stacks: []Stack{{Design: 0, Count: 1}}, Pos: Point{0, 0}, Fuel: 100,
		Waypoints: []Waypoint{{Pos: Point{100, 0}, Warp: 1}}})
	moveFleets(&g)
	if got := g.Fleets[0].Fuel; got != 120 {
		t.Errorf("fuel %d, want 120", got)
	}
}

func TestConfirmedChaseVectors(t *testing.T) {
	qj5 := fmEngines[1]
	// Two fleets 20 ly apart chasing each other at warp 4: lower id moves
	// 12, higher 8.
	g := scoutGame(qj5, 1,
		Fleet{ID: 1, Stacks: []Stack{{Design: 0, Count: 1}}, Pos: Point{1200, 1000}, Fuel: 300, Waypoints: []Waypoint{{Target: TargetFleet, ID: 2, Warp: 4}}},
		Fleet{ID: 2, Stacks: []Stack{{Design: 0, Count: 1}}, Pos: Point{1220, 1000}, Fuel: 300, Waypoints: []Waypoint{{Target: TargetFleet, ID: 1, Warp: 4}}},
	)
	moveFleets(&g)
	if a, b := g.Fleets[0].Pos.X, g.Fleets[1].Pos.X; a != 1212 || b != 1212 {
		t.Errorf("ends at %d and %d, want 1212 both", a, b)
	}
}

package engine

import "testing"

// BINARY-ONLY rules KERNEL.md added in stars-elegy PR #14, worked from the
// rule text with no oracle observation.

func TestPredictionChaserRangeReducedByMoved(t *testing.T) {
	// A (fuel 3, R 18 at warp 6) chases B, which chases the stationary C.
	// Round 1: A steps 8 (a fifth of 36) while B is still moving and pays
	// 2 mg; B reaches C. Round 2: A's step is the remaining 28, but R − moved
	// = 10, so A moves 10 (18 in all), ends with 0 and runs dry.
	qj5 := fmEngines[1]
	g := scoutGame(qj5, 1,
		Fleet{ID: 1, Stacks: []Stack{{0, 1}}, Pos: Point{0, 0}, Fuel: 3, Waypoints: []Waypoint{{Target: TargetFleet, ID: 2, Warp: 6}}},
		Fleet{ID: 2, Stacks: []Stack{{0, 1}}, Pos: Point{100, 0}, Fuel: 300, Waypoints: []Waypoint{{Target: TargetFleet, ID: 3, Warp: 6}}},
		Fleet{ID: 3, Stacks: []Stack{{0, 1}}, Pos: Point{110, 0}},
	)
	ev := moveFleets(&g)
	a := g.Fleets[0]
	if a.Pos != (Point{18, 0}) || a.Fuel != 0 || a.Waypoints[0].Warp != 1 {
		t.Errorf("A at %v fuel %d warp %d, want (18,0), 0, warp 1", a.Pos, a.Fuel, a.Waypoints[0].Warp)
	}
	if len(ev) == 0 || ev[0].Kind != EventOutOfFuel || ev[0].Fleet != 1 {
		t.Errorf("events %v, want A out of fuel first", ev)
	}
}

func TestPredictionChaserRamScoopPerRound(t *testing.T) {
	// Fuel Mizer scouts 20 ly apart chasing each other at warp 4 (free, k 1).
	// A gains 4, 4, then 3 on arrival (trunc(4 − 0.99999)); B gains 4, 4 and
	// then stops. Over the whole year A would have gained 12.
	fm := fmEngines[2]
	g := scoutGame(fm, 1,
		Fleet{ID: 1, Stacks: []Stack{{0, 1}}, Pos: Point{1200, 1000}, Fuel: 100, Waypoints: []Waypoint{{Target: TargetFleet, ID: 2, Warp: 4}}},
		Fleet{ID: 2, Stacks: []Stack{{0, 1}}, Pos: Point{1220, 1000}, Fuel: 100, Waypoints: []Waypoint{{Target: TargetFleet, ID: 1, Warp: 4}}},
	)
	moveFleets(&g)
	if a, b := g.Fleets[0].Fuel, g.Fleets[1].Fuel; a != 111 || b != 108 {
		t.Errorf("fuel %d and %d, want 111 and 108", a, b)
	}
}

func TestPredictionChaserTopUp(t *testing.T) {
	// The FM-004 TU case as a chase of a stationary fleet 126 ly away: QJ5
	// scout, warp 9, fuel 102 moves 81, has 36 and is topped up to 37.
	qj5 := fmEngines[1]
	g := scoutGame(qj5, 1,
		Fleet{ID: 1, Stacks: []Stack{{0, 1}}, Pos: Point{0, 0}, Fuel: 102, Waypoints: []Waypoint{{Target: TargetFleet, ID: 2, Warp: 9}}},
		Fleet{ID: 2, Stacks: []Stack{{0, 1}}, Pos: Point{126, 0}},
	)
	moveFleets(&g)
	if f := g.Fleets[0]; f.Pos != (Point{81, 0}) || f.Fuel != 37 {
		t.Errorf("at %v fuel %d, want (81,0) and 37", f.Pos, f.Fuel)
	}
}

func TestPredictionResearchSwitchSameYear(t *testing.T) {
	// Energy 0 → 1 costs 50; the leftover 60 moves to biotech, whose level
	// 1 now costs 50 + 10 = 60, gained the same year. The explicit choice
	// then resets to "same field".
	s := AddResearch(ResearchState{Current: Energy, Next: Biotech}, pgRace(), 110, false)
	if s.Levels[Energy] != 1 || s.Levels[Biotech] != 1 || s.Accumulated[Biotech] != 0 || s.Current != Biotech || s.Next != NextSameField {
		t.Errorf("state %+v", s)
	}
	// Without a level-up, an explicit choice stays.
	s = AddResearch(ResearchState{Current: Energy, Next: Biotech}, pgRace(), 10, false)
	if s.Current != Energy || s.Next != Biotech {
		t.Errorf("no level-up: state %+v", s)
	}
}

func TestPredictionResearchLowestFieldStays(t *testing.T) {
	// Energy → 1 (50), leftover 60 to weapons (the lowest, first in field
	// order) → 1 (60), leftover 0 to propulsion, which cannot level.
	s := AddResearch(ResearchState{Current: Energy, Next: NextLowestField}, pgRace(), 110, false)
	if s.Levels[Energy] != 1 || s.Levels[Weapons] != 1 || s.Current != Propulsion || s.Next != NextLowestField {
		t.Errorf("state %+v", s)
	}
}

func TestPredictionGeneralizedResearchOnlyCurrentSwitches(t *testing.T) {
	r := pgRace()
	r.LRT.GeneralizedResearch = true
	s := ResearchState{Current: Energy, Next: Biotech}
	s.Accumulated[Weapons] = 50
	s = AddResearch(s, r, 0, false)
	if s.Levels[Weapons] != 1 || s.Current != Energy || s.Next != Biotech {
		t.Errorf("state %+v", s)
	}
}

func TestPredictionDepletionClampReevaluated(t *testing.T) {
	// Concentration 5 with 1/256 left (cc 25, need 1) drops to 4; cc is now
	// 10, so the next point needs 1250 and the remaining 199 leave
	// trunc(1051·256/1250) = 215 (a clamp fixed at 25 would leave 154).
	if got := deplete(Deposit{Concentration: 5, Fraction: 1}, 200); got != (Deposit{Concentration: 4, Fraction: 215}) {
		t.Errorf("deposit %+v, want conc 4 frac 215", got)
	}
}

func TestPredictionEmptyQueueSkipsTax(t *testing.T) {
	// An empty queue sends nothing to research, not even the tax; a queue
	// emptied this year is removed, so next year everything goes to research.
	in := ProductionInput{Colony: pgColony(), Resources: 100, GrownPop: 1100, ResearchBudget: 15}
	if r, _ := RunProduction(&Planet{HasQueue: true}, in); r != 0 {
		t.Errorf("empty queue: research %d, want 0", r)
	}
	p := &Planet{HasQueue: true, Queue: []QueueItem{q(ItemDefenses, 1, 0)}, Surface: Minerals{5, 5, 5}}
	RunProduction(p, in)
	if p.HasQueue {
		t.Fatal("emptied queue kept")
	}
	if r, _ := RunProduction(p, in); r != 100 {
		t.Errorf("next year: research %d, want 100", r)
	}
}

func TestPredictionEmptyPlanetNoDeaths(t *testing.T) {
	if p, c := GrowPopulation(0, 0, 1000, 10, -20); p != 0 || c != 0 {
		t.Errorf("empty hostile planet: (%d,%d), want (0,0)", p, c)
	}
	g := pgHomeworld()
	g.Planets[0].Env = [3]int{0, 0, 0}
	g.Planets[0].Population, g.Planets[0].GrowthCarry = 0, 0
	r, err := GenerateTurn(g, nil, Jrc3(), highRand{})
	if err != nil {
		t.Fatal(err)
	}
	if p := r.Game.Planets[0]; p.Population != 0 || p.GrowthCarry != 0 {
		t.Errorf("owned empty planet: (%d,%d), want (0,0)", p.Population, p.GrowthCarry)
	}
}

func TestPredictionCargoTiesKeepDesignOrder(t *testing.T) {
	// Two designs on one engine (f(6) = 100), 10 ly: tenths trunc((m+cargo)/2).
	// 1 kT goes to the first design in the fleet: light first → 1 + 10 = 11
	// tenths (2 mg); heavy first → 10 + 0 = 10 tenths (1 mg).
	var e Engine
	e.Fuel[6] = 100
	g := Game{Designs: []Design{
		{Name: "light", Mass: 1, Engine: e, Engines: 1, CargoCapacity: 10},
		{Name: "heavy", Mass: 20, Engine: e, Engines: 1, CargoCapacity: 10},
	}}
	light := Fleet{Stacks: []Stack{{0, 1}, {1, 1}}, Cargo: Cargo{Minerals: Minerals{1, 0, 0}}}
	heavy := Fleet{Stacks: []Stack{{1, 1}, {0, 1}}, Cargo: Cargo{Minerals: Minerals{1, 0, 0}}}
	if a, b := g.FuelCost(&light, 6, 10), g.FuelCost(&heavy, 6, 10); a != 2 || b != 1 {
		t.Errorf("light first %d mg, heavy first %d mg, want 2 and 1", a, b)
	}
}

func TestPredictionMiningDrawOrder(t *testing.T) {
	// Planets in id order, ironium, boranium, germanium: planet 1 takes the
	// first three draws though it is second in the slice.
	g := pgHomeworld()
	one := g.Planets[0]
	for m := range NumMinerals {
		one.Deposits[m] = Deposit{Concentration: 113, Fraction: 0} // remainder 30
	}
	one.Surface = Minerals{}
	two := one
	one.ID, two.ID = 1, 2
	two.Pos = Point{1, 1}
	g.Planets = []Planet{two, one}
	r, err := GenerateTurn(g, nil, Jrc3(), &seqRand{draws: []int{0, 0, 99, 99, 99, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Game.Planets[1].Surface; got != (Minerals{12, 12, 11}) {
		t.Errorf("planet 1 surface %v, want [12 12 11]", got)
	}
	if got := r.Game.Planets[0].Surface; got != (Minerals{11, 11, 12}) {
		t.Errorf("planet 2 surface %v, want [11 11 12]", got)
	}
}

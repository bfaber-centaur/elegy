package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// caView is a small Rototill view: homeworld 1 at (0,0) with a starbase,
// planets 2..5 on the x axis, Scout (slot 0), Colony Ship (slot 1) and
// Midget Miner (slot 2) designs, all with Long Hump 6.
func caView(t *testing.T, year int) *View {
	t.Helper()
	cat := engine.Components()
	mk := func(name, hull string, fills ...engine.SlotFill) engine.Design {
		d, err := cat.NewDesign(name, hull, fills)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	// AI.md §3, CA expert: 800 colonists per resource, factories
	// 15/10/15, mines 15/5/15.
	race := engine.Race{PRT: engine.PRTClaimAdjuster, GrowthRate: 15, ColonistsPerResource: 800,
		FactoryOutput: 15, FactoryCost: 10, FactoriesOperated: 15, MineOutput: 15, MineCost: 5, MinesOperated: 15}
	race.Env = [3]engine.EnvRange{{Immune: true}, {Center: 50, Low: 24, High: 76}, {Center: 50, Low: 25, High: 75}}
	v := &View{
		Year: year, Player: 0, Level: Expert, Rules: engine.ElegyRules(),
		Self: engine.Player{Race: race, Research: engine.ResearchState{Levels: [engine.NumFields]int{3, 3, 3, 3, 3, 0}, Current: engine.Biotech}},
		Ships: []Design{
			{Slot: 0, Index: 10, Design: mk("Scout", "Scout", fill(0, "Long Hump 6", 1), fill(1, "Bat Scanner", 1)), Created: 2400},
			{Slot: 1, Index: 11, Design: mk("Colony", "Colony Ship", fill(0, "Long Hump 6", 1), fill(1, "Colonization Module", 1)), Created: 2400},
			{Slot: 2, Index: 12, Design: mk("Miner", "Midget Miner", fill(0, "Long Hump 6", 1), fill(1, "Robo-Midget Miner", 2)), Created: 2400},
		},
		Known: map[int]engine.PlanetReport{},
		PRT:   map[int]engine.PRT{},
	}
	station, fort := mk("Station", "Space Station"), mk("Fort", "Orbital Fort")
	for k := range 5 {
		d := station
		if k == 1 || k == 3 {
			d = fort
		}
		d.Name = starbaseNames[k]
		v.Starbases = append(v.Starbases, Design{Slot: k, Index: 20 + k, Design: d, Created: 2400, Picture: k / 2})
	}
	for i := 1; i <= 5; i++ {
		v.Universe = append(v.Universe, PlanetPos{ID: i, Pos: engine.Point{X: 1000 + 30*(i-1), Y: 1000}})
	}
	v.Planets = []engine.Planet{{ID: 1, Pos: v.Universe[0].Pos, Owner: 0, Homeworld: true, Env: [3]int{50, 50, 50},
		HasStarbase: true, StarbaseDesign: 20, Population: 1200, HasQueue: true}}
	v.Known[1] = engine.PlanetReport{Planet: 1, Level: engine.ReportOwn, Pos: v.Universe[0].Pos, Owner: 0, Env: [3]int{50, 50, 50}}
	return v
}

func fleet(id, number int, pos engine.Point, design, n int) engine.Fleet {
	return engine.Fleet{ID: id, Number: number, Owner: 0, Pos: pos, Stacks: []engine.Stack{{Design: design, Count: n}}, Fuel: 100}
}

func ordersOf[T engine.Order](orders []engine.Order) []T {
	var out []T
	for _, o := range orders {
		if x, ok := o.(T); ok {
			out = append(out, x)
		}
	}
	return out
}

// Pass 2 step 1: a colony ship at a home with 5,000 colonists loads 2,500
// and goes to the nearest known planet habitable after terraforming at
// its ideal warp (Long Hump 6: 6); a planet chosen is skipped by the next
// colony ship.
func TestRototillColonyShips(t *testing.T) {
	v := caView(t, 2401)
	home := v.Universe[0].Pos
	v.Fleets = []engine.Fleet{fleet(100, 1, home, 11, 1), fleet(101, 2, home, 11, 1)}
	v.Known[2] = engine.PlanetReport{Planet: 2, Level: engine.ReportPosition, Owner: engine.NoOwner}                        // no environment
	v.Known[3] = engine.PlanetReport{Planet: 3, Level: engine.ReportNormal, Owner: engine.NoOwner, Env: [3]int{50, 99, 50}} // hostile even after terraforming
	v.Known[4] = engine.PlanetReport{Planet: 4, Level: engine.ReportNormal, Owner: engine.NoOwner, Env: [3]int{10, 60, 40}}
	v.Known[5] = engine.PlanetReport{Planet: 5, Level: engine.ReportNormal, Owner: engine.NoOwner, Env: [3]int{10, 50, 50}}
	res := PlayRototill(v, top{})
	loads := ordersOf[engine.CargoOrder](res.Orders)
	if len(loads) != 2 || loads[0].Amounts[engine.CargoColonists] != 25 || loads[0].ID != 1 {
		t.Errorf("loads %+v", loads)
	}
	wps := ordersOf[engine.WaypointOrder](res.Orders)
	if len(wps) != 2 || wps[0].Waypoints[0].ID != 4 || wps[1].Waypoints[0].ID != 5 {
		t.Fatalf("waypoints %+v", wps)
	}
	if w := wps[0].Waypoints[0]; w.Warp != 6 || w.Task.Kind != engine.TaskColonize {
		t.Errorf("colonize waypoint %+v", w)
	}
}

// Pass 2 step 1, no colonists and away from home: an engine of Fuel Mizer
// or later goes to the nearest own starbase at warp 4.
func TestRototillEmptyColonyShipGoesHome(t *testing.T) {
	v := caView(t, 2401)
	v.Fleets = []engine.Fleet{fleet(100, 1, v.Universe[2].Pos, 11, 1)}
	res := PlayRototill(v, top{})
	wps := ordersOf[engine.WaypointOrder](res.Orders)
	if len(wps) != 1 || wps[0].Waypoints[0].ID != 1 || wps[0].Waypoints[0].Warp != 4 || wps[0].Waypoints[0].Task.Kind != engine.TaskNone {
		t.Errorf("orders %+v", wps)
	}
}

// Pass 1: a colony ship carrying colonists to a planet another player now
// owns invades it when habitable for Rototill; otherwise its route is cut.
func TestRototillPassOne(t *testing.T) {
	v := caView(t, 2401)
	f := fleet(100, 1, v.Universe[0].Pos, 11, 1)
	f.Cargo.Colonists = 25
	f.Waypoints = []engine.Waypoint{toPlanet(4, v.Universe[3].Pos, 6, engine.Task{Kind: engine.TaskColonize})}
	g := fleet(101, 2, v.Universe[0].Pos, 11, 1)
	g.Waypoints = []engine.Waypoint{toPlanet(3, v.Universe[2].Pos, 6, engine.Task{Kind: engine.TaskColonize})}
	v.Fleets = []engine.Fleet{f, g}
	v.Known[4] = engine.PlanetReport{Planet: 4, Level: engine.ReportNormal, Owner: 1, Env: [3]int{10, 60, 40}}
	v.Known[3] = engine.PlanetReport{Planet: 3, Level: engine.ReportNormal, Owner: 1, Env: [3]int{50, 99, 50}}
	res := PlayRototill(v, top{})
	wps := ordersOf[engine.WaypointOrder](res.Orders)
	if len(wps) != 2 {
		t.Fatalf("orders %+v", wps)
	}
	if tk := wps[0].Waypoints[0].Task; tk.Kind != engine.TaskTransport || tk.Transport[engine.CargoColonists].Action != engine.UnloadAll {
		t.Errorf("invasion %+v", wps[0])
	}
	if wps[1].Fleet != 101 || len(wps[1].Waypoints) != 0 || wps[1].Task.Kind != engine.TaskNone {
		t.Errorf("cut %+v", wps[1])
	}
}

// §2: the first own planet with a starbase and 100,000 colonists queues a
// Colony Ship when none is alive; never in year index 0.
func TestRototillProduction(t *testing.T) {
	for _, c := range []struct {
		year, pop int
		alive     bool
		want      bool
	}{
		{2401, 1000, false, true},
		{2400, 1000, false, false},
		{2401, 999, false, false},
		{2401, 1000, true, false},
	} {
		v := caView(t, c.year)
		v.Planets[0].Population = c.pop
		if c.alive {
			v.Fleets = []engine.Fleet{fleet(100, 1, v.Universe[3].Pos, 11, 1)}
		}
		res := PlayRototill(v, top{})
		got := false
		for _, q := range ordersOf[engine.QueueOrder](res.Orders) {
			for _, it := range q.Queue {
				if it.Kind == engine.ItemShip {
					got = got || it == (engine.QueueItem{Kind: engine.ItemShip, Slot: 1, Count: 1})
				}
			}
		}
		if got != c.want {
			t.Errorf("%+v: colony ship queued %v", c, got)
		}
	}
}

// Pass 2 step 4: the scout goes to the nearest planet never seen that no
// other own fleet targets, at its ideal warp, with no task.
func TestRototillScout(t *testing.T) {
	v := caView(t, 2401)
	home := v.Universe[0].Pos
	v.Known[2] = engine.PlanetReport{Planet: 2, Level: engine.ReportPosition, Owner: engine.NoOwner}
	other := fleet(101, 2, v.Universe[3].Pos, 12, 1)
	other.Waypoints = []engine.Waypoint{toPlanet(3, v.Universe[2].Pos, 5, engine.Task{})}
	v.Fleets = []engine.Fleet{fleet(100, 1, home, 10, 1), other}
	res := PlayRototill(v, top{})
	wps := ordersOf[engine.WaypointOrder](res.Orders)
	if len(wps) != 1 || wps[0].Fleet != 100 || wps[0].Waypoints[0].ID != 4 || wps[0].Waypoints[0].Warp != 6 {
		t.Errorf("orders %+v", wps)
	}
}

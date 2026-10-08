package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// designCost agrees with the engine's owner cost: a starbase's build cost
// from it, halved as COMPONENTS.md "Starbases" says, is the engine's.
func TestDesignCostMatchesEngine(t *testing.T) {
	v := caView(t, 2401)
	for _, d := range v.Starbases {
		c := designCost(d.Design, v.Self.Race, v.Self.Research.Levels)
		half := func(x int) int { return (x + 1) / 2 }
		got := engine.Cost{Resources: half(c.Resources), Minerals: engine.Minerals{half(c.Minerals[0]), half(c.Minerals[1]), half(c.Minerals[2])}}
		if want := engine.StarbaseBuildCost(d.Design, v.Self.Race, v.Self.Research.Levels); got != want {
			t.Errorf("%s: %+v, engine %+v", d.Design.Name, got, want)
		}
	}
}

func homeAutomation(t *testing.T, v *View, rng engine.Rand) (*automation, *engine.Planet) {
	t.Helper()
	a := &automation{v: v, pers: Rototill, rng: rng, q: &queues{v: v}, order: []int{1}, marked: map[int]bool{}}
	return a, v.ownPlanet(1)
}

// Mines and factories: with minerals to spare, factories go to the back
// and mines to the front, each limited by room and resources.
func TestMinesAndFactories(t *testing.T) {
	v := caView(t, 2401)
	p := &v.Planets[0]
	p.Population, p.Mines, p.Factories = 1000, 10, 10 // 100,000 colonists
	p.Surface = engine.Minerals{500, 500, 500}
	p.HasQueue, p.Queue = true, nil
	a, p := homeAutomation(t, v, top{})
	a.minesAndFactories(p)
	q := a.q.get(p)
	// Resources: 100,000/800 + 15·10/10 = 140. Rooms: 150 − 10 = 140 each.
	// Factories: min(140, Ge 500/4, 140/10) = 14, costing all 140; mines
	// then have no resources left.
	want := []engine.QueueItem{{Kind: engine.ItemFactory, Count: 14}}
	if len(q) != len(want) || q[0] != want[0] {
		t.Errorf("queue %+v, want %+v", q, want)
	}
	// Starved of a mineral: mines at the front with all the resources.
	p.Surface = engine.Minerals{500, 500, 0}
	p.Homeworld = false // no concentration-30 floor: nothing mined
	a, p = homeAutomation(t, v, top{})
	a.minesAndFactories(p)
	if q := a.q.get(p); len(q) != 1 || q[0] != (engine.QueueItem{Kind: engine.ItemMine, Count: 28}) {
		t.Errorf("starved queue %+v", q)
	}
}

// Starbase upgrade in the current family: Random(100) ≤ 5 and every
// surface mineral ≥ 200 queue the next variant (slot 0 → 2).
func TestStarbaseUpgrade(t *testing.T) {
	v := caView(t, 2401)
	p := &v.Planets[0]
	p.Surface = engine.Minerals{200, 200, 200}
	a, p := homeAutomation(t, v, &script{t: t, draws: []int{5}})
	if !a.upgrade(p) || a.q.get(p)[0] != (engine.QueueItem{Kind: engine.ItemStarbase, Slot: 2, Count: 1}) {
		t.Errorf("queue %+v", a.q.get(p))
	}
	a, p = homeAutomation(t, v, &script{t: t, draws: []int{6}})
	if a.upgrade(p) {
		t.Error("draw 6 upgraded")
	}
}

// Defenses: 160,000 colonists and fewer defenses than population/8,000
// add min(room, 4).
func TestDefenses(t *testing.T) {
	v := caView(t, 2401)
	p := &v.Planets[0]
	p.Population, p.Defenses = 1600, 10
	a, p := homeAutomation(t, v, top{})
	if !a.defenses(p) || a.q.get(p)[0] != (engine.QueueItem{Kind: engine.ItemDefenses, Count: 4}) {
		t.Errorf("queue %+v", a.q.get(p))
	}
	p.Defenses = 20
	a, p = homeAutomation(t, v, top{})
	if a.defenses(p) {
		t.Error("defenses at population/8,000 still queued")
	}
}

// Hubs from year index 20: starbase planets, then rich developed planets.
func TestHubs(t *testing.T) {
	v := caView(t, 2420)
	v.Planets = append(v.Planets,
		engine.Planet{ID: 2, Pos: v.Universe[1].Pos, Owner: 0, Population: 80, Mines: 20, Factories: 20, Surface: engine.Minerals{7000, 0, 0}},
		engine.Planet{ID: 3, Pos: v.Universe[2].Pos, Owner: 0, Population: 79, Mines: 20, Factories: 20, Surface: engine.Minerals{7000, 0, 0}})
	if h := v.hubs(Rototill, []int{3, 2, 1}); len(h) != 2 || h[0] != 1 || h[1] != 2 {
		t.Errorf("hubs %v", h)
	}
	if h := v.hubs(Robotoid, []int{3, 2, 1}); len(h) != 1 {
		t.Errorf("Robotoid hubs %v: planet 2 is within 50 ly of hub 1", h)
	}
	v.Year = 2419
	if h := v.hubs(Rototill, []int{1}); h != nil {
		t.Errorf("year index 19 hubs %v", h)
	}
}

package ai

import (
	"slices"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// AI.md §7 step 4, "Under attack", at a homeworld with 300 resources,
// room for 48 defenses and a bomber in orbit: n = 300/25 = 12, less 12/6
// = 10; r = 270. With 322 kT of ironium or more available m = 64 ≥ n, so
// 10 defenses go to the front (A61). With 22 kT, m = 4 < n: extra =
// (270 − 4·100)/150 → 0, so 4 defenses; for a Mineral Alchemy race
// (270 − 4·25)/150 = 1, so 5 defenses and 5 alchemy in front of them.
// The other cases each separate one detail:
//   - 251 resources (67 factories): n = 10 − 10/6 = 9 (10/5 would give 8);
//   - 20 mines mine 9 kT, so m = 1; Mineral Alchemy: extra = (270 −
//     25)/150 = 1 (a divisor of 100 would give 2);
//   - 44 defenses installed leave room for 4, so n = 4;
//   - n = m: 10,000 colonists and 1,500 factories give 3,500 resources, n
//     = 140 − 23 = 117, capped at the room of 100, and m = 100, so 100
//     defenses (with n < m the Mineral Alchemy branch would add (3,150 −
//     2,500)/150 = 4 more and 20 alchemy);
//   - a fully known foreign design with no bomb part, an own bomber, or a
//     bomber known only partly: nothing.
func TestUnderAttack(t *testing.T) {
	cat := engine.Components()
	bomber, err := cat.NewDesign("B", "B-17 Bomber", []engine.SlotFill{fill(0, "Quick Jump 5", 2), fill(1, "Lady Finger Bomb", 1)})
	if err != nil {
		t.Fatal(err)
	}
	scout, err := cat.NewDesign("S", "Scout", []engine.SlotFill{fill(0, "Quick Jump 5", 1), fill(1, "Bat Scanner", 1)})
	if err != nil {
		t.Fatal(err)
	}
	def := func(n int) engine.QueueItem { return engine.QueueItem{Kind: engine.ItemDefenses, Count: n} }
	alch := func(n int) engine.QueueItem { return engine.QueueItem{Kind: engine.ItemMineralAlchemy, Count: n} }
	type setup struct {
		pop, factories, mines, defenses int
	}
	base := setup{1200, 100, 50, 0}
	for _, c := range []struct {
		name    string
		known   *engine.Design // the orbiting design, when known in full
		owner   int            // the orbiting fleet's owner
		ironium int
		ma      bool
		queued  bool // defenses already queued
		set     setup
		want    []engine.QueueItem
	}{
		{"minerals enough", &bomber, 1, 300, false, false, base, []engine.QueueItem{def(10)}},
		{"short of ironium", &bomber, 1, 0, false, false, base, []engine.QueueItem{def(4)}},
		{"short, Mineral Alchemy", &bomber, 1, 0, true, false, base, []engine.QueueItem{alch(5), def(5)}},
		{"design not known in full", nil, 1, 300, false, false, base, nil},
		{"defenses already queued", &bomber, 1, 300, false, true, base, []engine.QueueItem{def(1)}},
		{"no bomb part", &scout, 1, 300, false, false, base, nil},
		{"own bomber", &bomber, 0, 300, false, false, base, nil},
		{"n/6", &bomber, 1, 300, false, false, setup{1200, 67, 50, 0}, []engine.QueueItem{def(9)}},
		{"extra/150", &bomber, 1, 0, true, false, setup{1200, 100, 20, 0}, []engine.QueueItem{alch(5), def(2)}},
		{"defense room", &bomber, 1, 300, false, false, setup{1200, 100, 50, 44}, []engine.QueueItem{def(4)}},
		{"n = m", &bomber, 1, 1000, true, false, setup{10000, 1500, 50, 0}, []engine.QueueItem{def(100)}},
	} {
		v := caView(t, 2430)
		p := &v.Planets[0]
		p.Population, p.Factories, p.Mines, p.Defenses = c.set.pop, c.set.factories, c.set.mines, c.set.defenses
		p.Surface = engine.Minerals{c.ironium, 1000, 1000}
		v.Self.Race.LRT.MineralAlchemy = c.ma
		if c.queued {
			p.Queue = []engine.QueueItem{def(1)}
		}
		if c.known != nil {
			v.Foreign = map[int]engine.Design{50: *c.known}
		}
		v.Others = []engine.FleetSighting{{Fleet: 900, Owner: c.owner, Pos: p.Pos, Stacks: []engine.Stack{{Design: 50, Count: 1}}}}
		a := &automation{v: v, res: &Result{}, q: &queues{v: v}, rng: top{}}
		a.underAttack(p)
		if got := a.q.get(p); !slices.Equal(got, c.want) && (len(got) != 0 || len(c.want) != 0) {
			t.Errorf("%s: queue %+v, want %+v", c.name, got, c.want)
		}
	}
}

// Step 4 runs from year index (universe size + 2)·10: caView's universe
// spans 120 ly, size 0, so from year index 20. A bombed planet gets no
// defenses from the automation in year index 19 and does in 20 (the
// planet is too small for step 3's defenses).
func TestUnderAttackGate(t *testing.T) {
	cat := engine.Components()
	bomber, err := cat.NewDesign("B", "B-17 Bomber", []engine.SlotFill{fill(0, "Quick Jump 5", 2), fill(1, "Lady Finger Bomb", 1)})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		y    int
		want bool
	}{{19, false}, {20, true}} {
		v := caView(t, FirstYear+c.y)
		p := &v.Planets[0]
		p.Factories, p.Mines = 100, 50
		p.Surface = engine.Minerals{300, 1000, 1000}
		v.Foreign = map[int]engine.Design{50: bomber}
		v.Others = []engine.FleetSighting{{Fleet: 900, Owner: 1, Pos: p.Pos, Stacks: []engine.Stack{{Design: 50, Count: 1}}}}
		a := &automation{v: v, res: &Result{}, pers: Rototill, q: &queues{v: v}, rng: top{}, order: []int{p.ID}}
		a.run()
		if got := a.q.has(p, engine.ItemDefenses); got != c.want {
			t.Errorf("year index %d: defenses queued %v, want %v (queue %+v)", c.y, got, c.want, a.q.get(p))
		}
	}
}

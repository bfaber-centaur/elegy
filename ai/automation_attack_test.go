package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// AI.md §7 step 4, "Under attack", at a homeworld with 300 resources,
// room for 48 defenses and a bomber in orbit: n = 300/25 = 12, less 12/6
// = 10; r = 270. With 322 kT of ironium or more available m = 64 ≥ n, so
// 10 defenses go to the front (A61). With 22 kT, m = 4 < n: extra =
// (270 − 4·100)/150 → 0, so 4 defenses; for a Mineral Alchemy race
// (270 − 4·25)/150 = 1, so 5 defenses and 5 alchemy in front of them.
func TestUnderAttack(t *testing.T) {
	cat := engine.Components()
	bomber, err := cat.NewDesign("B", "B-17 Bomber", []engine.SlotFill{fill(0, "Quick Jump 5", 2), fill(1, "Lady Finger Bomb", 1)})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name    string
		known   bool // the bomber's design known in full
		ironium int
		ma      bool
		queued  bool // defenses already queued
		want    []engine.QueueItem
	}{
		{"minerals enough", true, 300, false, false, []engine.QueueItem{{Kind: engine.ItemDefenses, Count: 10}}},
		{"short of ironium", true, 0, false, false, []engine.QueueItem{{Kind: engine.ItemDefenses, Count: 4}}},
		{"short, Mineral Alchemy", true, 0, true, false, []engine.QueueItem{{Kind: engine.ItemMineralAlchemy, Count: 5}, {Kind: engine.ItemDefenses, Count: 5}}},
		{"design not known in full", false, 300, false, false, nil},
		{"defenses already queued", true, 300, false, true, []engine.QueueItem{{Kind: engine.ItemDefenses, Count: 1}}},
	} {
		v := caView(t, 2430)
		p := &v.Planets[0]
		p.Factories, p.Mines = 100, 50
		p.Surface = engine.Minerals{c.ironium, 1000, 1000}
		v.Self.Race.LRT.MineralAlchemy = c.ma
		if c.queued {
			p.Queue = []engine.QueueItem{{Kind: engine.ItemDefenses, Count: 1}}
		}
		if c.known {
			v.Foreign = map[int]engine.Design{50: bomber}
		}
		v.Others = []engine.FleetSighting{{Fleet: 900, Owner: 1, Pos: p.Pos, Stacks: []engine.Stack{{Design: 50, Count: 1}}}}
		a := &automation{v: v, res: &Result{}, q: &queues{v: v}, rng: top{}}
		a.underAttack(p)
		got := a.q.get(p)
		if len(got) != len(c.want) {
			t.Errorf("%s: queue %+v, want %+v", c.name, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: queue %+v, want %+v", c.name, got, c.want)
				break
			}
		}
	}
}

package ai

import (
	"slices"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// The completion estimate (ESTIMATES.md "Production completion") at the
// CA homeworld of caView with 100 factories and 50 mines: 300 resources a
// year with no research budget, and 22 kT of each mineral mined a year
// (concentration floored at 30). Defenses cost 15 resources and 5 kT of
// each mineral, so 40 of them with ample minerals take 2 years and one
// takes 1; with half the resources sent to research they take 4, but
// still 2 when the planet sends only leftover resources to research. With
// no mining and no germanium a factory never finishes
// (100); behind Auto Alchemy it buys its germanium and the three finish
// in year 4.
func TestCompletion(t *testing.T) {
	for _, c := range []struct {
		name    string
		surface engine.Minerals
		noMines bool
		budget  int
		left    bool // the planet sends only leftover resources to research
		q       []engine.QueueItem
		want    func(int) bool
	}{
		{"one year", engine.Minerals{1000, 1000, 1000}, false, 0, false, []engine.QueueItem{{Kind: engine.ItemDefenses, Count: 1}}, func(y int) bool { return y == 1 }},
		{"two years", engine.Minerals{1000, 1000, 1000}, false, 0, false, []engine.QueueItem{{Kind: engine.ItemDefenses, Count: 40}}, func(y int) bool { return y == 2 }},
		{"half to research", engine.Minerals{1000, 1000, 1000}, false, 50, false, []engine.QueueItem{{Kind: engine.ItemDefenses, Count: 40}}, func(y int) bool { return y == 4 }},
		{"leftover only", engine.Minerals{1000, 1000, 1000}, false, 50, true, []engine.QueueItem{{Kind: engine.ItemDefenses, Count: 40}}, func(y int) bool { return y == 2 }},
		{"never", engine.Minerals{100, 100, 0}, true, 0, false, []engine.QueueItem{{Kind: engine.ItemFactory, Count: 3}}, func(y int) bool { return y == 100 }},
		{"alchemy", engine.Minerals{100, 100, 0}, true, 0, false, []engine.QueueItem{{Kind: engine.ItemAutoAlchemy, Count: 1}, {Kind: engine.ItemFactory, Count: 3}}, func(y int) bool { return y == 4 }},
	} {
		v := caView(t, 2430)
		p := &v.Planets[0]
		p.Factories, p.Mines = 100, 50
		p.Surface = c.surface
		if c.noMines {
			p.Homeworld = false
		}
		p.LeftoverOnly = c.left
		if got := v.completion(p, c.q, c.budget); !c.want(got) {
			t.Errorf("%s: completion %d", c.name, got)
		}
	}
}

// AI.md §7 step 5, "Blocked queues", at the same homeworld (mine room
// 180 − 50 = 130, so m = min(130, 300/5) = 60):
//   - Defenses ×100 with no surface minerals finish in year 23 on 22 kT a
//     year; 60 mines first raise the mining to 49 kT a year (year 11),
//     which beats buying a few kT a year with Auto Alchemy (year 19), so
//     the mines go in front;
//   - Factory ×3 with no germanium and no mining never finishes, with or
//     without mines (100), and in year 4 behind Auto Alchemy, so Auto
//     Alchemy goes in front;
//   - with no mine room (m < 1) Auto Alchemy goes in front unconditionally;
//   - 120 mines queued behind the head leave room for 10, so 10 mines;
//   - Factory ×100 with ample minerals finishes in year 3 but costs 1000
//     resources, more than (3 − 1) × 300, so resources hold it: unchanged,
//     even with no mine room;
//   - a head finishing next year, and a head of mines: unchanged.
//
// The ties: Defenses ×2 with 10 mines and no ironium or germanium finish
// in year 3, and in year 2 both behind 60 mines and behind Auto Alchemy,
// so the mines go in; Defenses ×5 with 50 mines finish in year 2 either
// way, so nothing goes in; Auto Factories ×26 with 20 factories, 120
// mines (m = min(60, 180/5) = 36) and no ironium finish in year 3 as they
// are and behind Auto Alchemy, and in year 4 behind the mines, so nothing
// goes in.
func TestBlockedQueue(t *testing.T) {
	def := func(n int) engine.QueueItem { return engine.QueueItem{Kind: engine.ItemDefenses, Count: n} }
	mine := func(n int) engine.QueueItem { return engine.QueueItem{Kind: engine.ItemMine, Count: n} }
	fact := func(n int) engine.QueueItem { return engine.QueueItem{Kind: engine.ItemFactory, Count: n} }
	alch := engine.QueueItem{Kind: engine.ItemAutoAlchemy, Count: 1}
	autoFact := engine.QueueItem{Kind: engine.ItemAutoFactories, Count: 26}
	q := func(it ...engine.QueueItem) []engine.QueueItem { return it }
	for _, c := range []struct {
		name      string
		surface   engine.Minerals
		noMines   bool
		factories int
		mines     int
		queue     []engine.QueueItem
		want      []engine.QueueItem
	}{
		{"mines help", engine.Minerals{}, false, 100, 50, q(def(100)), q(mine(60), def(100))},
		{"only alchemy helps", engine.Minerals{100, 100, 0}, true, 100, 50, q(fact(3)), q(alch, fact(3))},
		{"no mine room", engine.Minerals{}, false, 100, 180, q(def(100)), q(alch, def(100))},
		{"mines queued", engine.Minerals{}, false, 100, 50, q(def(100), mine(120)), q(mine(10), def(100), mine(120))},
		{"resources hold it", engine.Minerals{1000, 1000, 1000}, false, 100, 180, q(fact(100)), q(fact(100))},
		{"next year", engine.Minerals{1000, 1000, 1000}, false, 100, 50, q(def(1)), q(def(1))},
		{"mines head", engine.Minerals{}, false, 100, 50, q(mine(5)), q(mine(5))},
		{"mines tie alchemy", engine.Minerals{0, 1000, 0}, false, 100, 10, q(def(2)), q(mine(60), def(2))},
		{"mines no sooner", engine.Minerals{0, 1000, 0}, false, 100, 50, q(def(5)), q(def(5))},
		{"alchemy no sooner", engine.Minerals{0, 1000, 200}, false, 20, 120, q(autoFact), q(autoFact)},
	} {
		v := caView(t, 2430)
		p := &v.Planets[0]
		p.Factories, p.Mines = c.factories, c.mines
		p.Surface = c.surface
		if c.noMines {
			p.Homeworld = false
		}
		p.Queue = c.queue
		a := &automation{v: v, res: &Result{}, q: &queues{v: v}, rng: top{}}
		a.blockedQueue(p)
		if got := a.q.get(p); !slices.Equal(got, c.want) {
			t.Errorf("%s: queue %+v, want %+v", c.name, got, c.want)
		}
	}
}

// Step 5 runs in automation.run before the mines-and-factories fill: the
// germanium-starved Factory ×3 of TestBlockedQueue gets Auto Alchemy in
// front, and the fill then adds nothing, since the head is Auto Alchemy.
func TestBlockedQueueRun(t *testing.T) {
	v := caView(t, 2430)
	p := &v.Planets[0]
	p.Factories, p.Mines, p.Homeworld = 100, 50, false
	p.Surface = engine.Minerals{100, 100, 0}
	fact := engine.QueueItem{Kind: engine.ItemFactory, Count: 3}
	p.Queue = []engine.QueueItem{fact}
	a := &automation{v: v, res: &Result{}, pers: Rototill, q: &queues{v: v}, rng: top{}, order: []int{p.ID}}
	a.run()
	want := []engine.QueueItem{{Kind: engine.ItemAutoAlchemy, Count: 1}, fact}
	if got := a.q.get(p); !slices.Equal(got, want) {
		t.Errorf("queue %+v, want %+v", got, want)
	}
}

// The heads step 5 passes over, and its year-1 test, each at a planet
// with no mine room, where a head that passed would get Auto Alchemy
// in front unconditionally:
//   - at three times capacity (30,000 colonists, 2,650 resources a year,
//     falling as the population shrinks), Mine ×1030, Mineral Alchemy ×52
//     and Terraform ×103 finish in year 3 at a cost within (3 − 1) ×
//     2,650, and Auto Mines ×531 and Auto Min/Max Terraform ×54 never
//     finish;
//   - an Auto Alchemy head (here in front of a germanium-starved Factory
//     ×3) would otherwise get a second one every year;
//   - a ship item whose slot holds no design costs nothing and finishes in
//     year 1, so the cost test alone would let it through.
func TestBlockedQueueSkips(t *testing.T) {
	over := func(k engine.ItemKind, n int) engine.QueueItem { return engine.QueueItem{Kind: k, Count: n} }
	alch := engine.QueueItem{Kind: engine.ItemAutoAlchemy, Count: 1}
	fact := engine.QueueItem{Kind: engine.ItemFactory, Count: 3}
	for _, c := range []struct {
		name  string
		pop   int
		queue []engine.QueueItem
	}{
		{"mines", 30000, []engine.QueueItem{over(engine.ItemMine, 1030)}},
		{"auto mines", 30000, []engine.QueueItem{over(engine.ItemAutoMines, 531)}},
		{"mineral alchemy", 30000, []engine.QueueItem{over(engine.ItemMineralAlchemy, 52)}},
		{"terraform", 30000, []engine.QueueItem{over(engine.ItemTerraform, 103)}},
		{"auto min terraform", 30000, []engine.QueueItem{over(engine.ItemAutoMinTerraform, 54)}},
		{"auto max terraform", 30000, []engine.QueueItem{over(engine.ItemAutoMaxTerraform, 54)}},
		{"auto alchemy", 1200, []engine.QueueItem{alch, fact}},
		{"no design", 1200, []engine.QueueItem{{Kind: engine.ItemShip, Count: 1, Slot: 9}}},
	} {
		v := caView(t, 2430)
		p := &v.Planets[0]
		p.Factories, p.Population = 100, c.pop
		col := engine.NewColony(p, &v.Self)
		p.Mines = col.OperableMines(p.Population) // no mine room
		p.Surface = engine.Minerals{100, 100, 0}
		if c.name == "auto alchemy" {
			p.Homeworld = false
		}
		p.Queue = c.queue
		a := &automation{v: v, res: &Result{}, q: &queues{v: v}, rng: top{}}
		a.blockedQueue(p)
		if got := a.q.get(p); !slices.Equal(got, c.queue) {
			t.Errorf("%s: queue %+v, want it unchanged", c.name, got)
		}
	}
}

// Step 5 at a planet that sends only leftover resources to research
// takes its resources without the research share, as completion does:
// with a 50% budget, Defenses ×100 of TestBlockedQueue still get m =
// min(130, 300/5) = 60 mines, not 150/5 = 30.
func TestBlockedQueueLeftoverOnly(t *testing.T) {
	v := caView(t, 2430)
	p := &v.Planets[0]
	p.Factories, p.Mines, p.LeftoverOnly = 100, 50, true
	def := engine.QueueItem{Kind: engine.ItemDefenses, Count: 100}
	p.Queue = []engine.QueueItem{def}
	a := &automation{v: v, res: &Result{}, q: &queues{v: v}, rng: top{}, budget: 50}
	a.blockedQueue(p)
	want := []engine.QueueItem{{Kind: engine.ItemMine, Count: 60}, def}
	if got := a.q.get(p); !slices.Equal(got, want) {
		t.Errorf("queue %+v, want %+v", got, want)
	}
}

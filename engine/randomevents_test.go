package engine

import (
	"reflect"
	"testing"
)

// Random event vectors from stars-elegy KERNEL.md "Random events"
// (CONFIRMED, KX-004), replayed with the draw order KERNEL.md gives.

func eventGame(yearIndex, owner, pop int) Game {
	p := Planet{ID: 1, Owner: owner, Population: pop, Env: [3]int{50, 50, 50}, OrigEnv: [3]int{50, 50, 50}, HasQueue: true,
		Queue: []QueueItem{{Kind: ItemAutoFactories, Count: 5}, {Kind: ItemFactory, Count: 3}, {Kind: ItemMine, Count: 2}}}
	for m := range NumMinerals {
		p.Deposits[m].Concentration = 112
	}
	return Game{Rules: ElegyRules(), Year: 2400 + yearIndex, RandomEvents: true, Players: make([]Player, 2), Planets: []Planet{p}}
}

func TestConfirmedCometStrike(t *testing.T) {
	// Population killed 25/45/65/85%: small 9237 → 6928, medium 9237 →
	// 5081, large 8110 → 2839, huge 9237 → 1386.
	for _, c := range []struct{ size, pop, want int }{{0, 9237, 6928}, {1, 9237, 5081}, {2, 8110, 2839}, {3, 9237, 1386}} {
		g := eventGame(30, 0, c.pop)
		g.randomEvents(&seqRand{draws: []int{0, 0, c.size}})
		if got := g.Planets[0].Population; got != c.want {
			t.Errorf("size %d: %d, want %d", c.size, got, c.want)
		}
	}

	// S2 (LEGACY BUG): a small comet moves gravity +4 while the owner's
	// message names radiation. Struck ironium: +3000 base → 3050/16 = 190
	// kT, concentration +50; unstruck minerals 50/16 = 3 kT. The queue
	// keeps only Auto Factories ×5.
	g := eventGame(30, 0, 9237)
	ev := g.randomEvents(&seqRand{draws: []int{0, 0, 0, 2, 1, 2, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1}})
	p := g.Planets[0]
	if p.Env != [3]int{54, 50, 50} || p.OrigEnv != [3]int{54, 50, 50} {
		t.Errorf("env %v orig %v, want gravity 54", p.Env, p.OrigEnv)
	}
	if p.Surface != (Minerals{190, 3, 3}) || p.Deposits[0].Concentration != 162 || p.Deposits[1].Concentration != 112 {
		t.Errorf("surface %v deposits %v", p.Surface, p.Deposits)
	}
	if !reflect.DeepEqual(p.Queue, []QueueItem{{Kind: ItemAutoFactories, Count: 5}}) {
		t.Errorf("queue %v", p.Queue)
	}
	want := []Event{
		{Kind: EventCometStrike, Player: 0, Planet: 1, Fleet: -1, Count: 0, Axes: []int{2}},
		{Kind: EventCometStrike, Player: 1, Planet: 1, Fleet: -1, Count: 0},
	}
	if !reflect.DeepEqual(ev, want) {
		t.Errorf("events %+v", ev)
	}

	// Huge comet: concentration 112 → 200 (capped).
	g = eventGame(30, NoOwner, 0)
	g.randomEvents(&seqRand{draws: []int{0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 49, 14}})
	if c := g.Planets[0].Deposits[0].Concentration; c != 200 {
		t.Errorf("huge comet concentration %d, want 200", c)
	}

	// S3: no comet effect at year index 5; none on a protected planet.
	for _, g := range []Game{eventGame(5, NoOwner, 0), eventGame(15, 0, 51)} {
		before := g.Planets[0]
		g.randomEvents(&seqRand{draws: []int{0, 0, 3, 1, 1}})
		if !reflect.DeepEqual(g.Planets[0], before) {
			t.Errorf("year %d: planet changed: %+v", g.Year, g.Planets[0])
		}
	}
}

func TestConfirmedClimateChange(t *testing.T) {
	// S3: unowned planet at year index 5, gravity 50 → 44. S5: owned
	// planet at index 30, radiation 50 → 44, owner message naming
	// radiation, queue cut to Auto Factories ×5.
	g := eventGame(5, NoOwner, 0)
	if ev := g.randomEvents(&seqRand{draws: []int{1, 0, 0, 0, 0, 0, 1, 1}}); len(ev) != 0 || g.Planets[0].Env != [3]int{44, 50, 50} {
		t.Errorf("S3: env %v events %v", g.Planets[0].Env, ev)
	}
	g = eventGame(30, 0, 9237)
	ev := g.randomEvents(&seqRand{draws: []int{1, 0, 0, 2, 0, 0, 1, 1}})
	want := []Event{{Kind: EventClimateChange, Player: 0, Planet: 1, Fleet: -1, Axes: []int{2}}}
	if p := g.Planets[0]; p.Env != [3]int{50, 50, 44} || p.OrigEnv != p.Env || len(p.Queue) != 1 || !reflect.DeepEqual(ev, want) {
		t.Errorf("S5: env %v orig %v queue %v events %+v", p.Env, p.OrigEnv, p.Queue, ev)
	}
}

func TestConfirmedNewMinerals(t *testing.T) {
	// +13 ironium on an unowned planet (no message); the owner gets a
	// message even when the concentration is at the cap.
	g := eventGame(30, NoOwner, 0)
	if ev := g.randomEvents(&seqRand{draws: []int{1, 1, 0, 0, 0, 8}}); len(ev) != 0 || g.Planets[0].Deposits[0].Concentration != 125 {
		t.Errorf("concentration %d events %v", g.Planets[0].Deposits[0].Concentration, ev)
	}
	g = eventGame(30, 0, 100)
	g.Planets[0].Deposits[2].Concentration = 180
	ev := g.randomEvents(&seqRand{draws: []int{1, 1, 0, 0, 2, 8}})
	if g.Planets[0].Deposits[2].Concentration != 180 || len(ev) != 1 || ev[0].Kind != EventNewMinerals || ev[0].Count != 2 {
		t.Errorf("at the cap: %v events %v", g.Planets[0].Deposits[2], ev)
	}
	// Year index below 10: nothing.
	g = eventGame(9, NoOwner, 0)
	g.randomEvents(&seqRand{draws: []int{1, 1, 0, 0, 0, 8}})
	if g.Planets[0].Deposits[0].Concentration != 112 {
		t.Error("new minerals before year index 10")
	}
}

func TestConfirmedRandomEventsOff(t *testing.T) {
	// E0: with the option off nothing runs and nothing is drawn.
	g := eventGame(30, 0, 9237)
	g.RandomEvents = false
	if ev := g.randomEvents(panicRand{}); ev != nil {
		t.Errorf("events %v", ev)
	}
}

func TestPredictionCometAxesSwitch(t *testing.T) {
	// The S2 shuffle [2 1 0] for a small comet: radiation with the legacy
	// switch, gravity (the axis that moved) without it.
	a := [3]int{2, 1, 0}
	if got := cometAxes(a, 1, true); !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("legacy %v", got)
	}
	if got := cometAxes(a, 2, false); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("fixed %v", got)
	}
}

// cometDraws is the draw sequence of a comet strike (KERNEL.md "Comet
// strike") on a game's only planet that gives a comet of size e, striking
// the minerals in order (its first min(e, 2) + 1 entries are struck),
// adding gain kT of each mineral to the surface, raising each struck
// mineral's concentration by dc, and moving the axes in order by moves.
func cometDraws(t *testing.T, e int, order [3]int, gain, dc [3]int, moves []int) []int {
	t.Helper()
	huge := e == 3
	d := []int{0, 0, e, 0, 0, 0}
	moved := min(e, 2) + 1
	struck := [3]bool{}
	for k := range moved {
		struck[order[k]] = true
	}
	for m := range 3 {
		if struck[m] {
			d = append(d, 0) // the base, before the struck 3000 + rand(17000)
		} else {
			d = append(d, 16*gain[m]-50)
		}
	}
	// The order's two swaps.
	bo := [3]int{0, 1, 2}
	d = append(d, order[0])
	bo[0], bo[order[0]] = bo[order[0]], bo[0]
	j := 1
	if bo[2] == order[1] {
		j = 2
	}
	d = append(d, j-1)
	for k := range moved {
		m := order[k]
		r := dc[m] - 50
		if huge {
			r -= 15
		}
		r15 := 0
		if huge {
			r15 = min(14, max(0, r-49))
			r -= r15
		}
		d = append(d, 16*gain[m]-3050, min(49, r))
		if huge {
			d = append(d, r15)
		}
	}
	for _, s := range moves {
		a := max(s, -s)
		if huge {
			d = append(d, min(2, a-6), a-6-min(2, a-6))
		} else {
			d = append(d, a-3)
		}
		if s < 0 {
			d = append(d, 1)
		} else {
			d = append(d, 0)
		}
	}
	for i, v := range d {
		if v < 0 {
			t.Fatalf("draw %d is %d: the outcome is out of the comet's range", i, v)
		}
	}
	return d
}

// TestCometStrikeS11 replays the seven comet strikes of KX-004 S11
// (stars-elegy 4d3a3d9 PARITY.md "Comets on terraformed planets"),
// conditioned on the draws, not as parity: each strike is given the size,
// struck minerals and moves the original's result shows, and Elegy's
// environment, original environment, concentrations and surface gain are
// compared with the original's. A comet moves the current and original
// value of each axis it hits by the same amount, each clamped to 1..99 on
// its own (CONFIRMED, S11); an unowned struck planet keeps the comet's
// minerals (MEASURED, S11, three strikes); a planet with no separate
// original keeps none.
func TestCometStrikeS11(t *testing.T) {
	for _, c := range []struct {
		name               string
		owner, e           int
		env, orig          [3]int
		conc               [3]int
		order              [3]int
		gain, dc           [3]int
		moves              []int
		wantEnv, wantOrig  [3]int
		wantConc           [3]int
		checkConcentration bool
	}{
		{"planet 11, medium", 0, 1, [3]int{97, 48, 50}, [3]int{60, 40, 30}, [3]int{79, 49, 13}, [3]int{1, 2, 0},
			[3]int{10, 880, 607}, [3]int{0, 85, 86}, []int{3, 4}, [3]int{99, 52, 50}, [3]int{63, 44, 30}, [3]int{79, 134, 99}, true},
		{"planet 2, huge", 0, 3, [3]int{4, 93, 50}, [3]int{50, 50, 95}, [3]int{112, 84, 56}, [3]int{0, 1, 2},
			[3]int{365, 814, 881}, [3]int{200, 84, 76}, []int{-7, 8, 10}, [3]int{1, 99, 60}, [3]int{43, 58, 99}, [3]int{200, 168, 132}, true},
		{"planet 15, small", 1, 0, [3]int{50, 50, 50}, [3]int{97, 20, 20}, [3]int{89, 53, 71}, [3]int{0, 1, 2},
			[3]int{895, 18, 15}, [3]int{86, 0, 0}, []int{4}, [3]int{54, 50, 50}, [3]int{99, 20, 20}, [3]int{175, 53, 71}, true},
		{"planet 22, large", NoOwner, 2, [3]int{22, 3, 84}, [3]int{25, 10, 97}, [3]int{70, 52, 88}, [3]int{0, 1, 2},
			[3]int{1207, 452, 1139}, [3]int{57, 71, 51}, []int{-3, -4, -3}, [3]int{19, 1, 81}, [3]int{22, 6, 94}, [3]int{127, 123, 139}, true},
		{"planet 22, medium", NoOwner, 1, [3]int{22, 3, 84}, [3]int{25, 10, 97}, [3]int{70, 52, 88}, [3]int{0, 2, 1},
			[3]int{1038, 11, 512}, [3]int{95, 0, 62}, []int{4, -4}, [3]int{26, 1, 84}, [3]int{29, 6, 97}, [3]int{165, 52, 150}, true},
		// Planet 17 is a homeworld whose concentrations also move by
		// mining; only its environment and surface gain are compared.
		{"planet 17, large", 0, 2, [3]int{50, 50, 50}, [3]int{50, 50, 50}, [3]int{30, 30, 88}, [3]int{0, 1, 2},
			[3]int{257, 379, 1107}, [3]int{50, 50, 50}, []int{5, -3, 4}, [3]int{55, 47, 54}, [3]int{55, 47, 54}, [3]int{}, false},
		{"planet 19, small", NoOwner, 0, [3]int{52, 78, 29}, [3]int{52, 78, 29}, [3]int{23, 38, 74}, [3]int{2, 0, 1},
			[3]int{17, 8, 395}, [3]int{0, 0, 82}, []int{-4}, [3]int{48, 78, 29}, [3]int{48, 78, 29}, [3]int{23, 38, 156}, true},
	} {
		pop := 0
		if c.owner != NoOwner {
			pop = 9101
		}
		g := eventGame(30, c.owner, pop)
		p := &g.Planets[0]
		p.Env, p.OrigEnv = c.env, c.orig
		for m := range NumMinerals {
			p.Deposits[m].Concentration = c.conc[m]
		}
		g.cometStrike(&seqRand{draws: cometDraws(t, c.e, c.order, c.gain, c.dc, c.moves)}, []int{0})
		got := g.Planets[0]
		if got.Env != c.wantEnv || got.OrigEnv != c.wantOrig {
			t.Errorf("%s: environment %v, original %v; want %v, %v", c.name, got.Env, got.OrigEnv, c.wantEnv, c.wantOrig)
		}
		if got.Surface != Minerals(c.gain) {
			t.Errorf("%s: surface %v, want %v", c.name, got.Surface, c.gain)
		}
		var conc [3]int
		for m := range NumMinerals {
			conc[m] = got.Deposits[m].Concentration
		}
		if c.checkConcentration && conc != c.wantConc {
			t.Errorf("%s: concentrations %v, want %v", c.name, conc, c.wantConc)
		}
	}
}

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
	return Game{Year: 2400 + yearIndex, RandomEvents: true, Players: make([]Player, 2), Planets: []Planet{p}}
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

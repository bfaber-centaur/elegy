package engine

import (
	"reflect"
	"testing"
)

// traderObjects gives the engine a Trader part word: parts by name in
// OBJECTS.md "Encounters" bit order, and each player's owned bits.
type traderObjects struct {
	SpaceObjects
	owned map[[2]int]bool
}

var testTraderParts = []string{
	"Multi Cargo Pod", "Multi Function Pod", "Langston Shell", "Mega Poly Shell",
	"Alien Miner", "Hush-a-Boom", "Anti Matter Torpedo", "Multi Contained Munition",
	"Mini Morph", "Enigma Pulsar", "Genesis Device", "Jump Gate",
}

func (o *traderObjects) TraderBit(part string) int {
	for b, n := range testTraderParts {
		if n == part {
			return b
		}
	}
	return -1
}
func (o *traderObjects) OwnsTraderBit(p, b int) bool { return o.owned[[2]int{p, b}] }
func (o *traderObjects) GiveTraderBit(p, b int)      { o.owned[[2]int{p, b}] = true }

// The Mini Morph of CB-048: Enigma Pulsar ×2, Multi Cargo Pod ×3, Multi
// Function Pod, Mega Poly Shell, Langston Shell (and lasers).
func cb048Morph(t *testing.T) Design {
	t.Helper()
	d, err := Components().NewDesign("Morph", "Mini Morph", []SlotFill{
		{0, "Enigma Pulsar", 2}, {1, "Multi Cargo Pod", 3}, {2, "Multi Function Pod", 1},
		{3, "Mega Poly Shell", 1}, {4, "Langston Shell", 1}, {5, "Laser", 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Mystery Trader chances (COMBAT.md, CONFIRMED CB-048): each kill event
// adds each Trader part slot's count once, whatever the ships killed; the
// hull never counts; each chance stops at 25.
func TestConfirmedTraderChances(t *testing.T) {
	d := cb048Morph(t)
	g := &Game{Objects: &traderObjects{owned: map[[2]int]bool{}}}
	var c traderChances
	c.addParts(g, d, 1)
	want := traderChances{0: 3, 1: 1, 2: 1, 3: 1, 9: 2}
	if c != want {
		t.Fatalf("one kill event: %v, want %v", c, want)
	}
	for range 10 {
		c.addParts(g, d, 1)
	}
	if want := (traderChances{0: 25, 1: 11, 2: 11, 3: 11, 9: 22}); c != want {
		t.Errorf("eleven kill events: %v, want %v", c, want)
	}
	// Through a battle's kill event, six ships killed count once.
	e := tok([]Design{d}, 0, 1, 10)
	tb := newTestBattle(panicRand{}, []Design{d}, e)
	tb.g.Objects = g.Objects
	tb.killEvent(e, 6)
	if tb.traderChance != (traderChances{0: 3, 1: 1, 2: 1, 3: 1, 9: 2}) {
		t.Errorf("kill event of six: %v", tb.traderChance)
	}
	// No space objects: no part is a Trader part.
	var none traderChances
	none.addParts(&Game{}, d, 1)
	if none != (traderChances{}) {
		t.Errorf("without objects: %v", none)
	}
}

// Step 3 of a tech attempt (COMBAT.md "Tech from battle", CONFIRMED
// CB-048): rand(13) picks k; an owned item or one with no chance makes no
// second draw; otherwise rand(100) < c gives Trader part bit k and ends
// the attempt.
func TestConfirmedTechAttemptTraderItem(t *testing.T) {
	obj := &traderObjects{owned: map[[2]int]bool{{0, 1}: true}}
	g := &Game{Objects: obj, Players: make([]Player, 1)}
	chance := traderChances{0: 3, 1: 1, 9: 2}
	// Gate 60 (passes); k 5 (no chance); k 1 (owned); k 0, rand(100) 3
	// (not below 3); k 9, rand(100) 1 (below 2): bit 9.
	rng := &recordBounds{r: &seqRand{draws: []int{60, 5, 1, 0, 3, 9, 1}}}
	ev := techAttempt(g, rng, 0, [NumFields]int{}, chance, map[int]bool{})
	if want := []int{100, 13, 13, 13, 100, 13, 100}; !reflect.DeepEqual(rng.bounds, want) {
		t.Errorf("draws %v, want %v", rng.bounds, want)
	}
	if len(ev) != 1 || ev[0].Kind != EventTraderPartFound || ev[0].Count != 9 || !obj.owned[[2]int{0, 9}] {
		t.Errorf("events %+v, owned %v; want Enigma Pulsar (bit 9)", ev, obj.owned)
	}
	// A gain marks the player: a second attempt this turn draws nothing.
	gained := map[int]bool{0: true}
	rng = &recordBounds{r: &seqRand{}}
	if ev := techAttempt(g, rng, 0, [NumFields]int{}, chance, gained); ev != nil || len(rng.bounds) != 0 {
		t.Errorf("second attempt: %+v, draws %v", ev, rng.bounds)
	}
}

// A scrap at a starbase (TAKEOVER.md "Other waypoint tasks", MEASURED
// TK-305): the planet owner's attempt has the scrapped ships' Trader part
// counts as chances. ASSUMPTION K11: once per ship, so two ships of a
// design with two Hush-a-Boom give 4: rand(100) = 3 gains it, where one
// ship's 2 would not.
func TestScrapTraderChance(t *testing.T) {
	for _, ships := range []int{1, 2} {
		l := newTKLab(t, 3)
		obj := &traderObjects{owned: map[[2]int]bool{}}
		l.g.Objects = obj
		bomber := l.design("B-17 Bomber", SlotFill{0, "Quick Jump 5", 2}, SlotFill{1, "Hush-a-Boom", 2})
		pi := l.planet(1, 0, 100)
		l.g.Planets[pi].HasStarbase = true
		l.g.Planets[pi].StarbaseDesign = l.design("Orbital Fort")
		fi := l.fleet(0, pi, 0, Stack{Design: bomber, Count: ships})
		// Gate 60; k 5 (Hush-a-Boom), rand(100) 3.
		ev := l.g.scrap(fi, &seqRand{draws: []int{60, 5, 3}}, map[int]bool{}, scrapYear{})
		got := obj.owned[[2]int{1, 5}]
		if got != (ships == 2) {
			t.Errorf("%d ships: Hush-a-Boom given %v, events %+v", ships, got, ev)
		}
	}
}

package engine

import (
	"reflect"
	"testing"
)

// numberedShipyard is shipyard with player 0's fleets numbered 1–3 and
// 5–12 and a queue of three one-ship items at planet 0.
func numberedShipyard(t *testing.T) *Game {
	g := shipyard(t)
	for i, n := range []int{1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12} {
		g.Fleets = append(g.Fleets, Fleet{ID: 100 + i, Number: n, Owner: 0, Stacks: []Stack{{Design: 0, Count: 1}}})
	}
	p := &g.Planets[0]
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 0, Count: 1}, {Kind: ItemShip, Slot: 0, Count: 1}, {Kind: ItemShip, Slot: 0, Count: 1}}
	return g
}

func builtNumbers(g *Game) []int {
	var out []int
	for _, f := range g.Fleets {
		if f.ID > 110 { // built: IDs after the 11 above
			out = append(out, f.Number)
		}
	}
	return out
}

func TestConfirmedGoneFleetHoldsNumber(t *testing.T) {
	// KERNEL.md "Fleet numbers" (CONFIRMED KX-007; RG-001 2432): fleet 3
	// colonizes from orbit before production, and the year's three new
	// fleets take 4, 13 and 14; number 3 is free only next year.
	g := numberedShipyard(t)
	g.holdFleetNumbers()
	g.removeFleets(map[int]bool{102: true}) // fleet 3 goes away
	shipyardRun(g, 0, 10000)
	if got := builtNumbers(g); !reflect.DeepEqual(got, []int{4, 13, 14}) {
		t.Errorf("new fleets %v, want [4 13 14]", got)
	}
	g.heldNumbers = nil // the year's files are written
	if n := g.lowestFreeFleetNumber(0); n != 3 {
		t.Errorf("next year's first number %d, want 3", n)
	}
}

func TestBinaryOnlyOrderRemovalFreesNumber(t *testing.T) {
	// KERNEL.md "Fleet numbers" (BINARY-ONLY): a fleet deleted while the
	// orders are read, merged away or emptied, frees its number at once.
	g := numberedShipyard(t)
	g.removeFleets(map[int]bool{102: true})
	g.holdFleetNumbers()
	shipyardRun(g, 0, 10000)
	if got := builtNumbers(g); !reflect.DeepEqual(got, []int{3, 4, 13}) {
		t.Errorf("new fleets %v, want [3 4 13]", got)
	}
}

func TestConfirmedScrappedFleetHoldsNumberInTurn(t *testing.T) {
	// The same through GenerateTurn (KERNEL.md "Fleet numbers", CONFIRMED
	// for scrapping, KX-007): fleet 1 is scrapped at the planet before
	// production builds one ship, which takes number 2, and the returned
	// game holds no number for next year.
	l := newTKLab(t, 3)
	scout := l.design("Scout", SlotFill{0, "Long Hump 6", 1})
	dock := l.design("Space Dock")
	l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: scout}, {Owner: 0, Starbase: true, Slot: 0, Design: dock}}
	pi := l.planet(0, 0, 100000)
	l.g.Planets[pi].HasStarbase, l.g.Planets[pi].StarbaseDesign = true, dock
	l.g.Planets[pi].Surface = Minerals{10000, 10000, 10000}
	l.g.Planets[pi].HasQueue, l.g.Planets[pi].Queue = true, []QueueItem{{Kind: ItemShip, Slot: 0, Count: 1}}
	fi := l.fleet(0, pi, 0, Stack{Design: scout, Count: 1})
	l.g.Fleets[fi].Number, l.g.Fleets[fi].Task = 1, Task{Kind: TaskScrap}
	r := l.turn(&seqRand{})
	var numbers []int
	for _, f := range r.Game.Fleets {
		numbers = append(numbers, f.Number)
	}
	if !reflect.DeepEqual(numbers, []int{2}) {
		t.Errorf("fleet numbers %v, want [2]: the built ship, not reusing the scrapped 1", numbers)
	}
	if r.Game.heldNumbers != nil {
		t.Error("the returned game still holds numbers")
	}
}

func TestBinaryOnlyNewFleetGoneHoldsNumber(t *testing.T) {
	// A fleet created after the orders that goes away the same year holds
	// its number too (KERNEL.md "Fleet numbers": every fleet that goes away
	// during the generation).
	g := numberedShipyard(t)
	g.holdFleetNumbers()
	n := g.lowestFreeFleetNumber(0)
	if n != 4 {
		t.Fatalf("first new number %d, want 4", n)
	}
	if m := g.lowestFreeFleetNumber(0); m != 13 {
		t.Errorf("number after 4 was taken and its fleet went away: %d, want 13", m)
	}
}

func TestBinaryOnlyMergedInOrdersFreesNumberInTurn(t *testing.T) {
	// The same through GenerateTurn (KERNEL.md "Fleet numbers",
	// BINARY-ONLY): fleet 1 is merged into fleet 2 while the orders are
	// read, so the ship built this year takes number 1 at once.
	l := newTKLab(t, 3)
	scout := l.design("Scout", SlotFill{0, "Long Hump 6", 1})
	dock := l.design("Space Dock")
	l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: scout}, {Owner: 0, Starbase: true, Slot: 0, Design: dock}}
	pi := l.planet(0, 0, 100000)
	l.g.Planets[pi].HasStarbase, l.g.Planets[pi].StarbaseDesign = true, dock
	l.g.Planets[pi].Surface = Minerals{10000, 10000, 10000}
	l.g.Planets[pi].HasQueue, l.g.Planets[pi].Queue = true, []QueueItem{{Kind: ItemShip, Slot: 0, Count: 1}}
	a := l.fleet(0, pi, 0, Stack{Design: scout, Count: 1})
	b := l.fleet(0, pi, 0, Stack{Design: scout, Count: 1})
	l.g.Fleets[a].Number, l.g.Fleets[b].Number = 1, 2
	g := withRules(l.g)
	merge := MergeOrder{Into: g.Fleets[b].ID, From: []int{g.Fleets[a].ID}}
	r, err := GenerateTurn(g, []PlayerOrders{{Player: 0, GameID: g.ID, Year: g.Year, Orders: []Order{merge}}}, &seqRand{})
	if err != nil {
		t.Fatal(err)
	}
	var numbers []int
	for _, f := range r.Game.Fleets {
		numbers = append(numbers, f.Number)
	}
	if !reflect.DeepEqual(numbers, []int{2, 1}) {
		t.Errorf("fleet numbers %v, want [2 1]: the merged fleet's 1 reused by the built ship", numbers)
	}
}

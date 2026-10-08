package engine

import (
	"errors"
	"testing"
)

// shipsGame has player 0's fleet 1 of four freighters (hold 100, tank 200
// each) with 410 kT of ironium and 1001 mg of fuel, a waypoint, plan 3
// and the repeat flag, and fleet 2 of one freighter beside it.
func shipsGame() *Game {
	g := ordersGame()
	g.Fleets[0].Stacks = []Stack{{Design: 1, Count: 4}}
	g.Fleets[0].Cargo = Cargo{Minerals: Minerals{410, 0, 0}}
	g.Fleets[0].Fuel = 1001
	g.Fleets[0].Number, g.Fleets[1].Number, g.Fleets[3].Number = 1, 2, 3
	g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1300}, Warp: 6}}
	g.Fleets[0].Repeat = true
	return g
}

func TestConfirmedSplitSharesByCapacity(t *testing.T) {
	// ORDERS.md "Split" (CONFIRMED CO-01): one of four ships takes
	// ⌊410/4⌋ = 102 kT and ⌊1001/4⌋ = 250 mg; the new fleet keeps the
	// source's plan and waypoints and takes the lowest free number
	// (ASSUMPTION L20, L21).
	g := shipsGame()
	if errs, _ := apply(g, 0, SplitOrder{Fleet: 1, Ships: []Stack{{Design: 1, Count: 1}}}); errs[0] != nil {
		t.Fatal(errs)
	}
	src, nf := g.Fleets[0], g.Fleets[len(g.Fleets)-1]
	if src.Stacks[0].Count != 3 || src.Cargo.Minerals[0] != 308 || src.Fuel != 751 {
		t.Errorf("source %+v", src)
	}
	if nf.Owner != 0 || nf.Number != 4 || nf.Stacks[0].Count != 1 || nf.Cargo.Minerals[0] != 102 || nf.Fuel != 250 ||
		nf.Plan != 3 || len(nf.Waypoints) != 1 || nf.Waypoints[0] != src.Waypoints[0] || !nf.Repeat || nf.Pos != src.Pos {
		t.Errorf("new fleet %+v", nf)
	}
}

func TestConfirmedSplitAllRemainderOnSource(t *testing.T) {
	// CO-02: Split All of three ships over 100 kT / 1000 mg leaves 34/334
	// on the source and 33/333 on each new fleet.
	g := shipsGame()
	g.Fleets[0].Stacks[0].Count, g.Fleets[0].Cargo.Minerals[0], g.Fleets[0].Fuel = 3, 100, 1000
	one := []Stack{{Design: 1, Count: 1}}
	if errs, _ := apply(g, 0, SplitOrder{Fleet: 1, Ships: one}, SplitOrder{Fleet: 1, Ships: one}); errs[0] != nil || errs[1] != nil {
		t.Fatal(errs)
	}
	n := len(g.Fleets)
	for k, f := range []Fleet{g.Fleets[0], g.Fleets[n-2], g.Fleets[n-1]} {
		want := [2]int{33, 333}
		if k == 0 {
			want = [2]int{34, 334}
		}
		if got := [2]int{f.Cargo.Minerals[0], f.Fuel}; got != want || f.Stacks[0].Count != 1 {
			t.Errorf("fleet %d: %v, %d ships, want %v", f.ID, got, f.Stacks[0].Count, want)
		}
	}
}

func TestConfirmedMoveShips(t *testing.T) {
	// CO-03: one of two ships moves with ⌊200/2⌋ kT and ⌊600/2⌋ mg into a
	// fleet beside it; it joins that fleet's stack.
	g := shipsGame()
	g.Fleets[0].Stacks[0].Count, g.Fleets[0].Cargo.Minerals[0], g.Fleets[0].Fuel = 2, 200, 600
	g.Fleets[1].Fuel = 0
	if errs, _ := apply(g, 0, MoveShipsOrder{Fleet: 2, With: 1, Ships: []Stack{{Design: 1, Count: 1}}}); errs[0] != nil {
		t.Fatal(errs)
	}
	a, b := g.Fleets[0], g.Fleets[1]
	if a.Stacks[0].Count != 1 || a.Cargo.Minerals[0] != 100 || a.Fuel != 300 || b.Stacks[0].Count != 2 || b.Cargo.Minerals[0] != 100 || b.Fuel != 300 {
		t.Errorf("fleets %+v %+v", a, b)
	}
}

func TestMoveShipsChecks(t *testing.T) {
	// More ships than the fleet has, a fleet elsewhere, another player's
	// fleet and a design named twice are refused; a fleet emptied by a
	// move is removed (ASSUMPTION L23); a stack holds at most 32765
	// (MEASURED CO-06).
	g := shipsGame()
	errs, _ := apply(g, 0,
		MoveShipsOrder{Fleet: 1, With: 2, Ships: []Stack{{Design: 1, Count: -5}}},
		MoveShipsOrder{Fleet: 1, With: 4, Ships: []Stack{{Design: 1, Count: -1}}},
		MoveShipsOrder{Fleet: 1, With: 3, Ships: []Stack{{Design: 1, Count: -1}}},
		SplitOrder{Fleet: 1, Ships: []Stack{{Design: 1, Count: 1}, {Design: 1, Count: 1}}},
		MoveShipsOrder{Fleet: 1, With: 2, Ships: []Stack{{Design: 1, Count: 1}}},
	)
	if !errors.Is(errs[0], ErrOutOfRange) || !errors.Is(errs[1], ErrNotTogether) || !errors.Is(errs[2], ErrNotYours) || !errors.Is(errs[3], ErrOutOfRange) || errs[4] != nil {
		t.Fatal(errs)
	}
	if g.fleetIndex(2) >= 0 || g.Fleets[0].Stacks[0].Count != 5 || g.Fleets[0].Fuel != 1021 {
		t.Errorf("fleets %+v", g.Fleets)
	}
	g = shipsGame()
	g.Fleets[0].Stacks[0].Count, g.Fleets[1].Stacks[0].Count = 16000, 17000
	if errs, _ := apply(g, 0, MoveShipsOrder{Fleet: 1, With: 2, Ships: []Stack{{Design: 1, Count: 16999}}}); errs[0] != nil {
		t.Fatal(errs)
	}
	if n := g.Fleets[0].Stacks[0].Count; n != maxExchangeStack {
		t.Errorf("stack %d, want %d", n, maxExchangeStack)
	}
}

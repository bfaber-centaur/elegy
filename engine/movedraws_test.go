package engine

import (
	"reflect"
	"testing"
)

// scriptRand returns its draws in turn (then 1) and counts the calls.
type scriptRand struct {
	draws []int
	calls int
}

func (s *scriptRand) Intn(n int) int {
	s.calls++
	if len(s.draws) == 0 {
		return 1 % n
	}
	d := s.draws[0]
	s.draws = s.draws[1:]
	return d % n
}

// drawGame has player 0 (Cheap Engines when ce) and designs 0 (unrated
// engine, 100 mg tank, 10 kT hold) and 1 (rated for warp 10); the fleets'
// engines burn no fuel.
func drawGame(ce bool, fleets ...Fleet) Game {
	var free Engine
	rated := Engine{BattleWarp10: true}
	g := Game{Rules: ElegyRules(), Players: []Player{{}}, Fleets: fleets, Designs: []Design{
		{Name: "unrated", Mass: 10, Engine: free, Engines: 1, FuelCapacity: 100, CargoCapacity: 10},
		{Name: "rated", Mass: 10, Engine: rated, Engines: 1, FuelCapacity: 100, CargoCapacity: 10},
	}}
	g.Players[0].Race.LRT.CheapEngines = ce
	return g
}

func mover(id, warp int, stacks ...Stack) Fleet {
	return Fleet{ID: id, Number: id, Stacks: stacks, Fuel: 100, Waypoints: []Waypoint{{Pos: Point{200, 0}, Warp: warp}}}
}

func TestConfirmedCheapEnginesStall(t *testing.T) {
	// KERNEL.md "Other movement rules" (KX-006, FM-101 F; stars-elegy
	// fda4173): one rand(10) per Cheap Engines fleet at warp 7 to 10, 0
	// stops it with 0x0f2, keeping its fuel and waypoints and making no
	// warp-10 loss draws; warp 6 and a stargate jump draw nothing.
	g := drawGame(true,
		mover(1, 6, Stack{Design: 1, Count: 1}),
		mover(2, 7, Stack{Design: 1, Count: 1}),
		mover(3, 10, Stack{Design: 0, Count: 3}),
		mover(4, StargateWarp, Stack{Design: 1, Count: 1}),
	)
	r := &scriptRand{draws: []int{5, 0}}
	ev, _ := g.moveAll(r)
	if r.calls != 2 {
		t.Errorf("%d draws, want 2: fleet 2's and fleet 3's stall draws", r.calls)
	}
	if g.Fleets[1].Pos == (Point{}) {
		t.Error("fleet 2 drew 5 and should have moved")
	}
	f := g.Fleets[2]
	if f.Pos != (Point{}) || f.Fuel != 100 || len(f.Waypoints) != 1 || f.Stacks[0].Count != 3 {
		t.Errorf("stalled fleet %+v: want it unmoved with its fuel, waypoint and ships", f)
	}
	var stalls []int
	for _, e := range ev {
		if e.Kind == EventEngineStall {
			stalls = append(stalls, e.Fleet)
		}
	}
	if !reflect.DeepEqual(stalls, []int{3}) {
		t.Errorf("stall events for fleets %v, want [3]", stalls)
	}
	// Without Cheap Engines nothing is drawn below warp 10.
	g = drawGame(false, mover(1, 9, Stack{Design: 0, Count: 1}))
	r = &scriptRand{}
	g.moveAll(r)
	if r.calls != 0 {
		t.Errorf("non-CE warp 9: %d draws, want 0", r.calls)
	}
}

func TestConfirmedWarp10Losses(t *testing.T) {
	// KERNEL.md "Other movement rules" (KX-006 OB-024, FM-102; stars-elegy
	// fda4173): one rand(10) per ship of an unrated stack at warp 10, 0
	// loses the ship; a rated stack draws nothing; the lost ships' fuel
	// share goes with them (FM round 2: 30000 − trunc(30000·11/100)).
	draws := make([]int, 100)
	for i := range draws {
		draws[i] = 1
	}
	for _, i := range []int{3, 9, 17, 22, 40, 41, 55, 60, 71, 88, 99} {
		draws[i] = 0
	}
	f := mover(1, 10, Stack{Design: 0, Count: 100})
	f.Fuel = 30000
	g := drawGame(false, f, mover(2, 10, Stack{Design: 1, Count: 50}))
	g.Designs[0].FuelCapacity = 300
	r := &scriptRand{draws: draws}
	ev, _ := g.moveAll(r)
	if r.calls != 100 {
		t.Errorf("%d draws, want 100 (none for the rated fleet)", r.calls)
	}
	if got := g.Fleets[0]; got.Stacks[0].Count != 89 || got.Fuel != 30000-30000*11/100 {
		t.Errorf("fleet 1: %d ships, %d mg; want 89 and %d", got.Stacks[0].Count, got.Fuel, 30000-30000*11/100)
	}
	if len(ev) == 0 || ev[0].Kind != EventWarp10ShipsLost || ev[0].Fleet != 1 || ev[0].Count != 11 {
		t.Errorf("events %v, want ships lost: fleet 1, 11", ev)
	}
	if g.Fleets[1].Stacks[0].Count != 50 {
		t.Error("the rated fleet lost ships")
	}
}

func TestPredictionWarp10WholeFleetLost(t *testing.T) {
	// Every ship lost (0x0e1): the fleet is gone after movement and does
	// not move; its stall draw came first (KX-006 order).
	g := drawGame(true, mover(1, 10, Stack{Design: 0, Count: 2}, Stack{Design: 1, Count: 0}))
	r := &scriptRand{draws: []int{1, 0, 0}}
	ev, _ := g.moveAll(r)
	if r.calls != 3 {
		t.Errorf("%d draws, want 3: the stall, then one per ship", r.calls)
	}
	if len(ev) != 1 || ev[0].Kind != EventWarp10FleetLost || ev[0].Count != 2 {
		t.Errorf("events %v, want the whole fleet lost", ev)
	}
	if g.Fleets[0].Pos != (Point{}) {
		t.Error("the lost fleet moved")
	}
	g.dropShiplessFleets()
	if len(g.Fleets) != 0 {
		t.Error("the lost fleet was not removed")
	}
}

func TestPredictionWarp10CargoShare(t *testing.T) {
	// ASSUMPTION M2: the lost ships' cargo share, by cargo capacity, goes
	// with them: 2 of 4 ships lost from 21 kT ironium and 3 kT colonists
	// take 12 kT, 10 and 1 by item and the 1 kT left from ironium.
	f := mover(1, 10, Stack{Design: 0, Count: 4})
	f.Cargo = Cargo{Minerals: Minerals{21, 0, 0}, Colonists: 3}
	g := drawGame(false, f)
	g.moveAll(&scriptRand{draws: []int{0, 1, 0, 1}})
	if c := g.Fleets[0].Cargo; c.Minerals[Ironium] != 10 || c.Colonists != 2 {
		t.Errorf("cargo %+v, want 10 kT ironium and 2 kT colonists left", c)
	}
}

func TestAssumptionWarp10RoundsDown(t *testing.T) {
	// ASSUMPTION M2 and M3: the lost shares are truncated. 1 of 3 ships
	// lost takes trunc(100/3) = 33 mg of fuel and trunc(10/3) = 3 kT of
	// cargo.
	f := mover(1, 10, Stack{Design: 0, Count: 3})
	f.Cargo = Cargo{Minerals: Minerals{10, 0, 0}}
	g := drawGame(false, f)
	// The loss itself, before movement's fuel use and scooping.
	g.warp10Losses(&g.Fleets[0], &scriptRand{draws: []int{0, 1, 1}})
	if got := g.Fleets[0]; got.Fuel != 67 || got.Cargo.Minerals[Ironium] != 7 {
		t.Errorf("fuel %d, ironium %d; want 67 and 7", got.Fuel, got.Cargo.Minerals[Ironium])
	}
}

func TestAssumptionStallKeepsARColonists(t *testing.T) {
	// ASSUMPTION M1: a stalled Alternate Reality fleet loses no colonists
	// in flight; one that moves does.
	for _, c := range []struct {
		stall int
		want  int
	}{{0, 100}, {1, 97}} {
		f := mover(1, 7, Stack{Design: 1, Count: 1})
		f.Cargo.Colonists = 100
		g := drawGame(true, f)
		g.Players[0].Race.PRT = PRTAlternateReality
		g.Designs[1].CargoCapacity = 1000
		g.moveAll(&scriptRand{draws: []int{c.stall}})
		if got := g.Fleets[0].Cargo.Colonists; got != c.want {
			t.Errorf("stall draw %d: %d kT colonists, want %d", c.stall, got, c.want)
		}
	}
}

func TestBinaryOnlySlotOrderOwnShipSlots(t *testing.T) {
	// slotOrder reads only the owner's ship-design slots: another owner's
	// slot and a starbase slot naming the same design index are ignored
	// (design indexes can be shared, orders_design.go designShared), and
	// a design with no slot sorts after every slot by its design index,
	// as shipSlot orders a fleet's stacks.
	g := drawGame(false)
	g.Players = append(g.Players, Player{})
	f := Fleet{Owner: 0, Stacks: []Stack{{Design: 3}, {Design: 1}, {Design: 2}, {Design: 0}}}
	g.DesignSlots = []DesignSlot{
		{Owner: 1, Slot: 0, Design: 1},
		{Owner: 0, Starbase: true, Slot: 0, Design: 1},
		{Owner: 0, Slot: 4, Design: 0},
		{Owner: 0, Slot: 6, Design: 1},
	}
	// Designs 0 (slot 4) and 1 (slot 6), then the slotless 2 and 3.
	if got, want := g.slotOrder(&f), []int{3, 1, 2, 0}; !reflect.DeepEqual(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
}

func TestBinaryOnlyWarp10SlotOrder(t *testing.T) {
	// Stacks draw in their design-slot order (BINARY-ONLY): design 0 in
	// slot 5 draws after its unrated copy, design 2, in slot 2.
	g := drawGame(false, mover(1, 10, Stack{Design: 0, Count: 1}, Stack{Design: 2, Count: 1}))
	g.Designs = append(g.Designs, g.Designs[0])
	g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 5, Design: 0}, {Owner: 0, Slot: 2, Design: 2}}
	g.moveAll(&scriptRand{draws: []int{0, 1}})
	if s := g.Fleets[0].Stacks; len(s) != 1 || s[0].Design != 0 {
		t.Errorf("stacks %+v, want design 2 (slot 2, first draw 0) lost", s)
	}
}

func TestConfirmedMoveDrawOrderByFleet(t *testing.T) {
	// KERNEL.md "Random draws" item 3 (KX-006): each fleet makes its stall
	// draw and then its loss draws before the next fleet draws.
	g := drawGame(true, mover(1, 10, Stack{Design: 0, Count: 2}), mover(2, 10, Stack{Design: 0, Count: 1}))
	// Fleet 1: stall 1, losses 0 1; fleet 2: stall 1, loss 0.
	g.moveAll(&scriptRand{draws: []int{1, 0, 1, 1, 0}})
	if g.Fleets[0].Stacks[0].Count != 1 || len(g.Fleets[1].Stacks) != 0 {
		t.Errorf("fleets %+v / %+v: want fleet 1 down to 1 ship and fleet 2 lost", g.Fleets[0].Stacks, g.Fleets[1].Stacks)
	}
}

func TestPredictionWarp10LostFleetRemoved(t *testing.T) {
	// A fleet lost at warp 10 is gone from the year's game, also in a game
	// without space objects (removed with the other shipless fleets).
	g := drawGame(false, mover(1, 10, Stack{Design: 0, Count: 1}), mover(2, 0, Stack{Design: 0, Count: 1}))
	g.Year = 2400
	r, err := GenerateTurn(g, nil, &seqRand{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Game.Fleets) != 1 || r.Game.Fleets[0].ID != 2 {
		t.Errorf("fleets %+v, want only fleet 2", r.Game.Fleets)
	}
}

func TestAssumptionChaseTargetLostAtWarp10(t *testing.T) {
	// ASSUMPTION (NOT RUN; ORDERS.md "Target destroyed during movement"
	// is measured for mines only): a chaser of a fleet that loses every
	// ship at warp 10 goes to the coordinates its waypoint held before
	// movement, which settlement keeps. The lost fleet does not move, so
	// these are also where it was lost.
	target := mover(1, 10, Stack{Design: 0, Count: 1})
	chaser := Fleet{ID: 2, Number: 2, Stacks: []Stack{{Design: 1, Count: 1}}, Fuel: 100, Pos: Point{0, 60},
		Waypoints: []Waypoint{{Pos: Point{0, 0}, Warp: 5, Target: TargetFleet, ID: 1}}}
	g := drawGame(false, target, chaser)
	ev, _ := g.moveAll(&scriptRand{draws: []int{0}})
	if len(ev) == 0 || ev[0].Kind != EventWarp10FleetLost {
		t.Fatalf("events %v, want the target lost", ev)
	}
	c := g.Fleets[1]
	if c.Pos != (Point{0, 35}) || c.Waypoints[0].Pos != (Point{0, 0}) {
		t.Errorf("chaser at %v with waypoint %v, want (0,35) heading for (0,0)", c.Pos, c.Waypoints[0].Pos)
	}
}

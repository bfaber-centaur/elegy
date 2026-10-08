package engine

import (
	"reflect"
	"testing"
)

func at(x, y, warp int) Waypoint { return Waypoint{Pos: Point{x, y}, Warp: warp} }

func TestConfirmedReachingAWaypoint(t *testing.T) {
	// ORDERS.md "Reaching a waypoint" (WU-A, WU-FALLBACK): the fleet has
	// reached (120,0), its first waypoint, from (100,0).
	reached := at(120, 0, 5)
	tests := []struct {
		name   string
		repeat bool
		after  []Waypoint // the waypoints after the reached one
		want   []Waypoint
		idle   bool
	}{
		{"repeat off drops it", false, []Waypoint{at(100, 0, 5)}, []Waypoint{at(100, 0, 5)}, false},
		{"repeat on moves it to the end", true, []Waypoint{at(100, 0, 5)}, []Waypoint{at(100, 0, 5), reached}, false},
		{"one forward leg collapses even with repeat", true, nil, nil, true},
		{"coincident last waypoint is not appended again", true, []Waypoint{at(120, 0, 5)}, []Waypoint{at(120, 0, 5)}, false},
		{"last waypoint reached, repeat off", false, nil, nil, true},
	}
	for _, tt := range tests {
		f := Fleet{Pos: Point{120, 0}, Repeat: tt.repeat, Waypoints: append([]Waypoint{reached}, tt.after...)}
		var g Game
		ev := g.arrive(&f)
		if !reflect.DeepEqual(f.Waypoints, tt.want) {
			t.Errorf("%s: waypoints %v, want %v", tt.name, f.Waypoints, tt.want)
		}
		if idle := len(ev) == 1 && ev[0].Kind == EventFleetArrived; idle != tt.idle {
			t.Errorf("%s: events %v, want idle %v", tt.name, ev, tt.idle)
		}
	}
}

func TestPredictionPatrolNeverRepeats(t *testing.T) {
	// ORDERS.md "Reaching a waypoint": a patrol waypoint never repeats
	// (MEASURED, WU wuPNR).
	patrol := Waypoint{Pos: Point{120, 0}, Warp: 5, Task: Task{Kind: TaskPatrol, Range: 50}}
	f := Fleet{Pos: Point{120, 0}, Repeat: true, Waypoints: []Waypoint{patrol, at(100, 0, 5)}}
	var g Game
	g.arrive(&f)
	if !reflect.DeepEqual(f.Waypoints, []Waypoint{at(100, 0, 5)}) || f.Task.Kind != TaskPatrol {
		t.Errorf("waypoints %v task %v, want only (100,0) and the patrol task", f.Waypoints, f.Task)
	}
}

func TestConfirmedWaypointCheck(t *testing.T) {
	// ORDERS.md "Targets that moved, died or were captured" (WU-A): a
	// waypoint aimed at a fleet that exists takes its position, whoever
	// owns it; one aimed at a fleet that is gone becomes a plain go-to at
	// its last coordinates.
	g := Game{Fleets: []Fleet{
		{ID: 1, Owner: 0, Waypoints: []Waypoint{{Pos: Point{180, 60}, Target: TargetFleet, ID: 2}, {Pos: Point{180, 70}, Target: TargetFleet, ID: 99}}},
		{ID: 2, Owner: 1, Pos: Point{180, 85}},
	}}
	g.waypointCheck()
	want := []Waypoint{{Pos: Point{180, 85}, Target: TargetFleet, ID: 2}, {Pos: Point{180, 70}}}
	if got := g.Fleets[0].Waypoints; !reflect.DeepEqual(got, want) {
		t.Errorf("waypoints %v, want %v", got, want)
	}
}

func TestPredictionEmptyTransportIsNoTask(t *testing.T) {
	// A transport task whose unloads ran is no task (MEASURED, TK-501
	// fleet 4).
	g := Game{
		Designs: []Design{{Name: "freighter", CargoCapacity: 100}},
		Planets: []Planet{{ID: 1, Pos: Point{10, 10}, Owner: 0}},
		Players: []Player{{}},
		Fleets: []Fleet{{ID: 1, Owner: 0, Pos: Point{10, 10}, Stacks: []Stack{{Design: 0, Count: 1}},
			Cargo: Cargo{Minerals: Minerals{20, 0, 0}},
			Task:  Task{Kind: TaskTransport, Transport: [NumCargo]Transport{Ironium: {Action: UnloadAll}}}}},
	}
	g.unloadPhase(g.phaseStart())
	if f := g.Fleets[0]; f.Task != (Task{}) || f.Cargo.Minerals[Ironium] != 0 || g.Planets[0].Surface[Ironium] != 20 {
		t.Errorf("task %v cargo %v surface %v, want no task and 20 kT unloaded", f.Task, f.Cargo, g.Planets[0].Surface)
	}
}

func routeGame() Game {
	var lh6 Engine
	lh6.Name = "Long Hump 6"
	for w := 1; w <= 10; w++ {
		lh6.Fuel[w] = []int{0, 0, 20, 60, 100, 100, 105, 450, 750, 900, 1080}[w]
	}
	return Game{
		Designs: []Design{{Name: "scout", Mass: 25, Engine: lh6, Engines: 1, FuelCapacity: 50}},
		Planets: []Planet{
			{ID: 1, Pos: Point{0, 0}, Owner: 0, HasRoute: true, RouteTo: 2},
			{ID: 2, Pos: Point{160, 0}, Owner: 1},
			{ID: 3, Pos: Point{0, 50}, Owner: 1, HasRoute: true, RouteTo: 2},
		},
		Players: []Player{{}, {}},
	}
}

func TestConfirmedRouteTask(t *testing.T) {
	// ORDERS.md "Route task" (WU-ROUTE): at its owner's planet with a route
	// destination, an idle fleet carrying the route task is sent on with
	// the route task again, at the route warp, whoever owns the
	// destination.
	g := routeGame()
	f := Fleet{ID: 1, Owner: 0, Pos: Point{0, 0}, Stacks: []Stack{{Design: 0, Count: 1}}, Fuel: 2000, Task: Task{Kind: TaskRoute}}
	ev := g.routeTask(&f)
	w := g.routeWarp(&f, 0, 1)
	want := []Waypoint{{Pos: Point{160, 0}, Warp: w, Target: TargetPlanet, ID: 2, Task: Task{Kind: TaskRoute}}}
	if w == 0 || !reflect.DeepEqual(f.Waypoints, want) || len(ev) != 1 || ev[0].Kind != EventFleetRouted {
		t.Errorf("waypoints %v events %v, want %v and a routed message", f.Waypoints, ev, want)
	}
	// Another player's planet, even with a route: the fleet stays idle.
	f = Fleet{ID: 1, Owner: 0, Pos: Point{0, 50}, Stacks: []Stack{{Design: 0, Count: 1}}, Fuel: 2000, Task: Task{Kind: TaskRoute}}
	if ev := g.routeTask(&f); f.Waypoints != nil || ev != nil {
		t.Errorf("foreign planet: waypoints %v events %v, want idle", f.Waypoints, ev)
	}
}

// transferGame has player 0's fleet 1 (two ships of design 0) ordered to
// go to player 1, whose slot 0 holds design 1, a copy of design 0.
func transferGame() Game {
	d := Design{Name: "freighter", CargoCapacity: 100}
	return Game{
		Designs:     []Design{d, d},
		DesignSlots: []DesignSlot{{Owner: 0, Slot: 0, Design: 0}, {Owner: 1, Slot: 0, Design: 1}},
		Players:     []Player{{}, {}},
		Fleets: []Fleet{
			{ID: 1, Number: 1, Owner: 0, Pos: Point{5, 5}, Stacks: []Stack{{Design: 0, Count: 2}}, Fuel: 30, Cargo: Cargo{Minerals: Minerals{100, 0, 0}},
				Waypoints: []Waypoint{at(50, 50, 5)}, Task: Task{Kind: TaskTransferFleet, Player: 1}},
			{ID: 2, Number: 1, Owner: 1, Pos: Point{9, 9}},
		},
	}
}

func TestConfirmedTransferFleet(t *testing.T) {
	// ORDERS.md "Transfer fleet", PARITY.md "Transfer fleet task" (FO-04,
	// WU-B2): the recipient gets a fleet with the same ships, cargo and
	// fuel and its lowest unused number; the design goes into a free slot
	// as a new copy, not matched to the recipient's existing copy.
	g := transferGame()
	ev := g.loadPass(true)
	if len(g.Fleets) != 2 {
		t.Fatalf("fleets %v, want the recipient's two", g.Fleets)
	}
	nf := g.Fleets[1]
	if nf.Owner != 1 || nf.Number != 2 || nf.Pos != (Point{5, 5}) || nf.Fuel != 30 || nf.Cargo.Minerals[Ironium] != 100 || nf.Waypoints != nil || nf.Task != (Task{}) {
		t.Errorf("new fleet %+v", nf)
	}
	d, ok := g.PlayerDesign(1, false, 1)
	if !ok || d != 2 || nf.Stacks[0] != (Stack{Design: 2, Count: 2}) {
		t.Errorf("stack %v, recipient slot 1 = %d %v; want a new copy (design 2) in slot 1", nf.Stacks, d, ok)
	}
	if len(ev) != 2 || ev[0].Kind != EventFleetGiven || ev[1].Kind != EventFleetReceived || ev[1].Player != 1 {
		t.Errorf("events %v", ev)
	}

	// ASSUMPTION W2: a slot of the recipient's holding the same design
	// entry matches, and no slot is added.
	g = transferGame()
	g.DesignSlots[1].Design = 0
	g.loadPass(true)
	if len(g.DesignSlots) != 2 || g.Fleets[1].Stacks[0].Design != 0 {
		t.Errorf("matching slot: slots %v stacks %v", g.DesignSlots, g.Fleets[1].Stacks)
	}
}

func TestConfirmedTransferFleetRefused(t *testing.T) {
	// ORDERS.md "Transfer fleet": refused when the recipient treats the
	// giver as an enemy or the fleet carries colonists (CONFIRMED), or the
	// recipient is not a live player (BINARY-ONLY). The task is cleared
	// and the fleet keeps its owner (MEASURED, WU-B2).
	tests := []struct {
		name string
		edit func(*Game)
		why  GiftRefusal
	}{
		{"enemy", func(g *Game) { g.Players[1].Relations = []Relation{RelationEnemy, RelationFriend} }, GiftRefusedByRecipient},
		{"colonists", func(g *Game) { g.Fleets[0].Cargo.Colonists = 1 }, GiftColonistsAboard},
		{"dead", func(g *Game) { g.Players[1].Dead = true }, GiftRecipientAbsent},
		{"no slot", func(g *Game) {
			for s := 1; s < maxShipDesigns; s++ {
				g.DesignSlots = append(g.DesignSlots, DesignSlot{Owner: 1, Slot: s, Design: 1})
			}
		}, GiftNoRoom},
	}
	for _, tt := range tests {
		g := transferGame()
		tt.edit(&g)
		ev := g.loadPass(true)
		f := g.Fleets[0]
		if f.ID != 1 || f.Owner != 0 || f.Task != (Task{}) || len(ev) == 0 || ev[0].Kind != EventFleetGiftRefused || GiftRefusal(ev[0].Count) != tt.why {
			t.Errorf("%s: fleet %+v events %v", tt.name, f, ev)
		}
	}
}

func TestPredictionTransferAfterMovementOnly(t *testing.T) {
	// KERNEL.md "Turn order": the transfer-fleet task is a load-phase task
	// after movement (step 6c.5), not before movement (step 2.5).
	g := transferGame()
	g.loadPass(false)
	if g.Fleets[0].Owner != 0 || g.Fleets[0].Task.Kind != TaskTransferFleet {
		t.Errorf("before movement: fleet %+v", g.Fleets[0])
	}
}

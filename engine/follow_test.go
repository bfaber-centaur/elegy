package engine

import (
	"errors"
	"reflect"
	"testing"
)

// followLab is player 0's scouts in deep space, n fleets with ids 1..n at
// (1200, 1200), with fuel for the year; player 1 has none, so no battle.
func followLab(t *testing.T, n int) *tkLab {
	l := newTKLab(t, 3)
	scout := l.design("Scout", SlotFill{0, "Long Hump 6", 1})
	pi := l.planet(NoOwner, 0, 0)
	for range n {
		fi := l.fleet(0, pi, 0, Stack{Design: scout, Count: 1})
		l.g.Fleets[fi].Pos = Point{1200, 1200}
		l.g.Fleets[fi].Fuel = 300
	}
	return l
}

// followTurn generates the lab's year with player 0's orders.
func followTurn(t *testing.T, l *tkLab, orders ...Order) TurnResult {
	t.Helper()
	g := withRules(l.g)
	r, err := GenerateTurn(g, []PlayerOrders{{Player: 0, GameID: g.ID, Year: g.Year, Orders: orders}}, &seqRand{})
	if err != nil {
		t.Fatal(err)
	}
	for _, res := range r.Orders {
		if res.Err != nil {
			t.Fatalf("order %d: %v", res.Index, res.Err)
		}
	}
	return r
}

// followEvents is the follow messages among events, by fleet.
func followEvents(events []Event) map[int][]EventKind {
	got := map[int][]EventKind{}
	for _, e := range events {
		if e.Kind == EventFollowNoLeader || e.Kind == EventFollowDone {
			got[e.Fleet] = append(got[e.Fleet], e.Kind)
		}
	}
	return got
}

// followed is fleet id in g.
func followed(t *testing.T, g Game, id int) Fleet {
	t.Helper()
	i := g.fleetIndex(id)
	if i < 0 {
		t.Fatalf("fleet %d gone", id)
	}
	return g.Fleets[i]
}

func TestFollowCopiesLeaderWaypoint(t *testing.T) {
	// KERNEL.md "Turn order" step 1a.3 (BINARY-ONLY): the follower gets a
	// copy of its leader's next waypoint carrying its own waypoint-0 task.
	l := followLab(t, 2)
	leg := Waypoint{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace, Task: Task{Kind: TaskPatrol, Range: 50}}
	l.g.Fleets[0].Waypoints = []Waypoint{leg}
	g := l.g.clone()
	a := ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 2, Leader: 1, Task: Task{Kind: TaskNone}}}}}, []int{0})
	if a.Results[0].Err != nil {
		t.Fatal(a.Results[0].Err)
	}
	following, ev := g.follow(a.follows)
	want := leg
	want.Task = Task{Kind: TaskNone}
	if got := g.Fleets[1].Waypoints; !reflect.DeepEqual(got, []Waypoint{want}) || len(ev) != 0 || !following[2] {
		t.Errorf("waypoints %v events %v following %v, want %v, none, fleet 2", got, ev, following, want)
	}
}

func TestFollowOneYear(t *testing.T) {
	// ORDERS.md "Waypoint 0 aimed at a fleet" (stars-elegy c3aab85): a
	// follower of a fleet with a next waypoint moves with it and the two
	// end at the same position (MEASURED for the computer players' orders,
	// AI-27); after the year the follower has no waypoint left, reached or
	// not, and is told so at the end of movement (0x137, CONFIRMED
	// fo/fo02, fo/fo03). The leader keeps its own orders.
	l := followLab(t, 2)
	leg := Waypoint{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}
	l.g.Fleets[0].Waypoints = []Waypoint{leg}
	r := followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1})
	lead, f := followed(t, r.Game, 1), followed(t, r.Game, 2)
	if f.Pos != lead.Pos || f.Pos == (Point{1200, 1200}) {
		t.Errorf("follower at %v, leader at %v; want both moved to one spot", f.Pos, lead.Pos)
	}
	if f.Waypoints != nil || !reflect.DeepEqual(lead.Waypoints, []Waypoint{leg}) {
		t.Errorf("follower waypoints %v, leader %v; want none and %v", f.Waypoints, lead.Waypoints, leg)
	}
	if got := followEvents(r.Events); !reflect.DeepEqual(got, map[int][]EventKind{2: {EventFollowDone}}) {
		t.Errorf("follow messages %v, want 0x137 for fleet 2", got)
	}
	// Reaching the copied waypoint: dropped all the same, and still told.
	l = followLab(t, 2)
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1210, 1200}, Warp: 5, Target: TargetSpace}}
	r = followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1})
	if f := followed(t, r.Game, 2); f.Pos != (Point{1210, 1200}) || f.Waypoints != nil {
		t.Errorf("follower at %v with %v, want at (1210,1200) with none", f.Pos, f.Waypoints)
	}
	if got := followEvents(r.Events); !reflect.DeepEqual(got, map[int][]EventKind{2: {EventFollowDone}}) {
		t.Errorf("follow messages %v, want 0x137 for fleet 2", got)
	}
}

func TestFollowNoLeader(t *testing.T) {
	// MESSAGES.md 0x138 (CONFIRMED fo/fo04): a follower whose leader has no
	// orders and is not following, or no longer exists, is told and stops
	// following; it does not move (none of AI-27's 7 such followers moved)
	// and gets no 0x137.
	l := followLab(t, 2)
	r := followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1})
	if f := followed(t, r.Game, 2); f.Pos != (Point{1200, 1200}) || f.Waypoints != nil {
		t.Errorf("idle leader: follower at %v with %v, want unmoved", f.Pos, f.Waypoints)
	}
	if got := followEvents(r.Events); !reflect.DeepEqual(got, map[int][]EventKind{2: {EventFollowNoLeader}}) {
		t.Errorf("idle leader: follow messages %v, want 0x138 for fleet 2", got)
	}
	// The leader merged away later in the replay (ASSUMPTION F1): gone.
	l = followLab(t, 3)
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	r = followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1}, MergeOrder{Into: 3, From: []int{1}})
	if got := followEvents(r.Events); !reflect.DeepEqual(got, map[int][]EventKind{2: {EventFollowNoLeader}}) {
		t.Errorf("merged leader: follow messages %v, want 0x138 for fleet 2", got)
	}
	// A leader that does not exist when the order applies rejects it (F1),
	// as does the fleet itself.
	for _, leader := range []int{9, 2} {
		g := withRules(followLab(t, 2).g)
		a := ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 2, Leader: leader}}}}, []int{0})
		if !errors.Is(a.Results[0].Err, ErrNoSuchObject) || a.follows[2] != 0 {
			t.Errorf("leader %d: %v, follows %v; want rejected", leader, a.Results[0].Err, a.follows)
		}
	}
}

func TestFollowChains(t *testing.T) {
	// KERNEL.md step 1a.3: a leader that is itself a follower passes on
	// the waypoint it copied; up to 8 passes resolve chains. Fleets 1..10
	// each follow the next and fleet 10 follows fleet 11, which has a
	// waypoint: in fleet order each pass resolves one more link, so the 8
	// passes reach fleets 10 down to 3, and fleets 1 and 2 are left
	// unresolved, unmoved, with no 0x138 and with 0x137 (ASSUMPTION F3).
	l := followLab(t, 11)
	l.g.Fleets[10].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	var orders []Order
	for id := 1; id <= 10; id++ {
		orders = append(orders, FollowOrder{Fleet: id, Leader: id + 1})
	}
	r := followTurn(t, l, orders...)
	lead := followed(t, r.Game, 11)
	done := map[int][]EventKind{}
	for id := 1; id <= 10; id++ {
		done[id] = []EventKind{EventFollowDone}
		f := followed(t, r.Game, id)
		if moved := f.Pos == lead.Pos; moved != (id >= 3) || f.Waypoints != nil {
			t.Errorf("fleet %d at %v with %v; leader at %v; want moved %v", id, f.Pos, f.Waypoints, lead.Pos, id >= 3)
		}
	}
	if got := followEvents(r.Events); !reflect.DeepEqual(got, done) {
		t.Errorf("follow messages %v, want 0x137 for fleets 1..10", got)
	}
	// The other way round, fleet k following fleet k−1 and fleet 1 the
	// leader, one pass resolves the whole chain (ASSUMPTION F2).
	l = followLab(t, 11)
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	orders = nil
	for id := 2; id <= 11; id++ {
		orders = append(orders, FollowOrder{Fleet: id, Leader: id - 1})
	}
	r = followTurn(t, l, orders...)
	lead = followed(t, r.Game, 1)
	for id := 2; id <= 11; id++ {
		if f := followed(t, r.Game, id); f.Pos != lead.Pos {
			t.Errorf("reverse chain: fleet %d at %v, leader at %v", id, f.Pos, lead.Pos)
		}
	}
}

func TestFollowCycle(t *testing.T) {
	// MESSAGES.md 0x138: circular chains get no message. Fleets 1 and 2
	// following each other never resolve: no waypoint, no move, no 0x138,
	// and 0x137 at the end of movement (ASSUMPTION F3).
	l := followLab(t, 2)
	r := followTurn(t, l, FollowOrder{Fleet: 1, Leader: 2}, FollowOrder{Fleet: 2, Leader: 1})
	for id := 1; id <= 2; id++ {
		if f := followed(t, r.Game, id); f.Pos != (Point{1200, 1200}) || f.Waypoints != nil {
			t.Errorf("fleet %d at %v with %v, want unmoved with none", id, f.Pos, f.Waypoints)
		}
	}
	want := map[int][]EventKind{1: {EventFollowDone}, 2: {EventFollowDone}}
	if got := followEvents(r.Events); !reflect.DeepEqual(got, want) {
		t.Errorf("follow messages %v, want %v", got, want)
	}
}

func TestFollowReplacedByWaypointOrder(t *testing.T) {
	// A later waypoint order for the same fleet replaces its waypoints,
	// waypoint 0 included: the fleet no longer follows.
	l := followLab(t, 2)
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	own := Waypoint{Pos: Point{1200, 1300}, Warp: 4, Target: TargetSpace}
	r := followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1}, WaypointOrder{Fleet: 2, Waypoints: []Waypoint{own}})
	if f := followed(t, r.Game, 2); !reflect.DeepEqual(f.Waypoints, []Waypoint{own}) {
		t.Errorf("waypoints %v, want %v", f.Waypoints, own)
	}
	if got := followEvents(r.Events); len(got) != 0 {
		t.Errorf("follow messages %v, want none", got)
	}
}

func TestFollowNewFleetName(t *testing.T) {
	// A follow order may name a fleet split off earlier in the same file
	// by its new-fleet name (SplitOrder.NewFleet), as follower or leader.
	l := followLab(t, 2)
	l.g.Fleets[0].Stacks[0].Count = 2
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	r := followTurn(t, l,
		SplitOrder{Fleet: 1, Ships: []Stack{{Design: 0, Count: 1}}, NewFleet: -1},
		FollowOrder{Fleet: -1, Leader: 1},
		FollowOrder{Fleet: 2, Leader: -1})
	lead := followed(t, r.Game, 1)
	for _, f := range r.Game.Fleets {
		if f.Pos != lead.Pos {
			t.Errorf("fleet %d at %v, leader at %v", f.ID, f.Pos, lead.Pos)
		}
	}
	if got := followEvents(r.Events); len(got) != 2 {
		t.Errorf("follow messages %v, want 0x137 for two fleets", got)
	}
}

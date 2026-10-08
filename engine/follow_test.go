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

// doneFor is player's 0x137 for fleet id.
func doneFor(player, id int) Event {
	return Event{Kind: EventFollowDone, Player: player, Planet: -1, Fleet: id}
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
	done, ev := g.follow(a.follows)
	want := leg
	want.Task = Task{Kind: TaskNone}
	if got := g.Fleets[1].Waypoints; !reflect.DeepEqual(got, []Waypoint{want}) || len(ev) != 0 || !reflect.DeepEqual(done, []Event{doneFor(0, 2)}) {
		t.Errorf("waypoints %v events %v done %v, want %v, none, fleet 2", got, ev, done, want)
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
	// unresolved, unmoved, with no 0x138 and no 0x137 (ASSUMPTION F3).
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
		if id >= 3 {
			done[id] = []EventKind{EventFollowDone}
		}
		f := followed(t, r.Game, id)
		if moved := f.Pos == lead.Pos; moved != (id >= 3) || f.Waypoints != nil {
			t.Errorf("fleet %d at %v with %v; leader at %v; want moved %v", id, f.Pos, f.Waypoints, lead.Pos, id >= 3)
		}
	}
	if got := followEvents(r.Events); !reflect.DeepEqual(got, done) {
		t.Errorf("follow messages %v, want 0x137 for fleets 3..10", got)
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
	// following each other never resolve: no waypoint, no move, no 0x138
	// and no 0x137 (ASSUMPTION F3 for two or more fleets).
	l := followLab(t, 2)
	r := followTurn(t, l, FollowOrder{Fleet: 1, Leader: 2}, FollowOrder{Fleet: 2, Leader: 1})
	for id := 1; id <= 2; id++ {
		if f := followed(t, r.Game, id); f.Pos != (Point{1200, 1200}) || f.Waypoints != nil {
			t.Errorf("fleet %d at %v with %v, want unmoved with none", id, f.Pos, f.Waypoints)
		}
	}
	if got := followEvents(r.Events); len(got) != 0 {
		t.Errorf("follow messages %v, want none", got)
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

func TestFollowOrderReplacesOrders(t *testing.T) {
	// ORDERS.md "Waypoint 0 aimed at a fleet": the order is a waypoint-0
	// change "which leaves the fleet with that one waypoint", so the
	// fleet's earlier waypoints go and waypoint 0's task is the order's.
	// With an idle leader the follower therefore stays put (0x138).
	l := followLab(t, 2)
	l.g.Fleets[1].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	l.g.Fleets[1].Task = Task{Kind: TaskPatrol, Range: 50}
	order := Task{Kind: TaskTransport}
	order.Transport[Ironium] = Transport{LoadExactly, 10}
	g := withRules(l.g)
	a := ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 2, Leader: 1, Task: order}}}}, []int{0})
	if a.Results[0].Err != nil {
		t.Fatal(a.Results[0].Err)
	}
	if f := g.Fleets[1]; f.Waypoints != nil || !reflect.DeepEqual(f.Task, order) {
		t.Errorf("waypoints %v task %+v, want none and %+v", f.Waypoints, f.Task, order)
	}
	r := followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1})
	if f := followed(t, r.Game, 2); f.Pos != (Point{1200, 1200}) {
		t.Errorf("follower of an idle leader moved to %v", f.Pos)
	}
	// An invalid task rejects the order, as in a waypoint order.
	bad := Task{Kind: TaskTransport}
	bad.Transport[Ironium] = Transport{LoadExactly, -1}
	g = withRules(followLab(t, 2).g)
	a = ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 2, Leader: 1, Task: bad}}}}, []int{0})
	if !errors.Is(a.Results[0].Err, ErrOutOfRange) || a.follows[2] != 0 {
		t.Errorf("negative transport amount: %v, follows %v; want rejected", a.Results[0].Err, a.follows)
	}
}

func TestFollowAnotherPlayersFleet(t *testing.T) {
	// KERNEL.md step 1a.3 names "another fleet", whoever owns it: player
	// 0's fleet follows player 1's and copies its next waypoint.
	l := followLab(t, 2)
	l.g.Fleets[0].Owner = 1
	leg := Waypoint{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}
	l.g.Fleets[0].Waypoints = []Waypoint{leg}
	g := withRules(l.g)
	a := ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 2, Leader: 1}}}}, []int{0})
	if a.Results[0].Err != nil {
		t.Fatal(a.Results[0].Err)
	}
	done, _ := g.follow(a.follows)
	if !reflect.DeepEqual(done, []Event{doneFor(0, 2)}) || !reflect.DeepEqual(g.Fleets[1].Waypoints, []Waypoint{leg}) {
		t.Errorf("done %v waypoints %v, want fleet 2 with %v", done, g.Fleets[1].Waypoints, leg)
	}
}

func TestFollowMergedAway(t *testing.T) {
	// MESSAGES.md 0x137 (CONFIRMED fo/fo03 G): a follower whose merge task
	// targets its moving leader merges into it before movement and no
	// longer exists, and still gets 0x137 for its own id. The leader
	// moves with the follower's ships.
	l := followLab(t, 2)
	leg := Waypoint{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}
	l.g.Fleets[0].Waypoints = []Waypoint{leg}
	r := followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1, Task: Task{Kind: TaskMerge, Fleet: 1}})
	if i := r.Game.fleetIndex(2); i >= 0 {
		t.Fatalf("follower still there: %+v", r.Game.Fleets[i])
	}
	if lead := followed(t, r.Game, 1); lead.Pos == (Point{1200, 1200}) || lead.Stacks[0].Count != 2 {
		t.Errorf("leader at %v with %v, want moved with both ships", lead.Pos, lead.Stacks)
	}
	var done []Event
	for _, e := range r.Events {
		if e.Kind == EventFollowDone || e.Kind == EventFollowNoLeader {
			done = append(done, e)
		}
	}
	if !reflect.DeepEqual(done, []Event{doneFor(0, 2)}) {
		t.Errorf("follow messages %v, want 0x137 for fleet 2", done)
	}
}

func TestFollowLeaderElsewhere(t *testing.T) {
	// FO-03-F (MESSAGES.md 0x138, ORDERS.md "Waypoint 0 aimed at a
	// fleet"; outcome CONFIRMED, re-choice BINARY-ONLY): a follower with a
	// merge task aimed at an idle leader 195 ly away is re-aimed at itself,
	// the only fleet of the leader's owner at its position. It does not
	// move or merge, gets no 0x138 or 0x137, and its merge with itself
	// completes, clearing the task.
	l := followLab(t, 2)
	l.g.Fleets[0].Pos = Point{1395, 1200}
	r := followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1, Task: Task{Kind: TaskMerge, Fleet: 1}})
	f, lead := followed(t, r.Game, 2), followed(t, r.Game, 1)
	if f.Pos != (Point{1200, 1200}) || f.Waypoints != nil || f.Task != (Task{}) || f.Stacks[0].Count != 1 {
		t.Errorf("follower %+v, want unmoved, unmerged, no waypoints or task", f)
	}
	if lead.Pos != (Point{1395, 1200}) || lead.Stacks[0].Count != 1 {
		t.Errorf("leader %+v, want unmoved and unmerged", lead)
	}
	for _, e := range r.Events {
		if e.Fleet == 2 && e.Kind != EventFleetArrived {
			t.Errorf("follower message %v, want none (0x04e is not sent by Elegy)", e)
		}
	}
	// A leader elsewhere that has orders is not followed either.
	l = followLab(t, 2)
	l.g.Fleets[0].Pos = Point{1395, 1200}
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	r = followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1})
	if f := followed(t, r.Game, 2); f.Pos != (Point{1200, 1200}) {
		t.Errorf("follower of a distant moving leader moved to %v", f.Pos)
	}
	if got := followEvents(r.Events); len(got) != 0 {
		t.Errorf("follow messages %v, want none", got)
	}
}

func TestFollowLeaderReChosen(t *testing.T) {
	// KERNEL.md step 1a.3 (BINARY-ONLY): a leader elsewhere is replaced by
	// a fleet of the leader's owner at the follower's position, the first
	// in fleet order (ASSUMPTION F5). Fleet 4 is aimed at fleet 1, which is
	// elsewhere; fleets 2 and 3 are at fleet 4's position, and fleet 2,
	// first, has a waypoint, so fleet 4 copies it.
	l := followLab(t, 4)
	l.g.Fleets[0].Pos = Point{1395, 1200}
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1500, 1200}, Warp: 5, Target: TargetSpace}}
	mine := Waypoint{Pos: Point{1200, 1300}, Warp: 5, Target: TargetSpace}
	l.g.Fleets[1].Waypoints = []Waypoint{mine}
	l.g.Fleets[2].Waypoints = []Waypoint{{Pos: Point{1100, 1200}, Warp: 5, Target: TargetSpace}}
	g := withRules(l.g)
	a := ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 4, Leader: 1}}}}, []int{0})
	if a.Results[0].Err != nil {
		t.Fatal(a.Results[0].Err)
	}
	done, ev := g.follow(a.follows)
	if !reflect.DeepEqual(g.Fleets[3].Waypoints, []Waypoint{mine}) || len(ev) != 0 || !reflect.DeepEqual(done, []Event{doneFor(0, 4)}) {
		t.Errorf("waypoints %v events %v done %v, want %v, none, fleet 4", g.Fleets[3].Waypoints, ev, done, mine)
	}
	// A merge task aimed at the old leader is re-aimed with it.
	g = withRules(l.g)
	a = ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 4, Leader: 1, Task: Task{Kind: TaskMerge, Fleet: 1}}}}}, []int{0})
	g.follow(a.follows)
	if got := g.Fleets[3].Task; got != (Task{Kind: TaskMerge, Fleet: 2}) {
		t.Errorf("task %+v, want merge with fleet 2", got)
	}
	// A distant leader of another player with no fleet at the follower's
	// position: treated as gone, 0x138 (ASSUMPTION F6).
	l = followLab(t, 2)
	l.g.Fleets[0].Owner = 1
	l.g.Fleets[0].Pos = Point{1395, 1200}
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	g = withRules(l.g)
	a = ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 2, Leader: 1, Task: Task{Kind: TaskMerge, Fleet: 1}}}}}, []int{0})
	done, ev = g.follow(a.follows)
	if want := []Event{{Kind: EventFollowNoLeader, Player: 0, Planet: -1, Fleet: 2}}; !reflect.DeepEqual(ev, want) || done != nil || g.Fleets[1].Task.Fleet != 1 {
		t.Errorf("events %v done %v task %+v, want %v, none, merge with fleet 1", ev, done, g.Fleets[1].Task, want)
	}
}

func TestFollowDoneInFleetOrder(t *testing.T) {
	// MESSAGES.md "Order within movement": fleets in number order, then
	// 0x137 once every fleet has moved. Fleets 3 and 2 sit in that order
	// in the fleet list; their 0x137 still come in fleet order.
	l := followLab(t, 3)
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5, Target: TargetSpace}}
	l.g.Fleets[1], l.g.Fleets[2] = l.g.Fleets[2], l.g.Fleets[1]
	g := withRules(l.g)
	a := ApplyOrders(&g, []PlayerOrders{{Player: 0, Orders: []Order{FollowOrder{Fleet: 3, Leader: 1}, FollowOrder{Fleet: 2, Leader: 1}}}}, []int{0})
	done, _ := g.follow(a.follows)
	if want := []Event{doneFor(0, 2), doneFor(0, 3)}; !reflect.DeepEqual(done, want) {
		t.Errorf("0x137 %v, want %v", done, want)
	}
}

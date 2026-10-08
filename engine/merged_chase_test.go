package engine

import (
	"reflect"
	"testing"
)

// mergedChaseGame has player 0's fleets 1 and 2 at (1100,1100) and player
// 1's fleet 3 elsewhere; the files merge 1 into 2 and send fleet 3 after
// fleet 1, at the position player 1 saw it.
func mergedChaseGame() (*Game, []PlayerOrders) {
	g := opsGame()
	at := Point{1100, 1100}
	g.Fleets = []Fleet{
		{ID: 1, Number: 1, Owner: 0, Pos: at, Stacks: []Stack{{Design: 0, Count: 1}}},
		{ID: 2, Number: 2, Owner: 0, Pos: at, Stacks: []Stack{{Design: 0, Count: 1}}},
		{ID: 3, Number: 1, Owner: 1, Pos: Point{1050, 1050}, Stacks: []Stack{{Design: 0, Count: 1}}},
	}
	files := []PlayerOrders{
		{Player: 0, GameID: g.ID, Year: g.Year, Orders: []Order{MergeOrder{Into: 2, From: []int{1}}}},
		{Player: 1, GameID: g.ID, Year: g.Year, Orders: []Order{
			WaypointOrder{Fleet: 3, Waypoints: []Waypoint{{Pos: at, Warp: 5, Target: TargetFleet, ID: 1}}},
		}},
	}
	return g, files
}

// ORDERS.md "Targets that moved, died or were captured", "A fleet target
// merged away during order replay" (BINARY-ONLY), merging player first:
// the chase order is accepted with the stale id, and the waypoint check
// after the orders retargets it to the fleet now at its coordinates,
// spending a draw.
func TestMergedChaseMergerFirst(t *testing.T) {
	g, files := mergedChaseGame()
	a := ApplyOrders(g, files, []int{0, 1})
	for _, r := range a.Results {
		if r.Err != nil {
			t.Fatalf("player %d order %d: %v", r.Player, r.Index, r.Err)
		}
	}
	wp := &g.Fleets[g.fleetIndex(3)].Waypoints[0]
	if wp.ID != 1 || wp.Pos != (Point{1100, 1100}) {
		t.Fatalf("after the replay: waypoint %+v, want the stale fleet 1 at (1100,1100)", *wp)
	}
	rng := &recordRand{}
	g.chaseMerged(a.merged, rng)
	g.waypointCheck()
	if wp.Target != TargetFleet || wp.ID != 2 {
		t.Errorf("waypoint %+v, want fleet 2", *wp)
	}
	if !reflect.DeepEqual(rng.bounds, []int{1}) {
		t.Errorf("draws %v, want one rand(1)", rng.bounds)
	}
}

// Chasing player first: the merge retargets the waypoint to the fleet it
// joined, keeping the coordinates, and the waypoint check draws nothing.
func TestMergedChaseChaserFirst(t *testing.T) {
	g, files := mergedChaseGame()
	a := ApplyOrders(g, files, []int{1, 0})
	for _, r := range a.Results {
		if r.Err != nil {
			t.Fatalf("player %d order %d: %v", r.Player, r.Index, r.Err)
		}
	}
	wp := &g.Fleets[g.fleetIndex(3)].Waypoints[0]
	if wp.Target != TargetFleet || wp.ID != 2 || wp.Pos != (Point{1100, 1100}) {
		t.Fatalf("after the replay: waypoint %+v, want fleet 2 at (1100,1100)", *wp)
	}
	rng := &recordRand{}
	g.chaseMerged(a.merged, rng)
	if len(rng.bounds) != 0 || wp.ID != 2 {
		t.Errorf("draws %v, waypoint %+v; want no draws and fleet 2", rng.bounds, *wp)
	}
}

// Merging player first with nothing of the owner's at the order's
// coordinates: the stale id stays, and the "target gone" rule makes the
// waypoint a plain go-to there.
func TestMergedChaseNoCandidate(t *testing.T) {
	g, files := mergedChaseGame()
	seen := Point{1090, 1090}
	files[1].Orders[0] = WaypointOrder{Fleet: 3, Waypoints: []Waypoint{{Pos: seen, Warp: 5, Target: TargetFleet, ID: 1}}}
	a := ApplyOrders(g, files, []int{0, 1})
	rng := &recordRand{}
	g.chaseMerged(a.merged, rng)
	g.waypointCheck()
	wp := g.Fleets[g.fleetIndex(3)].Waypoints[0]
	if wp.Target != TargetSpace || wp.Pos != seen || len(rng.bounds) != 0 {
		t.Errorf("waypoint %+v, draws %v; want a plain go-to at %v and no draws", wp, rng.bounds, seen)
	}
}

// Several candidates (ASSUMPTION W8): the k-th candidate in fleet order
// draws rand(k) and is taken on 0; a candidate some waypoint already
// picked this year draws a further rand(2). Two chasers (fleets 3, then
// 5, in fleet order), two candidates (fleets 2, then 4): fleet 3's draws
// rand(1) = 0, rand(2) = 1 pick fleet 2; fleet 5's rand(1) = 0, the
// rand(2) for the already-picked fleet 2, then rand(2) = 0 pick fleet 4.
func TestMergedChaseSeveralCandidates(t *testing.T) {
	g, files := mergedChaseGame()
	at := Point{1100, 1100}
	g.Fleets = append(g.Fleets,
		Fleet{ID: 4, Number: 3, Owner: 0, Pos: at, Stacks: []Stack{{Design: 0, Count: 1}}},
		Fleet{ID: 5, Number: 2, Owner: 1, Pos: Point{1060, 1060}, Stacks: []Stack{{Design: 0, Count: 1}}},
	)
	files[1].Orders = append(files[1].Orders,
		WaypointOrder{Fleet: 5, Waypoints: []Waypoint{{Pos: at, Warp: 5, Target: TargetFleet, ID: 1}}})
	a := ApplyOrders(g, files, []int{0, 1})
	rng := &seqRand{draws: []int{0, 1, 0, 1, 0}}
	bounds := &recordBounds{r: rng}
	g.chaseMerged(a.merged, bounds)
	if got := []int{g.Fleets[g.fleetIndex(3)].Waypoints[0].ID, g.Fleets[g.fleetIndex(5)].Waypoints[0].ID}; !reflect.DeepEqual(got, []int{2, 4}) {
		t.Errorf("chasers' targets %v, want [2 4]", got)
	}
	if want := []int{1, 2, 1, 2, 2}; !reflect.DeepEqual(bounds.bounds, want) {
		t.Errorf("draws %v, want %v", bounds.bounds, want)
	}
}

// recordBounds records each draw's bound and passes it to r.
type recordBounds struct {
	r      Rand
	bounds []int
}

func (b *recordBounds) Intn(n int) int {
	b.bounds = append(b.bounds, n)
	return b.r.Intn(n)
}

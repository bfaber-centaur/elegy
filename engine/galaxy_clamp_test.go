package engine

import "testing"

// ORDERS.md "Waypoint upkeep", "Outside the galaxy" (stars-elegy
// PARITY.md "A transport task aimed at a packet (ET-2)"): the waypoint
// check clamps each fleet's position into the galaxy box, each axis on
// its own, with no message and no fuel. The lab's galaxy is tiny,
// 1000..1400 on each axis, as in the ET starts.
func clampLab(t *testing.T) *tkLab {
	l := followLab(t, 2)
	l.g.Size = 0
	return l
}

// Fleet K (ET-2c, CONFIRMED by prediction): a stationary fleet with no
// task at (1450, 1200) ends at (1400, 1200), with no message and its fuel
// untouched. Fleet 2 below the box's low edge on the other axis is the
// same rule (BINARY-ONLY).
func TestConfirmedGalaxyClampStationary(t *testing.T) {
	l := clampLab(t)
	l.g.Fleets[0].Pos = Point{1450, 1200}
	l.g.Fleets[1].Pos = Point{1200, 950}
	r := followTurn(t, l)
	for _, c := range []struct {
		id   int
		want Point
	}{{1, Point{1400, 1200}}, {2, Point{1200, 1000}}} {
		if f := followed(t, r.Game, c.id); f.Pos != c.want || f.Fuel != 300 {
			t.Errorf("fleet %d at %v with fuel %d, want %v and 300", c.id, f.Pos, f.Fuel, c.want)
		}
	}
	for _, e := range r.Events {
		if e.Fleet == 1 || e.Fleet == 2 {
			t.Errorf("event %+v, want none", e)
		}
	}
}

// Fleet Q (ET-2b, MEASURED; the end-of-year clamp BINARY-ONLY): from
// (1500, 1350), clamped to (1400, 1350) at the start of the year, it
// flies 81 ly at warp 9 toward (1478, 1288), a waypoint outside the box
// that the check leaves alone, reaches (1463, 1300), and the end-of-year
// check clamps it to (1400, 1300). The waypoint is not reached and stays.
func TestMeasuredGalaxyClampEndOfYear(t *testing.T) {
	l := clampLab(t)
	l.g.Fleets[0].Pos = Point{1500, 1350}
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1478, 1288}, Warp: 9}}
	r := followTurn(t, l)
	f := followed(t, r.Game, 1)
	if f.Pos != (Point{1400, 1300}) || len(f.Waypoints) != 1 || f.Waypoints[0].Pos != (Point{1478, 1288}) {
		t.Errorf("fleet Q at %v, waypoints %v; want (1400, 1300), still bound for (1478, 1288)", f.Pos, f.Waypoints)
	}
	// Fleet 2, inside the box, is not moved.
	if f := followed(t, r.Game, 2); f.Pos != (Point{1200, 1200}) {
		t.Errorf("fleet 2 at %v, want (1200, 1200)", f.Pos)
	}
}

// The check runs before the follow step (KERNEL.md step 1a.3, corrected in
// stars-elegy 33eeb4a; BINARY-ONLY): a follower outside the box is
// clamped onto its leader's position first, so it follows that leader
// rather than being left following itself (F3).
func TestGalaxyClampBeforeFollow(t *testing.T) {
	l := clampLab(t)
	l.g.Fleets[0].Pos = Point{1400, 1200}
	l.g.Fleets[0].Waypoints = []Waypoint{{Pos: Point{1300, 1200}, Warp: 5}}
	l.g.Fleets[1].Pos = Point{1450, 1200}
	r := followTurn(t, l, FollowOrder{Fleet: 2, Leader: 1})
	if got := followEvents(r.Events); len(got[2]) != 1 || got[2][0] != EventFollowDone {
		t.Errorf("follow messages %v, want 0x137 for fleet 2", got)
	}
	if f := followed(t, r.Game, 2); f.Pos != followed(t, r.Game, 1).Pos {
		t.Errorf("follower at %v, want with its leader at %v", f.Pos, followed(t, r.Game, 1).Pos)
	}
}

// Within one check the positions are clamped first, so a waypoint aimed
// at a fleet outside the box takes its clamped position (KERNEL.md step
// 1a.3, BINARY-ONLY). A waypoint aimed at a position outside the box is
// not clamped.
func TestGalaxyClampThenWaypoints(t *testing.T) {
	g := Game{Rules: ElegyRules(), Fleets: []Fleet{
		{ID: 1, Owner: 0, Pos: Point{1200, 1200}, Waypoints: []Waypoint{{Pos: Point{1450, 1200}, Target: TargetFleet, ID: 2}, {Pos: Point{1500, 1500}}}},
		{ID: 2, Owner: 1, Pos: Point{1450, 1200}},
	}}
	g.waypointCheck()
	if wps := g.Fleets[0].Waypoints; wps[0].Pos != (Point{1400, 1200}) || wps[1].Pos != (Point{1500, 1500}) {
		t.Errorf("waypoints %v, want fleet 2's clamped (1400, 1200), then (1500, 1500) as it was", wps)
	}
}

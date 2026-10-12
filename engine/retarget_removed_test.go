package engine

import "testing"

// removedGame has player 0's fleet 1 at planet 5 (1293,1597), player 0's
// fleet 2 at (1200,1200) and fleet 3 in deep space at (1100,1100), and
// player 1's fleet 4 at planet 5 too. Player 1's fleets 5, 6 and 7 chase
// fleets 1, 2 and 3 from afar.
func removedGame() *Game {
	g := opsGame()
	g.Planets = []Planet{{ID: 5, Pos: Point{1293, 1597}}, {ID: 6, Pos: Point{1000, 1000}}}
	chase := func(id int, at Point) []Waypoint {
		return []Waypoint{{Pos: Point{900, 900}}, {Pos: at, Warp: 6, Target: TargetFleet, ID: id}}
	}
	g.Fleets = []Fleet{
		{ID: 1, Number: 1, Owner: 0, Pos: Point{1293, 1597}},
		{ID: 2, Number: 2, Owner: 0, Pos: Point{1200, 1200}},
		{ID: 3, Number: 3, Owner: 0, Pos: Point{1100, 1100}},
		{ID: 4, Number: 1, Owner: 1, Pos: Point{1293, 1597}},
		{ID: 5, Number: 2, Owner: 1, Pos: Point{900, 900}, Waypoints: chase(1, Point{1290, 1590})},
		{ID: 6, Number: 3, Owner: 1, Pos: Point{900, 900}, Waypoints: chase(2, Point{1200, 1200})},
		{ID: 7, Number: 4, Owner: 1, Pos: Point{900, 900}, Waypoints: chase(3, Point{1100, 1100})},
	}
	for i := range g.Fleets {
		g.Fleets[i].Stacks = []Stack{{Design: 0, Count: 1}}
	}
	return g
}

// ORDERS.md "Targets that moved, died or were captured" (BINARY-ONLY):
// removing a fleet retargets every waypoint that names it to what stands
// at its position. MEASURED in RG-002 Y3 and Y4 (8 of 8 seeds): a fleet
// destroyed at planet 51 left the waypoint chasing it aimed at the
// planet, although two of the chasing player's own fleets stood there
// (ASSUMPTION W10: only the removed fleet's owner's fleets are taken).
func TestMeasuredRemovedTargetBecomesPlanet(t *testing.T) {
	g := removedGame()
	g.Fleets[0].Stacks[0].Count = 0 // destroyed in battle
	removeEmptyFleets(g)
	if g.fleetIndex(1) >= 0 {
		t.Fatal("fleet 1 not removed")
	}
	wp := g.Fleets[g.fleetIndex(5)].Waypoints[1]
	if wp.Target != TargetPlanet || wp.ID != 5 || wp.Pos != (Point{1293, 1597}) {
		t.Errorf("waypoint %+v, want planet 5 at (1293,1597)", wp)
	}
	for _, id := range []int{6, 7} {
		if wp := g.Fleets[g.fleetIndex(id)].Waypoints[1]; wp.Target != TargetFleet {
			t.Errorf("fleet %d's waypoint %+v changed", id, wp)
		}
	}
}

// A fleet of the removed fleet's owner standing there is taken, before the
// planet (BINARY-ONLY), whichever rule removes the fleet; with nothing
// there the waypoint keeps the stale id, and the waypoint check makes it
// deep space at its last position (MEASURED WF-04).
func TestBinaryOnlyRemovedTargetRetargets(t *testing.T) {
	g := removedGame()
	g.Fleets = append(g.Fleets,
		Fleet{ID: 8, Number: 4, Owner: 0, Pos: Point{1200, 1200}, Stacks: []Stack{{Design: 0, Count: 1}}},
		Fleet{ID: 9, Number: 5, Owner: 0, Pos: Point{1200, 1200}, Stacks: []Stack{{Design: 0, Count: 1}},
			Waypoints: []Waypoint{{Pos: Point{1200, 1200}}, {Pos: Point{1200, 1200}, Target: TargetFleet, ID: 2}}},
	)
	g.removeFleets(map[int]bool{2: true, 3: true, 8: true})
	// Fleet 9 is never its own target: its waypoint keeps the stale id.
	if wp := g.Fleets[g.fleetIndex(9)].Waypoints[1]; wp.Target != TargetFleet || wp.ID != 2 {
		t.Errorf("fleet 9's waypoint %+v, want the stale fleet 2", wp)
	}
	if wp := g.Fleets[g.fleetIndex(6)].Waypoints[1]; wp.Target != TargetFleet || wp.ID != 9 || wp.Pos != (Point{1200, 1200}) {
		t.Errorf("fleet 6's waypoint %+v, want fleet 9 (fleet 8 removed too)", wp)
	}
	wp := &g.Fleets[g.fleetIndex(7)].Waypoints[1]
	if wp.Target != TargetFleet || wp.ID != 3 {
		t.Errorf("fleet 7's waypoint %+v, want the stale fleet 3", *wp)
	}
	g.waypointCheck()
	if wp.Target != TargetSpace || wp.Pos != (Point{1100, 1100}) {
		t.Errorf("after the check %+v, want deep space at (1100,1100)", *wp)
	}
	// Retargeted to a fleet, the waypoint now follows it.
	g.Fleets[g.fleetIndex(9)].Pos = Point{1250, 1250}
	g.waypointCheck()
	if wp := g.Fleets[g.fleetIndex(6)].Waypoints[1]; wp.Pos != (Point{1250, 1250}) {
		t.Errorf("fleet 6's waypoint %+v, want at fleet 9", wp)
	}
}

// ASSUMPTION W7: a merge order retargets to the fleet joined, even when
// another of the owner's fleets there comes first in fleet order.
func TestMergeOrderRetargetsToJoinedFleet(t *testing.T) {
	g := removedGame()
	g.Fleets = append(g.Fleets,
		Fleet{ID: 8, Number: 4, Owner: 0, Pos: Point{1200, 1200}, Stacks: []Stack{{Design: 0, Count: 1}}},
		Fleet{ID: 9, Number: 5, Owner: 0, Pos: Point{1200, 1200}, Stacks: []Stack{{Design: 0, Count: 1}}},
	)
	if err := g.MergeFleets(0, 9, []int{2}); err != nil {
		t.Fatal(err)
	}
	if wp := g.Fleets[g.fleetIndex(6)].Waypoints[1]; wp.Target != TargetFleet || wp.ID != 9 {
		t.Errorf("waypoint %+v, want fleet 9, the fleet joined", wp)
	}
}

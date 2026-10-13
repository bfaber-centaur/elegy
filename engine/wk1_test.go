package engine

import (
	"math/rand"
	"testing"
)

// TestConfirmedWK1TargetDiedInBattle: a waypoint aimed at a fleet that
// dies in battle becomes, at the end of the year, the planet standing
// exactly where the fleet died, else deep space there, and the follower's
// owner gets 0x028 once (stars-elegy PARITY.md "A waypoint aimed at a
// fleet destroyed in battle (WK-1)", stars-elegy 5962b72, CONFIRMED by
// prediction for waypoint 1, 2 of 2 streams). Player 0's attackers wait at (1100,1100); player 1's prey
// dies there; player 0's follower, 100 ly away and too slow to arrive,
// has its waypoint aimed at the prey.
func TestConfirmedWK1TargetDiedInBattle(t *testing.T) {
	at := Point{1100, 1100}
	for _, c := range []struct {
		name    string
		planets []Planet
		start   Point // the prey's start; it flies to at
		want    Waypoint
		tank    bool // a fleet of the prey's owner that survives there
	}{
		{"A orbiting", []Planet{{ID: 7, Pos: at, Owner: NoOwner}}, at, Waypoint{Pos: at, Warp: 1, Target: TargetPlanet, ID: 7}, false},
		{"D deep space 5 ly from a planet", []Planet{{ID: 10, Pos: Point{1105, 1100}, Owner: NoOwner}}, at, Waypoint{Pos: at, Warp: 1}, false},
		{"M flew onto the planet", []Planet{{ID: 0, Pos: at, Owner: NoOwner}, {ID: 1, Pos: Point{1080, 1100}, Owner: NoOwner}}, Point{1080, 1100}, Waypoint{Pos: at, Warp: 1, Target: TargetPlanet, ID: 0}, false},
		// RG-003 2469 (MEASURED): another fleet of the prey's owner
		// survives there, and the waypoint still goes to the planet.
		{"A with a survivor of the prey's owner", []Planet{{ID: 7, Pos: at, Owner: NoOwner}}, at, Waypoint{Pos: at, Warp: 1, Target: TargetPlanet, ID: 7}, true},
	} {
		prey := testDesign(tFrigate, 20, Slot{tLaser, 1})
		g := combatLabGame(prey, 6, 3)
		g.Planets = c.planets
		g.Fleets[1].Pos = c.start
		if c.start != at {
			g.Fleets[1].Waypoints = []Waypoint{{Pos: at, Warp: 9}}
		}
		if c.tank {
			tank := testDesign(tFrigate, 20)
			tank.Hull.Armor = 1000000
			g.Designs = append(g.Designs, tank)
			g.Fleets = append(g.Fleets, Fleet{ID: 4, Owner: 1, Pos: at, Stacks: []Stack{{Design: 2, Count: 1}}})
		}
		g.Fleets = append(g.Fleets, Fleet{ID: 3, Owner: 0, Pos: Point{1100, 1000}, Stacks: []Stack{{Design: 0, Count: 1}},
			Waypoints: []Waypoint{{Pos: c.start, Warp: 1, Target: TargetFleet, ID: 2}}})
		r, err := GenerateTurn(withRules(g), nil, rand.New(rand.NewSource(7)))
		if err != nil {
			t.Fatal(err)
		}
		var follower *Fleet
		tank := false
		for i := range r.Game.Fleets {
			switch r.Game.Fleets[i].ID {
			case 4:
				tank = true
			case 2:
				t.Fatalf("%s: the prey survived", c.name)
			case 3:
				follower = &r.Game.Fleets[i]
			}
		}
		if tank != c.tank {
			t.Fatalf("%s: survivor %v, want %v", c.name, tank, c.tank)
		}
		if follower == nil || len(follower.Waypoints) != 1 || follower.Waypoints[0] != c.want {
			t.Errorf("%s: follower %+v, want waypoint %+v", c.name, follower, c.want)
		}
		gone := 0
		for _, e := range r.Events {
			if e.Kind == EventTargetGone {
				gone++
				if e.Player != 0 || e.Fleet != 3 {
					t.Errorf("%s: 0x028 %+v, want player 0's, fleet 3", c.name, e)
				}
			}
		}
		if gone != 1 {
			t.Errorf("%s: %d 0x028, want 1", c.name, gone)
		}
	}
}

// TestWK1GoneOwed: removing a fleet owes 0x028 to each fleet whose
// waypoint it retargets to the planet there (WK-1; BINARY-ONLY for a
// removal other than battle) and to each the next waypoint check finds
// aimed at a gone fleet; none to one a stand-in fleet takes over
// (ASSUMPTION W10). The messages go out as the files are written, to the
// follower's owner, and only for a follower that still exists.
func TestWK1GoneOwed(t *testing.T) {
	g := Game{Rules: ElegyRules(), Players: []Player{{}, {}},
		Planets: []Planet{{ID: 7, Pos: Point{1150, 1150}}},
		Fleets: []Fleet{
			{ID: 1, Owner: 1, Pos: Point{1150, 1150}}, // removed at planet 7
			{ID: 2, Owner: 1, Pos: Point{1190, 1100}}, // removed in deep space
			{ID: 3, Owner: 1, Pos: Point{1120, 1100}}, // removed; fleet 4 stands in
			{ID: 4, Owner: 1, Pos: Point{1120, 1100}}, // the stand-in
			{ID: 5, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1140, 1140}, Target: TargetFleet, ID: 1}}},
			{ID: 6, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1180, 1100}, Target: TargetFleet, ID: 2}}},
			{ID: 7, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1110, 1100}, Target: TargetFleet, ID: 3}}},
			{ID: 8, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1140, 1140}, Target: TargetFleet, ID: 1}}}, // removed before the files
		}}
	g.removeFleets(map[int]bool{1: true, 2: true, 3: true})
	g.waypointCheck()
	g.removeFleets(map[int]bool{8: true})
	// A waypoint the file check itself finds aimed at a gone fleet, at
	// the planet: the planet, and 0x028 then.
	g.Fleets = append(g.Fleets, Fleet{ID: 9, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1150, 1150}, Target: TargetFleet, ID: 98}}})
	want := map[int]Waypoint{
		5: {Pos: Point{1150, 1150}, Target: TargetPlanet, ID: 7},
		6: {Pos: Point{1180, 1100}},
		7: {Pos: Point{1120, 1100}, Target: TargetFleet, ID: 4},
	}
	want9 := Waypoint{Pos: Point{1150, 1150}, Target: TargetPlanet, ID: 7}
	for _, f := range g.Fleets {
		if w, ok := want[f.ID]; ok && (len(f.Waypoints) != 1 || f.Waypoints[0] != w) {
			t.Errorf("fleet %d: waypoints %v, want %v", f.ID, f.Waypoints, w)
		}
	}
	// Player 0 sees the stand-in, so its waypoint keeps it.
	ev := g.fileRetarget([]PlayerView{{Player: 0, Fleets: []FleetSighting{{Fleet: 4, Owner: 1}}}, {Player: 1}})
	var got []int
	for _, e := range ev {
		if e.Kind != EventTargetGone || e.Player != 0 {
			t.Errorf("event %+v, want player 0's 0x028", e)
		}
		got = append(got, e.Fleet)
	}
	if len(got) != 3 || got[0] != 5 || got[1] != 6 || got[2] != 9 {
		t.Errorf("0x028 to fleets %v, want [5 6 9]", got)
	}
	if w := g.Fleets[len(g.Fleets)-1].Waypoints; len(w) != 1 || w[0] != want9 {
		t.Errorf("fleet 9: waypoints %v, want %v", w, want9)
	}
	if g.goneTargets != nil {
		t.Errorf("owed list %v after the files, want nil", g.goneTargets)
	}
}

// TestMeasuredBattleRemovalNoStandIn: a fleet destroyed in battle has no
// stand-in, so a waypoint chasing it goes to the planet where it died
// even when another fleet of its owner stays there (MEASURED, RG-003
// 2469 and 2470, retargetRemoved); with any other removal the stand-in
// takes over (ASSUMPTION W10). Only the planet case owes 0x028.
func TestMeasuredBattleRemovalNoStandIn(t *testing.T) {
	for _, battle := range []bool{true, false} {
		g := Game{Rules: ElegyRules(), Players: []Player{{}, {}},
			Planets: []Planet{{ID: 83, Pos: Point{1150, 1150}}},
			Fleets: []Fleet{
				{ID: 1, Owner: 1, Pos: Point{1150, 1150}}, // stays
				{ID: 2, Owner: 1, Pos: Point{1150, 1150}}, // removed
				{ID: 3, Owner: 1, Waypoints: []Waypoint{{Pos: Point{1140, 1150}, Target: TargetFleet, ID: 2}}},
			}}
		g.removeFleetsAs(map[int]bool{2: true}, !battle)
		want := Waypoint{Pos: Point{1150, 1150}, Target: TargetPlanet, ID: 83}
		owed := 1
		if !battle {
			want, owed = Waypoint{Pos: Point{1150, 1150}, Target: TargetFleet, ID: 1}, 0
		}
		if got := g.Fleets[1].Waypoints; len(got) != 1 || got[0] != want {
			t.Errorf("battle %v: waypoints %v, want %v", battle, got, want)
		}
		if len(g.goneTargets) != owed {
			t.Errorf("battle %v: owed %v, want %d", battle, g.goneTargets, owed)
		}
	}
}

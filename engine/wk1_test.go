package engine

import (
	"math/rand"
	"slices"
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
// follower's owner, and only for a follower that still exists. Several
// followers of one gone fleet each get their own 0x028 (fleets 5 and 10,
// 6 and 11): CONFIRMED by prediction for a fleet destroyed in battle (WK-2,
// stars-elegy d5b23e3, 3 of 3 streams; TestConfirmedWK2SeveralFollowersAndWaypoints),
// BINARY-ONLY for the other removals these fleets stand for.
func TestWK1GoneOwed(t *testing.T) {
	g := Game{Rules: ElegyRules(), Players: []Player{{}, {}},
		Planets: []Planet{{ID: 7, Pos: Point{1150, 1150}}},
		Fleets: []Fleet{
			{ID: 1, Number: 1, Owner: 1, Pos: Point{1150, 1150}}, // removed at planet 7
			{ID: 2, Number: 2, Owner: 1, Pos: Point{1190, 1100}}, // removed in deep space
			{ID: 3, Number: 3, Owner: 1, Pos: Point{1120, 1100}}, // removed; fleet 4 stands in
			{ID: 4, Owner: 1, Pos: Point{1120, 1100}},            // the stand-in
			{ID: 5, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1140, 1140}, Target: TargetFleet, ID: 1}}},
			{ID: 6, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1180, 1100}, Target: TargetFleet, ID: 2}}},
			{ID: 7, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1110, 1100}, Target: TargetFleet, ID: 3}}},
			{ID: 8, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1140, 1140}, Target: TargetFleet, ID: 1}}}, // removed before the files
			// Second followers of fleets 1 and 2, kept: each owes its own 0x028.
			{ID: 10, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1140, 1140}, Target: TargetFleet, ID: 1}}},
			{ID: 11, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1180, 1100}, Target: TargetFleet, ID: 2}}},
		}}
	// During a generation (removedFleets set), each 0x028 also names the
	// gone target's owner and number. Fleet 98 stands for one removed
	// earlier in the year whose follower no check has reached yet.
	g.removedFleets = map[int][2]int{98: {1, 98}}
	g.removeFleets(map[int]bool{1: true, 2: true, 3: true})
	g.waypointCheck()
	g.removeFleets(map[int]bool{8: true})
	// A waypoint the file check itself finds aimed at a gone fleet, at
	// the planet: the planet, and 0x028 then.
	g.Fleets = append(g.Fleets, Fleet{ID: 9, Owner: 0, Waypoints: []Waypoint{{Pos: Point{1150, 1150}, Target: TargetFleet, ID: 98}}})
	want := map[int]Waypoint{
		5:  {Pos: Point{1150, 1150}, Target: TargetPlanet, ID: 7},
		6:  {Pos: Point{1180, 1100}},
		7:  {Pos: Point{1120, 1100}, Target: TargetFleet, ID: 4},
		10: {Pos: Point{1150, 1150}, Target: TargetPlanet, ID: 7},
		11: {Pos: Point{1180, 1100}},
	}
	want9 := Waypoint{Pos: Point{1150, 1150}, Target: TargetPlanet, ID: 7}
	for _, f := range g.Fleets {
		if w, ok := want[f.ID]; ok && (len(f.Waypoints) != 1 || f.Waypoints[0] != w) {
			t.Errorf("fleet %d: waypoints %v, want %v", f.ID, f.Waypoints, w)
		}
	}
	// Player 0 sees the stand-in, so its waypoint keeps it.
	ev := g.fileRetarget([]PlayerView{{Player: 0, Fleets: []FleetSighting{{Fleet: 4, Owner: 1}}}, {Player: 1}})
	var got [][3]int
	for _, e := range ev {
		if e.Kind != EventTargetGone || e.Player != 0 || len(e.Axes) != 2 {
			t.Errorf("event %+v, want player 0's 0x028 naming its target", e)
			continue
		}
		got = append(got, [3]int{e.Fleet, e.Axes[0], e.Axes[1]})
	}
	if want := [][3]int{{5, 1, 1}, {10, 1, 1}, {6, 1, 2}, {11, 1, 2}, {9, 1, 98}}; !slices.Equal(got, want) {
		t.Errorf("0x028 (follower, target owner, number) %v, want %v", got, want)
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

// TestConfirmedWK2SeveralFollowersAndWaypoints: stars-elegy PARITY.md
// "Several followers, and waypoints 2 and up, aimed at a fleet destroyed
// in battle (WK-2)", stars-elegy d5b23e3 (#207), CONFIRMED by
// prediction, 3 of 3 streams. Player 1's
// victims fly 20 ly south and die in deep space; player 0's followers are
// far away. Each waypoint aimed at a victim becomes deep space where it
// died, a waypoint between them is unchanged, and every such waypoint
// owes its own 0x028 (A: three followers of one victim; B1: waypoints 1
// and 3 on one victim; B2: waypoints 1 and 2 on two victims; B3:
// waypoint 1 deep space, waypoint 2 on a victim). The
// observed message sequence is MEASURED; it also follows victim id and
// battle order, so the ordering rule is UNRESOLVED. Elegy's walk, fleets
// then waypoints, gives the observed sequence, and the test pins it.
func TestConfirmedWK2SeveralFollowersAndWaypoints(t *testing.T) {
	died, start := Point{1100, 1100}, Point{1100, 1080}
	died2, start2 := Point{1300, 1100}, Point{1300, 1080}
	prey := testDesign(tFrigate, 20, Slot{tLaser, 1})
	g := combatLabGame(prey, 6, 3)
	g.Fleets[1].Pos = start
	g.Fleets[1].Waypoints = []Waypoint{{Pos: died, Warp: 9}}
	g.Fleets = append(g.Fleets,
		Fleet{ID: 5, Owner: 0, Pos: died2, Stacks: []Stack{{Design: 0, Count: 6}}},
		Fleet{ID: 6, Owner: 1, Pos: start2, Stacks: []Stack{{Design: 1, Count: 3}}, Waypoints: []Waypoint{{Pos: died2, Warp: 9}}},
	)
	far, between := Point{1200, 1300}, Point{1250, 1250}
	chase := func(at Point, id int) Waypoint { return Waypoint{Pos: at, Warp: 1, Target: TargetFleet, ID: id} }
	followers := []Fleet{
		{ID: 10, Waypoints: []Waypoint{chase(start, 2)}},                                           // A
		{ID: 11, Waypoints: []Waypoint{chase(start, 2)}},                                           // A
		{ID: 12, Waypoints: []Waypoint{chase(start, 2), {Pos: between, Warp: 1}, chase(start, 2)}}, // A and B1
		{ID: 13, Waypoints: []Waypoint{chase(start, 2), chase(start2, 6)}},                         // B2
		{ID: 14, Waypoints: []Waypoint{{Pos: between, Warp: 1}, chase(start2, 6)}},                 // B3
	}
	for _, f := range followers {
		f.Owner, f.Pos, f.Stacks = 0, far, []Stack{{Design: 0, Count: 1}}
		g.Fleets = append(g.Fleets, f)
	}
	victim := map[int][2]int{}
	for i, f := range g.Fleets {
		if f.ID == 2 || f.ID == 6 {
			g.Fleets[i].Number = f.ID
			victim[f.ID] = [2]int{f.Owner, f.ID}
		}
	}
	r, err := GenerateTurn(withRules(g), nil, rand.New(rand.NewSource(7)))
	if err != nil {
		t.Fatal(err)
	}
	gone := func(at Point) Waypoint { return Waypoint{Pos: at, Warp: 1} }
	want := map[int][]Waypoint{
		10: {gone(died)},
		11: {gone(died)},
		12: {gone(died), {Pos: between, Warp: 1}, gone(died)},
		13: {gone(died), gone(died2)},
		14: {{Pos: between, Warp: 1}, gone(died2)},
	}
	for _, f := range r.Game.Fleets {
		switch f.ID {
		case 2, 6:
			t.Fatalf("victim %d survived", f.ID)
		}
		if w, ok := want[f.ID]; ok && !slices.Equal(f.Waypoints, w) {
			t.Errorf("fleet %d: waypoints %+v, want %+v", f.ID, f.Waypoints, w)
		}
	}
	// Each 0x028 names the follower and the gone target (MESSAGES.md
	// 0x028; WK-2's pairs, as (15,515) then (15,516) for one follower of
	// two victims).
	type pair struct {
		follower int
		target   [2]int
	}
	var got []pair
	for _, e := range r.Events {
		if e.Kind == EventTargetGone {
			if e.Player != 0 || len(e.Axes) != 2 {
				t.Errorf("0x028 %+v, want player 0's naming its target", e)
				continue
			}
			got = append(got, pair{e.Fleet, [2]int{e.Axes[0], e.Axes[1]}})
		}
	}
	v2, v6 := victim[2], victim[6]
	if v2 == v6 {
		t.Fatalf("victims share owner and number %v", v2)
	}
	if want := []pair{{10, v2}, {11, v2}, {12, v2}, {12, v2}, {13, v2}, {13, v6}, {14, v6}}; !slices.Equal(got, want) {
		t.Errorf("0x028 (follower, target) %v, want %v", got, want)
	}
}

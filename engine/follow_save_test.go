package engine_test

import (
	"bytes"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// TestFollowDoesNotPersist plays a year in which player 0's second fleet
// follows its first, sent to another planet, then saves and loads the
// game. ORDERS.md "Waypoint 0 aimed at a fleet" (stars-elegy c3aab85): the
// follow lasts one year and no waypoint 0 aimed at a fleet survives the
// generation. Elegy keeps the link only in the year's replay, so the
// state after the year, and the same state reloaded, has no follower and
// no waypoint aimed at a fleet, and both play on identically.
func TestFollowDoesNotPersist(t *testing.T) {
	s := newgame.Settings{Rules: engine.ElegyRules(), Size: newgame.Tiny, Density: newgame.Normal, Positions: newgame.Moderate}
	for range 2 {
		s.Players = append(s.Players, newgame.PlayerSetup{Race: races.Default()})
	}
	g, err := game.New(engine.ElegyRules(), s, 7)
	if err != nil {
		t.Fatal(err)
	}
	var mine []int
	for _, f := range g.State.Fleets {
		if f.Owner == 0 {
			mine = append(mine, f.ID)
		}
	}
	if len(mine) < 2 {
		t.Fatalf("player 0 has fleets %v, want two", mine)
	}
	home := g.State.Fleets[0].Pos
	dest := -1
	for _, p := range g.State.Planets {
		if p.Pos != home && (dest < 0 || p.ID < dest) {
			dest = p.ID
		}
	}
	lead, follower := mine[0], mine[1]
	orders := []engine.Order{
		engine.WaypointOrder{Fleet: lead, Waypoints: []engine.Waypoint{{Warp: 5, Target: engine.TargetPlanet, ID: dest}}},
		engine.FollowOrder{Fleet: follower, Leader: lead},
	}
	drivers := []game.Driver{
		game.DriverFunc(func(r game.Report) ([]engine.Order, error) { return orders, nil }),
		game.Idle,
	}
	y, err := g.Advance(drivers)
	if err != nil {
		t.Fatal(err)
	}
	for _, res := range y.Result.Orders {
		if res.Err != nil {
			t.Fatalf("order %d of player %d: %v", res.Index, res.Player, res.Err)
		}
	}
	done := false
	for _, e := range y.Result.Events {
		done = done || e.Kind == engine.EventFollowDone && e.Fleet == follower
	}
	if !done {
		t.Fatal("the follower was not told it followed (0x137)")
	}
	noFleetTargets := func(name string, st engine.Game) {
		t.Helper()
		for _, f := range st.Fleets {
			for _, wp := range f.Waypoints {
				if wp.Target == engine.TargetFleet {
					t.Errorf("%s: fleet %d keeps a waypoint aimed at fleet %d", name, f.ID, wp.ID)
				}
			}
			if f.ID == follower && f.Waypoints != nil {
				t.Errorf("%s: the follower keeps waypoints %v", name, f.Waypoints)
			}
		}
	}
	noFleetTargets("after the year", g.State)

	var buf bytes.Buffer
	if err := g.Save(&buf); err != nil {
		t.Fatal(err)
	}
	h, err := game.Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	noFleetTargets("reloaded", h.State)
	idle := []game.Driver{game.Idle, game.Idle}
	for _, gg := range []*game.Game{g, h} {
		if _, err := gg.Advance(idle); err != nil {
			t.Fatal(err)
		}
	}
	a, err := g.Hash()
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("the reloaded game played on differently: %s, %s", a, b)
	}
}

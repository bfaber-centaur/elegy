package engine

import (
	"slices"
	"testing"
)

// stubTerraform is a Terraformer that records its calls, to test where
// GenerateTurn and Launch use it.
type stubTerraform struct {
	rate   int
	mined  []int // fleet ids, in call order
	adjust int
}

func (s *stubTerraform) MiningRate(*Game, *Fleet) int { return s.rate }

func (s *stubTerraform) RemoteMine(g *Game, fi int, _ Rand) (Minerals, bool) {
	s.mined = append(s.mined, g.Fleets[fi].ID)
	return Minerals{}, true
}

// The production methods: no reach, no capacity, nothing built.
func (*stubTerraform) Reach(Race, [NumFields]int) [3]int      { return [3]int{} }
func (*stubTerraform) Capacity(Planet, Race, [3]int) int      { return 0 }
func (*stubTerraform) Improve(*Planet, Race, [3]int) int      { return -1 }
func (*stubTerraform) UnitCost(Race) int                      { return 100 }
func (*stubTerraform) AutoUnits(int, int, bool, int, int) int { return 0 }

func (s *stubTerraform) Adjust(g *Game) []Event {
	s.adjust++
	return []Event{{Kind: EventPlanetImproved, Player: 0, Planet: g.Planets[0].ID, Fleet: -1}}
}

// PRODUCTION-LAUNCH.md "Default task" (CONFIRMED SL-11): an Alternate
// Reality player's new fleet that can mine gets remote mining; a JOAT one,
// or one that cannot mine, gets no task.
func TestConfirmedLaunchDefaultRemoteMining(t *testing.T) {
	for _, c := range []struct {
		prt  PRT
		rate int
		want TaskKind
	}{{PRTAlternateReality, 8, TaskRemoteMine}, {PRTJackOfAllTrades, 8, TaskNone}, {PRTAlternateReality, 0, TaskNone}} {
		g := launchGame(t)
		g.Players[0].Race.PRT = c.prt
		g.Terraform = &stubTerraform{rate: c.rate}
		_, id := g.Launch(0, 0, 1)
		if k := g.Fleets[g.fleetIndex(id)].Task.Kind; k != c.want {
			t.Errorf("PRT %v rate %d: task %v, want %v", c.prt, c.rate, k, c.want)
		}
	}
}

// KERNEL.md "Remote mining" and "Turn order" steps 6c.2 and 7.4: a
// remote-mining fleet that stayed put mines once, after movement; one
// that moved this year does not (CONFIRMED T-35). The Orbital Adjusters
// run once, and their messages are the turn's.
func TestPredictionTerraformInTurn(t *testing.T) {
	g := pgHomeworld()
	scout, _ := slScouts(t)
	g.Designs = []Design{scout}
	pos := g.Planets[0].Pos
	mine := Task{Kind: TaskRemoteMine}
	g.Fleets = []Fleet{
		{ID: 1, Number: 1, Owner: 0, Pos: pos, Fuel: 50, Stacks: []Stack{{Design: 0, Count: 1}}, Task: mine},
		{ID: 2, Number: 2, Owner: 0, Pos: pos, Fuel: 50, Stacks: []Stack{{Design: 0, Count: 1}}, Task: mine,
			Waypoints: []Waypoint{{Pos: Point{pos.X + 20, pos.Y}, Warp: 5, Task: mine}}},
	}
	stub := &stubTerraform{}
	g.Terraform = stub
	r, err := GenerateTurn(withRules(g), nil, highRand{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(stub.mined, []int{1}) || stub.adjust != 1 {
		t.Errorf("mined %v, adjust calls %d; want [1] and 1", stub.mined, stub.adjust)
	}
	if !slices.ContainsFunc(r.Events, func(e Event) bool { return e.Kind == EventPlanetImproved }) {
		t.Error("no Orbital Adjuster message")
	}
}

// SCANNING.md "Designs" (BINARY-ONLY): a Packet Physics player learns in
// full the starbase design that caught its packet.
func TestPredictionPacketDesignSeen(t *testing.T) {
	scout, _ := slScouts(t)
	g := Game{Rules: ElegyRules(), Players: make([]Player, 2), Designs: []Design{scout}}
	sights := []ObjectSight{{Designs: []int{0}}, {}}
	views := viewsWith(g, nil, nil, make([]map[int]bool, 2), sights)
	if want := []DesignSighting{{Design: 0, Full: true, Hull: "Scout", Mass: scout.Mass}}; !slices.Equal(views[0].Designs, want) || len(views[1].Designs) != 0 {
		t.Errorf("designs %+v / %+v, want %+v / none", views[0].Designs, views[1].Designs, want)
	}
}

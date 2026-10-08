package races

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// slRaces are the two SL vector races (vectors sl/sl-starbases.json):
// the default economy with Improved Starbases, JOAT and AR.
func slRaces() []Design {
	joat, ar := Default(), Default()
	joat.Race.LRT.ImprovedStarbases = true
	ar.Race.PRT, ar.Race.LRT.ImprovedStarbases = engine.PRTAlternateReality, true
	return []Design{joat, ar}
}

func TestMeasuredGameRaceCheckSL(t *testing.T) {
	// RACES.md "In a running game": both SL races were illegal, and after
	// year 1 the original had raised colonists per resource to 2400 and
	// 2500 (SL-starbases player expectations; KERNEL.md step 2a, MEASURED
	// SL-12). Each player gets 0x117 and the other 0x182 (RD-P12).
	designs := slRaces()
	g := &engine.Game{Players: []engine.Player{{Race: designs[0].Race, ResearchBudget: 15}, {Race: designs[1].Race, ResearchBudget: 15}}}
	r := &GameRaces{Designs: designs, Computer: []bool{false, false}}
	events := r.CheckRaces(g)
	if c0, c1 := g.Players[0].Race.ColonistsPerResource, g.Players[1].Race.ColonistsPerResource; c0 != 2400 || c1 != 2500 {
		t.Errorf("colonists per resource %d and %d, want 2400 and 2500", c0, c1)
	}
	if !r.Designs[0].Tampered || !r.Designs[1].Tampered {
		t.Error("races not marked tampered")
	}
	want := []engine.Event{
		{Kind: engine.EventRacePenalized, Player: 0, Planet: -1, Fleet: -1},
		{Kind: engine.EventRaceHacked, Player: 1, Planet: -1, Fleet: -1, Count: 0},
		{Kind: engine.EventRacePenalized, Player: 1, Planet: -1, Fleet: -1},
		{Kind: engine.EventRaceHacked, Player: 0, Planet: -1, Fleet: -1, Count: 1},
	}
	if len(events) != len(want) {
		t.Fatalf("events %+v", events)
	}
	for i := range want {
		if events[i].Kind != want[i].Kind || events[i].Player != want[i].Player || events[i].Count != want[i].Count {
			t.Errorf("event %d %+v, want %+v", i, events[i], want[i])
		}
	}
	// RD-P19: a tampered race with positive points and nothing to repair
	// is left alone the next year, with no message.
	before := g.Players[0].Race
	if ev := r.CheckRaces(g); len(ev) != 0 || g.Players[0].Race != before {
		t.Errorf("second year: events %+v, race changed %v", ev, g.Players[0].Race != before)
	}
}

func TestGameRaceCheckComputerAndDead(t *testing.T) {
	// RD-P20: a computer race is not punished and no one is told.
	// ASSUMPTION R1: a dead player is not checked.
	designs := slRaces()
	g := &engine.Game{Players: []engine.Player{{Race: designs[0].Race, ResearchBudget: 150}, {Race: designs[1].Race, Dead: true}}}
	r := &GameRaces{Designs: designs, Computer: []bool{true, false}}
	if ev := r.CheckRaces(g); len(ev) != 0 {
		t.Errorf("events %+v", ev)
	}
	if g.Players[0].Race.ColonistsPerResource != 1000 || g.Players[0].ResearchBudget != 15 || g.Players[1].Race.ColonistsPerResource != 1000 {
		t.Errorf("players %+v", g.Players)
	}
}

func TestCloneRacesSharesNothing(t *testing.T) {
	// A check run on the clone leaves the original's designs (and their
	// tampered flags) as they were.
	designs := slRaces()
	r := &GameRaces{Designs: designs, Computer: []bool{false, false}}
	before := append([]Design(nil), r.Designs...)
	c := r.CloneRaces().(*GameRaces)
	g := &engine.Game{Players: []engine.Player{{Race: designs[0].Race, ResearchBudget: 15}, {Race: designs[1].Race, ResearchBudget: 15}}}
	if ev := c.CheckRaces(g); len(ev) == 0 {
		t.Fatal("the SL races were not penalized")
	}
	if !reflect.DeepEqual(r.Designs, before) || !c.Designs[0].Tampered {
		t.Errorf("original %+v, clone %+v", r.Designs, c.Designs)
	}
	c.Computer[0] = true
	if r.Computer[0] {
		t.Error("Computer shared")
	}
}

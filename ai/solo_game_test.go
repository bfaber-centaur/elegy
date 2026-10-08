package ai

import (
	"encoding/json"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// soloGame plays a small game of an idle human and one expert computer
// player of definition-file type typ for years years. Each year the
// computer player plans first, on the game's random stream, from its own
// report (AI.md §1), and its orders go into GenerateTurn. It returns the
// final game and every rejected order and unsupported step.
func soloGame(t *testing.T, typ int, play func(*View, engine.Rand) Result, seed uint64, years int) (engine.Game, []string) {
	t.Helper()
	ca, err := newgame.ComputerPlayer(typ, newgame.Expert)
	if err != nil {
		t.Fatal(err)
	}
	s := newgame.Settings{Rules: engine.ElegyRules(), Size: newgame.Tiny, Density: newgame.Normal, Positions: newgame.Moderate,
		Players: []newgame.PlayerSetup{{Race: races.Default()}, ca}}
	rng := newgame.NewRand(seed)
	res, err := newgame.Generate(s, rng)
	if err != nil {
		t.Fatal(err)
	}
	g := res.Game
	const me = 1
	var universe []PlanetPos
	for _, p := range g.Planets {
		universe = append(universe, PlanetPos{ID: p.ID, Pos: p.Pos})
	}
	history := map[int]engine.PlanetReport{}
	created := map[SlotKey]NewDesign{}
	views := engine.Views(g, engine.PopulationEstimates(g, rng))
	var events []engine.Event
	var results []engine.OrderResult
	var problems []string
	for range years {
		r, err := game.NewReport(g, me, views, events, results)
		if err != nil {
			t.Fatal(err)
		}
		v := NewView(r, Expert, universe, history, created)
		out := play(v, rng)
		for _, d := range out.Designs {
			created[SlotKey{d.Starbase, d.Slot}] = d
		}
		problems = append(problems, out.Unsupported...)
		for _, rep := range r.View.Planets {
			history[rep.Planet] = rep
		}
		file := engine.PlayerOrders{Player: me, GameID: g.ID, Year: g.Year, Orders: out.Orders}
		tr, err := engine.GenerateTurn(g, []engine.PlayerOrders{file}, rng)
		if err != nil {
			t.Fatalf("year %d: %v", g.Year, err)
		}
		for _, o := range tr.Orders {
			if o.Err != nil {
				problems = append(problems, "rejected: "+o.Err.Error())
			}
		}
		g, views, events, results = tr.Game, tr.Views, tr.Events, tr.Orders
	}
	return g, problems
}

// A Rototill game runs 40 years with every order accepted, and replays
// identically from the same seed.
func TestRototillPlaysAlone(t *testing.T) {
	soloCheck(t, "Rototill", 4, PlayRototill, 40)
}

// A Cybertron game runs 60 years with every order accepted, and replays
// identically from the same seed. 60 years reach its Privateers,
// Destroyers and first warship group (cybertron.md §2).
func TestCybertronPlaysAlone(t *testing.T) {
	soloCheck(t, "Cybertron", 5, PlayCybertron, 60)
}

func soloCheck(t *testing.T, name string, typ int, play func(*View, engine.Rand) Result, years int) {
	g, problems := soloGame(t, typ, play, 11, years)
	for _, p := range problems {
		if len(p) >= 9 && p[:9] == "rejected:" {
			t.Errorf("%s", p)
		}
	}
	owned, ships := 0, 0
	for _, p := range g.Planets {
		if p.Owner == 1 {
			owned++
		}
	}
	for _, f := range g.Fleets {
		if f.Owner == 1 {
			for _, s := range f.Stacks {
				ships += s.Count
			}
		}
	}
	t.Logf("year %d: %s owns %d planets, %d ships; tech %v; %d unsupported steps", g.Year, name, owned, ships, g.Players[1].Research.Levels, len(problems))
	if owned == 0 {
		t.Errorf("%s lost every planet", name)
	}

	again, _ := soloGame(t, typ, play, 11, years)
	a, _ := json.Marshal(g)
	b, _ := json.Marshal(again)
	if string(a) != string(b) {
		t.Error("the same seed gave a different game")
	}
}

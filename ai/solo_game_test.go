package ai

import (
	"encoding/json"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// aiPlayer is a computer player of definition-file type typ and its
// planner.
type aiPlayer struct {
	typ  int
	play func(*View, engine.Rand) Result
}

// aiGame plays a small game of an idle human (player 0) and expert
// computer players 1.. for years years. Each year the computer players
// plan first, in player order, on the game's one random stream, each from
// its own report and planet history (AI.md §1), and their orders go into
// GenerateTurn. It returns the final game and every rejected order and
// unsupported step.
func aiGame(t *testing.T, seed uint64, years int, ais ...aiPlayer) (engine.Game, []string) {
	t.Helper()
	setups := []newgame.PlayerSetup{{Race: races.Default()}}
	for _, a := range ais {
		ps, err := newgame.ComputerPlayer(a.typ, newgame.Expert)
		if err != nil {
			t.Fatal(err)
		}
		setups = append(setups, ps)
	}
	s := newgame.Settings{Rules: engine.ElegyRules(), Size: newgame.Tiny, Density: newgame.Normal, Positions: newgame.Moderate, Players: setups}
	rng := newgame.NewRand(seed)
	res, err := newgame.Generate(s, rng)
	if err != nil {
		t.Fatal(err)
	}
	g := res.Game
	var universe []PlanetPos
	for _, p := range g.Planets {
		universe = append(universe, PlanetPos{ID: p.ID, Pos: p.Pos})
	}
	history := make([]map[int]engine.PlanetReport, len(ais))
	for i := range ais {
		history[i] = map[int]engine.PlanetReport{}
	}
	views := engine.Views(g, engine.PopulationEstimates(g, rng))
	var events []engine.Event
	var results []engine.OrderResult
	var problems []string
	for range years {
		var files []engine.PlayerOrders
		for i, a := range ais {
			me := i + 1
			r, err := game.NewReport(g, me, views, events, results)
			if err != nil {
				t.Fatal(err)
			}
			v := NewView(r, Expert, universe, history[i])
			out := a.play(v, rng)
			problems = append(problems, out.Unsupported...)
			for _, rep := range r.View.Planets {
				history[i][rep.Planet] = rep
			}
			files = append(files, engine.PlayerOrders{Player: me, GameID: g.ID, Year: g.Year, Orders: out.Orders})
		}
		tr, err := engine.GenerateTurn(g, files, rng)
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

// soloGame is aiGame with one computer player.
func soloGame(t *testing.T, typ int, play func(*View, engine.Rand) Result, seed uint64, years int) (engine.Game, []string) {
	t.Helper()
	return aiGame(t, seed, years, aiPlayer{typ, play})
}

// A Rototill game runs 40 years with every order accepted, and replays
// identically from the same seed.
func TestRototillPlaysAlone(t *testing.T) {
	soloCheck(t, "Rototill", 4, PlayRototill, 40)
}

// A Cybertron game runs 60 years with every order accepted, and replays
// identically from the same seed. 60 years reach its Privateers,
// Destroyers and first warship group (cybertron.md §2). Cybertron sees
// other planets only through its scanner-shot packets (§6), so owning
// more than its homeworld shows they run.
func TestCybertronPlaysAlone(t *testing.T) {
	if owned := soloCheck(t, "Cybertron", 5, PlayCybertron, 60); owned < 2 {
		t.Errorf("Cybertron owns %d planets, want more than its homeworld", owned)
	}
}

// A Robotoid game runs 60 years with every order accepted, and replays
// identically from the same seed.
func TestRobotoidPlaysAlone(t *testing.T) {
	soloCheck(t, "Robotoid", 1, PlayRobotoid, 60)
}

// soloCheck plays the solo game, checks it and returns the planets the
// computer player owns at the end.
func soloCheck(t *testing.T, name string, typ int, play func(*View, engine.Rand) Result, years int) int {
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
	return owned
}

// Robotoid, Rototill and Cybertron play one game together for 60 years:
// every order is accepted, each keeps a planet, and the same seed replays
// to an identical game. A smoke test, not a parity check.
func TestAllThreeTogether(t *testing.T) {
	ais := []aiPlayer{{1, PlayRobotoid}, {4, PlayRototill}, {5, PlayCybertron}}
	g, problems := aiGame(t, 11, 60, ais...)
	for _, p := range problems {
		if len(p) >= 9 && p[:9] == "rejected:" {
			t.Errorf("%s", p)
		}
	}
	for i, name := range []string{"Robotoid", "Rototill", "Cybertron"} {
		owned := 0
		for _, p := range g.Planets {
			if p.Owner == i+1 {
				owned++
			}
		}
		t.Logf("year %d: %s owns %d planets", g.Year, name, owned)
		if owned == 0 {
			t.Errorf("%s lost every planet", name)
		}
	}
	again, _ := aiGame(t, 11, 60, ais...)
	a, _ := json.Marshal(g)
	b, _ := json.Marshal(again)
	if string(a) != string(b) {
		t.Error("the same seed gave a different game")
	}
}

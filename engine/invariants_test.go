package engine_test

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// checkYearFixture is the first two years of a new seven-player game (a
// default human and the six computer-player races) with no orders; each
// year passes CheckYear.
func checkYearFixture(t *testing.T) []engine.TurnResult {
	t.Helper()
	players := []newgame.PlayerSetup{{Race: races.Default()}}
	for typ := 1; typ <= 6; typ++ {
		p, err := newgame.ComputerPlayer(typ, newgame.Expert)
		if err != nil {
			t.Fatal(err)
		}
		players = append(players, p)
	}
	s := newgame.Settings{Rules: engine.ElegyRules(), Size: newgame.Small, Density: newgame.Normal, Positions: newgame.Moderate, Players: players}
	rng := newgame.NewRand(5)
	res, err := newgame.Generate(s, rng)
	if err != nil {
		t.Fatal(err)
	}
	g := res.Game
	var out []engine.TurnResult
	for range 2 {
		r, err := engine.GenerateTurn(g, nil, rng)
		if err != nil {
			t.Fatalf("year %d: %v", g.Year, err)
		}
		if err := engine.CheckYear(g, r); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
		g = r.Game
	}
	return out
}

// CheckYear reports a broken invariant: each corruption of a good year
// is named.
func TestCheckYearReports(t *testing.T) {
	rs := checkYearFixture(t)
	prev, good := rs[0].Game, rs[1]
	corrupt := func(name string, f func(r *engine.TurnResult)) {
		r := good
		r.Game.Planets = append([]engine.Planet(nil), good.Game.Planets...)
		r.Game.Fleets = append([]engine.Fleet(nil), good.Game.Fleets...)
		r.Game.Players = append([]engine.Player(nil), good.Game.Players...)
		r.Views = append([]engine.PlayerView(nil), good.Views...)
		f(&r)
		if engine.CheckYear(prev, r) == nil {
			t.Errorf("%s: not reported", name)
		}
	}
	corrupt("unowned population", func(r *engine.TurnResult) {
		for i := range r.Game.Planets {
			if r.Game.Planets[i].Owner == engine.NoOwner {
				r.Game.Planets[i].Population = 5
				return
			}
		}
	})
	corrupt("fleet without ships", func(r *engine.TurnResult) { r.Game.Fleets[0].Stacks = nil })
	corrupt("duplicate fleet id", func(r *engine.TurnResult) { r.Game.Fleets[1].ID = r.Game.Fleets[0].ID })
	corrupt("research fell", func(r *engine.TurnResult) { r.Game.Players[0].Research.Levels[0] = -1 })
	corrupt("own planet missing from view", func(r *engine.TurnResult) { r.Views[0].Planets = nil })
	corrupt("year", func(r *engine.TurnResult) { r.Game.Year++ })
}

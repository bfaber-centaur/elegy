package engine_test

import (
	"errors"
	"strings"
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
// is reported with its own problem.
func TestCheckYearReports(t *testing.T) {
	rs := checkYearFixture(t)
	good := rs[1]
	corrupt := func(name, want string, f func(prev *engine.Game, r *engine.TurnResult)) {
		prev := rs[0].Game
		prev.Players = append([]engine.Player(nil), prev.Players...)
		r := good
		r.Game.Planets = append([]engine.Planet(nil), good.Game.Planets...)
		r.Game.Fleets = append([]engine.Fleet(nil), good.Game.Fleets...)
		r.Game.Players = append([]engine.Player(nil), good.Game.Players...)
		r.Views = append([]engine.PlayerView(nil), good.Views...)
		f(&prev, &r)
		err := engine.CheckYear(prev, r)
		var ie *engine.InvariantError
		if !errors.As(err, &ie) {
			t.Errorf("%s: got %v, want an *InvariantError", name, err)
			return
		}
		if len(ie.Problems) != 1 || !strings.Contains(ie.Problems[0], want) {
			t.Errorf("%s: problems %q, want only one containing %q", name, ie.Problems, want)
		}
	}
	unowned := func(r *engine.TurnResult) int {
		for i := range r.Game.Planets {
			if r.Game.Planets[i].Owner == engine.NoOwner {
				return i
			}
		}
		t.Fatal("no unowned planet")
		return -1
	}
	corrupt("unowned population", "unowned planet", func(_ *engine.Game, r *engine.TurnResult) {
		r.Game.Planets[unowned(r)].Population = 5
	})
	corrupt("negative population", "population -1", func(_ *engine.Game, r *engine.TurnResult) {
		for i := range r.Game.Planets {
			if r.Game.Planets[i].Owner != engine.NoOwner {
				r.Game.Planets[i].Population = -1
				return
			}
		}
	})
	corrupt("environment", "environment", func(_ *engine.Game, r *engine.TurnResult) {
		r.Game.Planets[unowned(r)].Env[0] = 100
	})
	corrupt("fleet without ships", "has no ships", func(_ *engine.Game, r *engine.TurnResult) {
		r.Game.Fleets[0].Stacks, r.Game.Fleets[0].Fuel = nil, 0
	})
	corrupt("duplicate fleet id", "twice", func(_ *engine.Game, r *engine.TurnResult) {
		r.Game.Fleets[1].ID = r.Game.Fleets[0].ID
	})
	corrupt("duplicate fleet number", "duplicate or below 1", func(_ *engine.Game, r *engine.TurnResult) {
		for i := 1; i < len(r.Game.Fleets); i++ {
			if r.Game.Fleets[i].Owner == r.Game.Fleets[0].Owner {
				r.Game.Fleets[i].Number = r.Game.Fleets[0].Number
				return
			}
		}
		t.Fatal("no second fleet of one owner")
	})
	// The level stays in range: the year before had it one higher.
	corrupt("research fell", "fell from", func(prev *engine.Game, r *engine.TurnResult) {
		lv := r.Game.Players[0].Research.Levels[0]
		if lv >= engine.MaxTechLevel {
			t.Fatal("field 0 at the top level")
		}
		prev.Players[0].Research.Levels[0] = lv + 1
	})
	corrupt("research out of range", "level 27", func(_ *engine.Game, r *engine.TurnResult) {
		r.Game.Players[0].Research.Levels[0] = engine.MaxTechLevel + 1
	})
	corrupt("own planet missing from view", "lacks their planet", func(_ *engine.Game, r *engine.TurnResult) {
		r.Views[0].Planets = nil
	})
	corrupt("year", "after", func(_ *engine.Game, r *engine.TurnResult) { r.Game.Year++ })
}

// An owned planet may end the year with no colonists (TAKEOVER.md "Unload
// and load amounts": a load after movement can take them all, and the
// planet is lost only at the next year's growth); CheckYear allows it.
func TestCheckYearAllowsOwnedEmptyPlanet(t *testing.T) {
	rs := checkYearFixture(t)
	r := rs[1]
	r.Game.Planets = append([]engine.Planet(nil), r.Game.Planets...)
	for i := range r.Game.Planets {
		if r.Game.Planets[i].Owner != engine.NoOwner {
			r.Game.Planets[i].Population = 0
			break
		}
	}
	if err := engine.CheckYear(rs[0].Game, r); err != nil {
		t.Fatal(err)
	}
}

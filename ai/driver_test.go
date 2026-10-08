package ai

import (
	"bytes"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// loopSetup is a new tiny game with an idle human and expert Robotoid,
// Rototill and Cybertron, and fresh drivers for them.
func loopSetup(t *testing.T, rules engine.Ruleset, seed uint64) (*game.Game, []game.Driver) {
	t.Helper()
	s := newgame.Settings{Size: newgame.Tiny, Density: newgame.Normal, Positions: newgame.Moderate,
		Players: []newgame.PlayerSetup{{Race: races.Default()}}}
	for _, typ := range []int{1, 4, 5} {
		ps, err := newgame.ComputerPlayer(typ, newgame.Expert)
		if err != nil {
			t.Fatal(err)
		}
		s.Players = append(s.Players, ps)
	}
	g, err := game.New(rules, s, seed)
	if err != nil {
		t.Fatal(err)
	}
	return g, freshDrivers()
}

func freshDrivers() []game.Driver {
	return []game.Driver{game.Idle, NewDriver(Robotoid, Expert), NewDriver(Rototill, Expert), NewDriver(Cybertron, Expert)}
}

// advance plays years years and returns each year's state hash and every
// rejected order.
func advance(t *testing.T, g *game.Game, drivers []game.Driver, years int) (hashes, rejected []string) {
	t.Helper()
	for range years {
		y, err := g.Advance(drivers)
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range y.Result.Orders {
			if o.Err != nil {
				rejected = append(rejected, o.Err.Error())
			}
		}
		h, err := g.Hash()
		if err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, h)
	}
	return hashes, rejected
}

// loopGame plays the game of loopSetup through the game loop
// (game.Game.Advance) for years years. It returns the game, each year's
// state hash and every rejected order.
func loopGame(t *testing.T, rules engine.Ruleset, seed uint64, years int) (*game.Game, []string, []string) {
	t.Helper()
	g, drivers := loopSetup(t, rules, seed)
	hashes, rejected := advance(t, g, drivers, years)
	return g, hashes, rejected
}

// The three personalities play 50 years through the game loop, under
// Elegy's rules and under jrc3-faithful's: every order is accepted, each
// keeps a planet, and the same seed gives the same state every year. A
// smoke test, not a parity check.
func TestDriversInGameLoop(t *testing.T) {
	for _, rules := range []engine.Ruleset{engine.ElegyRules(), engine.FaithfulRules()} {
		t.Run(rules.ID, func(t *testing.T) {
			g, hashes, rejected := loopGame(t, rules, 7, 50)
			for _, r := range rejected {
				t.Errorf("rejected: %s", r)
			}
			for i, name := range []string{"Robotoid", "Rototill", "Cybertron"} {
				owned := 0
				for _, p := range g.State.Planets {
					if p.Owner == i+1 {
						owned++
					}
				}
				t.Logf("year %d: %s owns %d planets", g.State.Year, name, owned)
				if owned == 0 {
					t.Errorf("%s lost every planet", name)
				}
			}
			_, again, _ := loopGame(t, rules, 7, 50)
			for i := range hashes {
				if hashes[i] != again[i] {
					t.Fatalf("year %d: the same seed gave a different state", 2401+i)
				}
			}
		})
	}
}

// A report without the game's stream is refused rather than planned with
// another one.
func TestDriverNeedsRand(t *testing.T) {
	if _, err := NewDriver(Rototill, Expert).Orders(game.Report{}); err != ErrNoRand {
		t.Errorf("error %v, want ErrNoRand", err)
	}
}

// A game saved in the middle and reloaded into fresh drivers continues
// exactly as the uninterrupted game: every year's state hash matches,
// for 50 years. So does a game saved and reloaded into fresh drivers
// every year, as cmd/elegy's turn command runs it. Under Elegy's rules and
// jrc3-faithful's.
func TestDriversSaveReload(t *testing.T) {
	const years, at = 50, 25
	reload := func(t *testing.T, g *game.Game) *game.Game {
		t.Helper()
		var buf bytes.Buffer
		if err := g.Save(&buf); err != nil {
			t.Fatal(err)
		}
		loaded, err := game.Load(&buf)
		if err != nil {
			t.Fatal(err)
		}
		return loaded
	}
	for _, rules := range []engine.Ruleset{engine.ElegyRules(), engine.FaithfulRules()} {
		t.Run(rules.ID, func(t *testing.T) {
			_, want, _ := loopGame(t, rules, 7, years)
			check := func(how string, got []string) {
				t.Helper()
				for i := range want {
					if got[i] != want[i] {
						t.Fatalf("%s: year %d differs from the uninterrupted game", how, 2401+i)
					}
				}
			}

			g, drivers := loopSetup(t, rules, 7)
			first, _ := advance(t, g, drivers, at)
			rest, _ := advance(t, reload(t, g), freshDrivers(), years-at)
			check("reloaded after 2425", append(first, rest...))

			g, _ = loopSetup(t, rules, 7)
			var yearly []string
			for range years {
				h, _ := advance(t, g, freshDrivers(), 1)
				yearly = append(yearly, h...)
				g = reload(t, g)
			}
			check("reloaded every year", yearly)
		})
	}
}

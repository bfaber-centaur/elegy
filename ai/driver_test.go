package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// loopGame plays an idle human and expert Robotoid, Rototill and
// Cybertron through the game loop (game.Game.Advance) for years years. It
// returns the game, each year's state hash and every rejected order.
func loopGame(t *testing.T, rules engine.Ruleset, seed uint64, years int) (*game.Game, []string, []string) {
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
	drivers := []game.Driver{game.Idle, NewDriver(Robotoid, Expert), NewDriver(Rototill, Expert), NewDriver(Cybertron, Expert)}
	var hashes, rejected []string
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

package engine_test

import (
	"testing"

	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/regress"
)

// Two long-game cases from the nightly run where a computer player's
// chase order named another player's fleet that its owner had merged away
// earlier in the same replay, and was rejected (seed 10 in 2457 with the
// elegy rules, seed 2 in 2498 with jrc3-faithful). They now play past
// that year with no failure, replaying and reloading the same (ORDERS.md
// "Targets that moved, died or were captured", "A fleet target merged
// away during order replay").
func TestMergedChaseLongGames(t *testing.T) {
	if testing.Short() {
		t.Skip("long games")
	}
	opponents := []string{"robotoid", "rototill", "cybertron"}
	for _, c := range []regress.Case{
		{Seed: 10, Rules: "elegy", Opponents: opponents, Size: newgame.Medium, Years: 58, SaveAt: 40},
		{Seed: 2, Rules: "jrc3-faithful", Opponents: opponents, Size: newgame.Medium, Years: 99, SaveAt: 77},
	} {
		r := regress.Run(c)
		if r.Failed() || r.Years != c.Years {
			t.Error(r.Report())
		}
	}
}

package ai

import (
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// ShufflePlanets is a computer player's own-planet order for the turn
// (AI.md §2, BINARY-ONLY): its planets in planet-id order, then for i = 0
// .. n−2 swap i with i + Random(n − i). It is made once per turn and used
// by every per-planet pass that names "the shuffled order". ids are the
// player's own planet ids in any order; the result is a new slice.
//
// The original makes no shuffle in a tutorial game; Elegy has no
// tutorial game.
func ShufflePlanets(ids []int, rng engine.Rand) []int {
	order := slices.Clone(ids)
	slices.Sort(order)
	for i := 0; i < len(order)-1; i++ {
		j := i + rng.Intn(len(order)-i)
		order[i], order[j] = order[j], order[i]
	}
	return order
}

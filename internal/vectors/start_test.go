package vectors

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

func TestStartFleetsIn(t *testing.T) {
	// A fleet of the start keeps its vector id while it exists; a fleet
	// made since takes its number less one, the first such fleet in game
	// order winning, as fleetAt resolves orders.
	s := Start{Fleets: map[[2]int]int{{1, 4}: 7}}
	g := &engine.Game{Fleets: []engine.Fleet{
		{ID: 20, Owner: 1, Number: 3},
		{ID: 21, Owner: 1, Number: 3},
		{ID: 22, Owner: 1, Number: 5}, // number 5 is id 4: the start's fleet 7 wins
		{ID: 7, Owner: 1, Number: 9},
		{ID: 23, Owner: 2, Number: 3},
	}}
	got := s.FleetsIn(g)
	want := map[[2]int]int{{1, 2}: 20, {1, 4}: 7, {2, 2}: 23}
	if len(got) != len(want) {
		t.Fatalf("FleetsIn %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("FleetsIn %v: %d, want %d", k, got[k], v)
		}
	}
}

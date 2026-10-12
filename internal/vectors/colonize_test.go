package vectors

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestRG001ColonizeCostLevels checks the RG-001 Y5 restart (2433 → 2434)
// on every seed variant: player 1's Mini-Colony Ships colonize planets 97
// and 114 after movement. Player 1 starts the year at energy 3 and ends it
// at 5; the original's files give each planet 10/6/9, ⌊3/4⌋ of the ship's
// cost at energy 3 (MEASURED), where energy 5 gives 9/6/8.
func TestRG001ColonizeCostLevels(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg001-y5.json")
	if err != nil {
		t.Fatal(err)
	}
	st, err := v.Start()
	if err != nil {
		t.Fatal(err)
	}
	for variant := range Seeds {
		games, stop := Generate(v, variant, engine.GenerateTurn)
		if stop != "" {
			t.Fatalf("variant %d: %s", variant, stop)
		}
		g := games[len(games)-1]
		for _, vid := range []int{97, 114} {
			for _, p := range g.Planets {
				if p.ID == st.Planets[vid] && (p.Owner != 1 || p.Surface != (engine.Minerals{10, 6, 9})) {
					t.Errorf("variant %d planet %d: owner %d surface %v, want player 1 with 10/6/9", variant, vid, p.Owner, p.Surface)
				}
			}
		}
	}
}

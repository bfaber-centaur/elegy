package vectors

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestRG001EmptiedPlanetCarry checks the chained RG-001 run at planet 50,
// on every seed variant: player 6 bombs it empty in 2432 after the year's
// growth (carry 48 in Elegy), the year's file drops its record (no mines,
// factories, defenses or scanner; stars-elegy 9a355fe TAKEOVER.md
// "Capture: what a planet keeps", BINARY-ONLY), and player 6's colony of
// 2433 grows from carry 0: population 21 with carry 56 in 2434, carry 14
// in 2435, as the original's files hold (MEASURED, RG-001). Kept, the
// carry gives 22 with 4, then 60.
func TestRG001EmptiedPlanetCarry(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg001.json")
	if err != nil {
		t.Fatal(err)
	}
	st, err := v.Start()
	if err != nil {
		t.Fatal(err)
	}
	id := st.Planets[50]
	for variant := range Seeds {
		games, stop := Generate(v, variant, engine.GenerateTurn)
		if stop != "" {
			t.Fatalf("variant %d: %s", variant, stop)
		}
		got := map[int][2]int{}
		for _, g := range games {
			for _, p := range g.Planets {
				if p.ID == id {
					got[g.Year] = [2]int{p.Population, p.GrowthCarry}
				}
			}
		}
		if got[2433][1] != 0 || got[2434] != [2]int{21, 56} || got[2435][1] != 14 {
			t.Errorf("variant %d: (population, carry) by year %v; want carry 0 in 2433, 21 with 56 in 2434, carry 14 in 2435", variant, got)
		}
	}
}

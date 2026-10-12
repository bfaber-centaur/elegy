package terraform_test

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/internal/vectors"
	"github.com/bfaber-centaur/elegy/terraform"
)

// draws is an engine.Rand returning fixed values, then n − 1 (a miss).
type draws []int

func (d *draws) Intn(n int) int {
	if len(*d) == 0 {
		return n - 1
	}
	v := (*d)[0]
	*d = (*d)[1:]
	return v
}

// TestPacketTerraformRG001 checks the two Packet Physics packet impacts
// of RG-001 conditioned on the draws, not as parity: the original's
// generator cannot be replayed, so each case gives PacketTerraform the
// successes the original's result implies and compares the result with
// the original's (MEASURED, RG-001 per-year restarts; the rule is
// OBJECTS.md "PP terraforming"). Player 5 (PP) launched both packets.
//
//   - 2432, planet 92: one temperature and one radiation success, none
//     permanent; environment 15/16/99 to 15/17/98, original unchanged.
//   - 2435, planet 95: one temperature success, not permanent;
//     71/28/92 to 71/29/92, original unchanged.
//
// In Elegy's own runs of these restarts two successes move an axis by
// two clicks in both states, so neither result is a larger success count
// capped by the limit, and a radiation success on planet 95 moves it, so
// the original had none there. u is the cargo left after Elegy's decay; the minerals it
// adds to each planet equal the original's.
func TestPacketTerraformRG001(t *testing.T) {
	for _, c := range []struct {
		file         string
		planet       int
		u            engine.Minerals
		draws        draws
		env, origEnv [3]int
	}{
		// Boranium 227 is three chunks: miss, miss, hit (not permanent);
		// germanium 171 is two: hit (not permanent), miss.
		{"rg001-y3.json", 92, engine.Minerals{0, 227, 171}, draws{199, 199, 0, 9, 0, 9}, [3]int{15, 17, 98}, [3]int{15, 16, 99}},
		// Boranium 225: hit (not permanent), then misses; germanium misses.
		{"rg001-y6.json", 95, engine.Minerals{0, 225, 113}, draws{0, 9}, [3]int{71, 29, 92}, [3]int{71, 28, 92}},
	} {
		v, err := vectors.ReadFile("../engine/testdata/vectors/rg/" + c.file)
		if err != nil {
			t.Fatal(err)
		}
		st, err := v.Start()
		if err != nil {
			t.Fatal(err)
		}
		pi := -1
		for i, p := range st.Game.Planets {
			if p.ID == st.Planets[c.planet] {
				pi = i
			}
		}
		if pi < 0 {
			t.Fatalf("%s: no planet %d", c.file, c.planet)
		}
		d := c.draws
		terraform.PacketTerraform(&st.Game, pi, 5, c.u, &d)
		if p := st.Game.Planets[pi]; p.Env != c.env || p.OrigEnv != c.origEnv {
			t.Errorf("%s planet %d: environment %v, original %v; want %v, %v", c.file, c.planet, p.Env, p.OrigEnv, c.env, c.origEnv)
		}
	}
}

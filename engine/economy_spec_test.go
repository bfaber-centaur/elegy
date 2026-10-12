package engine

import "testing"

// KERNEL.md "Mining" step 4 (stars-elegy b1e38ef): below concentration 5
// depletion clamps cc to 10 (CONFIRMED, KB-1A: concentration 4 with 500
// mines ended with fraction 251, where the clamp 25 gives 245). A
// non-homeworld planet, fraction 0, eff 10: p = trunc(4·500/100) = 20,
// need = 1250, f' = trunc(1230·256/1250) = 251.
func TestConfirmedDepletionClampBelow5(t *testing.T) {
	gain, d := MineYear(Deposit{Concentration: 4}, 500, 10, false, highRand{})
	if gain != 20 || d != (Deposit{Concentration: 4, Fraction: 251}) {
		t.Errorf("gain %d, deposit %+v; want 20 and conc 4 frac 251", gain, d)
	}
}

// KERNEL.md "Population growth" step 3 (stars-elegy b1e38ef, BINARY-ONLY):
// for very large products the game uses trunc(g/100)·P whenever that is
// at least 10,000,000. G 19, hab 99: g = 1881, uncrowded.
func TestBinaryOnlyGrowthLargeProduct(t *testing.T) {
	for _, c := range []struct{ pop, want int }{
		// 18·600,000 = 10,800,000 ≥ 10,000,000: t = 10,800,000, q = 108,000
		// (trunc(1881·600,000/100) would give 112,860).
		{600_000, 708_000},
		// 18·400,000 = 7,200,000 < 10,000,000: t = trunc(1881·400,000/100)
		// = 7,524,000, q = 75,240.
		{400_000, 475_240},
	} {
		if p, k := GrowPopulation(c.pop, 50, 10_000_000, 19, 99); p != c.want || k != 50 {
			t.Errorf("P %d: %d carry %d, want %d carry 50", c.pop, p, k, c.want)
		}
	}
}

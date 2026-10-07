package engine

// Rand is the game's random number source. Intn returns a uniform draw in
// 0..n−1. The original generator is not specified; tests inject one.
type Rand interface {
	Intn(n int) int
}

// Deposit is one mineral's concentration state on a planet.
type Deposit struct {
	// Concentration is the stored concentration byte.
	Concentration int
	// Fraction is the 1/256ths of the current concentration point still
	// remaining; 0 means a full 256.
	Fraction int
}

// MineYear runs one year of planetary mining for one mineral and returns
// the kT added to the surface and the deposit's new state.
//
// mines is the number of working mines (or, for remote mining, the robot
// rate); eff is the race's mine output (10 for Alternate Reality, and for
// remote mining pass 10 so amt = prod). homeworld applies the
// concentration-30 output floor.
//
// KERNEL.md "Mining": output and depletion CONFIRMED (48 PG years,
// homeworld, 10 mines, eff 10); the random +1 mechanism and the homeworld
// floor are BINARY-ONLY.
func MineYear(d Deposit, mines, eff int, homeworld bool, rng Rand) (gain int, out Deposit) {
	used := d.Concentration
	if homeworld && used < 30 {
		used = 30
	}
	prod := used * mines
	amt := prod * eff / 10
	gain = amt / 100
	if rem := amt % 100; rem != 0 && rng.Intn(100) < rem {
		gain++
	}
	return gain, deplete(d, prod/100)
}

// depletionConcentration is the stored concentration clamped for the
// depletion computation. KERNEL.md does not say whether the clamp is
// re-evaluated as the concentration drops inside one year; it is here.
func depletionConcentration(conc int) int {
	switch {
	case conc > 100:
		return 100
	case conc < 5:
		return 10
	case conc < 25:
		return 25
	}
	return conc
}

func deplete(d Deposit, p int) Deposit {
	for p > 0 && d.Concentration > 1 {
		cc := depletionConcentration(d.Concentration)
		s := d.Fraction
		if s == 0 {
			s = 256
		}
		need := s * 12500 / 256 / cc
		if need <= p {
			p -= need
			d.Concentration--
			d.Fraction = 0
			continue
		}
		f := (need - p) * 256 / (12500 / cc)
		if f < 1 {
			f = 1
		}
		if f >= s {
			f = s - 1
		}
		d.Fraction = f
		if f == 0 {
			d.Concentration--
		}
		break
	}
	return d
}

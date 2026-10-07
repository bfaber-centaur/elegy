package engine

// Population is held in units of 100 colonists, as in KERNEL.md.
const ColonistsPerUnit = 100

// alternateRealityMaxPop is the AR maximum population by starbase hull, in
// hull order (BINARY-ONLY).
var alternateRealityMaxPop = [...]int{2500, 5000, 10000, 20000, 30000}

// MaxPopulation returns a planet's maximum population in units.
//
// starbaseHull is 0 when the planet has no starbase of the owner's, else
// 1..5 in hull order; it matters only for Alternate Reality.
//
// KERNEL.md "Maximum population": CONFIRMED for an ordinary race at hab 100;
// racial modifiers and hab < 5 are BINARY-ONLY.
func MaxPopulation(race Race, hab int, starbaseHull int) int {
	var mp int
	if race.PRT == PRTAlternateReality {
		if starbaseHull >= 1 && starbaseHull <= len(alternateRealityMaxPop) {
			mp = alternateRealityMaxPop[starbaseHull-1]
		}
	} else {
		mp = 500
		if hab >= 5 {
			mp = 100 * hab
		}
		switch race.PRT {
		case PRTHyperExpansion:
			mp -= mp / 2
		case PRTJackOfAllTrades:
			mp += mp / 5
		}
	}
	if race.LRT.OnlyBasicRemoteMining {
		mp += mp / 10
	}
	return mp
}

// GrowPopulation advances one year of population growth (or death) on a
// planet: pop in units, carry in hundredths of a unit (the persistent
// excessPop byte), maxPop in units, growthRate G in percent (already doubled
// for Hyper-Expansion), hab in percent.
//
// KERNEL.md "Population growth". CONFIRMED for G 10, hab 100, max 10,000
// (g = 1000, so the quantized g ≥ 1000 crowding branch), uncrowded and
// crowded up to 54% of capacity. The g < 1000 crowding branch, the
// within-10-of-max freeze, overcrowding and hostile deaths are BINARY-ONLY.
func GrowPopulation(pop, carry, maxPop, growthRate, hab int) (newPop, newCarry int) {
	if pop <= 0 {
		// Not specified by KERNEL.md; an empty planet has nothing to grow.
		return pop, carry
	}
	if hab < 0 {
		return hostileDeaths(pop, carry, hab)
	}

	g := growthRate * hab
	if pop >= maxPop/4 {
		c := crowdingPermille(pop, maxPop)
		switch {
		case pop < maxPop:
			f := (1000 - c) * (1000 - c)
			if g < 1000 {
				g = g * f / 562500
			} else {
				g = 10 * (g / 10 * f / 562500)
			}
		case pop <= maxPop+10:
			return pop, carry
		default:
			g = 2 * max(-300, c/-10+99)
		}
	}

	t := g * pop / 100
	if alt := g / 100 * pop; alt >= 10_000_000 {
		t = alt
	}
	q, r := t/100, t%100
	if q == 0 && r == 0 {
		r = 1
	}
	carry += r
	if carry >= 100 {
		q++
		carry -= 100
	} else if carry < 0 {
		q--
		carry += 100
	}
	return pop + q, carry
}

// crowdingPermille is trunc(1000·P/max). A zero maximum (Alternate Reality
// without a starbase) is not covered by KERNEL.md; it is treated as
// maximally overcrowded.
func crowdingPermille(pop, maxPop int) int {
	if maxPop <= 0 {
		return 1 << 30
	}
	return 1000 * pop / maxPop
}

// hostileDeaths is the hab < 0 branch (BINARY-ONLY).
func hostileDeaths(pop, carry, hab int) (int, int) {
	t := max(1, -hab*pop/10)
	q, r := t/100, t%100
	carry -= r
	if carry < 0 {
		carry += 100
		q++
	}
	return max(0, pop-q), carry
}

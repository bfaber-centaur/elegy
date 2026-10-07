package races

import "github.com/bfaber-centaur/elegy/engine"

// RandomTemplate is the wizard's Random race: HE, growth 15, 17–83 on
// every axis, marked random (RACES.md "Random race").
func RandomTemplate(name string) Design {
	d := Default()
	d.Race.PRT = engine.PRTHyperExpansion
	for i := range d.Race.Env {
		d.Race.Env[i] = engine.EnvRange{Low: 17, High: 83, Center: 50}
	}
	d.Name = name
	d.Random = true
	return d
}

// randomTries is the number of adjust steps before the fallback.
const randomTries = 251

func span(lo, hi int) engine.EnvRange {
	return engine.EnvRange{Low: lo, High: hi, Center: lo + (hi-lo)/2}
}

// Generate replaces a Random race template with a generated race scoring
// 0..50 (RACES.md "Random race", CONFIRMED in outcome: RD-4..RD-6's 14
// generated races scored 3..44; draw details BINARY-ONLY). The name is
// kept; the caller replaces the name "Random" with a computer-player name.
//
// Elegy draws from its own generator; only the distributions and the
// 0..50 target are the original's.
func Generate(name string, rng engine.Rand) Design {
	d := Default()
	d.Name = name
	r := &d.Race

	// 1. Habitat and growth.
	switch k := rng.Intn(25); {
	case k < 4:
		for a := range r.Env {
			r.Env[a] = engine.EnvRange{Immune: true}
		}
		r.GrowthRate = 2 + rng.Intn(4)
	case k < 7:
		for a := range r.Env {
			r.Env[a] = span(0, 100)
		}
		r.GrowthRate = 3 + rng.Intn(4)
	case k < 9:
		for a := range engine.Radiation {
			if rng.Intn(2) == 0 {
				r.Env[a] = span(0, 100)
			} else {
				r.Env[a] = span(17, 83)
			}
		}
		r.Env[engine.Radiation] = span(17, 83)
		r.GrowthRate = 2 + rng.Intn(5)
	default:
		for a := range r.Env {
			w := 2 * (10 + rng.Intn(40))
			lo := rng.Intn(101 - w)
			r.Env[a] = span(lo, lo+w)
		}
		switch {
		case k <= 11:
			r.Env[rng.Intn(3)] = engine.EnvRange{Immune: true}
		case k <= 13:
			r.Env[rng.Intn(3)] = span(0, 100)
		case k <= 16:
			a := rng.Intn(3)
			lo := rng.Intn(81)
			r.Env[a] = span(lo, lo+20)
		}
		r.GrowthRate = 7 + rng.Intn(9)
	}

	// 2. Research.
	if rng.Intn(3) != 0 {
		for f := range r.ResearchCosts {
			r.ResearchCosts[f] = researchCost(rng.Intn(3))
		}
	}

	// 3. Traits.
	r.PRT = AllPRTs[rng.Intn(len(AllPRTs))]
	if rng.Intn(4) != 0 {
		for i := range NumLRTs {
			d.SetLRT(i, rng.Intn(2) == 0)
		}
	}
	d.ExpensiveAt3 = rng.Intn(2) == 0
	r.FactoryLessGermanium = rng.Intn(2) == 0

	// 4. Economy. ASSUMPTION: "uniform over its whole range" for
	// colonists per resource is uniform over 700..2500 in steps of 100.
	if rng.Intn(3) == 0 {
		d.Spend = rng.Intn(5)
	} else {
		r.ColonistsPerResource = (7 + rng.Intn(19)) * 100
		r.FactoryOutput = MinFactoryOutput + rng.Intn(MaxFactoryOutput-MinFactoryOutput+1)
		r.FactoryCost = MinFactoryCost + rng.Intn(MaxFactoryCost-MinFactoryCost+1)
		r.FactoriesOperated = MinOperated + rng.Intn(MaxOperated-MinOperated+1)
		r.MineOutput = MinMineOutput + rng.Intn(MaxMineOutput-MinMineOutput+1)
		r.MineCost = MinMineCost + rng.Intn(MaxMineCost-MinMineCost+1)
		r.MinesOperated = MinOperated + rng.Intn(MaxOperated-MinOperated+1)
		d.Spend = rng.Intn(MaxSpend + 1)
	}

	// 6. Adjust to 0..50 points: one change per step, kept only when it
	// brings the points strictly closer to 0..50.
	dist := func(p int) int {
		switch {
		case p < 0:
			return -p
		case p > 50:
			return p - 50
		}
		return 0
	}
	cur := dist(Points(d))
	for range randomTries {
		if cur == 0 {
			return d
		}
		next := d
		adjust(&next, rng)
		if n := dist(Points(next)); n < cur {
			d, cur = next, n
		}
	}
	if cur == 0 {
		return d
	}
	def := Default()
	def.Name = name
	return def
}

// economy steps for the adjust loop: colonists per resource and the six
// factory and mine settings, with their ranges.
//
// ASSUMPTION: "one random setting among colonists, factories and mines"
// is uniform over these seven settings.
var economy = []struct {
	field    func(*engine.Race) *int
	step     int
	min, max int
}{
	{func(r *engine.Race) *int { return &r.ColonistsPerResource }, 100, MinColonists, MaxColonists},
	{func(r *engine.Race) *int { return &r.FactoryOutput }, 1, MinFactoryOutput, MaxFactoryOutput},
	{func(r *engine.Race) *int { return &r.FactoryCost }, 1, MinFactoryCost, MaxFactoryCost},
	{func(r *engine.Race) *int { return &r.FactoriesOperated }, 1, MinOperated, MaxOperated},
	{func(r *engine.Race) *int { return &r.MineOutput }, 1, MinMineOutput, MaxMineOutput},
	{func(r *engine.Race) *int { return &r.MineCost }, 1, MinMineCost, MaxMineCost},
	{func(r *engine.Race) *int { return &r.MinesOperated }, 1, MinOperated, MaxOperated},
}

// adjust makes one random change (RACES.md "Random race" step 6): 3 in
// 10 a research field, 3 in 10 an LRT, 3 in 10 an economy setting, 1 in
// 20 an axis, 1 in 20 growth.
//
// ASSUMPTION: where RACES.md says "X, else Y" for research, the economy
// and growth, Elegy picks the direction with a fair coin and makes no
// change when that direction is at its limit; for an LRT, "set to off,
// else on" is a toggle.
func adjust(d *Design, rng engine.Rand) {
	r := &d.Race
	switch k := rng.Intn(20); {
	case k < 6:
		f := rng.Intn(engine.NumFields)
		lv := researchLevel(r.ResearchCosts[f])
		if rng.Intn(2) == 0 {
			lv = max(0, lv-1) // dearer
		} else {
			lv = min(2, lv+1) // cheaper
		}
		r.ResearchCosts[f] = researchCost(lv)
	case k < 12:
		i := rng.Intn(NumLRTs)
		d.SetLRT(i, !d.HasLRT(i))
	case k < 18:
		e := economy[rng.Intn(len(economy))]
		v := e.field(r)
		if rng.Intn(2) == 0 {
			*v = max(e.min, *v-e.step)
		} else {
			*v = min(e.max, *v+e.step)
		}
	case k < 19:
		a := rng.Intn(3)
		if r.Env[a].Immune {
			lo := rng.Intn(31)
			r.Env[a] = span(lo, lo+70)
		} else {
			r.Env[a] = engine.EnvRange{Immune: true}
		}
	default:
		if rng.Intn(2) == 0 {
			if r.GrowthRate > 1 {
				r.GrowthRate--
			}
		} else if r.GrowthRate < 15 {
			r.GrowthRate++
		}
	}
}

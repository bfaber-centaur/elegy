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

	// 4. Economy: with probability 1/3 the default economy; otherwise
	// each setting uniform over its range (colonists per resource
	// 700..2500 in steps of 100).
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
	// Distance is max(points − 50, −points), 0 inside 0..50.
	dist := func(x Design) int {
		p := Points(x)
		return max(0, p-50, -p)
	}
	for range randomTries {
		if dist(d) == 0 {
			return d
		}
		adjust(&d, rng, dist)
	}
	if dist(d) == 0 {
		return d
	}
	def := Default()
	def.Name = name
	return def
}

// economy is the seven economy settings the adjust loop picks from,
// uniformly (RACES.md "Random race" step 6), with their steps and ranges.
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

// adjust takes one adjust step (RACES.md "Random race" step 6): one draw
// for the kind of change (3 in 10 a research field, 3 in 10 an LRT, 3 in
// 10 an economy setting, 1 in 20 an axis, 1 in 20 growth) and one for the
// field, trait, setting or axis; its options are tried in order and the
// first that brings the points strictly closer to 0..50 is kept. A try
// blocked at a limit changes nothing.
func adjust(d *Design, rng engine.Rand, dist func(Design) int) {
	cur := dist(*d)
	try := func(change func(*Design)) bool {
		next := *d
		change(&next)
		if dist(next) < cur {
			*d = next
			return true
		}
		return false
	}
	switch k := rng.Intn(20); {
	case k < 6:
		f := rng.Intn(engine.NumFields)
		lv := researchLevel(d.Race.ResearchCosts[f])
		set := func(l int) func(*Design) {
			return func(x *Design) { x.Race.ResearchCosts[f] = researchCost(l) }
		}
		if lv > 0 && try(set(lv-1)) { // dearer
			return
		}
		if lv < 2 {
			try(set(lv + 1)) // cheaper
		}
	case k < 12:
		i := rng.Intn(NumLRTs)
		if !try(func(x *Design) { x.SetLRT(i, false) }) {
			try(func(x *Design) { x.SetLRT(i, true) })
		}
	case k < 18:
		e := economy[rng.Intn(len(economy))]
		if !try(func(x *Design) { v := e.field(&x.Race); *v = max(e.min, *v-e.step) }) {
			try(func(x *Design) { v := e.field(&x.Race); *v = min(e.max, *v+e.step) })
		}
	case k < 19:
		a := rng.Intn(3)
		if d.Race.Env[a].Immune {
			lo := rng.Intn(31)
			try(func(x *Design) { x.Race.Env[a] = span(lo, lo+70) })
		} else {
			try(func(x *Design) { x.Race.Env[a] = engine.EnvRange{Immune: true} })
		}
	default:
		g := d.Race.GrowthRate
		if g > 1 && try(func(x *Design) { x.Race.GrowthRate = g - 1 }) {
			return
		}
		if g < 15 {
			try(func(x *Design) { x.Race.GrowthRate = g + 1 })
		}
	}
}

package newgame

import (
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
)

// MaxPlanets is the most planets a galaxy holds (UNIVERSE.md "Count").
const MaxPlanets = 999

// minSpacing2 is the minimum spacing: no two placed planets have
// d² ≤ 144 (12 ly). UNIVERSE.md "Count" step 3, CONFIRMED.
const minSpacing2 = 144

// planetCount is the nominal planet count N (UNIVERSE.md "Count" step 1,
// CONFIRMED).
func planetCount(size Size, density Density) int {
	w := (int(size) + 1) * 400
	n := w * w / 5000
	n += n / 4 * (int(density) - 1)
	if density == Packed {
		n += n / 4
	}
	return min(n, MaxPlanets)
}

// placePlanets returns the planet positions in planet-list order
// (UNIVERSE.md "Count" steps 2–4, CONFIRMED).
func (g *generator) placePlanets() []engine.Point {
	n := planetCount(g.s.Size, g.s.Density)
	m := min(MaxPlanets, n+n/7)

	// Candidates are uniform with x, y in 1010 .. 1010+W−20.
	// ELEGY CHOICE: the range is read as inclusive at both ends.
	span := g.w - 20 + 1
	cand := make([]engine.Point, m)
	for i := range cand {
		cand[i] = engine.Point{X: Origin + 10 + g.rand(span), Y: Origin + 10 + g.rand(span)}
	}

	// Minimum spacing (CONFIRMED): the candidates are sorted by x;
	// taking each survivor in that order, every later candidate within
	// d² ≤ 144 goes. A removed candidate removes nothing, so this is
	// "keep a candidate unless it is within 12 ly of one already kept".
	// The planet list stays in this x order.
	// ELEGY CHOICE: the original's sort is not stable for equal x;
	// Elegy's is.
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].X < cand[j].X })
	kept := make([]engine.Point, 0, m)
	for _, c := range cand {
		ok := true
		for _, k := range kept {
			if d2(c, k) <= minSpacing2 {
				ok = false
				break
			}
		}
		if ok {
			kept = append(kept, c)
		}
	}

	// Remove random candidates until N remain; fewer remain when the
	// spacing pass already removed more than M − N, and none is added
	// back (tiny and small packed games can end below N, CONFIRMED UG22,
	// UG23, UG24).
	for len(kept) > n {
		i := g.rand(len(kept))
		kept = append(kept[:i], kept[i+1:]...)
	}
	return kept
}

// clump applies the galaxy clumping option (UNIVERSE.md "Clumping
// option", CONFIRMED UG04, UG10): n times, a random planet moves toward
// its nearest other planet when that one is more than 12 ly away.
func (g *generator) clump(pos []engine.Point, n int) {
	if len(pos) < 2 {
		return
	}
	for range n {
		a := g.rand(len(pos))
		// ELEGY CHOICE: ties for nearest go to the earliest planet.
		b, best := -1, 0
		for j, p := range pos {
			if j == a {
				continue
			}
			if d := d2(pos[a], p); b < 0 || d < best {
				b, best = j, d
			}
		}
		if best <= minSpacing2 {
			continue
		}
		pa, pb := pos[a], pos[b]
		var mv func(a, b int) int
		switch {
		case best >= 1601:
			mv = func(a, b int) int { return (a + 2*b) / 3 }
		case best >= 626:
			mv = func(a, b int) int { return (a + b) / 2 }
		case best >= 325:
			mv = func(a, b int) int { return (2*a + b) / 3 }
		default: // 145–324
			mv = func(a, b int) int { return (4*a + b) / 5 }
		}
		pos[a] = engine.Point{X: mv(pa.X, pb.X), Y: mv(pa.Y, pb.Y)}
	}
}

// makePlanets fills in every planet: name, environment, artifact and
// mineral concentrations. Surface minerals stay 0 (UNIVERSE.md "Mineral
// concentrations": only homeworlds and second planets get any).
func (g *generator) makePlanets(pos []engine.Point) {
	n := len(pos)
	planets := make([]engine.Planet, n)
	g.res.NameIndex = make([]int, n)
	g.res.Artifact = make([]bool, n)
	used := make([]bool, MaxPlanets)
	for i := range planets {
		p := &planets[i]
		p.ID = i
		p.Pos = pos[i]
		p.Owner = engine.NoOwner

		// Names (CONFIRMED): a uniform index, stepped to the next
		// unused one. ELEGY CHOICE: the step wraps from 998 to 0.
		k := g.rand(MaxPlanets)
		for used[k] {
			k = (k + 1) % MaxPlanets
		}
		used[k] = true
		g.res.NameIndex[i] = k

		p.Env = g.randomEnv()
		if !g.s.NoRandomEvents {
			g.res.Artifact[i] = g.rand(3) == 0
		}
		conc := g.concentrations(p.Env[engine.Radiation])
		for m := range engine.NumMinerals {
			p.Deposits[m] = engine.Deposit{Concentration: conc[m]}
		}
	}
	g.res.Game.Planets = planets
}

// randomEnv is a new planet's environment (UNIVERSE.md "Environment",
// CONFIRMED): gravity and temperature 1 + rand(90) + rand(10), radiation
// 1 + rand(99).
func (g *generator) randomEnv() [3]int {
	var e [3]int
	e[engine.Gravity] = 1 + g.rand(90) + g.rand(10)
	e[engine.Temperature] = 1 + g.rand(90) + g.rand(10)
	e[engine.Radiation] = 1 + g.rand(99)
	return e
}

// concentrations are a new planet's mineral concentrations (UNIVERSE.md
// "Mineral concentrations", CONFIRMED).
func (g *generator) concentrations(radiation int) [engine.NumMinerals]int {
	var c [engine.NumMinerals]int
	if g.s.MaxMinerals {
		for m := range c {
			c[m] = 100
		}
		return c
	}
	for m := range c {
		v := 31 + g.rand(45) + g.rand(45)
		if radiation > 89 && v < 99 {
			v += g.rand(99-v) / 2
		}
		if g.s.BBS && v < 40 {
			v += 5
		}
		c[m] = v
	}
	// Low overrides: r = rand(27); 4, 3, 2 or 1 overrides for r = 0,
	// 1–2, 3–6, 7–8; one for 9–17; none otherwise. Each hits a random
	// mineral, so one mineral can be hit twice.
	times := 0
	switch r := g.rand(27); {
	case r == 0:
		times = 4
	case r <= 2:
		times = 3
	case r <= 6:
		times = 2
	case r <= 17:
		times = 1
	}
	for range times {
		c[g.rand(engine.NumMinerals)] = 1 + g.rand(30)
	}
	return c
}

package races

import "github.com/bfaber-centaur/elegy/engine"

// clampSettings clamps every setting except the habitat to its range
// (RACES.md "Repairs": "every other setting is clamped"; "In a running
// game" step 1, the silent clamps). An out-of-range PRT becomes JOAT
// (CONFIRMED RD-P7), stat 15 becomes 0 (RD-P5), colonists per resource
// are clamped to 700–2500 (RD-P6), growth below 0 becomes 1 and above 20
// becomes 20 (BINARY-ONLY). Growth 0 is left for the scoring repair.
//
// ASSUMPTION: colonists per resource is stored in units of 100 (7–25), so
// a value that is not a multiple of 100 is truncated to one; a research
// cost outside the three settings becomes normal.
func clampSettings(d *Design) bool {
	before := *d
	r := &d.Race
	if !validPRT(r.PRT) {
		r.PRT = engine.PRTJackOfAllTrades
	}
	switch {
	case r.GrowthRate < 0:
		r.GrowthRate = 1
	case r.GrowthRate > MaxGrowth:
		r.GrowthRate = MaxGrowth
	}
	r.ColonistsPerResource = clamp(r.ColonistsPerResource/100*100, MinColonists, MaxColonists)
	r.FactoryOutput = clamp(r.FactoryOutput, MinFactoryOutput, MaxFactoryOutput)
	r.FactoryCost = clamp(r.FactoryCost, MinFactoryCost, MaxFactoryCost)
	r.FactoriesOperated = clamp(r.FactoriesOperated, MinOperated, MaxOperated)
	r.MineOutput = clamp(r.MineOutput, MinMineOutput, MaxMineOutput)
	r.MineCost = clamp(r.MineCost, MinMineCost, MaxMineCost)
	r.MinesOperated = clamp(r.MinesOperated, MinOperated, MaxOperated)
	for f := range r.ResearchCosts {
		if c := r.ResearchCosts[f]; c != engine.ResearchExpensive && c != engine.ResearchNormal && c != engine.ResearchCheap {
			r.ResearchCosts[f] = engine.ResearchNormal
		}
	}
	d.Spend = clamp(d.Spend, 0, MaxSpend)
	d.Stat15 = 0
	return !equal(before, *d)
}

// scoringRepair is the repair the scoring itself makes (RACES.md
// "Repairs"; "In a running game" step 2: "by now only the habitat, or
// growth 0"): per non-immune axis `lo` clamped to 0..100, `hi` to
// lo..100 and the centre forced to lo + (hi − lo)/2 (CONFIRMED RD-4,
// RD-P4, RD-P8); growth below 1 becomes 1 (CONFIRMED RD-4).
//
// ASSUMPTION: RACES.md also says an axis with a value outside 0..100 is
// made immune (BINARY-ONLY), but RD-P8 (gravity low −5) clamped the low
// instead. Elegy follows RD-P8 and never makes an axis immune here.
func scoringRepair(d *Design) bool {
	before := *d
	for a := range d.Race.Env {
		e := &d.Race.Env[a]
		if e.Immune {
			continue
		}
		e.Low = clamp(e.Low, 0, 100)
		e.High = clamp(e.High, e.Low, 100)
		e.Center = e.Low + (e.High-e.Low)/2
	}
	if d.Race.GrowthRate < MinGrowth {
		d.Race.GrowthRate = MinGrowth
	}
	return !equal(before, *d)
}

func equal(a, b Design) bool { return a == b }

// Repaired returns the race after every creation-time repair and whether
// anything changed (RACES.md "Repairs").
func Repaired(d Design) (Design, bool) {
	c := clampSettings(&d)
	s := scoringRepair(&d)
	return d, c || s
}

// Points is the race's advantage points (RACES.md "Advantage points",
// CONFIRMED RD-1..RD-6), scored on the repaired race. A race is legal
// when its points are at least 0.
func Points(d Design) int {
	d, _ = Repaired(d)
	return rawPoints(d) / 3
}

// rawPoints is P for an already repaired race.
func rawPoints(d Design) int {
	r := d.Race
	g := r.GrowthRate
	col := r.ColonistsPerResource / 100
	immune := 0
	for _, e := range r.Env {
		if e.Immune {
			immune++
		}
	}

	// 3. Growth.
	hab := habIntegral(r) / 2000
	var base, mult int
	switch {
	case g <= 5:
		base, mult = (6-g)*4200+1650, g
	case g <= 13:
		base = map[int]int{6: 5250, 7: 3900, 8: 2250, 9: 1875}[g]
		if g >= 10 {
			base = 1650
		}
		mult = 2*g - 5
	case g <= 19:
		base, mult = 1650, (g-6)*3
	default:
		base, mult = 1650, 45
	}
	p := base - mult*hab/24

	// 4. Habitat shape.
	for _, e := range r.Env {
		if !e.Immune {
			p += 4 * abs(e.Center-50)
		}
	}
	if immune >= 2 {
		p -= 150
	}
	fn, fo := r.FactoriesOperated, r.FactoryOutput
	if fn > 10 || fo > 10 {
		a := max(1, fn-9)
		k := 2
		if r.PRT == engine.PRTHyperExpansion {
			k = 3
		}
		b := max(1, fo-9) * k
		if immune < 2 {
			p += a * b * g / -9
		} else {
			p -= a * b * g / 2
		}
	}

	// 5. Colonists per resource.
	switch {
	case col < 8:
		p -= 2400
	case col == 8:
		p -= 1260
	case col == 9:
		p -= 600
	case col > 10:
		p += 120 * (col - 10)
	}

	// 6. Economy.
	if r.PRT == engine.PRTAlternateReality {
		p += 210
	} else {
		fc := r.FactoryCost
		var e int
		if fo >= 10 {
			e = (fo - 10) * -121
		} else {
			e = (fo - 10) * -100
		}
		if fc > 10 {
			e += 55 * (fc - 10)
		} else {
			e -= 60 * (10 - fc) * (10 - fc)
		}
		n := 10 - fn
		if n <= 0 {
			e += n * 35
		} else {
			e += n * 40
		}
		if e > 700 {
			e = (e-700)/3 + 700
		}
		if n < -6 {
			switch {
			case n < -14:
				e -= 360
			case n < -11:
				e += (n + 7) * 45
			default:
				e += (n + 6) * 30
			}
		}
		if fo > 12 {
			e += (12 - fo) * 60
		}
		p += e
		if r.FactoryLessGermanium {
			p -= 175
		}
		mo, mc := r.MineOutput, r.MineCost
		if mo >= 10 {
			p += (mo - 10) * -169
		} else {
			p += (mo - 10) * -100
		}
		if mc >= 3 {
			p += (mc-3)*65 + 80
		} else {
			p -= 360
		}
		m := 10 - r.MinesOperated
		if m <= 0 {
			p += m * 35
		} else {
			p += m * 40
		}
	}

	// 7. Traits.
	p -= prtCost[r.PRT]
	n, good, bad := 0, 0, 0
	for i, t := range lrtList {
		if !d.HasLRT(i) {
			continue
		}
		n++
		p += t.value
		if t.value < 0 {
			good++
		} else {
			bad++
		}
	}
	if n > 4 {
		p -= 10 * n * (n - 4)
	}
	if dd := bad - good; dd > 3 {
		p -= 60 * (dd - 3)
	}
	if dd := good - bad; dd > 3 {
		p -= 40 * (dd - 3)
	}
	if r.LRT.NoAdvancedScanners {
		switch r.PRT {
		case engine.PRTPacketPhysics:
			p -= 280
		case engine.PRTSuperStealth:
			p -= 200
		case engine.PRTJackOfAllTrades:
			p -= 40
		}
	}

	// 8. Research.
	s := 0
	for _, c := range r.ResearchCosts {
		s += researchLevel(c) - 1
	}
	switch {
	case s < 0:
		p += [6]int{150, 330, 540, 780, 1050, 1380}[-s-1]
		if s < -4 && col < 10 {
			p -= 190
		}
	case s > 0:
		p -= 130 * s * s
		switch s {
		case 5:
			p += 520
		case 6:
			p += 1430
		}
	}
	if d.ExpensiveAt3 {
		p -= 180
	}
	if r.PRT == engine.PRTAlternateReality && r.ResearchCosts[engine.Energy] == engine.ResearchCheap {
		p -= 100
	}
	return p
}

// habIntegral is H (RACES.md "Habitability integral", CONFIRMED).
func habIntegral(r engine.Race) int {
	type axis struct {
		immune bool
		vals   []int // test values after the move toward the centre
		dprime []int // δ' per test value
		width  int
	}
	weights := [3]int{7, 5, 6}
	allow := [3]int{0, 5, 15}
	if r.LRT.TotalTerraforming {
		allow = [3]int{0, 8, 17}
	}
	total := 0.0
	for k := range 3 {
		t := allow[k]
		var axes [3]axis
		for a, e := range r.Env {
			if e.Immune {
				axes[a] = axis{immune: true, vals: []int{50}, dprime: []int{0}}
				continue
			}
			low, top := max(0, e.Low-t), min(100, e.High+t)
			ax := axis{width: top - low}
			for i := range 11 {
				x := low + ax.width*i/10
				dp := 0
				if k > 0 {
					delta := e.Center - x
					switch {
					case abs(delta) <= t:
						dp = 0
					case delta > 0:
						dp = delta - t
					default:
						dp = delta + t
					}
					x = e.Center - dp
				}
				ax.vals = append(ax.vals, x)
				ax.dprime = append(ax.dprime, dp)
			}
			axes[a] = ax
		}
		gSum := 0.0
		for gi, gv := range axes[0].vals {
			tSum := 0.0
			for ti, tv := range axes[1].vals {
				rSum := 0
				for ri, rv := range axes[2].vals {
					h := engine.Habitability(r, [3]int{gv, tv, rv})
					if k > 0 {
						if sum := axes[0].dprime[gi] + axes[1].dprime[ti] + axes[2].dprime[ri]; sum > t {
							h = max(0, h-(sum-t))
						}
					}
					rSum += weights[k] * h * h
				}
				if axes[2].immune {
					rSum *= 11
				} else {
					rSum = rSum * axes[2].width / 100
				}
				tSum += float64(rSum)
			}
			if axes[1].immune {
				tSum *= 11.0
			} else {
				tSum *= float64(axes[1].width) * 0.01
			}
			gSum += tSum
		}
		if axes[0].immune {
			gSum *= 11.0
		} else {
			gSum *= float64(axes[0].width) * 0.01
		}
		total += gSum
	}
	return int(total*0.1 + 0.5)
}

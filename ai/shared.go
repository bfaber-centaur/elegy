package ai

import "github.com/bfaber-centaur/elegy/engine"

// researchAndStarbases is the start of every personality's turn (AI.md
// §1): research (§4) and starbase designs (§5).
func researchAndStarbases(pers Personality, v *View, rng engine.Rand, res *Result) {
	y := v.Year - FirstYear
	if o, ok := Research(pers, y, v.Self.Research, v.Self.ResearchBudget); ok {
		res.Orders = append(res.Orders, o)
	}
	sb := StarbaseInput{Personality: pers, Year: v.Year, Race: v.Self.Race, Levels: v.Self.Research.Levels, TraderItems: v.TraderItems}
	for _, d := range v.Starbases {
		if d.Slot >= 0 && d.Slot < len(sb.Designs) {
			sb.Designs[d.Slot] = &SlotDesign{Hull: d.Design.Hull.Name, Name: d.Design.Name, Created: d.Created, Picture: d.Picture}
		}
	}
	for _, p := range v.Planets {
		if p.HasStarbase {
			for _, d := range v.Starbases {
				if d.Index == p.StarbaseDesign && d.Slot < len(sb.Built) {
					sb.Built[d.Slot] = true
				}
			}
		}
	}
	orders, made := StarbaseDesigns(sb, rng)
	res.Orders = append(res.Orders, orders...)
	res.Designs = append(res.Designs, made...)
}

// preferWormhole is AI.md §11's wormhole preference after a colonizable
// planet search: wormholes with w ≤ 4·D score (7 − class)·10 when known,
// else 90 when w ≤ D or 50 otherwise; the best (ties: smaller w) is taken
// when Random(100) is below its score. D is the candidate's squared
// distance, or 99,999,999 without a candidate, and w the end's squared
// distance (wormholeD2). Only a fleet at an own planet before year index
// 120 asks.
func (v *View) preferWormhole(y int, rng engine.Rand, from engine.Point, target, dd int) (Wormhole, bool) {
	if y >= 120 {
		return Wormhole{}, false
	}
	if target < 0 {
		dd = noCandidateD2
	}
	wrap := v.Rules.Legacy.AIWormholeDistanceWrap
	var best Wormhole
	bs, bw, found := 0, 0, false
	for _, w := range v.Wormholes {
		wd := wormholeD2(from, w.Pos, wrap)
		if wd > 4*dd {
			continue
		}
		s := 50
		switch {
		case w.Known:
			s = (7 - w.Class) * 10
		case wd <= dd:
			s = 90
		}
		if !found || s > bs || (s == bs && wd < bw) {
			best, bs, bw, found = w, s, wd, true
		}
	}
	if !found || rng.Intn(100) >= bs {
		return Wormhole{}, false
	}
	return best, true
}

// noCandidateD2 is the candidate's squared distance in the wormhole
// preference when the search found no planet (AI.md §11 "Wormhole
// distance arithmetic").
const noCandidateD2 = 99_999_999

// wormholeD2 is a wormhole end's squared distance w in the wormhole
// preference. With wrap (LEGACY BUG, AI.md §11 "Wormhole distance
// arithmetic", BINARY-ONLY) it is computed in 16 bits: the sum of the
// squares modulo 65,536, read as a signed value, so an end about 182 to
// 255 ly away has a negative w and one 256 ly away in x has w = 0.
// Without it, w is exact.
func wormholeD2(a, b engine.Point, wrap bool) int {
	d := d2(a, b)
	if wrap {
		d = int(int16(uint16(d)))
	}
	return d
}

// nearestOwnStarbase is AI.md §11 "Nearest own starbase" from a position.
func (v *View) nearestOwnStarbase(from engine.Point) (int, bool) {
	best, bd := -1, 0
	for _, p := range v.Planets {
		if !p.HasStarbase {
			continue
		}
		if dd := d2(from, p.Pos); best < 0 || dd < bd || (dd == bd && p.ID < best) {
			best, bd = p.ID, dd
		}
	}
	return best, best >= 0
}

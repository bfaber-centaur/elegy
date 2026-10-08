package ai

import "github.com/bfaber-centaur/elegy/engine"

// researchAndStarbases is the start of every personality's turn (AI.md
// §1): research (§4) and starbase designs (§5).
func researchAndStarbases(pers Personality, v *View, rng engine.Rand, res *Result) {
	y := v.Year - FirstYear
	if o, ok := Research(pers, y, v.Self.Research, v.Self.ResearchBudget); ok {
		res.Orders = append(res.Orders, o)
	}
	sb := StarbaseInput{Personality: pers, Year: v.Year, Race: v.Self.Race, Levels: v.Self.Research.Levels}
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
// planet search: wormholes within twice the candidate's distance (any
// distance without a candidate) score (7 − class)·10 when known, else 90
// when nearer than the candidate or 50 otherwise; the best (ties: nearer)
// is taken when Random(100) is below its score. Only a fleet at an own
// planet before year index 120 asks.
//
// The original's distance test overflows for wormholes about 182 ly or
// more away, which then count as near (LEGACY BUG). This code compares
// exact distances; it is inert until the game supplies wormholes, and
// whether to reproduce the overflow is an open decision
// (docs/AI-STATUS.md).
func (v *View) preferWormhole(y int, rng engine.Rand, from engine.Point, target, dd int) (Wormhole, bool) {
	if y >= 120 {
		return Wormhole{}, false
	}
	var best Wormhole
	bs, bdd, found := 0, 0, false
	for _, w := range v.Wormholes {
		wd := d2(from, w.Pos)
		if target >= 0 && wd > 4*dd {
			continue
		}
		s := 50
		switch {
		case w.Known:
			s = (7 - w.Class) * 10
		case target < 0 || wd < dd:
			s = 90
		}
		if !found || s > bs || (s == bs && wd < bdd) {
			best, bs, bdd, found = w, s, wd, true
		}
	}
	if !found || rng.Intn(100) >= bs {
		return Wormhole{}, false
	}
	return best, true
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

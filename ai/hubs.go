package ai

import "github.com/bfaber-centaur/elegy/engine"

// maxHubs is the most hubs a computer player keeps (AI.md §6).
const maxHubs = 64

// hubs is AI.md §6 (BINARY-ONLY) for Robotoid and Rototill from year index
// 20: every own planet with a starbase is a hub, and a starbase-less own
// planet is one with at least 8,000 colonists, 20 mines, 20 factories and
// Σ over minerals (surface + 4·concentration²) ≥ 7000 (Robotoid: only with
// no hub within 50 ly). The list starts empty every year (AI.md §1), so
// step 1's drop never matters. order is the turn's shuffled own-planet
// order. Freighter assignment (steps 3 and 4) is not implemented: neither
// Rototill nor this package's planners have freighters yet.
//
// ASSUMPTION A11: own planets are taken in the shuffled order, starbase
// planets first (robotoid.md §4 pass B takes starbase planets in that
// order; AI.md §6 does not order the rest).
func (v *View) hubs(pers Personality, order []int) []int {
	if v.Year-FirstYear < 20 || pers == Cybertron {
		return nil
	}
	var hubs []int
	for _, id := range order {
		if p := v.ownPlanet(id); p != nil && p.HasStarbase && len(hubs) < maxHubs {
			hubs = append(hubs, id)
		}
	}
	for _, id := range order {
		p := v.ownPlanet(id)
		if p == nil || p.HasStarbase || len(hubs) >= maxHubs {
			continue
		}
		rich := 0
		for m := range engine.NumMinerals {
			c := p.Deposits[m].Concentration
			rich += p.Surface[m] + 4*c*c
		}
		if p.Population*100 < 8000 || p.Mines < 20 || p.Factories < 20 || rich < 7000 {
			continue
		}
		if pers == Robotoid && v.hubWithin(hubs, p.Pos, 50) {
			continue
		}
		hubs = append(hubs, id)
	}
	return hubs
}

func (v *View) hubWithin(hubs []int, pos engine.Point, r int) bool {
	for _, h := range hubs {
		if q := v.ownPlanet(h); q != nil && d2(q.Pos, pos) <= r*r {
			return true
		}
	}
	return false
}

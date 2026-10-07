package engine

// EventOriginalDrift: a Claim Adjuster planet's original environment moved
// one click toward the owner's centre. Axes = the axis.
const EventOriginalDrift EventKind = EventColonistsKilledByEngine + 1

// terraformReach is how far a race at these levels can terraform each
// axis from its original value (KERNEL.md "Terraforming", reach per
// axis): the largest available terraform part for the axis, a Total
// Terraform part counting for every axis. An immune axis has no reach.
func terraformReach(race Race, levels [NumFields]int) [3]int {
	var reach [3]int
	for _, c := range Components().Components {
		if c.Category != CatTerraform {
			continue
		}
		if ok, err := c.Buildable(race, levels, false); !ok || err != nil {
			continue
		}
		amount := c.num("amount")
		for axis, name := range [3]string{"gravity", "temperature", "radiation"} {
			if a := c.str("axis"); a == name || a == "all" {
				reach[axis] = max(reach[axis], amount)
			}
		}
	}
	for axis, r := range race.Env {
		if r.Immune {
			reach[axis] = 0
		}
	}
	return reach
}

// terraformLimit is the value an axis can be terraformed to: from the
// current value toward the centre, within orig ± reach clipped to 1–99,
// stopping at the centre. It is the current value when there is no room.
func terraformLimit(cur, orig, centre, reach int) int {
	switch {
	case cur < centre:
		return max(cur, min(centre, 99, orig+reach))
	case cur > centre:
		return min(cur, max(centre, 1, orig-reach))
	}
	return cur
}

// claimAdjusterYearEnd is KERNEL.md "Turn order" step 7.3, "Terraforming"
// → Claim Adjuster (CONFIRMED, KX-003 S3/S3L, KX-005, TK-108,
// TK-118..121). For each Claim Adjuster planet in planet order the
// original value of one axis may drift a click toward the owner's centre,
// then every axis moves in one step to its limit with the owner's reach
// at the levels after this year's research. Nothing is built or spent.
//
// After a capture the planet holds the original environment the capture
// restored (TAKEOVER.md), so the reach is measured from it.
func (g *Game) claimAdjusterYearEnd(rng Rand) []Event {
	var events []Event
	for _, i := range g.planetOrder() {
		p := &g.Planets[i]
		if p.Owner == NoOwner || g.Players[p.Owner].Race.PRT != PRTClaimAdjuster {
			continue
		}
		pl := &g.Players[p.Owner]
		axis := rng.Intn(3)
		r := pl.Race.Env[axis]
		if !r.Immune && p.OrigEnv[axis] != r.Center && rng.Intn(10) == 0 &&
			(p.Population >= 1000 || rng.Intn(1000) < p.Population) {
			if p.OrigEnv[axis] < r.Center {
				p.OrigEnv[axis]++
			} else {
				p.OrigEnv[axis]--
			}
			events = append(events, Event{Kind: EventOriginalDrift, Player: p.Owner, Planet: p.ID, Fleet: -1, Axes: []int{axis}})
		}
		reach := terraformReach(pl.Race, pl.Research.Levels)
		for a, r := range pl.Race.Env {
			if r.Immune {
				continue
			}
			p.Env[a] = terraformLimit(p.Env[a], p.OrigEnv[a], r.Center, reach[a])
		}
	}
	return events
}

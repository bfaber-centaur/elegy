package engine

// TaskScrap is the "scrap fleet" waypoint task (TAKEOVER.md "Other
// waypoint tasks", CONFIRMED T-34).
const TaskScrap TaskKind = TaskRoute + 5

// Scrap messages. Elegy's wording.
const (
	// EventFleetScrapped: Player's Fleet was scrapped at Planet (−1 in
	// deep space).
	EventFleetScrapped EventKind = iota + EventPacketDesignSeen + 1
	// EventScrapRecycled: ships scrapped at Player's Planet added Count
	// resources to its production (Ultimate Recycling).
	EventScrapRecycled
)

// scrapYear collects what the year's scraps leave for production: the
// ships' resource cost for their owners, by planet index, at planets
// whose owner has Ultimate Recycling (KERNEL.md "Production").
type scrapYear map[int]int

// scrap scraps fleet f where it is (TAKEOVER.md "Other waypoint tasks",
// CONFIRMED T-34), before movement only. With C the fleet's mineral cost
// for its owner (ORDERS.md and COMPONENTS.md "Cost for an owner"), each
// mineral gives:
//   - at a planet with a starbase, 4C/5, or 9C/10 when the planet's owner
//     has Ultimate Recycling;
//   - at a planet without a starbase, C/3, or 9C/20 with Ultimate
//     Recycling;
//   - in deep space, C/3 as a new salvage object, owned by the fleet's
//     owner and marked fresh (OBJECTS.md "Salvage": scrapping in deep
//     space always makes a new object, which keeps its full amount
//     through the year it was made).
//
// The fleet's mineral cargo is added on top. Colonists join the planet
// only when the fleet's owner owns it (before growth); elsewhere they are
// lost. At a starbase the planet's owner makes one tech attempt against
// the highest requirement each field reaches among the scrapped ships'
// hulls and parts (TAKEOVER.md; COMBAT.md "Tech from battle"; CONFIRMED
// TK-203), drawing as the scrap runs (KERNEL.md "Random draw order",
// CONFIRMED KB-3A). The fleet's owner gains nothing.
//
// Each mineral is floor(k·C/d) of the whole fleet's cost (T-34: 9C/20 of
// 210 kT gave 94).
//
// ASSUMPTION K9: the production bonus, like the minerals, follows the
// planet owner's Ultimate Recycling (KERNEL.md "Production" says "a planet
// with Ultimate Recycling"; KB-2A had one player in both roles).
//
// Not modelled: the 30,000 kT limit on one salvage object (OBJECTS.md
// "Salvage") for a scrap in deep space.
func (g *Game) scrap(fi int, rng Rand, gained map[int]bool, recycled scrapYear) []Event {
	f := &g.Fleets[fi]
	owner := g.Players[f.Owner]
	var cost Cost
	var seen [NumFields]int
	for _, s := range f.Stacks {
		if s.Count <= 0 {
			continue
		}
		d := g.Designs[s.Design]
		c := designCost(d, owner.Race, owner.Research.Levels)
		cost.Resources += s.Count * c.Resources
		for m := range NumMinerals {
			cost.Minerals[m] += s.Count * c.Minerals[m]
		}
		req := d.techReq()
		for k := range NumFields {
			seen[k] = max(seen[k], req[k])
		}
	}
	pi := g.planetAt(f.Pos)
	events := []Event{{Kind: EventFleetScrapped, Player: f.Owner, Planet: -1, Fleet: f.ID}}
	num, den := 1, 3
	if pi >= 0 {
		p := &g.Planets[pi]
		events[0].Planet = p.ID
		ur := p.Owner != NoOwner && g.Players[p.Owner].Race.LRT.UltimateRecycling
		switch {
		case p.HasStarbase && ur:
			num, den = 9, 10
		case p.HasStarbase:
			num, den = 4, 5
		case ur:
			num, den = 9, 20
		}
		if ur {
			recycled[pi] += cost.Resources
		}
	}
	var left Minerals
	for m := range NumMinerals {
		left[m] = cost.Minerals[m]*num/den + f.Cargo.Minerals[m]
	}
	if pi < 0 {
		if left != (Minerals{}) {
			g.newSalvage(f.Owner, f.Pos, left, true)
		}
		return events
	}
	p := &g.Planets[pi]
	for m := range NumMinerals {
		p.Surface[m] += left[m]
	}
	if p.Owner == f.Owner {
		p.Population += f.Cargo.Colonists
	}
	if p.HasStarbase && p.Owner != NoOwner {
		events = append(events, techAttempt(g, rng, p.Owner, seen, gained)...)
	}
	return events
}

// recycledResources is a planet's production resources r raised by x,
// the resource cost of the ships scrapped there this year:
// r + trunc(x·r/(x + r)) (KERNEL.md "Production", CONFIRMED KB-2A:
// x 2,410, r 500 → 914).
func recycledResources(r, x int) int {
	if x <= 0 || x+r <= 0 {
		return r
	}
	return r + x*r/(x+r)
}

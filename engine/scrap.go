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
// The attempt's Mystery Trader chances are the scrapped ships' Trader
// part counts (TAKEOVER.md "Other waypoint tasks": "the part's chance is
// the number seen", MEASURED TK-305: 12 one-ship designs with 2
// Hush-a-Boom each gave 24). ASSUMPTION K11: a stack counts its part
// counts once per ship, and the chance is capped at 25 as in battle
// (COMBAT.md "Mystery Trader chances"); TK-305 had one ship per design
// and stayed below the cap.
//
// Each mineral is floor(k·C/d) of the whole fleet's cost (T-34: 9C/20 of
// 210 kT gave 94).
//
// ASSUMPTION K9: the production bonus, like the minerals, follows the
// planet owner's Ultimate Recycling (KERNEL.md "Production" says "a planet
// with Ultimate Recycling"; KB-2A had one player in both roles).
//
// In deep space the salvage is a new object under the 30,000 kT limit
// (OBJECTS.md "Salvage": "Scrapping in deep space always makes a new
// object", and every addition goes in under the limit; COMBAT.md
// "Salvage" gives the 10 kT steps, the overflow CONFIRMED CB-040): what
// does not fit goes into further objects of the fleet's owner at the same
// spot, which are not marked fresh ("An overflow object made by the
// 30,000 kT limit is not marked").
func (g *Game) scrap(fi int, rng Rand, gained map[int]bool, recycled scrapYear) []Event {
	f := &g.Fleets[fi]
	owner := g.Players[f.Owner]
	var cost Cost
	var seen [NumFields]int
	var chance traderChances
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
		chance.addParts(g, d, s.Count)
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
			for i, m := range splitSalvage(left) {
				g.newSalvage(f.Owner, f.Pos, m, i == 0)
			}
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
		events = append(events, techAttempt(g, rng, p.Owner, seen, chance, gained)...)
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

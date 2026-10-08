package ai

import (
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// automation is AI.md §7 "Planet automation" (BINARY-ONLY) after a
// personality's own work, for Robotoid, Rototill and Cybertron. order is
// the turn's shuffled own-planet order, hubs the turn's hubs, marked the
// planets the personality's own pass marked, budget the research budget
// of this turn's research order.
//
// Implemented: step 2 (starbases for hubs; Cybertron's own rule,
// cybertron.md §4.3, instead), step 3 (starbase upgrade and defenses; the planetary scanner
// step queues nothing in J-RC3; packet flings need a mass driver of warp
// 10, which none of the three personalities' starbases has outside PP,
// and Cybertron is excluded; terraforming is never run by Robotoid or
// Cybertron, and CA cannot build terraform items), and the mines and
// factories fill.
//
// Not implemented: step 1 (the warp re-pick, AI.md §11 "Warp choice"),
// step 4 (under attack) and step 5 (blocked queues); docs/AI-STATUS.md.
type automation struct {
	v      *View
	pers   Personality
	rng    engine.Rand
	q      *queues
	order  []int
	hubs   []int
	marked map[int]bool
	budget int
}

func (a *automation) run() {
	if a.pers == Cybertron {
		a.cybertronStarbases()
	} else {
		a.hubStarbases()
	}
	for _, id := range a.order {
		a.perPlanet(a.v.ownPlanet(id))
	}
	for _, id := range a.order {
		a.minesAndFactories(a.v.ownPlanet(id))
	}
}

// currentStation is the current Space Station design slot: 0, or 5 when
// slot 5 is newer (AI.md §7 step 2 and "Starbase upgrade").
func (a *automation) currentStation() int { return a.newer(0, 5) }

// currentFort is the current Orbital Fort slot: 1, or 6 when slot 6 is
// newer.
func (a *automation) currentFort() int { return a.newer(1, 6) }

func (a *automation) newer(x, y int) int {
	dx, okx := a.starbase(x)
	dy, oky := a.starbase(y)
	if oky && (!okx || dy.Created > dx.Created) {
		return y
	}
	return x
}

func (a *automation) starbase(slot int) (Design, bool) {
	for _, d := range a.v.Starbases {
		if d.Slot == slot {
			return d, true
		}
	}
	return Design{}, false
}

func (a *automation) starbaseQueued(p *engine.Planet) bool {
	return a.q.has(p, engine.ItemStarbase)
}

// hubStarbases is step 2: every own planet with no starbase, at least
// 8,000 colonists, not marked, that is a hub, gets the current starbase
// design ×1 unless a starbase is already queued. (Not in a tutorial game
// after year index 30; Elegy has none.)
func (a *automation) hubStarbases() {
	isHub := map[int]bool{}
	for _, h := range a.hubs {
		isHub[h] = true
	}
	for _, id := range a.order {
		p := a.v.ownPlanet(id)
		if p.HasStarbase || p.Population*100 < 8000 || a.marked[id] || !isHub[id] || a.starbaseQueued(p) {
			continue
		}
		a.q.add(p, engine.QueueItem{Kind: engine.ItemStarbase, Slot: a.currentStation(), Count: 1}, false)
	}
}

// cybertronStarbases is Cybertron's step 2 (cybertron.md §4.3, MEASURED
// AI-20): every own planet in planet-id order with no starbase, value
// above 14 and at least 50,000 colonists (after this turn's fleet pass)
// gets, unless a starbase is already queued, ×1 of the current Space
// Station design when every mineral concentration is above 15, else of the
// current Orbital Fort design.
func (a *automation) cybertronStarbases() {
	ids := make([]int, 0, len(a.v.Planets))
	for _, p := range a.v.Planets {
		ids = append(ids, p.ID)
	}
	slices.Sort(ids)
	for _, id := range ids {
		p := a.v.ownPlanet(id)
		if p.HasStarbase || engine.Habitability(a.v.Self.Race, p.Env) <= 14 || p.Population*100 < 50000 || a.starbaseQueued(p) {
			continue
		}
		slot := a.currentStation()
		for _, d := range p.Deposits {
			if d.Concentration <= 15 {
				slot = a.currentFort()
			}
		}
		a.q.add(p, engine.QueueItem{Kind: engine.ItemStarbase, Slot: slot, Count: 1}, false)
	}
}

// perPlanet is step 3 for one planet.
func (a *automation) perPlanet(p *engine.Planet) {
	least := 6000
	if a.pers == Robotoid {
		least = 4000
	}
	if p.Population*100 < least && !a.marked[p.ID] {
		return
	}
	if a.v.available(p, a.budget).short(a.v.queueCost(a.q.get(p))) {
		return
	}
	switch {
	case a.pers == Robotoid || !a.marked[p.ID]:
		if !a.upgrade(p) {
			a.defenses(p)
		}
	case a.pers == Cybertron:
		a.upgrade(p)
	}
}

// family returns the starbase family holding slot s, as variant slots.
func family(s int) []int {
	for _, f := range [][]int{stationA, stationB, fortA, fortB} {
		for _, k := range f {
			if k == s {
				return f
			}
		}
	}
	return nil
}

// upgrade is "Starbase upgrade": a planet with a starbase and no starbase
// item queued (Cybertron only from year index 40).
//   - In the current family: no upgrade at the top variant or when the
//     next variant's slot is empty; otherwise with Random(100) ≤ 5 and
//     every surface mineral ≥ 200, the next variant.
//   - In the old family: e = year index − the current base's creation year
//     index − 10, at least 0, halved below 50; with Random(100) < e + 5,
//     the same variant in the current family.
//
// ASSUMPTION A13: the draw comes before the mineral test, and the upgrade
// is appended.
func (a *automation) upgrade(p *engine.Planet) bool {
	y := a.v.Year - FirstYear
	if !p.HasStarbase || a.starbaseQueued(p) || (a.pers == Cybertron && y < 40) {
		return false
	}
	slot := -1
	for _, d := range a.v.Starbases {
		if d.Index == p.StarbaseDesign {
			slot = d.Slot
		}
	}
	fam := family(slot)
	if fam == nil {
		return false
	}
	cur := a.currentStation()
	if slotHull(slot) == orbitalFort {
		cur = a.currentFort()
	}
	curFam := family(cur)
	variant := 0
	for i, k := range fam {
		if k == slot {
			variant = i
		}
	}
	if fam[0] == curFam[0] {
		if variant == len(fam)-1 {
			return false
		}
		if _, ok := a.starbase(fam[variant+1]); !ok {
			return false
		}
		if a.rng.Intn(100) > 5 {
			return false
		}
		for _, m := range p.Surface {
			if m < 200 {
				return false
			}
		}
		return a.q.add(p, engine.QueueItem{Kind: engine.ItemStarbase, Slot: fam[variant+1], Count: 1}, false)
	}
	base, _ := a.starbase(cur)
	e := max(0, y-(base.Created-FirstYear)-10)
	if e < 50 {
		e /= 2
	}
	if a.rng.Intn(100) >= e+5 {
		return false
	}
	return a.q.add(p, engine.QueueItem{Kind: engine.ItemStarbase, Slot: curFam[variant], Count: 1}, false)
}

// defenses is "Defenses": population ≥ 160,000, defenses below
// population/8,000, none queued and room → min(room, 4) defenses.
//
// ASSUMPTION A14: "room" is the operable defenses (KERNEL.md "Caps")
// less those installed and queued, as AI.md §7 defines room for mines
// and factories; "population/8,000" is in colonists.
func (a *automation) defenses(p *engine.Planet) bool {
	pop := p.Population * 100
	if pop < 160000 || p.Defenses >= pop/8000 || a.q.has(p, engine.ItemDefenses) {
		return false
	}
	col := engine.NewColony(p, &a.v.Self)
	room := col.OperableDefenses(p.Population) - p.Defenses - a.q.count(p, engine.ItemDefenses)
	if room <= 0 {
		return false
	}
	return a.q.add(p, engine.QueueItem{Kind: engine.ItemDefenses, Count: min(room, 4)}, false)
}

// minesAndFactories is the queue fill after the steps (AI.md §7 "Mines and
// factories"), with left = available − queue cost:
//   - resources left < 0: nothing;
//   - some mineral left ≤ 0: unless the head is auto alchemy or alchemy,
//     min(mine room, resources / mine cost) mines at the front;
//   - otherwise min(factory room, Ge left / factory Ge, resources / factory
//     cost) factories at the back, then min(mine room, resources left /
//     mine cost) mines at the front;
//   - every tech at 26 and year index > 100: also resources left /
//     alchemy cost + 1 alchemy.
//
// ASSUMPTION A15: "resources" is the planet's available resources, and
// "resources left" in the mines step is after the factories just
// queued; the alchemy is appended and uses what is left after the mines.
// Room counts queued plain items only.
func (a *automation) minesAndFactories(p *engine.Planet) {
	v := a.v
	race := v.Self.Race
	avail := v.available(p, a.budget)
	left := avail.sub(v.queueCost(a.q.get(p)))
	if left.Resources < 0 {
		return
	}
	col := engine.NewColony(p, &v.Self)
	mineRoom := col.OperableMines(p.Population) - p.Mines - a.q.count(p, engine.ItemMine)
	factRoom := col.OperableFactories(p.Population) - p.Factories - a.q.count(p, engine.ItemFactory)
	mineCost := max(1, race.MineCost)
	starved := false
	for _, m := range left.Minerals {
		if m <= 0 {
			starved = true
		}
	}
	if starved {
		if q := a.q.get(p); len(q) > 0 && (q[0].Kind == engine.ItemAutoAlchemy || q[0].Kind == engine.ItemMineralAlchemy) {
			return
		}
		a.q.add(p, engine.QueueItem{Kind: engine.ItemMine, Count: min(mineRoom, avail.Resources/mineCost)}, true)
		return
	}
	fc := engine.ItemCost(race, engine.ItemFactory)
	n := min(factRoom, avail.Resources/max(1, fc.Resources))
	if ge := fc.Minerals[engine.Germanium]; ge > 0 {
		n = min(n, left.Minerals[engine.Germanium]/ge)
	}
	if a.q.add(p, engine.QueueItem{Kind: engine.ItemFactory, Count: n}, false) {
		left.Resources -= n * fc.Resources
	}
	if m := min(mineRoom, left.Resources/mineCost); a.q.add(p, engine.QueueItem{Kind: engine.ItemMine, Count: m}, true) {
		left.Resources -= m * mineCost
	}
	all26 := true
	for _, l := range v.Self.Research.Levels {
		all26 = all26 && l >= 26
	}
	if all26 && v.Year-FirstYear > 100 {
		ac := max(1, engine.ItemCost(race, engine.ItemMineralAlchemy).Resources)
		a.q.add(p, engine.QueueItem{Kind: engine.ItemMineralAlchemy, Count: max(0, left.Resources)/ac + 1}, false)
	}
}

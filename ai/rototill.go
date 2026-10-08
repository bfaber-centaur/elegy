package ai

import (
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// PlayRototill plays one year for a Rototill (CA) computer player: stars-elegy
// docs/ai/rototill.md, MEASURED over 166 player-years (AI-14..AI-17)
// except the branches marked not exercised there. v is changed as the turn
// plans.
//
// Steps (rototill.md §1):
//  1. research (AI.md §4) and starbase designs (AI.md §5); Rototill never
//     makes, deletes, ages, splits or merges ship designs (AI-14);
//  2. U, the planet loop and production (§2);
//  3. fleet passes 1 and 2 (§3);
//  4. planet automation and the queue fill (AI.md §7), with the turn's
//     shuffled planet order (AI.md §2) and hubs (AI.md §6).
//
// Not yet run: the warp re-pick (AI.md §11 "Warp choice") and
// automation steps 4 and 5; docs/AI-STATUS.md lists them.
func PlayRototill(v *View, rng engine.Rand) Result {
	var res Result
	v.fleetOrder()
	y := v.Year - FirstYear

	if o, ok := Research(Rototill, y, v.Self.Research, v.Self.ResearchBudget); ok {
		res.Orders = append(res.Orders, o)
	}
	sb := StarbaseInput{Personality: Rototill, Year: v.Year, Race: v.Self.Race, Levels: v.Self.Research.Levels}
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

	var own []int
	for _, p := range v.Planets {
		own = append(own, p.ID)
	}
	order := ShufflePlanets(own, rng)
	hubs := v.hubs(Rototill, order)

	t := &rototillTurn{v: v, rng: rng, res: &res, y: y, alive: v.alive(), taken: map[int]bool{}, q: &queues{v: v}}
	t.planets()
	t.pass1()
	t.pass2()
	auto := &automation{v: v, pers: Rototill, rng: rng, q: t.q, order: order, hubs: hubs,
		marked: t.flagged, budget: ResearchBudget(Rototill, y, v.Self.Research.Levels)}
	auto.run()
	res.Orders = append(res.Orders, t.q.flush()...)
	return res
}

type rototillTurn struct {
	v     *View
	rng   engine.Rand
	res   *Result
	y     int
	alive map[int]int
	// dest is the armada-destination score of other players' planets;
	// habitable marks those habitable for Rototill after terraforming.
	dest      map[int]int
	habitable map[int]bool
	// flagged are own planets with negative desirability, for planet
	// automation.
	flagged map[int]bool
	// taken are planets an own fleet targets, and planets chosen this
	// turn (rototill.md §3).
	taken map[int]bool
	q     *queues
}

// planets is rototill.md §2 (MEASURED, AI-15): U, then every planet in
// planet-id order.
//
// ASSUMPTION A8: "known at level 3 or more" is a report at Elegy's normal
// level or above (environment and concentrations known). rototill.md
// leaves its meaning open. "Desirability" is the planet's value for the
// race (engine.Habitability).
func (t *rototillTurn) planets() {
	v := t.v
	t.dest, t.habitable, t.flagged = map[int]int{}, map[int]bool{}, map[int]bool{}
	u := 0
	for _, pp := range v.Universe {
		r, known := v.Known[pp.ID]
		if !known || v.owner(pp.ID) != engine.NoOwner {
			continue
		}
		if h, ok := v.terraformedHab(r); ok && h > 0 {
			u++
		}
	}
	queued := false
	for _, pp := range v.Universe {
		id := pp.ID
		owner := v.owner(id)
		switch {
		case owner != engine.NoOwner && owner != v.Player:
			r := v.Known[id]
			t.dest[id] = 1
			if r.Starbase {
				t.dest[id] = 2
			}
			if h, ok := v.terraformedHab(r); ok && h > 0 {
				t.habitable[id] = true
			}
		case owner == v.Player:
			p := v.ownPlanet(id)
			if engine.Habitability(v.Self.Race, p.Env) < 0 {
				t.flagged[id] = true
				continue
			}
			if queued || t.y == 0 || !p.HasStarbase || p.Population*100 < 100000 || t.starbaseQueued(p) {
				continue
			}
			// The first qualifying planet of the year decides; at most one
			// colony ship a year.
			queued = true
			if n := t.alive[1]; n == 0 || n+1 < u {
				t.q.add(p, engine.QueueItem{Kind: engine.ItemShip, Slot: 1, Count: 1}, false)
			}
		}
	}
}

// starbaseQueued is rototill.md §2's queue test: a starbase design other
// than starbase slot 0 in the queue (LEGACY BUG reproduced: a queued ship
// does not count).
func (t *rototillTurn) starbaseQueued(p *engine.Planet) bool {
	for _, it := range t.q.get(p) {
		if it.Kind == engine.ItemStarbase && it.Slot != 0 {
			return true
		}
	}
	return false
}

// pass1 is rototill.md §3 pass 1 for own fleets: transports and fleets
// with Colony Ships whose planet belongs to another player either invade
// it or have their route cut. The enemy and attack lists pass 1 also
// builds feed only rules Rototill cannot reach (armed scouts, slots 13
// and 14), and the remote-miner rules need slots 7 and 8, which Rototill
// never designs (AI-14); neither is built here.
func (t *rototillTurn) pass1() {
	v := t.v
	for i := range v.Fleets {
		f := &v.Fleets[i]
		colony := v.holds(f, 1)
		trans := v.transport(f)
		if !colony && !trans {
			continue
		}
		var id int
		var ok bool
		if len(f.Waypoints) > 0 {
			if f.Waypoints[0].Target == engine.TargetPlanet {
				id, ok = f.Waypoints[0].ID, true
			}
		} else {
			id, ok = v.planetAt(f.Pos)
		}
		if !ok {
			continue
		}
		if _, known := v.Known[id]; !known && v.ownPlanet(id) == nil {
			if trans {
				t.res.Orders = append(t.res.Orders, cutRoute(f))
			}
			continue
		}
		owner := v.owner(id)
		if owner == engine.NoOwner || owner == v.Player {
			continue
		}
		if t.habitable[id] && f.Cargo.Colonists > 0 && v.PRT[owner] != engine.PRTAlternateReality {
			task := engine.Task{Kind: engine.TaskTransport}
			task.Transport[engine.CargoColonists] = engine.Transport{Action: engine.UnloadAll}
			pos, _ := v.planetPos(id)
			if pos == f.Pos {
				f.Task = task
				t.res.Orders = append(t.res.Orders, engine.WaypointOrder{Fleet: f.ID, Task: task, Waypoints: slices.Clone(f.Waypoints)})
			} else {
				f.Waypoints[0].Task = task
				t.res.Orders = append(t.res.Orders, engine.WaypointOrder{Fleet: f.ID, Task: f.Task, Waypoints: slices.Clone(f.Waypoints)})
			}
			continue
		}
		t.res.Orders = append(t.res.Orders, cutRoute(f))
	}
}

// pass2 is rototill.md §3 pass 2 for own fleets in fleet order. Fleets
// with two or more waypoints get no order.
func (t *rototillTurn) pass2() {
	v := t.v
	for i := range v.Fleets {
		f := &v.Fleets[i]
		if len(f.Waypoints) > 0 && f.Waypoints[0].Target == engine.TargetPlanet {
			t.taken[f.Waypoints[0].ID] = true
		}
	}
	for i := range v.Fleets {
		f := &v.Fleets[i]
		if len(f.Waypoints) > 0 {
			continue
		}
		switch {
		case v.holds(f, 1):
			t.colonyShip(f)
		case v.transport(f):
			// Not exercised; Rototill never designs a transport. With no
			// own starbase planet, pass 2 stops here for every later
			// fleet (LEGACY BUG, reproduced). Otherwise the hub-freighter
			// rule (AI.md §11) applies, which is not implemented.
			if _, ok := t.nearestOwnStarbase(f.Pos); !ok {
				return
			}
			t.res.unsupported("fleet %d: hub freighter rule", f.ID)
		case v.holds(f, 13, 14):
			// Unreachable: Rototill never designs slots 13 and 14.
		case v.holds(f, 0):
			t.scout(f)
		}
	}
}

// colonyShip is pass 2 step 1.
func (t *rototillTurn) colonyShip(f *engine.Fleet) {
	v := t.v
	here, orbits := v.planetAt(f.Pos)
	var own *engine.Planet
	if orbits {
		own = v.ownPlanet(here)
	}
	if (own != nil && own.Population*100 >= 5000) || f.Cargo.Colonists > 0 {
		if own != nil {
			t.load(f, own, 25) // 2,500 colonists
		}
		target, dd := t.nearestColonizable(f.Pos)
		if d, _ := v.ship(1); engineRank(d.Design.Engine.Name) > engineRank("Quick Jump 5") && orbits && own != nil {
			if w, ok := t.preferWormhole(f.Pos, target, dd); ok {
				t.res.Orders = append(t.res.Orders, moveOrder(f, engine.Waypoint{Pos: w.Pos, Warp: v.idealWarp(f), Target: engine.TargetWormhole, ID: w.End}))
				return
			}
		}
		if target >= 0 {
			pos, _ := v.planetPos(target)
			t.taken[target] = true
			t.res.Orders = append(t.res.Orders, moveOrder(f, toPlanet(target, pos, v.idealWarp(f), engine.Task{Kind: engine.TaskColonize})))
		}
		return
	}
	if own != nil && own.HasStarbase {
		return
	}
	if d, _ := v.ship(1); engineRank(d.Design.Engine.Name) >= engineRank("Fuel Mizer") {
		if id, ok := t.nearestOwnStarbase(f.Pos); ok {
			pos, _ := v.planetPos(id)
			t.res.Orders = append(t.res.Orders, moveOrder(f, toPlanet(id, pos, 4, engine.Task{})))
			return
		}
	}
	t.res.unsupported("fleet %d: scrap (rototill.md §3 pass 2 step 1)", f.ID)
}

// load moves up to n units of colonists from an own planet into the
// fleet, limited by the planet's population and the free hold, and
// changes both in the planner's picture (AI.md §11 "Supplies").
func (t *rototillTurn) load(f *engine.Fleet, p *engine.Planet, n int) {
	n = min(n, p.Population, t.v.cargoCapacity(f)-cargoMass(f.Cargo))
	if n <= 0 {
		return
	}
	p.Population -= n
	f.Cargo.Colonists += n
	var amounts [engine.NumCargo + 1]int
	amounts[engine.CargoColonists] = n
	t.res.Orders = append(t.res.Orders, engine.CargoOrder{Fleet: f.ID, Target: engine.TargetPlanet, ID: p.ID, Amounts: amounts})
}

// nearestColonizable is AI.md §11 "Nearest colonizable planet" for
// Rototill (MEASURED AI-16): planets unowned in its view that are in its
// view and whose habitability after terraforming is not negative, not
// taken; the nearest by squared distance, first in planet-id order on
// ties. It returns −1 when there is none.
//
// rototill.md §3 marks the chosen planet so later fleets this turn skip
// it; AI.md §11 says the non-Robotoid marks are computed once per turn.
// The personality file is followed (see docs/AI-STATUS.md).
func (t *rototillTurn) nearestColonizable(from engine.Point) (int, int) {
	v := t.v
	best, bd := -1, 0
	for _, pp := range v.Universe {
		r, known := v.Known[pp.ID]
		if !known || t.taken[pp.ID] || v.owner(pp.ID) != engine.NoOwner {
			continue
		}
		if h, ok := v.terraformedHab(r); !ok || h < 0 {
			continue
		}
		if dd := d2(from, pp.Pos); best < 0 || dd < bd {
			best, bd = pp.ID, dd
		}
	}
	return best, bd
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
func (t *rototillTurn) preferWormhole(from engine.Point, target, dd int) (Wormhole, bool) {
	if t.y >= 120 {
		return Wormhole{}, false
	}
	var best Wormhole
	bs, bdd, found := 0, 0, false
	for _, w := range t.v.Wormholes {
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
	if !found || t.rng.Intn(100) >= bs {
		return Wormhole{}, false
	}
	return best, true
}

// nearestOwnStarbase is AI.md §11 "Nearest own starbase" from a position.
func (t *rototillTurn) nearestOwnStarbase(from engine.Point) (int, bool) {
	best, bd := -1, 0
	for _, p := range t.v.Planets {
		if !p.HasStarbase {
			continue
		}
		if dd := d2(from, p.Pos); best < 0 || dd < bd || (dd == bd && p.ID < best) {
			best, bd = p.ID, dd
		}
	}
	return best, best >= 0
}

// scout is pass 2 step 4 for slot-0 fleets.
func (t *rototillTurn) scout(f *engine.Fleet) {
	v := t.v
	d, _ := v.ship(0)
	if d.Design.Engine.Name == "Quick Jump 5" && f.Fuel < 2 {
		t.res.unsupported("fleet %d: scrap (rototill.md §3 pass 2 step 4)", f.ID)
		return
	}
	if armed(d.Design) {
		// Not reachable for CA's Scout: the shared attack-target rule.
		t.res.unsupported("fleet %d: armed scout attack target", f.ID)
		return
	}
	best, bd := -1, 0
	for _, pp := range v.Universe {
		if _, seen := v.Known[pp.ID]; seen || t.taken[pp.ID] {
			continue
		}
		if dd := d2(f.Pos, pp.Pos); best < 0 || dd < bd {
			best, bd = pp.ID, dd
		}
	}
	warp := v.idealWarp(f)
	if _, orbits := v.planetAt(f.Pos); orbits && best >= 0 {
		if w, ok := t.nearestWormhole(f.Pos, bd); ok && t.rng.Intn(100) < 5 {
			t.res.Orders = append(t.res.Orders, moveOrder(f, engine.Waypoint{Pos: w.Pos, Warp: warp, Target: engine.TargetWormhole, ID: w.End}))
			return
		}
	}
	if best < 0 {
		if w, ok := t.nearestWormhole(f.Pos, -1); ok {
			t.res.Orders = append(t.res.Orders, moveOrder(f, engine.Waypoint{Pos: w.Pos, Warp: warp, Target: engine.TargetWormhole, ID: w.End}))
			return
		}
		best = t.bestDestination()
		if best < 0 && len(v.Universe) > 0 {
			best = v.Universe[t.rng.Intn(len(v.Universe))].ID
		}
	}
	if best < 0 {
		return
	}
	t.taken[best] = true
	pos, _ := v.planetPos(best)
	t.res.Orders = append(t.res.Orders, moveOrder(f, toPlanet(best, pos, warp, engine.Task{})))
}

// nearestWormhole is the nearest known wormhole end within squared
// distance limit (any distance when limit < 0).
//
// ASSUMPTION A9: rototill.md's scout wormhole ("a wormhole within that
// distance") is the nearest one, and Random(100) is drawn only when one
// exists.
func (t *rototillTurn) nearestWormhole(from engine.Point, limit int) (Wormhole, bool) {
	var best Wormhole
	bd, found := 0, false
	for _, w := range t.v.Wormholes {
		wd := d2(from, w.Pos)
		if limit >= 0 && wd > limit {
			continue
		}
		if !found || wd < bd {
			best, bd, found = w, wd, true
		}
	}
	return best, found
}

// bestDestination is the scout fallback's "best armada destination
// counted from the homeworld". ASSUMPTION A10: the highest destination
// score of §2, ties to the planet nearer the homeworld, then the lower
// id; −1 when no other player's planet is known.
func (t *rototillTurn) bestDestination() int {
	v := t.v
	var home engine.Point
	for _, p := range v.Planets {
		if p.Homeworld {
			home = p.Pos
		}
	}
	best, bs, bd := -1, 0, 0
	for _, pp := range v.Universe {
		s := t.dest[pp.ID]
		if s == 0 {
			continue
		}
		dd := d2(home, pp.Pos)
		if best < 0 || s > bs || (s == bs && dd < bd) {
			best, bs, bd = pp.ID, s, dd
		}
	}
	return best
}

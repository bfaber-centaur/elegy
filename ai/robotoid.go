package ai

import (
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// PlayRobotoid plays one year for a Robotoid (HE) computer player:
// stars-elegy docs/ai/robotoid.md. Ship designs are CONFIRMED (AI-8), the
// fleet passes MEASURED (AI-12) and the production order MEASURED in
// structure (AI-9); the rest is BINARY-ONLY. v is changed as the turn
// plans.
//
// Steps (robotoid.md §1):
//  1. research, starbase designs and hubs (AI.md §4–§6);
//  2. from year index 51, merges by design slot (AI.md §10);
//  3. the armada parameters;
//  4. design ageing;
//  5. from year index 81, splits (AI.md §10);
//  6. ship designs (§2);
//  7. the guard-fleet count, threat marks and production (§3);
//  8. fleets (§4);
//  9. planet automation (AI.md §7) and the queue fill.
//
// Not yet run: automation steps 4 and 5. Steps the engine cannot order
// (load tasks) go to Result.Unsupported.
func PlayRobotoid(v *View, rng engine.Rand) Result {
	var res Result
	v.fleetOrder()
	y := v.Year - FirstYear
	researchAndStarbases(Robotoid, v, rng, &res)
	var own []int
	for _, p := range v.Planets {
		own = append(own, p.ID)
	}
	order := ShufflePlanets(own, rng)
	hubs := v.hubs(Robotoid, order)

	t := &robotoidTurn{turn: newTurn(v, rng, &res), order: order, hubs: hubs}
	if y > 50 {
		// ASSUMPTION A30: "the slots 2–7 and 9–10" is one merge.
		for _, g := range [][]int{{2, 3, 4, 5, 6, 7, 9, 10}, {0}, {14, 15}} {
			t.merge(g)
		}
	}
	t.params()
	t.ageing()
	if y > 80 {
		t.splitOut(func(k int) bool { return t.obsolete[k] })
		for _, g := range [][]int{{0}, {1}, {11, 12, 13}} {
			t.splitOut(func(k int) bool { return slices.Contains(g, k) })
		}
	}
	t.sd.robotoidDesigns(v.Level, v.alive())
	t.sd.syncView(v)
	for i := range v.Fleets {
		if v.holds(&v.Fleets[i], 14, 15) {
			t.guardFleets++
		}
	}
	t.notes()
	t.colonizerYes = t.colonizerTest()
	budget := ResearchBudget(Robotoid, y, v.Self.Research.Levels)
	for _, id := range order {
		t.produce(v.ownPlanet(id), budget)
	}
	t.passA()
	t.passB()
	t.passC()
	auto := &automation{v: v, res: &res, pers: Robotoid, rng: rng, q: t.q, order: order, hubs: hubs, budget: budget}
	auto.run()
	res.Orders = append(res.Orders, t.q.flush()...)
	return res
}

type robotoidTurn struct {
	*turn
	order, hubs []int

	// potency P and armada size A (§1 step 3); the age limit T.
	potency, size, limit int
	// The groups' current designs (§1 step 4), −1 for none.
	d1415, d1113, d910, d25, d67 int

	guardFleets  int
	colonizerYes bool
	// enemies and attack are pass A's lists, in reverse fleet order.
	enemies []engine.FleetSighting
	attack  []int
}

// params is §1 step 3.
func (t *robotoidTurn) params() {
	y := t.y
	t.potency, t.size = 4, 6
	if y > 130 {
		t.potency = min(50, (y-120)/20+4)
	}
	if y > 115 {
		t.size = min(12, (y-100)/22+6)
	}
	switch {
	case y < 120:
		t.limit = 50
	case y < 200:
		t.limit = 70
	default:
		t.limit = 100
	}
}

// ageing is §1 step 4: slots 14–15, 11–13, 9–10, 2–5 with limit T and
// 6–7 with 3T/2. A Nubian in slot 14 or 15 is never marked obsolete.
func (t *robotoidTurn) ageing() {
	alive := t.v.alive()
	T := t.limit
	t.d1415, _ = t.sd.ageGroup([]int{14, 15}, T, alive, t.obsolete)
	for _, k := range []int{14, 15} {
		if t.sd.slots[k].hull == "Nubian" {
			delete(t.obsolete, k)
		}
	}
	t.d1113, _ = t.sd.ageGroup([]int{11, 12, 13}, T, alive, t.obsolete)
	t.d910, _ = t.sd.ageGroup([]int{9, 10}, T, alive, t.obsolete)
	t.d25, _ = t.sd.ageGroup([]int{2, 3, 4, 5}, T, alive, t.obsolete)
	t.d67, _ = t.sd.ageGroup([]int{6, 7}, 3*T/2, alive, t.obsolete)
	t.sd.syncView(t.v)
}

// current is a group's current design slot when it still holds a design
// (the slot may have been deleted by ageing), else −1.
func (t *robotoidTurn) current(k int) int {
	if k >= 0 && t.sd.slots[k].present {
		return k
	}
	return -1
}

// universeSize is the universe size 0..4 (tiny to huge).
//
// ASSUMPTION A36: the view does not carry the size; it is read from the
// map's extent (400 ly per size step from tiny's 400).
func universeSize(v *View) int {
	hi := 0
	for _, p := range v.Universe {
		hi = max(hi, p.Pos.X-1000, p.Pos.Y-1000)
	}
	return min(4, max(0, (hi-1)/400))
}

// colonyDesign reports an own design with a Mini-Colony Ship or Colony
// Ship hull.
func colonyHull(h string) bool { return h == "Mini-Colony Ship" || h == "Colony Ship" }

// colonizerTest is §3's colonizer test, once per turn.
//
// ASSUMPTION A37: the ships ever built of the colony designs (b) are not
// known to Elegy's planner, so the rule on b is skipped.
func (t *robotoidTurn) colonizerTest() bool {
	v := t.v
	if t.lvl == Easy && t.y%2 == 1 {
		return false
	}
	if t.y < 30 {
		return true
	}
	any := false
	for _, d := range v.Ships {
		any = any || colonyHull(d.Design.Hull.Name)
	}
	if !any {
		return false
	}
	c := t.colonyFleets()
	if c > 20*universeSize(t.v)+10 {
		return false
	}
	e := 0
	for _, pp := range v.Universe {
		if v.owner(pp.ID) != engine.NoOwner {
			e++
		}
	}
	if 5*(e+c) > 4*len(v.Universe) {
		return false
	}
	if c < 5 {
		return t.rng.Intn(2) == 0
	}
	return false
}

func (t *robotoidTurn) colonyFleets() int {
	c := 0
	for i := range t.v.Fleets {
		for _, s := range t.v.Fleets[i].Stacks {
			if d, ok := t.v.design(s.Design); ok && s.Count > 0 && colonyHull(d.Hull.Name) {
				c++
				break
			}
		}
	}
	return c
}

// strength is AI.md §11's war-fleet strength.
func (t *robotoidTurn) strength(f *engine.Fleet) int {
	return t.count(f, 2, 3, 4, 5) + 2*t.count(f, 6, 7)
}

// produce is §3 for one own planet.
//
// ASSUMPTION A34: the colonizer step checks y > 4 first, and with no hubs
// the Random(8 × hubs) draw is not made (it would be Random(0)).
// ASSUMPTION A35: "population × max growth %" uses the population in
// hundreds and the race's growth rate; "the planet's resources" are its
// resources this year.
// ASSUMPTION A38: with no own fleet here that is not too weak, the armada
// step does nothing and production goes on to the warships.
// ASSUMPTION A39: the D67 draw is made only when D67 exists.
func (t *robotoidTurn) produce(p *engine.Planet, budget int) {
	v := t.v
	if !p.HasStarbase || p.Population*100 < 20000 || t.q.has(p, engine.ItemShip) {
		return
	}
	alive := v.alive()
	planets := len(v.Planets)
	add := func(slot, n int) { t.q.add(p, engine.QueueItem{Kind: engine.ItemShip, Slot: slot, Count: n}, false) }

	// 1. Freighters.
	if d := t.current(t.d1113); d >= 0 {
		f := max(planets/8, 4*len(t.hubs))
		n := alive[11] + alive[12] + alive[13]
		if 10*n < 8*f || (n < f && t.rng.Intn(3) == 0) {
			add(d, 1)
		}
	}
	// 2. Colonizers.
	if t.y > 4 && (t.colonizerYes || (t.colonyFleets() < 26 && len(t.hubs) > 0 && t.rng.Intn(8*len(t.hubs)) == 0)) {
		n := 1
		if t.y < 21 {
			n = 2
		}
		col := engine.NewColony(p, &v.Self)
		pg, r := p.Population*v.Self.Race.GrowthRate, col.Resources(p.Population, p.Factories)
		if pg > 2300 && r > 35 && t.lvl >= Standard {
			n++
		}
		if pg > 3600 && r > 50 && t.lvl >= Harder {
			n++
		}
		add(1, n)
	}
	// 3. Frigates.
	if d, ok := v.ship(0); ok && d.Design.Hull.Name == "Frigate" && t.rng.Intn(4) == 0 {
		s := 0
		for i := range v.Fleets {
			if v.Fleets[i].Pos == p.Pos {
				s = t.count(&v.Fleets[i], 0)
				break
			}
		}
		switch {
		case s < 10:
			if t.rng.Intn(2*s+1) == 0 {
				add(0, 4)
			}
		case s <= 16:
			if t.rng.Intn(10) == 0 && t.rng.Intn(2*s+1) == 0 {
				add(0, 4)
			}
		}
	}
	// 4. rich: the literal reading (LEGACY BUG candidate).
	rich := p.Surface[engine.Ironium] >= 5000 && p.Surface[engine.Boranium] >= 5000 && p.Surface[engine.Germanium] < 5000
	// 5. Armada.
	if d := t.current(t.d910); d >= 0 {
		for i := range v.Fleets {
			f := &v.Fleets[i]
			if f.Pos != p.Pos || t.strength(f) < t.potency {
				continue
			}
			if t.count(f, 9, 10) < t.size {
				n := 4
				if rich {
					n = 6
				}
				add(d, n)
				return
			}
			break
		}
	}
	// 6. Warships.
	left := v.available(p, budget).sub(v.queueCost(t.q.get(p)))
	if d := t.current(t.d25); d >= 0 {
		if alive[d] < planets/7+6 || t.rng.Intn(2) != 0 {
			k := d
			if d67 := t.current(t.d67); d67 >= 0 && t.rng.Intn(2) == 0 {
				k = d67
			}
			if left.short(funds{}) {
				return
			}
			c := v.itemCost(engine.QueueItem{Kind: engine.ItemShip, Slot: k, Count: 1})
			need := funds{Resources: 3 * c.Resources / 5}
			for m := range engine.NumMinerals {
				need.Minerals[m] = 3 * c.Minerals[m] / 5
			}
			if !left.short(need) {
				n := 1
				if rich {
					n = 5
				}
				add(k, n)
				return
			}
		}
	}
	// 7. Slots 14/15.
	if d := t.current(t.d1415); d >= 0 && alive[d] < planets/12+8 {
		c := v.itemCost(engine.QueueItem{Kind: engine.ItemShip, Slot: d, Count: 1})
		left = v.available(p, budget).sub(v.queueCost(t.q.get(p)))
		n := 0
		for n < 5 {
			next := left
			next.Resources -= c.Resources
			for m := range engine.NumMinerals {
				next.Minerals[m] -= c.Minerals[m]
			}
			if next.short(funds{}) {
				break
			}
			left = next
			n++
		}
		if n > 0 {
			add(d, n)
		}
	}
}

// attackFleet is AI.md §11 "Fleet classes": walking its designs in slot
// order (slots with ships only), a warship → yes; a Frigate → yes if its
// power > 0, else no and stop (LEGACY BUG candidate, reproduced); a Nubian
// or Meta Morph with cargo below 500 and power > 0 → yes.
func (t *turn) attackFleet(f *engine.Fleet) bool {
	v := t.v
	type st struct {
		slot int
		d    engine.Design
	}
	var ds []st
	for _, s := range f.Stacks {
		if s.Count <= 0 {
			continue
		}
		if d, ok := v.design(s.Design); ok {
			ds = append(ds, st{v.shipSlot(s.Design), d})
		}
	}
	slices.SortStableFunc(ds, func(a, b st) int { return a.slot - b.slot })
	for _, x := range ds {
		switch h := x.d.Hull.Name; {
		case warshipHull(h):
			return true
		case h == "Frigate":
			return hasPower(x.d)
		case (h == "Nubian" || h == "Meta Morph") && x.d.CargoCapacity < 500 && hasPower(x.d):
			return true
		}
	}
	return false
}

func warshipHull(h string) bool {
	switch h {
	case "Destroyer", "Cruiser", "Battle Cruiser", "Battleship", "Dreadnought":
		return true
	}
	return false
}

// hasPower is "power > 0" for a design (KERNEL.md "Scores and victory
// conditions", "Power of a design"). The power is a sum of non-negative
// beam, torpedo and bomb terms, scaled by capacitors and the battle-speed
// factor, which never turn a positive beam term to zero or below; so it
// is positive exactly when one weapon term is (ASSUMPTION A29, from the
// formula as published).
func hasPower(d engine.Design) bool {
	for _, s := range d.Slots {
		p := s.Part
		switch p.Kind {
		case engine.PartBeam:
			v := (p.Range + 3) * p.Damage * s.Count / 4
			if p.Sapper {
				v /= 3
			}
			if v > 0 {
				return true
			}
		case engine.PartTorpedo:
			if (p.Range-2)*p.Damage*s.Count/2 > 0 {
				return true
			}
		case engine.PartBomb:
			if (p.KillRate+p.InstallKill)*s.Count*2 > 0 {
				return true
			}
		}
	}
	return false
}

// passA is §4 pass A.
//
// ASSUMPTION A43: player positions are not "close" (the view does not
// carry the setting).
//
// ASSUMPTION A54: an attack fleet's waypoint-0 marker task is not
// cleared. robotoid.md refers to it ("below") without defining it; the
// only task Elegy's Robotoid sets that could be one is the scouts'
// lay-mines task, and scouts are never attack fleets.
func (t *robotoidTurn) passA() {
	v := t.v
	for i := len(v.Others) - 1; i >= 0; i-- {
		if v.Others[i].Owner != v.Player {
			t.enemies = append(t.enemies, v.Others[i])
		}
	}
	for i := 0; i < len(v.Fleets); i++ {
		f := &v.Fleets[i]
		if len(f.Waypoints) > 0 {
			w := f.Waypoints[0]
			if w.Target == engine.TargetWormhole && d2(f.Pos, w.Pos) > 200*200 {
				t.emit(f, cutRoute(f))
			}
		}
		if t.attackFleet(f) {
			t.attack = append([]int{f.ID}, t.attack...)
		}
		if v.holds(f, 0) && t.y >= 41 && len(f.Waypoints) == 0 {
			t.scout(f)
			continue
		}
		t.ownFleetA(f)
	}
}

func (t *robotoidTurn) ownFleetA(f *engine.Fleet) {
	v := t.v
	if v.transport(f) {
		var id int
		var ok bool
		if len(f.Waypoints) == 0 || f.Task.Kind != engine.TaskNone {
			id, ok = v.planetAt(f.Pos)
		} else if f.Waypoints[0].Target == engine.TargetPlanet {
			id, ok = f.Waypoints[0].ID, true
		}
		if ok && v.ownPlanet(id) == nil && f.Cargo.Colonists == 0 && (len(f.Waypoints) > 0 || f.Task.Kind != engine.TaskNone) {
			t.emit(f, cutRoute(f))
		}
	}
	if t.attackFleet(f) && v.holds(f, 2, 3, 4, 5, 6, 7) {
		if id, ok := t.headsTo(f); ok {
			if o := v.owner(id); o != engine.NoOwner && o != v.Player {
				t.targeted[id] = true
			}
		}
	}
	if (t.y > 20 || !v.holds(f, 0)) && len(f.Waypoints) == 0 && v.holds(f, 1) && t.y > 4 {
		t.colonizer(f)
	}
	if t.y <= 20 && v.holds(f, 0) {
		// Scrap (robotoid.md §4, AI.md §8 AI-3).
		t.emit(f, scrapOrder(f))
	}
}

// colonizer is pass A's rule for an idle fleet with colonizers.
func (t *robotoidTurn) colonizer(f *engine.Fleet) {
	v := t.v
	here, orbits := v.planetAt(f.Pos)
	if orbits && t.lvl >= Harder && f.Cargo.Colonists > 0 {
		if o := v.owner(here); o != engine.NoOwner && o != v.Player && v.PRT[o] != engine.PRTAlternateReality {
			task := engine.Task{Kind: engine.TaskTransport}
			task.Transport[engine.CargoColonists] = engine.Transport{Action: engine.UnloadAll}
			f.Task = task
			pos, _ := v.planetPos(here)
			if id, ok := v.nearestOwnStarbase(pos); ok {
				sp, _ := v.planetPos(id)
				t.emit(f, moveOrder(f, toPlanet(id, sp, 4, engine.Task{})))
			} else {
				t.emit(f, engine.WaypointOrder{Fleet: f.ID, Task: task, Waypoints: slices.Clone(f.Waypoints)})
			}
			return
		}
	}
	target, dd := t.nearestColonizable(f)
	var own *engine.Planet
	if orbits {
		own = v.ownPlanet(here)
	}
	if own != nil {
		if w, ok := v.preferWormhole(t.y, t.rng, f.Pos, target, dd); ok {
			t.load(f, own, 10)
			t.emit(f, moveOrder(f, toWormhole(w, v.idealWarp(f))))
			return
		}
	}
	if target < 0 {
		if orbits {
			// Scrap (robotoid.md §4, AI.md §8 AI-4).
			t.emit(f, scrapOrder(f))
		}
		return
	}
	if own != nil {
		t.load(f, own, 10)
	}
	pos, _ := v.planetPos(target)
	t.emit(f, moveOrder(f, toPlanet(target, pos, v.idealWarp(f), engine.Task{Kind: engine.TaskColonize})))
}

// nearestColonizable is AI.md §11's search for Robotoid: any planet
// unowned in its view, seen or not, that no other own fleet's waypoint 1
// targets with the colonize task, recomputed for each fleet.
func (t *robotoidTurn) nearestColonizable(f *engine.Fleet) (int, int) {
	v := t.v
	taken := map[int]bool{}
	for i := range v.Fleets {
		g := &v.Fleets[i]
		if g.ID != f.ID && len(g.Waypoints) > 0 && g.Waypoints[0].Target == engine.TargetPlanet && g.Waypoints[0].Task.Kind == engine.TaskColonize {
			taken[g.Waypoints[0].ID] = true
		}
	}
	best, bd := -1, 0
	for _, pp := range v.Universe {
		if taken[pp.ID] || v.owner(pp.ID) != engine.NoOwner {
			continue
		}
		if dd := d2(f.Pos, pp.Pos); best < 0 || dd < bd {
			best, bd = pp.ID, dd
		}
	}
	return best, bd
}

// scout is pass A's rule for scouts from year index 41 with one waypoint.
func (t *robotoidTurn) scout(f *engine.Fleet) {
	v := t.v
	moved := false
	if t.count(f, 0) > 6 && t.rng.Intn(5) == 0 {
		// ASSUMPTION A44: a pick of the planet it orbits means no move.
		here, _ := v.planetAt(f.Pos)
		if id, ok := t.randomNearby(f.Pos, 105, false); ok && id != here {
			pos, _ := v.planetPos(id)
			t.emit(f, moveOrder(f, toPlanet(id, pos, v.idealWarp(f), engine.Task{})))
			moved = true
		}
	}
	if !moved && f.Task.Kind == engine.TaskNone {
		t.emit(f, taskHere(f, layMines))
		return
	}
	if len(f.Waypoints) == 0 && v.holds(f, 1) && t.y > 4 {
		t.colonizer(f)
	}
}

// passB is §4 pass B: hub freighters.
func (t *robotoidTurn) passB() {
	v := t.v
	first := -1
	for _, id := range t.order {
		if p := v.ownPlanet(id); p != nil && p.HasStarbase {
			first = id
			break
		}
	}
	if first < 0 {
		return
	}
	assigned := t.assignHubs()
	for i := range v.Fleets {
		f := &v.Fleets[i]
		if !v.transport(f) || len(f.Waypoints) > 0 {
			continue
		}
		if f.Plan != 4 {
			if 4 < len(v.Self.Plans) {
				t.emit(f, engine.FleetPlanOrder{Fleet: f.ID, Plan: 4})
				f.Plan = 4
			} else {
				t.res.unsupported("fleet %d: battle plan 4 (the player has %d plans)", f.ID, len(v.Self.Plans))
			}
		}
		src, ok := assigned[f.ID]
		if !ok {
			src = first
		}
		t.hubFreighter(f, src)
	}
}

// assignHubs is AI.md §6 steps 3 and 4: every own transport fleet goes to
// the nearest hub with fewer than 8; then each hub with fewer than 4 takes
// one fleet from the first hub that has at least 2 more than it.
//
// ASSUMPTION A40: the fleet taken is the giving hub's last assigned.
func (t *robotoidTurn) assignHubs() map[int]int {
	v := t.v
	lists := make([][]int, len(t.hubs))
	for i := range v.Fleets {
		f := &v.Fleets[i]
		if !v.transport(f) {
			continue
		}
		best, bd := -1, 0
		for h, id := range t.hubs {
			if len(lists[h]) >= 8 {
				continue
			}
			pos, _ := v.planetPos(id)
			if dd := d2(f.Pos, pos); best < 0 || dd < bd {
				best, bd = h, dd
			}
		}
		if best >= 0 {
			lists[best] = append(lists[best], f.ID)
		}
	}
	for h := range lists {
		if len(lists[h]) >= 4 {
			continue
		}
		for g := range lists {
			if len(lists[g]) >= len(lists[h])+2 {
				n := len(lists[g]) - 1
				lists[h] = append(lists[h], lists[g][n])
				lists[g] = lists[g][:n]
				break
			}
		}
	}
	out := map[int]int{}
	for h, l := range lists {
		for _, id := range l {
			out[id] = t.hubs[h]
		}
	}
	return out
}

// hubFreighter is AI.md §11 "Hub freighters" for a fleet working for
// source planet src.
//
// Not built: unowned planets marked for pickup (the planner's memory is
// empty each year), Robotoid's small foreign colonies (their fixed values
// are not published), salvage (not in the view), and the colonist
// rules (ASSUMPTION A41: robotoid.md gives no amounts beyond ranges).
// Elegy's transport task can unload but not load, so a load at the target
// is reported as unsupported and the fleet moves with no task.
func (t *robotoidTurn) hubFreighter(f *engine.Fleet, src int) {
	v := t.v
	sp := v.ownPlanet(src)
	if sp == nil {
		return
	}
	// 1. The source's scarce mineral and mode.
	s := sp.Surface
	scarce, low, ref := engine.Ironium, s[engine.Ironium], -1
	for _, m := range []int{engine.Boranium, engine.Germanium} {
		if s[m] < low {
			ref, low, scarce = low, s[m], m
		}
	}
	mode := 0
	switch {
	case ref >= 0 && low < ref/4:
		mode = 2
	case ref >= 0 && low < ref/2:
		mode = 1
	}
	capacity := v.cargoCapacity(f)
	if capacity <= 0 {
		return
	}
	held := cargoMass(f.Cargo)
	pct := held * 100 / capacity
	firstDesign := -1
	if len(f.Stacks) > 0 {
		firstDesign = f.Stacks[0].Design
	}
	skip := map[int]bool{}
	for i := range v.Fleets {
		g := &v.Fleets[i]
		if g.ID != f.ID && len(g.Stacks) > 0 && g.Stacks[0].Design == firstDesign && len(g.Waypoints) > 0 && g.Waypoints[0].Target == engine.TargetPlanet {
			skip[g.Waypoints[0].ID] = true
		}
	}
	best, bs := -1, 0
	for _, pp := range v.Universe {
		if skip[pp.ID] {
			continue
		}
		dist := isqrt(d2(f.Pos, pp.Pos))
		tt := max(1, (dist+24)/25)
		score := 0
		switch p := v.ownPlanet(pp.ID); {
		case pp.ID == src:
			if f.Pos == pp.Pos || pct < 35 {
				continue
			}
			score = 20 * pct / tt
			if held >= capacity {
				score = 25000
			}
		case p != nil && !p.HasStarbase && !t.q.has(p, engine.ItemStarbase):
			amount := 0
			for m := range engine.NumMinerals {
				switch {
				case mode == 2 && m != scarce:
				case mode == 1 && m != scarce:
					amount += p.Surface[m] / 2
				default:
					amount += p.Surface[m]
				}
			}
			amount = min(amount, capacity-held)
			if amount < 10 {
				continue
			}
			score = amount * 100 / capacity * 100 / tt
		default:
			continue
		}
		if score > bs {
			best, bs = pp.ID, score
		}
	}
	if best < 0 {
		return
	}
	pos, _ := v.planetPos(best)
	task := engine.Task{}
	if best == src {
		task.Kind = engine.TaskTransport
		for _, c := range []int{engine.Ironium, engine.Boranium, engine.Germanium} {
			task.Transport[c] = engine.Transport{Action: engine.UnloadAll}
		}
	} else {
		t.res.unsupported("fleet %d: load task at planet %d (AI.md §11 hub freighters)", f.ID, best)
	}
	t.emit(f, moveOrder(f, toPlanet(best, pos, 4, task)))
}

func isqrt(x int) int {
	r := 0
	for (r+1)*(r+1) <= x {
		r++
	}
	return r
}

// passC is §4 pass C.
func (t *robotoidTurn) passC() {
	v := t.v
	for i := 0; i < len(v.Fleets); i++ {
		f := &v.Fleets[i]
		if t.obsoleteFleet(f) {
			continue
		}
		if v.holds(f, 2, 3, 4, 5, 6, 7, 8, 9, 10) {
			t.armada(f)
			continue
		}
		if !t.attackFleet(f) {
			continue
		}
		if len(f.Waypoints) > 0 && f.Waypoints[0].Target == engine.TargetFleet {
			continue
		}
		if t.joinUp(i) {
			continue
		}
		t.attackTarget(f, t.enemies)
	}
}

// obsoleteFleet is pass C's obsolete rule (not exercised in AIX). It
// reports whether the rule settled the fleet.
func (t *robotoidTurn) obsoleteFleet(f *engine.Fleet) bool {
	v := t.v
	if len(f.Stacks) == 0 {
		return false
	}
	for _, s := range f.Stacks {
		if !t.obsolete[v.shipSlot(s.Design)] {
			return false
		}
	}
	if id, orbits := v.planetAt(f.Pos); orbits {
		if p := v.ownPlanet(id); p != nil {
			if p.HasStarbase || t.rng.Intn(5) == 0 {
				// Scrap (robotoid.md §4 pass C).
				t.emit(f, scrapOrder(f))
			}
			return true
		}
	}
	if len(f.Waypoints) > 0 && f.Waypoints[0].Target == engine.TargetPlanet {
		return true
	}
	if id, ok := v.nearestOwnStarbase(f.Pos); ok {
		pos, _ := v.planetPos(id)
		t.emit(f, moveOrder(f, toPlanet(id, pos, 4, engine.Task{})))
		return true
	}
	return false
}

// joinUp is pass C's join-up (not exercised in AIX).
//
// ASSUMPTION A42: the Random(20) draw is made only when the count test
// says join and the fleet holds 20 or more D1415 ships.
func (t *robotoidTurn) joinUp(i int) bool {
	f := &t.v.Fleets[i]
	hi, lo := 70, 60
	if t.y >= 121 {
		hi, lo = 50, 40
	}
	c := t.guardFleets
	join := c > hi || (c > lo && t.rng.Intn(3) == 0)
	if join && t.d1415 >= 0 && t.count(f, t.d1415) >= 20 {
		join = t.rng.Intn(20) == 0
	}
	return join && t.joinBuddy(i, []int{14, 15}, 36, 72)
}

// troops are the ships of slots 9 and 10 (AI.md §11 "Armada").
func (t *robotoidTurn) troops(f *engine.Fleet) int { return t.count(f, 9, 10) }

// armada is AI.md §11 "Armada (invasion) fleets".
//
// ASSUMPTION A31: "the nearest object of interest" is the nearest planet.
// ASSUMPTION A32: a launch by chance loads colonists as a normal launch
// does. The invasion step is reported: Elegy's planner does not estimate
// the defense percentage yet.
func (t *robotoidTurn) armada(f *engine.Fleet) {
	v := t.v
	P, A := t.potency, t.size
	if len(f.Waypoints) > 0 {
		w := f.Waypoints[0]
		switch w.Target {
		case engine.TargetFleet:
			if d2(f.Pos, w.Pos) <= 250*250 {
				return
			}
		case engine.TargetPlanet:
			if o := v.owner(w.ID); (o != engine.NoOwner && o != v.Player) || !v.Seen[w.ID] {
				return
			}
			if p := v.ownPlanet(w.ID); p != nil && p.HasStarbase {
				return
			}
		}
	}
	here, orbits := v.planetAt(f.Pos)
	if !orbits {
		best, bd := -1, 0
		for _, pp := range v.Universe {
			if o := v.owner(pp.ID); o == engine.NoOwner || o == v.Player {
				continue
			}
			if dd := d2(f.Pos, pp.Pos); dd <= 150*150 && (best < 0 || dd < bd) {
				best, bd = pp.ID, dd
			}
		}
		if best < 0 {
			best = t.nearestPlanet(f.Pos)
		}
		if best >= 0 {
			pos, _ := v.planetPos(best)
			t.emit(f, moveOrder(f, toPlanet(best, pos, 4, engine.Task{})))
		}
		return
	}
	w := t.strength(f)
	chance := func(a, b int) bool {
		switch {
		case w > a || w >= 60:
			return t.rng.Intn(10) < 5
		case w > b:
			return t.rng.Intn(10) < 7
		case w > 120:
			return t.rng.Intn(10) < 7
		}
		return false
	}
	own := v.ownPlanet(here)
	if own != nil && own.HasStarbase {
		if w < P || t.troops(f) < A {
			if t.lvl < Harder || !chance(2*P, 3*P) {
				return
			}
		}
		pop := own.Population * 100
		div := 0
		switch {
		case pop > 300000:
			div = 10
		case pop > 200000:
			div = 15
		case pop > 100000:
			div = 20
		}
		if div > 0 {
			t.load(f, own, pop/div/100)
		}
		t.launch(f, here)
		return
	}
	if w < P/2 || t.troops(f) < min(3, A/2-1) {
		f.Task = engine.Task{}
		if t.lvl >= Harder && chance(2*P, 4*P) {
			t.launch(f, here)
			return
		}
		pos, _ := v.planetPos(here)
		if id, ok := v.nearestOwnStarbase(pos); ok {
			sp, _ := v.planetPos(id)
			t.emit(f, moveOrder(f, toPlanet(id, sp, 4, engine.Task{})))
		} else {
			t.emit(f, engine.WaypointOrder{Fleet: f.ID, Waypoints: slices.Clone(f.Waypoints)})
		}
		return
	}
	switch o := v.owner(here); {
	case o == engine.NoOwner:
		t.launch(f, here)
	case own != nil:
		if own.Population*100 > 100000 {
			t.load(f, own, own.Population/5)
		}
		t.launch(f, here)
	default:
		if f.Cargo.Colonists > 0 {
			t.res.unsupported("fleet %d: armada invasion test at planet %d", f.ID, here)
		}
	}
}

// launch is AI.md §11's launch target: the best-scoring threat-marked
// planet other than here (threat + distance bonus); a planet already
// chosen this turn is considered only with Random(4) == 0 and then
// outranks all others (LEGACY BUG candidate, reproduced). Ties: nearer.
// None: the nearest other player's fleet. Move order, no task, warp 4.
//
// ASSUMPTION A33: the chosen planets are drawn in planet-id order and the
// first that draws 0 wins.
func (t *robotoidTurn) launch(f *engine.Fleet, here int) {
	v := t.v
	best, bs, bd := -1, 0, 0
	for _, pp := range v.Universe {
		m, ok := t.threat[pp.ID]
		if !ok || pp.ID == here {
			continue
		}
		dd := d2(f.Pos, pp.Pos)
		if t.targeted[pp.ID] {
			if t.rng.Intn(4) == 0 {
				best = pp.ID
				break
			}
			continue
		}
		s := m + distanceBonus(dd)
		if best < 0 || s > bs || (s == bs && dd < bd) {
			best, bs, bd = pp.ID, s, dd
		}
	}
	if best >= 0 {
		pos, _ := v.planetPos(best)
		t.targeted[best] = true
		t.emit(f, moveOrder(f, toPlanet(best, pos, 4, engine.Task{})))
		return
	}
	if e, ok := t.nearestEnemy(f.Pos, -1); ok {
		t.emit(f, moveOrder(f, engine.Waypoint{Pos: e.Pos, Warp: 4, Target: engine.TargetFleet, ID: e.Fleet}))
	}
}

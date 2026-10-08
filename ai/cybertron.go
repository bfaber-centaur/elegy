package ai

import (
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// PlayCybertron plays one year for a Cybertron (PP) computer player:
// stars-elegy docs/ai/cybertron.md. Ship designs are CONFIRMED (AI-19),
// starbase queueing MEASURED (AI-20) and the fleet pass MEASURED (AI-21);
// the rest is BINARY-ONLY. v is changed as the turn plans.
//
// Steps (cybertron.md §1):
//  1. research and starbase designs (AI.md §4, §5);
//  2. ship designs (§2);
//  3. merges by design slot (AI.md §10);
//  4. the strength unit, the armada parameters and the age limit;
//  5. design ageing (§3);
//  6. from year index 81, splits (AI.md §10);
//  7. planet notes (§4.1), fleets (§5) and production (§4.2);
//  8. planet automation (AI.md §7) with Cybertron's starbase rule (§4.3),
//     then the queue fill.
//
// Under Elegy's clean per-player state the armada parameters are 0
// (cybertron.md §1 step 4; AI.md §1 "State leaking between computer
// players"), so every armada that is idle at a planet picks a target. The
// original's inherited values, which kept AIX's armadas at home (AI-18),
// would come only from a legacy ruleset switch, which is not implemented.
//
// Not yet run: packets (§6), the warp re-pick (AI.md §11 "Warp choice")
// and automation steps 4 and 5; docs/AI-STATUS.md lists them. Steps the
// engine cannot order (scrap, lay mines, invasion) go to
// Result.Unsupported.
func PlayCybertron(v *View, rng engine.Rand) Result {
	var res Result
	v.fleetOrder()
	y := v.Year - FirstYear
	researchAndStarbases(Cybertron, v, rng, &res)

	// ASSUMPTION A22: the own-planet shuffle (AI.md §2) is drawn at the
	// same point as Rototill's, after the starbase designs.
	var own []int
	for _, p := range v.Planets {
		own = append(own, p.ID)
	}
	order := ShufflePlanets(own, rng)

	t := &cyberTurn{turn: newTurn(v, rng, &res), taken: map[int]bool{}, inbound: map[int]int{}}
	t.sd.cybertronDesigns(v.Level, y, v.alive())
	t.sd.syncView(v)
	for _, g := range [][]int{{0}, {4, 5}, {14, 15}, {6, 7, 8, 9}, {10, 11, 12, 13}} {
		t.merge(g)
	}
	t.params()
	t.ageing()
	if y > 80 {
		t.splitOut(func(k int) bool { return t.obsolete[k] })
		for _, g := range [][]int{{0}, {1}, {2, 3}} {
			t.splitOut(func(k int) bool { return slices.Contains(g, k) })
		}
	}
	t.notes()
	t.passA()
	t.passB()
	budget := ResearchBudget(Cybertron, y, v.Self.Research.Levels)
	for _, id := range order {
		t.produce(v.ownPlanet(id), budget)
	}
	auto := &automation{v: v, pers: Cybertron, rng: rng, q: t.q, order: order, budget: budget}
	auto.run()
	if len(v.Planets) > 0 {
		res.unsupported("packets (cybertron.md §6) not implemented")
	}
	res.Orders = append(res.Orders, t.q.flush()...)
	return res
}

type cyberTurn struct {
	*turn

	// Parameters (§1 step 4): the strength unit, the armada potency and
	// size, and the design age limit.
	s, potency, size, limit int
	// Ageing results (§3): the newest Destroyer, guard and freighter
	// slots, freighter ships, the newer warship group (−1 for none), and
	// which warship groups aged out this turn.
	dd, gg, fr, nFr, gr int
	aged6, aged10       bool

	// Notes (§5): planets colony ships took this turn, and the
	// inbound-colonist notes as the freighter rule reads them.
	taken   map[int]bool
	inbound map[int]int

	// Pass A counts.
	guardFleets, scoutFleets, ddFleets, grFleets int
	attack                                       []int // own fleet ids
}

// params is §1 step 4.
func (t *cyberTurn) params() {
	y := t.y
	t.s = 1
	if y > 50 {
		t.s = (y-50)/10 + 1
	}
	if y > 100 {
		t.s += ((y - 100) / 10) * (y / 100)
	}
	// Clean per-player state: Cybertron's armada rule reads values only
	// another computer player sets, so they are 0 (AI.md §1).
	t.potency, t.size = 0, 0
	switch {
	case y < 120:
		t.limit = 50
	case y < 200:
		t.limit = 70
	case y < 400:
		t.limit = 100
	default:
		t.limit = 300
	}
}

// ageing is §3.
func (t *cyberTurn) ageing() {
	sd, alive := t.sd, t.v.alive()
	t.dd, _ = sd.ageGroup([]int{4, 5}, t.limit, alive, t.obsolete)
	delete(t.obsolete, t.dd)
	t.gg, _ = sd.ageGroup([]int{14, 15}, t.limit, alive, t.obsolete)
	delete(t.obsolete, t.dd) // LEGACY BUG reproduced: GG can stay obsolete
	t.fr, t.nFr = sd.ageGroup([]int{2, 3}, t.limit, alive, t.obsolete)
	delete(t.obsolete, t.fr)
	for _, g := range []int{6, 10} {
		if !sd.slots[g].present || sd.age(g) <= t.limit {
			continue
		}
		if g == 6 {
			t.aged6 = true
		} else {
			t.aged10 = true
		}
		kept := false
		for k := g + 3; k >= g; k-- {
			if !sd.slots[k].present {
				continue
			}
			switch {
			case alive[k] > 0:
				t.obsolete[k] = true
				kept = true
			case k == g && kept:
				t.obsolete[k] = true
			default:
				sd.delete(k)
			}
		}
	}
	// GR: the group whose first slot holds the newer design.
	// ASSUMPTION A20: on equal creation years, group 6–9.
	t.gr = -1
	for _, g := range []int{6, 10} {
		if sd.slots[g].present && (t.gr < 0 || sd.slots[g].created > sd.slots[t.gr].created) {
			t.gr = g
		}
	}
	sd.syncView(t.v)
}

// armedSlots are the slots pass A counts as armed (cybertron.md §5).
var armedSlots = []int{4, 5, 6, 7, 8, 9, 10, 11, 12, 13}

// passA is §5 pass A.
//
// ASSUMPTION A28: cybertron.md says armed fleets count as Destroyer
// fleets while only group 6–9 aged out this turn; in every other case
// they count as group fleets.
func (t *cyberTurn) passA() {
	v := t.v
	ddMode := t.aged6 && !t.aged10
	for i := range v.Fleets {
		f := &v.Fleets[i]
		if v.holds(f, 14, 15) {
			t.guardFleets++
			continue
		}
		if v.holds(f, 0) {
			t.scoutFleets++
		}
		if !v.holds(f, armedSlots...) {
			continue
		}
		t.attack = append(t.attack, f.ID)
		if ddMode {
			t.ddFleets++
			continue
		}
		t.grFleets++
		id, ok := t.headsTo(f)
		if ok {
			if o := v.owner(id); o != engine.NoOwner && o != v.Player {
				t.targeted[id] = true
			}
		}
	}
	for i := range v.Fleets {
		f := &v.Fleets[i]
		if len(f.Waypoints) > 0 && f.Waypoints[0].Target == engine.TargetPlanet {
			t.taken[f.Waypoints[0].ID] = true
		}
	}
}

// passB is §5 pass B: every own fleet, first rule that applies.
func (t *cyberTurn) passB() {
	v := t.v
	for i := 0; i < len(v.Fleets); i++ {
		f := &v.Fleets[i]
		if t.obsoleteFleet(f) {
			continue
		}
		if v.holds(f, 14, 15) {
			if _, orbits := v.planetAt(f.Pos); orbits {
				continue
			}
		}
		switch {
		case v.holds(f, armedSlots...) && !v.holds(f, 4):
			t.armada(f)
			if len(f.Waypoints) == 0 && t.rng.Intn(100) < 75 {
				group := []int{6, 7, 8, 9}
				if first := t.firstArmed(f); first > 9 {
					group = []int{10, 11, 12, 13}
				}
				t.joinBuddy(i, group, 100, 200)
			}
		case v.holds(f, 4):
			if t.s <= 2*t.count(f, 4) || len(f.Waypoints) > 0 {
				t.attackTarget(f, t.v.Others)
			}
			if len(f.Waypoints) == 0 && t.rng.Intn(100) < 75 {
				t.joinBuddy(i, []int{4, 5}, 100, 200)
			}
		case len(f.Waypoints) > 0:
		default:
			t.unarmed(i)
		}
	}
}

func (t *cyberTurn) firstArmed(f *engine.Fleet) int {
	first := 99
	for _, s := range f.Stacks {
		if k := t.v.shipSlot(s.Design); slices.Contains(armedSlots, k) && k < first {
			first = k
		}
	}
	return first
}

// obsoleteFleet is pass B rule 1 (not exercised in AIX). It reports
// whether the rule applied.
func (t *cyberTurn) obsoleteFleet(f *engine.Fleet) bool {
	v := t.v
	if len(f.Stacks) == 0 {
		return false
	}
	for _, s := range f.Stacks {
		k := v.shipSlot(s.Design)
		if k == 0 || k == 1 || !t.obsolete[k] {
			return false
		}
	}
	if len(f.Waypoints) > 0 {
		return true
	}
	if id, orbits := v.planetAt(f.Pos); orbits {
		if p := v.ownPlanet(id); p != nil {
			if p.HasStarbase || t.rng.Intn(5) == 0 {
				t.res.unsupported("fleet %d: scrap (cybertron.md §5 pass B rule 1)", f.ID)
			}
			return true
		}
	}
	if id, ok := v.nearestOwnStarbase(f.Pos); ok {
		pos, _ := v.planetPos(id)
		t.emit(f, moveOrder(f, toPlanet(id, pos, 4, engine.Task{})))
		return true
	}
	return false
}

// armada is §5 "Armada targeting". ASSUMPTION A23: with the clean-state
// parameters (0) no fleet is ever weak, so the stay, retreat and
// hard-level draw branches are unreachable and not built; they come with
// the legacy switch.
func (t *cyberTurn) armada(f *engine.Fleet) {
	v := t.v
	if len(f.Waypoints) > 0 {
		w := f.Waypoints[0]
		switch w.Target {
		case engine.TargetFleet:
			if d2(f.Pos, w.Pos) <= 250*250 {
				return
			}
		case engine.TargetPlanet:
			o := v.owner(w.ID)
			if (o != engine.NoOwner && o != v.Player) || !v.Seen[w.ID] {
				return
			}
			if p := v.ownPlanet(w.ID); p != nil && p.HasStarbase {
				return
			}
		}
	}
	here, orbits := v.planetAt(f.Pos)
	if !orbits {
		// ASSUMPTION A24: the move has no task and warp 4, as the other
		// armada moves.
		best, bd := -1, 0
		for _, pp := range v.Universe {
			if o := v.owner(pp.ID); o == engine.NoOwner || o == v.Player {
				continue
			}
			if dd := d2(f.Pos, pp.Pos); dd <= 450*450 && (best < 0 || dd < bd) {
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
	if o := v.owner(here); o != engine.NoOwner && o != v.Player {
		return // not weak at an owned planet: it stays
	}
	t.armadaTarget(f)
}

// armadaTarget is armada targeting step 5: the best score (threat mark
// plus distance bonus) above 1, ties to the closer; a targeted planet
// counts only when its own Random(4) == 0, drawn in planet-id order.
func (t *cyberTurn) armadaTarget(f *engine.Fleet) {
	v := t.v
	best, bs, bd := -1, 0, 0
	for _, pp := range v.Universe {
		m, ok := t.threat[pp.ID]
		if !ok {
			continue
		}
		if t.targeted[pp.ID] && t.rng.Intn(4) != 0 {
			continue
		}
		dd := d2(f.Pos, pp.Pos)
		s := m + distanceBonus(dd)
		if s > 1 && (best < 0 || s > bs || (s == bs && dd < bd)) {
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

// unarmed is pass B rule 5 for an idle unarmed fleet at index i.
func (t *cyberTurn) unarmed(i int) {
	v := t.v
	f := &v.Fleets[i]
	switch {
	case t.y < 6 && v.holds(f, 0):
		t.res.unsupported("fleet %d: scrap (cybertron.md §5 early scouts)", f.ID)
	case v.holds(f, 1):
		t.colonyShip(f)
	case t.y > 40 && v.holds(f, 0) && !v.holds(f, 2, 3):
		t.minelayer(i)
	case v.holds(f, 2, 3):
		if f.Plan != 4 {
			if 4 < len(v.Self.Plans) {
				t.emit(f, engine.FleetPlanOrder{Fleet: f.ID, Plan: 4})
				f.Plan = 4
			} else {
				t.res.unsupported("fleet %d: battle plan 4 (the player has %d plans)", f.ID, len(v.Self.Plans))
			}
		}
		t.freighter(f)
	}
}

// colonyShip is pass B's colony-ship rule: the nearest colonizable planet
// (AI.md §11, the non-Robotoid form; the target counts as taken for later
// colony ships this turn, as cybertron.md §5 says); none, no order.
// Otherwise at an own planet it loads colonists up to 250 kT, clamped to
// its hold, then gets the colonize order.
func (t *cyberTurn) colonyShip(f *engine.Fleet) {
	v := t.v
	target, dd := t.nearestColonizable(f.Pos, true)
	here, orbits := v.planetAt(f.Pos)
	var own *engine.Planet
	if orbits {
		own = v.ownPlanet(here)
	}
	if own != nil {
		if w, ok := v.preferWormhole(t.y, t.rng, f.Pos, target, dd); ok {
			t.emit(f, moveOrder(f, engine.Waypoint{Pos: w.Pos, Warp: v.idealWarp(f), Target: engine.TargetWormhole, ID: w.End}))
			return
		}
	}
	if target < 0 {
		return
	}
	if own != nil {
		t.load(f, own, 250-f.Cargo.Colonists)
	}
	t.taken[target] = true
	pos, _ := v.planetPos(target)
	t.emit(f, moveOrder(f, toPlanet(target, pos, v.idealWarp(f), engine.Task{Kind: engine.TaskColonize})))
}

// nearestColonizable is AI.md §11's search for Cybertron: planets unowned
// in its view, in its view, with habitability after terraforming not
// negative, not taken when skipTaken; −1 when none.
func (t *cyberTurn) nearestColonizable(from engine.Point, skipTaken bool) (int, int) {
	v := t.v
	best, bd := -1, 0
	for _, pp := range v.Universe {
		r, known := v.Known[pp.ID]
		if !known || (skipTaken && t.taken[pp.ID]) || v.owner(pp.ID) != engine.NoOwner {
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

// minelayer is pass B's slot-0 rule (not exercised in AIX). Elegy cannot
// lay mines yet, so the task is reported; a move to a random nearby planet
// is still given, with no task.
func (t *cyberTurn) minelayer(i int) {
	v := t.v
	f := &v.Fleets[i]
	if t.scoutFleets > 55 || (t.scoutFleets > 40 && t.rng.Intn(3) != 0) {
		if t.joinBuddy(i, []int{0}, 72, 108) {
			return
		}
	}
	if t.count(f, 0) > 6 && t.rng.Intn(5) == 0 {
		if id, ok := t.randomNearby(f.Pos, 105, true); ok {
			pos, _ := v.planetPos(id)
			t.emit(f, moveOrder(f, toPlanet(id, pos, 4, engine.Task{})))
			t.res.unsupported("fleet %d: lay-mines task at the new waypoint", f.ID)
			return
		}
	}
	t.res.unsupported("fleet %d: lay mines here (cybertron.md §5 slot-0 fleets)", f.ID)
}

// freighter is §5's freighter rule.
func (t *cyberTurn) freighter(f *engine.Fleet) {
	v := t.v
	here, orbits := v.planetAt(f.Pos)
	if !orbits {
		if id := t.nearestPlanet(f.Pos); id >= 0 {
			pos, _ := v.planetPos(id)
			t.emit(f, moveOrder(f, toPlanet(id, pos, 4, engine.Task{})))
		}
		return
	}
	var dest int
	if p := v.ownPlanet(here); p != nil {
		if p.Population*100 <= 200000 {
			t.load(f, p, -f.Cargo.Colonists)
			dest = t.pickup(f.Pos)
		} else {
			loaded := t.load(f, p, 1000)
			dest = t.dropOff(f.Pos)
			if dest < 0 {
				t.load(f, p, -loaded)
				return
			}
		}
	} else {
		o := v.owner(here)
		switch {
		case o == engine.NoOwner || v.PRT[o] == engine.PRTAlternateReality || v.Known[here].Starbase:
			if f.Cargo.Colonists > 0 {
				dest = t.dropOff(f.Pos)
			} else {
				dest = t.pickup(f.Pos)
			}
		default:
			t.res.unsupported("fleet %d: freighter invasion (cybertron.md §5, not exercised)", f.ID)
			return
		}
	}
	if dest < 0 {
		return
	}
	pos, _ := v.planetPos(dest)
	t.emit(f, moveOrder(f, toPlanet(dest, pos, 4, engine.Task{})))
	t.note(dest)
}

// note records an inbound-colonist note as the original stores it: notes
// are lost, except that a note about planet 0 lands on the last planet
// (LEGACY BUG, MEASURED AI-21, reproduced).
func (t *cyberTurn) note(id int) {
	if id == 0 && len(t.v.Universe) > 0 {
		t.inbound[t.v.Universe[len(t.v.Universe)-1].ID]++
	}
}

// pickup is the nearest own planet with a starbase and over 220,000
// colonists (it can be the planet itself); −1 for none.
func (t *cyberTurn) pickup(from engine.Point) int {
	best, bd := -1, 0
	for _, p := range t.v.Planets {
		if !p.HasStarbase || p.Population*100 <= 220000 {
			continue
		}
		if dd := d2(from, p.Pos); best < 0 || dd < bd {
			best, bd = p.ID, dd
		}
	}
	return best
}

// dropOff is the nearest own planet within 400 ly with fewer than 20,000
// colonists, else one within 400 ly where population + 21,000 × inbound
// count is under 100,000; −1 for none.
func (t *cyberTurn) dropOff(from engine.Point) int {
	for _, ok := range []func(p *engine.Planet) bool{
		func(p *engine.Planet) bool { return p.Population*100 < 20000 },
		func(p *engine.Planet) bool { return p.Population*100+21000*t.inbound[p.ID] < 100000 },
	} {
		best, bd := -1, 0
		for i := range t.v.Planets {
			p := &t.v.Planets[i]
			dd := d2(from, p.Pos)
			if dd > 400*400 || !ok(p) {
				continue
			}
			if best < 0 || dd < bd {
				best, bd = p.ID, dd
			}
		}
		if best >= 0 {
			return best
		}
	}
	return -1
}

// value is a planet's value for the race now and after the terraforming
// it can reach (ESTIMATES.md "Value and optimal value").
func (t *cyberTurn) value(p *engine.Planet) (v, o int) {
	v = engine.Habitability(t.v.Self.Race, p.Env)
	o, _ = t.v.terraformedHab(engine.PlanetReport{Level: engine.ReportOwn, Env: p.Env})
	return v, o
}

// produce is §4.2 for one own planet.
func (t *cyberTurn) produce(p *engine.Planet, budget int) {
	v := t.v
	avail := v.available(p, budget)
	cost := v.queueCost(t.q.get(p))
	if avail.short(cost) {
		return
	}
	val, opt := t.value(p)
	r := avail.Resources - cost.Resources
	g := v.Self.Race.GrowthRate * p.Population
	add := func(it engine.QueueItem) { t.q.add(p, it, false) }
	switch {
	case val < 10:
		add(engine.QueueItem{Kind: engine.ItemTerraform, Count: r/70 + 1})
	case val < opt && r > 70:
		add(engine.QueueItem{Kind: engine.ItemTerraform, Count: 1})
	}
	if sb, ok := v.design(p.StarbaseDesign); p.HasStarbase && ok && sb.Hull.Name == orbitalFort {
		return
	}
	alive := v.alive()
	// Colony ship.
	if alive[1] < 40 && t.colonizer(p) {
		add(engine.QueueItem{Kind: engine.ItemShip, Slot: 1, Count: 1})
		if g > 15000 && t.y < 100 {
			add(engine.QueueItem{Kind: engine.ItemShip, Slot: 1, Count: 1})
		}
	}
	// Freighter.
	if p.Population*100 > 200000 && t.sd.slots[2].present && t.fr >= 0 && t.nFr < 50 && t.takesColonists(p) {
		add(engine.QueueItem{Kind: engine.ItemShip, Slot: t.fr, Count: 1})
	}
	// Frigates.
	if d, ok := v.ship(0); ok && d.Design.Hull.Name == "Frigate" && t.rng.Intn(4) == 0 && alive[0] < 10000 {
		c := 0
		for i := range v.Fleets {
			if v.Fleets[i].Pos == p.Pos {
				c = t.count(&v.Fleets[i], 0)
				break
			}
		}
		switch {
		case c < 10:
			if t.rng.Intn(2*c+1) == 0 {
				add(engine.QueueItem{Kind: engine.ItemShip, Slot: 0, Count: 4})
			}
		case c < 17:
			if t.rng.Intn(10) == 0 && t.rng.Intn(2*c+1) == 0 {
				add(engine.QueueItem{Kind: engine.ItemShip, Slot: 0, Count: 4})
			}
		}
	}
	t.queueAttackFleet(p, add)
}

// colonizer is §4.2's colonizer test.
//
// ASSUMPTION A26: "Random(2)" passes when it draws 0.
func (t *cyberTurn) colonizer(p *engine.Planet) bool {
	if t.y < 60 {
		return true
	}
	id, dd := t.nearestColonizable(p.Pos, false)
	switch {
	case id < 0 || dd > 353*353:
		return false
	case dd > 250*250:
		return t.rng.Intn(2) == 0
	}
	return true
}

// takesColonists reports an own planet within 400 ly of p where
// population + 21,000 × inbound count is under 100,000.
func (t *cyberTurn) takesColonists(p *engine.Planet) bool {
	for _, q := range t.v.Planets {
		if d2(p.Pos, q.Pos) <= 400*400 && q.Population*100+21000*t.inbound[q.ID] < 100000 {
			return true
		}
	}
	return false
}

// queueAttackFleet is §4.2 step 2.4.
//
// ASSUMPTION A27: the guard draw is made only when GG holds a design and
// fewer than 40 guard fleets exist; the third and fourth group draws are
// made whether or not those slots hold designs.
func (t *cyberTurn) queueAttackFleet(p *engine.Planet, add func(engine.QueueItem)) {
	v := t.v
	guard := -1
	if t.gg >= 0 && t.sd.slots[t.gg].present && t.guardFleets < 40 {
		near := false
		for _, pp := range v.Universe {
			if o := v.owner(pp.ID); o != engine.NoOwner && o != v.Player && d2(p.Pos, pp.Pos) <= 300*300 {
				near = true
				break
			}
		}
		chance := 10
		if near {
			chance = 50
		}
		if t.rng.Intn(100) < chance {
			guard = t.gg
		}
	}
	dd, gr := t.dd, t.gr
	if t.ddFleets > 120 {
		dd = -1
	}
	if t.grFleets > 250 {
		gr = -1
	}
	r1, r2 := t.rng.Intn(100), t.rng.Intn(100)
	col := engine.NewColony(p, &v.Self)
	mineRoom := col.OperableMines(p.Population) - p.Mines - t.q.count(p, engine.ItemMine)
	factRoom := col.OperableFactories(p.Population) - p.Factories - t.q.count(p, engine.ItemFactory)
	need := 60
	if mineRoom < 100 || factRoom < 100 {
		need = 90
	}
	if r2 < need {
		return
	}
	ship := func(k, n int) {
		if k >= 0 && t.sd.slots[k].present {
			add(engine.QueueItem{Kind: engine.ItemShip, Slot: k, Count: n})
		}
	}
	switch {
	case gr >= 0 && r1 > 50:
		ship(gr, 2)
		ship(gr+1, 2)
		if t.rng.Intn(100) < 75 {
			ship(gr+2, 1)
		}
		if t.rng.Intn(100) < 50 {
			ship(gr+3, 1)
		}
	case guard >= 0 && r1 > 25:
		ship(guard, 1)
	default:
		ship(dd, 1)
	}
}

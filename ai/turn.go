package ai

import (
	"slices"

	"github.com/bfaber-centaur/elegy/engine"
)

// turn is the state every personality's turn shares: the view, the
// random stream, the result, its design slots and queues, and the marks
// and notes the shared rules of AI.md §10 and §11 read and write.
type turn struct {
	v   *View
	rng engine.Rand
	res *Result
	y   int
	lvl Level
	sd  *shipDesigns
	q   *queues

	// obsolete marks design slots ageing marked this turn (AI.md §10).
	obsolete map[int]bool
	// threat is the AI threat mark of other players' planets, targeted
	// the planets an armada or group fleet heads to (AI.md §11).
	threat   map[int]int
	targeted map[int]bool

	// nextSplit is the last id given to a fleet split off this turn,
	// counting down from −1. The split order names the new fleet with it
	// (engine.SplitOrder.NewFleet, ASSUMPTION L28), so later orders in the
	// file can use it.
	nextSplit int
}

func newTurn(v *View, rng engine.Rand, res *Result) *turn {
	return &turn{v: v, rng: rng, res: res, y: v.Year - FirstYear, lvl: v.Level, q: &queues{v: v},
		sd: newShipDesigns(v, rng, res), obsolete: map[int]bool{}, threat: map[int]int{}, targeted: map[int]bool{}}
}

// emit appends an order for a fleet. A fleet split off this turn has the
// negative id its split order named (turn.nextSplit).
func (t *turn) emit(f *engine.Fleet, o engine.Order) {
	t.res.Orders = append(t.res.Orders, o)
}

// count is a fleet's ships of the slots.
func (t *turn) count(f *engine.Fleet, slots ...int) int {
	n := 0
	for _, s := range f.Stacks {
		if slices.Contains(slots, t.v.shipSlot(s.Design)) {
			n += s.Count
		}
	}
	return n
}

func ships(f *engine.Fleet) int {
	n := 0
	for _, s := range f.Stacks {
		n += s.Count
	}
	return n
}

// merge is AI.md §10 "Merging" for a set of slots: walking own fleets in
// fleet order, every fleet with ships of those slots merges into the
// first such fleet at the same place. Up to 32 places are tracked per
// pass; further places get another pass. A merge closes the gap in the
// fleet list, so the walk skips the fleet that followed the merged one
// (LEGACY BUG, BINARY-ONLY, reproduced as part of the faithful planner).
// The exception for fleets at the maximum mining rate is not built:
// Cybertron's class lists hold no mining hull.
//
// ASSUMPTION A19: a later pass skips the places an earlier pass tracked.
func (t *turn) merge(slots []int) {
	v := t.v
	done := map[engine.Point]bool{}
	for {
		first := map[engine.Point]int{}
		overflow := false
		for i := 0; i < len(v.Fleets); i++ {
			f := &v.Fleets[i]
			if done[f.Pos] || !v.holds(f, slots...) {
				continue
			}
			j, ok := first[f.Pos]
			if !ok {
				if len(first) >= 32 {
					overflow = true
					continue
				}
				first[f.Pos] = i
				continue
			}
			into := &v.Fleets[j]
			t.res.Orders = append(t.res.Orders, engine.MergeOrder{Into: into.ID, From: []int{f.ID}})
			absorb(into, f)
			v.Fleets = slices.Delete(v.Fleets, i, i+1)
			// The fleet now at i is skipped by the loop's i++.
		}
		for p := range first {
			done[p] = true
		}
		if !overflow {
			return
		}
	}
}

// absorb adds fleet b's ships, cargo and fuel to fleet a in the planner's
// picture.
func absorb(a, b *engine.Fleet) {
	for _, s := range b.Stacks {
		if i := slices.IndexFunc(a.Stacks, func(x engine.Stack) bool { return x.Design == s.Design }); i >= 0 {
			a.Stacks[i].Count += s.Count
		} else {
			a.Stacks = append(a.Stacks, s)
		}
	}
	a.Fuel += b.Fuel
	a.Cargo.Colonists += b.Cargo.Colonists
	for m := range a.Cargo.Minerals {
		a.Cargo.Minerals[m] += b.Cargo.Minerals[m]
	}
}

// splitOut is AI.md §10 "Splitting" with a slot test: while the player
// owns at most 500 fleets, the first own fleet holding ships of both
// tested and other slots is split, the tested ships moving to a new
// fleet at the same place with the same waypoints and battle plan, and
// the scan restarts. The split order names the new fleet (engine
// SplitOrder.NewFleet), so later orders this turn can give it orders,
// split it again or have others join it.
func (t *turn) splitOut(test func(slot int) bool) {
	v := t.v
	for len(v.Fleets) <= 500 {
		i := slices.IndexFunc(v.Fleets, func(f engine.Fleet) bool {
			in, out := false, false
			for _, s := range f.Stacks {
				if s.Count == 0 {
					continue
				}
				if test(v.shipSlot(s.Design)) {
					in = true
				} else {
					out = true
				}
			}
			return in && out
		})
		if i < 0 {
			return
		}
		src := &v.Fleets[i]
		var moved []engine.Stack
		var kept []engine.Stack
		for _, s := range src.Stacks {
			if test(v.shipSlot(s.Design)) {
				moved = append(moved, engine.Stack{Design: s.Design, Count: s.Count})
			} else {
				kept = append(kept, s)
			}
		}
		t.nextSplit--
		t.res.Orders = append(t.res.Orders, engine.SplitOrder{Fleet: src.ID, Ships: moved, NewFleet: t.nextSplit})
		nf := engine.Fleet{ID: t.nextSplit, Number: t.freeNumber(), Owner: src.Owner, Pos: src.Pos, Stacks: moved,
			Waypoints: slices.Clone(src.Waypoints), Plan: src.Plan, Task: src.Task, Repeat: src.Repeat}
		capOf := func(st []engine.Stack) int {
			n := 0
			for _, s := range st {
				if d, ok := v.design(s.Design); ok {
					n += s.Count * d.CargoCapacity
				}
			}
			return n
		}
		if all := capOf(src.Stacks); all > 0 {
			share := capOf(moved)
			nf.Cargo.Colonists = src.Cargo.Colonists * share / all
			src.Cargo.Colonists -= nf.Cargo.Colonists
			for m := range nf.Cargo.Minerals {
				nf.Cargo.Minerals[m] = src.Cargo.Minerals[m] * share / all
				src.Cargo.Minerals[m] -= nf.Cargo.Minerals[m]
			}
		}
		src.Stacks = kept
		v.Fleets = append(v.Fleets, nf)
		v.fleetOrder()
	}
}

func (t *turn) freeNumber() int {
	used := map[int]bool{}
	for _, f := range t.v.Fleets {
		used[f.Number] = true
	}
	n := 1
	for used[n] {
		n++
	}
	return n
}

// notes is the threat marks (cybertron.md §4.1, robotoid.md §1): every
// planet owned by another player gets min(6, e/250 + 1), plus 1 with a
// starbase, where e is the report's population estimate in units of 400
// colonists (cybertron.md §4.1; a planet with no estimate counts as 0).
//
// ASSUMPTION A21: robotoid.md §1 does not name the unit; Robotoid uses
// Cybertron's. The low-mineral notes are kept by Cybertron's packets.
func (t *turn) notes() {
	v := t.v
	for _, pp := range v.Universe {
		o := v.owner(pp.ID)
		if o == engine.NoOwner || o == v.Player {
			continue
		}
		r := v.Known[pp.ID]
		m := min(6, r.PopEstimate/400/250+1)
		if r.Starbase {
			m++
		}
		t.threat[pp.ID] = m
	}
}

// joinBuddy is AI.md §11 "Join a buddy" for the fleet at index i: the
// nearest own fleet earlier in fleet order with ships of the slots; within
// r1 it joins, within r2 when Random(2) != 0. Joining clears the waypoint-0
// task and moves to that fleet at warp 6.
func (t *turn) joinBuddy(i int, slots []int, r1, r2 int) bool {
	v := t.v
	f := &v.Fleets[i]
	best, bd := -1, 0
	for j := range i {
		if !v.holds(&v.Fleets[j], slots...) {
			continue
		}
		if dd := d2(f.Pos, v.Fleets[j].Pos); best < 0 || dd < bd {
			best, bd = j, dd
		}
	}
	if best < 0 || bd > r2*r2 || (bd > r1*r1 && t.rng.Intn(2) == 0) {
		return false
	}
	b := &v.Fleets[best]
	f.Task = engine.Task{}
	t.emit(f, moveOrder(f, engine.Waypoint{Pos: b.Pos, Warp: 6, Target: engine.TargetFleet, ID: b.ID}))
	return true
}

// distanceBonus is the armada score's distance bonus.
func distanceBonus(dd int) int {
	for _, b := range []struct{ r, s int }{{50, 7}, {100, 5}, {150, 4}, {200, 3}, {300, 2}, {500, 1}} {
		if dd < b.r*b.r {
			return b.s
		}
	}
	return 0
}

func (t *turn) nearestPlanet(from engine.Point) int {
	best, bd := -1, 0
	for _, pp := range t.v.Universe {
		if dd := d2(from, pp.Pos); best < 0 || dd < bd {
			best, bd = pp.ID, dd
		}
	}
	return best
}

func (t *turn) nearestEnemy(from engine.Point, limit int) (engine.FleetSighting, bool) {
	var best engine.FleetSighting
	bd, found := 0, false
	for _, e := range t.v.Others {
		if e.Owner == t.v.Player {
			continue
		}
		dd := d2(from, e.Pos)
		if limit >= 0 && dd > limit*limit {
			continue
		}
		if !found || dd < bd {
			best, bd, found = e, dd, true
		}
	}
	return best, found
}

// attackTarget is AI.md §11 "Attack target", with "computer players form
// alliances" off.
//
// ASSUMPTION A25: step 3's "can reach an own starbase" is not tested (the
// waypoint fuel estimate is not exported yet), and with the planner's
// empty memory no planet is marked visited.
func (t *turn) attackTarget(f *engine.Fleet, enemies []engine.FleetSighting) {
	v := t.v
	var target engine.Waypoint
	found := false
	var best engine.FleetSighting
	bd, have := 0, false
	for _, e := range enemies {
		if e.Owner == v.Player {
			continue
		}
		n := 0
		for j := range v.Fleets {
			g := &v.Fleets[j]
			if g.ID != f.ID && len(g.Waypoints) > 0 && g.Waypoints[0].Target == engine.TargetFleet && g.Waypoints[0].ID == e.Fleet {
				n += ships(g)
			}
		}
		if n > 0 && t.rng.Intn(3) == 0 {
			continue
		}
		if 5*n > ships(f) && t.rng.Intn(15) == 0 {
			continue
		}
		dd := d2(f.Pos, e.Pos)
		if dd <= 1000*1000 && (!have || dd < bd) {
			best, bd, have = e, dd, true
		}
	}
	switch {
	case have && bd <= 180*180:
		target, found = engine.Waypoint{Pos: best.Pos, Warp: 4, Target: engine.TargetFleet, ID: best.Fleet}, true
	case have:
		if 2*f.Fuel < t.fuelCapacity(f) {
			if id, ok := v.nearestOwnStarbase(f.Pos); ok {
				pos, _ := v.planetPos(id)
				t.emit(f, moveOrder(f, toPlanet(id, pos, 4, engine.Task{})))
				return
			}
		}
		pid, pd := -1, 0
		for _, pp := range v.Universe {
			if t.ownTargets(f, pp.ID) {
				continue
			}
			if dd := d2(best.Pos, pp.Pos); pid < 0 || dd < pd {
				pid, pd = pp.ID, dd
			}
		}
		if here, orbits := v.planetAt(f.Pos); pid >= 0 && !(orbits && pid == here) {
			pos, _ := v.planetPos(pid)
			target, found = toPlanet(pid, pos, 4, engine.Task{}), true
		}
	default:
		pid, pd := -1, 0
		for _, pp := range v.Universe {
			if o := v.owner(pp.ID); o == engine.NoOwner || o == v.Player {
				continue
			}
			if dd := d2(f.Pos, pp.Pos); pid < 0 || dd < pd {
				pid, pd = pp.ID, dd
			}
		}
		if pid < 0 {
			pid = t.nearestPlanet(f.Pos)
		}
		if pid < 0 && len(v.Universe) > 0 {
			pid = v.Universe[t.rng.Intn(len(v.Universe))].ID
		}
		if pid >= 0 {
			pos, _ := v.planetPos(pid)
			target, found = toPlanet(pid, pos, 4, engine.Task{}), true
		}
	}
	if !found {
		return
	}
	if target.Target == engine.TargetPlanet && len(f.Waypoints) > 0 && f.Waypoints[0].Target == engine.TargetPlanet {
		return
	}
	t.emit(f, moveOrder(f, target))
}

// ownTargets reports whether an own fleet other than f heads to the
// planet.
func (t *turn) ownTargets(f *engine.Fleet, id int) bool {
	for j := range t.v.Fleets {
		g := &t.v.Fleets[j]
		if g.ID != f.ID && len(g.Waypoints) > 0 && g.Waypoints[0].Target == engine.TargetPlanet && g.Waypoints[0].ID == id {
			return true
		}
	}
	return false
}

func (t *turn) fuelCapacity(f *engine.Fleet) int {
	n := 0
	for _, s := range f.Stacks {
		if d, ok := t.v.design(s.Design); ok {
			n += s.Count * d.FuelCapacity
		}
	}
	return n
}

// load moves up to n kT of colonists between an own planet and the fleet
// (negative n unloads), limited by what the source holds and the target
// can take, and changes both in the planner's picture (AI.md §11
// "Supplies"). It returns the amount moved into the fleet.
func (t *turn) load(f *engine.Fleet, p *engine.Planet, n int) int {
	if n > 0 {
		n = min(n, p.Population, t.v.cargoCapacity(f)-cargoMass(f.Cargo))
	} else {
		n = max(n, -f.Cargo.Colonists)
	}
	if n == 0 {
		return 0
	}
	p.Population -= n
	f.Cargo.Colonists += n
	var amounts [engine.NumCargo + 1]int
	amounts[engine.CargoColonists] = n
	t.emit(f, engine.CargoOrder{Fleet: f.ID, Target: engine.TargetPlanet, ID: p.ID, Amounts: amounts})
	return n
}

// randomNearby is AI.md §11 "Random nearby planet": a uniform reservoir
// pick among planets within r (Random(k) for the k-th candidate, the
// first included); with avoid, a pick with a starbase is redrawn up to
// twice, the last pick standing.
func (t *turn) randomNearby(from engine.Point, r int, avoid bool) (int, bool) {
	v := t.v
	pick := func() int {
		got, k := -1, 0
		for _, pp := range v.Universe {
			if d2(from, pp.Pos) > r*r {
				continue
			}
			k++
			if t.rng.Intn(k) == 0 {
				got = pp.ID
			}
		}
		return got
	}
	id := pick()
	for try := 0; avoid && try < 2 && id >= 0 && v.Known[id].Starbase; try++ {
		id = pick()
	}
	return id, id >= 0
}

// headsTo is the planet a fleet heads to, or orbits when idle.
func (t *turn) headsTo(f *engine.Fleet) (int, bool) {
	if len(f.Waypoints) > 0 {
		w := f.Waypoints[0]
		return w.ID, w.Target == engine.TargetPlanet
	}
	return t.v.planetAt(f.Pos)
}

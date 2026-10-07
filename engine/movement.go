package engine

import (
	"math"
	"sort"
)

// Point is a position in light-years.
type Point struct{ X, Y int }

func distance(a, b Point) float64 {
	return math.Hypot(float64(b.X-a.X), float64(b.Y-a.Y))
}

// Engine is an engine part as far as movement and battle speed need it.
type Engine struct {
	Name string
	// Fuel is the fuel factor f(w) for warp 0..10 (index 0 unused).
	Fuel [11]int
	// BattleWarp10 marks the engines whose battle speed uses warp 10
	// (Interspace-10, Enigma Pulsar, Trans-Star 10, Trans-Galactic Mizer
	// Scoop, Galaxy Scoop; COMBAT.md "Token values").
	BattleWarp10 bool
	// EnigmaPulsar counts toward the half-step speed bonus.
	EnigmaPulsar bool
}

// Design is a ship or starbase design. Movement reads Mass, Engine,
// Engines, CargoCapacity and FuelCapacity; battles also read Hull, Slots
// and Cost.
type Design struct {
	Name          string
	Mass          int // kT per ship
	Engine        Engine
	Engines       int // engines per ship
	CargoCapacity int // kT per ship
	FuelCapacity  int // mg per ship

	Hull  Hull
	Slots []Slot
	// Cost is the current cost of one ship to its owner, with the
	// owner's discounts and miniaturization already applied (KERNEL.md
	// and COMBAT.md give no formula for those).
	Cost Cost
}

// Stack is a number of ships of one design (an index into Game.Designs).
// Damage is the stack's battle damage.
type Stack struct {
	Design int
	Count  int
	Damage Damage
}

// Damage on a stack (COMBAT.md "Conventions"): Pct of its ships are damaged,
// each by Units/500 of the design's armor.
type Damage struct {
	Pct, Units int
}

// Cargo is carried minerals and colonists, in kT.
type Cargo struct {
	Minerals  Minerals
	Colonists int
}

func (c Cargo) mass() int {
	return c.Minerals[Ironium] + c.Minerals[Boranium] + c.Minerals[Germanium] + c.Colonists
}

// TargetKind is what a waypoint points at.
type TargetKind int

const (
	TargetSpace TargetKind = iota
	TargetPlanet
	TargetFleet
)

// Waypoint is a fleet destination. Fleet.Waypoints[0] is the current
// destination; the fleet's own position is not stored as a waypoint.
type Waypoint struct {
	Pos    Point
	Warp   int
	Target TargetKind
	ID     int // planet or fleet id for TargetPlanet / TargetFleet
}

type Fleet struct {
	ID        int
	Owner     int
	Pos       Point
	Stacks    []Stack
	Fuel      int // mg
	Cargo     Cargo
	Waypoints []Waypoint
	Plan      int // battle plan index in the owner's Plans
}

// fleetDesign is one design group inside a fleet, with its share of cargo.
type fleetDesign struct {
	d     Design
	n     int
	cargo int
}

// groups returns the fleet's ships grouped by design, ordered by increasing
// f(w), with the fleet's cargo assigned to them in that order up to each
// group's capacity. Ties in f(w) keep the fleet's own design order
// (KERNEL.md "Fuel cost", BINARY-ONLY).
func (g *Game) groups(f *Fleet, warp int) []fleetDesign {
	var out []fleetDesign
	index := map[int]int{}
	for _, s := range f.Stacks {
		if i, ok := index[s.Design]; ok {
			out[i].n += s.Count
			continue
		}
		index[s.Design] = len(out)
		out = append(out, fleetDesign{d: g.Designs[s.Design], n: s.Count})
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].d.Engine.Fuel[warp] < out[b].d.Engine.Fuel[warp]
	})
	left := f.Cargo.mass()
	for i := range out {
		take := min(left, out[i].n*out[i].d.CargoCapacity)
		out[i].cargo = take
		left -= take
	}
	return out
}

// FuelCost is the fuel in mg a fleet uses to move dist light-years at warp.
// KERNEL.md "Fuel cost", CONFIRMED (FM-001..003).
func (g *Game) FuelCost(f *Fleet, warp, dist int) int {
	tenths := 0
	for _, gr := range g.groups(f, warp) {
		tenths += gr.d.Engine.Fuel[warp] * dist * (gr.n*gr.d.Mass + gr.cargo) / 2000
	}
	return (tenths + 9) / 10
}

// fuelRange is R, the distance the fleet's fuel pays for at warp, and
// whether it is unlimited.
func (g *Game) fuelRange(f *Fleet, warp int) (r int, unlimited bool) {
	sum := 0
	for _, gr := range g.groups(f, warp) {
		sum += gr.d.Engine.Fuel[warp] * 1000 * (gr.n*gr.d.Mass + gr.cargo) / 2000
	}
	c1000 := sum / 10
	switch {
	case c1000 == 0:
		return 0, true
	case c1000 > 100_000:
		return f.Fuel / (c1000 / 1000), false
	}
	return f.Fuel * 1000 / c1000, false
}

// freeWarp is the fastest warp at which a leg of dist light-years costs no
// fuel (the lowest warp with a non-zero cost, minus one), or 0 when even
// warp 1 costs fuel (BINARY-ONLY: the warp is then left unchanged). dist is
// the whole leg from the fleet's position at the start of the year
// (stars-elegy PARITY.md "Binary-model check (FM-004)").
func (g *Game) freeWarp(f *Fleet, dist int) int {
	for w := 1; w <= 10; w++ {
		if g.FuelCost(f, w, dist) > 0 {
			return w - 1
		}
	}
	return 10
}

func (g *Game) tankCapacity(f *Fleet) int {
	c := 0
	for _, s := range f.Stacks {
		c += s.Count * g.Designs[s.Design].FuelCapacity
	}
	return c
}

// ramScoopGain is the fuel engines that are free at warp make over dist
// light-years. CONFIRMED (FM-002..004) for one engine per ship in the first
// slot; more than one engine per ship (e > 1) is BINARY-ONLY.
func (g *Game) ramScoopGain(f *Fleet, warp, dist int) int {
	gain := 0
	for _, s := range f.Stacks {
		d := g.Designs[s.Design]
		if d.Engine.Fuel[warp] != 0 {
			continue
		}
		free := 0
		for j := 1; j <= 3 && warp+j <= 10 && d.Engine.Fuel[warp+j] == 0; j++ {
			free = j
		}
		k := [...]int{1, 3, 6, 10}[free]
		gain += s.Count * d.Engines * k * dist
	}
	return gain
}

// along returns the point a move of a light-years from p toward dest (at
// distance d) ends on: each coordinate rounded half away from zero.
func along(p, dest Point, a int, d float64) Point {
	step := func(x0, x1 int) int {
		v := float64(x1-x0) * float64(a) / d
		if x1 > x0 {
			return x0 + int(v+0.5)
		}
		return x0 + int(v-0.5)
	}
	return Point{step(p.X, dest.X), step(p.Y, dest.Y)}
}

// arrives is the arrival test for a move of a light-years toward a
// destination at distance d.
func arrives(d float64, a int) bool {
	t := int(d - 0.99999)
	return t < a || t <= 0
}

func (g *Game) fleetIndex(id int) int {
	for i := range g.Fleets {
		if g.Fleets[i].ID == id {
			return i
		}
	}
	return -1
}

func (g *Game) planetPos(id int) (Point, bool) {
	for _, p := range g.Planets {
		if p.ID == id {
			return p.Pos, true
		}
	}
	return Point{}, false
}

func (g *Game) destination(wp Waypoint) Point {
	switch wp.Target {
	case TargetPlanet:
		if p, ok := g.planetPos(wp.ID); ok {
			return p
		}
	case TargetFleet:
		if i := g.fleetIndex(wp.ID); i >= 0 {
			return g.Fleets[i].Pos
		}
	}
	return wp.Pos
}

// OrbitedPlanet returns the id of the planet a fleet orbits: a fleet is in
// orbit exactly when its coordinates equal the planet's.
func (g *Game) OrbitedPlanet(f *Fleet) (int, bool) {
	for _, p := range g.Planets {
		if p.Pos == f.Pos {
			return p.ID, true
		}
	}
	return 0, false
}

func (g *Game) fleetEvent(f *Fleet, kind EventKind, count int) Event {
	return Event{Kind: kind, Player: f.Owner, Planet: -1, Fleet: f.ID, Count: count}
}

// arrive completes a fleet's current waypoint.
func (g *Game) arrive(f *Fleet) []Event {
	f.Waypoints = f.Waypoints[1:]
	if len(f.Waypoints) == 0 {
		f.Waypoints = nil
		return []Event{g.fleetEvent(f, EventFleetArrived, 0)}
	}
	return nil
}

func moving(f *Fleet) bool {
	return len(f.Waypoints) > 0 && f.Waypoints[0].Warp > 0
}

// moveFleets runs the movement phase: ordinary fleets in id order, then
// fleets chasing other fleets in rounds, then waypoint settlement.
// KERNEL.md "Fleet movement".
func moveFleets(g *Game) []Event {
	order := make([]int, len(g.Fleets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return g.Fleets[order[a]].ID < g.Fleets[order[b]].ID })

	var events []Event
	var chasers []int
	for _, i := range order {
		f := &g.Fleets[i]
		if !moving(f) {
			continue
		}
		if f.Waypoints[0].Target == TargetFleet && g.fleetIndex(f.Waypoints[0].ID) >= 0 {
			chasers = append(chasers, i)
			continue
		}
		events = append(events, g.moveOrdinary(f)...)
	}
	events = append(events, g.moveChasers(chasers)...)
	return append(events, g.settleWaypoints(order)...)
}

// legMove is one fuel-checked move toward dest: the fuel rules shared by
// ordinary fleets and by each chase round.
type legMove struct {
	f      *Fleet
	warp   int
	dest   Point
	d      float64 // distance to dest
	leg    int     // trunc(d + 0.9999), the whole leg
	enough bool    // could pay for the whole leg
}

func (g *Game) newLegMove(f *Fleet, warp int, dest Point) legMove {
	d := distance(f.Pos, dest)
	leg := int(d + 0.9999)
	return legMove{f: f, warp: warp, dest: dest, d: d, leg: leg, enough: f.Fuel >= g.FuelCost(f, warp, leg)}
}

// limit applies the fuel range r (unlimited when unl) to a move of a
// light-years and reports whether R limited it.
func (m legMove) limit(a, r int, unl bool) (int, bool) {
	if !m.enough && !unl && a > r {
		return r, true
	}
	return a, false
}

// place moves the fleet a light-years and reports whether it reached dest.
func (m legMove) place(a int, rLimited bool) bool {
	switch {
	case rLimited && a == 0:
		return false // R = 0: the fleet does not move
	case arrives(m.d, a):
		m.f.Pos = m.dest
		return true
	}
	m.f.Pos = along(m.f.Pos, m.dest, a, m.d)
	return false
}

// ranDry is KERNEL.md "Running dry" (CONFIRMED, FM-002 24 and 29), after
// paying: fuel 0, limited by R or
// paid a non-zero cost, could not afford the whole leg, and short of the
// destination (or R = 0).
func (m legMove) ranDry(rLimited bool, cost int, arrived bool, a int) bool {
	return m.f.Fuel == 0 && (rLimited || cost > 0) && !m.enough && (!arrived || rLimited && a == 0)
}

// dryOut lowers the leg's warp to the fastest free warp for the whole leg
// from the fleet's start-of-year position (or leaves it when none is free).
func (g *Game) dryOut(wp *Waypoint, f *Fleet, leg int) Event {
	if fw := g.freeWarp(f, leg); fw > 0 {
		wp.Warp = fw
	}
	return g.fleetEvent(f, EventOutOfFuel, 0)
}

// afterMove applies the top-up and free-warp fuel gain to a fleet that did
// not run dry, returning the fuel added.
func (g *Game) afterMove(m legMove, a int, arrived bool) (int, []Event) {
	f := m.f
	before := f.Fuel
	if m.enough && !arrived {
		// Top-up (CONFIRMED, FM-004 TU; the tank cap is not exercised):
		// per-year rounding never strands a fleet that could pay for the
		// whole leg.
		rest := g.FuelCost(f, m.warp, int(distance(f.Pos, m.dest)+0.9999))
		f.Fuel = max(f.Fuel, min(rest, g.tankCapacity(f)))
	}
	var events []Event
	if gain := g.ramScoopGain(f, m.warp, max(0, min(int(m.d-0.99999), a))); gain > 0 {
		gain = min(gain, g.tankCapacity(f)-f.Fuel)
		if gain > 0 {
			f.Fuel += gain
			events = append(events, g.fleetEvent(f, EventRamScoopFuel, gain))
		}
	}
	return f.Fuel - before, events
}

// moveOrdinary moves a fleet toward a fixed destination for one year.
func (g *Game) moveOrdinary(f *Fleet) []Event {
	wp := &f.Waypoints[0]
	m := g.newLegMove(f, wp.Warp, g.destination(*wp))
	r, unl := g.fuelRange(f, m.warp)
	a, rLimited := m.limit(min(m.leg, m.warp*m.warp), r, unl)

	arrived := m.place(a, rLimited)
	cost := g.FuelCost(f, m.warp, a)
	f.Fuel = max(0, f.Fuel-cost)
	if rLimited {
		f.Fuel = 0
	}

	if m.ranDry(rLimited, cost, arrived, a) {
		return []Event{g.dryOut(wp, f, m.leg)}
	}
	_, events := g.afterMove(m, a, arrived)
	return events
}

// moveChasers moves fleets whose destination is another fleet, in rounds.
// KERNEL.md "Chasing another fleet": rounds, steps and charging on the
// year's total are CONFIRMED (FM-001..003); the per-round fuel rules (rule
// 6: R reduced by the distance already moved, running dry, top-up and ram
// scoop per round) are BINARY-ONLY.
func (g *Game) moveChasers(chasers []int) []Event {
	type chase struct {
		rem, moved int
		// fuel0 is the start-of-year fuel plus any top-up and ram-scoop
		// fuel gained in earlier rounds; each round charges the year's
		// total distance against it.
		fuel0  int
		active bool
	}
	state := map[int]*chase{} // by fleet index
	for _, i := range chasers {
		w := g.Fleets[i].Waypoints[0].Warp
		state[i] = &chase{rem: w * w, fuel0: g.Fleets[i].Fuel, active: true}
	}

	var events []Event
	for round := 0; round < 10; round++ {
		for _, i := range chasers {
			c := state[i]
			if !c.active {
				continue
			}
			f := &g.Fleets[i]
			wp := &f.Waypoints[0]
			ti := g.fleetIndex(wp.ID)
			tc, targetChasing := state[ti]
			targetChasing = targetChasing && tc.active

			step := c.rem
			if targetChasing {
				step = min(c.rem, (c.rem+c.moved+4)/5)
			}
			m := g.newLegMove(f, wp.Warp, g.Fleets[ti].Pos)
			start := *f
			start.Fuel = c.fuel0
			r, unl := g.fuelRange(&start, m.warp)
			a, rLimited := m.limit(min(m.leg, step), max(0, r-c.moved), unl)

			arrived := m.place(a, rLimited)
			total := c.moved + step
			if arrived || rLimited {
				total = c.moved + a
			}
			before := f.Fuel
			f.Fuel = max(0, c.fuel0-g.FuelCost(f, m.warp, total))
			if rLimited {
				f.Fuel = 0
			}
			cost := before - f.Fuel

			if arrived {
				c.active = false
				if targetChasing {
					tc.active = false // the target stops for the year
				}
			} else {
				c.moved += step
				c.rem -= step
				c.active = c.rem > 0
			}
			if m.ranDry(rLimited, cost, arrived, a) {
				events = append(events, g.dryOut(wp, f, m.leg))
				c.active = false
				continue
			}
			gain, ev := g.afterMove(m, a, arrived)
			c.fuel0 += gain
			events = append(events, ev...)
		}
	}
	return events
}

// settleWaypoints runs after all movement (KERNEL.md "Chasing another
// fleet" rules 6 and 7, CONFIRMED FM-001..003): every waypoint aimed at a
// fleet takes that fleet's end position, then every fleet sitting exactly
// on its next waypoint completes it.
func (g *Game) settleWaypoints(order []int) []Event {
	for _, i := range order {
		for j := range g.Fleets[i].Waypoints {
			wp := &g.Fleets[i].Waypoints[j]
			if wp.Target == TargetFleet {
				if t := g.fleetIndex(wp.ID); t >= 0 {
					wp.Pos = g.Fleets[t].Pos
				}
			}
		}
	}
	var events []Event
	for _, i := range order {
		f := &g.Fleets[i]
		if len(f.Waypoints) > 0 && g.destination(f.Waypoints[0]) == f.Pos {
			events = append(events, g.arrive(f)...)
		}
	}
	return events
}

// refuelFleets sets every fleet orbiting a planet of its owner's that has a
// starbase with a dock to its tank capacity, also lowering fuel above it.
// KERNEL.md "Refuelling at a starbase", CONFIRMED (FM-004 DK).
func refuelFleets(g *Game) {
	for i := range g.Fleets {
		f := &g.Fleets[i]
		for _, p := range g.Planets {
			if p.Pos == f.Pos && p.Owner == f.Owner && p.StarbaseDock {
				f.Fuel = g.tankCapacity(f)
				break
			}
		}
	}
}

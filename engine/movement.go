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

	// Hull and Slots carry the hull's and parts' values and base costs.
	// Engines count in Slots as PartEngine parts for cost; Engine and
	// Engines give their fuel table and number for movement.
	Hull  Hull
	Slots []Slot
	// SlotPos is the hull slot each entry of Slots fills, when the design
	// came from NewDesign; the same-hull starbase replacement cost compares
	// slots by it (launch.go).
	SlotPos []int
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
	ID     int  // planet or fleet id for TargetPlanet / TargetFleet
	Task   Task // the task the fleet takes up on arriving
}

type Fleet struct {
	// ID is Elegy's internal key, unique across players. Number is the
	// owner's fleet number as the specs count it (PRODUCTION-LAUNCH.md
	// "the owner's lowest unused fleet number"); fleet order is by owner,
	// then Number, then ID.
	ID        int
	Number    int
	Owner     int
	Pos       Point
	Stacks    []Stack
	Fuel      int // mg
	Cargo     Cargo
	Waypoints []Waypoint
	Plan      int // battle plan index in the owner's Plans
	// Name is the name the owner gave the fleet, or empty (orders.go).
	Name string
	// Task is the task of the original's waypoint 0. A fleet in transit
	// keeps it (ORDERS.md "Waypoint 0 and a task still in progress",
	// MEASURED WU-ROUTE), and takes up the task of the waypoint it
	// arrives at.
	Task Task
	// Heading and HeadingWarp are the direction of this year's last
	// movement step (halved until each component fits in ±127) and its
	// warp, as other players see them (SCANNING.md "Heading"). Both are
	// zero when the fleet did not move this year.
	Heading     Point
	HeadingWarp int
	// Repeat is the fleet's repeat-orders flag (ORDERS.md "Reaching a
	// waypoint").
	Repeat bool
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
func (s FleetShips) groups(warp int) []fleetDesign {
	var out []fleetDesign
	index := map[int]int{}
	for _, st := range s.Stacks {
		if i, ok := index[st.Index]; ok {
			out[i].n += st.Count
			continue
		}
		index[st.Index] = len(out)
		out = append(out, fleetDesign{d: st.Design, n: st.Count})
	}
	sort.SliceStable(out, func(a, b int) bool {
		return s.factor(out[a].d, warp) < s.factor(out[b].d, warp)
	})
	left := s.CargoMass
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
	return g.Ships(f).FuelCost(warp, dist)
}

// FuelCost is the fuel in mg the ships use to move dist light-years at
// warp (KERNEL.md "Fuel cost", CONFIRMED FM-001..003): cargo on the
// cheapest engine first, rounded up once.
func (s FleetShips) FuelCost(warp, dist int) int {
	tenths := 0
	for _, gr := range s.groups(warp) {
		tenths += fuelTermWrap(s.factor(gr.d, warp), dist, gr.n*gr.d.Mass+gr.cargo, s.FuelWrap)
	}
	return (tenths + 9) / 10
}

// underEngined is the engine factor of a ship design whose engine slot is
// empty or not filled to the hull's maximum (KERNEL.md "Designs without a
// full set of engines", CONFIRMED FM-105). NewDesign refuses such designs;
// the rule covers designs made any other way.
const underEngined = 99999

// engineFactor is f, the design's fuel factor at warp.
func engineFactor(d Design, warp int) int {
	if !d.Hull.Starbase && len(d.Hull.Slots) > 0 && d.Engines < d.Hull.Slots[0].Max {
		return underEngined
	}
	return d.Engine.Fuel[warp]
}

// factor is f for a design among the ships: the engine factor, less
// trunc(15f/100) for an Improved Fuel Efficiency owner (KERNEL.md "Other
// movement rules", CONFIRMED KB-4A E, FM-101). The under-engined factor
// is not reduced (FM-105 with an IFE owner).
func (s FleetShips) factor(d Design, warp int) int {
	e := engineFactor(d, warp)
	if e != underEngined && s.ImprovedFuelEfficiency {
		e -= 15 * e / 100
	}
	return e
}

// fuelTermWrap is one stack's fuel term trunc(f·L·M/2000) in tenths of
// a mg (KERNEL.md "Fuel cost"), for factor f, L light-years and mass M
// (ships plus cargo, kT), under the game's FuelWrap switch
// (FleetShips.FuelWrap); wrap reproduces the original's LEGACY
// BUG (Legacy.FuelWrap; KERNEL.md "Designs without a full set of
// engines", CONFIRMED FM-105): in its integer form the product f·L·M
// keeps its low 32 bits and is divided as a signed 32-bit number. Only an
// under-engined design's factor makes it wrap. Without wrap the product
// is exact.
func fuelTermWrap(f, l, m int, wrap bool) int {
	integer := m < 200 || f*l < 500000 && m < 4000 || f*l < 100000 && m < 20000
	if wrap && integer {
		return int(int32(uint32(f*l*m)) / 2000)
	}
	return f * l * m / 2000
}

// fuelRange is R, the distance the fleet's fuel pays for at warp, and
// whether it is unlimited.
func (g *Game) fuelRange(f *Fleet, warp int) (r int, unlimited bool) {
	return g.Ships(f).FuelRange(f.Fuel, warp)
}

// FuelRange is R, the distance fuel mg pays for at warp, and whether it
// is unlimited (KERNEL.md "Not enough fuel"; ESTIMATES.md "Est. range",
// CONFIRMED ES-001): with C1000 the cost of 1000 ly, truncated,
// unlimited when C1000 is 0, else trunc(fuel·1000/C1000), or
// trunc(fuel/trunc(C1000/1000)) above 100,000.
func (s FleetShips) FuelRange(fuel, warp int) (r int, unlimited bool) {
	sum := 0
	for _, gr := range s.groups(warp) {
		sum += fuelTermWrap(s.factor(gr.d, warp), 1000, gr.n*gr.d.Mass+gr.cargo, s.FuelWrap)
	}
	c1000 := sum / 10
	switch {
	case c1000 == 0:
		return 0, true
	case c1000 > 100_000:
		return fuel / (c1000 / 1000), false
	}
	return fuel * 1000 / c1000, false
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
	return g.Ships(f).ramScoopGain(warp, dist)
}

func (s FleetShips) ramScoopGain(warp, dist int) int {
	gain := 0
	for _, st := range s.Stacks {
		d := st.Design
		if engineFactor(d, warp) != 0 {
			continue
		}
		free := 0
		for j := 1; j <= 3 && warp+j <= 10 && d.Engine.Fuel[warp+j] == 0; j++ {
			free = j
		}
		k := [...]int{1, 3, 6, 10}[free]
		gain += st.Count * d.Engines * k * dist
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

// arrive completes a fleet's current waypoint (ORDERS.md "Reaching a
// waypoint", CONFIRMED by the WU batch): the reached waypoint becomes the
// fleet's location and its task the fleet's task. With repeat orders off
// it is dropped from the list. With repeat orders on a copy goes to the
// end of the list, unless the list's last waypoint is already at its
// position: so a single forward leg collapses to the reached waypoint and
// the fleet ends idle, and coincident last waypoints do not grow the list
// (CONFIRMED, WU-FALLBACK). A patrol waypoint never repeats (MEASURED,
// WU wuPNR). A fleet left with no waypoint is idle and its owner told.
func (g *Game) arrive(f *Fleet) []Event {
	reached := f.Waypoints[0]
	f.Task = reached.Task
	f.Waypoints = f.Waypoints[1:]
	if n := len(f.Waypoints); f.Repeat && reached.Task.Kind != TaskPatrol && n > 0 && f.Waypoints[n-1].Pos != reached.Pos {
		f.Waypoints = append(f.Waypoints[:n:n], reached)
	}
	if len(f.Waypoints) == 0 {
		f.Waypoints = nil
		return []Event{g.fleetEvent(f, EventFleetArrived, 0)}
	}
	return nil
}

// EventColonistsLostInFlight: Count = kT of colonists an Alternate Reality
// fleet lost before moving.
const EventColonistsLostInFlight EventKind = EventMergeRefused + 1

// arColonistLoss is KERNEL.md "Alternate Reality colonists in flight"
// (CONFIRMED, TK-117, OT-6): an AR fleet with more than 10 kT of colonists
// loses trunc((C+11)·3/100) kT, before movement, in every year its next
// waypoint has a warp above 0: also when that waypoint is its own
// position, when it has no fuel to move, and once a year for a chaser. A
// loss of 0 kT sends no message.
func (g *Game) arColonistLoss(f *Fleet) []Event {
	c := f.Cargo.Colonists
	if c <= 10 || f.Owner < 0 || f.Owner >= len(g.Players) || g.Players[f.Owner].Race.PRT != PRTAlternateReality {
		return nil
	}
	lost := (c + 11) * 3 / 100
	if lost == 0 { // 11..21 kT lose nothing (TK-117: 11 → 11) and get no message
		return nil
	}
	f.Cargo.Colonists -= lost
	return []Event{g.fleetEvent(f, EventColonistsLostInFlight, lost)}
}

// generateFuel adds each year's fuel from generators after movement: 50
// mg per fuel-generating part and 200 mg per fuel-transport ship, capped
// at the tank (KERNEL.md "Other movement rules", CONFIRMED CS-003-W,
// KB-4A G, X, FM-102 G–K).
func (g *Game) generateFuel() {
	for i := range g.Fleets {
		f := &g.Fleets[i]
		add := 0
		for _, s := range f.Stacks {
			d := g.Designs[s.Design]
			if d.Hull.FuelTransport {
				add += 200 * s.Count
			}
			for _, sl := range d.Slots {
				add += sl.Part.FuelPerYear * sl.Count * s.Count
			}
		}
		if add > 0 {
			f.Fuel = max(f.Fuel, min(f.Fuel+add, g.tankCapacity(f)))
		}
	}
}

// EventColonistsKilledByEngine: Count = kT of colonists a Radiating
// Hydro-Ram Scoop fleet lost.
const EventColonistsKilledByEngine EventKind = EventStarbaseBuilt + 1

// radiatingColonists is KERNEL.md's Radiating Hydro-Ram Scoop rule
// (CONFIRMED for mid 50, KB-4A H, FM-102 Z1, Z2; the exemptions
// BINARY-ONLY): a fleet with the engine that moved this year loses
// max(1, trunc(C·trunc((86 − mid)/2)/100)) kT of its C kT of colonists,
// at most all, with mid the owner's radiation midpoint.
func (g *Game) radiatingColonists(f *Fleet) []Event {
	c := f.Cargo.Colonists
	if c <= 0 || f.Owner < 0 || f.Owner >= len(g.Players) {
		return nil
	}
	rad := g.Players[f.Owner].Race.Env[Radiation]
	if rad.Immune || rad.Low+rad.High >= 170 {
		return nil
	}
	has := false
	for _, s := range f.Stacks {
		if g.Designs[s.Design].Engine.Name == "Radiating Hydro-Ram Scoop" {
			has = true
		}
	}
	if !has {
		return nil
	}
	mid := (rad.Low + rad.High) / 2
	lost := min(c, max(1, c*((86-mid)/2)/100))
	f.Cargo.Colonists -= lost
	return []Event{g.fleetEvent(f, EventColonistsKilledByEngine, lost)}
}

// moving reports a fleet that moves this year. A fleet whose current task
// is "lay mines" holds its place (KERNEL.md "Other movement rules",
// CONFIRMED OB-014-D, OB-019).
func moving(f *Fleet) bool {
	return len(f.Waypoints) > 0 && f.Waypoints[0].Warp > 0 && f.Task.Kind != TaskLayMines
}

// moveFleets runs the movement phase: ordinary fleets in fleet order
// (owner, then fleet number), then fleets chasing other fleets in rounds,
// then waypoint settlement. KERNEL.md "Turn order" step 3 and "Fleet
// movement".
func moveFleets(g *Game) []Event {
	ev, _ := g.moveAll(nil)
	return ev
}

// moveAll is moveFleets with the year's generator for minefield checks
// and stargates, and wormhole transit on arrival (KERNEL.md "Turn order"
// step 3.3). rng may be nil when the game has no space objects. gated
// holds the fleets that jumped by stargate, which get no repair this
// year.
func (g *Game) moveAll(rng Rand) (events []Event, gated map[int]bool) {
	gated = map[int]bool{}
	order := g.fleetOrder()

	for i := range g.Fleets {
		g.Fleets[i].Heading, g.Fleets[i].HeadingWarp = Point{}, 0
	}
	var chasers []int
	for _, i := range order {
		f := &g.Fleets[i]
		if !moving(f) {
			continue
		}
		events = append(events, g.arColonistLoss(f)...)
		if f.Waypoints[0].Warp == StargateWarp {
			events = append(events, g.stargate(i, rng, gated)...)
			continue
		}
		if f.Waypoints[0].Target == TargetFleet && g.fleetIndex(f.Waypoints[0].ID) >= 0 {
			chasers = append(chasers, i)
			continue
		}
		events = append(events, g.moveOrdinary(i, rng)...)
	}
	events = append(events, g.moveChasers(chasers, rng)...)
	return append(events, g.settleWaypoints(order)...), gated
}

// stargate is a fleet's stargate order (OBJECTS.md "Stargates"): on a
// jump the fleet is at its waypoint, which settleWaypoints then
// completes; it shows no heading or warp to others' scans, gets no
// repair this year, and other players' fleets chasing it stop at its
// departure point. A refused fleet stays with its waypoints. Without
// space objects a stargate order does nothing.
func (g *Game) stargate(fi int, rng Rand, gated map[int]bool) []Event {
	if g.Objects == nil {
		return nil
	}
	f := &g.Fleets[fi]
	from := f.Pos
	jumped, lost, ev := g.Objects.Stargate(g, fi, g.destination(f.Waypoints[0]), rng)
	f = &g.Fleets[fi]
	if lost {
		f.Stacks = nil // removed after movement
		return ev
	}
	if jumped {
		gated[f.ID] = true
		f.Heading, f.HeadingWarp = Point{}, 0
		g.loseFollowers(f, from)
	}
	return ev
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
	}
	m.f.Heading, m.f.HeadingWarp = scanHeading(m.dest.X-m.f.Pos.X, m.dest.Y-m.f.Pos.Y), m.warp
	switch {
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
func (g *Game) moveOrdinary(fi int, rng Rand) []Event {
	f := &g.Fleets[fi]
	wp := &f.Waypoints[0]
	m := g.newLegMove(f, wp.Warp, g.destination(*wp))
	r, unl := g.fuelRange(f, m.warp)
	a, rLimited := m.limit(min(m.leg, m.warp*m.warp), r, unl)

	from := f.Pos
	arrived := m.place(a, rLimited)
	cost := g.FuelCost(f, m.warp, a)
	f.Fuel = max(0, f.Fuel-cost)
	if rLimited {
		f.Fuel = 0
	}
	if hit, ev := g.mineStop(fi, m, from, a, arrived, rng); hit {
		return ev
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
func (g *Game) moveChasers(chasers []int, rng Rand) []Event {
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
			from := f.Pos
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

			if hit, ev := g.mineStop(i, m, from, a, arrived, rng); hit {
				events = append(events, ev...)
				c.active = false
				continue
			}
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
			reached := f.Waypoints[0]
			events = append(events, g.arrive(f)...)
			if reached.Target == TargetWormhole && g.Objects != nil {
				events = append(events, g.transit(i, reached.ID)...)
			}
		}
	}
	return events
}

// transit takes fleet fi through the wormhole end it reached (OBJECTS.md
// "Travel", CONFIRMED OB-005 C, D): only a waypoint aimed at the end
// itself, not a plain position on it. Other players' waypoints aimed at
// the fleet lose it and become plain positions where it entered.
func (g *Game) transit(fi, end int) []Event {
	f := &g.Fleets[fi]
	entry := f.Pos
	ev := g.Objects.TransitWormhole(g, fi, end)
	g.loseFollowers(&g.Fleets[fi], entry)
	return ev
}

// refuelFleets sets every fleet orbiting a planet of its owner's that has a
// starbase with a dock to its tank capacity, also lowering fuel above it.
// KERNEL.md "Refuelling at a starbase", CONFIRMED (FM-004 DK).
func refuelFleets(g *Game) {
	for i := range g.Fleets {
		f := &g.Fleets[i]
		for _, p := range g.Planets {
			// The owner's starbase, or one whose owner treats the fleet's
			// owner as a friend; not an Orbital Fort (KERNEL.md "Other
			// movement rules", CONFIRMED KB-4A F1–F5, FM-103).
			if p.Pos == f.Pos && p.Owner != NoOwner && p.StarbaseDock && (p.Owner == f.Owner || g.relation(p.Owner, f.Owner) == RelationFriend) {
				f.Fuel = g.tankCapacity(f)
				break
			}
		}
	}
}

// mineStop checks a fleet's movement step of a ly from `from` (the fleet
// has already been placed at the step's end) for a minefield stop
// (OBJECTS.md "Hits on moving fleets"): only at warp 1–10, only when the
// fleet moved, and with every field the space objects hold. On a hit the
// fleet goes back to its stop point, the hit is applied, and the step
// ends: no arrival, no top-up and no ram-scoop fuel ("On a hit"; the fuel
// for the step is already charged). A chaser stopped this way moves no
// further this year.
//
// ASSUMPTION O11: "the fuel for the full planned leg is already spent"
// is the step's normal charge, made before the check; the stop does not
// run the fleet dry.
func (g *Game) mineStop(fi int, m legMove, from Point, a int, arrived bool, rng Rand) (bool, []Event) {
	if g.Objects == nil || rng == nil || a <= 0 || m.warp < 1 || m.warp > 10 || m.d == 0 {
		return false, nil
	}
	stop, kind, hit := g.Objects.MineCheck(g, fi, from, m.dest, a, rng)
	if !hit {
		return false, nil
	}
	g.Fleets[fi].Pos = stop
	return true, g.Objects.MineHit(g, fi, kind, rng)
}

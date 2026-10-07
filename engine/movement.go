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

// Engine is an engine part as far as movement needs it.
type Engine struct {
	Name string
	// Fuel is the fuel factor f(w) for warp 0..10 (index 0 unused).
	Fuel [11]int
}

// Design is a ship design as far as movement needs it.
type Design struct {
	Name          string
	Mass          int // kT per ship
	Engine        Engine
	Engines       int // engines per ship
	CargoCapacity int // kT per ship
	FuelCapacity  int // mg per ship
}

// Stack is a number of ships of one design (an index into Game.Designs).
type Stack struct {
	Design int
	Count  int
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
}

// fleetDesign is one design group inside a fleet, with its share of cargo.
type fleetDesign struct {
	d     Design
	n     int
	cargo int
}

// groups returns the fleet's ships grouped by design, ordered by increasing
// f(w), with the fleet's cargo assigned to them in that order up to each
// group's capacity. KERNEL.md does not say how ties in f(w) are ordered;
// here they keep design order.
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
// fleets chasing other fleets in rounds. KERNEL.md "Fleet movement".
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
	return append(events, g.moveChasers(chasers)...)
}

// moveOrdinary moves a fleet toward a fixed destination for one year.
func (g *Game) moveOrdinary(f *Fleet) []Event {
	wp := &f.Waypoints[0]
	w := wp.Warp
	dest := g.destination(*wp)
	d := distance(f.Pos, dest)
	leg := int(d + 0.9999)
	a := min(leg, w*w)

	enough := f.Fuel >= g.FuelCost(f, w, leg)
	move, dry := a, false
	if !enough {
		if r, unlimited := g.fuelRange(f, w); !unlimited && a > r {
			move, dry = r, true
		}
	}

	var events []Event
	arrived := false
	switch {
	case dry && move == 0:
		// R = 0: the fleet does not move.
	case arrives(d, move):
		f.Pos = dest
		arrived = true
	default:
		f.Pos = along(f.Pos, dest, move, d)
	}

	f.Fuel -= g.FuelCost(f, w, move)
	if dry || f.Fuel <= 0 && !enough && !arrived {
		dry = true
		f.Fuel = 0
	}
	f.Fuel = max(0, f.Fuel)

	switch {
	case dry:
		if fw := g.freeWarp(f, leg); fw > 0 {
			wp.Warp = fw
		}
		events = append(events, g.fleetEvent(f, EventOutOfFuel, 0))
	default:
		if enough && !arrived {
			// Top-up (CONFIRMED, FM-004 TU; the tank cap is not exercised):
			// per-year rounding never strands a fleet that could pay for
			// the whole leg.
			rest := g.FuelCost(f, w, int(distance(f.Pos, dest)+0.9999))
			f.Fuel = max(f.Fuel, min(rest, g.tankCapacity(f)))
		}
		if gain := g.ramScoopGain(f, w, max(0, min(int(d-0.99999), move))); gain > 0 {
			gain = min(gain, g.tankCapacity(f)-f.Fuel)
			if gain > 0 {
				f.Fuel += gain
				events = append(events, g.fleetEvent(f, EventRamScoopFuel, gain))
			}
		}
	}

	if arrived {
		events = append(events, g.arrive(f)...)
	}
	return events
}

// moveChasers moves fleets whose destination is another fleet, in rounds.
// KERNEL.md "Chasing another fleet", CONFIRMED (FM-001..003). KERNEL.md does
// not give fuel-limit, out-of-fuel or ram-scoop rules for chasers; none are
// applied here beyond charging fuel.
func (g *Game) moveChasers(chasers []int) []Event {
	type chase struct {
		rem, moved, fuel0 int
		active            bool
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
			wp := f.Waypoints[0]
			ti := g.fleetIndex(wp.ID)
			tc, targetChasing := state[ti]
			targetChasing = targetChasing && tc.active

			step := c.rem
			if targetChasing {
				step = min(c.rem, (c.rem+c.moved+4)/5)
			}
			dest := g.Fleets[ti].Pos
			d := distance(f.Pos, dest)
			a := min(int(d+0.9999), step)

			total := c.moved + step
			if arrives(d, a) {
				f.Pos = dest
				total = c.moved + a
				c.active = false
				events = append(events, g.arrive(f)...)
				if targetChasing {
					tc.active = false // the target stops for the year
				}
				// KERNEL.md does not say so, but in FM-001..003 a target that
				// is itself chasing the arriving fleet (mutual chase) has
				// also arrived, whether or not it had finished moving.
				t := &g.Fleets[ti]
				if _, chasing := state[ti]; chasing && len(t.Waypoints) > 0 &&
					t.Waypoints[0].Target == TargetFleet && g.destination(t.Waypoints[0]) == t.Pos {
					events = append(events, g.arrive(t)...)
				}
			} else {
				f.Pos = along(f.Pos, dest, a, d)
				c.moved += step
				c.rem -= step
				c.active = c.rem > 0
			}
			f.Fuel = max(0, c.fuel0-g.FuelCost(f, wp.Warp, total))
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

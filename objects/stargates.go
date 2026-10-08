package objects

import (
	"math"
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
)

// GateWarp is the waypoint warp that asks for a stargate jump (OBJECTS.md
// "Stargates", CONFIRMED GT-004).
const GateWarp = 11

// anyRange is what an "any" range counts as (OBJECTS.md "Stargates").
const anyRange = 8000

// Gate is a stargate's limits: Mass 0 means any mass; Range is anyRange
// for any range.
type Gate struct {
	Mass, Range int
}

// StarbaseGate is the stargate in a starbase design's orbital slots
// (OBJECTS.md "What makes a gate", CONFIRMED GT-004; limits from the
// component table's safe_mass and safe_range, null meaning any).
func StarbaseGate(d engine.Design) (Gate, bool) {
	for _, sl := range d.Slots {
		if sl.Count <= 0 || sl.Part.Kind != engine.PartStargate {
			continue
		}
		c, ok := engine.Components().Lookup(sl.Part.Name)
		if !ok {
			continue
		}
		g := Gate{Range: anyRange}
		if v, ok := c.Stats["safe_mass"].(float64); ok {
			g.Mass = int(v)
		}
		if v, ok := c.Stats["safe_range"].(float64); ok {
			g.Range = int(v)
		}
		return g, true
	}
	return Gate{}, false
}

// PlanetGate is planet pi's gate: its starbase design's stargate.
func PlanetGate(g *engine.Game, pi int) (Gate, bool) {
	p := &g.Planets[pi]
	if p.Owner < 0 || !p.HasStarbase || p.StarbaseDesign < 0 || p.StarbaseDesign >= len(g.Designs) {
		return Gate{}, false
	}
	return StarbaseGate(g.Designs[p.StarbaseDesign])
}

// GateDanger is a design's loss percentage through a jump of d ly with
// range r and mass limits ms, md for a ship of mass m (OBJECTS.md
// "Danger", CONFIRMED GT-001): f = 10000; d > r: f = ⌊(5r − d)·2500/r⌋;
// for each limit M with 0 < M < m, f = ⌊⌊(5M − m)·2500/M⌋·f/10000⌋; a
// factor ≤ 0 gives 100, else ⌊(10000 − f)/100⌋.
func GateDanger(d, r, m int, limits ...int) int {
	f := 10000
	if d > r {
		f = (5*r - d) * 2500 / r
		if f <= 0 {
			return 100
		}
	}
	for _, M := range limits {
		if M > 0 && M < m {
			x := (5*M - m) * 2500 / M
			if x <= 0 {
				return 100
			}
			f = x * f / 10000
		}
	}
	if f <= 0 {
		return 100
	}
	return (10000 - f) / 100
}

// GateRefusal is why a jump was refused; GateOK when it went ahead.
type GateRefusal int

const (
	GateOK GateRefusal = iota
	NoSourceGate
	NoDestinationPlanet
	NoDestinationGate
	DestinationNotFriend
	ForeignColonists
	TooFar
	TooHeavy
)

// GateDesign is one design's outcome in a jump.
type GateDesign struct {
	Design    int
	Ships     int
	Pct       int
	Destroyed int // ships destroyed by the loss rolls
	Lost      bool
	Damage    int // per surviving ship, before averaging
}

// GateJump is the outcome of a stargate order.
type GateJump struct {
	Refused GateRefusal
	Source  int // source planet index, or -1 (Jump Gate)
	Dest    int // destination planet index, or -1
	// Unloaded is the cargo put on the source planet (it happens even when
	// the jump is then refused for range or mass).
	Unloaded engine.Cargo
	Designs  []GateDesign
	// FleetLost: the fleet is gone (every design lost, or the mixed-fleet
	// legacy count came to 0); the caller deletes it.
	FleetLost bool
}

// LegacyGateMixedFleetLoss reproduces the original's mixed-fleet count
// (OBJECTS.md "Mixed fleets", LEGACY BUG, MEASURED GT-001 H2, GT-002,
// CONFIRMED GT-003 W0–W5): designs lost entirely count twice against the
// fleet's number of designs and designs wiped out by the rolls once; at
// exactly 0 the whole fleet is deleted, survivors included. On by
// default.
var LegacyGateMixedFleetLoss = true

func friendly(g *engine.Game, owner, fleetOwner int) bool {
	return relation(g, owner, fleetOwner) == engine.RelationFriend
}

func allJumpGates(g *engine.Game, f *engine.Fleet) bool {
	if len(f.Stacks) == 0 {
		return false
	}
	for _, st := range f.Stacks {
		if st.Count <= 0 {
			continue
		}
		has := false
		for _, sl := range g.Designs[st.Design].Slots {
			if sl.Count > 0 && sl.Part.Name == "Jump Gate" {
				has = true
			}
		}
		if !has {
			return false
		}
	}
	return true
}

func planetAt(g *engine.Game, p engine.Point) int {
	for i := range g.Planets {
		if g.Planets[i].Pos == p {
			return i
		}
	}
	return -1
}

// Jump runs fleet fi's stargate order to dest (OBJECTS.md "Stargates",
// CONFIRMED GT-001..GT-004, OB-021, OB-022; marked parts BINARY-ONLY).
//
// Source: the gate of the planet the fleet is at, owned by the fleet
// owner or by a player who lists the fleet owner as a friend; without one
// every ship must carry a Jump Gate, and the destination gate's limits
// stand for both ends. Destination: a planet exactly at dest with a gate,
// owned the same way. Refusals in order: source, destination planet,
// destination gate, destination owner, colonists from a planet not the
// fleet owner's, then (after the unload) range and mass. The cargo
// (minerals and colonists) goes onto the source planet first unless the
// owner is Interstellar Traveler or the fleet jumps by Jump Gate (LEGACY
// BUG, MEASURED OB-021). Refused: d > 5R or a ship heavier than 5× either
// mass limit. Otherwise each design's GateDanger applies: pct 100 loses
// the design; 0 < pct < 100 destroys each ship on rand(100) < ⌊pct/3⌋
// (never for IT) and damages survivors by max(1, ⌊pct·armor/100⌋). The
// fleet moves to the destination with no fuel used and no minefield
// checks. A refused fleet stays with its waypoints. The caller handles
// messages, waypoints, the moved mark, no repair this year, chasers and
// deleting a lost fleet.
//
// Designs go in design-number order, every pct first; a design at 0%
// is skipped, one at 100% removed, both without draws. Otherwise, unless
// the owner is IT, each ship draws rand(100) < ⌊pct/3⌋ to be destroyed,
// and each destroyed ship draws rand(500) < u while D > 0, lowering D.
// Damage: D = max(1, ⌊p₀·n/100⌋) damaged ships with Dm = max(1,
// ⌊u·A/500⌋) each (0 and 0 when undamaged), Nw = max(1, ⌊pct·A/100⌋); if
// D > 0 and Nw + Dm ≥ A, D more ships are destroyed; the s left take avg
// = ⌊(Nw·s + Dm·D)/s⌋, stored as p = 100, u = max(1, ⌊avg·500/A⌋). When
// ships are lost, they take ⌊fuel·L/T⌋ of the fuel by fuel capacity, and
// cargo still aboard is lost the same way by cargo capacity (OBJECTS.md
// "Stargates", BINARY-ONLY; fits OB-021, GT-001).
//
// ASSUMPTION G4: when the extra destruction would take more ships than
// are left (D > s, which corrupts the original's record), every ship of
// the design is destroyed. A design's ships are taken as one stack, with
// its first stack's damage.
func Jump(g *engine.Game, fi int, dest engine.Point, rng engine.Rand) GateJump {
	f := &g.Fleets[fi]
	out := GateJump{Source: -1, Dest: -1}
	it := prt(g, f.Owner) == engine.PRTInterstellarTraveler
	src := planetAt(g, f.Pos)
	var sg Gate
	srcOK := false
	if src >= 0 {
		if gt, ok := PlanetGate(g, src); ok && friendly(g, g.Planets[src].Owner, f.Owner) {
			sg, srcOK, out.Source = gt, true, src
		}
	}
	byJumpGate := false
	if !srcOK {
		if !allJumpGates(g, f) {
			out.Refused = NoSourceGate
			return out
		}
		byJumpGate = true
	}
	di := planetAt(g, dest)
	if di < 0 {
		out.Refused = NoDestinationPlanet
		return out
	}
	out.Dest = di
	dg, ok := PlanetGate(g, di)
	if !ok {
		out.Refused = NoDestinationGate
		return out
	}
	if !friendly(g, g.Planets[di].Owner, f.Owner) {
		out.Refused = DestinationNotFriend
		return out
	}
	if byJumpGate {
		sg = dg
	}
	if !it && !byJumpGate {
		if f.Cargo.Colonists > 0 && g.Planets[src].Owner != f.Owner {
			out.Refused = ForeignColonists
			// Minerals alone are unloaded onto the friend's planet.
			out.Unloaded.Minerals = f.Cargo.Minerals
			for m := range f.Cargo.Minerals {
				g.Planets[src].Surface[m] += f.Cargo.Minerals[m]
			}
			f.Cargo.Minerals = engine.Minerals{}
			return out
		}
		out.Unloaded = f.Cargo
		for m := range f.Cargo.Minerals {
			g.Planets[src].Surface[m] += f.Cargo.Minerals[m]
		}
		g.Planets[src].Population += f.Cargo.Colonists // kT of colonists = units of 100
		f.Cargo = engine.Cargo{}
	}
	dx, dy := float64(dest.X-f.Pos.X), float64(dest.Y-f.Pos.Y)
	d := int(math.Sqrt(dx*dx + dy*dy))
	if d > 5*sg.Range {
		out.Refused = TooFar
		return out
	}
	// Designs in design-number order (the owner's ship slots); every
	// pct first, so a mass refusal comes before any draw.
	type group struct{ design, ships, slot, pct int }
	var groups []group
	idx := map[int]int{}
	for _, st := range f.Stacks {
		if st.Count <= 0 {
			continue
		}
		if i, ok := idx[st.Design]; ok {
			groups[i].ships += st.Count
			continue
		}
		idx[st.Design] = len(groups)
		groups = append(groups, group{design: st.Design, ships: st.Count, slot: designNumber(g, f.Owner, st.Design)})
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].slot < groups[j].slot })
	for i := range groups {
		gr := &groups[i]
		m := g.Designs[gr.design].Mass
		if (sg.Mass > 0 && m > 5*sg.Mass) || (dg.Mass > 0 && m > 5*dg.Mass) {
			out.Refused = TooHeavy
			return out
		}
		gr.pct = GateDanger(d, sg.Range, m, sg.Mass, dg.Mass)
	}
	fuelCapBefore, cargoCapBefore := fleetFuelCap(g, f), fleetCargoCap(g, f)
	count := len(groups)
	lostAny := false
	for _, gr := range groups {
		dsg := g.Designs[gr.design]
		gd := GateDesign{Design: gr.design, Ships: gr.ships, Pct: gr.pct}
		switch {
		case gd.Pct <= 0:
		case gd.Pct >= 100:
			gd.Lost, lostAny = true, true
			count -= 2
		default:
			armor := designArmor(dsg)
			old := firstDamage(f, gr.design)
			D, Dm := 0, 0
			if old.Pct > 0 && old.Units > 0 {
				D = max(1, old.Pct*gr.ships/100)
				Dm = max(1, old.Units*armor/500)
			}
			if k := gd.Pct / 3; k > 0 && !it {
				for range gr.ships {
					if rng.Intn(100) < k {
						gd.Destroyed++
						if D > 0 && rng.Intn(500) < old.Units {
							D--
						}
					}
				}
			}
			nw := max(1, gd.Pct*armor/100)
			gd.Damage = nw
			left := gr.ships - gd.Destroyed
			if left > 0 && D > 0 && nw+Dm >= armor {
				gd.Destroyed += min(D, left)
				left = max(0, left-D)
			}
			if gd.Destroyed > 0 {
				lostAny = true
			}
			if left <= 0 {
				gd.Lost = true
				count--
			} else {
				avg := (nw*left + Dm*D) / left
				setDesign(f, gr.design, left, engine.Damage{Pct: 100, Units: max(1, avg*500/armor)})
			}
		}
		out.Designs = append(out.Designs, gd)
	}
	kept := f.Stacks[:0]
	for _, st := range f.Stacks {
		lost := false
		for _, gd := range out.Designs {
			if gd.Design == st.Design && gd.Lost {
				lost = true
			}
		}
		if !lost && st.Count > 0 {
			kept = append(kept, st)
		}
	}
	f.Stacks = kept
	if len(f.Stacks) == 0 || (LegacyGateMixedFleetLoss && count == 0) {
		out.FleetLost = true
		return out
	}
	if lostAny {
		if fuelCapBefore > 0 {
			f.Fuel -= f.Fuel * (fuelCapBefore - fleetFuelCap(g, f)) / fuelCapBefore
		}
		if cargoCapBefore > 0 {
			loseShare(&f.Cargo, cargoCapBefore-fleetCargoCap(g, f), cargoCapBefore)
		}
	}
	f.Pos = dest
	return out
}

// designNumber is a design's number in its owner's ship slots, for
// ordering; a design in no slot sorts after them, by index.
func designNumber(g *engine.Game, owner, design int) int {
	for _, ds := range g.DesignSlots {
		if ds.Owner == owner && !ds.Starbase && ds.Design == design {
			return ds.Slot
		}
	}
	return maxShipDesigns + design
}

// firstDamage is the damage of a design's first stack in the fleet.
func firstDamage(f *engine.Fleet, design int) engine.Damage {
	for _, st := range f.Stacks {
		if st.Design == design && st.Count > 0 {
			return st.Damage
		}
	}
	return engine.Damage{}
}

// setDesign leaves a design's n survivors in its first stack with dmg
// and empties its other stacks.
func setDesign(f *engine.Fleet, design, n int, dmg engine.Damage) {
	first := true
	for si := range f.Stacks {
		st := &f.Stacks[si]
		if st.Design != design || st.Count <= 0 {
			continue
		}
		if first {
			st.Count, st.Damage, first = n, dmg, false
		} else {
			st.Count = 0
		}
	}
}

// loseShare takes ⌊C·lost/total⌋ of the cargo C, split as ⌊c·x/C⌋ per
// item, then the remainder 1 kT at a time over ironium, boranium,
// germanium and colonists in one pass over the items still holding cargo.
func loseShare(c *engine.Cargo, lost, total int) {
	items := []*int{&c.Minerals[engine.Ironium], &c.Minerals[engine.Boranium], &c.Minerals[engine.Germanium], &c.Colonists}
	C := 0
	for _, v := range items {
		C += *v
	}
	if C == 0 || lost <= 0 {
		return
	}
	x := C * lost / total
	taken := 0
	for _, v := range items {
		t := *v * x / C
		*v -= t
		taken += t
	}
	for _, v := range items {
		if taken >= x {
			break
		}
		if *v > 0 {
			*v--
			taken++
		}
	}
}

func fleetCargoCap(g *engine.Game, f *engine.Fleet) int {
	c := 0
	for _, st := range f.Stacks {
		c += st.Count * g.Designs[st.Design].CargoCapacity
	}
	return c
}

func fleetFuelCap(g *engine.Game, f *engine.Fleet) int {
	c := 0
	for _, st := range f.Stacks {
		c += st.Count * g.Designs[st.Design].FuelCapacity
	}
	return c
}

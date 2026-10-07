package engine

// Ships and starbases leaving production: stars-elegy
// docs/PRODUCTION-LAUNCH.md (PR #57 head 2496594). Rules are tagged as
// there; Elegy's own choices are ASSUMPTION Ln (docs/ORDERS-LAYER-STATUS.md).
//
// Elegy's production queue has no ship or starbase items yet, so nothing
// calls Launch or BuildStarbase from production. They are the build step
// such an item will call when it completes.

// Launch and starbase messages.
const (
	EventShipsBuilt          EventKind = iota + EventCargoGiftLost + 1 // Planet, Fleet = the new or joined fleet, Count = ships
	EventFleetRouted                                                   // Planet = the destination, Fleet, Count = warp
	EventFleetNotRouted                                                // Planet = the destination, Fleet: no warp with fuel
	EventShipsJoinedFleet                                              // Planet, Fleet = the fleet joined, Count = ships
	EventShipsLostFleetLimit                                           // Planet, Count = ships lost at the 512-fleet limit
	EventPlansLost                                                     // Planet: the owner lacks the tech for the design
	EventStarbaseBuilt                                                 // Planet, Count = the dock limit in kT, DockUnlimited, or 0 without a dock
)

// TaskRoute is the route task a routed fleet carries on its second
// waypoint. What it does on arrival is the waypoint lane's (ORDERS.md
// "Route task"); Elegy does not carry it out yet.
const TaskRoute TaskKind = TaskMerge + 1

// maxFleets is a player's fleet limit (PRODUCTION-LAUNCH.md "The 512-fleet
// limit", CONFIRMED SL-08..SL-10).
const maxFleets = 512

// joinLimit is the largest stack a build event may join at the fleet
// limit (CONFIRMED, SL-10 variant).
const joinLimit = 32765

// noFuelWarpEngines skip the drop to a fuel-free warp; warp10Engines keep
// warp 10 (PRODUCTION-LAUNCH.md "Ideal warp of the fleet").
var (
	noFuelWarpEngines = map[string]bool{"Trans-Galactic Mizer Scoop": true, "Galaxy Scoop": true}
	warp10Engines     = map[string]bool{"Interspace-10": true, "Enigma Pulsar": true, "Trans-Star 10": true,
		"Trans-Galactic Mizer Scoop": true, "Galaxy Scoop": true}
)

// idealWarp is the fleet's ideal warp (PRODUCTION-LAUNCH.md "Ideal warp of
// the fleet", CONFIRMED for Long Hump 6 and Quick Jump 5 via SL-04..SL-07,
// BINARY-ONLY for other engines): from warp 10, each design in fleet order
// in turn lowers it.
func (g *Game) idealWarp(f *Fleet) int {
	w := 10
	for _, s := range f.Stacks {
		d := g.Designs[s.Design]
		if d.Engines == 0 {
			w = 0
			continue
		}
		fuel := d.Engine.Fuel
		for w > 0 && fuel[w] >= 121 {
			w--
		}
		if fuel[w] > 0 && !noFuelWarpEngines[d.Engine.Name] {
			switch {
			case w >= 5 && fuel[w-1] == 0:
				w--
			case w >= 6 && fuel[w-2] == 0:
				w -= 2
			case w >= 7 && fuel[w-3] == 0:
				w -= 3
			}
		}
		if w == 10 && !warp10Engines[d.Engine.Name] {
			w = 9
		}
	}
	return w
}

// starbaseDock reports whether the starbase at planet index pi has a dock
// (any starbase hull but the Orbital Fort, PRODUCTION-LAUNCH.md "Route
// warp"): from its design when the planet has one, else Planet.StarbaseDock.
func (g *Game) starbaseDock(pi int) bool {
	p := &g.Planets[pi]
	if p.HasStarbase && p.StarbaseDesign >= 0 && p.StarbaseDesign < len(g.Designs) {
		return g.Designs[p.StarbaseDesign].Hull.Dock != 0
	}
	return p.StarbaseDock
}

// routeWarp is the warp of a new fleet's route waypoint from planet index
// src to planet index dst (PRODUCTION-LAUNCH.md "Route warp", CONFIRMED
// SL-04..SL-07): the ideal warp; then, to the owner's own destination
// with starbases at both ends and a dock at the destination, the highest
// warp 9..1 whose fuel range covers d when the ideal warp is below 9;
// then the step-down rule, then down while the leg costs more than the
// fuel aboard.
//
// Stargates are not modelled: the gate branch is skipped.
func (g *Game) routeWarp(f *Fleet, src, dst int) int {
	d := int(distance(g.Planets[src].Pos, g.Planets[dst].Pos))
	w := g.idealWarp(f)
	if g.Planets[dst].Owner == f.Owner && g.hasStarbase(src) && g.hasStarbase(dst) && w < 9 && g.starbaseDock(dst) {
		w = 0
		for v := 9; v >= 1; v-- {
			if r, unlimited := g.fuelRange(f, v); unlimited || r >= d {
				w = v
				break
			}
		}
	}
	if w >= 1 && w <= 10 {
		m := d / w / w
		for w >= 3 && d/(w-1)/(w-1) <= m {
			w--
		}
		for w > 1 && g.FuelCost(f, w, d) > f.Fuel {
			w--
		}
	}
	return w
}

// lowestFreeFleetNumber is the number of a player's next new fleet
// (PRODUCTION-LAUNCH.md "The new fleet", CONFIRMED SL-02: the owner's
// lowest unused fleet number, counting from 1). It sorts among the
// owner's fleets by that number (stars-elegy #57 2496594, BINARY-ONLY;
// Game.fleetOrder).
func (g *Game) lowestFreeFleetNumber(owner int) int {
	used := map[int]bool{}
	for _, f := range g.Fleets {
		if f.Owner == owner {
			used[f.Number] = true
		}
	}
	n := 1
	for used[n] {
		n++
	}
	return n
}

// newFleetID is an unused Fleet.ID, Elegy's internal key.
func (g *Game) newFleetID() int {
	id := 0
	for _, f := range g.Fleets {
		id = max(id, f.ID)
	}
	return id + 1
}

// joinDamage is the damage of a stack of n ships, damage dmg and armor
// armor after b undamaged ships join it at the fleet limit
// (PRODUCTION-LAUNCH.md "The 512-fleet limit", CONFIRMED SL-10).
func joinDamage(dmg Damage, n, b, armor int) Damage {
	if n == 0 || dmg.Units == 0 || dmg.Pct == 0 || armor == 0 {
		return Damage{}
	}
	damaged := max(1, dmg.Pct*n/100)
	total := dmg.Units * armor / 10 * damaged / 50
	pct := max(1, damaged*100/(n+b))
	damaged2 := max(1, pct*(n+b)/100)
	return Damage{Pct: pct, Units: 5 * total / damaged2 * 100 / armor}
}

// designArmor is a design's armor per ship: the hull's and its parts'.
func designArmor(d Design) int {
	a := d.Hull.Armor
	for _, s := range d.Slots {
		a += s.Count * s.Part.Armor
	}
	return a
}

// Launch is one build event (PRODUCTION-LAUNCH.md "Ships"): count ships of
// design (an index into Game.Designs) completed by one queue item at
// planet index pi. It returns the events and the id of the fleet the
// ships are in, or -1.
//
// Without a starbase nothing is built (BINARY-ONLY). Without the tech for
// the hull and every part nothing is built and the owner is told
// (BINARY-ONLY). Below 512 fleets the ships make a new fleet: the lowest
// free number, undamaged, full tanks, no cargo, standing at the planet
// with no task, plan 0, no name (CONFIRMED SL-01, SL-02); with a route
// destination it gets a waypoint there with the route task (CONFIRMED
// SL-04..SL-07). At 512 fleets the ships join the owner's first fleet at
// the planet, in fleet order, whose stack of the design stays at or below
// 32765, or are lost (CONFIRMED SL-08..SL-10).
//
// Not modelled: the Alternate Reality default remote-mining task (Elegy
// has no remote mining), and the "did not move" mark, which GenerateTurn
// must leave off a fleet built this year (CONFIRMED SL-03).
func (g *Game) Launch(pi, design, count int) ([]Event, int) {
	p := &g.Planets[pi]
	owner := p.Owner
	if !g.hasStarbase(pi) || count <= 0 {
		return nil, -1
	}
	d := g.Designs[design]
	req := d.techReq()
	for f := range NumFields {
		if g.Players[owner].Research.Levels[f] < req[f] {
			return []Event{{Kind: EventPlansLost, Player: owner, Planet: p.ID, Fleet: -1}}, -1
		}
	}
	fleets := 0
	for _, f := range g.Fleets {
		if f.Owner == owner {
			fleets++
		}
	}
	if fleets >= maxFleets {
		return g.joinAtLimit(pi, design, count)
	}
	f := Fleet{ID: g.newFleetID(), Number: g.lowestFreeFleetNumber(owner), Owner: owner, Pos: p.Pos, Stacks: []Stack{{Design: design, Count: count}}}
	f.Fuel = g.tankCapacity(&f)
	events := []Event{{Kind: EventShipsBuilt, Player: owner, Planet: p.ID, Fleet: f.ID, Count: count}}
	if p.HasRoute {
		if dst := g.planetIndex(p.RouteTo); dst >= 0 {
			w := g.routeWarp(&f, pi, dst)
			f.Waypoints = []Waypoint{{Pos: g.Planets[dst].Pos, Warp: w, Target: TargetPlanet, ID: p.RouteTo, Task: Task{Kind: TaskRoute}}}
			kind := EventFleetRouted
			if w == 0 {
				kind = EventFleetNotRouted
			}
			events = append(events, Event{Kind: kind, Player: owner, Planet: p.RouteTo, Fleet: f.ID, Count: w})
		}
	}
	g.Fleets = append(g.Fleets, f)
	return events, f.ID
}

func (g *Game) joinAtLimit(pi, design, count int) ([]Event, int) {
	p := &g.Planets[pi]
	for _, i := range g.fleetOrder() {
		f := &g.Fleets[i]
		if f.Owner != p.Owner || f.Pos != p.Pos {
			continue
		}
		j := -1
		for k, s := range f.Stacks {
			if s.Design == design {
				j = k
			}
		}
		if j < 0 {
			g.insertStack(f, Stack{Design: design, Count: count})
		} else {
			s := &f.Stacks[j]
			if s.Count+count > joinLimit {
				continue
			}
			s.Damage = joinDamage(s.Damage, s.Count, count, designArmor(g.Designs[design]))
			s.Count += count
		}
		return []Event{{Kind: EventShipsJoinedFleet, Player: p.Owner, Planet: p.ID, Fleet: f.ID, Count: count}}, f.ID
	}
	return []Event{{Kind: EventShipsLostFleetLimit, Player: p.Owner, Planet: p.ID, Fleet: -1, Count: count}}, -1
}

// insertStack adds a stack to a fleet in its owner's design-slot order: a
// fleet keeps one count per design slot, so ships of a design the fleet
// lacks take their slot's place (PRODUCTION-LAUNCH.md, answer to Elegy's
// question Q18, stars-elegy #57 2496594, BINARY-ONLY). A design with no
// slot of the owner goes last.
//
// The order of a fleet's stacks elsewhere is the kernel lane's
// representation; this keeps the build path consistent with the rule.
func (g *Game) insertStack(f *Fleet, s Stack) {
	slot := func(d int) int {
		for _, ds := range g.DesignSlots {
			if ds.Owner == f.Owner && ds.Design == d && !ds.Starbase {
				return ds.Slot
			}
		}
		return maxShipDesigns
	}
	at := len(f.Stacks)
	for k, t := range f.Stacks {
		if slot(t.Design) > slot(s.Design) {
			at = k
			break
		}
	}
	f.Stacks = append(f.Stacks[:at:at], append([]Stack{s}, f.Stacks[at:]...)...)
}

// DockAllows is the production-queue check for a ship item (ORDERS.md
// "Production queue (starbase dock)" and PRODUCTION-LAUNCH.md "Can the
// planet build it", Elegy's chosen rule; the original host builds any
// ship at any starbase, CONFIRMED SL-12): the planet's starbase must have
// a dock, and the design's hull mass must be within the dock's limit.
func (g *Game) DockAllows(pi, design int) bool {
	p := &g.Planets[pi]
	if !p.HasStarbase || p.StarbaseDesign < 0 || p.StarbaseDesign >= len(g.Designs) {
		return false
	}
	dock := g.Designs[p.StarbaseDesign].Hull.Dock
	return dock == DockUnlimited || dock > 0 && g.Designs[design].Hull.Mass <= dock
}

// BuildStarbase completes a starbase item of design at planet index pi
// (PRODUCTION-LAUNCH.md "Starbases"): without the tech nothing is built
// and there is no message (BINARY-ONLY); otherwise the new starbase
// replaces the old one, keeps its damage units (CONFIRMED SL-12), and the
// owner is told what it can build, by its dock (stars-elegy #57 2496594,
// BINARY-ONLY): no ships (Orbital Fort, Count 0), ships up to N kT (Space
// Dock, Count N), or any size (Count DockUnlimited).
//
// Not modelled: removing queued ship items and resetting starbase items
// when the new hull is earlier in the hull list (CONFIRMED SL-12; the
// queue has no such items yet), and the mass driver (no packets).
func (g *Game) BuildStarbase(pi, design int) []Event {
	p := &g.Planets[pi]
	d := g.Designs[design]
	req := d.techReq()
	for f := range NumFields {
		if g.Players[p.Owner].Research.Levels[f] < req[f] {
			return nil
		}
	}
	p.HasStarbase, p.StarbaseDesign = true, design
	p.StarbaseHull = d.Hull.StarbaseNumber
	p.StarbaseDock = d.Hull.Dock != 0
	return []Event{{Kind: EventStarbaseBuilt, Player: p.Owner, Planet: p.ID, Fleet: -1, Count: d.Hull.Dock}}
}

// StarbaseReplacementCost is what a starbase design costs to build where a
// starbase of another hull stands (PRODUCTION-LAUNCH.md "Cost of a
// replacement", MEASURED SL-12): per component, with c the new design's
// owner cost and o the old one's, max(⌊c/2⌋, c − ⌊o/2⌋); then the
// Improved Starbases or Alternate Reality reduction and the halving, as
// for any starbase (StarbaseBuildCost).
//
// The same-hull rule (BINARY-ONLY) compares slot positions, which Elegy's
// designs do not record; ok is false for a same-hull replacement.
func StarbaseReplacementCost(newD, oldD Design, race Race, levels [NumFields]int) (c Cost, ok bool) {
	if newD.Hull.Name == oldD.Hull.Name {
		return Cost{}, false
	}
	return starbaseCharge(replacementBase(designCost(newD, race, levels), designCost(oldD, race, levels)), race), true
}

func replacementBase(c, o Cost) Cost {
	f := func(c, o int) int { return max(c/2, c-o/2) }
	r := Cost{Resources: f(c.Resources, o.Resources)}
	for m := range NumMinerals {
		r.Minerals[m] = f(c.Minerals[m], o.Minerals[m])
	}
	return r
}

// starbaseCharge is COMPONENTS.md "Starbases" after the owner cost: with
// Improved Starbases or Alternate Reality c − c/5, then halved rounding
// up.
func starbaseCharge(c Cost, race Race) Cost {
	isb := race.LRT.ImprovedStarbases || race.PRT == PRTAlternateReality
	f := func(v int) int {
		if isb {
			v -= v / 5
		}
		return (v + 1) / 2
	}
	c.Resources = f(c.Resources)
	for m := range NumMinerals {
		c.Minerals[m] = f(c.Minerals[m])
	}
	return c
}

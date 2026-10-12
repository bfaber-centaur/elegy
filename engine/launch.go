package engine

// Ships and starbases leaving production: stars-elegy
// docs/PRODUCTION-LAUNCH.md (main b654cb3). Rules are tagged as
// there; Elegy's own choices are ASSUMPTION Ln (docs/ORDERS-LAYER-STATUS.md).
//
// Elegy's production queue has no ship or starbase items yet, so nothing
// calls Launch or BuildStarbase from production. They are the build step
// such an item will call when it completes.

// Launch and starbase messages.
const (
	EventShipsBuilt          EventKind = iota + EventColonistsLostGiven + 1 // Planet, Fleet = the new or joined fleet, Count = ships
	EventFleetRouted                                                        // Planet = the destination, Fleet, Count = warp
	EventFleetNotRouted                                                     // Planet = the destination, Fleet: no warp with fuel
	EventShipsJoinedFleet                                                   // Planet, Fleet = the fleet joined, Count = ships
	EventShipsLostFleetLimit                                                // Planet, Count = ships lost at the 512-fleet limit
	EventPlansLost                                                          // Planet: the owner lacks the tech for the design
	EventStarbaseBuilt                                                      // Planet, Count = the dock limit in kT, DockUnlimited, or 0 without a dock
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

// IdealWarp is the fleet's ideal warp (FleetShips.IdealWarp).
func (g *Game) IdealWarp(f *Fleet) int { return g.Ships(f).IdealWarp() }

// IdealWarp is the ships' ideal warp (PRODUCTION-LAUNCH.md "Ideal warp of
// the fleet", CONFIRMED for Long Hump 6 and Quick Jump 5 via SL-04..SL-07,
// BINARY-ONLY for other engines; ESTIMATES.md "Fleets" uses the same
// value): from warp 10, each design in fleet order in turn lowers it.
func (s FleetShips) IdealWarp() int { return s.idealWarp(true) }

// RawWarp is IdealWarp without the free-warp step-down (step 3 of
// PRODUCTION-LAUNCH.md "Ideal warp of the fleet"): the `raw` value the
// computer players cap a re-picked warp at (AI.md §11, "the same without
// the free-warp step-down").
func (s FleetShips) RawWarp() int { return s.idealWarp(false) }

func (s FleetShips) idealWarp(stepDown bool) int {
	w := 10
	for _, st := range s.Stacks {
		d := st.Design
		if d.Engines == 0 {
			w = 0
			continue
		}
		fuel := d.Engine.Fuel
		for w > 0 && fuel[w] >= 121 {
			w--
		}
		if stepDown && fuel[w] > 0 && !noFuelWarpEngines[d.Engine.Name] {
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
// with starbases at both ends, the gate setting (warp 11) when gateSafe
// allows the jump, or else, with a dock at the destination, the highest
// warp 9..1 whose fuel range covers d when the ideal warp is below 9;
// then the step-down rule, then down while the leg costs more than the
// fuel aboard. The gate choice is CONFIRMED by the SL rows with gates at
// both ends (11 within range, 6 beyond it) and MEASURED for the route
// task (ORDERS.md "Route task", wuRSG2).
func (g *Game) routeWarp(f *Fleet, src, dst int) int {
	d := int(distance(g.Planets[src].Pos, g.Planets[dst].Pos))
	w := g.IdealWarp(f)
	own := g.Planets[dst].Owner == f.Owner && g.hasStarbase(src) && g.hasStarbase(dst)
	if own && g.gateSafe(f, src, dst, d) {
		return StargateWarp
	}
	if own && w < 9 && g.starbaseDock(dst) {
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

// anyGateRange is what an "any" gate range counts as (OBJECTS.md
// "Stargates").
const anyGateRange = 8000

// planetGate is the stargate on planet pi's starbase design: its mass
// limit (0 for any) and range (OBJECTS.md "What makes a gate"; limits
// from the component table's safe_mass and safe_range, null meaning any).
func (g *Game) planetGate(pi int) (mass, rng int, ok bool) {
	p := &g.Planets[pi]
	if !p.HasStarbase || p.StarbaseDesign < 0 || p.StarbaseDesign >= len(g.Designs) {
		return 0, 0, false
	}
	for _, sl := range g.Designs[p.StarbaseDesign].Parts() {
		if sl.Count <= 0 || sl.Part.Kind != PartStargate {
			continue
		}
		c, found := Components().Lookup(sl.Part.Name)
		if !found {
			continue
		}
		rng = anyGateRange
		if v, isNum := c.Stats["safe_mass"].(float64); isNum {
			mass = int(v)
		}
		if v, isNum := c.Stats["safe_range"].(float64); isNum {
			rng = int(v)
		}
		return mass, rng, true
	}
	return 0, 0, false
}

// gateSafe is the gate condition of the route warp (PRODUCTION-LAUNCH.md
// "Route warp"; ORDERS.md "Route task"): both planets' starbases have
// stargates, the fleet carries no minerals or colonists, and a jump of
// its heaviest design over d is allowed with no damage: d within the
// source gate's range and no design heavier than either gate's mass limit
// (OBJECTS.md "Stargates", "Limits", "Danger"). routeWarp asks only when
// both planets are the fleet owner's.
//
// ASSUMPTION W6: without space objects (Game.Objects) a gate jump does
// nothing, so routing never chooses one.
func (g *Game) gateSafe(f *Fleet, src, dst, d int) bool {
	if g.Objects == nil || f.Cargo.Minerals != (Minerals{}) || f.Cargo.Colonists != 0 {
		return false
	}
	ms, r, ok := g.planetGate(src)
	if !ok {
		return false
	}
	md, _, ok := g.planetGate(dst)
	if !ok || d > r {
		return false
	}
	for _, st := range f.Stacks {
		if st.Count <= 0 {
			continue
		}
		m := g.Designs[st.Design].Mass
		if (ms > 0 && m > ms) || (md > 0 && m > md) {
			return false
		}
	}
	return true
}

// lowestFreeFleetNumber is the number of a player's next new fleet
// (PRODUCTION-LAUNCH.md "The new fleet", CONFIRMED SL-02: the owner's
// lowest unused fleet number, counting from 1). It sorts among the
// owner's fleets by that number (BINARY-ONLY;
// Game.fleetOrder). A number held this year by a fleet that went away is
// not unused (holdFleetNumbers).
func (g *Game) lowestFreeFleetNumber(owner int) int {
	used := map[int]bool{}
	for _, f := range g.Fleets {
		if f.Owner == owner {
			used[f.Number] = true
		}
	}
	n := 1
	for used[n] || g.heldNumbers[[2]int{owner, n}] {
		n++
	}
	if g.heldNumbers != nil {
		g.heldNumbers[[2]int{owner, n}] = true
	}
	return n
}

// NewFleetNumber is lowestFreeFleetNumber for space objects that create
// fleets (a Mystery Trader's gift).
func (g *Game) NewFleetNumber(owner int) int { return g.lowestFreeFleetNumber(owner) }

// holdFleetNumbers starts the span, after the year's orders are read, in
// which a fleet that goes away keeps holding its number until the year's
// files are written (KERNEL.md "Fleet numbers": CONFIRMED for colonizing
// from orbit and scrapping, KX-007 and RG-001 2432, and for colonizing on
// arrival and a waypoint-0 merge, FN-02..05 in PARITY.md "Representative
// game (RG-001)", stars-elegy 30272be; BINARY-ONLY for the other ways a
// fleet goes away). A fleet deleted while the orders are
// read, merged away or emptied by a ship move, freed its number at once
// (BINARY-ONLY).
func (g *Game) holdFleetNumbers() {
	g.heldNumbers = map[[2]int]bool{}
	for _, f := range g.Fleets {
		g.heldNumbers[[2]int{f.Owner, f.Number}] = true
	}
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
// Without a starbase nothing is built (BINARY-ONLY). When the owner cannot
// build the design (canBuild) nothing is built and the owner is told the
// plans were lost (BINARY-ONLY). Below 512 fleets the ships make a new fleet: the lowest
// free number, undamaged, full tanks, no cargo, standing at the planet
// with no task, plan 0, no name (CONFIRMED SL-01, SL-02); with a route
// destination it gets a waypoint there with the route task (CONFIRMED
// SL-04..SL-07). At 512 fleets the ships join the owner's first fleet at
// the planet, in fleet order, whose stack of the design stays at or below
// 32765, or are lost (CONFIRMED SL-08..SL-10).
//
// An Alternate Reality owner's fleet that can mine gets the remote-mining
// task at the planet (defaultTask, CONFIRMED SL-11). GenerateTurn leaves
// the "did not move" mark off a fleet built this year (CONFIRMED SL-03).
func (g *Game) Launch(pi, design, count int) ([]Event, int) {
	p := &g.Planets[pi]
	owner := p.Owner
	if !g.hasStarbase(pi) || count <= 0 {
		return nil, -1
	}
	d := g.Designs[design]
	if !g.canBuild(owner, d) {
		return []Event{{Kind: EventPlansLost, Player: owner, Planet: p.ID, Fleet: -1}}, -1
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
	f.Task = g.defaultTask(&f)
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

// canBuild reports whether owner can build design d now
// (PRODUCTION-LAUNCH.md "Can the planet build it", BINARY-ONLY, stars-elegy
// 7419b90): the hull and every part are researched, allowed to the
// owner's race, and owned when a Mystery Trader item (COMPONENTS.md "Who
// can build what"). Reading a design in jrc3-faithful checks only the
// research (Legacy.KeepUnentitledParts), so such a design is stored but
// is not built.
func (g *Game) canBuild(owner int, d Design) bool {
	pl := &g.Players[owner]
	req := d.techReq()
	for f := range NumFields {
		if pl.Research.Levels[f] < req[f] {
			return false
		}
	}
	trader := g.traderItems(owner)
	names := []string{d.Hull.Name}
	for _, s := range d.Parts() {
		names = append(names, s.Part.Name)
	}
	cat := Components()
	for _, n := range names {
		c, ok := cat.Lookup(n)
		if !ok {
			continue
		}
		if ok, err := c.Allowed(pl.Race, trader[n]); err != nil || !ok {
			return false
		}
	}
	return true
}

// BuildStarbase completes a starbase item of design at planet index pi
// (PRODUCTION-LAUNCH.md "Starbases"): when the owner cannot build it
// (canBuild) nothing is built, the old starbase and the packet settings
// stay, and there is no message (BINARY-ONLY); otherwise the new starbase
// replaces the old one, keeps its damage units (CONFIRMED SL-12), and the
// owner is told what it can build, by its dock
// (BINARY-ONLY): no ships (Orbital Fort, Count 0), ships up to N kT (Space
// Dock, Count N), or any size (Count DockUnlimited).
//
// Production's queue change when the new hull is earlier in the hull list
// is in production.go (afterEarlierHull).
//
// Mass driver (PRODUCTION-LAUNCH.md "Starbases", stars-elegy a1c4c91,
// BINARY-ONLY): if the planet had no mass driver before and the new
// starbase has one, the planet's packet speed is set to the new driver's
// speed, whatever it was set to. The speed is Dw + t: RG-002 Y4 planet 88
// built a Station with two Mass Driver 7s on a planet with no starbase and
// read packet warp 8 (MEASURED, consistent: one observation, stars-elegy
// 6880639; Dw + t as at launch, BINARY-ONLY, stars-elegy 86ca470). If the
// planet had no driver and still has none, its packet destination and
// speed are cleared (BINARY-ONLY, stars-elegy 86ca470); production removes
// its queued mineral packet items (dropPacketItems). A planet that had a
// driver keeps its settings.
func (g *Game) BuildStarbase(pi, design int) []Event {
	p := &g.Planets[pi]
	d := g.Designs[design]
	if !g.canBuild(p.Owner, d) {
		return nil
	}
	hadDriver := false
	if p.HasStarbase && p.StarbaseDesign >= 0 && p.StarbaseDesign < len(g.Designs) {
		_, _, hadDriver = DriverWarp(g.Designs[p.StarbaseDesign])
	}
	if !hadDriver {
		if dw, t, ok := DriverWarp(d); ok {
			p.PacketSpeed = dw + t
		} else {
			p.HasPacketDest, p.PacketDest, p.PacketSpeed = false, 0, 0
		}
	}
	p.HasStarbase, p.StarbaseDesign = true, design
	p.StarbaseHull = d.Hull.StarbaseNumber
	p.StarbaseDock = d.Hull.Dock != 0
	return []Event{{Kind: EventStarbaseBuilt, Player: p.Owner, Planet: p.ID, Fleet: -1, Count: d.Hull.Dock}}
}

// DriverWarp is a starbase design's driver warp Dw and the two-driver
// bonus t (OBJECTS.md "Launch", CONFIRMED OB-028-C, D): Dw is the best
// mass driver's rating; t is 1 when two different slots each hold that
// best driver. ok is false without a mass driver.
func DriverWarp(d Design) (dw, t int, ok bool) {
	slots := 0
	for _, sl := range d.Parts() {
		if sl.Count <= 0 || sl.Part.Kind != PartMassDriver {
			continue
		}
		w := driverRating(sl.Part.Name)
		switch {
		case w > dw:
			dw, slots = w, 1
		case w == dw:
			slots++
		}
	}
	if dw == 0 {
		return 0, 0, false
	}
	if slots >= 2 {
		t = 1
	}
	return dw, t, true
}

func driverRating(name string) int {
	c, ok := Components().Lookup(name)
	if !ok {
		return 0
	}
	w, _ := c.Stats["warp"].(float64)
	return int(w)
}

// StarbaseReplacementCost is what a starbase design costs to build where a
// starbase stands (PRODUCTION-LAUNCH.md "Cost of a replacement"), per
// component, with c the new design's owner cost and o the old one's:
//   - a different hull: max(⌊c/2⌋, c − ⌊o/2⌋) (MEASURED SL-12);
//   - the same hull (BINARY-ONLY): c less the hull's cost, then for each
//     slot position filled in both designs, with N the new slot's cost
//     (count × part cost) and O the old slot's, the slot is charged
//     max(0, N − O) for the same part, max(N − ⌊8·O/10⌋, ⌊2·N/10⌋) for a
//     different part of the same kind, and max(N − ⌊7·O/10⌋, ⌊3·N/10⌋)
//     otherwise; the cost falls by N less the charge, not below 0.
//
// Then the Improved Starbases or Alternate Reality reduction and the
// halving, as for any starbase (StarbaseBuildCost). ok is false for a
// same-hull replacement whose designs do not record slot positions
// (Design.SlotPos; designs from NewDesign do).
func StarbaseReplacementCost(newD, oldD Design, race Race, levels [NumFields]int) (c Cost, ok bool) {
	if newD.Hull.Name != oldD.Hull.Name {
		return starbaseCharge(replacementBase(designCost(newD, race, levels), designCost(oldD, race, levels)), race), true
	}
	if len(newD.SlotPos) != len(newD.Slots) || len(oldD.SlotPos) != len(oldD.Slots) {
		return Cost{}, false
	}
	c = designCost(newD, race, levels)
	hull := itemCost(newD.Hull.Cost, newD.Hull.TechReq, PartOther, race, levels)
	fall := func(by Cost) {
		c.Resources = max(0, c.Resources-by.Resources)
		for m := range NumMinerals {
			c.Minerals[m] = max(0, c.Minerals[m]-by.Minerals[m])
		}
	}
	fall(hull)
	slotCost := func(s Slot) Cost {
		pc := itemCost(s.Part.Cost, s.Part.TechReq, s.Part.Kind, race, levels)
		pc.Resources *= s.Count
		for m := range NumMinerals {
			pc.Minerals[m] *= s.Count
		}
		return pc
	}
	for i, ns := range newD.Slots {
		for j, os := range oldD.Slots {
			if oldD.SlotPos[j] != newD.SlotPos[i] {
				continue
			}
			charge := func(n, o int) int {
				switch {
				case ns.Part.Name == os.Part.Name:
					return max(0, n-o)
				case ns.Part.Kind == os.Part.Kind:
					return max(n-8*o/10, 2*n/10)
				}
				return max(n-7*o/10, 3*n/10)
			}
			n, o := slotCost(ns), slotCost(os)
			by := Cost{Resources: n.Resources - charge(n.Resources, o.Resources)}
			for m := range NumMinerals {
				by.Minerals[m] = n.Minerals[m] - charge(n.Minerals[m], o.Minerals[m])
			}
			fall(by)
		}
	}
	return starbaseCharge(c, race), true
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

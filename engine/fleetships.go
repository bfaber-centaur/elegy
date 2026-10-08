package engine

// ShipStack is one stack of a fleet's ships for the movement estimates:
// Index is the design's index (stacks of one design group together for
// the fuel cost), Design its design.
type ShipStack struct {
	Index  int
	Design Design
	Count  int
}

// FleetShips is what the fuel and warp rules read of a fleet: its stacks
// in fleet order, its cargo mass in kT (minerals and colonists), whether
// its owner has Improved Fuel Efficiency, and the game's
// Legacy.FuelWrap setting. A planner that holds only its own view of the
// designs (and the game's ruleset, which every player knows) can build
// one without a Game. FuelWrap has no default: it must be copied from the
// game's Rules.Legacy.FuelWrap (Report.Rules for a player), as Game.Ships
// does.
type FleetShips struct {
	Stacks                 []ShipStack
	CargoMass              int
	ImprovedFuelEfficiency bool
	FuelWrap               bool
}

// Ships is fleet f as FleetShips.
func (g *Game) Ships(f *Fleet) FleetShips {
	s := FleetShips{CargoMass: f.Cargo.mass(), FuelWrap: g.Rules.Legacy.FuelWrap}
	for _, st := range f.Stacks {
		s.Stacks = append(s.Stacks, ShipStack{Index: st.Design, Design: g.Designs[st.Design], Count: st.Count})
	}
	if f.Owner >= 0 && f.Owner < len(g.Players) {
		s.ImprovedFuelEfficiency = g.Players[f.Owner].Race.LRT.ImprovedFuelEfficiency
	}
	return s
}

// fuelTransportYearly is the fuel a Fuel Transport or Super-Fuel Xport
// ship makes each year (ESTIMATES.md "Est. fuel usage"; KERNEL.md).
const fuelTransportYearly = 200

// EstLegFuel is the client's "Est. fuel usage" of one leg of d
// light-years at warp (ESTIMATES.md "Est. fuel usage", CONFIRMED ES-001):
// 0 at warp 0 or above 10; else the cost of the leg in one go or, when the
// client allows it more than one year, the larger of that and its cost
// year by year, lowered by the fuel the ships make on the way (ram scoops
// over a full year, and 200 mg a year per fuel-transport hull).
func (s FleetShips) EstLegFuel(d float64, warp int) int {
	if warp <= 0 || warp > 10 {
		return 0
	}
	v := warp * warp
	y := int((d+0.99999)/float64(v) + 0.9999)
	li := int(d + 0.99999)
	fuel := s.FuelCost(warp, int(d+0.9999))
	if y <= 1 {
		return fuel
	}
	b := s.FuelCost(warp, v)
	fuel = max(fuel, (y-1)*b+s.FuelCost(warp, max(0, li-(y-1)*v)))
	gain := s.ramScoopGain(warp, v)
	for _, st := range s.Stacks {
		if st.Design.Hull.FuelTransport {
			gain += fuelTransportYearly * st.Count
		}
	}
	switch {
	case gain <= 0:
	case gain >= b:
		fuel = b
	default:
		fuel = min(fuel, b+(y-1)*(b-gain))
	}
	return fuel
}

// EstRange is the client's "Est. range" of fleet f (ESTIMATES.md
// "Est. range", CONFIRMED ES-001): its fuel range at its ideal warp, and
// whether that is unlimited ("Infinite").
func (g *Game) EstRange(f *Fleet) (r int, unlimited bool) {
	s := g.Ships(f)
	return s.FuelRange(f.Fuel, s.IdealWarp())
}

// EstFuelUsage is the client's "Est. fuel usage" of fleet f to its n-th
// waypoint after waypoint 0, n ≥ 1 (ESTIMATES.md "Est. fuel usage",
// CONFIRMED ES-001): the legs' estimates (FleetShips.EstLegFuel) summed in
// order, restarting at 0 after a leg that ends at one of the owner's
// planets whose starbase has a dock, and the largest running total so
// far. A stargate leg (warp above 10) counts 0.
func (g *Game) EstFuelUsage(f *Fleet, n int) int {
	s := g.Ships(f)
	from, total, shown := f.Pos, 0, 0
	for i := 0; i < n && i < len(f.Waypoints); i++ {
		wp := f.Waypoints[i]
		total += s.EstLegFuel(distance(from, wp.Pos), wp.Warp)
		shown = max(shown, total)
		if pi := g.planetAt(wp.Pos); pi >= 0 && g.Planets[pi].Owner == f.Owner && g.hasStarbase(pi) && g.starbaseDock(pi) {
			total = 0
		}
		from = wp.Pos
	}
	return shown
}

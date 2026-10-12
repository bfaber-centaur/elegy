package engine

const (
	// EventColonistsBred: Count = kT of colonists an Inner Strength
	// Fleet's colonists bred into its own hold (MESSAGES.md 0x0fb).
	EventColonistsBred EventKind = iota + EventFollowDone + 1
	// EventColonistsBredLanded: Count = kT of bred colonists that did not
	// fit in the Fleet's hold and joined its owner's Planet (MESSAGES.md
	// 0x158).
	EventColonistsBredLanded
)

// breedInTransit is KERNEL.md "Turn order" step 3b, "Inner Strength
// colonists breed in transit" (stars-elegy fda4173, CONFIRMED OT-5): each
// Inner Strength fleet carrying C kT of colonists breeds
// g = trunc(C·growth/200) kT. What fits in the free hold joins the cargo;
// the rest joins the planet the fleet orbits when its owner owns it, and
// is otherwise lost. A g of 0 becomes 1 on a rand(3) draw of 0
// (BINARY-ONLY); fleets draw in fleet order (ASSUMPTION: KERNEL.md does
// not give the draw order).
func (g *Game) breedInTransit(rng Rand) []Event {
	var events []Event
	for _, i := range g.fleetOrder() {
		f := &g.Fleets[i]
		c := f.Cargo.Colonists
		if c <= 0 || f.Owner < 0 || f.Owner >= len(g.Players) {
			continue
		}
		race := g.Players[f.Owner].Race
		if race.PRT != PRTInnerStrength {
			continue
		}
		bred := c * race.GrowthRate / 200
		if bred == 0 && rng.Intn(3) == 0 {
			bred = 1
		}
		if bred == 0 {
			continue
		}
		free := max(0, g.cargoCapacity(f)-f.Cargo.mass())
		held := min(bred, free)
		if held > 0 {
			f.Cargo.Colonists += held
			events = append(events, g.fleetEvent(f, EventColonistsBred, held))
		}
		rest := bred - held
		if rest == 0 {
			continue
		}
		if pi := g.planetAt(f.Pos); pi >= 0 && g.Planets[pi].Owner == f.Owner {
			g.Planets[pi].Population += rest
			events = append(events, Event{Kind: EventColonistsBredLanded, Player: f.Owner, Planet: g.Planets[pi].ID, Fleet: f.ID, Count: rest})
		}
	}
	return events
}

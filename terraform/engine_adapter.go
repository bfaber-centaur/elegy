package terraform

import "github.com/bfaber-centaur/elegy/engine"

// Rules is the engine.Terraformer: production's terraform items (step 4),
// remote mining at step 6c.2 and the Orbital Adjusters at step 7.4
// (KERNEL.md "Turn order"; docs/TERRAFORM-STATUS.md, "Turn wiring"). It
// keeps no state.
type Rules struct{}

var _ engine.Terraformer = Rules{}

// MiningRate is MiningRate.
func (Rules) MiningRate(g *engine.Game, f *engine.Fleet) int { return MiningRate(g, f) }

// RemoteMine is RemoteMine.
func (Rules) RemoteMine(g *engine.Game, fi int, rng engine.Rand) (engine.Minerals, bool) {
	return RemoteMine(g, fi, rng)
}

// Adjust runs Adjust and turns each change into its messages (MESSAGES.md
// 0x12c, 0x15a and 0x15b). Messages go per fleet; Count is the planet
// owner's habitability after the fleet's clicks and, for a changed value,
// Axes[0] is it before them.
//
//   - The clicks changed the planet owner's habitability: the fleet owner
//     is told, and the planet owner too when it is someone else.
//   - A hostile fleet's clicks left it unchanged, even with an axis moved:
//     the fleet owner alone gets "cannot degrade any further" (CONFIRMED,
//     GAP-001 W1; KERNEL.md "Terraforming", stars-elegy 0d63a84).
//
// Not modelled: 0x12d for a friendly fleet that left the value unchanged,
// and the messages for a fleet that made no click at all.
func (Rules) Adjust(g *engine.Game) []engine.Event {
	var out []engine.Event
	for _, a := range Adjust(g) {
		f, p := &g.Fleets[a.Fleet], &g.Planets[a.Planet]
		if !a.HabChanged {
			if a.Hostile {
				out = append(out, engine.Event{Kind: engine.EventPlanetNotDegraded, Player: f.Owner, Planet: p.ID, Fleet: f.ID, Count: a.After})
			}
			continue
		}
		kind := engine.EventPlanetImproved
		if a.Hostile {
			kind = engine.EventPlanetDegraded
		}
		out = append(out, engine.Event{Kind: kind, Player: f.Owner, Planet: p.ID, Fleet: f.ID, Count: a.After, Axes: []int{a.Before}})
		if p.Owner != f.Owner {
			out = append(out, engine.Event{Kind: kind, Player: p.Owner, Planet: p.ID, Fleet: f.ID, Count: a.After, Axes: []int{a.Before}})
		}
	}
	return out
}

// Reach is Reach.
func (Rules) Reach(race engine.Race, levels [engine.NumFields]int) [3]int {
	return Reach(race, levels)
}

// Capacity is Capacity.
func (Rules) Capacity(p engine.Planet, race engine.Race, reach [3]int) int {
	return Capacity(p, race, reach)
}

// Improve is Improve.
func (Rules) Improve(p *engine.Planet, race engine.Race, reach [3]int) int {
	return Improve(p, race, reach)
}

// UnitCost is UnitCost.
func (Rules) UnitCost(race engine.Race) int { return UnitCost(race) }

// AutoUnits is AutoUnits.
func (Rules) AutoUnits(count, capacity int, minOnly bool, popChange, hab int) int {
	return AutoUnits(count, capacity, minOnly, popChange, hab)
}

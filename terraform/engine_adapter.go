package terraform

import "github.com/bfaber-centaur/elegy/engine"

// Rules is the engine.Terraformer GenerateTurn calls: remote mining at
// step 6c.2 and the Orbital Adjusters at step 7.4 (KERNEL.md "Turn
// order"). It keeps no state.
type Rules struct{}

var _ engine.Terraformer = Rules{}

// MiningRate is MiningRate.
func (Rules) MiningRate(g *engine.Game, f *engine.Fleet) int { return MiningRate(g, f) }

// RemoteMine is RemoteMine.
func (Rules) RemoteMine(g *engine.Game, fi int, rng engine.Rand) (engine.Minerals, bool) {
	return RemoteMine(g, fi, rng)
}

// Adjust runs Adjust and turns each change into its messages (MESSAGES.md
// 0x12c and 0x15a) when the planet owner's value changed: the fleet
// owner is told, and the planet owner too when it is someone else. Count
// is the planet owner's habitability after.
//
// Not modelled: the messages for a fleet that left the value unchanged
// (0x12d, 0x15b).
func (Rules) Adjust(g *engine.Game) []engine.Event {
	var out []engine.Event
	for _, a := range Adjust(g) {
		if !a.HabChanged {
			continue
		}
		f, p := &g.Fleets[a.Fleet], &g.Planets[a.Planet]
		kind := engine.EventPlanetImproved
		if a.Hostile {
			kind = engine.EventPlanetDegraded
		}
		hab := engine.Habitability(g.Players[p.Owner].Race, p.Env)
		out = append(out, engine.Event{Kind: kind, Player: f.Owner, Planet: p.ID, Fleet: f.ID, Count: hab})
		if p.Owner != f.Owner {
			out = append(out, engine.Event{Kind: kind, Player: p.Owner, Planet: p.ID, Fleet: f.ID, Count: hab})
		}
	}
	return out
}

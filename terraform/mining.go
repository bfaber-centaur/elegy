package terraform

import "github.com/bfaber-centaur/elegy/engine"

// MaxMiningRate caps a fleet's remote-mining rate (KERNEL.md "Remote
// mining", CONFIRMED KB-1A).
const MaxMiningRate = 4000

// MiningRate is fleet f's remote-mining rate m: the sum over its ships of
// each mining robot's mining_rate (Robo-Midget 5, Robo-Mini 4, Robo 12,
// Robo-Maxi 18, Robo-Super 27, Robo-Ultra 25, Alien 10; the Orbital
// Adjuster 0), at most 4,000 (KERNEL.md "Remote mining", COMPONENTS.md
// "Remote mining"; CONFIRMED T-35, CS-003-B X1/X2, KB-1A).
func MiningRate(g *engine.Game, f *engine.Fleet) int {
	cat := engine.Components()
	m := 0
	for _, st := range f.Stacks {
		if st.Design < 0 || st.Design >= len(g.Designs) {
			continue
		}
		per := 0
		for _, sl := range g.Designs[st.Design].Slots {
			c, ok := cat.Lookup(sl.Part.Name)
			if !ok {
				continue
			}
			if v, ok := c.Stats["mining_rate"].(float64); ok {
				per += sl.Count * int(v)
			}
		}
		m += per * st.Count
	}
	return min(m, MaxMiningRate)
}

// CanRemoteMine reports whether fleet f's robots mine the planet at index
// pi (KERNEL.md "Remote mining", CONFIRMED T-35, KB-1B): an unowned
// planet, or an Alternate Reality planet mined by its owner's own fleet.
// Miners at any other owned planet mine nothing.
//
// ASSUMPTION R1: another player's miners at an Alternate Reality planet
// (BINARY-ONLY) mine nothing, as at any other owned planet.
func CanRemoteMine(g *engine.Game, f *engine.Fleet, pi int) bool {
	p := &g.Planets[pi]
	if p.Owner == engine.NoOwner || p.Owner < 0 {
		return true
	}
	return p.Owner == f.Owner && p.Owner < len(g.Players) &&
		g.Players[p.Owner].Race.PRT == engine.PRTAlternateReality
}

// RemoteMine runs one year of remote mining by fleet fi at the planet it
// orbits (KERNEL.md "Remote mining", turn order step 6c). Which fleets
// mine is the turn engine's: a fleet whose task is remote mining and that
// did not move this year, so a fleet that arrived this year mines nothing
// until the next (CONFIRMED T-35; TAKEOVER.md, the order stays).
//
// Each mineral, in order ironium, boranium, germanium, is mined as
// planetary mining is with m in place of the mines and the race's mine
// output ignored (engine.MineYear with eff 10): output conc·m/100 kT, the
// same random +1 and depletion, no homeworld floor. At an Alternate
// Reality planet this is a separate mining step from the planet's own
// (KB-1B); the order of the two steps is BINARY-ONLY and the turn
// engine's.
//
// It returns the kT added to the planet's surface and whether the fleet
// mined (orbiting a planet it can mine, with a rate above 0); a fleet
// that does not mine makes no draws.
func RemoteMine(g *engine.Game, fi int, rng engine.Rand) (engine.Minerals, bool) {
	f := &g.Fleets[fi]
	var gain engine.Minerals
	pi := planetAt(g, f.Pos)
	if pi < 0 || !CanRemoteMine(g, f, pi) {
		return gain, false
	}
	m := MiningRate(g, f)
	if m == 0 {
		return gain, false
	}
	p := &g.Planets[pi]
	for k := range engine.NumMinerals {
		gain[k], p.Deposits[k] = engine.MineYear(p.Deposits[k], m, 10, false, rng)
		p.Surface[k] += gain[k]
	}
	return gain, true
}

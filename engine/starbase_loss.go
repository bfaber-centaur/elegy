package engine

// loseStarbase removes planet pi's starbase and strips its queue, as a
// starbase destroyed in battle or removed by deleting its design does
// (COMBAT.md, CONFIRMED CB-047 for battle: the ship items are gone and
// the planetary items stay; PARITY.md "Queued ships after deleting the
// starbase design", stars-elegy ee47fff, MEASURED CO-07b for the outcome
// of a design delete: nothing spent, the year's resources all to
// research). The queue loses every ship item and every packet item but
// the automatic one, and a starbase item left in it restarts at 0%
// (BINARY-ONLY, both causes). Planetary items stay.
func (g *Game) loseStarbase(pi int) {
	p := &g.Planets[pi]
	p.HasStarbase, p.StarbaseHull, p.StarbaseDock, p.StarbaseDamage = false, 0, false, 0
	var q []QueueItem
	for _, it := range p.Queue {
		if it.Kind == ItemShip || it.Kind.packet() && it.Kind != ItemAutoPackets {
			continue
		}
		if it.Kind == ItemStarbase {
			it.Percent = 0
		}
		q = append(q, it)
	}
	p.Queue = q
}

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
//
// The planet's packet destination and speed go with the starbase (ELEGY
// CHOICE, matching PK-1: stars-elegy PRODUCTION-LAUNCH.md "A starbase
// destroyed in battle", e1b2ad9, MEASURED that after the year the host
// file holds neither and that a rebuilt starbase does not bring them
// back). Whether the original clears them at destruction or keeps them
// unwritten is UNRESOLVED; the two differ in no observed case. A design
// delete clears them too (ASSUMPTION: PK-1 measured battle only).
func (g *Game) loseStarbase(pi int) {
	p := &g.Planets[pi]
	p.HasStarbase, p.StarbaseHull, p.StarbaseDock, p.StarbaseDamage = false, 0, false, 0
	p.HasPacketDest, p.PacketDest, p.PacketSpeed = false, 0, 0
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

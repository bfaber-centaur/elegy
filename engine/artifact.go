package engine

// EventArtifactFound: Player found the ancient artifact on Planet, which
// gave Count research points in field Item (0x05e).
const EventArtifactFound EventKind = EventTargetOutOfView + 1

// findArtifact is TAKEOVER.md "Ancient artifacts" (CONFIRMED TK-303,
// TK-304, TK-306; amounts MEASURED; stars-elegy 7c06f47): at the end of a
// drop resolution on planet pi, a planet holding an artifact that has an
// owner afterwards (a colony, a capture, or a defender that beat the
// drop off) gives that owner 100 + rand(301) research points in field
// rand(6), the field drawn first (draw order BINARY-ONLY), and the
// artifact is gone. Below 1,000 colonists the points are scaled by
// pop/1000. The points join the field's stored research before the
// level-up check that follows the drops. With random events off nothing
// happens (BINARY-ONLY).
//
// Slower tech does not halve the points (MEASURED TK-304; LEGACY BUG in
// the binary reading, where the halving is computed and never used), under
// Legacy.ArtifactFullSlowerTech, on in every current built-in. Off, half
// the points are stored, rounded up as stealResearch stores stolen
// research under slower tech (ELEGY CHOICE); the message keeps the points
// found.
//
// ASSUMPTION A1: the scaling truncates, points × colonists ÷ 1000;
// TAKEOVER.md gives the factor, not the rounding.
//
// resolveDrops calls it only when the planet has an owner: a tie, the one
// way a drop resolution leaves the planet unowned, returns before it.
func (g *Game) findArtifact(pi int, rng Rand) []Event {
	p := &g.Planets[pi]
	if !g.RandomEvents || !p.Artifact {
		return nil
	}
	f := rng.Intn(NumFields)
	points := 100 + rng.Intn(301)
	if colonists := p.Population * 100; colonists < 1000 {
		points = points * colonists / 1000
	}
	p.Artifact = false
	stored := points
	if g.SlowerTech && !g.Rules.Legacy.ArtifactFullSlowerTech {
		stored = (points + 1) / 2
	}
	g.Players[p.Owner].Research.Accumulated[f] += stored
	return []Event{{Kind: EventArtifactFound, Player: p.Owner, Planet: p.ID, Fleet: -1, Item: ItemKind(f), Count: points}}
}

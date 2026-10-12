package engine

// Remote-mining refusals (MESSAGES.md 0x075, 0x076, 0x077). Elegy's
// wording. Each cancels the fleet's task.
const (
	// EventRemoteMineNoModules: Player's Fleet has no remote-mining
	// modules; its mining order at Planet was cancelled (0x075).
	EventRemoteMineNoModules EventKind = iota + EventResearchStolen + 1
	// EventRemoteMineInhabited: Planet is inhabited; Player's Fleet's
	// mining order was cancelled (0x076).
	EventRemoteMineInhabited
	// EventRemoteMineDeepSpace: Player's Fleet is in deep space; its
	// mining order was cancelled (0x077).
	EventRemoteMineDeepSpace
)

// remoteMineTask is fleet i's remote-mining task after movement, for a
// fleet that did not move this year (KERNEL.md "Remote mining", stars-elegy
// b1e38ef; MESSAGES.md 0x075–0x077). The fleet mines when it orbits a
// planet it can mine (Terraformer.RemoteMine). Otherwise the order is
// refused with a message and the task cancelled, and the owner is told
// the fleet has finished its orders when this was its last waypoint
// (MESSAGES.md 0x04e (a) and Notes, oracle 0x076 then 0x04e):
//
//   - not orbiting a planet: 0x077 (BINARY-ONLY);
//   - no mining robots: 0x075 (CONFIRMED, CS-003);
//   - an owned planet it may not mine: 0x076 (CONFIRMED, TK-111), except
//     that an Alternate Reality fleet is skipped silently and keeps its
//     task (BINARY-ONLY).
//
// ASSUMPTION: the checks run in that order when more than one applies
// (deep space, then modules, then owner), and the silent skip keeps the
// task. Both are open oracle questions.
func (g *Game) remoteMineTask(i int, rng Rand) []Event {
	f := &g.Fleets[i]
	pi := g.planetAt(f.Pos)
	refuse := func(kind EventKind) []Event {
		ev := g.fleetEvent(f, kind, 0)
		if pi >= 0 {
			ev.Planet = g.Planets[pi].ID
		}
		f.Task = Task{}
		out := []Event{ev}
		if len(f.Waypoints) == 0 {
			out = append(out, g.fleetEvent(f, EventFleetArrived, 0))
		}
		return out
	}
	switch {
	case pi < 0:
		return refuse(EventRemoteMineDeepSpace)
	case g.Terraform.MiningRate(g, f) == 0:
		return refuse(EventRemoteMineNoModules)
	}
	p := &g.Planets[pi]
	ownAR := p.Owner == f.Owner && g.Players[f.Owner].Race.PRT == PRTAlternateReality
	if p.Owner >= 0 && !ownAR {
		if g.Players[f.Owner].Race.PRT == PRTAlternateReality {
			return nil
		}
		return refuse(EventRemoteMineInhabited)
	}
	g.Terraform.RemoteMine(g, i, rng)
	return nil
}

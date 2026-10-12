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
// e2fb8c1; MESSAGES.md 0x075–0x077). The fleet mines when it orbits a
// planet it can mine (Terraformer.RemoteMine). Otherwise the order is
// refused with a message and the task cancelled, and the owner is told
// the fleet has finished its orders when this was its last waypoint
// (MESSAGES.md 0x04e (a) and Notes, oracle 0x076 then 0x04e):
//
//   - not orbiting a planet: 0x077 (BINARY-ONLY);
//   - no mining robots: 0x075 (CONFIRMED, CS-003);
//   - an owned planet: 0x076 (CONFIRMED, TK-111), except that an
//     Alternate Reality fleet is skipped silently and keeps its task
//     (BINARY-ONLY). At its own planet it mined in mineOwnAR, or not at
//     all (a further waypoint), and is not mined again here.
//
// The checks run in that order and the first that fails decides, and the
// silently skipped Alternate Reality fleet keeps its task (BINARY-ONLY,
// KERNEL.md "Remote mining", stars-elegy e2fb8c1).
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
	if p.Owner >= 0 {
		if g.Players[f.Owner].Race.PRT == PRTAlternateReality {
			return nil // its own planet was mined in mineOwnAR
		}
		return refuse(EventRemoteMineInhabited)
	}
	g.Terraform.RemoteMine(g, i, rng)
	return nil
}

// mineOwnAR mines Alternate Reality planet pi with its owner's own remote
// miners, in the year's mining step right after the planet's own mines
// and before any production (KERNEL.md "Turn order" step 4.1 and "Remote
// mining", stars-elegy 8a3b9b5). The timing is MEASURED (RG-002 Y1, Y4
// restarts); the rest is BINARY-ONLY:
//
//   - a fleet mines when it has the planet's owner, orbits the planet
//     now (after movement, so one that arrived this year counts), has
//     the remote-mining task and has no further waypoint;
//   - each fleet is its own step (Terraformer.RemoteMine, with the
//     homeworld floor of 30), in fleet order, with its draws there;
//   - a fleet mined here keeps its task, and step 6c mines it no more
//     (remoteMineTask).
func (g *Game) mineOwnAR(pi int, rng Rand) {
	if g.Terraform == nil {
		return
	}
	p := &g.Planets[pi]
	for _, i := range g.fleetOrder() {
		f := &g.Fleets[i]
		if f.Owner == p.Owner && f.Pos == p.Pos && f.Task.Kind == TaskRemoteMine && len(f.Waypoints) == 0 {
			g.Terraform.RemoteMine(g, i, rng)
		}
	}
}

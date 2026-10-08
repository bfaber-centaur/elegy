package engine

import "slices"

// Waypoint upkeep and the tasks ORDERS.md "Waypoint upkeep and the
// remaining tasks" covers (stars-elegy main 5469029): the waypoint check,
// the route task, the transfer-fleet task and patrol. Reaching a waypoint
// is arrive (movement.go).

// TaskPatrol and TaskTransferFleet are the patrol and transfer-fleet
// waypoint tasks.
const (
	TaskPatrol        TaskKind = TaskRoute + 1 // Task.Range: the patrol range in ly
	TaskTransferFleet TaskKind = TaskRoute + 2 // Task.Player: the recipient
)

// waypointCheck is the waypoint check of KERNEL.md "Turn order" steps 1a.3
// and 7a.2 (ORDERS.md "Targets that moved, died or were captured",
// CONFIRMED): a waypoint aimed at a fleet that still exists takes the
// fleet's current position, whoever owns it now; a waypoint aimed at a
// fleet that no longer exists becomes a plain go-to at its last
// coordinates. A planet target never moves. Waypoints are clamped to the
// galaxy when ordered (ORDERS.md "Waypoint coordinates"), and refreshed
// positions are fleet positions, so nothing here needs clamping.
//
// A waypoint aimed at a wormhole end or a Mystery Trader takes the
// object's position the same way, and becomes a plain go-to when the
// object is gone ("Ordering with moving objects").
//
// Not modelled: the exception bit that holds a waypoint's coordinates
// (BINARY-ONLY; no order sets it) and following fleets (step 1a.3): Elegy
// does not keep waypoint 0's target, which that rule reads.
func (g *Game) waypointCheck() {
	for i := range g.Fleets {
		for j := range g.Fleets[i].Waypoints {
			wp := &g.Fleets[i].Waypoints[j]
			switch wp.Target {
			case TargetFleet:
				if t := g.fleetIndex(wp.ID); t >= 0 {
					wp.Pos = g.Fleets[t].Pos
				} else {
					wp.Target, wp.ID = TargetSpace, 0
				}
			case TargetWormhole, TargetTrader:
				if g.Objects == nil {
					wp.Target, wp.ID = TargetSpace, 0
				} else if p, ok := g.Objects.ObjectPos(wp.Target, wp.ID); ok {
					wp.Pos = p
				} else {
					wp.Target, wp.ID = TargetSpace, 0
				}
			}
		}
	}
}

// Route and transfer-fleet messages. Elegy's wording.
const (
	EventFleetGiven       EventKind = iota + EventOriginalDrift + 1 // Player = giver, Fleet = the new fleet, Count = recipient
	EventFleetReceived                                              // Player = recipient, Fleet = the new fleet, Count = giver
	EventFleetGiftRefused                                           // Player = giver, Fleet, Count = a GiftRefusal
	EventFleetGiftNoRoom                                            // Player = recipient, Fleet = the giver's fleet, Count = giver
)

// GiftRefusal is why a transfer-fleet task failed (EventFleetGiftRefused).
type GiftRefusal int

const (
	GiftRecipientAbsent GiftRefusal = iota
	GiftRefusedByRecipient
	GiftColonistsAboard
	GiftNoRoom
)

// routeTask runs a fleet's route task at a planet (ORDERS.md "Route
// task", CONFIRMED for the ideal-warp case): if the planet belongs to the
// fleet's owner and has a route destination, the fleet gets a fresh order
// to that planet carrying the route task again, so each arrival re-routes.
// The warp is the route warp new fleets use (PRODUCTION-LAUNCH.md, via
// routeWarp). Anywhere else the fleet stays idle. Who owns the
// destination does not matter (WU-ROUTE routed to another player's
// planet).
//
// ASSUMPTION W4: the task runs only for a fleet with no further waypoint
// (one that has reached the end of its orders), so a fleet already routed
// is not routed again before it leaves.
//
// Not modelled: the stargate choice (warp 11), as Elegy has no stargates.
func (g *Game) routeTask(f *Fleet) []Event {
	pi := g.planetAt(f.Pos)
	if pi < 0 || len(f.Waypoints) > 0 {
		return nil
	}
	p := &g.Planets[pi]
	if p.Owner != f.Owner || !p.HasRoute {
		return nil
	}
	dst := g.planetIndex(p.RouteTo)
	if dst < 0 || dst == pi {
		return nil
	}
	w := g.routeWarp(f, pi, dst)
	f.Waypoints = []Waypoint{{Pos: g.Planets[dst].Pos, Warp: w, Target: TargetPlanet, ID: p.RouteTo, Task: Task{Kind: TaskRoute}}}
	kind := EventFleetRouted
	if w == 0 {
		kind = EventFleetNotRouted
	}
	return []Event{{Kind: kind, Player: f.Owner, Planet: p.RouteTo, Fleet: f.ID, Count: w}}
}

// transferFleet runs a transfer-fleet task (ORDERS.md "Transfer fleet",
// PARITY.md "Transfer fleet task"). It is refused, the fleet keeping its
// owner, when:
//   - the recipient is not a live player (BINARY-ONLY);
//   - the recipient treats the giver as an enemy (CONFIRMED). A computer
//     player is refused by this check, as an expert computer is hostile;
//   - the fleet carries colonists (CONFIRMED);
//   - the recipient has no free or matching design slot for one of the
//     fleet's designs, or already has 512 fleets (LIMITS.md, MESSAGES.md
//     0x14a/0x14b, BINARY-ONLY).
//
// Otherwise the recipient gets a new fleet at the same place with the
// same ships, cargo and fuel, numbered with its lowest unused fleet
// number, and each design is held in a free slot of the recipient's
// holding a copy of it (CONFIRMED; not matched to a different existing
// design). WU-B2 (MEASURED) shows the new fleet as the recipient's first
// fleet, with no further waypoints and battle plan 0. Either way the task
// is cleared (MEASURED, WU-B2).
//
// ASSUMPTION W1: when several refusals apply, the first in the list
// above is reported.
// ASSUMPTION W2: a "matching" slot is one of the recipient's slots
// holding the same design (the same Game.Designs entry), which a copy
// never is; so every design takes a free slot.
// ASSUMPTION W3: the new fleet has no name and repeat orders off.
func (g *Game) transferFleet(fi int, gone map[int]bool) []Event {
	f := &g.Fleets[fi]
	to := f.Task.Player
	f.Task = Task{}
	refuse := func(r GiftRefusal) []Event {
		return []Event{{Kind: EventFleetGiftRefused, Player: f.Owner, Planet: -1, Fleet: f.ID, Count: int(r)}}
	}
	switch {
	case to < 0 || to >= len(g.Players) || to == f.Owner || g.Players[to].Dead:
		return refuse(GiftRecipientAbsent)
	case g.relation(to, f.Owner) == RelationEnemy:
		return refuse(GiftRefusedByRecipient)
	case f.Cargo.Colonists > 0:
		return refuse(GiftColonistsAboard)
	}
	fleets := 0
	for _, o := range g.Fleets {
		if o.Owner == to && !gone[o.ID] {
			fleets++
		}
	}
	used := map[int]bool{}
	held := map[int]bool{}
	for _, ds := range g.DesignSlots {
		if ds.Owner == to && !ds.Starbase {
			used[ds.Slot] = true
			held[ds.Design] = true
		}
	}
	var free []int
	for slot := range maxShipDesigns {
		if !used[slot] {
			free = append(free, slot)
		}
	}
	var need []int
	for _, s := range f.Stacks {
		if !held[s.Design] && !slices.Contains(need, s.Design) {
			need = append(need, s.Design)
		}
	}
	if fleets >= maxFleets || len(need) > len(free) {
		return append(refuse(GiftNoRoom), Event{Kind: EventFleetGiftNoRoom, Player: to, Planet: -1, Fleet: f.ID, Count: f.Owner})
	}
	from := f.Owner
	copied := map[int]int{}
	for k, d := range need {
		g.Designs = append(g.Designs, g.Designs[d])
		copied[d] = len(g.Designs) - 1
		g.DesignSlots = append(g.DesignSlots, DesignSlot{Owner: to, Slot: free[k], Design: copied[d]})
	}
	nf := Fleet{ID: g.newFleetID(), Number: g.lowestFreeFleetNumber(to), Owner: to, Pos: f.Pos,
		Fuel: f.Fuel, Cargo: f.Cargo}
	for _, s := range f.Stacks {
		if d, ok := copied[s.Design]; ok {
			s.Design = d
		}
		nf.Stacks = append(nf.Stacks, s)
	}
	gone[f.ID] = true
	g.Fleets = append(g.Fleets, nf)
	return []Event{
		{Kind: EventFleetGiven, Player: from, Planet: -1, Fleet: nf.ID, Count: to},
		{Kind: EventFleetReceived, Player: to, Planet: -1, Fleet: nf.ID, Count: from},
	}
}

// patrolRadius is the patrol's engage radius in ly (ORDERS.md "Patrol
// task", CONFIRMED: an enemy at 50 ly was engaged, one at 55 ly was not).
const patrolRadius = 50

// fileRetarget is the retarget made as each player's file is written
// (ORDERS.md "Targets that moved, died or were captured", BINARY-ONLY,
// seen in WU-A): a waypoint aimed at another player's fleet becomes a
// deep-space waypoint at that fleet's position, which the end-of-year
// waypoint check has just set. A waypoint aimed at the owner's own fleet
// stays a fleet target.
//
// ASSUMPTION W5: the written file is the game's state, so the waypoint
// no longer tracks the fleet in later years.
func (g *Game) fileRetarget() {
	for i := range g.Fleets {
		f := &g.Fleets[i]
		for j := range f.Waypoints {
			wp := &f.Waypoints[j]
			if wp.Target != TargetFleet {
				continue
			}
			if t := g.fleetIndex(wp.ID); t >= 0 && g.Fleets[t].Owner != f.Owner {
				wp.Target, wp.ID = TargetSpace, 0
			}
		}
	}
}

// patrol chooses each patrolling fleet's intercept at the end of the
// year, as the players' files are written (ORDERS.md "Patrol task",
// CONFIRMED for the choice, tie and warp; the timing BINARY-ONLY): a fleet
// whose waypoint-0 task is patrol and that has no other waypoint takes the
// nearest enemy fleet its owner sees this year within 50 ly, ties going to
// fleet order. The intercept is a waypoint aimed at that fleet, at its
// position, at warp min(10, range/5), carrying the patrol task (WU-SW50,
// WU-RNG20). The fleet moves the next year.
//
// ASSUMPTION P1: "enemy" is a fleet whose owner the patrolling player
// treats as an enemy.
func (g *Game) patrol(views []PlayerView) {
	order := g.fleetOrder()
	for _, i := range order {
		f := &g.Fleets[i]
		if f.Task.Kind != TaskPatrol || len(f.Waypoints) > 0 || f.Owner < 0 || f.Owner >= len(views) {
			continue
		}
		seen := map[int]bool{}
		for _, s := range views[f.Owner].Fleets {
			seen[s.Fleet] = true
		}
		best, bestD := -1, 0
		for _, j := range order {
			e := &g.Fleets[j]
			if !seen[e.ID] || g.relation(f.Owner, e.Owner) != RelationEnemy {
				continue
			}
			dx, dy := e.Pos.X-f.Pos.X, e.Pos.Y-f.Pos.Y
			if d := dx*dx + dy*dy; d <= patrolRadius*patrolRadius && (best < 0 || d < bestD) {
				best, bestD = j, d
			}
		}
		if best < 0 {
			continue
		}
		e := &g.Fleets[best]
		f.Waypoints = []Waypoint{{Pos: e.Pos, Warp: min(10, f.Task.Range/5), Target: TargetFleet, ID: e.ID, Task: f.Task}}
	}
}

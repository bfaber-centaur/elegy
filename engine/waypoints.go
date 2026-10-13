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
// fleet that no longer exists is resolved by targetGone. A planet target
// never moves.
//
// Each check first clamps every fleet's position, which is its waypoint
// 0, into the galaxy box, each axis on its own, with no message and no
// fuel (ORDERS.md "Waypoint upkeep", "Outside the galaxy"): CONFIRMED by
// prediction at the start of the year for a stationary fleet (ET-2c fleet
// K); at the end of the year BINARY-ONLY, fitting ET-2b fleet Q
// (MEASURED). ASSUMPTION: the checks after objects move (steps 3.2 and
// 5.1), which KERNEL.md names the same waypoint check, clamp too. Later
// waypoints are not clamped (BINARY-ONLY): one aimed at a packet outside
// the box keeps the packet's position. Ordered coordinates are clamped
// when ordered (ORDERS.md "Waypoint coordinates").
//
// A waypoint aimed at a wormhole end or a Mystery Trader takes the
// object's position the same way, and becomes a plain go-to when the
// object is gone ("Ordering with moving objects").
//
// So does a waypoint aimed at a mineral packet or salvage object
// (TargetPacketObject; ORDERS.md "Salvage and mineral packet targets",
// stars-elegy f2f9794, BINARY-ONLY): every check copies the object's
// position, so it follows a packet's move, and one that is gone becomes a
// plain go-to at its last coordinates with no message. The object is
// looked up by owner and number at each check, so a new object that took
// the freed number (numbers are reused, first gap first) is followed
// instead. While the object exists the target is kept when the players'
// files are written, whether or not the owner sees it, so fileRetarget
// leaves it alone (also MEASURED in RG-002: computer players' salvage
// targets came back in the next year's files).
//
// Not modelled: the exception bit that holds a waypoint's coordinates
// (BINARY-ONLY; no order sets it) and following fleets (step 1a.3): Elegy
// does not keep waypoint 0's target, which that rule reads.
func (g *Game) waypointCheck() {
	for i := range g.Fleets {
		g.Fleets[i].Pos = g.clampToGalaxy(g.Fleets[i].Pos)
	}
	for i := range g.Fleets {
		for j := range g.Fleets[i].Waypoints {
			wp := &g.Fleets[i].Waypoints[j]
			switch wp.Target {
			case TargetFleet:
				if t := g.fleetIndex(wp.ID); t >= 0 {
					wp.Pos = g.Fleets[t].Pos
				} else {
					g.targetGone(&g.Fleets[i], wp)
				}
			case TargetPacketObject:
				if p, ok := g.packetObjectPos(wp.Owner, wp.ID); ok {
					wp.Pos = p
				} else {
					wp.Target, wp.ID, wp.Owner = TargetSpace, 0, 0
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

// targetGone resolves wp, a waypoint of f aimed at a fleet that no longer
// exists (ORDERS.md "Targets that moved, died or were captured", "The
// target is gone"): it becomes the planet that stands exactly at its
// coordinates, else deep space there, and f's owner is owed 0x028, sent
// when the files are written (fileRetarget; MESSAGES.md 0x028, phase
// P8a). The coordinates are the target's last position: every waypoint
// check, and the settlement after movement (settleWaypoints), copies the
// target's position.
//
// A fleet destroyed in battle, or removed any other way, is resolved
// earlier, as it is removed (retargetRemoved); this is the rule for a
// waypoint left aimed at a removed fleet's id with nothing there, and for
// the file check.
//
// CONFIRMED by prediction for waypoint 1 aimed at a fleet the waypoint's
// owner destroyed in battle (WK-1, stars-elegy 5962b72, PARITY.md "A
// waypoint aimed at a fleet destroyed in battle (WK-1)", 2 of 2 streams):
// victims orbiting a planet left that planet;
// victims in deep space 5 ly from a planet left deep space at their
// position, not the nearest planet; victims that flew onto a planet and
// died there left that planet, not where they started; one 0x028 per
// follower. CONFIRMED by prediction for waypoints 1 to 3 and several
// followers of one victim (WK-2, stars-elegy d5b23e3, PARITY.md "Several
// followers, and waypoints 2 and up, aimed at a fleet destroyed in battle
// (WK-2)", 3 of 3 streams): each waypoint aimed at a victim is handled on
// its own, becomes deep space where it died, and owes one 0x028; a
// waypoint between two such is unchanged. The order of the 0x028
// messages is MEASURED there (follower order, then waypoint order, which
// Elegy's walk gives), but the rule behind it is UNRESOLVED: victim id
// and battle order fit as well. MEASURED for a target scrapped in deep
// space before movement (WF-04: deep space with 0x028). Other losses are
// BINARY-ONLY; waypoint 0, a kill by a third player, targets other than
// fleets and the planet case at waypoints 2 and up are NOT RUN.
func (g *Game) targetGone(f *Fleet, wp *Waypoint) {
	resolveGone(g, wp)
	g.goneTargets = append(g.goneTargets, f.ID)
}

// resolveGone makes wp, aimed at a fleet that no longer exists, the
// planet standing exactly at its coordinates, else deep space there.
func resolveGone(g *Game, wp *Waypoint) {
	wp.Target, wp.ID = TargetSpace, 0
	if pi := g.planetAt(wp.Pos); pi >= 0 {
		wp.Target, wp.ID = TargetPlanet, g.Planets[pi].ID
	}
}

// chaseMerged is the waypoint check's retarget of a waypoint that names
// a fleet a merge order, or a move-ships order that emptied it, removed
// earlier in the replay (merged: fleet id to owner), when the waypoint's
// own order came later (ORDERS.md "Targets
// that moved, died or were captured", "A fleet target merged away during
// order replay", BINARY-ONLY). The waypoint takes a fleet of the merged
// fleet's owner that stands exactly at its coordinates; with none it keeps
// the stale id, and waypointCheck makes it a plain go-to there. Each
// candidate costs a draw: rand(k) for the k-th, plus a rand(2) for a
// candidate some waypoint already picked this year.
//
// ASSUMPTION W8: ORDERS.md prefers a candidate that the chasing fleet's
// battle plan's primary target matches, by a weight it does not define.
// Elegy does not model that preference: the k-th candidate in fleet order
// is taken when its rand(k) is 0, a uniform choice, and the extra rand(2)
// is drawn without changing the choice. Waypoints are visited in fleet
// order, and a fleet is never its own candidate for a waypoint (a
// follower can be, below). During a generation a fleet made later in
// the replay never takes the merged fleet's id (Game.newFleetID).
func (g *Game) chaseMerged(merged, follows map[int]int, rng Rand) {
	if len(merged) == 0 {
		return
	}
	order := g.fleetOrder()
	picked := map[int]bool{}
	// pick draws for the fleets of owner at pos, other than fleet index i
	// (-1 for none), and returns the chosen fleet's id, or -1.
	pick := func(i, owner int, pos Point) int {
		choice, k := -1, 0
		for _, c := range order {
			cf := &g.Fleets[c]
			if c == i || cf.Owner != owner || cf.Pos != pos {
				continue
			}
			k++
			if rng.Intn(k) == 0 {
				choice = c
			}
			if picked[cf.ID] {
				rng.Intn(2)
			}
		}
		if choice < 0 {
			return -1
		}
		picked[g.Fleets[choice].ID] = true
		return g.Fleets[choice].ID
	}
	for _, i := range order {
		f := &g.Fleets[i]
		// A follower's waypoint 0, at its own position, aimed at a fleet
		// merged away earlier in the replay (ORDERS.md "Waypoint 0,
		// following", stars-elegy a1c4c91, BINARY-ONLY): the follow step
		// then follows the fleet chosen, or sends 0x138 with none. The
		// follower is a candidate when it has the merged fleet's owner,
		// and then follows itself (KERNEL.md step 1a.3, "possibly the
		// follower itself"; F3).
		if leader, ok := follows[f.ID]; ok && g.fleetIndex(leader) < 0 {
			if owner, ok := merged[leader]; ok {
				if id := pick(-1, owner, f.Pos); id >= 0 {
					follows[f.ID] = id
				}
			}
		}
		for j := range f.Waypoints {
			wp := &f.Waypoints[j]
			owner, ok := merged[wp.ID]
			if wp.Target != TargetFleet || !ok || g.fleetIndex(wp.ID) >= 0 {
				continue
			}
			if id := pick(i, owner, wp.Pos); id >= 0 {
				wp.ID = id
			}
		}
	}
}

// retargetRemoved retargets the waypoints that name a fleet in ids, which
// is about to be removed (merged, scrapped, colonized, given away, emptied,
// or destroyed in battle or by a space object), to what stands exactly at
// that fleet's position: a fleet of the same owner that stays (the first
// in fleet order), else the planet there (ORDERS.md "Targets that moved,
// died or were captured", "A fleet target merged away during order
// replay": "Removing a fleet retargets every waypoint that names it, any
// player's ... else the planet there, else deep space", BINARY-ONLY). The
// waypoint takes that position. No draws.
//
// MEASURED in RG-002 (lane B's triage, 8 of 8 seeds): a fleet removed
// at planet 79 (Y3) and one destroyed in battle at planet 51 (Y4) left
// the waypoints chasing them aimed at the planet.
//
// ASSUMPTION W10: only the removed fleet's owner's fleets are candidates.
// In Y4 two of the chasing player's own fleets stood at planet 51 and the
// planet was taken. The spec's "the replaying player's own fleets
// preferred" is the merge case, where that player owns the removed fleet.
//
// A fleet destroyed in battle has no stand-in (standIns false): MEASURED
// in RG-003 (stars-elegy b0f7dfa), year 7 (2469): player 5's fleet 15
// died in battle at planet 83, where player 5's fleet 1 stayed (MEASURED
// at the end of the year), and player 5's fleet 45, whose waypoint chased
// fleet 15, came back aimed at planet 83, not fleet 1; in year 8 (2470)
// player 1's five fleets chasing player 5's fleet 71, which died at
// planet 94 where player 5's fleet 74 had orbited, came back aimed at
// planet 94 (the vectors' planet numbers). WK-1 (CONFIRMED by
// prediction) had no surviving fleet of the victims' owner. Other
// removals keep W10.
//
// With nothing there the waypoint keeps the removed fleet's id, so the
// "target gone" rule (targetGone) makes it deep space at its last
// position at the next waypoint check, with 0x028 when the files are
// written (MEASURED WF-04: a target scrapped in deep space). Deep space is
// the spec's third case either way.
//
// Retargeted to the planet, the chasing fleet's owner is owed 0x028 too,
// sent when the files are written (fileRetarget): CONFIRMED by prediction
// for a target destroyed in battle (WK-1, 2 of 2 streams: one 0x028 per
// follower, the waypoint the planet where the target died, after
// movement); for the other removals BINARY-ONLY (ORDERS.md "The target is
// gone"). Several followers of one removed fleet each owe their own
// 0x028: CONFIRMED by prediction for a fleet destroyed in battle (WK-2,
// stars-elegy d5b23e3, 3 of 3 streams), BINARY-ONLY for the other
// removals. ASSUMPTION W10: a
// waypoint taken over by a stand-in fleet owes no 0x028.
func (g *Game) retargetRemoved(ids map[int]bool, standIns bool) {
	order := g.fleetOrder()
	for _, i := range order {
		if ids[g.Fleets[i].ID] {
			continue
		}
		for j := range g.Fleets[i].Waypoints {
			wp := &g.Fleets[i].Waypoints[j]
			if wp.Target != TargetFleet || !ids[wp.ID] {
				continue
			}
			gone := &g.Fleets[g.fleetIndex(wp.ID)]
			retarget := func(kind TargetKind, id int) {
				wp.Pos, wp.Target, wp.ID = gone.Pos, kind, id
			}
			if fi := g.standIn(order, ids, gone, i); standIns && fi >= 0 {
				retarget(TargetFleet, g.Fleets[fi].ID)
			} else if pi := g.planetAt(gone.Pos); pi >= 0 {
				retarget(TargetPlanet, g.Planets[pi].ID)
				g.goneTargets = append(g.goneTargets, g.Fleets[i].ID)
			}
		}
	}
}

// standIn is the fleet (index) that takes over from gone, a fleet in ids
// about to be removed, for the fleet at index chaser: the first in fleet
// order of gone's owner standing exactly at gone's position, not chaser
// and not being removed itself, or -1 (retargetRemoved; ASSUMPTION W10).
func (g *Game) standIn(order []int, ids map[int]bool, gone *Fleet, chaser int) int {
	for _, c := range order {
		cf := &g.Fleets[c]
		if c != chaser && !ids[cf.ID] && cf.Owner == gone.Owner && cf.Pos == gone.Pos {
			return c
		}
	}
	return -1
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
// planet). The gate choice (warp 11) is the route warp's (MEASURED,
// ORDERS.md "Route task", wuRSG2).
//
// ASSUMPTION W4: the task runs only for a fleet with no further waypoint
// (one that has reached the end of its orders), so a fleet already routed
// is not routed again before it leaves.
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
//   - the recipient treats the giver as an enemy (CONFIRMED), or is a
//     computer player, which is hostile when the gift is evaluated
//     whatever its stored relation (CONFIRMED, WU-AICOMP3/4: neutral);
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
	case g.relation(to, f.Owner) == RelationEnemy || g.Players[to].Computer:
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
		g.DesignSlots = append(g.DesignSlots, DesignSlot{Owner: to, Slot: free[k], Design: copied[d], Received: true})
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

const (
	EventTargetGone      EventKind = iota + EventRemoteMineDeepSpace + 1 // Fleet: a waypoint's target fleet is gone; it goes to the last position (0x028)
	EventTargetAtPlanet                                                  // Fleet, Planet: a target fleet out of sight at a planet; the waypoint goes to the planet (0x029)
	EventTargetOutOfView                                                 // Fleet: a target fleet out of sight in deep space; the waypoint goes to its position (0x02a)
)

// fileRetarget is the check of waypoints aimed at another player's fleet,
// friend or enemy, as the year's files are written (ORDERS.md "Targets
// that moved, died or were captured"; MESSAGES.md 0x028–0x02a). A target
// the waypoint owner's scanners see at the end of the year stays the
// target, in the game and in the owner's file. One out of every scanner
// becomes a plain waypoint at its actual end-of-year position (0x02a),
// or at its planet when it is at one (0x029, BINARY-ONLY). A target that
// is gone becomes a plain waypoint at its last position (0x028).
//
// MEASURED W5 (PARITY "Representative game (RG-001)" WF-01..05, one
// stream, stars-elegy 30272be; ORDERS.md "Contradicted in part"): a seen
// target, enemy or friend, stayed the target at its end-of-year position
// in the host file and the owner's file over two years; an unseen one
// became space at its actual end-of-year position with 0x02a; one
// scrapped before movement became space at its last position with 0x028.
func (g *Game) fileRetarget(views []PlayerView) []Event {
	var events []Event
	// The year's 0x028 owed by targetGone and retargetRemoved, in the
	// order they were owed, to the follower's owner now.
	for _, id := range g.goneTargets {
		if fi := g.fleetIndex(id); fi >= 0 {
			events = append(events, g.fleetEvent(&g.Fleets[fi], EventTargetGone, 0))
		}
	}
	g.goneTargets = nil
	for i := range g.Fleets {
		f := &g.Fleets[i]
		var seen map[int]bool
		for j := range f.Waypoints {
			wp := &f.Waypoints[j]
			if wp.Target != TargetFleet {
				continue
			}
			t := g.fleetIndex(wp.ID)
			if t < 0 {
				resolveGone(g, wp)
				events = append(events, g.fleetEvent(f, EventTargetGone, 0))
				continue
			}
			if g.Fleets[t].Owner == f.Owner {
				continue
			}
			if seen == nil {
				seen = map[int]bool{}
				if f.Owner >= 0 && f.Owner < len(views) {
					for _, s := range views[f.Owner].Fleets {
						seen[s.Fleet] = true
					}
				}
			}
			if seen[wp.ID] {
				continue
			}
			wp.Pos = g.Fleets[t].Pos
			if pid, ok := g.OrbitedPlanet(&g.Fleets[t]); ok {
				wp.Target, wp.ID = TargetPlanet, pid
				e := g.fleetEvent(f, EventTargetAtPlanet, 0)
				e.Planet = pid
				events = append(events, e)
				continue
			}
			wp.Target, wp.ID = TargetSpace, 0
			events = append(events, g.fleetEvent(f, EventTargetOutOfView, 0))
		}
	}
	return events
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

// TargetPacketObject is a waypoint's mineral packet or salvage object:
// Waypoint.Owner's object number Waypoint.ID. Packets and salvage are one
// kind of object in the original's files and share each owner's numbers
// (OBJECTS.md "Salvage", "Owner"; vectors README "packet_object").
const TargetPacketObject TargetKind = TargetSalvage + 1

// packetObjectPos is where owner's salvage or packet number is now.
func (g *Game) packetObjectPos(owner, number int) (Point, bool) {
	for _, s := range g.Salvage {
		if s.Owner == owner && s.Number == number {
			return s.Pos, true
		}
	}
	if g.Objects != nil {
		return g.Objects.PacketPos(owner, number)
	}
	return Point{}, false
}

// EventTransportNoHold: Player's Fleet reached a waypoint aimed at an
// object that holds no cargo with a transport task, and the task was
// cancelled (MESSAGES.md 0x11e: a minefield, a wormhole or the Trader,
// stars-elegy a1c4c91). Elegy's wording. Not sent yet: those targets are
// not modelled. A packet holds cargo (loadPacket).
const EventTransportNoHold EventKind = EventArtifactFound + 1

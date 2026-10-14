package engine

// Following another fleet (stars-elegy ORDERS.md "Waypoint 0 aimed at a
// fleet", KERNEL.md "Turn order" step 1a.3, MESSAGES.md 0x137 and 0x138;
// stars-elegy d4eafd9).
//
// Only an order makes a fleet follow another: its only waypoint, waypoint
// 0, names the leader. The follow lasts one year: after the year's
// generation the follower has one waypoint at its own position, aimed at
// a planet or deep space (ORDERS.md, MEASURED for the computer players'
// orders, AI-27; FO-04). Elegy keeps the link only in the year's replay
// state (Applied.follows), never in Game, so no saved game holds a
// waypoint 0 aimed at a fleet.

// Follow messages. Elegy's wording.
const (
	// EventFollowNoLeader: Player's Fleet was told to follow a fleet that
	// no longer exists, or has no orders of its own and is not following
	// anyone; it stops following (MESSAGES.md 0x138, CONFIRMED for
	// followers at their leader's position: fo/fo04, fo/fo02, fo/fo03
	// A-E).
	EventFollowNoLeader EventKind = iota + EventLoadWaiting + 1
	// EventFollowDone: Player's Fleet, given a follow order, has followed
	// and awaits orders; its follow step is dropped whether or not it
	// caught up (MESSAGES.md 0x137, end of movement). CONFIRMED for
	// followers at the position of a leader that is not itself a follower
	// (fo/fo02, fo/fo03 A-E), and for a follower that merged into its
	// leader before movement and no longer exists (fo/fo03 G).
	EventFollowDone
)

// maxFollowPasses is how many passes resolve chains of followers
// (KERNEL.md "Turn order" step 1a.3; MESSAGES.md 0x138).
const maxFollowPasses = 8

// FollowOrder makes Fleet follow Leader this year: Fleet's waypoints are
// replaced by its waypoint 0 aimed at Leader, carrying Task (ORDERS.md
// "Waypoint 0 aimed at a fleet": "a waypoint-0 change that names another
// fleet, which leaves the fleet with that one waypoint"). Leader is any
// other fleet, of any owner (KERNEL.md names only "another fleet").
//
// The leader is not checked: a waypoint edit checks only the ordering
// fleet and the index (ORDERS.md "Waypoint 0, following", stars-elegy
// a1c4c91, BINARY-ONLY). A leader merged away, or emptied by a move-ships
// order, earlier in the replay keeps its stale id, and the waypoint check
// before the follow step retargets it (Game.chaseMerged); one removed
// later in the replay is retargeted when it is removed
// (Applied.retargetFollows), as for any waypoint (ORDERS.md "A fleet
// target merged away during order replay"). A leader that is gone at the
// follow step, or never existed, gets the follower 0x138; the fleet
// itself as leader leaves it following itself (F3).
type FollowOrder struct {
	Fleet  int
	Leader int
	Task   Task
}

func (o FollowOrder) apply(g *Game, player int, a *Applied) error {
	i, err := g.ownFleet(player, o.Fleet)
	if err != nil {
		return err
	}
	if err := validTask(o.Task); err != nil {
		return err
	}
	f := &g.Fleets[i]
	f.Task = o.Task
	if f.Task.Kind == TaskTransport {
		// The task acts on the leader (CONFIRMED FO-02 R; FO-02 L to Q,
		// FO-04, FO-05 A to C).
		f.Task.ToFleet, f.Task.Fleet = true, o.Leader
	}
	f.Waypoints = nil
	if a.follows == nil {
		a.follows = map[int]int{}
	}
	a.follows[f.ID] = o.Leader
	return nil
}

// follow is step 1a.3 (KERNEL.md "Turn order"). First a leader that is
// not at its follower's position is re-chosen there (followLeader). Then,
// in up to 8 passes over the fleets in fleet order, a follower whose
// leader has a next waypoint gets a copy of it carrying the follower's
// own task (the copying rule BINARY-ONLY); one whose leader is gone, or
// has no orders and is not following, gets 0x138 and stops following; one
// whose leader is a follower still unresolved waits for a later pass. It
// returns the 0x137 messages, in fleet order, for the followers that
// followed; endFollow sends them after movement. They are made here so
// that a follower merged away before movement still gets its own
// (MESSAGES.md 0x137, CONFIRMED fo/fo03 G).
//
// ASSUMPTION F2: within a pass the fleets are taken one at a time, so a
// follower resolved earlier in the pass is a leader with a next waypoint
// for the rest of it.
//
// F3: a follower still unresolved after the passes gets no message, no
// waypoint and no 0x137, so it does not move. CONFIRMED for a follower
// left following itself (FO-03-F: no move, no 0x138, no 0x137);
// ASSUMPTION for a cycle of two or more fleets and a chain longer than
// the passes reach. MESSAGES.md says circular chains get no message.
func (g *Game) follow(follows map[int]int) (done []Event, events []Event) {
	leaders := map[int]int{}
	for id, leader := range follows {
		if i := g.fleetIndex(id); i >= 0 {
			leaders[id] = g.followLeader(&g.Fleets[i], leader)
		}
	}
	pending := map[int]bool{}
	for id := range leaders {
		pending[id] = true
	}
	order := g.fleetOrder()
	followed := map[int]bool{}
	for pass := 0; pass < maxFollowPasses && len(pending) > 0; pass++ {
		for _, i := range order {
			f := &g.Fleets[i]
			if !pending[f.ID] {
				continue
			}
			l := g.fleetIndex(leaders[f.ID])
			switch {
			case l >= 0 && pending[g.Fleets[l].ID]:
				continue
			case l >= 0 && len(g.Fleets[l].Waypoints) > 0:
				wp := g.Fleets[l].Waypoints[0]
				wp.Task = f.Task
				f.Waypoints = []Waypoint{wp}
				followed[f.ID] = true
			default:
				events = append(events, g.fleetEvent(f, EventFollowNoLeader, 0))
			}
			delete(pending, f.ID)
		}
	}
	for _, i := range order {
		if f := &g.Fleets[i]; followed[f.ID] {
			done = append(done, g.fleetEvent(f, EventFollowDone, 0))
		}
	}
	return done, events
}

// followLeader is f's leader at step 1a.3 (KERNEL.md "Turn order" step
// 1a.3, MESSAGES.md 0x138; the re-choice BINARY-ONLY, its outcome
// CONFIRMED FO-03-F). A leader at f's position, or one that no longer
// exists, stays. Any other is replaced by a fleet of the leader's owner
// at f's position, which can be f itself; f then follows itself and is
// never resolved (F3): no move, no 0x138, no 0x137. MESSAGES.md names the leader's owner, KERNEL.md
// "the same owner"; Elegy follows MESSAGES.md. A Merge with Fleet task
// aimed at the old leader is aimed at the new one with it, so a follower
// left following itself completes its merge with itself, merging nothing
// (FO-03-F: no merge, 0x04e, task cleared). So is a transport task
// (ASSUMPTION T21); one left aimed at the follower itself has no target
// (transportTarget).
//
// ASSUMPTION F5: when several fleets qualify, the first in fleet order is
// chosen.
//
// ASSUMPTION F6: when none does (a leader of another player, elsewhere,
// with no fleet at f's position), f follows nothing, as when left
// following itself: no move, no 0x138, no 0x137 (MESSAGES.md 0x138: "A
// follower whose leader is elsewhere does not get it"). Its task is left
// as it is.
func (g *Game) followLeader(f *Fleet, leader int) int {
	l := g.fleetIndex(leader)
	if l < 0 || g.Fleets[l].Pos == f.Pos {
		return leader
	}
	owner := g.Fleets[l].Owner
	for _, i := range g.fleetOrder() {
		if c := &g.Fleets[i]; c.Owner == owner && c.Pos == f.Pos {
			if (f.Task.Kind == TaskMerge || f.Task.ToFleet) && f.Task.Fleet == leader {
				f.Task.Fleet = c.ID
			}
			return c.ID
		}
	}
	return f.ID
}

// endFollow ends the year's follows at the end of movement (MESSAGES.md
// 0x137): every follower that followed drops its follow step, reached or
// not, and is told, even one merged away before movement (done, from
// follow). The follow does not persist (ORDERS.md "Waypoint 0 aimed at a
// fleet": the follower is left with its own position).
//
// ASSUMPTION F4: a follower that reached the copied waypoint has already
// had Elegy's arrival message (EventFleetArrived) from the waypoint
// settlement; 0x137 comes as well.
func (g *Game) endFollow(done []Event) []Event {
	for _, e := range done {
		if i := g.fleetIndex(e.Fleet); i >= 0 {
			g.Fleets[i].Waypoints = nil
		}
	}
	return done
}

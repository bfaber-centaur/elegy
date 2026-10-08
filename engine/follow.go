package engine

import "fmt"

// Following another fleet (stars-elegy ORDERS.md "Waypoint 0 aimed at a
// fleet", KERNEL.md "Turn order" step 1a.3, MESSAGES.md 0x137 and 0x138;
// stars-elegy c3aab85).
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
	// anyone; it stops following (MESSAGES.md 0x138, CONFIRMED fo/fo04).
	EventFollowNoLeader EventKind = iota + EventLoadWaiting + 1
	// EventFollowDone: Player's Fleet, given a follow order, has followed
	// and awaits orders; its follow step is dropped whether or not it
	// caught up (MESSAGES.md 0x137, end of movement, CONFIRMED fo/fo02,
	// fo/fo03).
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
// ASSUMPTION F1: a leader that does not exist when the order applies,
// including one merged away earlier in the replay, rejects the order, as
// a waypoint naming a target that does not exist does (ORDERS.md "Range
// and legality clamps", chosen rule); one merged away later in the
// replay is gone at
// step 1a.3 and the follower gets 0x138. ORDERS.md "A fleet target merged
// away during order replay" retargets waypoints, but says nothing of
// waypoint 0.
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
	if l := g.fleetIndex(o.Leader); l < 0 || l == i {
		return fmt.Errorf("follow: fleet %d: %w", o.Leader, ErrNoSuchObject)
	}
	f := &g.Fleets[i]
	f.Task = o.Task
	f.Waypoints = nil
	if a.follows == nil {
		a.follows = map[int]int{}
	}
	a.follows[f.ID] = o.Leader
	return nil
}

// follow is step 1a.3 (KERNEL.md "Turn order", the copying rule
// BINARY-ONLY): in up to 8 passes over the fleets in fleet order, a
// follower whose leader has a next waypoint gets a copy of it carrying
// the follower's own task; one whose leader is gone, or has no orders
// and is not following, gets 0x138 and stops following; one whose leader
// is a follower still unresolved waits for a later pass. It returns the
// fleets still following, which take EventFollowDone after movement
// (endFollow).
//
// ASSUMPTION F2: within a pass the fleets are taken one at a time, so a
// follower resolved earlier in the pass is a leader with a next waypoint
// for the rest of it.
//
// ASSUMPTION F3: a follower still unresolved after the passes (a circular
// chain, or a chain longer than the passes reach) gets no message and no
// waypoint, so it does not move, and is still following for 0x137.
// MESSAGES.md says only that circular chains get no 0x138.
func (g *Game) follow(follows map[int]int) (following map[int]bool, events []Event) {
	following = map[int]bool{}
	pending := map[int]bool{}
	for id := range follows {
		if g.fleetIndex(id) >= 0 {
			pending[id] = true
		}
	}
	order := g.fleetOrder()
	for pass := 0; pass < maxFollowPasses && len(pending) > 0; pass++ {
		for _, i := range order {
			f := &g.Fleets[i]
			if !pending[f.ID] {
				continue
			}
			l := g.fleetIndex(follows[f.ID])
			switch {
			case l >= 0 && pending[g.Fleets[l].ID]:
				continue
			case l >= 0 && len(g.Fleets[l].Waypoints) > 0:
				wp := g.Fleets[l].Waypoints[0]
				wp.Task = f.Task
				f.Waypoints = []Waypoint{wp}
				following[f.ID] = true
			default:
				events = append(events, g.fleetEvent(f, EventFollowNoLeader, 0))
			}
			delete(pending, f.ID)
		}
	}
	for id := range pending {
		following[id] = true
	}
	return following, events
}

// endFollow ends the year's follows at the end of movement (MESSAGES.md
// 0x137): every fleet still following drops its follow step, reached or
// not, and is told. The follow does not persist (ORDERS.md "Waypoint 0
// aimed at a fleet": the follower is left with its own position).
//
// ASSUMPTION F4: a follower that reached the copied waypoint has already
// had Elegy's arrival message (EventFleetArrived) from the waypoint
// settlement; 0x137 comes as well.
func (g *Game) endFollow(following map[int]bool) []Event {
	var events []Event
	for _, i := range g.fleetOrder() {
		f := &g.Fleets[i]
		if following[f.ID] {
			f.Waypoints = nil
			events = append(events, g.fleetEvent(f, EventFollowDone, 0))
		}
	}
	return events
}

package engine

// Transport tasks aimed at a fleet, and transport fuel (stars-elegy
// ORDERS.md "Cargo amounts and clamps", "Waypoint 0 aimed at a fleet";
// TAKEOVER.md "Unload and load amounts"; PARITY.md "Fleet Operations",
// FO-01 to FO-05; KERNEL.md "Fuel cannot be unloaded onto a planet").
//
// A transport task whose waypoint 0 is aimed at a fleet (Task.ToFleet)
// unloads into it and loads from it, with the amounts of TAKEOVER.md for
// a planet, A being what the fleet holds. Minerals and colonists are
// clamped by the receiving fleet's free cargo hold, fuel by its free fuel
// tank, independently, and what does not fit stays with the giver
// (CONFIRMED, FO-02 L to P). The unloads run in the unload passes and the
// loads in the load passes, before movement (CONFIRMED FO-02 R) and after
// it (FO-02 S, U), as for a planet.

// Messages for a transport task aimed at another player's fleet. Elegy's
// wording.
const (
	// EventFleetLoadRefused: Player's Fleet could not load cargo type
	// Axes[0] from another player's fleet Axes[1]; the task is cancelled
	// (MESSAGES.md 0x120, CONFIRMED fo/fo04). Sent after movement only.
	EventFleetLoadRefused EventKind = iota + EventPlanetCaptured + 1
	// EventColonistsToForeignFleet: Player's Fleet could not give
	// colonists to another player's fleet Axes[0]; the task is cancelled
	// and the colonists stay aboard (MESSAGES.md 0x155, CONFIRMED
	// fo/fo04).
	EventColonistsToForeignFleet
)

// transportTarget is the index of the fleet f's transport task is aimed
// at, or -1 when that fleet is gone or not at f's position.
//
// ASSUMPTION T16: a target fleet that is gone or elsewhere, or the fleet
// itself (a follower left following itself, F3), takes nothing
// and gives nothing. Unloads then clear moving nothing, and loads settle
// as from deep space with no salvage (noSource): satisfied when they want
// nothing, waiting before movement, cancelled after it with 0x123. A
// follower's leader elsewhere is first re-chosen at its position
// (followLeader), which retargets the task.
func (g *Game) transportTarget(f *Fleet) int {
	t := g.fleetIndex(f.Task.Fleet)
	if t < 0 || t == g.fleetIndex(f.ID) || g.Fleets[t].Pos != f.Pos {
		return -1
	}
	return t
}

// capacityFor is a fleet's capacity for cargo type c: its fuel tank for
// fuel, else its cargo hold.
//
// ASSUMPTION T22: fuel takes every transport action against the tank as
// the other types do against the hold, so "fill to v%" fuel targets
// ⌊v·tank/100⌋ (T8). Only "load all", "unload all" and "unload exactly"
// fuel were run (FO-01 E, FO-02 M, O; FM-102-N).
func (g *Game) capacityFor(f *Fleet, c int) int {
	if c == CargoFuel {
		return g.tankCapacity(f)
	}
	return g.cargoCapacity(f)
}

// unloadToFleet runs the unload actions of f's transport task aimed at a
// fleet, cargo types in order. "Unload all" offers C, "unload exactly v"
// min(v, C), and the unload direction of "set amount to" and "set
// waypoint to" what unloadWant gives against the target's holding; the
// target takes what fits (CONFIRMED FO-02 N, O for minerals and fuel;
// colonists to an own fleet, FO-02 P). The actions then clear, so they
// happen once, as at a planet.
//
// To another player's fleet (PARITY.md "Another player's fleet",
// CONFIRMED FO-04, FO-05): minerals move when the receiver's relation
// toward the giver is not enemy, and otherwise nothing moves, silently;
// the giver's own relation does not matter. Colonists are refused with
// 0x155 and the task is cancelled (MESSAGES.md 0x155).
//
// ASSUMPTION T17: fuel to another player's fleet follows the minerals'
// relation rule. ASSUMPTION T18: the colonists' refusal cancels the whole
// task where it is reached in cargo order, so the minerals before it have
// moved and the fuel after it does not.
func (g *Game) unloadToFleet(f *Fleet) []Event {
	ti := g.transportTarget(f)
	for c := range NumTransport {
		t := &f.Task.Transport[c]
		have := *held(f, c)
		amount := 0
		switch t.Action {
		case UnloadAll:
			amount = have
		case UnloadExactly:
			amount = min(t.Amount, have)
		case SetAmount, SetWaypoint:
			// A load, or nothing to move: the load pass settles it
			// (ASSUMPTION T11).
			if amount = t.unloadWant(have, g.targetHolds(f, c)); amount <= 0 {
				continue
			}
		default:
			continue
		}
		t.Action, t.Amount = TransportNone, 0
		if ti < 0 || amount <= 0 {
			continue
		}
		to := &g.Fleets[ti]
		if to.Owner != f.Owner {
			if c == CargoColonists {
				f.Task = Task{}
				return []Event{{Kind: EventColonistsToForeignFleet, Player: f.Owner, Planet: -1, Fleet: f.ID, Axes: []int{to.ID}}}
			}
			if g.relation(to.Owner, f.Owner) == RelationEnemy {
				continue
			}
		}
		amount = min(amount, g.free(to, c))
		*held(f, c) -= amount
		*held(to, c) += amount
	}
	if f.Task.Transport == ([NumTransport]Transport{}) {
		f.Task = Task{}
	}
	return nil
}

// loadFromFleet runs the load actions of f's transport task aimed at a
// fleet, cargo types in order, each against its own capacity: what
// loadWant gives, capped by what the target holds (CONFIRMED FO-02 L:
// "load all" Ir and Bo into a 110 kT free hold took 110 Ir, then no Bo;
// FO-02 M: a full hold still took 250 mg into the free tank). loaded
// settles each action, as at a planet, and settled the task.
//
// Another player's fleet is not a source for a fleet that cannot steal
// cargo (KERNEL.md "Loads with no usable source"; MEASURED FO-04 C, FO-05
// C: nothing loaded, 0x120, the task ended the same year): a load that
// wants more waits before movement and is cancelled after it. A fleet
// that can steal from fleets is not modelled; ASSUMPTION T10 holds for it
// as at a planet: its loads there wait and are never cancelled.
//
// ASSUMPTION T19: the cancel sends 0x120 for a mineral and nothing for
// colonists or fuel (MESSAGES.md 0x124, 0x125: never sent; "in the real
// turn the task is cancelled silently", BINARY-ONLY).
func (g *Game) loadFromFleet(f *Fleet, after bool) []Event {
	ti := g.transportTarget(f)
	if ti < 0 {
		return g.noSource(f, after, -1)
	}
	src := &g.Fleets[ti]
	if src.Owner != f.Owner {
		cancel := after && !g.stealsFromFleets(f)
		ev := g.noSource(f, cancel, -1)
		for i, e := range ev {
			if e.Kind == EventLoadRefused {
				ev = ev[:i]
				if c := e.Axes[0]; c < NumMinerals {
					ev = append(ev, Event{Kind: EventFleetLoadRefused, Player: f.Owner, Planet: -1, Fleet: f.ID, Axes: []int{c, src.ID}})
				}
				break
			}
		}
		return ev
	}
	for c := range NumTransport {
		t := &f.Task.Transport[c]
		capacity := g.capacityFor(f, c)
		avail := held(src, c)
		mine := held(f, c)
		want, ok := t.loadWant(capacity, *mine, g.free(f, c), *avail)
		if !ok {
			continue
		}
		amount := max(0, min(*avail, want))
		*avail -= amount
		*mine += amount
		t.loaded(capacity, *mine, *avail, g.free(f, c))
	}
	return g.settled(f, after, -1)
}

// stealsFromFleets reports a fleet with a part that steals cargo from
// fleets (COMPONENTS.md steals_cargo "fleets" or "fleets_and_planets":
// the Pick Pocket and Robber Baron Scanners), for which another player's
// fleet is a source (KERNEL.md "Loads with no usable source").
func (g *Game) stealsFromFleets(f *Fleet) bool {
	for _, st := range f.Stacks {
		for _, sl := range g.Designs[st.Design].Parts() {
			if st.Count > 0 && sl.Count > 0 && sl.Part.CargoScan {
				return true
			}
		}
	}
	return false
}

// transportFuelIgnored clears the fuel action of f's transport task
// when it is not aimed at a fleet: a planet neither takes nor supplies
// fuel, and deep space behaves the same way, every fuel action moving no
// fuel, sending no message and debiting nothing (TAKEOVER.md, CONFIRMED
// MG-002, MG-004, MG-006; KERNEL.md "Fuel cannot be unloaded onto a
// planet": the unload is cleared as usual, CONFIRMED FM-101..105; FO-01
// E: no fuel from an own planet without a starbase).
//
// ASSUMPTION T20: a fuel load is cleared too, as a load that wants
// nothing, rather than left to wait; and salvage and a packet are as
// deep space for fuel.
func (g *Game) transportFuelIgnored(f *Fleet) {
	f.Task.Transport[CargoFuel] = Transport{}
	if f.Task.Kind == TaskTransport && f.Task.Transport == ([NumTransport]Transport{}) {
		f.Task = Task{}
	}
}

// dropFleetTargets turns every waypoint 0 still aimed at a fleet into
// the planet or deep space there as the year's files are written
// (ORDERS.md "Waypoint 0 aimed at a fleet", BINARY-ONLY): a transport
// task left over acts there from next year.
func (g *Game) dropFleetTargets() {
	for i := range g.Fleets {
		if t := &g.Fleets[i].Task; t.ToFleet {
			t.ToFleet, t.Fleet = false, 0
		}
	}
}

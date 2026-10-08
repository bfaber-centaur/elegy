package engine

import "fmt"

// Ship moves between the player's own fleets (ORDERS.md "Split",
// CONFIRMED CO-01, CO-02, CO-03; FC-1).

// maxExchangeStack is the most ships of one design a ship move leaves in a
// fleet (ORDERS.md "Merge", MEASURED CO-06 for the ship-move / exchange
// path): the ships above it are lost and the order is not refused.
const maxExchangeStack = 32765

// SplitOrder makes a new fleet of the player's from the given ships of
// fleet Fleet (ORDERS.md "Split", CONFIRMED CO-01, CO-02): the new fleet
// carries exactly those ships, inherits the source's battle plan and its
// full waypoint list (with the task at its current location), and takes
// the share of the source's cargo and fuel that a ship move carries
// (MoveShipsOrder). Split All is one SplitOrder per new fleet, each
// taking its share of what is left, so the rounding remainder stays on
// the source (CO-02).
//
// ASSUMPTION L20: the new fleet takes its owner's lowest unused fleet
// number, as a new ship's fleet does (PRODUCTION-LAUNCH.md "The new
// fleet"); CO-02 and FC-1 agree. A player at the 512-fleet limit cannot
// split (refused with ErrOutOfRange).
//
// ASSUMPTION L21: the new fleet also takes the source's repeat-orders
// flag, which belongs to its waypoint list (ORDERS.md "Reaching a
// waypoint"), and has no name.
//
// NewFleet names the new fleet for the rest of the order file: when it is
// below 0, a later order in the same file may give that value wherever it
// names a fleet (its own fleet, a merge or move partner, a cargo or
// waypoint target, a task's fleet), and it stands for the new fleet's id.
// The id is the engine's, assigned as for any split; only the name is the
// planner's. 0 names nothing.
//
// ASSUMPTION L28: a NewFleet above 0, or one already used by an earlier
// split in the file, rejects the split (ErrOutOfRange). A name whose split
// was rejected stays unbound, so the orders that use it are rejected as
// naming no fleet. ORDERS.md has no such names: the original's client
// knows a new fleet's number at once.
type SplitOrder struct {
	Fleet    int
	Ships    []Stack // Design and Count; Damage is ignored
	NewFleet int
}

func (o SplitOrder) apply(g *Game, player int, a *Applied) error {
	if o.NewFleet > 0 || (o.NewFleet < 0 && a.newFleets[o.NewFleet] != 0) {
		return fmt.Errorf("split fleet %d: new fleet name %d: %w", o.Fleet, o.NewFleet, ErrOutOfRange)
	}
	i, err := g.ownFleet(player, o.Fleet)
	if err != nil {
		return err
	}
	fleets := 0
	for _, f := range g.Fleets {
		if f.Owner == player {
			fleets++
		}
	}
	if fleets >= maxFleets {
		return fmt.Errorf("split fleet %d: %d fleets: %w", o.Fleet, fleets, ErrOutOfRange)
	}
	if err := g.checkShips(&g.Fleets[i], o.Ships); err != nil {
		return err
	}
	src := g.Fleets[i]
	nf := Fleet{
		ID: g.newFleetID(), Number: g.lowestFreeFleetNumber(player), Owner: player, Pos: src.Pos,
		Waypoints: append([]Waypoint(nil), src.Waypoints...), Plan: src.Plan, Task: src.Task, Repeat: src.Repeat,
	}
	g.Fleets = append(g.Fleets, nf)
	g.moveShips(g.fleetIndex(src.ID), len(g.Fleets)-1, o.Ships)
	if o.NewFleet < 0 {
		a.newFleets[o.NewFleet] = nf.ID
	}
	return nil
}

// withNewFleets returns o with every fleet it names that is a new-fleet
// name of the file (SplitOrder.NewFleet) replaced by that fleet's id.
// Names not bound yet are left as they are.
func withNewFleets(o Order, names map[int]int) Order {
	if len(names) == 0 {
		return o
	}
	id := func(v int) int {
		if r, ok := names[v]; ok && v < 0 {
			return r
		}
		return v
	}
	task := func(t Task) Task {
		t.Fleet = id(t.Fleet)
		return t
	}
	switch o := o.(type) {
	case SplitOrder:
		o.Fleet = id(o.Fleet)
		return o
	case MoveShipsOrder:
		o.Fleet, o.With = id(o.Fleet), id(o.With)
		return o
	case MergeOrder:
		o.Into = id(o.Into)
		o.From = append([]int(nil), o.From...)
		for k := range o.From {
			o.From[k] = id(o.From[k])
		}
		return o
	case FleetPlanOrder:
		o.Fleet = id(o.Fleet)
		return o
	case RenameOrder:
		o.Fleet = id(o.Fleet)
		return o
	case RepeatOrder:
		o.Fleet = id(o.Fleet)
		return o
	case CargoOrder:
		o.Fleet = id(o.Fleet)
		if o.Target == TargetFleet {
			o.ID = id(o.ID)
		}
		return o
	case WaypointOrder:
		o.Fleet = id(o.Fleet)
		o.Task = task(o.Task)
		o.Waypoints = append([]Waypoint(nil), o.Waypoints...)
		for k := range o.Waypoints {
			if o.Waypoints[k].Target == TargetFleet {
				o.Waypoints[k].ID = id(o.Waypoints[k].ID)
			}
			o.Waypoints[k].Task = task(o.Waypoints[k].Task)
		}
		return o
	}
	return o
}

// MoveShipsOrder moves ships between two of the player's fleets at one
// location (ORDERS.md "Split", CONFIRMED CO-03): a positive count moves
// ships of that design from With into Fleet, a negative one from Fleet
// into With. The ships carry floor(amount × their capacity ÷ the giving
// fleet's capacity) of each cargo by hold and of fuel by tank, the
// rounding remainder staying with the giver. A stack holds at most 32765
// ships after a move; the rest are lost (MEASURED CO-06).
//
// Moved ships joining ships of their design combine damage by the merge
// order's rule, the units averaged over the damaged ships and rounded
// down (ORDERS.md
// "Merge", CONFIRMED CO-05, CO-05b, CO-05c).
//
// ASSUMPTION L22: the moved ships of a damaged stack keep its damage
// percentage and units, as do the ships left behind; ORDERS.md does not
// say which ships of a stack are the damaged ones.
//
// ASSUMPTION L23: ships leaving Fleet move first, then ships leaving
// With; a fleet left with no ships is removed, as merged fleets are
// (ORDERS.md "Merge").
//
// The fleets must be at one place, as for a cargo transfer between own
// fleets (ORDERS.md "Elegy implementation Q3").
type MoveShipsOrder struct {
	Fleet int
	With  int
	Ships []Stack // Design and Count; Damage is ignored
}

func (o MoveShipsOrder) apply(g *Game, player int, _ *Applied) error {
	a, err := g.ownFleet(player, o.Fleet)
	if err != nil {
		return err
	}
	b, err := g.ownFleet(player, o.With)
	if err != nil {
		return err
	}
	if a == b {
		return fmt.Errorf("move ships: fleet %d named twice: %w", o.Fleet, ErrOutOfRange)
	}
	if g.Fleets[a].Pos != g.Fleets[b].Pos {
		return fmt.Errorf("move ships: fleet %d is not with fleet %d: %w", o.With, o.Fleet, ErrNotTogether)
	}
	var out, in []Stack
	for _, s := range o.Ships {
		switch {
		case s.Count < 0:
			out = append(out, Stack{Design: s.Design, Count: -s.Count})
		case s.Count > 0:
			in = append(in, Stack{Design: s.Design, Count: s.Count})
		}
	}
	if err := g.checkShips(&g.Fleets[a], out); err != nil {
		return err
	}
	if err := g.checkShips(&g.Fleets[b], in); err != nil {
		return err
	}
	ida, idb := o.Fleet, o.With
	g.moveShips(a, b, out)
	g.moveShips(g.fleetIndex(idb), g.fleetIndex(ida), in)
	empty := map[int]bool{}
	for _, id := range []int{ida, idb} {
		if len(g.Fleets[g.fleetIndex(id)].Stacks) == 0 {
			empty[id] = true // ASSUMPTION L23
		}
	}
	g.removeFleets(empty)
	return nil
}

// checkShips reports whether fleet f holds the ships named, each design
// once with a positive count.
func (g *Game) checkShips(f *Fleet, ships []Stack) error {
	seen := map[int]bool{}
	for _, s := range ships {
		if s.Count <= 0 || seen[s.Design] {
			return fmt.Errorf("ships of design %d: %w", s.Design, ErrOutOfRange)
		}
		seen[s.Design] = true
		have := 0
		for _, t := range f.Stacks {
			if t.Design == s.Design {
				have += t.Count
			}
		}
		if s.Count > have {
			return fmt.Errorf("fleet %d has %d ships of design %d, not %d: %w", f.ID, have, s.Design, s.Count, ErrOutOfRange)
		}
	}
	return nil
}

// moveShips moves the named ships (checked by checkShips) from fleet index
// from to fleet index to with their share of cargo and fuel.
func (g *Game) moveShips(from, to int, ships []Stack) {
	if len(ships) == 0 {
		return
	}
	src, dst := &g.Fleets[from], &g.Fleets[to]
	hold, tank := g.cargoCapacity(src), g.tankCapacity(src)
	movedHold, movedTank := 0, 0
	for _, s := range ships {
		movedHold += s.Count * g.Designs[s.Design].CargoCapacity
		movedTank += s.Count * g.Designs[s.Design].FuelCapacity
	}
	share := func(amount, part, whole int) int {
		if whole <= 0 {
			return 0
		}
		return amount * part / whole
	}
	for m := range NumMinerals {
		n := share(src.Cargo.Minerals[m], movedHold, hold)
		src.Cargo.Minerals[m] -= n
		dst.Cargo.Minerals[m] += n
	}
	n := share(src.Cargo.Colonists, movedHold, hold)
	src.Cargo.Colonists -= n
	dst.Cargo.Colonists += n
	n = share(src.Fuel, movedTank, tank)
	src.Fuel -= n
	dst.Fuel += n

	for _, s := range ships {
		var dmg Damage
		for k := range src.Stacks {
			if src.Stacks[k].Design == s.Design {
				dmg = src.Stacks[k].Damage // ASSUMPTION L22
				src.Stacks[k].Count -= s.Count
				if src.Stacks[k].Count == 0 {
					src.Stacks = append(src.Stacks[:k:k], src.Stacks[k+1:]...)
				}
				break
			}
		}
		moved := Stack{Design: s.Design, Count: s.Count, Damage: dmg}
		j := -1
		for k, t := range dst.Stacks {
			if t.Design == s.Design {
				j = k
				break
			}
		}
		if j < 0 {
			j = g.insertStack(dst, moved)
		} else {
			d := &dst.Stacks[j]
			d.Damage = mergeDamage(*d, moved, mergeOrderDamaged)
			d.Count += moved.Count
		}
		dst.Stacks[j].Count = min(dst.Stacks[j].Count, maxExchangeStack)
	}
}

package engine

import (
	"fmt"
	"slices"
)

// Fleet operations (stars-elegy ORDERS.md "Fleet operations") and the
// design read (ORDERS.md "Range and legality clamps").

// Fleet operation events.
const (
	EventMergeRefused EventKind = iota + EventColonizeFailed + 1 // Player, Fleet: a Merge with Fleet task was refused
)

// maxStackShips is the merge order's per-design limit (ORDERS.md "Merge",
// BINARY-ONLY).
const maxStackShips = 32766

// legacyMergeDilution reproduces the original's LEGACY BUG in merging
// damaged stacks (ORDERS.md "Merge", CONFIRMED FO-01..07): when both
// stacks of a design were damaged, the units are divided by all ships of
// the merged stack instead of the damaged ones, which dilutes the damage.
// Set it to false to divide by the damaged ships.
const legacyMergeDilution = true

// legacyMergeOverflow reproduces the original's LEGACY BUG in the Merge
// with Fleet task (ORDERS.md "Merge", CONFIRMED FO): the task applies no
// ship-count cap, a stack of 32767 is kept, and a stack pushed to 32768 or
// beyond leaves the merged fleet with no ships, its cargo and fuel staying
// behind. Off by default: Elegy's chosen rule holds the task to the merge
// order's cap (absorb).
const legacyMergeOverflow = false

// mergeDamage combines two stacks of one design (ORDERS.md "Merge",
// CONFIRMED FO-01..07): with D = max(1, pct·count/100) damaged ships per
// damaged stack and n ships after the merge, the percentage is
// ceil(100·ΣD/n); one damaged stack keeps its units; two give
// ceil(Σ(D·units)/n) (dilute, the LEGACY BUG) or ceil(Σ(D·units)/ΣD).
func mergeDamage(a, b Stack, dilute bool) Damage {
	n := a.Count + b.Count
	damaged := func(s Stack) int { return max(1, s.Damage.Pct*s.Count/100) }
	switch {
	case a.Damage.Units == 0 && b.Damage.Units == 0:
		return Damage{}
	case b.Damage.Units == 0:
		return Damage{Pct: ceilDiv(100*damaged(a), n), Units: a.Damage.Units}
	case a.Damage.Units == 0:
		return Damage{Pct: ceilDiv(100*damaged(b), n), Units: b.Damage.Units}
	}
	da, db := damaged(a), damaged(b)
	div := n
	if !dilute {
		div = da + db
	}
	return Damage{Pct: ceilDiv(100*(da+db), n), Units: ceilDiv(da*a.Damage.Units+db*b.Damage.Units, div)}
}

// absorb adds src's ships, cargo and fuel to dst: ships add per design
// (ORDERS.md "Merge"), damage combined by mergeDamage with dilute. Unless
// overflow is set, a stack that would pass 32767 is held to maxStackShips
// and the ships above it are lost; a total of exactly 32767 is kept
// (ORDERS.md "Merge order", BINARY-ONLY). With overflow, a stack above
// 32767 empties dst of ships (the Merge with Fleet task's LEGACY BUG).
func (g *Game) absorb(dst, src *Fleet, overflow, dilute bool) {
	for _, s := range src.Stacks {
		if s.Count <= 0 {
			continue
		}
		j := -1
		for k, d := range dst.Stacks {
			if d.Design == s.Design {
				j = k
				break
			}
		}
		if j < 0 {
			j = g.insertStack(dst, s)
		} else {
			d := &dst.Stacks[j]
			d.Damage = mergeDamage(*d, s, dilute)
			d.Count += s.Count
		}
		if !overflow && dst.Stacks[j].Count > 32767 {
			dst.Stacks[j].Count = maxStackShips
		}
	}
	for m := range NumMinerals {
		dst.Cargo.Minerals[m] += src.Cargo.Minerals[m]
	}
	dst.Cargo.Colonists += src.Cargo.Colonists
	dst.Fuel += src.Fuel
	if overflow {
		for _, s := range dst.Stacks {
			if s.Count > 32767 {
				dst.Stacks = nil
				break
			}
		}
	}
}

// insertStack adds a stack of a design the fleet lacks in its owner's
// design-slot order and returns its index: a fleet keeps one count per
// design slot, so the new stack takes its slot's place
// (PRODUCTION-LAUNCH.md, BINARY-ONLY). Merges and the
// orders layer's launch share it.
func (g *Game) insertStack(f *Fleet, s Stack) int {
	slot := g.shipSlot(f.Owner, s.Design)
	at := len(f.Stacks)
	for k, t := range f.Stacks {
		if g.shipSlot(f.Owner, t.Design) > slot {
			at = k
			break
		}
	}
	f.Stacks = slices.Insert(f.Stacks, at, s)
	return at
}

// shipSlot is the owner's ship-design slot holding design d, which orders
// a fleet's stacks: the slot from Game.DesignSlots (orders_design.go). A
// design with no slot of the owner (states built without design orders)
// sorts after every slot, by its design index.
func (g *Game) shipSlot(owner, d int) int {
	for _, ds := range g.DesignSlots {
		if ds.Owner == owner && ds.Design == d && !ds.Starbase {
			return ds.Slot
		}
	}
	return maxShipDesigns + d
}

// MergeFleets is the merge order (ORDERS.md "Merge order", BINARY-ONLY):
// the fleets with ids from join the fleet with id into, all at one
// location and all the owner's, and are removed; into keeps its id. The
// order combines damage by its own rule, the units averaged over the
// damaged ships only (no dilution), and holds each design to 32766. Elegy
// validates ownership on every order (ORDERS.md "Ownership", chosen rule).
func (g *Game) MergeFleets(owner, into int, from []int) error {
	t := g.fleetIndex(into)
	if t < 0 || g.Fleets[t].Owner != owner {
		return fmt.Errorf("merge: fleet %d is not player %d's", into, owner)
	}
	ids := map[int]bool{}
	for _, id := range from {
		i := g.fleetIndex(id)
		switch {
		case i < 0 || g.Fleets[i].Owner != owner:
			return fmt.Errorf("merge: fleet %d is not player %d's", id, owner)
		case id == into || ids[id]:
			return fmt.Errorf("merge: fleet %d named twice", id)
		case g.Fleets[i].Pos != g.Fleets[t].Pos:
			return fmt.Errorf("merge: fleet %d is not with fleet %d", id, into)
		}
		ids[id] = true
	}
	for _, id := range from {
		g.absorb(&g.Fleets[g.fleetIndex(into)], &g.Fleets[g.fleetIndex(id)], false, false)
	}
	g.removeFleets(ids)
	return nil
}

// loadPass is a waypoint phase's load pass (TAKEOVER.md "Where each task
// happens", steps 2 and 5; "Other waypoint tasks"): each fleet in fleet
// order carries out its Merge with Fleet task. Loads and fleet transfers
// are not modelled.
func (g *Game) loadPass() []Event {
	var events []Event
	gone := map[int]bool{}
	for _, i := range g.fleetOrder() {
		f := &g.Fleets[i]
		if gone[f.ID] || f.Task.Kind != TaskMerge {
			continue
		}
		if ev, ok := g.mergeTask(f, gone); ok {
			gone[f.ID] = true
		} else {
			events = append(events, ev)
		}
	}
	g.removeFleets(gone)
	return events
}

// mergeTask runs a Merge with Fleet task (ORDERS.md "Merge", CONFIRMED
// FO-01..07): f joins the target, which keeps its id, if the target is at
// f's position; otherwise the task is refused and cleared, both fleets
// unchanged. A target that is gone, has already merged away, or is another
// player's fleet is refused the same way (BINARY-ONLY). It reports whether
// f merged.
func (g *Game) mergeTask(f *Fleet, gone map[int]bool) (Event, bool) {
	t := g.fleetIndex(f.Task.Fleet)
	if t < 0 || gone[f.Task.Fleet] || f.Task.Fleet == f.ID || g.Fleets[t].Owner != f.Owner || g.Fleets[t].Pos != f.Pos {
		f.Task = Task{}
		return Event{Kind: EventMergeRefused, Player: f.Owner, Planet: -1, Fleet: f.ID}, false
	}
	g.absorb(&g.Fleets[t], f, legacyMergeOverflow, legacyMergeDilution)
	return Event{}, true
}

// legacyKeepUnentitledParts reproduces the original's LEGACY BUG in
// reading a design (ORDERS.md "Design legality (Mystery Trader parts
// kept)", CONFIRMED): only parts above the owner's research tech are
// dropped, so a part the owner's race or Mystery Trader items do not
// entitle it to is kept and works. Off by default: Elegy's chosen rule
// drops every part the owner is not entitled to.
const legacyKeepUnentitledParts = false

// basicEngine back-fills an emptied engine slot (ORDERS.md: engine item
// 1, skipping the Hyper-Expansion-only item 0, BINARY-ONLY).
const basicEngine = "Quick Jump 5"

// ReadDesign builds an owner's design from the table (ORDERS.md "Design
// legality"): a filled slot whose part the owner may not build is dropped,
// and the design's mass and capacities are those of the parts that remain
// (BINARY-ONLY). With the chosen rule a part is kept only if the race may
// build it at these levels and, for a Mystery Trader part, owns it
// (traderItems, by part name); with legacyKeepUnentitledParts only the
// tech levels are checked.
//
// The hull (ORDERS.md "Design legality (hull not entitled, or every part
// stripped)", BINARY-ONLY): the original does not check it; Elegy's
// chosen rule rejects a design on a hull the owner may not build (not
// under legacyKeepUnentitledParts, which reproduces the original). A ship
// hull whose engine slot is left empty is back-filled with the basic
// engine, the Quick Jump 5, at the slot's capacity, as in the original.
func (c *Catalog) ReadDesign(name, hull string, fills []SlotFill, race Race, levels [NumFields]int, traderItems map[string]bool) (Design, error) {
	return c.readDesign(name, hull, fills, race, levels, traderItems, legacyKeepUnentitledParts)
}

func (c *Catalog) readDesign(name, hull string, fills []SlotFill, race Race, levels [NumFields]int, traderItems map[string]bool, techOnly bool) (Design, error) {
	keep := func(part string) (bool, error) {
		pc, ok := c.Lookup(part)
		if !ok {
			return false, fmt.Errorf("design %q: no part %q", name, part)
		}
		if techOnly {
			for f := range NumFields {
				if levels[f] < pc.TechReq[f] {
					return false, nil
				}
			}
			return true, nil
		}
		return pc.Buildable(race, levels, traderItems[part])
	}
	hc, ok := c.Lookup(hull)
	if !ok {
		return Design{}, fmt.Errorf("design %q: no hull %q", name, hull)
	}
	h, err := hc.Hull()
	if err != nil {
		return Design{}, fmt.Errorf("design %q: %w", name, err)
	}
	if !techOnly {
		ok, err := hc.Buildable(race, levels, traderItems[hull])
		if err != nil {
			return Design{}, err
		}
		if !ok {
			return Design{}, fmt.Errorf("design %q: hull %q is not available to this owner", name, hull)
		}
	}
	var kept []SlotFill
	engine := false
	for _, f := range fills {
		ok, err := keep(f.Part)
		if err != nil {
			return Design{}, err
		}
		if ok {
			kept = append(kept, f)
			engine = engine || f.Slot == 0
		}
	}
	if !h.Starbase && len(h.Slots) > 0 && !engine {
		kept = append([]SlotFill{{Slot: 0, Part: basicEngine, Count: h.Slots[0].Max}}, kept...)
	}
	return c.NewDesign(name, hull, kept)
}

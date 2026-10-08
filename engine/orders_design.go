package engine

import "fmt"

// Design orders (stars-elegy ORDERS.md "design change", "Design legality",
// "Ownership"). The design read itself, which parts survive and whether
// the hull is accepted, is Catalog.ReadDesign; this file is the order
// around it: whose slot, which slot, and what deleting a design does.

// DesignSlot gives one of a player's design slots its design, an index
// into Game.Designs. Ship and starbase designs have separate slots.
//
// Created is the calendar year the design order stored the design, and
// Picture its picture, 0..3 (AI.md "Storing a design", "Picture"; the
// computer players' starbase family switch and design ageing read them,
// CONFIRMED AI-2, AI-8, AI-19). Created 0 means a slot not made by a
// design order.
//
// Built is how many ships of the slot's design were ever built
// (stars-elegy ai/turindrone.md: a design's "built" count; robotoid.md
// §3 reads it). Production counts every ship it completes, also ships
// lost at the fleet limit. A save holds the whole Game, so it writes
// Built; an older save of the same version loads it as 0.
//
// ASSUMPTION L29: an edit in place starts Built again at 0, as a new
// design does; a starbase slot counts nothing.
type DesignSlot struct {
	Owner    int
	Starbase bool
	Slot     int
	Design   int
	Created  int
	Picture  int
	Built    int
}

// designPictures is the number of pictures per hull (AI.md "Picture":
// "the hull's four pictures").
const designPictures = 4

// Design slots per player (LIMITS.md "Designs and battle plans",
// BINARY-ONLY; the ship-slot count MEASURED by OB-026): a
// design order naming a slot past the last is refused.
const (
	maxShipDesigns     = 16
	maxStarbaseDesigns = 10
)

// designSlot returns the index in g.DesignSlots of a player's slot, or -1.
func (g *Game) designSlot(owner int, starbase bool, slot int) int {
	for i, s := range g.DesignSlots {
		if s.Owner == owner && s.Starbase == starbase && s.Slot == slot {
			return i
		}
	}
	return -1
}

// PlayerDesign returns the design in a player's slot, an index into
// g.Designs.
func (g *Game) PlayerDesign(owner int, starbase bool, slot int) (int, bool) {
	if i := g.designSlot(owner, starbase, slot); i >= 0 {
		return g.DesignSlots[i].Design, true
	}
	return 0, false
}

// designInUse reports whether a ship or starbase of design d exists.
func (g *Game) designInUse(d int) bool {
	for _, f := range g.Fleets {
		for _, s := range f.Stacks {
			if s.Design == d && s.Count > 0 {
				return true
			}
		}
	}
	for _, p := range g.Planets {
		if p.HasStarbase && p.StarbaseDesign == d {
			return true
		}
	}
	return false
}

// DesignOrder puts a design into one of the player's slots. The slot is
// the player's own by construction (the original re-checks the slot's
// owner, ORDERS.md "Ownership"). The design is read with
// Catalog.ReadDesign against the player's race and tech, so parts the
// player is not entitled to are dropped and an un-entitled hull is
// rejected (ORDERS.md "Design legality", Elegy's chosen rule).
//
// A slot number outside the player's slots (LIMITS.md) and a name longer
// than maxNameLength are rejected. A change to a slot whose design has
// ships or a starbase in play is refused; any other filled slot is
// overwritten in place, so whatever refers to the design (a queue entry
// building it) gets the edited design (ORDERS.md "Design change into an
// occupied slot": MEASURED CO-08 for the queue-only case; the client
// cannot edit a design in use by ships or a starbase, so that refusal is
// Elegy's rule and never fires on a legal order).
//
// A queue entry names the slot, not the design, so after an edit it
// builds the edited design (CO-08).
//
// Malformed fills are dropped or cut, never rejecting the design
// (ORDERS.md "Design read, four malformed cases"): a part the slot
// does not take is dropped, a count over the slot's capacity is cut to
// it, then ReadDesign strips parts above tech and back-fills an empty
// engine slot.
//
// ASSUMPTION L11: a hull of the wrong kind for the slot (a starbase hull
// for a ship slot, or the reverse) rejects the design. A fill naming a
// slot the hull lacks, or a count below 1, is dropped like a part the
// slot does not take; two fills for one slot, or an engine slot holding
// fewer than its capacity, still reject the design (NewDesign).
//
// The slot records the year the order applies as the design's creation
// year and the order's picture (AI.md "Storing a design": "The stored
// design's creation year is the current year").
//
// ASSUMPTION L25: a picture outside 0..3 rejects the design; AI.md names
// four pictures per hull and no order-time check.
//
// ASSUMPTION L26: a design edited in place in an occupied slot takes the
// year of the edit as its creation year and the order's picture; AI.md
// describes a computer player's replacement as a delete and a new design,
// and says nothing of an edit.
//
// Mystery Trader items are not modelled: the player owns none.
type DesignOrder struct {
	Starbase bool
	Slot     int
	Name     string
	Hull     string
	Fills    []SlotFill
	Picture  int
}

func (o DesignOrder) apply(g *Game, player int, _ *Applied) error {
	limit := maxShipDesigns
	if o.Starbase {
		limit = maxStarbaseDesigns
	}
	if o.Slot < 0 || o.Slot >= limit || len(o.Name) > maxNameLength {
		return fmt.Errorf("design %q slot %d: %w", o.Name, o.Slot, ErrOutOfRange)
	}
	if o.Picture < 0 || o.Picture >= designPictures {
		return fmt.Errorf("design %q picture %d: %w", o.Name, o.Picture, ErrOutOfRange)
	}
	cat := Components()
	if hc, ok := cat.Lookup(o.Hull); ok {
		if h, err := hc.Hull(); err == nil && h.Starbase != o.Starbase {
			return fmt.Errorf("design %q: hull %q: %w", o.Name, o.Hull, ErrOutOfRange)
		}
	}
	pl := &g.Players[player]
	d, err := cat.ReadDesign(o.Name, o.Hull, fitFills(cat, o.Hull, o.Fills), pl.Race, pl.Research.Levels, nil, g.Rules)
	if err != nil {
		return err
	}
	if i := g.designSlot(player, o.Starbase, o.Slot); i >= 0 {
		old := g.DesignSlots[i].Design
		if g.designInUse(old) {
			return fmt.Errorf("design slot %d: the design in it is in use: %w", o.Slot, ErrOutOfRange)
		}
		g.DesignSlots = append([]DesignSlot(nil), g.DesignSlots...)
		g.DesignSlots[i].Created, g.DesignSlots[i].Picture, g.DesignSlots[i].Built = g.Year, o.Picture, 0
		if g.designShared(old, i) {
			// A design index another slot also names (a state built
			// without design orders) is not overwritten under it.
			g.Designs = append(g.Designs[:len(g.Designs):len(g.Designs)], d)
			g.DesignSlots[i].Design = len(g.Designs) - 1
			return nil
		}
		designs := append([]Design(nil), g.Designs...)
		designs[old] = d
		g.Designs = designs
		return nil
	}
	g.Designs = append(g.Designs[:len(g.Designs):len(g.Designs)], d)
	g.DesignSlots = append(g.DesignSlots[:len(g.DesignSlots):len(g.DesignSlots)],
		DesignSlot{Owner: player, Starbase: o.Starbase, Slot: o.Slot, Design: len(g.Designs) - 1, Created: g.Year, Picture: o.Picture})
	return nil
}

// designShared reports whether a design slot other than the one at index
// skip names design d.
func (g *Game) designShared(d, skip int) bool {
	for i, s := range g.DesignSlots {
		if i != skip && s.Design == d {
			return true
		}
	}
	return false
}

// fitFills drops each fill whose part the hull's slot does not take and
// cuts a count over the slot's capacity to it (ORDERS.md "Design read,
// four malformed cases", cases 2 and 3). An unknown hull or part is left
// for ReadDesign to reject.
func fitFills(cat *Catalog, hull string, fills []SlotFill) []SlotFill {
	hc, ok := cat.Lookup(hull)
	if !ok {
		return fills
	}
	h, err := hc.Hull()
	if err != nil {
		return fills
	}
	var out []SlotFill
	for _, f := range fills {
		pc, ok := cat.Lookup(f.Part)
		if !ok {
			out = append(out, f)
			continue
		}
		p, err := pc.Part()
		if err != nil {
			out = append(out, f)
			continue
		}
		if f.Slot < 0 || f.Slot >= len(h.Slots) || f.Count < 1 || !kindIn(p.Kind, h.Slots[f.Slot].Kinds) {
			continue
		}
		f.Count = min(f.Count, h.Slots[f.Slot].Max)
		out = append(out, f)
	}
	return out
}

// DeleteDesignOrder empties one of the player's design slots (ORDERS.md
// "Design delete effect", MEASURED CO-07/CO-07b/CO-07c): ships of the
// design are removed, a fleet left without ships is removed, and a
// starbase of the design is removed from its planet, which keeps its
// population (KERNEL.md "Maximum population", BINARY-ONLY). The slot is
// cleared in place; later slots keep their numbers. When the removed
// ships shared a fleet with survivors, they take their share of its fuel
// and cargo as a ship move does: floor(amount × their capacity ÷ the
// fleet's capacity) of each, fuel by tank and cargo by hold, and the rest
// stays with the survivors unclamped (CO-07c).
//
// Every queue entry of the player's that builds the slot is dropped, even
// with progress (MEASURED CO-07: dropped at 66% done).
//
// ASSUMPTION P4: a queue the drop leaves empty is removed, as production
// removes a queue emptied during the year; KERNEL.md says a zero-item
// queue does not arise in play. The entry in
// Game.Designs stays, so other indices do not move.
type DeleteDesignOrder struct {
	Starbase bool
	Slot     int
}

func (o DeleteDesignOrder) apply(g *Game, player int, _ *Applied) error {
	i := g.designSlot(player, o.Starbase, o.Slot)
	if i < 0 {
		return fmt.Errorf("design slot %d: %w", o.Slot, ErrNoSuchObject)
	}
	d := g.DesignSlots[i].Design
	g.DesignSlots = append(g.DesignSlots[:i:i], g.DesignSlots[i+1:]...)
	kind := ItemShip
	if o.Starbase {
		kind = ItemStarbase
	}
	for k := range g.Planets {
		p := &g.Planets[k]
		if p.Owner != player || !p.HasQueue {
			continue
		}
		var q []QueueItem
		for _, it := range p.Queue {
			if it.Kind != kind || it.Slot != o.Slot {
				q = append(q, it)
			}
		}
		if len(q) != len(p.Queue) {
			p.Queue = q
			if len(q) == 0 {
				p.HasQueue = false // ASSUMPTION P4
			}
		}
	}
	for k := range g.Planets {
		p := &g.Planets[k]
		if p.Owner == player && p.HasStarbase && p.StarbaseDesign == d {
			p.HasStarbase, p.StarbaseHull, p.StarbaseDock, p.StarbaseDamage = false, 0, false, 0
		}
	}
	empty := map[int]bool{}
	for k := range g.Fleets {
		f := &g.Fleets[k]
		if f.Owner != player {
			continue
		}
		stacks := []Stack{}
		for _, s := range f.Stacks {
			if s.Design != d {
				stacks = append(stacks, s)
			}
		}
		if len(stacks) != len(f.Stacks) {
			if len(stacks) == 0 {
				empty[f.ID] = true
				f.Stacks = stacks
				continue
			}
			tank, hold := g.tankCapacity(f), g.cargoCapacity(f)
			gone := 0
			for _, s := range f.Stacks {
				if s.Design == d {
					gone += s.Count
				}
			}
			goneTank, goneHold := gone*g.Designs[d].FuelCapacity, gone*g.Designs[d].CargoCapacity
			if tank > 0 {
				f.Fuel -= f.Fuel * goneTank / tank
			}
			if hold > 0 {
				for m := range NumMinerals {
					f.Cargo.Minerals[m] -= f.Cargo.Minerals[m] * goneHold / hold
				}
				f.Cargo.Colonists -= f.Cargo.Colonists * goneHold / hold
			}
			f.Stacks = stacks
		}
	}
	g.removeFleets(empty)
	return nil
}

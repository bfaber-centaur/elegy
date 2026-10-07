package engine

import "fmt"

// Design orders (stars-elegy ORDERS.md "design change", "Design legality",
// "Ownership"). The design read itself, which parts survive and whether
// the hull is accepted, is Catalog.ReadDesign; this file is the order
// around it: whose slot, which slot, and what deleting a design does.

// DesignSlot gives one of a player's design slots its design, an index
// into Game.Designs. Ship and starbase designs have separate slots.
type DesignSlot struct {
	Owner    int
	Starbase bool
	Slot     int
	Design   int
}

// Design slots per player (LIMITS.md "Designs and battle plans",
// stars-elegy #65, BINARY-ONLY; the ship-slot count MEASURED by OB-026): a
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
// than maxNameLength are rejected. A new design for a slot whose design
// has ships or a starbase in play is refused (ORDERS.md "Design change
// into an occupied slot", stars-elegy #51, chosen rule; the original is
// BINARY-ONLY). Production has no ship items yet, so a queued design does
// not count.
//
// Malformed fills are dropped or cut, never rejecting the design
// (ORDERS.md "Design read, four malformed cases", #51): a part the slot
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
// Mystery Trader items are not modelled: the player owns none.
type DesignOrder struct {
	Starbase bool
	Slot     int
	Name     string
	Hull     string
	Fills    []SlotFill
}

func (o DesignOrder) apply(g *Game, player int, _ *Applied) error {
	limit := maxShipDesigns
	if o.Starbase {
		limit = maxStarbaseDesigns
	}
	if o.Slot < 0 || o.Slot >= limit || len(o.Name) > maxNameLength {
		return fmt.Errorf("design %q slot %d: %w", o.Name, o.Slot, ErrOutOfRange)
	}
	cat := Components()
	if hc, ok := cat.Lookup(o.Hull); ok {
		if h, err := hc.Hull(); err == nil && h.Starbase != o.Starbase {
			return fmt.Errorf("design %q: hull %q: %w", o.Name, o.Hull, ErrOutOfRange)
		}
	}
	pl := &g.Players[player]
	d, err := cat.ReadDesign(o.Name, o.Hull, fitFills(cat, o.Hull, o.Fills), pl.Race, pl.Research.Levels, nil)
	if err != nil {
		return err
	}
	if i := g.designSlot(player, o.Starbase, o.Slot); i >= 0 {
		if g.designInUse(g.DesignSlots[i].Design) {
			return fmt.Errorf("design slot %d: the design in it is in use: %w", o.Slot, ErrOutOfRange)
		}
		g.Designs = append(g.Designs[:len(g.Designs):len(g.Designs)], d)
		g.DesignSlots[i].Design = len(g.Designs) - 1
		return nil
	}
	g.Designs = append(g.Designs[:len(g.Designs):len(g.Designs)], d)
	g.DesignSlots = append(g.DesignSlots[:len(g.DesignSlots):len(g.DesignSlots)],
		DesignSlot{Owner: player, Starbase: o.Starbase, Slot: o.Slot, Design: len(g.Designs) - 1})
	return nil
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
// "Design delete effect", stars-elegy #51, chosen rule): ships of the
// design are removed, a fleet left without ships is removed, and a
// starbase of the design is removed from its planet, which keeps its
// population (KERNEL.md "Maximum population", BINARY-ONLY). The player's
// later slots of the same kind move down one, as later battle plans do.
// Production has no ship items yet, so no queue entry is dropped. The
// entry in Game.Designs stays, so other indices do not move.
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
	for k := range g.DesignSlots {
		if s := &g.DesignSlots[k]; s.Owner == player && s.Starbase == o.Starbase && s.Slot > o.Slot {
			s.Slot--
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
			f.Stacks = stacks
			if len(stacks) == 0 {
				empty[f.ID] = true
			}
		}
	}
	g.removeFleets(empty)
	return nil
}

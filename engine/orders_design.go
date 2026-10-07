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

// Design slots per player.
//
// ASSUMPTION L10: 16 ship and 10 starbase designs. No public spec gives
// the number yet (COVERAGE.md lists design slots under the limits still
// to collect).
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
// ASSUMPTION L11: a slot number outside the player's slots, a hull that
// is a starbase hull for a ship slot or a ship hull for a starbase slot,
// a name longer than maxNameLength, and a new design for a slot whose
// design has ships or a starbase in play are rejected. ORDERS.md does
// not say what replacing a design in use does.
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
	d, err := cat.ReadDesign(o.Name, o.Hull, o.Fills, pl.Race, pl.Research.Levels, nil)
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

// DeleteDesignOrder empties one of the player's design slots. A starbase
// of the design is removed from its planet, which keeps its population
// (KERNEL.md "Maximum population", BINARY-ONLY).
//
// ASSUMPTION L12: ships of a deleted design are removed, and a fleet left
// without ships is removed. No public spec says what happens to them.
// The entry in Game.Designs stays, so other indices do not move.
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

package game

import (
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// Check lists structural problems in a game state: references that point
// nowhere and quantities that cannot be negative. The loop runs it after
// every year so a bad year stops with the object named, rather than
// surfacing years later as a strange report.
//
// These are consistency checks of Elegy's own data model, not game
// rules; the engine's rule invariants are the engine's.
func Check(g engine.Game) []string {
	var out []string
	bad := func(format string, a ...any) { out = append(out, fmt.Sprintf(format, a...)) }
	np := len(g.Players)
	validOwner := func(o int) bool { return o >= 0 && o < np }

	planetIDs := map[int]bool{}
	for i, p := range g.Planets {
		if planetIDs[p.ID] {
			bad("planet %d (index %d): duplicate id", p.ID, i)
		}
		planetIDs[p.ID] = true
		if p.Owner != engine.NoOwner && !validOwner(p.Owner) {
			bad("planet %d: owner %d is not a player", p.ID, p.Owner)
		}
		if p.Population < 0 || p.Mines < 0 || p.Factories < 0 || p.Defenses < 0 {
			bad("planet %d: negative population or installations (%d pop, %d mines, %d factories, %d defenses)", p.ID, p.Population, p.Mines, p.Factories, p.Defenses)
		}
		if p.Owner == engine.NoOwner && p.Population != 0 {
			bad("planet %d: unowned with population %d", p.ID, p.Population)
		}
		for m, v := range p.Surface {
			if v < 0 {
				bad("planet %d: surface mineral %d is %d", p.ID, m, v)
			}
		}
		if p.HasStarbase && (p.StarbaseDesign < 0 || p.StarbaseDesign >= len(g.Designs)) {
			bad("planet %d: starbase design %d does not exist", p.ID, p.StarbaseDesign)
		}
	}

	fleetIDs := map[int]bool{}
	for _, f := range g.Fleets {
		if fleetIDs[f.ID] {
			bad("fleet %d: duplicate id", f.ID)
		}
		fleetIDs[f.ID] = true
		if !validOwner(f.Owner) {
			bad("fleet %d: owner %d is not a player", f.ID, f.Owner)
		}
		if len(f.Stacks) == 0 {
			bad("fleet %d: no ships", f.ID)
		}
		for _, s := range f.Stacks {
			if s.Design < 0 || s.Design >= len(g.Designs) {
				bad("fleet %d: design %d does not exist", f.ID, s.Design)
			}
			if s.Count <= 0 {
				bad("fleet %d: a stack of design %d has %d ships", f.ID, s.Design, s.Count)
			}
		}
		if f.Fuel < 0 || f.Cargo.Colonists < 0 {
			bad("fleet %d: negative fuel or colonists (%d mg, %d)", f.ID, f.Fuel, f.Cargo.Colonists)
		}
		for m, v := range f.Cargo.Minerals {
			if v < 0 {
				bad("fleet %d: cargo mineral %d is %d", f.ID, m, v)
			}
		}
		if f.Plan < 0 || (validOwner(f.Owner) && len(g.Players[f.Owner].Plans) > 0 && f.Plan >= len(g.Players[f.Owner].Plans)) {
			bad("fleet %d: battle plan %d does not exist", f.ID, f.Plan)
		}
	}

	type slotKey struct {
		owner    int
		starbase bool
		slot     int
	}
	slots := map[slotKey]bool{}
	for _, s := range g.DesignSlots {
		k := slotKey{s.Owner, s.Starbase, s.Slot}
		if slots[k] {
			bad("player %d design slot %d (starbase %v): filled twice", s.Owner, s.Slot, s.Starbase)
		}
		slots[k] = true
		if !validOwner(s.Owner) {
			bad("design slot of owner %d: not a player", s.Owner)
		}
		if s.Design < 0 || s.Design >= len(g.Designs) {
			bad("player %d design slot %d: design %d does not exist", s.Owner, s.Slot, s.Design)
		}
	}
	return out
}

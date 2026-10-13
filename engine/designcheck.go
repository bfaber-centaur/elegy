package engine

// checkDesigns is the design check at year generation (stars-elegy
// TAKEOVER.md "Design parts dropped when the year is generated";
// PARITY.md "Design check (CONFIRMED, T-37)"). Every ship design in a
// player's design slots is re-checked against its owner's tech, and a slot
// whose part fails the check is emptied: the stored design no longer has
// it, so the owner's ships of that design act without it from this year.
//
// A part is removed when the owner lacks the tech for it, except a part
// only some primary racial traits may build (Restriction.PRTOnly) or a
// Mystery Trader part, which stays whatever the owner's tech:
//   - Removal: MEASURED at tech 3 in every field (PARITY.md "Design parts
//     dropped": Cherry, Smart, Peerless, LBU-17 and LBU-32 removed, and
//     those ships did not bomb); CONFIRMED by prediction at electronics 7
//     with every other field 26 (T-37, TK-104-A, TK-105-A: LBU-17 and
//     LBU-32 removed).
//   - PRTOnly exemption: MEASURED, Retro (Claim Adjuster only; weapons 10,
//     biotech 12) stayed and bombed at tech 3. Smart and Peerless, which
//     every race but Inner Strength may build (PRTNot), were removed, so
//     PRTNot is no exemption (MEASURED).
//   - Mystery Trader exemption: MEASURED at tech 3 (Hush-a-Boom, Multi
//     Contained Munition stayed and acted) and CONFIRMED by prediction at
//     electronics 7 (TK-104-A: Hush-a-Boom, electronics 12, and the Multi
//     Contained Munition, electronics 16, stayed).
//   - A part restricted only by lesser racial traits is checked like an
//     unrestricted one (ASSUMPTION T15: none was run above tech).
//
// The check looks at tech only, never at whether the owner's race may
// build the part: the parity vectors' oracle states give JOAT owners
// parts of other races (the Mine Dispenser 40, cloaks, the Mini Gun, the
// Fuel Mizer) and their expected results show those parts working in the
// generated year. It is the same in every ruleset, and separate from
// Legacy.KeepUnentitledParts, which governs only reading a design from a
// design order (ReadDesign). Since tech levels never fall, a design made
// by an order never has a part this check removes; the check matters for
// designs written into a game from outside. The hull is not checked.
//
// A design received with a fleet transfer (DesignSlot.Received) is not
// checked, whichever player receives it: CONFIRMED by prediction for the
// year of the transfer and the year after (stars-elegy #200, merged at
// 1d4c97b; PARITY.md "The design check: timing, transfers, energy and loads", DC-5: the
// recipient's copy kept an out-of-tech scanner while the recipient's own
// design of it lost it; DC-2 MEASURED the same with the other player
// receiving). The giver keeps its own copy (MEASURED, DC-2 and DC-5).
// Keeping the exemption in later years is ASSUMPTION T15 (NOT RUN); why
// the copy escapes is unknown.
//
// From the same cases, CONFIRMED by prediction: a part is removed in the
// year its field levels up, whether the year starts with enough stored
// research or reaches it through the year's research (DC-1); an
// energy-gated part is checked like the others (DC-3); the check runs
// before the waypoint-0 load tasks, and a fleet keeps cargo above its
// design's new capacity (DC-6; that capacity is inferred from the
// stripped design). That the check uses the tech from before the year's
// research, ahead of the first level-up check, is ASSUMPTION T15 (DC-1,
// inferred: the dumps show the level only at year end).
//
// ASSUMPTION T15 (NOT RUN): the check runs in KERNEL.md "Turn order"
// step 1a.2, the ship-design housekeeping, after the year's orders;
// starbase designs are not checked; an emptied engine slot is back-filled
// with the basic engine at the slot's capacity, as ReadDesign does (the
// basic engine adds no fuel or cargo capacity); fleets keep their fuel
// even above the new capacity; and a received design stays exempt after
// the year following its transfer.
func (g *Game) checkDesigns() {
	cat := Components()
	for _, ds := range g.DesignSlots {
		if ds.Starbase || ds.Received || ds.Design < 0 || ds.Design >= len(g.Designs) || ds.Owner < 0 || ds.Owner >= len(g.Players) {
			continue
		}
		d, changed := cat.checkDesign(g.Designs[ds.Design], g.Players[ds.Owner].Research.Levels)
		if !changed {
			continue
		}
		g.Designs[ds.Design] = d
	}
}

// checkDesign is d with the parts the design check removes taken out, and
// whether any was. Mass, fuel and cargo capacity lose what the removed
// parts added (a misslotted part added none).
func (c *Catalog) checkDesign(d Design, levels [NumFields]int) (Design, bool) {
	drop := func(p Part) bool {
		if comp, ok := c.Lookup(p.Name); ok && (comp.MysteryTrader || len(comp.Restriction.PRTOnly) > 0) {
			return false
		}
		for f := range NumFields {
			if levels[f] < p.TechReq[f] {
				return true
			}
		}
		return false
	}
	pos := len(d.SlotPos) == len(d.Slots)
	var slots []Slot
	var slotPos []int
	out := d
	changed, engine := false, false
	for k, s := range d.Slots {
		if !drop(s.Part) {
			slots = append(slots, s)
			if pos {
				slotPos = append(slotPos, d.SlotPos[k])
			}
			continue
		}
		changed = true
		if d.misslotted(k) {
			continue
		}
		out.Mass -= s.Count * s.Part.Mass
		out.FuelCapacity -= s.Count * s.Part.FuelCapacity
		out.CargoCapacity -= s.Count * s.Part.CargoCapacity
		if s.Part.Kind == PartEngine {
			engine = true
		}
	}
	if !changed {
		return d, false
	}
	if engine {
		out.Engine, out.Engines = Engine{}, 0
		if !d.Hull.Starbase && len(d.Hull.Slots) > 0 {
			if e, p, ok := c.basicEngine(); ok {
				n := d.Hull.Slots[0].Max
				out.Engine, out.Engines = e, n
				out.Mass += n * p.Mass
				slots = append([]Slot{{Part: p, Count: n}}, slots...)
				if pos {
					slotPos = append([]int{0}, slotPos...)
				}
			}
		}
	}
	out.Slots = slots
	if pos {
		out.SlotPos = slotPos
	} else {
		out.SlotPos = nil
	}
	return out, true
}

// basicEngine is the basic engine's part and engine values.
func (c *Catalog) basicEngine() (Engine, Part, bool) {
	comp, ok := c.Lookup(basicEngine)
	if !ok {
		return Engine{}, Part{}, false
	}
	p, err := comp.Part()
	if err != nil {
		return Engine{}, Part{}, false
	}
	e, err := comp.Engine()
	if err != nil {
		return Engine{}, Part{}, false
	}
	return e, p, true
}

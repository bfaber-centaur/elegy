package engine

import "fmt"

// Cargo orders (stars-elegy ORDERS.md "Cargo amounts and clamps",
// "Cross-owner cargo", "Ownership"; TAKEOVER.md "Unload and load amounts"
// for the planet side).

// CargoFuel indexes fuel in CargoOrder.Amounts, after the minerals and
// colonists.
const CargoFuel = NumCargo

// Messages for cargo given to another player. Elegy's wording.
const (
	EventCargoGiftLost      EventKind = iota + EventGameLost + 1 // Player = giver, Fleet = giver's fleet, Count = kT or mg that did not fit
	EventColonistsLostGiven                                      // Player = giver, Planet, Count = colonists (units of 100) put onto an unowned planet or a starbase's planet
)

// CargoOrder moves cargo between one of the player's fleets and a planet
// or another fleet at the same place. Amounts holds, per cargo type (the
// three minerals, colonists, fuel), how much the fleet takes from the
// target (positive) or gives to it (negative); kT, colonists in units of
// 100, fuel in mg.
//
// Same owner: applied at once. Every amount is clamped to what the source
// holds and to the destination's free space, minerals and colonists
// against the cargo hold and fuel against the tank, independently
// (ORDERS.md "Cargo amounts and clamps", CONFIRMED FO-01..07). On the
// owner's planet unloaded colonists join the population and minerals the
// surface at once, and a load may take every colonist (TAKEOVER.md
// "Unload and load amounts", CONFIRMED TK-201).
//
// Another owner: only giving is allowed; taking from another player's
// planet or fleet acts on a foreign object and is rejected (ORDERS.md
// "Ownership", chosen rule). What is given is taken from the fleet as the
// order applies; relations do not matter (ORDERS.md "Cross-owner cargo";
// TAKEOVER.md "Manual cargo transfers to other players", CONFIRMED TK-501,
// TK-502):
//   - colonists onto another player's planet without a starbase are a
//     drop, resolved with the before-movement unloads;
//   - colonists onto an unowned planet, or a planet with a starbase, are
//     lost and the giver is told;
//   - minerals (and fuel, to a fleet) are credited in place in the
//     orders step's credit pass, after every debit and before movement
//     (creditGifts; ORDERS.md "Two passes, in place"; MEASURED for
//     planets, TK-405, TK-412, and for fleets, TK-406, TK-407, TK-409).
//
// Colonists given to another player's fleet are rejected: a legal client
// never writes such an order (TAKEOVER.md, TK-408,
// TK-414), so Elegy treats it as acting on a foreign object (ORDERS.md
// "Ownership", chosen rule).
//
// Preconditions (ORDERS.md "Elegy implementation Q4",
// chosen rules): the fleet must be at the target's position, or the
// order is refused; there is no deep-space jettison (a target is a planet
// or a fleet); fuel to or from a planet is dropped from the order, and
// its minerals and colonists still move (planets hold no fuel, FO-01 E).
// Any fleet with free cargo space may carry colonists (ORDERS.md "Elegy
// implementation Q3", chosen rule).
//
// A transfer between the player's own fleets moves the amounts given,
// not a capacity rebalance (ORDERS.md "Transfer between the player's own
// fleets", CONFIRMED CO-04). The client caps the amount by the receiver's
// free hold and what the source holds; Elegy applies the same caps to
// any order.
//
// ASSUMPTION L7: any failed check other than planet fuel rejects the
// whole order.
type CargoOrder struct {
	Fleet   int
	Target  TargetKind // TargetPlanet or TargetFleet
	ID      int
	Amounts [NumCargo + 1]int
}

// CargoGift is cargo given to another owner's planet or fleet, taken from
// the giver when the order applied and not yet credited.
type CargoGift struct {
	From      int // giving player
	FromFleet int
	Target    TargetKind
	ID        int
	Amounts   [NumCargo + 1]int // minerals and fuel; colonists are drops
}

// cargoCapacity is a fleet's cargo hold in kT.
func (g *Game) cargoCapacity(f *Fleet) int {
	c := 0
	for _, s := range f.Stacks {
		c += s.Count * g.Designs[s.Design].CargoCapacity
	}
	return c
}

// held is a pointer to cargo type c in a fleet.
func held(f *Fleet, c int) *int {
	switch {
	case c < NumMinerals:
		return &f.Cargo.Minerals[c]
	case c == CargoColonists:
		return &f.Cargo.Colonists
	}
	return &f.Fuel
}

// free is a fleet's free space for cargo type c.
func (g *Game) free(f *Fleet, c int) int {
	if c == CargoFuel {
		return max(0, g.tankCapacity(f)-f.Fuel)
	}
	return max(0, g.cargoCapacity(f)-f.Cargo.mass())
}

func (o CargoOrder) apply(g *Game, player int, a *Applied) error {
	fi, err := g.ownFleet(player, o.Fleet)
	if err != nil {
		return err
	}
	f := &g.Fleets[fi]
	if o.Target == TargetPlanet {
		o.Amounts[CargoFuel] = 0
	}
	taking := false
	for _, v := range o.Amounts {
		taking = taking || v > 0
	}
	switch o.Target {
	case TargetPlanet:
		pi := g.planetIndex(o.ID)
		if pi < 0 {
			return fmt.Errorf("cargo: planet %d: %w", o.ID, ErrNoSuchObject)
		}
		p := &g.Planets[pi]
		if p.Pos != f.Pos {
			return fmt.Errorf("cargo: fleet %d and planet %d: %w", f.ID, p.ID, ErrNotTogether)
		}
		if p.Owner != player {
			if taking {
				return fmt.Errorf("cargo: planet %d: %w", p.ID, ErrNotYours)
			}
			g.giveToPlanet(f, pi, o.Amounts, a)
			return nil
		}
		g.exchangeWithPlanet(f, p, o.Amounts)
		return nil
	case TargetFleet:
		ti := g.fleetIndex(o.ID)
		if ti < 0 || ti == fi {
			return fmt.Errorf("cargo: fleet %d: %w", o.ID, ErrNoSuchObject)
		}
		t := &g.Fleets[ti]
		if t.Pos != f.Pos {
			return fmt.Errorf("cargo: fleets %d and %d: %w", f.ID, t.ID, ErrNotTogether)
		}
		if t.Owner != player {
			switch {
			case taking:
				return fmt.Errorf("cargo: fleet %d: %w", t.ID, ErrNotYours)
			case o.Amounts[CargoColonists] != 0:
				return fmt.Errorf("cargo: fleet %d: %w", t.ID, ErrRefusedByOwner)
			}
			gift := CargoGift{From: player, FromFleet: f.ID, Target: TargetFleet, ID: t.ID}
			for c, v := range o.Amounts {
				n := min(-v, *held(f, c))
				*held(f, c) -= n
				gift.Amounts[c] = n
			}
			a.Gifts = append(a.Gifts, gift)
			return nil
		}
		for c, v := range o.Amounts {
			src, dst := t, f
			if v < 0 {
				src, dst, v = f, t, -v
			}
			n := min(v, *held(src, c), g.free(dst, c))
			*held(src, c) -= n
			*held(dst, c) += n
		}
		return nil
	}
	return fmt.Errorf("cargo: target: %w", ErrOutOfRange)
}

// exchangeWithPlanet moves cargo between a fleet and its owner's planet.
func (g *Game) exchangeWithPlanet(f *Fleet, p *Planet, amounts [NumCargo + 1]int) {
	for c := range NumCargo {
		onPlanet := &p.Population
		if c < NumMinerals {
			onPlanet = &p.Surface[c]
		}
		switch v := amounts[c]; {
		case v > 0:
			n := min(v, *onPlanet, g.free(f, c))
			*onPlanet -= n
			*held(f, c) += n
		case v < 0:
			n := min(-v, *held(f, c))
			*held(f, c) -= n
			*onPlanet += n
		}
	}
}

// giveToPlanet gives cargo to a planet the player does not own: colonists
// become a drop or are lost, minerals a gift.
func (g *Game) giveToPlanet(f *Fleet, pi int, amounts [NumCargo + 1]int, a *Applied) {
	p := &g.Planets[pi]
	gift := CargoGift{From: f.Owner, FromFleet: f.ID, Target: TargetPlanet, ID: p.ID}
	any := false
	for c := range NumCargo {
		n := min(-amounts[c], *held(f, c))
		if n <= 0 {
			continue
		}
		*held(f, c) -= n
		if c != CargoColonists {
			gift.Amounts[c] = n
			any = true
			continue
		}
		if p.Owner == NoOwner || g.hasStarbase(pi) {
			a.Events = append(a.Events, Event{Kind: EventColonistsLostGiven, Player: f.Owner, Planet: p.ID, Fleet: f.ID, Count: n})
		} else {
			a.drops = append(a.drops, drop{planet: pi, player: f.Owner, troops: n})
		}
	}
	if any {
		a.Gifts = append(a.Gifts, gift)
	}
}

// creditGifts is the second pass of the replay (ORDERS.md "Cross-owner
// cargo", "Two passes, in place"; TAKEOVER.md "Manual cargo transfers to
// other players"): every debit was taken as its order applied, and each
// gift, in the order given, is now credited in place, still within the
// orders step and before movement. There is no relation check. A planet
// takes all its minerals with no message (MEASURED, TK-405, TK-412). A
// fleet takes what fits its free hold and tank (MEASURED, TK-406, TK-407,
// TK-409); the remainder is lost, not returned, and the giver is told
// (ORDERS.md "Receiver short of room", CONFIRMED).
//
// A gift naming a receiver that is missing when its order is replayed,
// including one removed by an earlier order the same turn (merged,
// scrapped, or removed in a player's replay that ran first), is rejected
// there: the record is skipped whole and the giver keeps the cargo, as
// nothing is debited (ORDERS.md "Cross-owner cargo", "Missing endpoint",
// stars-elegy #87; BINARY-ONLY for the same-turn removal).
//
// ASSUMPTION L9: a receiving fleet removed by a later order, after the
// debit and before the credit pass, is treated the same way: the record
// is skipped and what was taken goes back to the giver's fleet, as much
// as fits it now; anything that cannot go back (the giver's fleet is gone
// too, or has filled up since) is lost and the giver told. ORDERS.md does
// not cover a removal between the passes.
//
// Not modelled: the binary's separate queued cross-player credit routine,
// which no legal order is known to reach (ORDERS.md "A separate queued
// credit routine exists in the binary", UNRESOLVED).
func (g *Game) creditGifts(gifts []CargoGift) []Event {
	var events []Event
	for _, gift := range gifts {
		if gift.Target == TargetPlanet {
			p := &g.Planets[g.planetIndex(gift.ID)]
			for m := range NumMinerals {
				p.Surface[m] += gift.Amounts[m]
			}
			continue
		}
		lost := 0
		ti := g.fleetIndex(gift.ID)
		if ti < 0 {
			ti = g.fleetIndex(gift.FromFleet)
		}
		if ti < 0 {
			for _, v := range gift.Amounts {
				lost += v
			}
			events = append(events, Event{Kind: EventCargoGiftLost, Player: gift.From, Planet: -1, Fleet: gift.FromFleet, Count: lost})
			continue
		}
		t := &g.Fleets[ti]
		for c, v := range gift.Amounts {
			n := min(v, g.free(t, c))
			*held(t, c) += n
			lost += v - n
		}
		if lost > 0 {
			events = append(events, Event{Kind: EventCargoGiftLost, Player: gift.From, Planet: -1, Fleet: gift.FromFleet, Count: lost})
		}
	}
	return events
}

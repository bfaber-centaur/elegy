package engine

import "fmt"

// Cargo orders (stars-elegy ORDERS.md "Cargo amounts and clamps",
// "Cross-owner cargo", "Ownership"; TAKEOVER.md "Unload and load amounts"
// for the planet side).

// CargoFuel indexes fuel in CargoOrder.Amounts, after the minerals and
// colonists.
const CargoFuel = NumCargo

// Messages for cargo given to another player.
const (
	EventCargoGiven    EventKind = iota + EventGameLost + 1 // Player = receiver, Fleet = giver's fleet, Count = kT or mg credited
	EventCargoGiftLost                                      // Player = giver, Fleet = giver's fleet, Count = kT or mg lost
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
// "Ownership", chosen rule). Colonists given to a planet the player does
// not own become a colonist drop at the front of the before-movement drop
// queue (ORDERS.md "Cross-owner cargo" and TAKEOVER.md "Order inside a
// phase", BINARY-ONLY). Minerals and fuel given to another owner are
// taken from the fleet now and credited later (Applied.Gifts,
// DeliverGifts). Colonists are never given to another player's fleet,
// and nothing is given to a fleet whose owner regards the giver as an
// enemy (TAKEOVER.md "Other waypoint tasks").
//
// ASSUMPTION L7: the fleet must be at the target's position (in orbit
// of the planet); fuel to or from a planet rejects the order (KERNEL.md
// "Fuel cannot be unloaded onto a planet" covers the waypoint unload, not
// a direct transfer); any failed check rejects the whole order; and any
// fleet with cargo space may carry colonists (ORDERS.md says the
// condition exists but is not pinned).
//
// ASSUMPTION L8: the relation check is made when the order is applied,
// so nothing is taken from the giver.
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
		if o.Amounts[CargoFuel] != 0 {
			return fmt.Errorf("cargo: fuel and planet %d: %w", p.ID, ErrOutOfRange)
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
			case o.Amounts[CargoColonists] != 0, g.relation(t.Owner, player) == RelationEnemy:
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
// become a drop, minerals a gift.
func (g *Game) giveToPlanet(f *Fleet, pi int, amounts [NumCargo + 1]int, a *Applied) {
	gift := CargoGift{From: f.Owner, FromFleet: f.ID, Target: TargetPlanet, ID: g.Planets[pi].ID}
	any := false
	for c := range NumCargo {
		n := min(-amounts[c], *held(f, c))
		if n <= 0 {
			continue
		}
		*held(f, c) -= n
		if c == CargoColonists {
			a.drops = append(a.drops, drop{planet: pi, player: f.Owner, troops: n})
			continue
		}
		gift.Amounts[c] = n
		any = true
	}
	if any {
		a.Gifts = append(a.Gifts, gift)
	}
}

// DeliverGifts credits cargo given to other owners (ORDERS.md "Cross-owner
// cargo", BINARY-ONLY): a planet takes all of it; a fleet takes what fits
// its free hold and tank, and the rest is lost. The receiver is told what
// arrived, the giver what was lost.
//
// When this runs in the year is open: ORDERS.md says after movement,
// TAKEOVER.md "Where each task happens" step 2 says before movement,
// after loads and merges. GenerateTurn does not call it yet.
//
// ASSUMPTION L9: cargo for a fleet or planet that no longer exists is
// lost.
func (g *Game) DeliverGifts(gifts []CargoGift) []Event {
	var events []Event
	for _, gift := range gifts {
		credited, lost := 0, 0
		var receiver int
		switch gift.Target {
		case TargetPlanet:
			pi := g.planetIndex(gift.ID)
			if pi < 0 {
				for _, v := range gift.Amounts {
					lost += v
				}
				break
			}
			p := &g.Planets[pi]
			receiver = p.Owner
			for m := range NumMinerals {
				p.Surface[m] += gift.Amounts[m]
				credited += gift.Amounts[m]
			}
		case TargetFleet:
			ti := g.fleetIndex(gift.ID)
			if ti < 0 {
				for _, v := range gift.Amounts {
					lost += v
				}
				break
			}
			t := &g.Fleets[ti]
			receiver = t.Owner
			for c, v := range gift.Amounts {
				n := min(v, g.free(t, c))
				*held(t, c) += n
				credited += n
				lost += v - n
			}
		}
		if credited > 0 && receiver != NoOwner {
			events = append(events, Event{Kind: EventCargoGiven, Player: receiver, Planet: -1, Fleet: gift.FromFleet, Count: credited})
		}
		if lost > 0 {
			events = append(events, Event{Kind: EventCargoGiftLost, Player: gift.From, Planet: -1, Fleet: gift.FromFleet, Count: lost})
		}
	}
	return events
}

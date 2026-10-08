package objects

import (
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
)

// Space as the engine's engine.SpaceObjects: each method runs this
// package's rules at the turn step the engine calls it from and turns the
// records into engine events. The rules themselves stay in the other
// files.

var _ engine.SpaceObjects = (*Space)(nil)

// CloneObjects returns an independent copy of the space objects.
func (s *Space) CloneObjects() engine.SpaceObjects {
	c := *s
	c.Minefields = append([]Minefield(nil), s.Minefields...)
	c.Wormholes = append([]Wormhole(nil), s.Wormholes...)
	for i := range c.Wormholes {
		for e := range c.Wormholes[i].Ends {
			end := &c.Wormholes[i].Ends[e]
			end.Known = append([]bool(nil), end.Known...)
			end.DestKnown = append([]bool(nil), end.DestKnown...)
		}
	}
	c.Traders = append([]Trader(nil), s.Traders...)
	for i := range c.Traders {
		c.Traders[i].Served = append([]bool(nil), c.Traders[i].Served...)
	}
	c.TraderParts = append(TraderParts(nil), s.TraderParts...)
	c.GiftDesigns = append([]int(nil), s.GiftDesigns...)
	c.Packets = append([]Packet(nil), s.Packets...)
	return &c
}

// WormholeEndID is an end's waypoint target ID (engine.TargetWormhole).
func WormholeEndID(wi, ei int) int { return 2*wi + ei }

// ObjectPos is where a waypoint's wormhole end or Trader is now.
func (s *Space) ObjectPos(kind engine.TargetKind, id int) (engine.Point, bool) {
	switch kind {
	case engine.TargetWormhole:
		if wi, ei := id/2, id%2; id >= 0 && wi < len(s.Wormholes) {
			return s.Wormholes[wi].Ends[ei].Pos, true
		}
	case engine.TargetTrader:
		if id >= 0 && id < len(s.Traders) {
			return s.Traders[id].Pos, true
		}
	}
	return engine.Point{}, false
}

func width(g *engine.Game) int { return (g.Size + 1) * 400 }

func impactContext(g *engine.Game) ImpactContext {
	return ImpactContext{DefenseShare: g.BombSurvival}
}

// impacts turns packet impacts into events and empties the planets a
// packet left uninhabited (OBJECTS-STATUS.md, the packets' turn hooks).
func impacts(g *engine.Game, ims []Impact) []engine.Event {
	var out []engine.Event
	for _, im := range ims {
		p := &g.Planets[im.Planet]
		who := p.Owner
		if who < 0 {
			who = im.Owner
		}
		out = append(out, engine.Event{Kind: engine.EventPacketImpact, Player: who, Planet: p.ID, Fleet: -1, Count: im.Killed})
		if im.Emptied {
			out = append(out, g.EmptyPlanet(im.Planet))
		}
	}
	return out
}

// MoveObjects moves the Traders, then the packets in flight (OBJECTS.md
// "Turn placement" step 3). A waypoint on a Trader that left becomes a
// plain position where it left, and its owner is told; the others follow
// the Trader's new index.
func (s *Space) MoveObjects(g *engine.Game, rng engine.Rand) []engine.Event {
	moves := s.MoveTraders(g.Size, rng)
	var out []engine.Event
	if len(moves) > 0 {
		index := make([]int, len(moves))
		n := 0
		for i, mv := range moves {
			index[i] = -1
			if !mv.Left {
				index[i] = n
				n++
			}
		}
		for fi := range g.Fleets {
			f := &g.Fleets[fi]
			told := false
			for j := range f.Waypoints {
				wp := &f.Waypoints[j]
				if wp.Target != engine.TargetTrader || wp.ID < 0 || wp.ID >= len(moves) {
					continue
				}
				if ni := index[wp.ID]; ni >= 0 {
					wp.ID = ni
					continue
				}
				wp.Pos, wp.Target, wp.ID = moves[wp.ID].To, engine.TargetSpace, 0
				if !told {
					out = append(out, engine.Event{Kind: engine.EventTraderLeft, Player: f.Owner, Planet: -1, Fleet: f.ID})
					told = true
				}
			}
		}
	}
	return append(out, impacts(g, s.MovePackets(g, impactContext(g), rng))...)
}

// MineCheck is CheckStep for fleet fi.
func (s *Space) MineCheck(g *engine.Game, fi int, from, toward engine.Point, distance int, rng engine.Rand) (int, int, bool) {
	st := CheckStep(g, s, &g.Fleets[fi], from, toward, distance, rng)
	return st.Distance, int(st.Kind), st.Hit
}

// MineHit is ApplyHit for fleet fi; the destroyed ships' minerals become
// salvage at the stop point.
func (s *Space) MineHit(g *engine.Game, fi int, kind int, rng engine.Rand) []engine.Event {
	h := ApplyHit(g, s, fi, MineKind(kind), rng)
	f := &g.Fleets[fi]
	out := []engine.Event{{Kind: engine.EventMineHit, Player: f.Owner, Planet: -1, Fleet: f.ID, Count: h.Paid}}
	out = append(out, shipsLost(f, h.Designs)...)
	if h.Salvage != (engine.Minerals{}) {
		g.Salvage = append(g.Salvage, engine.Salvage{Pos: f.Pos, Minerals: h.Salvage})
	}
	return out
}

func shipsLost(f *engine.Fleet, hits []DesignHit) []engine.Event {
	n := 0
	for _, h := range hits {
		if h.Destroyed {
			n += h.Ships
		}
	}
	if n == 0 {
		return nil
	}
	return []engine.Event{{Kind: engine.EventMineShipsLost, Player: f.Owner, Planet: -1, Fleet: f.ID, Count: n}}
}

// TransitWormhole takes fleet fi through wormhole end `end`.
func (s *Space) TransitWormhole(g *engine.Game, fi int, end int) []engine.Event {
	if end < 0 || end/2 >= len(s.Wormholes) {
		return nil
	}
	s.Transit(g, fi, end/2, end%2)
	f := &g.Fleets[fi]
	return []engine.Event{{Kind: engine.EventWormholeTransit, Player: f.Owner, Planet: -1, Fleet: f.ID, Count: end}}
}

// DecayObjects is OBJECTS.md "Turn placement" step 5: packets decay,
// detonating fields go off, then every field decays.
func (s *Space) DecayObjects(g *engine.Game) []engine.Event {
	s.DecayPackets(g)
	var out []engine.Event
	for _, d := range s.Detonate(g) {
		f := &g.Fleets[d.Fleet]
		out = append(out, engine.Event{Kind: engine.EventMineDetonation, Player: f.Owner, Planet: -1, Fleet: f.ID})
		out = append(out, shipsLost(f, d.Designs)...)
	}
	s.Decay(g)
	return out
}

// TraderAppears is the Trader's appearance; every player is told.
func (s *Space) TraderAppears(g *engine.Game, rng engine.Rand) []engine.Event {
	if _, ok := s.Appear(g.Year-2400, g.Size, g.RandomEvents, rng); !ok {
		return nil
	}
	var out []engine.Event
	for p := range g.Players {
		out = append(out, engine.Event{Kind: engine.EventTraderAppeared, Player: p, Planet: -1, Fleet: -1, Count: len(s.Traders) - 1})
	}
	return out
}

// MoveObjectsAgain flies this year's packets half a year, then moves the
// wormholes (OBJECTS.md "Turn placement" step 7). A waypoint on a moved
// end keeps following it only when its owner knew the end before the
// move; otherwise it becomes a plain position at the end's old position
// (CONFIRMED OB-025-F, OB-027).
//
// ASSUMPTION O12: "knew the end at the start of the year" is read just
// before the wormholes move, so an end a fleet of the owner passed
// through earlier this year counts as known.
func (s *Space) MoveObjectsAgain(g *engine.Game, rng engine.Rand) []engine.Event {
	out := impacts(g, s.FlyLaunched(g, impactContext(g), rng))
	known := make([][2][]bool, len(s.Wormholes))
	for wi := range s.Wormholes {
		for ei := range s.Wormholes[wi].Ends {
			known[wi][ei] = append([]bool(nil), s.Wormholes[wi].Ends[ei].Known...)
		}
	}
	moves := s.MoveWormholes(g, width(g), rng)
	for _, mv := range moves {
		out = append(out, engine.Event{Kind: engine.EventWormholeMoved, Player: -1, Planet: -1, Fleet: -1, Count: WormholeEndID(mv.Wormhole, mv.End)})
	}
	for fi := range g.Fleets {
		f := &g.Fleets[fi]
		for j := range f.Waypoints {
			wp := &f.Waypoints[j]
			if wp.Target != engine.TargetWormhole || wp.ID < 0 || wp.ID/2 >= len(s.Wormholes) {
				continue
			}
			wi, ei := wp.ID/2, wp.ID%2
			if has(known[wi][ei], f.Owner) {
				wp.Pos = s.Wormholes[wi].Ends[ei].Pos
				continue
			}
			for _, mv := range moves {
				if mv.Wormhole == wi && mv.End == ei {
					wp.Pos, wp.Target, wp.ID = mv.From, engine.TargetSpace, 0
				}
			}
		}
	}
	return out
}

// MeetTraders runs the Trader encounters.
//
// PLACEHOLDER: the engine has no computer players, so every player counts
// as human (Humans is the player count) and no planet trades.
func (s *Space) MeetTraders(g *engine.Game, rng engine.Rand) []engine.Event {
	ctx := TraderContext{YearIndex: g.Year - 2400, Humans: len(g.Players)}
	enc, trades := s.Meet(g, ctx, rng)
	var out []engine.Event
	for _, e := range enc {
		ev := engine.Event{Player: e.Owner, Planet: -1, Fleet: e.Fleet}
		switch {
		case e.Refused || e.Recovering:
			ev.Kind = engine.EventTraderRefused
			if e.Recovering {
				ev.Count = 1
			}
		default:
			ev.Kind, ev.Count = engine.EventTraderTrade, int(e.Reward.Kind)
		}
		out = append(out, ev)
		switch {
		case e.Reward.NoRoom:
			out = append(out, engine.Event{Kind: engine.EventTraderNoRoom, Player: e.Owner, Planet: -1, Fleet: e.Fleet})
		case e.Reward.NewFleet >= 0 && e.Reward.Kind == ItemShip && !e.Refused && !e.Recovering:
			out = append(out, engine.Event{Kind: engine.EventTraderGift, Player: e.Owner, Planet: -1, Fleet: e.Reward.NewFleet, Count: e.Reward.Ships})
		}
	}
	for _, t := range trades {
		out = append(out, engine.Event{Kind: engine.EventTraderPlanetTrade, Player: t.Owner, Planet: g.Planets[t.Planet].ID, Fleet: -1})
	}
	return out
}

// LayMines lays mines for the given fleets, one fleet at a time in fleet order
// (OBJECTS.md "Laying", "Order"). Salvage objects count toward the object
// limits as OtherObjects.
func (s *Space) LayMines(g *engine.Game, layers []engine.MineLayer) []engine.Event {
	s.OtherObjects = len(g.Salvage)
	order := append([]engine.MineLayer(nil), layers...)
	sort.SliceStable(order, func(i, j int) bool {
		a, b := &g.Fleets[order[i].Fleet], &g.Fleets[order[j].Fleet]
		if a.Owner != b.Owner {
			return a.Owner < b.Owner
		}
		if a.Number != b.Number {
			return a.Number < b.Number
		}
		return a.ID < b.ID
	})
	var out []engine.Event
	for _, l := range order {
		f := &g.Fleets[l.Fleet]
		if LayAmounts(g, f) == ([NumMineKinds]int{}) {
			out = append(out, engine.Event{Kind: engine.EventNoDispensers, Player: f.Owner, Planet: -1, Fleet: f.ID})
			continue
		}
		for _, r := range s.Lay(g, []Layer{{Fleet: l.Fleet, Half: l.Half}}) {
			kind := engine.EventMinesLaid
			if r.Field < 0 {
				kind = engine.EventMinesNotLaid
			}
			out = append(out, engine.Event{Kind: kind, Player: f.Owner, Planet: -1, Fleet: f.ID, Count: r.Amount})
		}
	}
	return out
}

// SweepMines is the year's mine sweeping.
func (s *Space) SweepMines(g *engine.Game) []engine.Event {
	var out []engine.Event
	for _, w := range s.Sweep(g) {
		ev := engine.Event{Kind: engine.EventMinesSwept, Player: w.Sweeper, Planet: -1, Fleet: -1, Count: w.Mines}
		if w.Fleet >= 0 {
			ev.Fleet = g.Fleets[w.Fleet].ID
		}
		if w.Planet >= 0 {
			ev.Planet = g.Planets[w.Planet].ID
		}
		out = append(out, ev)
	}
	return out
}

// SeeObjects marks the wormhole ends each player sees this year as known
// (SCANNING.md "Space objects", CONFIRMED OB-011, OB-017, OB-018): never
// beyond a scanner's normal range R; within it, an end is seen when the
// player already knows it, or d² ≤ P², or d² ≤ ⌊R²/16⌋. Minefield
// knowledge is not kept yet.
func (s *Space) SeeObjects(g *engine.Game, scanners []engine.ObjectScanner) {
	for wi := range s.Wormholes {
		for ei := range s.Wormholes[wi].Ends {
			e := &s.Wormholes[wi].Ends[ei]
			for _, sc := range scanners {
				d := d2(e.Pos, sc.Pos)
				if d <= sc.R*sc.R && (e.KnownBy(sc.Player) || d <= sc.P*sc.P || d <= sc.R*sc.R/16) {
					e.MarkKnown(sc.Player)
				}
			}
		}
	}
}

package objects

import "github.com/bfaber-centaur/elegy/engine"

// Origin is the galaxy's lowest coordinate on both axes (UNIVERSE.md
// "Conventions").
const Origin = 1000

// Wormhole is a pair of linked ends (OBJECTS.md "Wormholes").
type Wormhole struct {
	Ends [2]WormholeEnd
}

// WormholeEnd is one end of a wormhole.
type WormholeEnd struct {
	Pos engine.Point
	// Class is the end's stability class, 0..2 at creation; it never
	// changes (CONFIRMED OB-025).
	Class int
	// Years since the end last jumped.
	Years int
	// Known marks the players who know where this end is (scanning,
	// SCANNING.md, or transit). A jump clears it.
	Known []bool
	// DestKnown marks the players who know where this end leads. Only
	// transit sets it, on both ends; it is never cleared (CONFIRMED
	// WT-001 A, G; WT-004).
	DestKnown []bool
}

func mark(s []bool, player int) []bool {
	if player < 0 {
		return s
	}
	for len(s) <= player {
		s = append(s, false)
	}
	s[player] = true
	return s
}

func has(s []bool, player int) bool { return player >= 0 && player < len(s) && s[player] }

// KnownBy reports whether a player knows where the end is.
func (e WormholeEnd) KnownBy(player int) bool { return has(e.Known, player) }

// DestinationKnownBy reports whether a player knows where the end leads.
func (e WormholeEnd) DestinationKnownBy(player int) bool { return has(e.DestKnown, player) }

// MarkKnown records that a player knows where the end is (the scanning
// code calls this for a sighting).
func (e *WormholeEnd) MarkKnown(player int) { e.Known = mark(e.Known, player) }

// WormholePairs is (v, m) by universe size, tiny to huge: rand(v) + m
// pairs at creation (OBJECTS.md "Creation", CONFIRMED OB-006, UG01–UG21).
var WormholePairs = [5][2]int{{3, 0}, {3, 1}, {5, 1}, {4, 3}, {5, 4}}

// PlacementTries is the number of position tries per placement
// (OBJECTS.md "Creation", "Yearly movement").
const PlacementTries = 100

// Rejected is the badness of a try that is rejected outright; it can only
// be kept when every try was rejected (OBJECTS.md "Placement badness").
const Rejected = 15

// bandFlag is the flag a distance band sets: 8 for the closest band, then
// 4, 2, 1; 0 outside every band.
func bandFlag(d int, limits [4]int) int {
	for i, l := range limits {
		if d < l {
			return 8 >> i
		}
	}
	return 0
}

// Surroundings are what a wormhole end's placement is judged against.
type Surroundings struct {
	Width int // galaxy width W
	// Partner is the end's partner, if placed.
	Partner *engine.Point
	// Ends are every other wormhole end; Planets the planets; Objects
	// the positions a try may not land on exactly (fleets and other
	// space objects).
	Ends    []engine.Point
	Planets []engine.Point
	Objects []engine.Point
}

// Badness is a try's placement badness (OBJECTS.md "Placement badness",
// CONFIRMED at creation, UG01–UG21): a set of four flags OR-ed and
// compared as 0–15. A try outside the galaxy or exactly on a planet,
// fleet, object or other end is Rejected. Within 10 ly of an edge sets 4;
// the partner's bands (d² < 25, 100, 900, 4900), other ends' (< 16, 64,
// 225, 900) and planets' (< 25, 100, 400, 784) set 8, 4, 2, 1 from the
// closest band.
//
// Outside the galaxy is a coordinate below 1000 or above 1000 + W; one
// equal to 1000 + W is inside (BINARY-ONLY).
func (sr Surroundings) Badness(p engine.Point) int {
	if p.X < Origin || p.Y < Origin || p.X > Origin+sr.Width || p.Y > Origin+sr.Width {
		return Rejected
	}
	for _, list := range [][]engine.Point{sr.Planets, sr.Objects, sr.Ends} {
		for _, q := range list {
			if q == p {
				return Rejected
			}
		}
	}
	if sr.Partner != nil && *sr.Partner == p {
		return Rejected
	}
	bad := 0
	lo, hi := Origin+10, Origin+sr.Width-10
	if p.X < lo || p.X > hi || p.Y < lo || p.Y > hi {
		bad |= 4
	}
	if sr.Partner != nil {
		bad |= bandFlag(d2(p, *sr.Partner), [4]int{25, 100, 900, 4900})
	}
	for _, e := range sr.Ends {
		bad |= bandFlag(d2(p, e), [4]int{16, 64, 225, 900})
	}
	for _, pl := range sr.Planets {
		bad |= bandFlag(d2(p, pl), [4]int{25, 100, 400, 784})
	}
	return bad
}

// Place makes up to PlacementTries tries and keeps the first with the
// lowest badness, stopping at a try with none; a rejected try (15) beats
// no try at all. skip, if set, passes over a try without scoring it, the
// try still using up one of the PlacementTries (OBJECTS.md "During
// movement", BINARY-ONLY). ok is false only when every try was skipped.
func (sr Surroundings) Place(try func() engine.Point, skip func(engine.Point) bool) (p engine.Point, ok bool) {
	bestBad := Rejected + 1
	for range PlacementTries {
		q := try()
		if skip != nil && skip(q) {
			continue
		}
		bad := sr.Badness(q)
		if bad < bestBad {
			p, bestBad, ok = q, bad, true
		}
		if bad == 0 {
			break
		}
	}
	return p, ok
}

// UniformTry is a uniform try over the galaxy, 1000 + rand(W) per axis.
func UniformTry(w int, rng engine.Rand) func() engine.Point {
	return func() engine.Point {
		return engine.Point{X: Origin + rng.Intn(w), Y: Origin + rng.Intn(w)}
	}
}

// JumpChance is an end's yearly jump chance in percent (OBJECTS.md
// "Yearly movement", MEASURED OB-025): clamp(⌊years/5⌋ + class − 2, 0, 6).
func JumpChance(e WormholeEnd) int {
	return max(0, min(6, e.Years/5+e.Class-2))
}

// stabilityNames are the report's stability names by jump chance
// (OBJECTS.md "Stability", BINARY-ONLY).
var stabilityNames = [7]string{"Rock Solid", "Stable", "Mostly Stable", "Average", "Slightly Volatile", "Volatile", "Extremely Volatile"}

// StabilityName is the stability a player sees for an end: its current
// jump chance, by name.
func StabilityName(e WormholeEnd) string { return stabilityNames[JumpChance(e)] }

// WormholeMove is what one end did in the yearly movement.
type WormholeMove struct {
	Wormhole, End int
	From, To      engine.Point
	Jumped        bool
}

// surroundings are the placement surroundings of end (wi, ei) now.
func (s *Space) surroundings(g *engine.Game, width, wi, ei int) Surroundings {
	sr := Surroundings{Width: width}
	for i, wh := range s.Wormholes {
		for j, e := range wh.Ends {
			if i == wi && j == ei {
				continue
			}
			if i == wi {
				p := e.Pos
				sr.Partner = &p
				continue
			}
			sr.Ends = append(sr.Ends, e.Pos)
		}
	}
	for _, p := range g.Planets {
		sr.Planets = append(sr.Planets, p.Pos)
	}
	for _, f := range g.Fleets {
		sr.Objects = append(sr.Objects, f.Pos)
	}
	for _, m := range s.Minefields {
		sr.Objects = append(sr.Objects, m.Pos)
	}
	for _, t := range s.Traders {
		sr.Objects = append(sr.Objects, t.Pos)
	}
	return sr
}

// MoveWormholes runs the yearly wormhole movement after fleets move
// (OBJECTS.md "Yearly movement", CONFIRMED in part OB-005, OB-017,
// OB-025): each end in list order draws rand(100) < JumpChance to jump.
// A jump resets the years, clears who knows the end and places it from
// up to 100 uniform tries over the galaxy. Otherwise the end jiggles:
// years + 1, up to 100 tries of (x + rand(25) − 12, y + rand(25) − 12),
// a try equal to the old position skipped. The class never changes;
// destination knowledge is kept.
//
// Both moves judge their tries with the creation badness (partner, the
// other ends as they stand, planets; fleets, minefield centres and
// Traders as objects), each try drawing x before y; if every scored try
// is rejected the end moves to the first of them (OBJECTS.md "During
// movement", BINARY-ONLY).
func (s *Space) MoveWormholes(g *engine.Game, width int, rng engine.Rand) []WormholeMove {
	var out []WormholeMove
	for wi := range s.Wormholes {
		for ei := range s.Wormholes[wi].Ends {
			e := &s.Wormholes[wi].Ends[ei]
			from := e.Pos
			sr := s.surroundings(g, width, wi, ei)
			mv := WormholeMove{Wormhole: wi, End: ei, From: from}
			if rng.Intn(100) < JumpChance(*e) {
				e.Years = 0
				e.Known = nil
				e.Pos, _ = sr.Place(UniformTry(width, rng), nil)
				mv.Jumped = true
			} else {
				e.Years++
				try := func() engine.Point {
					x := from.X + rng.Intn(25) - 12
					y := from.Y + rng.Intn(25) - 12
					return engine.Point{X: x, Y: y}
				}
				if p, ok := sr.Place(try, func(p engine.Point) bool { return p == from }); ok {
					e.Pos = p
				}
			}
			mv.To = e.Pos
			out = append(out, mv)
		}
	}
	return out
}

// Transit moves fleet fi through end (wi, ei) of a wormhole to the
// partner end (OBJECTS.md "Travel", CONFIRMED OB-005 C, D, WT-001): the
// fleet lands on the partner's position (wormholes move after fleets, so
// this is the position from before this year's wormhole movement); both
// ends become known to its owner and record that the owner knows where
// they lead. Fuel, mass and stability play no part and there is no
// damage. The caller transits only a fleet whose reached waypoint targets
// the wormhole itself, and makes other players' fleets that followed it
// lose it.
func (s *Space) Transit(g *engine.Game, fi, wi, ei int) {
	f := &g.Fleets[fi]
	wh := &s.Wormholes[wi]
	in, out := &wh.Ends[ei], &wh.Ends[1-ei]
	f.Pos = out.Pos
	for _, e := range []*WormholeEnd{in, out} {
		e.Known = mark(e.Known, f.Owner)
		e.DestKnown = mark(e.DestKnown, f.Owner)
	}
}

// Destination is what a player sees as an end's destination (OBJECTS.md
// "Destination knowledge", CONFIRMED WT-001, WT-004; report BINARY-ONLY):
// the partner's position when the player knows where the end leads and
// sees the partner this year, else unknown.
func (s *Space) Destination(wi, ei, player int, partnerSeen bool) (engine.Point, bool) {
	e := s.Wormholes[wi].Ends[ei]
	if !e.DestinationKnownBy(player) || !partnerSeen {
		return engine.Point{}, false
	}
	return s.Wormholes[wi].Ends[1-ei].Pos, true
}

package objects

import (
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
)

// Scanner is one of a player's viewing objects, built by the turn
// engine from its fleets and planets: position, normal range R and
// penetrating range P in ly (SCANNING.md "Scanner ranges"). Fleet marks
// a fleet; only a fleet sees a minefield it sits inside.
type Scanner struct {
	Pos   engine.Point
	R, P  int
	Fleet bool
}

// WormholeSighting names one wormhole end.
type WormholeSighting struct{ Wormhole, End int }

// Sightings is what one player sees of the space objects this year
// (SCANNING.md "Space objects"). Fields are indices into the Space
// slices, in slice order.
type Sightings struct {
	Minefields []int
	Packets    []int
	Wormholes  []WormholeSighting
	Traders    []int
	// Salvage indexes engine.Game.Salvage.
	Salvage []int
	// Owners are the other players made known by a sighted minefield,
	// packet or salvage object, ascending.
	Owners []int
}

// PacketScanners are player v's packet scanners: for a Packet Physics
// player, each of its own packets in flight scans as a penetrating
// scanner with R = P = warp² ly (SCANNING.md "PP packet scanners",
// CONFIRMED OB-012). They see fleets, planets and space objects; the turn
// engine adds them to its own scanners for fleets and planets. Scan
// adds them itself.
//
// ASSUMPTION V1: every packet the player owns at the end of the year is
// in flight, including one launched this year.
func (s *Space) PacketScanners(g *engine.Game, v int) []Scanner {
	if prt(g, v) != engine.PRTPacketPhysics {
		return nil
	}
	var out []Scanner
	for _, p := range s.Packets {
		if p.Owner == v {
			r := p.Warp * p.Warp
			out = append(out, Scanner{Pos: p.Pos, R: r, P: r})
		}
	}
	return out
}

// Scan computes what player v sees of the space objects at the end of
// the year (SCANNING.md "Space objects", CONFIRMED OB-011..OB-014,
// OB-017, OB-018) from v's fleet and planet scanners. It adds v's packet
// scanners itself (PacketScanners) and records new knowledge: every
// minefield and wormhole end seen becomes known to v. Players are
// independent, so the order in which the turn engine scans them does not
// matter.
//
//   - A minefield is seen when d² ≤ P², d² ≤ ⌊R²/16⌋, the viewing fleet
//     is inside it (fleets only, BINARY-ONLY), or v already knows it and
//     d² ≤ R². This applies to v's own fields too: ownership never makes
//     a field known (SCANNING.md "Space objects", MEASURED MF-13a, MF-13c).
//     Own fields are still always listed in Sightings, as the turn file
//     lists them in the owner's view whatever the known set says.
//   - A wormhole end is seen only within d² ≤ R², and then when it is
//     known, d² ≤ P² or d² ≤ ⌊R²/16⌋.
//   - Own packets are always seen; others within d² ≤ R²; a PP player
//     sees every packet (OB-012).
//   - Every Trader is seen by everyone (OB-011-J).
//   - Salvage is seen as another player's packet is, its owner's own
//     included (OBJECTS.md "Salvage", "Visibility").
//   - Seeing another player's minefield, packet or salvage makes that
//     player known; wormholes and Traders have no owner (OB-017).
func (s *Space) Scan(g *engine.Game, v int, scanners []Scanner) Sightings {
	scanners = append(append([]Scanner(nil), scanners...), s.PacketScanners(g, v)...)
	var out Sightings
	owners := map[int]bool{}
	for i := range s.Minefields {
		m := &s.Minefields[i]
		seen := seesMinefield(*m, v, scanners)
		if seen {
			m.learn(v)
		}
		if m.Owner != v && !seen {
			continue
		}
		out.Minefields = append(out.Minefields, i)
		if m.Owner != v {
			owners[m.Owner] = true
		}
	}
	pp := prt(g, v) == engine.PRTPacketPhysics
	for i, p := range s.Packets {
		if p.Owner != v && !pp && !within(p.Pos, scanners, func(sc Scanner) int { return sc.R * sc.R }) {
			continue
		}
		out.Packets = append(out.Packets, i)
		if p.Owner != v && p.Owner >= 0 {
			owners[p.Owner] = true
		}
	}
	for wi := range s.Wormholes {
		for ei := range s.Wormholes[wi].Ends {
			e := &s.Wormholes[wi].Ends[ei]
			if !seesWormhole(*e, v, scanners) {
				continue
			}
			out.Wormholes = append(out.Wormholes, WormholeSighting{wi, ei})
			e.MarkKnown(v)
		}
	}
	for i := range s.Traders {
		out.Traders = append(out.Traders, i)
	}
	// Salvage is seen as a packet is, with no exception for its owner's
	// own; it never scans (OBJECTS.md "Salvage", "Visibility").
	for i, sv := range g.Salvage {
		if !pp && !within(sv.Pos, scanners, func(sc Scanner) int { return sc.R * sc.R }) {
			continue
		}
		out.Salvage = append(out.Salvage, i)
		if sv.Owner != v && sv.Owner >= 0 {
			owners[sv.Owner] = true
		}
	}
	for o := range owners {
		out.Owners = append(out.Owners, o)
	}
	sort.Ints(out.Owners)
	return out
}

// within reports whether some scanner has pos within d² ≤ limit(scanner).
func within(pos engine.Point, scanners []Scanner, limit func(Scanner) int) bool {
	for _, sc := range scanners {
		if d2(sc.Pos, pos) <= limit(sc) {
			return true
		}
	}
	return false
}

func seesMinefield(m Minefield, v int, scanners []Scanner) bool {
	known := m.KnownBy(v)
	for _, sc := range scanners {
		dd := d2(sc.Pos, m.Pos)
		if dd <= sc.P*sc.P || dd <= sc.R*sc.R/16 || (sc.Fleet && m.Contains(sc.Pos)) || (known && dd <= sc.R*sc.R) {
			return true
		}
	}
	return false
}

func seesWormhole(e WormholeEnd, v int, scanners []Scanner) bool {
	known := e.KnownBy(v)
	for _, sc := range scanners {
		dd := d2(sc.Pos, e.Pos)
		if dd <= sc.R*sc.R && (known || dd <= sc.P*sc.P || dd <= sc.R*sc.R/16) {
			return true
		}
	}
	return false
}

// KnownBy reports whether a player knows the minefield.
func (m Minefield) KnownBy(player int) bool { return has(m.Known, player) }

// learn records that a player knows the field. It copies Known before
// changing it, so a copy of the field made with a shallow copy of
// Space.Minefields (as Space.CloneObjects makes) never sees the change.
func (m *Minefield) learn(player int) {
	if player < 0 || m.KnownBy(player) {
		return
	}
	m.Known = mark(append([]bool(nil), m.Known...), player)
}

// LearnHit records that fleet f's owner learned the minefields that hit
// it: a player knows a field once it has been hit by it (SCANNING.md
// "Space objects", BINARY-ONLY). stop is the Stop CheckStep returned and
// pos the point where the fleet stopped.
//
// ASSUMPTION V2: overlapping fields of one kind form one stretch in
// CheckStep, so the player learns every field of the hit kind that
// contains the stop point and could stop the fleet.
func (s *Space) LearnHit(g *engine.Game, f *engine.Fleet, stop Stop, pos engine.Point) {
	if !stop.Hit {
		return
	}
	for i := range s.Minefields {
		m := &s.Minefields[i]
		if m.Kind == stop.Kind && m.Count > 0 && m.Contains(pos) && stopsFleet(g, *m, f.Owner) {
			m.learn(f.Owner)
		}
	}
}

// DemolitionSightings are the other players' fleets that player v's
// Space Demolition minefields see (SCANNING.md "Space Demolition (SD)
// minefields", CONFIRMED OB-014-B): every fleet inside one of v's fields
// that is not orbiting a planet. A cloaked fleet (cloak c > 0, from the
// turn engine's cloak rule) is seen only when rand(100) ≥ c
// (BINARY-ONLY).
//
// ASSUMPTION V3: "enemy" reads as any other player's fleet, as elsewhere
// in SCANNING.md; a fleet is orbiting when it is at a planet's position;
// fleets are considered in fleet order (owner, then number), each making
// at most one draw however many fields it is inside.
func (s *Space) DemolitionSightings(g *engine.Game, v int, cloak func(fleet int) int, rng engine.Rand) []int {
	if prt(g, v) != engine.PRTSpaceDemolition {
		return nil
	}
	planets := map[engine.Point]bool{}
	for _, p := range g.Planets {
		planets[p.Pos] = true
	}
	var out []int
	for _, fi := range fleetOrder(g) {
		f := &g.Fleets[fi]
		if f.Owner == v || planets[f.Pos] || !s.insideOwn(v, f.Pos) {
			continue
		}
		if c := cloak(fi); c > 0 && rng.Intn(100) < c {
			continue
		}
		out = append(out, fi)
	}
	sort.Ints(out)
	return out
}

func (s *Space) insideOwn(v int, pos engine.Point) bool {
	for _, m := range s.Minefields {
		if m.Owner == v && m.Count > 0 && m.Contains(pos) {
			return true
		}
	}
	return false
}

package objects

import "github.com/bfaber-centaur/elegy/engine"

// ObjectReport is what a player is shown of the minefields, packets,
// Mystery Traders and salvage it sees in a year: one entry per object in
// engine.ObjectsSeen, in that order. Wormhole ends are reported apart
// (game.KnownWormholeEnd).
//
// Who sees which object is SCANNING.md "Space objects" (CONFIRMED). What
// a sighting shows is SCANNING.md "Disclosure", "Space objects"
// (MEASURED, SC-038 in PARITY.md "Space objects in player files"): the
// whole object as the host holds it, at any distance, whoever owns it,
// except the per-player "known" marker, of which the viewer gets only its
// own entry (Minefield.KnownBy). A report holds only the objects seen that
// year; nothing carries over (SC-038, computer players' history files).
//
// ASSUMPTION V4 covers what SC-038 leaves open. A player's own packets are
// always in its report (Space.Scan; own minefields are, MEASURED MF-13);
// SC-038 saw own packets beyond every scanner missing from their owner's
// file once (OB-017 D–F), a single observation Elegy does not adopt. A
// packet's "decay state and whether it has moved" are Elegy's decay class
// and launched-this-year mark. Elegy keeps no per-year "seen" marker, so
// none is reported.
type ObjectReport struct {
	Minefields []MinefieldSighting
	Packets    []PacketSighting
	Traders    []TraderSighting
	Salvage    []SalvageSighting
}

// MinefieldSighting is a minefield as a player is shown it: owner and
// number, centre, mine count, kind and detonate setting (SC-038). Known is
// the viewer's own "known" entry.
type MinefieldSighting struct {
	Owner, Number int
	Pos           engine.Point
	Mines         int
	Kind          MineKind
	Detonate      bool
	Known         bool
}

// PacketSighting is a mineral packet as a player is shown it: owner and
// number, position, destination planet ID (Target), warp, cargo, decay
// class and whether it was launched this year (New: it has not yet made
// its first full move) (SC-038: destination, warp, minerals, decay state
// and whether it has moved).
type PacketSighting struct {
	Owner, Number int
	Pos           engine.Point
	Target, Warp  int
	Cargo         engine.Minerals
	Class         int
	New           bool
}

// TraderSighting is a Mystery Trader as every player is shown it: its
// index in object order, position, destination, warp, the players it has
// served and the item it carries (SC-038).
type TraderSighting struct {
	Index     int
	Pos, Dest engine.Point
	Warp      int
	Served    []bool
	Item      TraderItem
}

// SalvageSighting is a salvage object as a player is shown it: owner and
// number, position and minerals (SC-038). Seeing it makes its owner known
// (OBJECTS.md "Salvage", "Visibility").
type SalvageSighting struct {
	Owner, Number int
	Pos           engine.Point
	Minerals      engine.Minerals
}

// Report is player v's ObjectReport for the objects in seen, the
// engine's record of what v saw this year (Space.SeeObjects). Run it on
// the state the year ended with. An object no longer present is left
// out.
func (s *Space) Report(g *engine.Game, v int, seen engine.ObjectsSeen) ObjectReport {
	var out ObjectReport
	for _, k := range seen.Minefields {
		for _, m := range s.Minefields {
			if m.Owner == k[0] && m.Number == k[1] {
				out.Minefields = append(out.Minefields, MinefieldSighting{Owner: m.Owner, Number: m.Number, Pos: m.Pos, Mines: m.Count, Kind: m.Kind, Detonate: m.Detonate, Known: m.KnownBy(v)})
				break
			}
		}
	}
	for _, k := range seen.Packets {
		for _, p := range s.Packets {
			if p.Owner == k[0] && p.Number == k[1] {
				out.Packets = append(out.Packets, PacketSighting{Owner: p.Owner, Number: p.Number, Pos: p.Pos, Target: p.Target, Warp: p.Warp, Cargo: p.Cargo, Class: p.Class, New: p.New})
				break
			}
		}
	}
	for _, i := range seen.Traders {
		if i >= 0 && i < len(s.Traders) {
			t := s.Traders[i]
			out.Traders = append(out.Traders, TraderSighting{Index: i, Pos: t.Pos, Dest: t.Dest, Warp: t.Warp, Served: append([]bool(nil), t.Served...), Item: t.Item})
		}
	}
	for _, k := range seen.Salvage {
		for _, sv := range g.Salvage {
			if sv.Owner == k[0] && sv.Number == k[1] {
				out.Salvage = append(out.Salvage, SalvageSighting{Owner: sv.Owner, Number: sv.Number, Pos: sv.Pos, Minerals: sv.Minerals})
				break
			}
		}
	}
	return out
}

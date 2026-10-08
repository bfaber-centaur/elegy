package objects

import "github.com/bfaber-centaur/elegy/engine"

// ObjectReport is what a player is shown of the minefields, packets,
// Mystery Traders and salvage it sees in a year: one entry per object in
// engine.ObjectsSeen, in that order. Wormhole ends are reported apart
// (game.KnownWormholeEnd).
//
// Who sees which object is SCANNING.md "Space objects" (CONFIRMED). What
// a sighting shows is not in stars-elegy yet (asked of research on
// 2026-10-08), so the contents follow ASSUMPTION V4.
//
// ASSUMPTION V4: a sighting shows an object's owner, number and position.
// A minefield also shows its mine count and kind: computer players react
// to other players' fields by their mine count (AI.md "Warp choice",
// CONFIRMED AI-11) and kind (heavy, standard; speed bumps ignored), which
// they could not do without them. The owner's own minefields also show
// the detonate setting, and the owner's own packets their warp,
// destination and cargo, which the owner set or launched. Nothing else is
// shown: not another player's packet's warp, destination or cargo, not
// salvage contents, not a Trader's warp, destination or item, and not who
// else knows a field.
type ObjectReport struct {
	Minefields []MinefieldSighting
	Packets    []PacketSighting
	Traders    []TraderSighting
	Salvage    []SalvageSighting
}

// MinefieldSighting is a minefield as a player is shown it. Detonate is
// set only in its owner's report.
type MinefieldSighting struct {
	Owner, Number int
	Pos           engine.Point
	Mines         int
	Kind          MineKind
	Detonate      bool
}

// PacketSighting is a mineral packet as a player is shown it. Own is
// set, with the packet's warp, destination planet ID and cargo, only in
// its owner's report.
type PacketSighting struct {
	Owner, Number int
	Pos           engine.Point
	Own           *OwnPacket `json:",omitempty"`
}

// OwnPacket is what a packet's owner is shown besides its position.
type OwnPacket struct {
	Warp, Target int
	Cargo        engine.Minerals
}

// TraderSighting is a Mystery Trader as every player is shown it: its
// index in object order and its position.
type TraderSighting struct {
	Index int
	Pos   engine.Point
}

// SalvageSighting is a salvage object as a player is shown it. Seeing it
// makes its owner known (OBJECTS.md "Salvage", "Visibility").
type SalvageSighting struct {
	Owner, Number int
	Pos           engine.Point
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
				out.Minefields = append(out.Minefields, MinefieldSighting{Owner: m.Owner, Number: m.Number, Pos: m.Pos, Mines: m.Count, Kind: m.Kind, Detonate: m.Owner == v && m.Detonate})
				break
			}
		}
	}
	for _, k := range seen.Packets {
		for _, p := range s.Packets {
			if p.Owner == k[0] && p.Number == k[1] {
				ps := PacketSighting{Owner: p.Owner, Number: p.Number, Pos: p.Pos}
				if p.Owner == v {
					ps.Own = &OwnPacket{Warp: p.Warp, Target: p.Target, Cargo: p.Cargo}
				}
				out.Packets = append(out.Packets, ps)
				break
			}
		}
	}
	for _, i := range seen.Traders {
		if i >= 0 && i < len(s.Traders) {
			out.Traders = append(out.Traders, TraderSighting{Index: i, Pos: s.Traders[i].Pos})
		}
	}
	for _, k := range seen.Salvage {
		for _, sv := range g.Salvage {
			if sv.Owner == k[0] && sv.Number == k[1] {
				out.Salvage = append(out.Salvage, SalvageSighting{Owner: sv.Owner, Number: sv.Number, Pos: sv.Pos})
				break
			}
		}
	}
	return out
}

package game

import (
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

// history is every player's planet history (Report.History), by player
// and planet id.
type history []map[int]PlanetRecord

// record adds a year's views to the history: every planet report above
// ReportNone replaces that planet's record (ASSUMPTION G1). year is the
// year the views describe.
func (h history) record(year int, views []engine.PlayerView) history {
	for len(h) < len(views) {
		h = append(h, map[int]PlanetRecord{})
	}
	for v, view := range views {
		for _, r := range view.Planets {
			if r.Level == engine.ReportNone {
				continue
			}
			h[v][r.Planet] = PlanetRecord{Year: year, Report: deepCopy(r)}
		}
	}
	return h
}

// list is player's history in planet id order, a copy.
func (h history) list(player int) []PlanetRecord {
	if player >= len(h) {
		return nil
	}
	ids := make([]int, 0, len(h[player]))
	for id := range h[player] {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]PlanetRecord, len(ids))
	for i, id := range ids {
		out[i] = deepCopy(h[player][id])
	}
	return out
}

// clone is an independent copy of h.
func (h history) clone() history {
	out := make(history, len(h))
	for p, m := range h {
		out[p] = make(map[int]PlanetRecord, len(m))
		for id, r := range m {
			out[p][id] = r
		}
	}
	return out
}

// wormholeRecord is what a player last saw of a wormhole end.
type wormholeRecord struct {
	End       int          `json:"end"`
	Year      int          `json:"year"`
	Pos       engine.Point `json:"pos"`
	Stability int          `json:"stability"`
}

// wormholeHistory is every player's wormhole sightings, by player and
// end id.
type wormholeHistory []map[int]wormholeRecord

// record adds the ends each view saw this year, with their position and
// stability now. year is the year the views describe.
func (h wormholeHistory) record(year int, views []engine.PlayerView, sp *objects.Space) wormholeHistory {
	for len(h) < len(views) {
		h = append(h, map[int]wormholeRecord{})
	}
	if sp == nil {
		return h
	}
	for v, view := range views {
		for _, id := range view.Objects.Wormholes {
			wi, ei := id/2, id%2
			if id < 0 || wi >= len(sp.Wormholes) {
				continue
			}
			e := sp.Wormholes[wi].Ends[ei]
			h[v][id] = wormholeRecord{End: id, Year: year, Pos: e.Pos, Stability: objects.JumpChance(e)}
		}
	}
	return h
}

// list is player's sightings in end order.
func (h wormholeHistory) list(player int) []wormholeRecord {
	if player >= len(h) {
		return nil
	}
	ids := make([]int, 0, len(h[player]))
	for id := range h[player] {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]wormholeRecord, len(ids))
	for i, id := range ids {
		out[i] = h[player][id]
	}
	return out
}

func (h wormholeHistory) clone() wormholeHistory {
	out := make(wormholeHistory, len(h))
	for p, m := range h {
		out[p] = make(map[int]wormholeRecord, len(m))
		for id, r := range m {
			out[p][id] = r
		}
	}
	return out
}

// designRecord is what a player knows of another player's design, as
// last shown: the year, the hull and mass, and the whole design when the
// player was shown it in full. It is a snapshot: the owner may later put
// a new design in the same unused slot, under the same index
// (engine.DesignOrder), which the player has not seen.
type designRecord struct {
	Design int            `json:"design"`
	Year   int            `json:"year"`
	Hull   string         `json:"hull"`
	Mass   int            `json:"mass"`
	Full   *engine.Design `json:"full,omitempty"`
}

// designHistory is every player's knowledge of other players' designs,
// by player and design index.
type designHistory []map[int]designRecord

// record adds the designs each view showed this year. designs is the
// game's design list as the views show it. A full sighting snapshots the
// whole design. A partial sighting whose hull and mass match the full
// snapshot keeps it (the higher level, MEASURED SC-037 in computer
// players' files), which may be stale (ASSUMPTION G3): a player cannot
// tell from hull and mass that the parts changed. Any other partial
// sighting replaces the record (MEASURED SC-037 for a partial record;
// ASSUMPTION G3 for a full one). Records are never dropped. year is the
// year the views describe.
func (h designHistory) record(year int, views []engine.PlayerView, designs []engine.Design) designHistory {
	for len(h) < len(views) {
		h = append(h, map[int]designRecord{})
	}
	for v, view := range views {
		for _, d := range view.Designs {
			if d.Design < 0 || d.Design >= len(designs) {
				continue
			}
			r := designRecord{Design: d.Design, Year: year, Hull: d.Hull, Mass: d.Mass}
			old := h[v][d.Design]
			switch {
			case d.Full:
				full := deepCopy(designs[d.Design])
				r.Full = &full
			case old.Full != nil && old.Hull == d.Hull && old.Mass == d.Mass:
				r.Full = old.Full
			}
			h[v][d.Design] = r
		}
	}
	return h
}

// list is player's known designs in design index order.
func (h designHistory) list(player int) []designRecord {
	if player >= len(h) {
		return nil
	}
	ids := make([]int, 0, len(h[player]))
	for id := range h[player] {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]designRecord, len(ids))
	for i, id := range ids {
		out[i] = h[player][id]
	}
	return out
}

func (h designHistory) clone() designHistory {
	out := make(designHistory, len(h))
	for p, m := range h {
		out[p] = make(map[int]designRecord, len(m))
		for id, r := range m {
			out[p][id] = r
		}
	}
	return out
}

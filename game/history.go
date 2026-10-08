package game

import (
	"sort"

	"github.com/bfaber-centaur/elegy/engine"
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

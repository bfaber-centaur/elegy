package ai

import (
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// NewView builds a planner's View from a player's game.Report.
//
// The report does not carry three things a planner needs yet; the caller
// supplies them:
//   - universe, every planet's id and position (the map every player has);
//   - history, the player's latest report of each planet from earlier
//     years (AI.md §1 "What it sees"); this year's reports replace older
//     ones;
//   - created, the creation year and picture of designs the planner stored
//     in earlier years, by slot. A design missing from it counts
//     as created in the first year with picture 0: true of the starting
//     designs (UNIVERSE.md), and a placeholder for others.
//
// Wormholes and other players' PRT are left empty: a Rototill without
// them never takes a wormhole and treats every owner as not Alternate
// Reality. docs/AI-STATUS.md tracks these inputs.
func NewView(r game.Report, lvl Level, universe []PlanetPos, history map[int]engine.PlanetReport, created map[SlotKey]NewDesign) *View {
	v := &View{
		Year: r.Year, Player: r.Player, Level: lvl,
		Self: r.Self, Planets: r.Planets, Fleets: r.Fleets,
		Universe: universe,
		Known:    map[int]engine.PlanetReport{},
		Seen:     map[int]bool{},
		Others:   r.View.Fleets,
		PRT:      map[int]engine.PRT{},
	}
	for id, rep := range history {
		v.Known[id] = rep
	}
	for _, rep := range r.View.Planets {
		v.Known[rep.Planet] = rep
		v.Seen[rep.Planet] = true
	}
	for _, d := range r.Designs {
		od := Design{Slot: d.Slot.Slot, Index: d.Index, Design: d.Design, Created: FirstYear}
		if c, ok := created[SlotKey{d.Slot.Starbase, d.Slot.Slot}]; ok {
			od.Created, od.Picture = c.Created, c.Picture
		}
		if d.Slot.Starbase {
			v.Starbases = append(v.Starbases, od)
		} else {
			v.Ships = append(v.Ships, od)
		}
	}
	return v
}

// SlotKey names one of a player's design slots.
type SlotKey struct {
	Starbase bool
	Slot     int
}

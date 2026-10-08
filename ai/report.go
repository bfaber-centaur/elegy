package ai

import (
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// NewView builds a planner's View from a player's game.Report.
//
// The caller supplies two things the report holds in another shape:
//   - universe, every planet's id and position (the map every player has);
//   - history, the player's latest report of each planet from earlier
//     years (AI.md §1 "What it sees"); this year's reports replace older
//     ones.
//
// Each design's creation year and picture come from the report's design
// slots (engine.DesignSlot Created, Picture), which the game stores and
// saves, so a planner keeps nothing between years.
//
// Wormholes, other players' PRT and their designs are left empty here;
// ViewOf adds the report's wormholes and fully known designs. docs/AI-STATUS.md tracks these inputs.
func NewView(r game.Report, lvl Level, universe []PlanetPos, history map[int]engine.PlanetReport) *View {
	v := &View{
		Year: r.Year, Player: r.Player, Level: lvl, Rules: r.Rules,
		Self: r.Self, Planets: r.Planets, Fleets: r.Fleets,
		Universe:    universe,
		Known:       map[int]engine.PlanetReport{},
		Seen:        map[int]bool{},
		Others:      r.View.Fleets,
		PRT:         map[int]engine.PRT{},
		Foreign:     map[int]engine.Design{},
		TraderItems: r.TraderItems,
	}
	for id, rep := range history {
		v.Known[id] = rep
	}
	for _, rep := range r.View.Planets {
		v.Known[rep.Planet] = rep
		v.Seen[rep.Planet] = true
	}
	for _, d := range r.Designs {
		od := Design{Slot: d.Slot.Slot, Index: d.Index, Design: d.Design, Created: d.Slot.Created, Picture: d.Slot.Picture, Built: d.Slot.Built}
		if d.Slot.Starbase {
			v.Starbases = append(v.Starbases, od)
		} else {
			v.Ships = append(v.Ships, od)
		}
	}
	return v
}

package ai

import (
	"errors"
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// Driver is a game.Driver for one computer player: each year it builds
// the player's View from its game.Report alone and plays the
// personality's turn on the game's random stream (Report.Rand), as the
// original runs each computer player before the year in player order
// (AI.md §1 "Random numbers").
//
// Its only state across years is the creation year and picture of the
// designs it stored (AI.md §5, §10), which the engine's designs do not
// carry. ELEGY CHOICE (game.Driver): that state is the driver's and does
// not survive a save and load; after a load, the designs count as created
// in the first year with picture 0, as NewView says, until the engine
// carries the two values. The planners keep no other memory (AI.md §1,
// CONFIRMED AI-10).
type Driver struct {
	Personality Personality
	Level       Level

	created map[SlotKey]NewDesign
	// Unsupported holds the steps of the last turn Elegy could not order
	// (Result.Unsupported), for diagnostics.
	Unsupported []string
}

// NewDriver returns a driver for a computer player.
func NewDriver(p Personality, lvl Level) *Driver {
	return &Driver{Personality: p, Level: lvl, created: map[SlotKey]NewDesign{}}
}

// ErrNoRand is returned when the report carries no random stream.
var ErrNoRand = errors.New("ai: the report carries no random stream")

// Orders plays one year.
func (d *Driver) Orders(r game.Report) ([]engine.Order, error) {
	if r.Rand == nil {
		return nil, ErrNoRand
	}
	if d.created == nil {
		d.created = map[SlotKey]NewDesign{}
	}
	v := ViewOf(r, d.Level, d.created)
	var res Result
	switch d.Personality {
	case Robotoid:
		res = PlayRobotoid(v, r.Rand)
	case Rototill:
		res = PlayRototill(v, r.Rand)
	case Cybertron:
		res = PlayCybertron(v, r.Rand)
	default:
		return nil, fmt.Errorf("ai: personality %v is not implemented", d.Personality)
	}
	for _, nd := range res.Designs {
		d.created[SlotKey{nd.Starbase, nd.Slot}] = nd
	}
	d.Unsupported = res.Unsupported
	return res.Orders, nil
}

// ViewOf builds a planner's View from everything a game.Report carries:
// the universe map, the planet history and the known wormhole ends.
//
// ASSUMPTION A45: a known wormhole end's reported stability (0 Rock Solid
// .. 6 Extremely Volatile) stands for its movement class in the wormhole
// preference of AI.md §11, and counts as known.
//
// Other players' PRT is not in the report, so every other player counts
// as not Alternate Reality (the game withholds it).
func ViewOf(r game.Report, lvl Level, created map[SlotKey]NewDesign) *View {
	var universe []PlanetPos
	for _, p := range r.Universe {
		universe = append(universe, PlanetPos{ID: p.ID, Pos: p.Pos})
	}
	history := map[int]engine.PlanetReport{}
	for _, h := range r.History {
		history[h.Report.Planet] = h.Report
	}
	v := NewView(r, lvl, universe, history, created)
	for _, w := range r.Wormholes {
		v.Wormholes = append(v.Wormholes, Wormhole{End: w.End, Pos: w.Pos, Known: true, Class: w.Stability})
	}
	return v
}

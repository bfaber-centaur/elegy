package game

import (
	"encoding/json"
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// Report is what one player knows at the start of a year: the input a
// Driver decides that player's orders from, and the player-specific
// report a front end shows.
//
// It has two parts. The player's own objects (Self, Planets, Fleets,
// Designs) are copied in full from the game: a player knows everything
// about what it owns. Everything else is the engine's PlayerView for the
// year just generated (stars-elegy SCANNING.md), which holds only what
// the player's scanners, battles and messages revealed.
//
// Every slice is the Report's own copy; a driver may change it freely.
type Report struct {
	GameID uint64
	// Year is the year the orders are for: the game's current year.
	Year   int
	Player int

	// Self is the player's own record: race, research, battle plans,
	// relations and default queue.
	Self engine.Player
	// Planets are the player's own planets, in game order, with
	// everything about them.
	Planets []engine.Planet
	// Fleets are the player's own fleets, in game order.
	Fleets []engine.Fleet
	// Designs are the player's own designs, in design-slot order.
	Designs []OwnDesign

	// View is the player's knowledge of the rest of the universe at the
	// end of the previous year (engine.PlayerView): other players'
	// planets, fleets and designs as seen, scores the player may see, and
	// space objects seen.
	View engine.PlayerView

	// Events are the previous year's messages to this player (the events
	// whose Player is this player).
	Events []engine.Event
	// Orders are the outcomes of this player's order file for the
	// previous year: one entry per order, plus an entry with Index −1
	// when the whole file was refused.
	Orders []OrderOutcome
}

// OwnDesign is one of the player's designs: its slot and the design.
// Index is the design's index in the game's design list, which fleets'
// stacks and planets' starbases refer to.
type OwnDesign struct {
	Index  int
	Slot   engine.DesignSlot
	Design engine.Design
}

// OrderOutcome is the outcome of one of the player's orders. Message is
// empty when the order applied; otherwise it is the reason it was
// rejected. Err is the engine's error value, for errors.Is; it is not
// saved.
type OrderOutcome struct {
	Index   int
	Message string `json:",omitempty"`
	Err     error  `json:"-"`
}

// NewReport builds player's report from the game at the start of a year
// and what the previous year produced for every player: the views, the
// events and the order results (engine.TurnResult's Views, Events and
// Orders). For a new game, views are the starting knowledge and events
// and results are empty.
func NewReport(g engine.Game, player int, views []engine.PlayerView, events []engine.Event, results []engine.OrderResult) (Report, error) {
	if player < 0 || player >= len(g.Players) {
		return Report{}, fmt.Errorf("game: no player %d in a %d-player game", player, len(g.Players))
	}
	if player >= len(views) || views[player].Player != player {
		return Report{}, fmt.Errorf("game: no view for player %d", player)
	}
	r := Report{
		GameID: g.ID,
		Year:   g.Year,
		Player: player,
		Self:   deepCopy(g.Players[player]),
		View:   deepCopy(views[player]),
	}
	for _, p := range g.Planets {
		if p.Owner == player {
			r.Planets = append(r.Planets, deepCopy(p))
		}
	}
	for _, f := range g.Fleets {
		if f.Owner == player {
			r.Fleets = append(r.Fleets, deepCopy(f))
		}
	}
	for _, s := range g.DesignSlots {
		if s.Owner == player && s.Design >= 0 && s.Design < len(g.Designs) {
			r.Designs = append(r.Designs, OwnDesign{Index: s.Design, Slot: s, Design: deepCopy(g.Designs[s.Design])})
		}
	}
	for _, e := range events {
		if e.Player == player {
			r.Events = append(r.Events, deepCopy(e))
		}
	}
	for _, o := range results {
		if o.Player == player {
			out := OrderOutcome{Index: o.Index, Err: o.Err}
			if o.Err != nil {
				out.Message = o.Err.Error()
			}
			r.Orders = append(r.Orders, out)
		}
	}
	return r, nil
}

// deepCopy copies a value of the engine's plain data types through JSON,
// so the copy shares no slice, map or pointer with the original. The
// engine's state types hold only exported plain data (TestDeepCopy
// checks the types reports carry).
func deepCopy[T any](v T) T {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("game: copying %T: %v", v, err))
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		panic(fmt.Sprintf("game: copying %T: %v", v, err))
	}
	return out
}

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
	// Rules is the game's ruleset (engine.Game.Rules). A game's rules are
	// known to every player.
	Rules engine.Ruleset
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

	// History is the player's planet history: for every planet ever
	// reported to the player, the latest report and the year it
	// describes, by planet id. It includes the player's own colonies it
	// has since lost, still recorded as its own (stars-elegy AI.md §1
	// "What it sees", CONFIRMED AI-12, uses that record). See
	// PlanetRecord.
	History []PlanetRecord

	// Universe is every planet of the galaxy, seen or not: id, position
	// and name index. Every player knows where every planet is: a
	// computer player flies colonizers to planets it has never scanned
	// (stars-elegy AI.md §1 "What it sees", CONFIRMED AI-12) and scouts
	// to the nearest planet it has never seen (ai/rototill.md §3, MEASURED
	// AI-17). Nothing else about an unseen planet is known.
	Universe []UniversePlanet
	// Wormholes are the wormhole ends the player knows (OBJECTS.md
	// "Wormholes"), in end order.
	Wormholes []KnownWormholeEnd
	// KnownDesigns are the other players' designs the player knows, in
	// design index order; see KnownDesign.
	KnownDesigns []KnownDesign

	// Rand draws from the game's random stream. The loop sets it; a
	// driver that needs random numbers must draw them from it, and only
	// during its Orders call (AI.md §1 "Random numbers": computer players
	// draw from the host's single generator before the year is generated,
	// in player order). Nil outside the loop.
	Rand engine.Rand `json:"-"`
}

// UniversePlanet is a planet as every player knows it. NameIndex is the
// planet's index into the original's 999-name list (newgame.PlanetName
// gives Elegy's placeholder text).
type UniversePlanet struct {
	ID        int
	Pos       engine.Point
	NameIndex int
}

// KnownWormholeEnd is a wormhole end a player knows, as the player last
// saw it. End is the end's object id (2 × wormhole + end,
// objects.WormholeEndID). Year is the year of that sighting (the game
// year whose start it shows), Pos where the end was then and Stability
// the jump chance its report named then, 0 Rock Solid .. 6 Extremely
// Volatile (OBJECTS.md "Stability", BINARY-ONLY); the end's own class is
// not shown. Ends jiggle and their stability changes with age (OBJECTS.md
// "Jiggle"), and a player learns where an end is only by seeing it
// (SCANNING.md: a known end beyond normal range is not seen), so an end
// not seen this year keeps its last-seen values. Destination is where the
// end leads, set only when the player knows it and sees the other end
// this year (OBJECTS.md "Destination knowledge",
// objects.Space.Destination).
//
// ASSUMPTION G2: the report lists an end while the player still knows it
// (a jump makes everyone forget it, OBJECTS.md "Wormholes"), with its
// last-seen position and stability. stars-elegy gives no rule for how the
// original displays a known end it did not see this year.
type KnownWormholeEnd struct {
	End         int
	Year        int
	Pos         engine.Point
	Stability   int
	Destination *engine.Point `json:",omitempty"`
}

// KnownDesign is another player's design as a player knows it. Index is
// the design's index in the game's design list, which View.Fleets'
// stacks and planet reports' StarbaseDesign refer to. Year is the last
// year a view showed it (the game year whose start it shows).
//
// Hull and Mass are always known: a seen ship or starbase reveals its
// design's hull and mass (stars-elegy SCANNING.md "Designs", CONFIRMED).
// Full is set, and Design holds the whole design with its parts, once the
// player has been shown the design in full: a War Monger viewer sees every
// design in full, and everyone in a battle learns the other participants'
// designs (SCANNING.md "Designs", CONFIRMED SC-015, SC-031, SC-036;
// engine.DesignSighting.Full). Nothing else about a design seen only
// partially is reported.
//
// The record is what the player was shown: an owner may later put a new
// design in the same unused slot, under the same index
// (engine.DesignOrder), and the report keeps the old one until a view
// shows the new one.
//
// ASSUMPTION G3: a design stays known, and stays known in full once shown
// in full, in later years when no view shows it. A partial sighting whose
// hull and mass match keeps the full design last shown, which may be
// stale; a partial sighting with another hull or mass replaces it.
// stars-elegy says what a player's file holds the year a design is
// revealed, not whether later files keep it.
type KnownDesign struct {
	Index  int
	Year   int
	Hull   string
	Mass   int
	Full   bool
	Design *engine.Design `json:",omitempty"`
}

// PlanetRecord is one planet of a player's planet history: the latest
// report the player received about it and the year that report
// describes (the game year whose start it shows).
//
// ASSUMPTION G1: a newer report replaces the whole record, whatever its
// level. stars-elegy does not say how the original merges a lower-level
// report into its history record.
type PlanetRecord struct {
	Year   int
	Report engine.PlanetReport
}

// OwnDesign is one of the player's designs: its slot and the design.
// Index is the design's index in the game's design list, which fleets'
// stacks and planets' starbases refer to. Slot carries the design's
// creation year and picture (engine.DesignSlot Created, Picture), which
// a computer player's design ageing and starbase family rules read
// (AI.md "Storing a design", "Picture").
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
//
// NewReport leaves History, Universe, Wormholes, KnownDesigns and Rand
// empty; the
// loop's Game.Report fills them.
func NewReport(g engine.Game, player int, views []engine.PlayerView, events []engine.Event, results []engine.OrderResult) (Report, error) {
	if player < 0 || player >= len(g.Players) {
		return Report{}, fmt.Errorf("game: no player %d in a %d-player game", player, len(g.Players))
	}
	if player >= len(views) || views[player].Player != player {
		return Report{}, fmt.Errorf("game: no view for player %d", player)
	}
	r := Report{
		GameID: g.ID,
		Rules:  g.Rules,
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

package races

import "github.com/bfaber-centaur/elegy/engine"

// GameRaces is the race designs of a running game, one per player, with
// the settings the engine's Race does not hold (the tampered flag among
// them). It is the engine's RaceChecker.
type GameRaces struct {
	Designs  []Design
	Computer []bool // by player index
}

// CheckRaces runs YearlyCheck on every player (RACES.md "In a running
// game", CONFIRMED RD-P1..RD-P21) and writes the result back to the
// game: the engine's race and research budget take the clamped and
// penalized values. The player's own Race in the game is checked, so a
// change made elsewhere is seen; the design keeps the rest.
//
// A punished player gets EventRacePenalized (MESSAGES.md 0x117) and every
// other player EventRaceHacked naming it (0x182, CONFIRMED RD-P12: all
// four other human players in a six-player game). A computer player is
// never punished (RD-P20), so neither is sent for one.
//
// ASSUMPTION R1: a dead player's race is not checked, and a dead player
// is not told of another's penalty. RACES.md does not say.
//
// ASSUMPTION R2: 0x182 also goes to computer players; whether the
// original sends it to them is not observable (RD-P12).
func (r *GameRaces) CheckRaces(g *engine.Game) []engine.Event {
	var events []engine.Event
	for i := range g.Players {
		p := &g.Players[i]
		if p.Dead || i >= len(r.Designs) {
			continue
		}
		computer := i < len(r.Computer) && r.Computer[i]
		d := &r.Designs[i]
		d.Race = p.Race
		res := YearlyCheck(d, &p.ResearchBudget, computer)
		p.Race = d.Race
		if !res.Punished {
			continue
		}
		events = append(events, engine.Event{Kind: engine.EventRacePenalized, Player: i, Planet: -1, Fleet: -1})
		for j := range g.Players {
			if j != i && !g.Players[j].Dead {
				events = append(events, engine.Event{Kind: engine.EventRaceHacked, Player: j, Planet: -1, Fleet: -1, Count: i})
			}
		}
	}
	return events
}

var _ engine.RaceChecker = (*GameRaces)(nil)

// CloneRaces is a copy of r that shares nothing with it, for
// GenerateTurn's copy of the game (engine.RaceCloner): the check writes
// the designs (the tampered flag among them), so the input game's
// checker must not see a turn's changes.
func (r *GameRaces) CloneRaces() engine.RaceChecker {
	return &GameRaces{
		Designs:  append([]Design(nil), r.Designs...),
		Computer: append([]bool(nil), r.Computer...),
	}
}

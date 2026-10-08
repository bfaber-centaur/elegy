package engine

// Race check messages (RACES.md "In a running game", CONFIRMED RD-P12;
// MESSAGES.md 0x117 and 0x182). Elegy's wording.
const (
	EventRacePenalized EventKind = iota + EventPacketNoDriver + 1 // Player: the player's race was found illegal and adjusted
	EventRaceHacked                                               // Player = told, Count = the penalized player
)

// RaceChecker runs the yearly race check, KERNEL.md "Turn order" step 2a
// (RACES.md "In a running game"): after the pre-movement tasks and before
// movement, so a degraded race already applies to the year's production
// (MEASURED, SL-12, OT-6). It may change each player's Race and
// ResearchBudget and returns the step's events. The check scores races,
// which the races package does; it implements this interface so the
// engine needs no race scoring of its own.
type RaceChecker interface {
	CheckRaces(g *Game) []Event
}

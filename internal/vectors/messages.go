package vectors

import (
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// Message comparison (vectors/README.md "message": player `player` got
// message `message_id`, `present`). Elegy's events carry no message id, so
// pvMessages maps a year's events to the ids stars-elegy docs/MESSAGES.md
// gives for them. Only the ids pvMessageIDs lists are compared; any other
// id is skipped, so an id joins the comparison only with its mapping.

// pvMessageIDs maps each compared message id to what in Elegy sends it.
// An entry that starts with "none" is an id Elegy sends no event for: a
// present expectation of it is a mismatch, a missing mechanic.
func pvMessageIDs() map[int]string {
	ids := map[int]string{
		0x004: "EventDropRepelled, to the planet's owner",
		0x026: "EventOvercrowded",
		0x02e: "EventColonistsBeamedDown",
		0x035: "EventBuilt of factories, one in all (merge rule)",
		0x036: "EventBuilt of factories, more than one (merge rule)",
		0x037: "EventBuilt of mines, one in all (merge rule)",
		0x038: "EventBuilt of mines, more than one (merge rule)",
		0x03e: "EventQueueCompleted",
		0x03f: "EventNoQueue",
		0x04e: "EventFleetArrived",
		0x050: "EventLevelUp, a player without Generalized Research",
		0x078: "EventPartAvailable of a ship hull",
		0x145: "EventPartAvailable of a planetary defense",
		0x157: "EventPartAvailable of a planetary scanner",
		0x136: "EventLevelUp, a player with Generalized Research",
		0x159: "EventResearchStolen",
		0x05b: "EventFleetScrapped in deep space",
		0x05f: "EventPartAvailable of a component, not a ship or starbase hull, planetary scanner or defense",
		0x08c: "EventAlchemy",
		0x12a: "EventOrderClipped of an installation",
		0x12f: "EventOrderClipped of terraforming",
		0x083: "EventCometStrike, small",
		0x084: "EventCometStrike, medium",
		0x085: "EventCometStrike, large",
		0x086: "EventCometStrike, huge",
		0x08f: "EventBombed then EventPlanetEmptied, to the bomber",
		0x090: "EventBombed then EventPlanetEmptied, to the old owner",
	}
	for k, id := range pvSummaryIDs {
		if k != int(engine.SummaryNone) {
			ids[id] = "EventBattleSummary " + fmt.Sprint(engine.BattleSummary(k))
		}
	}
	return ids
}

// pvPlanetaryIDs is the level-up message id of each planetary scanner
// (0x157) and planetary defense (0x145), by name.
func pvPlanetaryIDs() map[string]int {
	ids := map[string]int{}
	cat := engine.Components()
	for _, s := range cat.PlanetScanners() {
		ids[s.Name] = 0x157
	}
	for _, d := range cat.Defenses() {
		ids[d.Name] = 0x145
	}
	return ids
}

// pvSummaryIDs is the MESSAGES.md id of each engine.BattleSummary, by its
// value.
var pvSummaryIDs = func() []int {
	ids := []int{0}
	for id := 0x091; id <= 0x0a8; id++ {
		if id == 0x0a4 {
			// SummaryMulti*: 0x0a4–0x0a8, then SummaryMultiMixed.
			ids = append(ids, 0x0a4, 0x0a5, 0x0a6, 0x0a7, 0x0a8, 0x07e)
			break
		}
		ids = append(ids, id)
	}
	return append(ids, 0x113, 0x114, 0x115, 0x116)
}()

// pvMessages is the set of (player, message id) that a year's events send,
// for the ids pvMessageIDs maps. g is the game at the end of the year.
func pvMessages(g *engine.Game, events []engine.Event) map[[2]int]bool {
	got := map[[2]int]bool{}
	for player, ids := range pvMessageSeq(g, events) {
		for _, id := range ids {
			got[[2]int{player, id}] = true
		}
	}
	return got
}

// pvMessageSeq is each player's messages of the year, by the ids
// pvMessageIDs maps, in the order the original's message file would hold
// them: MESSAGES.md "Order", messages are appended as the year generates
// them, and a step that deletes an earlier message (a fleet's earlier
// 0x04e, a planet's single-unit build count) removes it from its place.
// Each id is listed once, at its first place, as the vectors list them
// (stars-elegy tools/vectors/build_kx.py `messages`). Elegy's events are
// read as being in the year's order (ASSUMPTION: engine.GenerateTurn
// appends them as the turn runs). g is the game at the end of the year.
func pvMessageSeq(g *engine.Game, events []engine.Event) map[int][]int {
	type msg struct {
		player, id int
		gone       bool
	}
	var msgs []msg
	send := func(player, id int) int {
		if player < 0 {
			return -1
		}
		msgs = append(msgs, msg{player: player, id: id})
		return len(msgs) - 1
	}
	owner := func(planet int) int {
		for i := range g.Planets {
			if g.Planets[i].ID == planet {
				return g.Planets[i].Owner
			}
		}
		return engine.NoOwner
	}
	// Build counts (MESSAGES.md "Merging of build counts"): a planet's
	// earlier single-unit message of the year is removed and added into
	// its next count of the same installation; plural ones are not merged.
	type built struct {
		planet int
		item   engine.ItemKind
	}
	single := map[built]int{} // pending single-unit message: its index in msgs
	arrived := map[int]int{}  // a fleet's 0x04e: its index in msgs
	for i, ev := range events {
		switch ev.Kind {
		case engine.EventDropRepelled:
			// The defender's copy (0x004); Player is the attacker. The
			// defender is read as the planet's owner at the end of the
			// year (ASSUMPTION: the event does not name it), so a planet
			// captured later in the same year credits its new owner.
			send(owner(ev.Planet), 0x004)
		case engine.EventBuilt:
			one, more := 0, 0
			switch ev.Item {
			case engine.ItemFactory:
				one, more = 0x035, 0x036
			case engine.ItemMine:
				one, more = 0x037, 0x038
			default:
				continue
			}
			k := built{ev.Planet, ev.Item}
			n := ev.Count
			if j, ok := single[k]; ok {
				n++
				msgs[j].gone = true
				delete(single, k)
			}
			switch {
			case n > 1:
				send(ev.Player, more)
			case n == 1:
				if j := send(ev.Player, one); j >= 0 {
					single[k] = j
				}
			}
		case engine.EventBattleSummary:
			if ev.Count > 0 && ev.Count < len(pvSummaryIDs) {
				send(ev.Player, pvSummaryIDs[ev.Count])
			}
		case engine.EventFleetArrived:
			// MESSAGES.md 0x04e: an earlier copy for the same fleet is
			// removed first.
			if j, ok := arrived[ev.Fleet]; ok {
				msgs[j].gone = true
			}
			if j := send(ev.Player, 0x04e); j >= 0 {
				arrived[ev.Fleet] = j
			}
		case engine.EventOvercrowded:
			send(ev.Player, 0x026)
		case engine.EventQueueCompleted:
			send(ev.Player, 0x03e)
		case engine.EventNoQueue:
			send(ev.Player, 0x03f)
		case engine.EventAlchemy:
			send(ev.Player, 0x08c)
		case engine.EventOrderClipped:
			if ev.Item == engine.ItemTerraform {
				send(ev.Player, 0x12f)
			} else {
				send(ev.Player, 0x12a)
			}
		case engine.EventLevelUp:
			if ev.Player >= 0 && ev.Player < len(g.Players) {
				if g.Players[ev.Player].Race.LRT.GeneralizedResearch {
					send(ev.Player, 0x136)
				} else {
					send(ev.Player, 0x050)
				}
			}
		case engine.EventPartAvailable:
			cs := engine.Components().Components
			if ev.Count >= 0 && ev.Count < len(cs) {
				c := cs[ev.Count]
				switch {
				case c.Category == engine.CatHull:
					send(ev.Player, 0x078)
				case c.Category == engine.CatStarbaseHull:
					// 0x0d0, BINARY-ONLY: not mapped.
				case c.Category == engine.CatPlanetary && pvPlanetaryIDs()[c.Name] != 0:
					send(ev.Player, pvPlanetaryIDs()[c.Name])
				default:
					// Every other part, the Genesis Device included
					// (MESSAGES.md at stars-elegy f2778b3).
					send(ev.Player, 0x05f)
				}
			}
		case engine.EventResearchStolen:
			send(ev.Player, 0x159)
		case engine.EventColonistsBeamedDown:
			send(ev.Player, 0x02e)
		case engine.EventFleetScrapped:
			if ev.Planet == -1 {
				send(ev.Player, 0x05b)
			}
		case engine.EventCometStrike:
			if ev.Count >= 0 && ev.Count < 4 {
				send(ev.Player, 0x083+ev.Count)
			}
		case engine.EventPlanetEmptied:
			if i > 0 && events[i-1].Kind == engine.EventBombed && events[i-1].Planet == ev.Planet {
				send(events[i-1].Player, 0x08f)
				send(ev.Player, 0x090)
			}
		}
	}
	seq := map[int][]int{}
	seen := map[[2]int]bool{}
	for _, m := range msgs {
		if k := [2]int{m.player, m.id}; !m.gone && !seen[k] {
			seen[k] = true
			seq[m.player] = append(seq[m.player], m.id)
		}
	}
	return seq
}

// pvMessagePrior is a message expectation listed before another of the
// same player and year in a case.
type pvMessagePrior struct {
	id     int
	sample bool
}

// pvMessageOrder records, on each present message expectation, the
// present ones listed before it for the same player and year in its case.
// The vectors list a player's message ids in the order of their first
// place in the year's message file (stars-elegy tools/vectors/build_kx.py
// `messages`, which the observed-state builders share; ASSUMPTION for a
// vector not built that way: vectors/README.md does not state the order).
func pvMessageOrder(cases []pvCase) {
	for c := range cases {
		prior := map[[2]int][]pvMessagePrior{}
		for i := range cases[c].Expect {
			e := &cases[c].Expect[i]
			if e.Kind != "message" || e.Player == nil || e.MessageID == nil || e.Present == nil || !*e.Present {
				continue
			}
			k := [2]int{e.Year, *e.Player}
			e.before = prior[k]
			prior[k] = append(prior[k][:len(prior[k]):len(prior[k])], pvMessagePrior{*e.MessageID, e.Sample})
		}
	}
}

// messageOrderCheck compares a present message's place in the player's
// year with the nearest message listed before it that Elegy also sends:
// that one must come first. A message that is not a sample is placed
// against the nearest one that is not a sample either, so an order that
// draws decide stays a sample.
func messageOrderCheck(e pvExpect, seq []int) string {
	at := map[int]int{}
	for i, id := range seq {
		at[id] = i
	}
	for i := len(e.before) - 1; i >= 0; i-- {
		p := e.before[i]
		j, ok := at[p.id]
		if !ok || p.id == *e.MessageID || (p.sample && !e.Sample) {
			continue
		}
		if j > at[*e.MessageID] {
			return fmt.Sprintf("message 0x%03x to player %d comes before 0x%03x, want after", *e.MessageID, *e.Player, p.id)
		}
		return ""
	}
	return ""
}

// messageCheck compares a message expectation with the year's events.
func (l *pvLoaded) messageCheck(g *engine.Game, e pvExpect) string {
	if e.Player == nil || e.MessageID == nil || e.Present == nil {
		return "message: malformed expectation (player, message_id and present are needed)"
	}
	id, player := *e.MessageID, *e.Player
	if _, ok := pvMessageIDs()[id]; !ok {
		return fmt.Sprintf("skip: message 0x%03x has no event mapping", id)
	}
	if l.planned(player) {
		return "skip: " + pvPlannedWhy
	}
	if l.queued[player] && id >= 0x035 && id <= 0x038 {
		return "skip: a build message from a production queue not loaded"
	}
	if l.queued[player] && (id == 0x03e || id == 0x03f) {
		// A planet whose queue the harness cannot load runs with none.
		return "skip: a queue message from a production queue not loaded"
	}
	if e.Year >= len(l.events) {
		return "skip: no events for the year"
	}
	seq := pvMessageSeq(g, l.events[e.Year])[player]
	got := false
	for _, s := range seq {
		got = got || s == id
	}
	if got != *e.Present {
		return fmt.Sprintf("message 0x%03x to player %d present = %v, want %v", id, player, got, *e.Present)
	}
	if got {
		return messageOrderCheck(e, seq)
	}
	return ""
}

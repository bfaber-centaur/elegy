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
	return map[int]string{
		0x004: "EventDropRepelled, to the planet's owner",
		0x026: "none: no event for a population loss to overcrowding",
		0x02e: "none: no event for colonists beamed down by an unload task",
		0x035: "EventBuilt of factories, one in all (merge rule)",
		0x036: "EventBuilt of factories, more than one (merge rule)",
		0x037: "EventBuilt of mines, one in all (merge rule)",
		0x038: "EventBuilt of mines, more than one (merge rule)",
		0x04e: "EventFleetArrived",
		0x050: "none: no event for a research level gained",
		0x05b: "EventFleetScrapped in deep space",
		0x05f: "none: no event for a component a level made available",
		0x083: "EventCometStrike, small",
		0x084: "EventCometStrike, medium",
		0x085: "EventCometStrike, large",
		0x086: "EventCometStrike, huge",
		0x08f: "EventBombed then EventPlanetEmptied, to the bomber",
		0x090: "EventBombed then EventPlanetEmptied, to the old owner",
		0x095: "none: no per-player battle summary",
		0x09f: "none: no per-player battle summary",
		0x0a1: "none: no per-player battle summary",
		0x116: "none: no per-player battle summary",
	}
}

// pvMessages is the set of (player, message id) that a year's events send,
// for the ids pvMessageIDs maps. g is the game at the end of the year.
func pvMessages(g *engine.Game, events []engine.Event) map[[2]int]bool {
	got := map[[2]int]bool{}
	send := func(player, id int) {
		if player >= 0 {
			got[[2]int{player, id}] = true
		}
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
	single := map[built]int{} // pending single-unit message: its player, or absent
	for i, ev := range events {
		switch ev.Kind {
		case engine.EventDropRepelled:
			// The defender's copy (0x004); Player is the attacker. The
			// defender is read as the planet's owner at the end of the
			// year (ASSUMPTION: the event does not name it), so a planet
			// captured later in the same year credits its new owner.
			send(owner(ev.Planet), 0x004)
		case engine.EventBuilt:
			more := 0
			switch ev.Item {
			case engine.ItemFactory:
				more = 0x036
			case engine.ItemMine:
				more = 0x038
			default:
				continue
			}
			k := built{ev.Planet, ev.Item}
			n := ev.Count
			if _, ok := single[k]; ok {
				n++
				delete(single, k)
			}
			switch {
			case n > 1:
				send(ev.Player, more)
			case n == 1:
				single[k] = ev.Player
			}
		case engine.EventFleetArrived:
			send(ev.Player, 0x04e)
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
	for k, player := range single {
		id := 0x035
		if k.item == engine.ItemMine {
			id = 0x037
		}
		send(player, id)
	}
	return got
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
	if e.Year >= len(l.events) {
		return "skip: no events for the year"
	}
	got := pvMessages(g, l.events[e.Year])[[2]int{player, id}]
	if got != *e.Present {
		return fmt.Sprintf("message 0x%03x to player %d present = %v, want %v", id, player, got, *e.Present)
	}
	return ""
}

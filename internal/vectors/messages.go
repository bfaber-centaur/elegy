package vectors

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
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
		0x007: "EventPlanetCaptured, to the former owner",
		0x00c: "EventPlanetCaptured, to the winner",
		0x025: "EventHostileShrink",
		0x026: "EventOvercrowded",
		0x028: "EventTargetGone",
		0x02e: "EventColonistsBeamedDown",
		0x035: "EventBuilt of factories, one in all (merge rule)",
		0x036: "EventBuilt of factories, more than one (merge rule)",
		0x037: "EventBuilt of mines, one in all (merge rule)",
		0x038: "EventBuilt of mines, more than one (merge rule)",
		0x039: "EventBuilt of defenses, one in all (merge rule)",
		0x03a: "EventBuilt of defenses, more than one (merge rule)",
		0x03e: "EventQueueCompleted",
		0x03f: "EventNoQueue",
		0x023: "EventPlanetEmptied by growth, died out (or left, after a shrink)",
		0x024: "EventPlanetEmptied by growth, died out, Alternate Reality",
		0x040: "EventPlanetEmptied by growth, no colonists left",
		0x041: "EventPlanetEmptied by growth, no colonists left, Alternate Reality",
		0x04e: "EventFleetArrived",
		0x050: "EventLevelUp, a player without Generalized Research",
		0x078: "EventPartAvailable of a ship hull",
		0x145: "EventPartAvailable of a planetary defense",
		0x157: "EventPartAvailable of a planetary scanner",
		0x136: "EventLevelUp, a player with Generalized Research",
		0x159: "EventResearchStolen",
		0x156: "EventClaimAdjusted",
		0x15c: "EventOriginalDrift",
		0x05b: "EventFleetScrapped in deep space",
		0x059: "EventFleetScrapped at a planet without a starbase whose owner lacks Ultimate Recycling",
		0x140: "EventFleetScrapped at a planet without a starbase whose owner lacks Ultimate Recycling, to the planet's owner",
		0x05a: "EventFleetScrapped at a starbase whose owner lacks Ultimate Recycling",
		0x141: "EventFleetScrapped at a starbase whose owner lacks Ultimate Recycling, to the planet's owner, when it gave no technology",
		0x05f: "EventPartAvailable of a component, not a ship or starbase hull, planetary scanner or defense",
		0x08c: "EventAlchemy",
		0x02f: "EventShipsBuilt, one ship, no route",
		0x030: "EventShipsBuilt, several ships, no route",
		0x031: "EventShipsBuilt then EventFleetRouted, one ship",
		0x032: "EventShipsBuilt then EventFleetRouted, several ships",
		0x033: "EventShipsBuilt then EventFleetNotRouted, one ship",
		0x034: "EventShipsBuilt then EventFleetNotRouted, several ships",
		0x0ba: "EventShipsLostFleetLimit",
		0x0cd: "EventStarbaseBuilt",
		0x07b: "EventTerraformed",
		0x129: "EventPacketNoDriver",
		0x139: "EventShipsJoinedFleet",
		0x12a: "EventOrderClipped of an installation",
		0x12f: "EventOrderClipped of terraforming",
		0x083: "EventCometStrike, small",
		0x084: "EventCometStrike, medium",
		0x085: "EventCometStrike, large",
		0x086: "EventCometStrike, huge",
		0x08f: "EventBombed then EventPlanetEmptied, to the bomber",
		0x0c2: "EventMinesSwept by a fleet",
		0x0f4: "EventMinesSwept by a starbase",
		0x0be: "EventMinefieldSwept",
		0x090: "EventBombed then EventPlanetEmptied, to the old owner",
		0x17c: "EventBombed with several fleets in orbit, then EventPlanetEmptied, to the bomber",
		0x17d: "EventBombed with several fleets in orbit, then EventPlanetEmptied, to the old owner",
		0x117: "EventRacePenalized",
		0x182: "EventRaceHacked",
	}
	for k, id := range pvSummaryIDs {
		if k != int(engine.SummaryNone) {
			ids[id] = "EventBattleSummary " + fmt.Sprint(engine.BattleSummary(k))
		}
	}
	return ids
}

// The player slots of the invasion messages (MESSAGES.md "Invasion
// slots"): the player number plus 0x30 in 0x004 and 0x007, plus 0x20 in
// 0x00c (MEASURED; the high bits are article flags, BINARY-ONLY).
const (
	pvInvasionPlayer = 0x30
	pvCapturedPlayer = 0x20
)

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
// pvMessageIDs maps, each id once, at its first place in pvMessageList's
// sequence, as the vectors list them (stars-elegy
// tools/vectors/build_kx.py `messages`). g is the game at the end of the
// year.
func pvMessageSeq(g *engine.Game, events []engine.Event) map[int][]int {
	seq := map[int][]int{}
	for player, msgs := range pvMessageList(g, events) {
		seen := map[int]bool{}
		for _, m := range msgs {
			if !seen[m.id] {
				seen[m.id] = true
				seq[player] = append(seq[player], m.id)
			}
		}
	}
	return seq
}

// pvMsg is one message as the original's message file would hold it
// (vectors/README.md "Message sequences"): its id, focus and slot values.
// focusKnown and known say which of them Elegy's event determines; why
// names the first slot or focus it does not ("" when all are known).
type pvMsg struct {
	id, focus  int
	focusKnown bool
	params     []int
	known      []bool
	why        string
}

// Focus codes (MESSAGES.md "How messages work", focus codes) as stored
// 16-bit values.
const (
	pvFocusNone     = 0x10000 - 1
	pvFocusFleet    = 0x8000
	pvFocusResearch = 0x10000 - 2
	pvFocusDesign   = 0x10000 - 3
	pvFocusObject   = 0x10000 - 6
)

// Cargo message slots (0x02b to 0x02e; MESSAGES.md "Slot types" CM-1
// update, stars-elegy 4f59d8d, MEASURED): the cargo type of colonists,
// and the first location slot of a planet (the second is the planet).
const (
	pvCargoColonists = 3
	pvLocationPlanet = 0xffff
)

// pvCategoryBits is each component category's bit number b in a part
// announcement's category mask 1<<b (MESSAGES.md "Slots and focus of the
// component announcements" at stars-elegy a348c5d).
func pvCategoryBits() map[string]int {
	return map[string]int{
		engine.CatEngine: 0, engine.CatScanner: 1, engine.CatShield: 2, engine.CatArmor: 3,
		engine.CatBeam: 4, engine.CatTorpedo: 5, engine.CatBomb: 6, engine.CatMiningRobot: 7,
		engine.CatMineLayer: 8, engine.CatOrbital: 9, engine.CatStarbaseHull: 10, engine.CatElectrical: 11,
		engine.CatMechanical: 12, engine.CatTerraform: 13, engine.CatHull: 14, engine.CatPlanetary: 15,
	}
}

// pvMessageList is each player's messages of the year, by the ids
// pvMessageIDs maps, in the order the original's message file would hold
// them: MESSAGES.md "Order", messages are appended as the year generates
// them, and a step that deletes an earlier message (a fleet's earlier
// 0x04e, a planet's single-unit build count) removes it from its place.
// Elegy's events are read as being in the year's order (ASSUMPTION:
// engine.GenerateTurn appends them as the turn runs). g is the game at
// the end of the year.
//
// The slots follow the Slots column of MESSAGES.md "Catalogue" in stored
// order, a colonist amount taking two slots (low and high word, "Slot
// types"). A planet slot or focus is the vector's planet number, Elegy's
// planet ID − 1 (loadVector). A fleet is `owner << 9 | number` (stars-elegy
// ORACLE.md, "Fleet parameters are object ids"), its focus with bit 15 set,
// and its number Elegy's Number − 1 (loadVector); a fleet gone by the end
// of the year leaves it unknown.
func pvMessageList(g *engine.Game, events []engine.Event) map[int][]pvMsg {
	type msg struct {
		player int
		pvMsg
		gone bool
	}
	var msgs []msg
	send := func(player int, m pvMsg) int {
		if player < 0 {
			return -1
		}
		if m.known == nil {
			m.known = make([]bool, len(m.params))
			for i := range m.known {
				m.known[i] = true
			}
		}
		msgs = append(msgs, msg{player: player, pvMsg: m})
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
	fleet := func(id int) (int, bool) {
		for i := range g.Fleets {
			if f := &g.Fleets[i]; f.ID == id {
				return f.Owner<<9 | (f.Number - 1), true
			}
		}
		return 0, false
	}
	// atPlanet is a message about one planet: its focus and only slot.
	atPlanet := func(id, planet int) pvMsg {
		return pvMsg{id: id, focus: planet - 1, focusKnown: true, params: []int{planet - 1}}
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
	launched := map[int]int{} // a new fleet's build message: its index in msgs
	for i, ev := range events {
		switch ev.Kind {
		case engine.EventDropRepelled:
			// The defender's copy (0x004); Player is the attacker. The
			// defender is read as the planet's owner at the end of the
			// year (ASSUMPTION: the event does not name it), so a planet
			// captured later in the same year credits its new owner.
			// Slots: planet, troops dropped (two slots), attacker + 0x30
			// (MESSAGES.md "Invasion slots", MEASURED).
			send(owner(ev.Planet), pvMsg{id: 0x004, focus: ev.Planet - 1, focusKnown: true,
				params: []int{ev.Planet - 1, ev.Count & 0xffff, ev.Count >> 16, pvInvasionPlayer + ev.Player}})
		case engine.EventPlanetCaptured:
			// 0x00c to the winner, then 0x007 to the former owner
			// (MESSAGES.md "Invasion slots", MEASURED). 0x00c: former
			// owner + 0x20, planet; 0x007: winner + 0x30, planet, troops
			// the winner dropped (two slots).
			old := ev.Axes[0]
			send(ev.Player, pvMsg{id: 0x00c, focus: ev.Planet - 1, focusKnown: true,
				params: []int{pvCapturedPlayer + old, ev.Planet - 1}})
			send(old, pvMsg{id: 0x007, focus: ev.Planet - 1, focusKnown: true,
				params: []int{pvInvasionPlayer + ev.Player, ev.Planet - 1, ev.Count & 0xffff, ev.Count >> 16}})
		case engine.EventBuilt:
			one, more := 0, 0
			switch ev.Item {
			case engine.ItemFactory:
				one, more = 0x035, 0x036
			case engine.ItemMine:
				one, more = 0x037, 0x038
			case engine.ItemDefenses:
				one, more = 0x039, 0x03a
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
				send(ev.Player, pvMsg{id: more, focus: ev.Planet - 1, focusKnown: true, params: []int{n, ev.Planet - 1}})
			case n == 1:
				if j := send(ev.Player, atPlanet(one, ev.Planet)); j >= 0 {
					single[k] = j
				}
			}
		case engine.EventShipsBuilt:
			// MESSAGES.md 0x02f/0x030: a new fleet of one or several
			// ships; slots planet, (count,) design, focus the new fleet
			// (CONFIRMED CB-047, CB round 6). The design is owner·32 +
			// the ship design slot (MEASURED, "Design slot"). A route leg
			// that follows turns it into 0x031–0x034.
			if len(ev.Axes) != 1 {
				continue
			}
			params := []int{ev.Planet - 1, ev.Player*32 + ev.Axes[0]}
			id := 0x02f
			if ev.Count > 1 {
				params = []int{ev.Planet - 1, ev.Count, ev.Player*32 + ev.Axes[0]}
				id = 0x030
			}
			m := pvMsg{id: id, params: params, why: fmt.Sprintf("0x%03x new fleet gone by the end of the year", id)}
			if code, ok := fleet(ev.Fleet); ok {
				m.focus, m.focusKnown, m.why = pvFocusFleet|code, true, ""
			}
			if j := send(ev.Player, m); j >= 0 {
				launched[ev.Fleet] = j
			}
		case engine.EventFleetRouted, engine.EventFleetNotRouted:
			// MESSAGES.md 0x031/0x032 (routed, CONFIRMED SL round 1) and
			// 0x033/0x034 (no warp with fuel, BINARY-ONLY): the build
			// message with the route target as its last slot.
			// Only the leg Launch gives it: a later route task for the
			// same fleet in the year (routeTask) is not a build message.
			j, ok := launched[ev.Fleet]
			if !ok {
				continue
			}
			delete(launched, ev.Fleet)
			m := &msgs[j].pvMsg
			m.id += 2
			if ev.Kind == engine.EventFleetNotRouted {
				m.id += 2
			}
			m.params = append(m.params, ev.Planet-1)
			m.known = append(m.known, true)
			if m.why != "" {
				m.why = fmt.Sprintf("0x%03x new fleet gone by the end of the year", m.id)
			}
		case engine.EventShipsJoinedFleet:
			// MESSAGES.md 0x139: at the 512-fleet limit the new ships
			// join an own fleet at the planet. Slots planet, count,
			// design, fleet; focus the fleet (BINARY-ONLY). The design
			// slot is not compared.
			m := pvMsg{id: 0x139, params: []int{ev.Planet - 1, ev.Count, 0, 0}, known: []bool{true, true, false, false},
				why: "0x139 design slot; fleet gone by the end of the year"}
			if code, ok := fleet(ev.Fleet); ok {
				m = pvMsg{id: 0x139, focus: pvFocusFleet | code, focusKnown: true, params: []int{ev.Planet - 1, ev.Count, 0, code},
					known: []bool{true, true, false, true}, why: "0x139 design slot"}
			}
			send(ev.Player, m)
		case engine.EventShipsLostFleetLimit:
			// MESSAGES.md 0x0ba: at the 512-fleet limit with no own fleet
			// at the planet able to take them, the ships are lost. Slots
			// planet, count, design (BINARY-ONLY); the design slot is not
			// compared.
			send(ev.Player, pvMsg{id: 0x0ba, focus: ev.Planet - 1, focusKnown: true, params: []int{ev.Planet - 1, ev.Count, 0},
				known: []bool{true, true, false}, why: "0x0ba design slot"})
		case engine.EventStarbaseBuilt:
			// MESSAGES.md 0x0cd: a starbase built gives 0x0cd whatever
			// its dock, MEASURED for Orbital Fort, Space Dock, Space
			// Station and Ultra Station (SL-12, PD-1, PK-1; the dock
			// rule's 0x0ce/0x0cf are never observed). Death Star is NOT
			// RUN; it gets 0x0cd too (inferred). Slots: planet, then
			// the design as owner·32 + 16 + the starbase design slot
			// (MEASURED, SL-12).
			if len(ev.Axes) == 1 {
				send(ev.Player, pvMsg{id: 0x0cd, focus: ev.Planet - 1, focusKnown: true,
					params: []int{ev.Planet - 1, ev.Player*32 + 16 + ev.Axes[0]}})
			}
		case engine.EventTerraformed:
			// MESSAGES.md 0x07b: planet, 1 up / 0 down, axis, hab value.
			// The hab value is axis<<8 | the axis's new value (MEASURED
			// from every 0x07b entry of the vectors; the table does not
			// spell it out).
			up := 0
			if ev.Count > 0 {
				up = 1
			}
			a := ev.Axes[0]
			send(ev.Player, pvMsg{id: 0x07b, focus: ev.Planet - 1, focusKnown: true,
				params: []int{ev.Planet - 1, up, a, a<<8 | ev.Axes[1]}})
		case engine.EventBattleSummary:
			if ev.Count > 0 && ev.Count < len(pvSummaryIDs) {
				send(ev.Player, pvMsg{id: pvSummaryIDs[ev.Count], known: []bool{}, why: "battle summary focus and slots"})
			}
		case engine.EventFleetArrived:
			// MESSAGES.md 0x04e: an earlier copy for the same fleet is
			// removed first.
			if j, ok := arrived[ev.Fleet]; ok {
				msgs[j].gone = true
			}
			m := pvMsg{id: 0x04e, params: []int{0}, known: []bool{false}, why: "0x04e fleet gone by the end of the year"}
			if code, ok := fleet(ev.Fleet); ok {
				m = pvMsg{id: 0x04e, focus: pvFocusFleet | code, focusKnown: true, params: []int{code}}
			}
			if j := send(ev.Player, m); j >= 0 {
				arrived[ev.Fleet] = j
			}
		case engine.EventRacePenalized:
			// MESSAGES.md 0x117: no slots, focus none (stars-elegy
			// 28a3175, CONFIRMED RD-P12, KX-003).
			send(ev.Player, pvMsg{id: 0x117, focus: pvFocusNone, focusKnown: true, params: []int{}})
		case engine.EventRaceHacked:
			// MESSAGES.md 0x182: player (the penalized one), focus none.
			// The slot's value is in one KX-003 record only and the
			// article/plural flags "Slot types" allows are not measured
			// for it, so it is not compared.
			send(ev.Player, pvMsg{id: 0x182, focus: pvFocusNone, focusKnown: true,
				params: []int{ev.Count}, known: []bool{false}, why: "0x182 player slot"})
		case engine.EventOvercrowded:
			send(ev.Player, pvMsg{id: 0x026, focus: ev.Planet - 1, focusKnown: true, params: []int{ev.Planet - 1, ev.Count & 0xffff, ev.Count >> 16}})
		case engine.EventHostileShrink:
			// MESSAGES.md 0x025: planet, colonists before, colonists
			// after, each a 32-bit count in two 16-bit slots as in 0x026.
			old, now := ev.Count, ev.Axes[0]
			send(ev.Player, pvMsg{id: 0x025, focus: ev.Planet - 1, focusKnown: true, params: []int{ev.Planet - 1, old & 0xffff, old >> 16, now & 0xffff, now >> 16}})
		case engine.EventClaimAdjusted:
			// MESSAGES.md 0x156: planet, percent.
			send(ev.Player, pvMsg{id: 0x156, focus: ev.Planet - 1, focusKnown: true, params: []int{ev.Planet - 1, ev.Count}})
		case engine.EventOriginalDrift:
			// MESSAGES.md 0x15c: planet, hab (the axis).
			send(ev.Player, pvMsg{id: 0x15c, focus: ev.Planet - 1, focusKnown: true, params: []int{ev.Planet - 1, ev.Axes[0]}})
		case engine.EventQueueCompleted:
			send(ev.Player, atPlanet(0x03e, ev.Planet))
		case engine.EventNoQueue:
			send(ev.Player, atPlanet(0x03f, ev.Planet))
		case engine.EventPacketNoDriver:
			// MESSAGES.md 0x129, one per packet item removed (CONFIRMED
			// for no driver, GAP-001 A3): planet.
			send(ev.Player, atPlanet(0x129, ev.Planet))
		case engine.EventAlchemy:
			send(ev.Player, pvMsg{id: 0x08c, focus: ev.Planet - 1, focusKnown: true, params: []int{ev.Planet - 1, ev.Count}})
		case engine.EventOrderClipped:
			if ev.Item == engine.ItemTerraform {
				send(ev.Player, atPlanet(0x12f, ev.Planet))
			} else {
				send(ev.Player, atPlanet(0x12a, ev.Planet))
			}
		case engine.EventLevelUp:
			if ev.Player >= 0 && ev.Player < len(g.Players) {
				// 0x136 is 0x050 for a Generalized Research player, with
				// the same slots (MESSAGES.md "0x050 vs 0x136"): level,
				// field, and the field researched next (Axes[0]).
				id := 0x050
				if g.Players[ev.Player].Race.LRT.GeneralizedResearch {
					id = 0x136
				}
				send(ev.Player, pvMsg{id: id, focus: pvFocusResearch, focusKnown: true,
					params: []int{ev.Count, int(ev.Item), ev.Axes[0]}, known: []bool{true, true, true}})
			}
		case engine.EventPartAvailable:
			// A part announcement's slots are field, the category mask
			// 1<<b and the item's index in its category; the focus is
			// the part code 0xC000 | b<<8 | index, or −3 for a ship hull
			// (MESSAGES.md "Slots and focus of the component
			// announcements" at stars-elegy a348c5d: MEASURED for the
			// categories the vectors hold, BINARY-ONLY for beams,
			// torpedoes, mine layers and the Genesis Device).
			announce := func(id int, c engine.Component) pvMsg {
				b := pvCategoryBits()[c.Category]
				focus := 0xc000 | b<<8 | c.Index
				if c.Category == engine.CatHull {
					focus = pvFocusDesign
				}
				return pvMsg{id: id, focus: focus, focusKnown: true, params: []int{int(ev.Item), 1 << b, c.Index}}
			}
			cs := engine.Components().Components
			if ev.Count >= 0 && ev.Count < len(cs) {
				c := cs[ev.Count]
				switch {
				case c.Category == engine.CatHull:
					send(ev.Player, announce(0x078, c))
				case c.Category == engine.CatStarbaseHull:
					// 0x0d0, BINARY-ONLY: not mapped.
				case c.Category == engine.CatPlanetary && pvPlanetaryIDs()[c.Name] != 0:
					send(ev.Player, announce(pvPlanetaryIDs()[c.Name], c))
				default:
					// Every other part, the Genesis Device included
					// (MESSAGES.md at stars-elegy f2778b3).
					send(ev.Player, announce(0x05f, c))
				}
			}
		case engine.EventResearchStolen:
			// Slots: field, research added (MESSAGES.md 0x159).
			send(ev.Player, pvMsg{id: 0x159, focus: pvFocusResearch, focusKnown: true, params: []int{int(ev.Item), ev.Count}})
		case engine.EventColonistsBeamedDown:
			// Slots: fleet, amount (two slots), cargo type, location (two
			// slots). The cargo type of colonists is 3, and a planet
			// location is 65535 then the planet number (MEASURED,
			// stars-elegy CM-1, 4f59d8d: 14 and 39 records). Elegy sends
			// 0x02e only at the fleet's owner's own planet. The fleet
			// slot is owner·512 + (number − 1): CM-1 measured the plain
			// number for player 0's one-design fleets only, and the
			// owner·512 part for other players is an ASSUMPTION
			// (consistent with FS-1 and 0x028, not measured for 0x02e).
			// The amount's second slot as its high word is inferred
			// (CM-1: 0 in every record, at most 300). The fleet slot and
			// the focus need the fleet, so they are not compared when it
			// is gone by the end of the year.
			m := pvMsg{id: 0x02e, params: []int{0, ev.Count & 0xffff, ev.Count >> 16, pvCargoColonists, pvLocationPlanet, ev.Planet - 1},
				known: []bool{false, true, true, true, true, true}, why: "0x02e fleet gone by the end of the year"}
			if code, ok := fleet(ev.Fleet); ok {
				m.focus, m.focusKnown, m.params[0], m.known[0], m.why = pvFocusFleet|code, true, code, true, ""
			}
			send(ev.Player, m)
		case engine.EventTargetGone:
			// Slots: the follower, the gone target (MESSAGES.md 0x028;
			// params MEASURED, WK-1 and WK-2). The target slot is compared
			// when the event names the target (a fleet removed that year).
			m := pvMsg{id: 0x028, params: make([]int, 2), known: make([]bool, 2), why: "0x028 fleet gone by the end of the year"}
			if code, ok := fleet(ev.Fleet); ok {
				m = pvMsg{id: 0x028, focus: pvFocusFleet | code, focusKnown: true, params: []int{code, 0},
					known: []bool{true, false}, why: "0x028 target slot"}
				if len(ev.Axes) == 2 {
					m.params[1], m.known[1] = ev.Axes[0]<<9|(ev.Axes[1]-1), true
				}
			}
			send(ev.Player, m)
		case engine.EventFleetScrapped:
			if ev.Planet == -1 {
				// Slots: the salvage's object id, 0x2000 + owner·512 +
				// number, its number from the owner's pool shared with
				// packets (MESSAGES.md "Object ids in slots", MEASURED
				// SV-1), then the fleet: for a fleet of one design,
				// design·512 + fleet number, the design being its
				// ship-design slot (CONFIRMED by prediction, stars-elegy
				// FS-1). A fleet of several designs gave 0x2000 + design
				// ·512 + number once (MEASURED, FS-1; bit 13's meaning and
				// which design is named are not known), so its slot is
				// not compared.
				m := pvMsg{id: 0x05b, focus: pvFocusObject, focusKnown: true, params: []int{0, 0}, known: []bool{false, false},
					why: "0x05b salvage and fleet slots"}
				if len(ev.Axes) == 4 {
					salvage, number, slot, designs := ev.Axes[0], ev.Axes[1], ev.Axes[2], ev.Axes[3]
					var why []string
					if salvage >= 0 {
						m.params[0], m.known[0] = 1<<13|ev.Player<<9|salvage, true
					} else {
						why = append(why, "0x05b salvage slot (no salvage)")
					}
					switch {
					case designs != 1:
						why = append(why, "0x05b fleet slot of a fleet of several designs")
					case slot >= 16:
						// A design with no slot of the owner (a state
						// built without design orders).
						why = append(why, "0x05b fleet slot of a design with no slot")
					default:
						m.params[1], m.known[1] = slot<<9|(number-1), true
					}
					m.why = strings.Join(why, ", ")
				}
				send(ev.Player, m)
			} else if len(ev.Axes) == 7 {
				// At a planet the fleet's owner gets one message and the
				// planet's owner, if any, a second (MESSAGES.md, scrap
				// notes, CONFIRMED tk2/tk111), each of three slots by
				// MESSAGES.md's slot column: fleet, kT (one slot), planet.
				// No public record gives the kT or planet values, so they
				// are not compared.
				//
				// 0x059 (no starbase, no Ultimate Recycling): the fleet
				// slot is as in 0x05b, design·512 + number for a fleet of
				// one design (CONFIRMED by prediction, stars-elegy FS-1 p
				// and q). The original also sends 0x059 on a
				// colonization (cs/cs-003); Elegy does not, and no vector
				// carries a colonization's 0x059 (one would fail as not
				// sent).
				//
				// 0x140, to the planet's owner with 0x059: the fleet slot
				// is owner·512 + number (MEASURED, FS-1, 2 values). In
				// both the fleet's owner, the recipient and the planet's
				// owner were one player, so which of them the bits name
				// is not known; Elegy uses the fleet's owner.
				//
				// 0x05a (starbase, no Ultimate Recycling) and 0x141, to
				// the planet's owner with it when the scrap gave no
				// technology (0x13c/0x13d otherwise, not mapped): no
				// fleet slot is measured, so none is compared.
				//
				// Fleets of several designs and designs with no slot are
				// not compared in any of them. The Ultimate Recycling
				// messages (0x05c, 0x05d, 0x142, 0x143) are not mapped.
				number, slot, designs, how, owner, tech := ev.Axes[1], ev.Axes[2], ev.Axes[3], ev.Axes[4], ev.Axes[5], ev.Axes[6]
				msg := func(id int, why string) pvMsg {
					return pvMsg{id: id, focus: ev.Planet - 1, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false},
						why: fmt.Sprintf("0x%03x %s", id, why)}
				}
				fleetSlot := func(m *pvMsg, value int) {
					switch {
					case designs != 1:
						m.why += fmt.Sprintf(", 0x%03x fleet slot of a fleet of several designs", m.id)
					case slot >= 16:
						m.why += fmt.Sprintf(", 0x%03x fleet slot of a design with no slot", m.id)
					default:
						m.params[0], m.known[0] = value, true
					}
				}
				switch how {
				case 0:
					m := msg(0x059, "kT and planet slots")
					fleetSlot(&m, slot<<9|(number-1))
					send(ev.Player, m)
					// send drops 0x140 at an unowned planet (NoOwner).
					m = msg(0x140, "kT and planet slots")
					fleetSlot(&m, ev.Player<<9|(number-1))
					send(owner, m)
				case engine.ScrapAtStarbase:
					send(ev.Player, msg(0x05a, "fleet, kT and planet slots"))
					if tech == 0 {
						send(owner, msg(0x141, "fleet, kT and planet slots"))
					}
				}
			}
		case engine.EventMinesSwept:
			// Slots: sweeper (fleet or planet), amount (two slots), the
			// field's owner, type and centre (MESSAGES.md "Sweep slots",
			// MEASURED SW-1). The owner is the plain player number; that
			// it never carries a flag is UNRESOLVED.
			swept := []int{0, ev.Count & 0xffff, ev.Count >> 16, 0, 0, 0, 0}
			known := []bool{true, true, true, false, false, false, false}
			why := "field owner, type and location slots"
			if f, ok := sweptField(ev); ok {
				copy(swept[3:], []int{f.owner, f.kind, f.x, f.y})
				copy(known[3:], []bool{true, true, true, true})
				why = ""
				if ev.Planet >= 0 && f.kind == int(objects.SpeedBump) {
					// 0x0f4 with a speed bump is NOT RUN.
					known[4] = false
					why = "speed bump type slot (NOT RUN)"
				}
			}
			if ev.Planet >= 0 {
				swept[0] = ev.Planet - 1
				send(ev.Player, pvMsg{id: 0x0f4, focus: ev.Planet - 1, focusKnown: true, params: swept, known: known,
					why: whyNot(0x0f4, why)})
			} else if code, ok := fleet(ev.Fleet); ok {
				swept[0] = code
				send(ev.Player, pvMsg{id: 0x0c2, focus: pvFocusFleet | code, focusKnown: true, params: swept, known: known,
					why: whyNot(0x0c2, why)})
			} else {
				send(ev.Player, pvMsg{id: 0x0c2, params: swept, known: make([]bool, len(swept)),
					why: "0x0c2 fleet gone by the end of the year"})
			}
		case engine.EventMinefieldSwept:
			// Slots: the field's object id (owner·512 + number), amount
			// (two slots), type and centre (MESSAGES.md "Sweep slots",
			// "Object ids in slots"; MEASURED SW-1); the focus is the
			// field.
			params := []int{0, ev.Count & 0xffff, ev.Count >> 16, 0, 0, 0}
			known := []bool{false, true, true, false, false, false}
			why := "0x0be field, type and location slots"
			if f, ok := sweptField(ev); ok {
				params[0] = f.owner<<9 | f.number
				copy(params[3:], []int{f.kind, f.x, f.y})
				known = []bool{true, true, true, true, true, true}
				why = ""
			}
			send(ev.Player, pvMsg{id: 0x0be, focus: pvFocusObject, focusKnown: true, params: params, known: known, why: why})
		case engine.EventCometStrike:
			if ev.Count >= 0 && ev.Count < 4 {
				send(ev.Player, atPlanet(0x083+ev.Count, ev.Planet))
			}
		case engine.EventPlanetEmptied:
			// Growth's cause (MESSAGES.md 0x023, 0x040): an Alternate
			// Reality owner gets the variants 0x024 and 0x041
			// (BINARY-ONLY).
			ar := 0
			if ev.Player >= 0 && ev.Player < len(g.Players) && g.Players[ev.Player].Race.PRT == engine.PRTAlternateReality {
				ar = 1
			}
			switch ev.Count {
			case engine.EmptiedLeft:
				send(ev.Player, atPlanet(0x040+ar, ev.Planet))
			case engine.EmptiedDiedOut, engine.EmptiedLeftShownDiedOut:
				send(ev.Player, atPlanet(0x023+ar, ev.Planet))
			}
			if i > 0 && events[i-1].Kind == engine.EventBombed && events[i-1].Planet == ev.Planet {
				// The fleet whose plan started the pass, saved at bombing
				// time (it can be gone by the end of the year). With more
				// than one of the bomber's fleets in orbit the message is
				// 0x17c/0x17d (MESSAGES.md 0x08f, 0x17c; CONFIRMED
				// tk2/tk121). Which fleets count (all in orbit, or those
				// with bombs) and which fleet the message names when the
				// pass was started by one without bombs: ASSUMPTION.
				b := events[i-1]
				m := pvMsg{id: 0x08f, params: []int{0, ev.Planet - 1}, known: []bool{false, true}, why: "0x08f bombing fleet"}
				if len(b.Axes) == 2 && b.Axes[1] > 1 {
					m.id, m.why = 0x17c, "0x17c fleet slot"
				} else if len(b.Axes) == 2 {
					code := b.Player<<9 | (b.Axes[0] - 1)
					m.params[0], m.known[0], m.why = code, true, ""
					m.focus, m.focusKnown = pvFocusFleet|code, true
				}
				send(b.Player, m)
				m.id, m.focus, m.focusKnown = m.id+1, ev.Planet-1, true
				m.known = slices.Clone(m.known)
				m.params = slices.Clone(m.params)
				if m.why != "" {
					m.why = fmt.Sprintf("0x%03x%s", m.id, m.why[5:])
				}
				send(ev.Player, m)
			}
		}
	}
	out := map[int][]pvMsg{}
	for _, m := range msgs {
		if !m.gone {
			out[m.player] = append(out[m.player], m.pvMsg)
		}
	}
	return out
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

// pvSeqEntry is one entry of a message_sequence expectation (vectors
// README.md "Message sequences").
type pvSeqEntry struct {
	ID     int   `json:"id"`
	Focus  int   `json:"focus"`
	Params []int `json:"params"`
	Sample bool  `json:"sample"`
}

// pvComputerKept reports whether a computer player's message file keeps
// message id (MESSAGES.md "Computer players": 0x007, 0x023, 0x040 and
// 0x08f; the one computer-player setting that keeps everything is read as
// not used by the vectors, ASSUMPTION).
func pvComputerKept(id int) bool {
	return id == 0x007 || id == 0x023 || id == 0x040 || id == 0x08f
}

// pvQueueMessage reports whether message id comes from a planet's
// production queue: a build count, a finished queue or no queue.
func pvQueueMessage(id int) bool {
	return id >= 0x035 && id <= 0x038 || id == 0x03e || id == 0x03f
}

// pvSeqMatch reports whether Elegy's message m is the entry w: the same
// id, and the same focus and slot values where Elegy knows them.
func pvSeqMatch(m pvMsg, w pvSeqEntry) bool {
	if m.id != w.ID || m.focusKnown && m.focus != w.Focus {
		return false
	}
	if len(m.known) > 0 && len(w.Params) != len(m.params) {
		return false
	}
	for i, k := range m.known {
		if k && w.Params[i] != m.params[i] {
			return false
		}
	}
	return true
}

// String shows a message as "0x036 (focus 7, params [2 7])", with "?"
// for a focus or slot Elegy does not know.
func (m pvMsg) String() string {
	focus := "?"
	if m.focusKnown {
		focus = fmt.Sprint(m.focus)
	}
	params := make([]string, len(m.params))
	for i, v := range m.params {
		params[i] = "?"
		if m.known[i] {
			params[i] = fmt.Sprint(v)
		}
	}
	if len(m.known) == 0 {
		return fmt.Sprintf("0x%03x (focus %s)", m.id, focus)
	}
	return fmt.Sprintf("0x%03x (focus %s, params [%s])", m.id, focus, strings.Join(params, " "))
}

// String shows an entry as pvMsg.String does.
func (w pvSeqEntry) String() string {
	return fmt.Sprintf("0x%03x (focus %d, params %v)", w.ID, w.Focus, w.Params)
}

// sequenceCheck compares a message_sequence expectation (vectors
// README.md "Message sequences", order rule) with the messages Elegy's
// year sends that player (pvMessageList). The entries without `sample`
// must all be sent, in the same relative order, with the same focus and
// slot values where Elegy knows them; when no entry is a sample, nothing
// else may be sent. Only ids pvMessageIDs maps are compared: an entry of
// another id is skipped, and Elegy sends none. A sample entry is not
// compared: its place, presence and values are one stream's draws.
func (l *pvLoaded) sequenceCheck(g *engine.Game, e pvExpect) string {
	if e.Player == nil || e.Messages == nil {
		return "message_sequence: malformed expectation (player and messages are needed)"
	}
	player := *e.Player
	if l.planned(player) {
		return "skip: " + pvPlannedWhy
	}
	if e.Year >= len(l.events) {
		return "skip: no events for the year"
	}
	computer := player >= 0 && player < len(g.Players) && g.Players[player].Computer
	var got []pvMsg
	for _, m := range pvMessageList(g, l.events[e.Year])[player] {
		// A planet whose queue the harness cannot load runs with none,
		// so its queue messages are not compared.
		if computer && !pvComputerKept(m.id) || l.queued[player] && pvQueueMessage(m.id) {
			continue
		}
		got = append(got, m)
	}
	ids := pvMessageIDs()
	skip := ""
	skipOnce := func(why string) {
		if skip == "" {
			skip = why
		}
	}
	// A Claim Adjuster planet's 0x156 percent is measured after its 0x15c
	// drift (MESSAGES.md P7 order; KERNEL.md "Terraforming" → Claim
	// Adjuster), a draw, so a 0x156 is a sample when the record's drift
	// there is a sample or Elegy's run drew a drift there.
	drifted := map[int]bool{}
	for _, w := range e.Messages {
		if w.ID == 0x15c && w.Sample {
			drifted[w.Focus] = true
		}
	}
	for _, m := range got {
		if m.id == 0x15c {
			drifted[m.focus] = true
		}
	}
	samples := false
	var want []pvSeqEntry
	for _, w := range e.Messages {
		_, mapped := ids[w.ID]
		switch {
		case w.Sample, w.ID == 0x156 && drifted[w.Focus]:
			samples = true
		case !mapped:
			skipOnce(fmt.Sprintf("message_sequence entry 0x%03x has no event mapping", w.ID))
		case l.queued[player] && pvQueueMessage(w.ID):
			skipOnce("message_sequence entry from a production queue not loaded")
		default:
			want = append(want, w)
		}
	}
	var errs []string
	used := make([]bool, len(got))
	pos := 0
	for _, w := range want {
		j := pos
		for j < len(got) && !pvSeqMatch(got[j], w) {
			j++
		}
		if j < len(got) {
			used[j], pos = true, j+1
			if got[j].why != "" {
				skipOnce("message_sequence " + got[j].why + " not compared")
			}
			continue
		}
		errs = append(errs, pvSeqMiss(got, used, pos, w))
	}
	if !samples {
		for j, m := range got {
			if !used[j] {
				errs = append(errs, fmt.Sprintf("message %v = sent, want not sent", m))
			}
		}
	}
	if len(want) == 0 && samples {
		skipOnce("message_sequence of sample entries only")
	}
	return pvResult(errs, skip)
}

// pvSeqMiss describes an entry w that Elegy's messages got do not hold
// at or after pos: sent too early (before the message matched at pos−1),
// sent with other values, or not sent.
func pvSeqMiss(got []pvMsg, used []bool, pos int, w pvSeqEntry) string {
	for j := 0; j < pos; j++ {
		if !used[j] && pvSeqMatch(got[j], w) {
			return fmt.Sprintf("message %v = sent before %v, want after", w, got[pos-1])
		}
	}
	for j, m := range got {
		if !used[j] && m.id == w.ID {
			return fmt.Sprintf("message %v = sent as %v", w, m)
		}
	}
	return fmt.Sprintf("message %v = not sent", w)
}

// pvSweptField is the swept field an EventMinesSwept or
// EventMinefieldSwept names in Axes.
type pvSweptField struct{ owner, kind, x, y, number int }

func sweptField(ev engine.Event) (pvSweptField, bool) {
	if len(ev.Axes) != 5 {
		return pvSweptField{}, false
	}
	a := ev.Axes
	return pvSweptField{owner: a[0], kind: a[1], x: a[2], y: a[3], number: a[4]}, true
}

// whyNot prefixes a not-compared reason with the message id; an empty
// reason stays empty.
func whyNot(id int, why string) string {
	if why == "" {
		return ""
	}
	return fmt.Sprintf("0x%03x %s", id, why)
}

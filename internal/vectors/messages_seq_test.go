package vectors

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestPvMessageList pins each mapped message's focus and slots (vectors
// README.md "Message sequences"; MESSAGES.md "Catalogue").
func TestPvMessageList(t *testing.T) {
	g := &engine.Game{
		Planets: []engine.Planet{{ID: 5, Owner: 2}},
		Players: make([]engine.Player, 3),
		Fleets:  []engine.Fleet{{ID: 9, Owner: 1, Number: 4}},
	}
	g.Players[1].Race.LRT.GeneralizedResearch = true
	part, hull, defense := -1, -1, -1
	named := map[string]int{}
	for i, c := range engine.Components().Components {
		named[c.Name] = i
		switch {
		case c.Name == "Cow-hide Shield":
			part = i
		case c.Category == engine.CatHull && hull < 0:
			hull = i
		case c.Category == engine.CatPlanetary && pvPlanetaryIDs()[c.Name] == 0x145 && defense < 0:
			defense = i
		}
	}
	all := func(n int) []bool {
		k := make([]bool, n)
		for i := range k {
			k[i] = true
		}
		return k
	}
	sb := func(fleet, n int) engine.Event {
		return engine.Event{Kind: engine.EventShipsBuilt, Player: 1, Planet: 5, Fleet: fleet, Count: n, Axes: []int{2}}
	}
	route := func(k engine.EventKind, fleet int) engine.Event {
		return engine.Event{Kind: k, Player: 1, Planet: 7, Fleet: fleet, Count: 6}
	}
	for _, c := range []struct {
		name   string
		events []engine.Event
		player int
		want   []pvMsg
	}{
		{"one factory", []engine.Event{{Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemFactory, Count: 1}}, 2,
			[]pvMsg{{id: 0x035, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"race penalized", []engine.Event{{Kind: engine.EventRacePenalized, Player: 1, Planet: -1, Fleet: -1}, {Kind: engine.EventRaceHacked, Player: 2, Planet: -1, Fleet: -1, Count: 1}}, 1,
			[]pvMsg{{id: 0x117, focus: 0xffff, focusKnown: true, params: []int{}, known: []bool{}}}},
		{"another race penalized", []engine.Event{{Kind: engine.EventRacePenalized, Player: 1, Planet: -1, Fleet: -1}, {Kind: engine.EventRaceHacked, Player: 2, Planet: -1, Fleet: -1, Count: 1}}, 2,
			[]pvMsg{{id: 0x182, focus: 0xffff, focusKnown: true, params: []int{1}, known: []bool{false}, why: "0x182 player slot"}}},
		{"merged count", []engine.Event{{Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemMine, Count: 1}, {Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemMine, Count: 3}}, 2,
			[]pvMsg{{id: 0x038, focus: 4, focusKnown: true, params: []int{4, 4}, known: all(2)}}},
		{"defenses", []engine.Event{{Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemDefenses, Count: 20}}, 2,
			[]pvMsg{{id: 0x03a, focus: 4, focusKnown: true, params: []int{20, 4}, known: all(2)}}},
		{"overcrowded, long amount", []engine.Event{{Kind: engine.EventOvercrowded, Player: 2, Planet: 5, Count: 70000}}, 2,
			[]pvMsg{{id: 0x026, focus: 4, focusKnown: true, params: []int{4, 70000 - 65536, 1}, known: all(3)}}},
		// MESSAGES.md 0x025: each count in two 16-bit slots, as KX-004
		// S11's [2 9101 0 8928 0]; a long count fills the high slot.
		{"hostile shrink", []engine.Event{{Kind: engine.EventHostileShrink, Player: 2, Planet: 5, Fleet: -1, Count: 9101, Axes: []int{8928}}}, 2,
			[]pvMsg{{id: 0x025, focus: 4, focusKnown: true, params: []int{4, 9101, 0, 8928, 0}, known: all(5)}}},
		{"hostile shrink, long counts", []engine.Event{{Kind: engine.EventHostileShrink, Player: 2, Planet: 5, Fleet: -1, Count: 70000, Axes: []int{65000}}}, 2,
			[]pvMsg{{id: 0x025, focus: 4, focusKnown: true, params: []int{4, 70000 - 65536, 1, 65000, 0}, known: all(5)}}},
		// MESSAGES.md 0x156 (planet, percent) and 0x15c (planet, axis).
		{"claim adjusted", []engine.Event{{Kind: engine.EventClaimAdjusted, Player: 2, Planet: 5, Fleet: -1, Count: 91}}, 2,
			[]pvMsg{{id: 0x156, focus: 4, focusKnown: true, params: []int{4, 91}, known: all(2)}}},
		{"original drift", []engine.Event{{Kind: engine.EventOriginalDrift, Player: 2, Planet: 5, Fleet: -1, Axes: []int{2}}}, 2,
			[]pvMsg{{id: 0x15c, focus: 4, focusKnown: true, params: []int{4, 2}, known: all(2)}}},
		// MESSAGES.md 0x0cd (planet, owner·32 + 16 + starbase slot): SL-12's
		// player 1 Ultra Station in slot 2 gave 50.
		{"starbase built", []engine.Event{{Kind: engine.EventStarbaseBuilt, Player: 1, Planet: 5, Fleet: -1, Count: engine.DockUnlimited, Axes: []int{2}}}, 1,
			[]pvMsg{{id: 0x0cd, focus: 4, focusKnown: true, params: []int{4, 50}, known: all(2)}}},
		{"starbase built, no slot", []engine.Event{{Kind: engine.EventStarbaseBuilt, Player: 1, Planet: 5, Fleet: -1}}, 1, nil},
		{"no queue", []engine.Event{{Kind: engine.EventNoQueue, Player: 2, Planet: 5}}, 2,
			[]pvMsg{{id: 0x03f, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"queue completed", []engine.Event{{Kind: engine.EventQueueCompleted, Player: 2, Planet: 5}}, 2,
			[]pvMsg{{id: 0x03e, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"comet", []engine.Event{{Kind: engine.EventCometStrike, Player: 0, Planet: 5, Count: 1}}, 0,
			[]pvMsg{{id: 0x084, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"fleet finished", []engine.Event{{Kind: engine.EventFleetArrived, Player: 1, Fleet: 9}}, 1,
			[]pvMsg{{id: 0x04e, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3}, known: all(1)}}},
		{"fleet finished, then gone", []engine.Event{{Kind: engine.EventFleetArrived, Player: 1, Fleet: 8}}, 1,
			[]pvMsg{{id: 0x04e, params: []int{0}, known: []bool{false}, why: "0x04e fleet gone by the end of the year"}}},
		// MESSAGES.md 0x02f–0x034: planet, (count,) design as owner·32 +
		// slot, (route target); focus the new fleet.
		{"one ship built", []engine.Event{sb(9, 1)}, 1,
			[]pvMsg{{id: 0x02f, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 34}, known: all(2)}}},
		{"several ships built", []engine.Event{sb(9, 3)}, 1,
			[]pvMsg{{id: 0x030, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 3, 34}, known: all(3)}}},
		{"one ship routed", []engine.Event{sb(9, 1), route(engine.EventFleetRouted, 9)}, 1,
			[]pvMsg{{id: 0x031, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 34, 6}, known: all(3)}}},
		{"several ships routed", []engine.Event{sb(9, 3), route(engine.EventFleetRouted, 9)}, 1,
			[]pvMsg{{id: 0x032, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 3, 34, 6}, known: all(4)}}},
		{"one ship not routed", []engine.Event{sb(9, 1), route(engine.EventFleetNotRouted, 9)}, 1,
			[]pvMsg{{id: 0x033, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 34, 6}, known: all(3)}}},
		{"several ships not routed", []engine.Event{sb(9, 3), route(engine.EventFleetNotRouted, 9)}, 1,
			[]pvMsg{{id: 0x034, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 3, 34, 6}, known: all(4)}}},
		{"ship built, fleet gone", []engine.Event{sb(8, 1)}, 1,
			[]pvMsg{{id: 0x02f, params: []int{4, 34}, known: all(2), why: "0x02f new fleet gone by the end of the year"}}},
		{"ship routed, fleet gone", []engine.Event{sb(8, 1), route(engine.EventFleetRouted, 8)}, 1,
			[]pvMsg{{id: 0x031, params: []int{4, 34, 6}, known: all(3), why: "0x031 new fleet gone by the end of the year"}}},
		{"ship routed, then routed again on arrival", []engine.Event{sb(9, 1), route(engine.EventFleetRouted, 9), route(engine.EventFleetNotRouted, 9)}, 1,
			[]pvMsg{{id: 0x031, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 34, 6}, known: all(3)}}},
		{"ships built, no slot", []engine.Event{{Kind: engine.EventShipsBuilt, Player: 1, Planet: 5, Fleet: 9, Count: 1}}, 1, nil},
		{"route leg of another fleet", []engine.Event{sb(9, 1), route(engine.EventFleetRouted, 7)}, 1,
			[]pvMsg{{id: 0x02f, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 34}, known: all(2)}}},
		// MESSAGES.md 0x139 (planet, count, design, fleet; focus the
		// fleet) and 0x0ba (planet, count, design), BINARY-ONLY slots.
		{"ships joined at the fleet limit", []engine.Event{{Kind: engine.EventShipsJoinedFleet, Player: 1, Planet: 5, Fleet: 9, Count: 3}}, 1,
			[]pvMsg{{id: 0x139, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{4, 3, 0, 1<<9 | 3}, known: []bool{true, true, false, true}, why: "0x139 design slot"}}},
		{"ships joined at the fleet limit, fleet gone", []engine.Event{{Kind: engine.EventShipsJoinedFleet, Player: 1, Planet: 5, Fleet: 8, Count: 3}}, 1,
			[]pvMsg{{id: 0x139, params: []int{4, 3, 0, 0}, known: []bool{true, true, false, false}, why: "0x139 design slot; fleet gone by the end of the year"}}},
		{"ships lost at the fleet limit", []engine.Event{{Kind: engine.EventShipsLostFleetLimit, Player: 2, Planet: 5, Fleet: -1, Count: 2}}, 2,
			[]pvMsg{{id: 0x0ba, focus: 4, focusKnown: true, params: []int{4, 2, 0}, known: []bool{true, true, false}, why: "0x0ba design slot"}}},
		{"level up", []engine.Event{{Kind: engine.EventLevelUp, Player: 2, Item: 4, Count: 7, Axes: []int{3}}}, 2,
			[]pvMsg{{id: 0x050, focus: 0xfffe, focusKnown: true, params: []int{7, 4, 3}, known: all(3)}}},
		{"level up, Generalized Research", []engine.Event{{Kind: engine.EventLevelUp, Player: 1, Item: 4, Count: 7, Axes: []int{3}}}, 1,
			[]pvMsg{{id: 0x136, focus: 0xfffe, focusKnown: true, params: []int{7, 4, 3}, known: all(3)}}},
		{"research stolen", []engine.Event{{Kind: engine.EventResearchStolen, Player: 2, Planet: -1, Fleet: -1, Item: 3, Count: 116}}, 2,
			[]pvMsg{{id: 0x159, focus: 0xfffe, focusKnown: true, params: []int{3, 116}, known: all(2)}}},
		{"component", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 1, Count: part}}, 2,
			[]pvMsg{{id: 0x05f, focus: 0xC201, focusKnown: true, params: []int{1, 0x0004, 1}, known: all(3)}}},
		// The part codes of MESSAGES.md's examples (stars-elegy a348c5d).
		{"terraforming part", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 5, Count: named["Temp Terraform ±7"]}}, 2,
			[]pvMsg{{id: 0x05f, focus: 0xCD0D, focusKnown: true, params: []int{5, 0x2000, 13}, known: all(3)}}},
		{"planetary scanner", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 4, Count: named["Scoper 220"]}}, 2,
			[]pvMsg{{id: 0x157, focus: 0xCF03, focusKnown: true, params: []int{4, 0x8000, 3}, known: all(3)}}},
		// BINARY-ONLY: no vector holds a beam, torpedo, mine layer or
		// Genesis Device announcement; these rows pin MESSAGES.md's table.
		{"Genesis Device", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 5, Count: named["Genesis Device"]}}, 2,
			[]pvMsg{{id: 0x05f, focus: 0xCF0E, focusKnown: true, params: []int{5, 0x8000, 14}, known: all(3)}}},
		{"beam", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 1, Count: named["Gatling Gun"]}}, 2,
			[]pvMsg{{id: 0x05f, focus: 0xC408, focusKnown: true, params: []int{1, 0x0010, 8}, known: all(3)}}},
		{"torpedo", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 1, Count: named["Beta Torpedo"]}}, 2,
			[]pvMsg{{id: 0x05f, focus: 0xC501, focusKnown: true, params: []int{1, 0x0020, 1}, known: all(3)}}},
		{"mine layer", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 2, Count: named["Mine Dispenser 50"]}}, 2,
			[]pvMsg{{id: 0x05f, focus: 0xC801, focusKnown: true, params: []int{2, 0x0100, 1}, known: all(3)}}},
		{"ship hull", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 3, Count: hull}}, 2,
			[]pvMsg{{id: 0x078, focus: 0xFFFD, focusKnown: true, params: []int{3, 0x4000, 0}, known: all(3)}}},
		{"planetary defense", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 0, Count: defense}}, 2,
			[]pvMsg{{id: 0x145, focus: 0xCF09, focusKnown: true, params: []int{0, 0x8000, 9}, known: all(3)}}},
		// MESSAGES.md 0x07b; the hab value as in KX-005 T0 (temperature
		// down to 59 is 315, radiation up to 46 is 558).
		{"terraform click down", []engine.Event{{Kind: engine.EventTerraformed, Player: 2, Planet: 5, Fleet: -1, Item: engine.ItemTerraform, Count: -1, Axes: []int{1, 59}}}, 2,
			[]pvMsg{{id: 0x07b, focus: 4, focusKnown: true, params: []int{4, 0, 1, 315}, known: all(4)}}},
		{"terraform click up", []engine.Event{{Kind: engine.EventTerraformed, Player: 2, Planet: 5, Fleet: -1, Item: engine.ItemTerraform, Count: 1, Axes: []int{2, 46}}}, 2,
			[]pvMsg{{id: 0x07b, focus: 4, focusKnown: true, params: []int{4, 1, 2, 558}, known: all(4)}}},
		// SV-1: player 1's salvage number 2 is 8706.
		// FS-1 case d: player 1's fleet 2 (number 3), design 3, salvage 2.
		{"scrapped in deep space", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 1, Planet: -1, Fleet: 9, Axes: []int{2, 3, 3, 1}}}, 1,
			[]pvMsg{{id: 0x05b, focus: 0xfffa, focusKnown: true, params: []int{8706, 1538}, known: all(2)}}},
		// FS-1 case m: design 15, the top slot.
		{"scrapped in deep space, design 15", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: -1, Fleet: 9, Axes: []int{0, 9, 15, 1}}}, 0,
			[]pvMsg{{id: 0x05b, focus: 0xfffa, focusKnown: true, params: []int{8192, 7688}, known: all(2)}}},
		{"scrapped in deep space, several designs", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: -1, Fleet: 9, Axes: []int{1, 7, 2, 2}}}, 0,
			[]pvMsg{{id: 0x05b, focus: 0xfffa, focusKnown: true, params: []int{8193, 0}, known: []bool{true, false}, why: "0x05b fleet slot of a fleet of several designs"}}},
		{"scrapped in deep space, design with no slot", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: -1, Fleet: 9, Axes: []int{1, 7, 16, 1}}}, 0,
			[]pvMsg{{id: 0x05b, focus: 0xfffa, focusKnown: true, params: []int{8193, 0}, known: []bool{true, false}, why: "0x05b fleet slot of a design with no slot"}}},
		// FS-1 case p: player 0's fleet 0/7 (number 8), design 9, at its
		// own planet 3 with no starbase: 0x059 with fleet slot 4615, then
		// 0x140 with 7 (owner·512 + number, MEASURED).
		{"scrapped at a planet", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 9, 1, 0, 0, 0}}}, 0,
			[]pvMsg{{id: 0x059, focus: 2, focusKnown: true, params: []int{4615, 0, 0}, known: []bool{true, false, false}, why: "0x059 kT and planet slots"},
				{id: 0x140, focus: 2, focusKnown: true, params: []int{7, 0, 0}, known: []bool{true, false, false}, why: "0x140 kT and planet slots"}}},
		// FS-1 case q: player 1's fleet 1/6, design 4: 2054 and 518.
		{"scrapped at a planet, player 1", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 1, Planet: 3, Fleet: 9, Axes: []int{-1, 7, 4, 1, 0, 1, 0}}}, 1,
			[]pvMsg{{id: 0x059, focus: 2, focusKnown: true, params: []int{2054, 0, 0}, known: []bool{true, false, false}, why: "0x059 kT and planet slots"},
				{id: 0x140, focus: 2, focusKnown: true, params: []int{518, 0, 0}, known: []bool{true, false, false}, why: "0x140 kT and planet slots"}}},
		// At another player's planet the planet's owner gets 0x140.
		{"scrapped at a foreign planet, fleet owner", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 1, Planet: 3, Fleet: 9, Axes: []int{-1, 7, 4, 1, 0, 2, 0}}}, 1,
			[]pvMsg{{id: 0x059, focus: 2, focusKnown: true, params: []int{2054, 0, 0}, known: []bool{true, false, false}, why: "0x059 kT and planet slots"}}},
		{"scrapped at a foreign planet, planet owner", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 1, Planet: 3, Fleet: 9, Axes: []int{-1, 7, 4, 1, 0, 2, 0}}}, 2,
			[]pvMsg{{id: 0x140, focus: 2, focusKnown: true, params: []int{518, 0, 0}, known: []bool{true, false, false}, why: "0x140 kT and planet slots"}}},
		{"scrapped at an unowned planet", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 9, 1, 0, engine.NoOwner, 0}}}, 0,
			[]pvMsg{{id: 0x059, focus: 2, focusKnown: true, params: []int{4615, 0, 0}, known: []bool{true, false, false}, why: "0x059 kT and planet slots"}}},
		{"scrapped at a planet, several designs", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 2, 2, 0, 0, 0}}}, 0,
			[]pvMsg{{id: 0x059, focus: 2, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false}, why: "0x059 kT and planet slots, 0x059 fleet slot of a fleet of several designs"},
				{id: 0x140, focus: 2, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false}, why: "0x140 kT and planet slots, 0x140 fleet slot of a fleet of several designs"}}},
		{"scrapped at a planet, design with no slot", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 16, 1, 0, 0, 0}}}, 0,
			[]pvMsg{{id: 0x059, focus: 2, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false}, why: "0x059 kT and planet slots, 0x059 fleet slot of a design with no slot"},
				{id: 0x140, focus: 2, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false}, why: "0x140 kT and planet slots, 0x140 fleet slot of a design with no slot"}}},
		// At a starbase: 0x05a, then 0x141 to the planet's owner unless
		// the scrap gave it a technology. No slot is compared.
		{"scrapped at a starbase", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 9, 1, engine.ScrapAtStarbase, 0, 0}}}, 0,
			[]pvMsg{{id: 0x05a, focus: 2, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false}, why: "0x05a fleet, kT and planet slots"},
				{id: 0x141, focus: 2, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false}, why: "0x141 fleet, kT and planet slots"}}},
		{"scrapped at a foreign starbase, planet owner", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 9, 1, engine.ScrapAtStarbase, 1, 0}}}, 1,
			[]pvMsg{{id: 0x141, focus: 2, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false}, why: "0x141 fleet, kT and planet slots"}}},
		{"scrapped at a starbase, technology gained", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 9, 1, engine.ScrapAtStarbase, 0, 1}}}, 0,
			[]pvMsg{{id: 0x05a, focus: 2, focusKnown: true, params: []int{0, 0, 0}, known: []bool{false, false, false}, why: "0x05a fleet, kT and planet slots"}}},
		// 0x05c, 0x05d, 0x142 and 0x143 are not mapped.
		{"scrapped at a planet with Ultimate Recycling", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 9, 1, engine.ScrapRecycled, 0, 0}}}, 0, nil},
		{"scrapped at a starbase with Ultimate Recycling", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 0, Planet: 3, Fleet: 9, Axes: []int{-1, 8, 9, 1, engine.ScrapAtStarbase | engine.ScrapRecycled, 0, 0}}}, 0, nil},
		{"scrapped in deep space, no salvage", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 1, Planet: -1, Fleet: 9, Axes: []int{-1, 1, 0, 1}}}, 1,
			[]pvMsg{{id: 0x05b, focus: 0xfffa, focusKnown: true, params: []int{0, 0}, known: []bool{false, true}, why: "0x05b salvage slot (no salvage)"}}},
		{"repelled", []engine.Event{{Kind: engine.EventDropRepelled, Player: 1, Planet: 5, Count: 3}}, 2,
			[]pvMsg{{id: 0x004, focus: 4, focusKnown: true, params: []int{4, 3, 0, 0x31}, known: all(4)}}},
		// MESSAGES.md "Invasion slots": RG-003-Y7-2469's 0x004 is [51, 550, 0, 49].
		{"repelled, long troops", []engine.Event{{Kind: engine.EventDropRepelled, Player: 3, Planet: 5, Count: 70000}}, 2,
			[]pvMsg{{id: 0x004, focus: 4, focusKnown: true, params: []int{4, 70000 - 65536, 1, 0x33}, known: all(4)}}},
		{"captured, to the winner", []engine.Event{{Kind: engine.EventPlanetCaptured, Player: 1, Planet: 5, Fleet: -1, Count: 600, Axes: []int{2}}}, 1,
			[]pvMsg{{id: 0x00c, focus: 4, focusKnown: true, params: []int{0x22, 4}, known: all(2)}}},
		// RG-003-Y12-2474's 0x007 is [49, 49, 600, 0].
		{"captured, to the former owner", []engine.Event{{Kind: engine.EventPlanetCaptured, Player: 1, Planet: 5, Fleet: -1, Count: 70000, Axes: []int{2}}}, 2,
			[]pvMsg{{id: 0x007, focus: 4, focusKnown: true, params: []int{0x31, 4, 70000 - 65536, 1}, known: all(4)}}},
		{"bombed empty, to the bomber", []engine.Event{{Kind: engine.EventBombed, Player: 1, Planet: 5, Fleet: 9, Axes: []int{4, 1}}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 5}}, 1,
			[]pvMsg{{id: 0x08f, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3, 4}, known: all(2)}}},
		{"bombed empty, to the owner", []engine.Event{{Kind: engine.EventBombed, Player: 1, Planet: 5, Fleet: 9, Axes: []int{4, 1}}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 5}}, 2,
			[]pvMsg{{id: 0x090, focus: 4, focusKnown: true, params: []int{1<<9 | 3, 4}, known: all(2)}}},
		{"bombed empty by two fleets, to the bomber", []engine.Event{{Kind: engine.EventBombed, Player: 1, Planet: 5, Fleet: 9, Axes: []int{4, 2}}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 5}}, 1,
			[]pvMsg{{id: 0x17c, params: []int{0, 4}, known: []bool{false, true}, why: "0x17c fleet slot"}}},
		{"bombed empty by two fleets, to the owner", []engine.Event{{Kind: engine.EventBombed, Player: 1, Planet: 5, Fleet: 9, Axes: []int{4, 2}}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 5}}, 2,
			[]pvMsg{{id: 0x17d, focus: 4, focusKnown: true, params: []int{0, 4}, known: []bool{false, true}, why: "0x17d fleet slot"}}},
		{"bombed empty, no fleet named", []engine.Event{{Kind: engine.EventBombed, Player: 1, Planet: 5}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 5}}, 2,
			[]pvMsg{{id: 0x090, focus: 4, focusKnown: true, params: []int{0, 4}, known: []bool{false, true}, why: "0x090 bombing fleet"}}},
		{"battle summary", []engine.Event{{Kind: engine.EventBattleSummary, Player: 0, Count: int(engine.SummaryDuelWonUnhit)}}, 0,
			[]pvMsg{{id: 0x091, known: []bool{}, why: "battle summary focus and slots"}}},
		{"alchemy", []engine.Event{{Kind: engine.EventAlchemy, Player: 2, Planet: 5, Item: engine.ItemMineralAlchemy, Count: 8}}, 2,
			[]pvMsg{{id: 0x08c, focus: 4, focusKnown: true, params: []int{4, 8}, known: all(2)}}},
		{"packet item cancelled", []engine.Event{{Kind: engine.EventPacketNoDriver, Player: 2, Planet: 5, Item: engine.ItemIroniumPacket, Count: 1}}, 2,
			[]pvMsg{{id: 0x129, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"planet left", []engine.Event{{Kind: engine.EventPlanetEmptied, Player: 2, Planet: 5, Count: engine.EmptiedLeft}}, 2,
			[]pvMsg{{id: 0x040, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"installation order cut", []engine.Event{{Kind: engine.EventOrderClipped, Player: 2, Planet: 5, Item: engine.ItemMine, Count: 3}}, 2,
			[]pvMsg{{id: 0x12a, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"terraform order cut", []engine.Event{{Kind: engine.EventOrderClipped, Player: 2, Planet: 5, Item: engine.ItemTerraform, Count: 0}}, 2,
			[]pvMsg{{id: 0x12f, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		// Player 1's fleet slot pins Elegy's owner·512 choice and the
		// amount 70000 its high-word reading (ASSUMPTION and inferred,
		// not measured: stars-elegy CM-1).
		{"colonists beamed down", []engine.Event{{Kind: engine.EventColonistsBeamedDown, Player: 1, Planet: 5, Fleet: 9, Count: 70000}}, 1,
			[]pvMsg{{id: 0x02e, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3, 70000 - 65536, 1, 3, 65535, 4},
				known: all(6)}}},
		{"target gone, named", []engine.Event{{Kind: engine.EventTargetGone, Player: 1, Planet: -1, Fleet: 9, Axes: []int{2, 6}}}, 1,
			[]pvMsg{{id: 0x028, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3, 2<<9 | 5}, known: all(2), why: "0x028 target slot"}}},
		{"target gone, unnamed", []engine.Event{{Kind: engine.EventTargetGone, Player: 1, Planet: -1, Fleet: 9}}, 1,
			[]pvMsg{{id: 0x028, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3, 0}, known: []bool{true, false}, why: "0x028 target slot"}}},
		{"colonists beamed down, fleet gone", []engine.Event{{Kind: engine.EventColonistsBeamedDown, Player: 1, Planet: 5, Fleet: 8, Count: 50}}, 1,
			[]pvMsg{{id: 0x02e, params: []int{0, 50, 0, 3, 65535, 4}, known: []bool{false, true, true, true, true, true}, why: "0x02e fleet gone by the end of the year"}}},
		// MESSAGES.md "Sweep slots" with SW-1's fields: player 1's
		// standard field 0 at (1050,1030), heavy field 1 at (1050,1180)
		// and speed bump 2 at (1080,1360).
		{"starbase swept, long amount", []engine.Event{{Kind: engine.EventMinesSwept, Player: 2, Planet: 5, Fleet: -1, Count: 70000, Axes: []int{1, 0, 1050, 1030, 0}}}, 2,
			[]pvMsg{{id: 0x0f4, focus: 4, focusKnown: true, params: []int{4, 70000 - 65536, 1, 1, 0, 1050, 1030}, known: all(7)}}},
		{"starbase swept a speed bump", []engine.Event{{Kind: engine.EventMinesSwept, Player: 2, Planet: 5, Fleet: -1, Count: 33, Axes: []int{1, 2, 1080, 1360, 2}}}, 2,
			[]pvMsg{{id: 0x0f4, focus: 4, focusKnown: true, params: []int{4, 33, 0, 1, 2, 1080, 1360},
				known: []bool{true, true, true, true, false, true, true}, why: "0x0f4 speed bump type slot (NOT RUN)"}}},
		{"fleet swept", []engine.Event{{Kind: engine.EventMinesSwept, Player: 1, Planet: -1, Fleet: 9, Count: 100, Axes: []int{0, 1, 1050, 1180, 1}}}, 1,
			[]pvMsg{{id: 0x0c2, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3, 100, 0, 0, 1, 1050, 1180}, known: all(7)}}},
		{"fleet swept a speed bump", []engine.Event{{Kind: engine.EventMinesSwept, Player: 1, Planet: -1, Fleet: 9, Count: 33, Axes: []int{2, 2, 1080, 1360, 2}}}, 1,
			[]pvMsg{{id: 0x0c2, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3, 33, 0, 2, 2, 1080, 1360}, known: all(7)}}},
		{"fleet swept, no field named", []engine.Event{{Kind: engine.EventMinesSwept, Player: 1, Planet: -1, Fleet: 9, Count: 82}}, 1,
			[]pvMsg{{id: 0x0c2, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3, 82, 0, 0, 0, 0, 0},
				known: []bool{true, true, true, false, false, false, false}, why: "0x0c2 field owner, type and location slots"}}},
		{"fleet swept, then gone", []engine.Event{{Kind: engine.EventMinesSwept, Player: 1, Planet: -1, Fleet: 8, Count: 82, Axes: []int{0, 1, 1050, 1180, 1}}}, 1,
			[]pvMsg{{id: 0x0c2, params: []int{0, 82, 0, 0, 1, 1050, 1180}, known: make([]bool, 7), why: "0x0c2 fleet gone by the end of the year"}}},
		{"field swept, to its owner", []engine.Event{{Kind: engine.EventMinesSwept, Player: 2, Planet: 5, Fleet: -1, Count: 33, Axes: []int{1, 2, 1080, 1360, 2}}, {Kind: engine.EventMinefieldSwept, Player: 1, Planet: 5, Fleet: -1, Count: 33, Axes: []int{1, 2, 1080, 1360, 2}}}, 1,
			[]pvMsg{{id: 0x0be, focus: 0xfffa, focusKnown: true, params: []int{514, 33, 0, 2, 1080, 1360}, known: all(6)}}},
		{"field swept, no field named", []engine.Event{{Kind: engine.EventMinefieldSwept, Player: 0, Planet: 5, Fleet: -1, Count: 82}}, 0,
			[]pvMsg{{id: 0x0be, focus: 0xfffa, focusKnown: true, params: []int{0, 82, 0, 0, 0, 0},
				known: []bool{false, true, true, false, false, false}, why: "0x0be field, type and location slots"}}},
		{"repeats kept", []engine.Event{{Kind: engine.EventNoQueue, Player: 2, Planet: 5}, {Kind: engine.EventNoQueue, Player: 2, Planet: 5}}, 2,
			[]pvMsg{{id: 0x03f, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}, {id: 0x03f, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
	} {
		if got := pvMessageList(g, c.events)[c.player]; !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
	if got := pvMessageSeq(g, []engine.Event{{Kind: engine.EventNoQueue, Player: 2, Planet: 5}, {Kind: engine.EventNoQueue, Player: 2, Planet: 5}})[2]; !reflect.DeepEqual(got, []int{0x03f}) {
		t.Errorf("pvMessageSeq lists %v, want each id once", got)
	}
}

// TestSequenceCheck pins the message_sequence order rule (vectors
// README.md "Message sequences").
func TestSequenceCheck(t *testing.T) {
	noQueue := func(player, planet int) engine.Event {
		return engine.Event{Kind: engine.EventNoQueue, Player: player, Planet: planet}
	}
	built := func(player, planet, n int) engine.Event {
		return engine.Event{Kind: engine.EventBuilt, Player: player, Planet: planet, Item: engine.ItemFactory, Count: n}
	}
	l := pvLoaded{
		g:       engine.Game{Players: []engine.Player{{}, {Computer: true}, {}, {Computer: true}}},
		queued:  map[int]bool{2: true},
		ordered: map[int]bool{3: true},
		events: [][]engine.Event{nil, {
			built(0, 8, 2), noQueue(0, 6), noQueue(0, 7),
			{Kind: engine.EventLevelUp, Player: 0, Item: 2, Count: 5, Axes: []int{3}},
			built(2, 5, 3), noQueue(2, 5), {Kind: engine.EventOvercrowded, Player: 2, Planet: 5, Count: 12},
			noQueue(3, 5), {Kind: engine.EventBombed, Player: 3, Planet: 9}, {Kind: engine.EventPlanetEmptied, Player: 0, Planet: 9},
		}},
	}
	g := &l.g
	i := func(n int) *int { return &n }
	seq := func(player int, ms ...pvSeqEntry) pvExpect {
		if ms == nil {
			ms = []pvSeqEntry{}
		}
		return pvExpect{Kind: "message_sequence", Year: 1, Player: i(player), Messages: ms}
	}
	m := func(id, focus int, params ...int) pvSeqEntry { return pvSeqEntry{ID: id, Focus: focus, Params: params} }
	sample := func(w pvSeqEntry) pvSeqEntry { w.Sample = true; return w }
	b36 := m(0x036, 7, 2, 7)
	q5, q6 := m(0x03f, 5, 5), m(0x03f, 6, 6)
	lv := m(0x050, 0xfffe, 5, 2, 3)
	empt := m(0x090, 8, 77, 8)
	whole := []pvSeqEntry{b36, q5, q6, lv, empt}
	for _, c := range []struct {
		name string
		e    pvExpect
		want string
	}{
		{"whole sequence", seq(0, whole...), "skip: message_sequence 0x090 bombing fleet not compared"},
		{"slots compared beside the unknown one", seq(0, b36, q5, q6, lv, m(0x090, 8, 77, 9)),
			"message 0x090 (focus 8, params [77 9]) = sent as 0x090 (focus 8, params [? 8]); message 0x090 (focus 8, params [? 8]) = sent, want not sent"},
		{"next field compared", seq(0, b36, q5, q6, m(0x050, 0xfffe, 5, 2, 4), empt),
			"message 0x050 (focus 65534, params [5 2 4]) = sent as 0x050 (focus 65534, params [5 2 3]); message 0x050 (focus 65534, params [5 2 3]) = sent, want not sent\nalso skipped: message_sequence 0x090 bombing fleet not compared"},
		{"out of order", seq(0, b36, q6, q5, lv, empt),
			"message 0x03f (focus 5, params [5]) = sent before 0x03f (focus 6, params [6]), want after; message 0x03f (focus 5, params [5]) = sent, want not sent\nalso skipped: message_sequence 0x090 bombing fleet not compared"},
		{"other count", seq(0, m(0x036, 7, 3, 7), q5, q6, lv, empt),
			"message 0x036 (focus 7, params [3 7]) = sent as 0x036 (focus 7, params [2 7]); message 0x036 (focus 7, params [2 7]) = sent, want not sent\nalso skipped: message_sequence 0x090 bombing fleet not compared"},
		{"other focus", seq(0, m(0x036, 6, 2, 7), q5, q6, lv, empt),
			"message 0x036 (focus 6, params [2 7]) = sent as 0x036 (focus 7, params [2 7]); message 0x036 (focus 7, params [2 7]) = sent, want not sent\nalso skipped: message_sequence 0x090 bombing fleet not compared"},
		{"other slot count", seq(0, m(0x036, 7, 2, 7, 0), q5, q6, lv, empt),
			"message 0x036 (focus 7, params [2 7 0]) = sent as 0x036 (focus 7, params [2 7]); message 0x036 (focus 7, params [2 7]) = sent, want not sent\nalso skipped: message_sequence 0x090 bombing fleet not compared"},
		{"repeat not sent", seq(0, b36, q5, q5, q6, lv, empt),
			"message 0x03f (focus 5, params [5]) = sent as 0x03f (focus 6, params [6])\nalso skipped: message_sequence 0x090 bombing fleet not compared"},
		{"not sent", seq(0, b36, m(0x02e, 7, 1), q5, q6, lv, empt),
			"message 0x02e (focus 7, params [1]) = not sent\nalso skipped: message_sequence 0x090 bombing fleet not compared"},
		{"extra, none a sample", seq(0, b36, q6, lv, empt),
			"message 0x03f (focus 5, params [5]) = sent, want not sent\nalso skipped: message_sequence 0x090 bombing fleet not compared"},
		{"extra allowed beside a sample", seq(0, b36, sample(m(0x0a1, 0, 1)), q6, lv, empt), "skip: message_sequence 0x090 bombing fleet not compared"},
		{"a sample is not compared", seq(0, b36, sample(m(0x03f, 9, 9)), q5, q6, lv, empt), "skip: message_sequence 0x090 bombing fleet not compared"},
		{"a sample's order is not compared", seq(0, sample(q6), b36, q5, lv, empt), "skip: message_sequence 0x090 bombing fleet not compared"},
		{"samples only", seq(0, sample(b36)), "skip: message_sequence of sample entries only"},
		{"unmapped entry", seq(0, m(0x0ff, 7, 7, 8), b36, q5, q6, lv, empt), "skip: message_sequence entry 0x0ff has no event mapping"},
		{"unmapped entry, mismatch", seq(0, m(0x0ff, 7, 7, 8), b36, q6, q5, lv, empt),
			"message 0x03f (focus 5, params [5]) = sent before 0x03f (focus 6, params [6]), want after; message 0x03f (focus 5, params [5]) = sent, want not sent\nalso skipped: message_sequence entry 0x0ff has no event mapping"},
		{"empty list, messages sent", seq(0),
			"message 0x036 (focus 7, params [2 7]) = sent, want not sent; message 0x03f (focus 5, params [5]) = sent, want not sent; message 0x03f (focus 6, params [6]) = sent, want not sent; message 0x050 (focus 65534, params [5 2 3]) = sent, want not sent; message 0x090 (focus 8, params [? 8]) = sent, want not sent"},
		{"queue not loaded", seq(2, m(0x036, 4, 3, 4), m(0x03f, 4, 4), m(0x026, 4, 4, 12, 0)), "skip: message_sequence entry from a production queue not loaded"},
		{"queue not loaded, other message missing", seq(2, m(0x036, 4, 3, 4), m(0x026, 4, 4, 11, 0)),
			"message 0x026 (focus 4, params [4 11 0]) = sent as 0x026 (focus 4, params [4 12 0]); message 0x026 (focus 4, params [4 12 0]) = sent, want not sent\nalso skipped: message_sequence entry from a production queue not loaded"},
		{"computer player keeps 0x08f only", seq(3, m(0x08f, 0x8000|3<<9|4, 3<<9|4, 8)), "skip: message_sequence 0x08f bombing fleet not compared"},
		{"computer player, empty list", seq(3),
			"message 0x08f (focus ?, params [? 8]) = sent, want not sent"},
		{"planned player", seq(1), "skip: " + pvPlannedWhy},
		{"year without events", pvExpect{Kind: "message_sequence", Year: 2, Player: i(0), Messages: []pvSeqEntry{}}, "skip: no events for the year"},
		{"malformed", pvExpect{Kind: "message_sequence", Year: 1, Player: i(0)}, "message_sequence: malformed expectation (player and messages are needed)"},
	} {
		if got := l.sequenceCheck(g, c.e); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
	// Through pvCheck, as a vector's expectation.
	if got := l.pvCheck(g, seq(0, whole...)); got != "skip: message_sequence 0x090 bombing fleet not compared" {
		t.Errorf("pvCheck: %q", got)
	}
}

// TestReadFileSequence reads a message_sequence entry's fields.
func TestReadFileSequence(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/kx001/a1.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range v.Cases {
		for _, e := range c.Expect {
			if e.Kind == "message_sequence" && e.Year == 1 && *e.Player == 0 {
				want := []pvSeqEntry{{ID: 0x08c, Focus: 7, Params: []int{7, 8}}, {ID: 0x036, Focus: 7, Params: []int{2, 7}}}
				if !reflect.DeepEqual(e.Messages, want) {
					t.Errorf("messages %+v, want %+v", e.Messages, want)
				}
				return
			}
		}
	}
	t.Error("no message_sequence for player 0 in year 1")
}

// TestSequenceCheckClaimAdjuster pins the drift rule for 0x156: its
// percent is compared unless the record's 0x15c drift on that planet is a
// sample or Elegy's run drew a drift there (MESSAGES.md 0x156, 0x15c and
// the P7 order). Such a 0x156 counts as a sample, so a sequence holding
// one allows extra messages, as any sample does.
func TestSequenceCheckClaimAdjuster(t *testing.T) {
	adjusted := func(planet, pct int) engine.Event {
		return engine.Event{Kind: engine.EventClaimAdjusted, Player: 0, Planet: planet, Fleet: -1, Count: pct}
	}
	drift := engine.Event{Kind: engine.EventOriginalDrift, Player: 0, Planet: 6, Fleet: -1, Axes: []int{1}}
	i := func(n int) *int { return &n }
	seq := func(ms ...pvSeqEntry) pvExpect {
		return pvExpect{Kind: "message_sequence", Year: 1, Player: i(0), Messages: ms}
	}
	m := func(id, focus int, params ...int) pvSeqEntry { return pvSeqEntry{ID: id, Focus: focus, Params: params} }
	sample := func(w pvSeqEntry) pvSeqEntry { w.Sample = true; return w }
	for _, c := range []struct {
		name   string
		events []engine.Event
		e      pvExpect
		want   string
	}{
		{"drift and percents match", []engine.Event{adjusted(5, 91), drift, adjusted(6, 92)},
			seq(m(0x156, 4, 4, 91), m(0x15c, 5, 5, 1), m(0x156, 5, 5, 92)), ""},
		{"percent differs", []engine.Event{adjusted(5, 91), adjusted(6, 91)},
			seq(m(0x156, 4, 4, 90), m(0x156, 5, 5, 91)),
			"message 0x156 (focus 4, params [4 90]) = sent as 0x156 (focus 4, params [4 91]); message 0x156 (focus 4, params [4 91]) = sent, want not sent"},
		{"drift axis differs", []engine.Event{adjusted(5, 91), drift, adjusted(6, 92)},
			seq(m(0x156, 4, 4, 91), m(0x15c, 5, 5, 2), m(0x156, 5, 5, 92)),
			"message 0x15c (focus 5, params [5 2]) = sent as 0x15c (focus 5, params [5 1])"},
		{"drift not sent", []engine.Event{adjusted(5, 91), adjusted(6, 91)},
			seq(m(0x156, 4, 4, 91), m(0x15c, 5, 5, 1), m(0x156, 5, 5, 92)),
			"message 0x15c (focus 5, params [5 1]) = not sent; message 0x156 (focus 5, params [5 92]) = sent as 0x156 (focus 5, params [5 91]); message 0x156 (focus 5, params [5 91]) = sent, want not sent"},
		{"the record's drift a sample", []engine.Event{adjusted(5, 91), adjusted(6, 91)},
			seq(m(0x156, 4, 4, 91), sample(m(0x15c, 5, 5, 1)), m(0x156, 5, 5, 92)), ""},
		{"Elegy drew a drift", []engine.Event{adjusted(5, 91), drift, adjusted(6, 92)},
			seq(m(0x156, 4, 4, 91), m(0x156, 5, 5, 91)), ""},
		{"another planet's drift", []engine.Event{adjusted(5, 92), drift, adjusted(6, 92)},
			seq(m(0x156, 4, 4, 91), m(0x156, 5, 5, 91)),
			"message 0x156 (focus 4, params [4 91]) = sent as 0x156 (focus 4, params [4 92])"},
	} {
		l := pvLoaded{g: engine.Game{Players: []engine.Player{{}}}, events: [][]engine.Event{nil, c.events}}
		if got := l.sequenceCheck(&l.g, c.e); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

// TestSequenceCheckScrapDesign pins 0x05b's fleet slot through the
// sequence comparator with a design other than 0 (stars-elegy FS-1 case d,
// player 1's fleet 2 of design 3: 1538 = 3·512 + 2), which no RG vector
// has: every RG scrap is design 0.
func TestSequenceCheckScrapDesign(t *testing.T) {
	l := pvLoaded{
		g:      engine.Game{Players: []engine.Player{{}, {}}},
		events: [][]engine.Event{nil, {{Kind: engine.EventFleetScrapped, Player: 1, Planet: -1, Fleet: 9, Axes: []int{1, 3, 3, 1}}}},
	}
	i := func(n int) *int { return &n }
	seq := func(params ...int) pvExpect {
		return pvExpect{Kind: "message_sequence", Year: 1, Player: i(1), Messages: []pvSeqEntry{{ID: 0x05b, Focus: 0xfffa, Params: params}}}
	}
	for _, c := range []struct {
		name   string
		params []int
		want   string
	}{
		{"design 3", []int{8705, 1538}, ""},
		{"owner bits instead", []int{8705, 512 + 2},
			"message 0x05b (focus 65530, params [8705 514]) = sent as 0x05b (focus 65530, params [8705 1538]); message 0x05b (focus 65530, params [8705 1538]) = sent, want not sent"},
		{"no design bits", []int{8705, 2},
			"message 0x05b (focus 65530, params [8705 2]) = sent as 0x05b (focus 65530, params [8705 1538]); message 0x05b (focus 65530, params [8705 1538]) = sent, want not sent"},
	} {
		if got := l.sequenceCheck(&l.g, seq(c.params...)); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

// TestSequenceCheckScrapAtPlanet pins 0x059's and 0x140's fleet slots
// through the sequence comparator (stars-elegy FS-1 case q: player 1's
// fleet 1/6 of design 4 at its own planet without a starbase gave 0x059
// 2054 = 4·512 + 6, then 0x140 518 = 1·512 + 6). No vector's sequence
// holds either. The kT and planet slots are not compared.
func TestSequenceCheckScrapAtPlanet(t *testing.T) {
	l := pvLoaded{
		g:      engine.Game{Players: []engine.Player{{}, {}}},
		events: [][]engine.Event{nil, {{Kind: engine.EventFleetScrapped, Player: 1, Planet: 3, Fleet: 9, Axes: []int{-1, 7, 4, 1, 0, 1, 0}}}},
	}
	i := func(n int) *int { return &n }
	seq := func(scrap, planet int) pvExpect {
		return pvExpect{Kind: "message_sequence", Year: 1, Player: i(1), Messages: []pvSeqEntry{
			{ID: 0x059, Focus: 2, Params: []int{scrap, 40, 2}}, {ID: 0x140, Focus: 2, Params: []int{planet, 40, 2}}}}
	}
	for _, c := range []struct {
		name          string
		scrap, planet int
		want          string
	}{
		{"FS-1 q", 2054, 518, "skip: message_sequence 0x059 kT and planet slots not compared"},
		{"0x059 with owner bits", 512 + 6, 518,
			"message 0x059 (focus 2, params [518 40 2]) = sent as 0x059 (focus 2, params [2054 ? ?]); message 0x059 (focus 2, params [2054 ? ?]) = sent, want not sent\nalso skipped: message_sequence 0x140 kT and planet slots not compared"},
		{"0x059 with no design bits", 6, 518,
			"message 0x059 (focus 2, params [6 40 2]) = sent as 0x059 (focus 2, params [2054 ? ?]); message 0x059 (focus 2, params [2054 ? ?]) = sent, want not sent\nalso skipped: message_sequence 0x140 kT and planet slots not compared"},
		{"0x140 with design bits", 2054, 2054,
			"message 0x140 (focus 2, params [2054 40 2]) = sent as 0x140 (focus 2, params [518 ? ?]); message 0x140 (focus 2, params [518 ? ?]) = sent, want not sent\nalso skipped: message_sequence 0x059 kT and planet slots not compared"},
		{"0x140 with no owner bits", 2054, 6,
			"message 0x140 (focus 2, params [6 40 2]) = sent as 0x140 (focus 2, params [518 ? ?]); message 0x140 (focus 2, params [518 ? ?]) = sent, want not sent\nalso skipped: message_sequence 0x059 kT and planet slots not compared"},
	} {
		if got := l.sequenceCheck(&l.g, seq(c.scrap, c.planet)); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

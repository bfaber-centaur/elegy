package vectors

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestPvMessages pins the event to message id mapping (MESSAGES.md).
func TestPvMessages(t *testing.T) {
	g := &engine.Game{Planets: []engine.Planet{{ID: 5, Owner: 2}, {ID: 6, Owner: engine.NoOwner}}, Players: make([]engine.Player, 3)}
	g.Players[1].Race.LRT.GeneralizedResearch = true
	part := func(name string) engine.Event {
		for i, c := range engine.Components().Components {
			if c.Name == name {
				return engine.Event{Kind: engine.EventPartAvailable, Player: 2, Planet: -1, Fleet: -1, Count: i}
			}
		}
		t.Fatalf("no component %q", name)
		return engine.Event{}
	}
	summary := func(player int, k engine.BattleSummary) engine.Event {
		return engine.Event{Kind: engine.EventBattleSummary, Player: player, Planet: -1, Fleet: -1, Count: int(k)}
	}
	built := func(planet int, item engine.ItemKind, n int) engine.Event {
		return engine.Event{Kind: engine.EventBuilt, Player: 2, Planet: planet, Fleet: -1, Item: item, Count: n}
	}
	for _, c := range []struct {
		name   string
		events []engine.Event
		want   [][2]int
	}{
		{"none", nil, nil},
		{"repelled: to the planet's owner, not the attacker", []engine.Event{{Kind: engine.EventDropRepelled, Player: 1, Planet: 5}}, [][2]int{{2, 0x004}}},
		{"repelled at an emptied planet", []engine.Event{{Kind: engine.EventDropRepelled, Player: 1, Planet: 6}}, nil},
		{"repelled at a planet not in the game", []engine.Event{{Kind: engine.EventDropRepelled, Player: 1, Planet: 7}}, nil},
		{"one factory", []engine.Event{built(5, engine.ItemFactory, 1)}, [][2]int{{2, 0x035}}},
		{"factories", []engine.Event{built(5, engine.ItemFactory, 3)}, [][2]int{{2, 0x036}}},
		{"one mine", []engine.Event{built(5, engine.ItemMine, 1)}, [][2]int{{2, 0x037}}},
		{"mines", []engine.Event{built(5, engine.ItemMine, 2)}, [][2]int{{2, 0x038}}},
		{"defenses not mapped", []engine.Event{built(5, engine.ItemDefenses, 2)}, nil},
		// MG-002: 1 then 1 gave one 0x036; 2 then 1 gave 0x036 then 0x035;
		// 1 then 4 gave one 0x036.
		{"1 then 1", []engine.Event{built(5, engine.ItemFactory, 1), built(5, engine.ItemFactory, 1)}, [][2]int{{2, 0x036}}},
		{"2 then 1", []engine.Event{built(5, engine.ItemFactory, 2), built(5, engine.ItemFactory, 1)}, [][2]int{{2, 0x035}, {2, 0x036}}},
		{"1 then 4", []engine.Event{built(5, engine.ItemFactory, 1), built(5, engine.ItemFactory, 4)}, [][2]int{{2, 0x036}}},
		{"singles on two planets", []engine.Event{built(5, engine.ItemFactory, 1), built(6, engine.ItemFactory, 1)}, [][2]int{{2, 0x035}}},
		{"a factory and a mine", []engine.Event{built(5, engine.ItemFactory, 1), built(5, engine.ItemMine, 1)}, [][2]int{{2, 0x035}, {2, 0x037}}},
		{"finished", []engine.Event{{Kind: engine.EventFleetArrived, Player: 3, Fleet: 9}}, [][2]int{{3, 0x04e}}},
		{"scrapped in deep space", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 3, Planet: -1}}, [][2]int{{3, 0x05b}}},
		{"scrapped at a planet", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 3, Planet: 5}}, nil},
		{"scrapped at planet 0", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 3, Planet: 0}}, nil},
		{"small comet", []engine.Event{{Kind: engine.EventCometStrike, Player: 1, Planet: 5, Count: 0}}, [][2]int{{1, 0x083}}},
		{"huge comet", []engine.Event{{Kind: engine.EventCometStrike, Player: 1, Planet: 5, Count: 3}}, [][2]int{{1, 0x086}}},
		{"bombed empty", []engine.Event{{Kind: engine.EventBombed, Player: 4, Planet: 6}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 6}}, [][2]int{{2, 0x090}, {4, 0x08f}}},
		{"emptied, not by bombing", []engine.Event{{Kind: engine.EventPlanetEmptied, Player: 2, Planet: 6}}, nil},
		{"emptied after another planet's bombing", []engine.Event{{Kind: engine.EventBombed, Player: 4, Planet: 5}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 6}}, nil},
		{"queue completed", []engine.Event{{Kind: engine.EventQueueCompleted, Player: 2, Planet: 5, Fleet: -1}}, [][2]int{{2, 0x03e}}},
		{"no queue", []engine.Event{{Kind: engine.EventNoQueue, Player: 2, Planet: 5, Fleet: -1}}, [][2]int{{2, 0x03f}}},
		{"alchemy", []engine.Event{{Kind: engine.EventAlchemy, Player: 2, Planet: 5, Fleet: -1, Item: engine.ItemMineralAlchemy, Count: 1}}, [][2]int{{2, 0x08c}}},
		{"factory order reduced", []engine.Event{{Kind: engine.EventOrderClipped, Player: 2, Planet: 5, Fleet: -1, Item: engine.ItemFactory, Count: 3}}, [][2]int{{2, 0x12a}}},
		{"defense order removed", []engine.Event{{Kind: engine.EventOrderClipped, Player: 2, Planet: 5, Fleet: -1, Item: engine.ItemDefenses, Count: 0}}, [][2]int{{2, 0x12a}}},
		{"terraform order reduced", []engine.Event{{Kind: engine.EventOrderClipped, Player: 2, Planet: 5, Fleet: -1, Item: engine.ItemTerraform, Count: 2}}, [][2]int{{2, 0x12f}}},
		{"overcrowded", []engine.Event{{Kind: engine.EventOvercrowded, Player: 2, Planet: 5, Fleet: -1, Count: 30}}, [][2]int{{2, 0x026}}},
		{"level up", []engine.Event{{Kind: engine.EventLevelUp, Player: 2, Planet: -1, Fleet: -1, Count: 3, Axes: []int{3}}}, [][2]int{{2, 0x050}}},
		{"level up, Generalized Research", []engine.Event{{Kind: engine.EventLevelUp, Player: 1, Planet: -1, Fleet: -1, Count: 3, Axes: []int{3}}}, [][2]int{{1, 0x136}}},
		{"research stolen", []engine.Event{{Kind: engine.EventResearchStolen, Player: 2, Planet: -1, Fleet: -1, Item: engine.ItemKind(3), Count: 40}}, [][2]int{{2, 0x159}}},
		{"level up, no player", []engine.Event{{Kind: engine.EventLevelUp, Player: -1, Planet: -1, Fleet: -1, Count: 3, Axes: []int{3}}}, nil},
		{"level up, player not in the game", []engine.Event{{Kind: engine.EventLevelUp, Player: 3, Planet: -1, Fleet: -1, Count: 3, Axes: []int{3}}}, nil},
		{"component", []engine.Event{part("Cow-hide Shield")}, [][2]int{{2, 0x05f}}},
		{"terraforming component", []engine.Event{part("Total Terraform ±5")}, [][2]int{{2, 0x05f}}},
		{"ship hull", []engine.Event{part("Medium Freighter")}, [][2]int{{2, 0x078}}},
		{"starbase hull (0x0d0, not mapped)", []engine.Event{part("Space Dock")}, nil},
		{"planetary scanner", []engine.Event{part("Viewer 90")}, [][2]int{{2, 0x157}}},
		{"planetary defense", []engine.Event{part("Laser Battery")}, [][2]int{{2, 0x145}}},
		{"Genesis Device", []engine.Event{part("Genesis Device")}, [][2]int{{2, 0x05f}}},
		{"component not in the catalog", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Count: -1}}, nil},
		{"component past the catalog", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Count: len(engine.Components().Components)}}, nil},
		{"no recipient", []engine.Event{{Kind: engine.EventFleetArrived, Player: -1}}, nil},
		{"duel won unhit", []engine.Event{summary(0, engine.SummaryDuelWonUnhit)}, [][2]int{{0, 0x091}}},
		{"both survived", []engine.Event{summary(1, engine.SummaryDuelBothSurvived)}, [][2]int{{1, 0x095}}},
		{"stacks both survived", []engine.Event{summary(1, engine.SummaryStackBothSurvived)}, [][2]int{{1, 0x09a}}},
		{"won with no loss", []engine.Event{summary(1, engine.SummaryWonNoLoss)}, [][2]int{{1, 0x09b}}},
		{"lone ship lost some kills", []engine.Event{summary(1, engine.SummaryLoneLostSomeKills)}, [][2]int{{1, 0x0a3}}},
		{"multi won no loss", []engine.Event{summary(2, engine.SummaryMultiWonNoLoss)}, [][2]int{{2, 0x0a4}}},
		{"multi one side no loss", []engine.Event{summary(2, engine.SummaryMultiOneSideNoLoss)}, [][2]int{{2, 0x0a8}}},
		{"multi mixed", []engine.Event{summary(2, engine.SummaryMultiMixed)}, [][2]int{{2, 0x07e}}},
		{"lone ship won all", []engine.Event{summary(0, engine.SummaryLoneWonAll)}, [][2]int{{0, 0x113}}},
		{"lone enemy survived", []engine.Event{summary(0, engine.SummaryLoneEnemySurvived)}, [][2]int{{0, 0x116}}},
		{"no summary", []engine.Event{summary(0, engine.SummaryNone)}, nil},
	} {
		got := pvMessages(g, c.events)
		want := map[[2]int]bool{}
		for _, m := range c.want {
			want[m] = true
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %v, want %v", c.name, got, want)
		}
	}
}

// TestMessageCheck pins the message expectation check.
func TestMessageCheck(t *testing.T) {
	comp := func(name string) int {
		for k, c := range engine.Components().Components {
			if c.Name == name {
				return k
			}
		}
		t.Fatalf("no component %q", name)
		return 0
	}
	l := pvLoaded{
		g:      engine.Game{Players: []engine.Player{{}, {Computer: true}, {}}},
		queued: map[int]bool{2: true},
		events: [][]engine.Event{nil, {{Kind: engine.EventFleetArrived, Player: 0}, {Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemMine, Count: 2}, {Kind: engine.EventAlchemy, Player: 0, Planet: 3, Item: engine.ItemMineralAlchemy, Count: 1}, {Kind: engine.EventOrderClipped, Player: 2, Planet: 5, Item: engine.ItemFactory, Count: 4}, {Kind: engine.EventOrderClipped, Player: 0, Planet: 3, Item: engine.ItemTerraform, Count: 0}, {Kind: engine.EventResearchStolen, Player: 0, Planet: -1, Fleet: -1, Item: engine.ItemKind(2), Count: 12}, {Kind: engine.EventLevelUp, Player: 0, Planet: -1, Fleet: -1, Item: engine.ItemKind(2), Count: 4, Axes: []int{3}}, {Kind: engine.EventLevelUp, Player: 2, Planet: -1, Fleet: -1, Item: engine.ItemKind(1), Count: 6, Axes: []int{3}},
			{Kind: engine.EventPartAvailable, Player: 0, Planet: -1, Fleet: -1, Count: comp("Destroyer")},
			{Kind: engine.EventPartAvailable, Player: 2, Planet: -1, Fleet: -1, Count: comp("Laser Battery")},
			{Kind: engine.EventPartAvailable, Player: 0, Planet: -1, Fleet: -1, Count: comp("Scoper 150")},
			{Kind: engine.EventTerraformed, Player: 2, Planet: 5, Fleet: -1, Item: engine.ItemTerraform, Count: 1, Axes: []int{0, 51}},
			{Kind: engine.EventHostileShrink, Player: 0, Planet: 3, Fleet: -1, Count: 486, Axes: []int{479}}}},
	}
	g := &engine.Game{Players: []engine.Player{{}, {}, {}}}
	g.Players[0].Race.LRT.GeneralizedResearch = true
	i := func(n int) *int { return &n }
	b := func(v bool) *bool { return &v }
	msg := func(player, id int, present bool, year int) pvExpect {
		return pvExpect{Kind: "message", Year: year, Player: i(player), MessageID: i(id), Present: b(present)}
	}
	for _, c := range []struct {
		name string
		e    pvExpect
		want string
	}{
		{"present", msg(0, 0x04e, true, 1), ""},
		{"absent as wanted", msg(0, 0x026, false, 1), ""},
		{"missing", msg(0, 0x026, true, 1), "message 0x026 to player 0 present = false, want true"},
		{"not wanted", msg(0, 0x04e, false, 1), "message 0x04e to player 0 present = true, want false"},
		{"another player's", msg(2, 0x04e, true, 1), "message 0x04e to player 2 present = false, want true"},
		{"unmapped id", msg(0, 0x0be, true, 1), "skip: message 0x0be has no event mapping"},
		{"alchemy", msg(0, 0x08c, true, 1), ""},
		{"alchemy, not sent", msg(2, 0x08c, true, 1), "message 0x08c to player 2 present = false, want true"},
		{"installation order reduced", msg(2, 0x12a, true, 1), ""},
		{"installation order reduced, not sent", msg(0, 0x12a, true, 1), "message 0x12a to player 0 present = false, want true"},
		{"hostile shrink", msg(0, 0x025, true, 1), ""},
		{"hostile shrink, not sent", msg(2, 0x025, true, 1), "message 0x025 to player 2 present = false, want true"},
		{"terraform click", msg(2, 0x07b, true, 1), ""},
		{"terraform click, not sent", msg(0, 0x07b, true, 1), "message 0x07b to player 0 present = false, want true"},
		{"terraform order reduced", msg(0, 0x12f, true, 1), ""},
		{"terraform order reduced, not sent", msg(2, 0x12f, true, 1), "message 0x12f to player 2 present = false, want true"},
		{"research stolen", msg(0, 0x159, true, 1), ""},
		{"research stolen, not sent", msg(2, 0x159, true, 1), "message 0x159 to player 2 present = false, want true"},
		{"level up, Generalized Research", msg(0, 0x136, true, 1), ""},
		{"level up, Generalized Research, not sent", msg(2, 0x136, true, 1), "message 0x136 to player 2 present = false, want true"},
		{"ship hull", msg(0, 0x078, true, 1), ""},
		{"ship hull, not sent", msg(2, 0x078, true, 1), "message 0x078 to player 2 present = false, want true"},
		{"planetary defense", msg(2, 0x145, true, 1), ""},
		{"planetary defense, not sent", msg(0, 0x145, true, 1), "message 0x145 to player 0 present = false, want true"},
		{"planetary scanner", msg(0, 0x157, true, 1), ""},
		{"planetary scanner, not sent", msg(2, 0x157, true, 1), "message 0x157 to player 2 present = false, want true"},
		{"no queue, not sent", msg(0, 0x03f, true, 1), "message 0x03f to player 0 present = false, want true"},
		{"queue not loaded, no queue", msg(2, 0x03f, true, 1), "skip: a queue message from a production queue not loaded"},
		{"queue not loaded, queue completed", msg(2, 0x03e, false, 1), "skip: a queue message from a production queue not loaded"},
		{"planned player", msg(1, 0x04e, true, 1), "skip: " + pvPlannedWhy},
		{"queue not loaded", msg(2, 0x038, true, 1), "skip: a build message from a production queue not loaded"},
		{"queue not loaded, one factory", msg(2, 0x035, true, 1), "skip: a build message from a production queue not loaded"},
		{"queue not loaded, other message", msg(2, 0x04e, true, 1), "message 0x04e to player 2 present = false, want true"},
		{"year without events", msg(0, 0x04e, true, 2), "skip: no events for the year"},
		{"malformed", pvExpect{Kind: "message", Year: 1, Player: i(0), MessageID: i(0x04e)}, "message: malformed expectation (player, message_id and present are needed)"},
	} {
		if got := l.messageCheck(g, c.e); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	// Through pvCheck, as a vector's expectation.
	if got := l.pvCheck(g, msg(0, 0x026, true, 1)); got != "message 0x026 to player 0 present = false, want true" {
		t.Errorf("pvCheck: %q", got)
	}
}

// Every battle summary has its own message id, the last one included.
func TestPvSummaryIDs(t *testing.T) {
	if len(pvSummaryIDs) != int(engine.SummaryLoneEnemySurvived)+1 {
		t.Fatalf("%d ids for %d summaries", len(pvSummaryIDs), int(engine.SummaryLoneEnemySurvived)+1)
	}
	seen := map[int]bool{}
	for k, id := range pvSummaryIDs[1:] {
		if seen[id] {
			t.Errorf("summary %d: id 0x%03x repeated", k+1, id)
		}
		seen[id] = true
		if _, ok := pvMessageIDs()[id]; !ok {
			t.Errorf("summary %d: id 0x%03x not compared", k+1, id)
		}
	}
}

// TestPvMessageSeq pins the order of a player's messages: as the year
// sends them, a removed message taken out of its place, each id once at
// its first place (MESSAGES.md "Order"; build_kx.py `messages`).
func TestPvMessageSeq(t *testing.T) {
	g := &engine.Game{Planets: []engine.Planet{{ID: 5, Owner: 2}}}
	built := func(planet int, item engine.ItemKind, n int) engine.Event {
		return engine.Event{Kind: engine.EventBuilt, Player: 2, Planet: planet, Fleet: -1, Item: item, Count: n}
	}
	arrived := func(fleet int) engine.Event {
		return engine.Event{Kind: engine.EventFleetArrived, Player: 2, Fleet: fleet}
	}
	scrapped := engine.Event{Kind: engine.EventFleetScrapped, Player: 2, Planet: -1}
	comet := engine.Event{Kind: engine.EventCometStrike, Player: 2, Planet: 5}
	for _, c := range []struct {
		name   string
		events []engine.Event
		want   []int
	}{
		{"as sent", []engine.Event{scrapped, arrived(1), comet}, []int{0x05b, 0x04e, 0x083}},
		{"each id once, at its first place", []engine.Event{arrived(1), scrapped, arrived(2)}, []int{0x04e, 0x05b}},
		{"a fleet's earlier 0x04e removed", []engine.Event{arrived(1), scrapped, arrived(1)}, []int{0x05b, 0x04e}},
		{"another fleet's 0x04e kept", []engine.Event{arrived(2), scrapped, arrived(1)}, []int{0x04e, 0x05b}},
		{"a merged single moves to the count", []engine.Event{built(5, engine.ItemFactory, 1), scrapped, built(5, engine.ItemFactory, 1)}, []int{0x05b, 0x036}},
		{"a single left in its place", []engine.Event{built(5, engine.ItemMine, 1), scrapped}, []int{0x037, 0x05b}},
		{"a plural count not moved", []engine.Event{built(5, engine.ItemMine, 2), scrapped, built(5, engine.ItemMine, 3)}, []int{0x038, 0x05b}},
		{"no recipient", []engine.Event{{Kind: engine.EventFleetArrived, Player: -1, Fleet: 1}, scrapped}, []int{0x05b}},
	} {
		if got := pvMessageSeq(g, c.events)[2]; !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %#x, want %#x", c.name, got, c.want)
		}
	}
	// A no-recipient single is never placed, so a later single starts a
	// new count.
	if got := pvMessageSeq(g, []engine.Event{{Kind: engine.EventBuilt, Player: -1, Planet: 5, Item: engine.ItemFactory, Count: 1}, built(5, engine.ItemFactory, 1)}); !reflect.DeepEqual(got, map[int][]int{2: {0x035}}) {
		t.Errorf("no-recipient single: %#x", got)
	}
}

// TestMessageOrder pins the order check: each present message against the
// nearest one listed before it that Elegy sends, samples kept apart.
func TestMessageOrder(t *testing.T) {
	i := func(n int) *int { return &n }
	b := func(v bool) *bool { return &v }
	msg := func(year, player, id int, present, sample bool) pvExpect {
		return pvExpect{Kind: "message", Year: year, Player: i(player), MessageID: i(id), Present: b(present), Sample: sample}
	}
	cases := []pvCase{{Expect: []pvExpect{
		msg(1, 0, 0x05b, true, false), // 0
		msg(1, 1, 0x083, true, false), // 1: another player
		msg(2, 0, 0x083, true, false), // 2: another year
		msg(1, 0, 0x026, false, false),
		{Kind: "planet", Year: 1},
		msg(1, 0, 0x04e, true, false), // 5
		msg(1, 0, 0x083, true, true),  // 6: a sample
		msg(1, 0, 0x036, true, false), // 7
	}}, {Expect: []pvExpect{msg(1, 0, 0x036, true, false)}}}
	pvMessageOrder(cases)
	ids := func(e pvExpect) []pvMessagePrior { return e.before }
	for _, c := range []struct {
		at   int
		want []pvMessagePrior
	}{
		{0, nil}, {1, nil}, {2, nil}, {3, nil},
		{5, []pvMessagePrior{{0x05b, false}}},
		{6, []pvMessagePrior{{0x05b, false}, {0x04e, false}}},
		{7, []pvMessagePrior{{0x05b, false}, {0x04e, false}, {0x083, true}}},
	} {
		if got := ids(cases[0].Expect[c.at]); !reflect.DeepEqual(got, c.want) {
			t.Errorf("before of %d: %v, want %v", c.at, got, c.want)
		}
	}
	if got := cases[1].Expect[0].before; got != nil {
		t.Errorf("another case's: %v", got)
	}
	e := cases[0].Expect
	for _, c := range []struct {
		name string
		e    pvExpect
		seq  []int
		want string
	}{
		{"first listed", e[0], []int{0x04e, 0x05b}, ""},
		{"in order", e[5], []int{0x05b, 0x04e}, ""},
		{"out of order", e[5], []int{0x04e, 0x05b}, "message 0x04e to player 0 comes before 0x05b, want after"},
		{"the nearest one decides", e[7], []int{0x05b, 0x04e, 0x036}, ""},
		{"the nearest one out of order", e[7], []int{0x05b, 0x036, 0x04e}, "message 0x036 to player 0 comes before 0x04e, want after"},
		{"one Elegy does not send is passed over", e[7], []int{0x036, 0x05b}, "message 0x036 to player 0 comes before 0x05b, want after"},
		{"none sent before", e[7], []int{0x036}, ""},
		{"only the nearest one sent decides", e[7], []int{0x04e, 0x036, 0x05b}, ""},
		{"a sample is passed over by an exact one", e[7], []int{0x05b, 0x04e, 0x036, 0x083}, ""},
		{"a sample against the nearest", e[6], []int{0x05b, 0x083, 0x04e}, "message 0x083 to player 0 comes before 0x04e, want after"},
	} {
		if got := messageOrderCheck(c.e, c.seq); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	// A repeated id is placed against the one before it, not itself.
	twice := []pvCase{{Expect: []pvExpect{msg(1, 0, 0x05b, true, false), msg(1, 0, 0x04e, true, false), msg(1, 0, 0x04e, true, false)}}}
	pvMessageOrder(twice)
	if got := messageOrderCheck(twice[0].Expect[2], []int{0x05b, 0x04e}); got != "" {
		t.Errorf("repeated id: %q", got)
	}
	if got := messageOrderCheck(twice[0].Expect[2], []int{0x04e, 0x05b}); got != "message 0x04e to player 0 comes before 0x05b, want after" {
		t.Errorf("repeated id, out of order: %q", got)
	}
	// A sample is placed against an earlier sample.
	samples := []pvCase{{Expect: []pvExpect{msg(1, 0, 0x05b, true, true), msg(1, 0, 0x04e, true, true)}}}
	pvMessageOrder(samples)
	if got := messageOrderCheck(samples[0].Expect[1], []int{0x04e, 0x05b}); got != "message 0x04e to player 0 comes before 0x05b, want after" {
		t.Errorf("sample after a sample: %q", got)
	}
	// Through messageCheck: a present message out of order fails.
	l := pvLoaded{events: [][]engine.Event{nil, {{Kind: engine.EventFleetArrived, Player: 0, Fleet: 1}, {Kind: engine.EventFleetScrapped, Player: 0, Planet: -1}}}}
	if got := l.messageCheck(&engine.Game{}, e[5]); got != "message 0x04e to player 0 comes before 0x05b, want after" {
		t.Errorf("messageCheck: %q", got)
	}
	if got := l.messageCheck(&engine.Game{}, e[0]); got != "" {
		t.Errorf("messageCheck, first listed: %q", got)
	}
}

// TestReadFileMessageOrder pins that ReadFile records the message order.
func TestReadFileMessageOrder(t *testing.T) {
	v, err := ReadFile("../../engine/testdata/vectors/rg/rg002.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range v.Cases {
		for _, e := range c.Expect {
			if len(e.before) > 0 {
				return
			}
		}
	}
	t.Error("no message expectation has the ones listed before it")
}

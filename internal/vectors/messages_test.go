package vectors

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestPvMessages pins the event to message id mapping (MESSAGES.md).
func TestPvMessages(t *testing.T) {
	g := &engine.Game{Planets: []engine.Planet{{ID: 5, Owner: 2}, {ID: 6, Owner: engine.NoOwner}}}
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
		{"no recipient", []engine.Event{{Kind: engine.EventFleetArrived, Player: -1}}, nil},
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
	l := pvLoaded{
		g:      engine.Game{Players: []engine.Player{{}, {Computer: true}, {}}},
		queued: map[int]bool{2: true},
		events: [][]engine.Event{nil, {{Kind: engine.EventFleetArrived, Player: 0}, {Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemMine, Count: 2}}},
	}
	g := &engine.Game{}
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
		{"unmapped id", msg(0, 0x03f, true, 1), "skip: message 0x03f has no event mapping"},
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

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
	part := -1
	for i, c := range engine.Components().Components {
		if c.Name == "Cow-hide Shield" {
			part = i
		}
	}
	all := func(n int) []bool {
		k := make([]bool, n)
		for i := range k {
			k[i] = true
		}
		return k
	}
	for _, c := range []struct {
		name   string
		events []engine.Event
		player int
		want   []pvMsg
	}{
		{"one factory", []engine.Event{{Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemFactory, Count: 1}}, 2,
			[]pvMsg{{id: 0x035, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"merged count", []engine.Event{{Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemMine, Count: 1}, {Kind: engine.EventBuilt, Player: 2, Planet: 5, Item: engine.ItemMine, Count: 3}}, 2,
			[]pvMsg{{id: 0x038, focus: 4, focusKnown: true, params: []int{4, 4}, known: all(2)}}},
		{"overcrowded, long amount", []engine.Event{{Kind: engine.EventOvercrowded, Player: 2, Planet: 5, Count: 70000}}, 2,
			[]pvMsg{{id: 0x026, focus: 4, focusKnown: true, params: []int{4, 70000 - 65536, 1}, known: all(3)}}},
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
		{"level up", []engine.Event{{Kind: engine.EventLevelUp, Player: 2, Item: 4, Count: 7}}, 2,
			[]pvMsg{{id: 0x050, focus: 0xfffe, focusKnown: true, params: []int{7, 4, 0}, known: []bool{true, true, false}, why: "0x050 slot 3 (field researched next)"}}},
		{"level up, Generalized Research", []engine.Event{{Kind: engine.EventLevelUp, Player: 1, Item: 4, Count: 7}}, 1,
			[]pvMsg{{id: 0x136, focus: 0xfffe, focusKnown: true, params: []int{7, 4, 0}, known: []bool{true, true, false}, why: "0x136 slot 3 (field researched next)"}}},
		{"research stolen", []engine.Event{{Kind: engine.EventResearchStolen, Player: 2, Planet: -1, Fleet: -1, Item: 3, Count: 116}}, 2,
			[]pvMsg{{id: 0x159, focus: 0xfffe, focusKnown: true, params: []int{3, 116}, known: all(2)}}},
		{"component", []engine.Event{{Kind: engine.EventPartAvailable, Player: 2, Item: 1, Count: part}}, 2,
			[]pvMsg{{id: 0x05f, params: []int{1, 0, 0}, known: []bool{true, false, false}, why: "0x05f part focus and slots (no category-mask table)"}}},
		{"scrapped in deep space", []engine.Event{{Kind: engine.EventFleetScrapped, Player: 1, Planet: -1, Fleet: 9}}, 1,
			[]pvMsg{{id: 0x05b, focus: 0xfffa, focusKnown: true, params: []int{0, 0}, known: []bool{false, false}, why: "0x05b salvage and fleet slots"}}},
		{"repelled", []engine.Event{{Kind: engine.EventDropRepelled, Player: 1, Planet: 5, Count: 3}}, 2,
			[]pvMsg{{id: 0x004, focus: 4, focusKnown: true, params: []int{4, 0, 0, 0}, known: []bool{true, false, false, false}, why: "0x004 colonists and player slots"}}},
		{"bombed empty, to the bomber", []engine.Event{{Kind: engine.EventBombed, Player: 1, Planet: 5}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 5}}, 1,
			[]pvMsg{{id: 0x08f, params: []int{0, 4}, known: []bool{false, true}, why: "0x08f and 0x090 bombing fleet"}}},
		{"bombed empty, to the owner", []engine.Event{{Kind: engine.EventBombed, Player: 1, Planet: 5}, {Kind: engine.EventPlanetEmptied, Player: 2, Planet: 5}}, 2,
			[]pvMsg{{id: 0x090, focus: 4, focusKnown: true, params: []int{0, 4}, known: []bool{false, true}, why: "0x08f and 0x090 bombing fleet"}}},
		{"battle summary", []engine.Event{{Kind: engine.EventBattleSummary, Player: 0, Count: int(engine.SummaryDuelWonUnhit)}}, 0,
			[]pvMsg{{id: 0x091, known: []bool{}, why: "battle summary focus and slots"}}},
		{"alchemy", []engine.Event{{Kind: engine.EventAlchemy, Player: 2, Planet: 5, Item: engine.ItemMineralAlchemy, Count: 8}}, 2,
			[]pvMsg{{id: 0x08c, focus: 4, focusKnown: true, params: []int{4, 8}, known: all(2)}}},
		{"installation order cut", []engine.Event{{Kind: engine.EventOrderClipped, Player: 2, Planet: 5, Item: engine.ItemMine, Count: 3}}, 2,
			[]pvMsg{{id: 0x12a, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"terraform order cut", []engine.Event{{Kind: engine.EventOrderClipped, Player: 2, Planet: 5, Item: engine.ItemTerraform, Count: 0}}, 2,
			[]pvMsg{{id: 0x12f, focus: 4, focusKnown: true, params: []int{4}, known: all(1)}}},
		{"colonists beamed down", []engine.Event{{Kind: engine.EventColonistsBeamedDown, Player: 1, Planet: 5, Fleet: 9, Count: 70000}}, 1,
			[]pvMsg{{id: 0x02e, focus: 0x8000 | 1<<9 | 3, focusKnown: true, params: []int{1<<9 | 3, 70000 - 65536, 1, 0, 0, 0},
				known: []bool{true, true, true, false, false, false}, why: "0x02e mineral and location slots"}}},
		{"colonists beamed down, fleet gone", []engine.Event{{Kind: engine.EventColonistsBeamedDown, Player: 1, Planet: 5, Fleet: 8, Count: 50}}, 1,
			[]pvMsg{{id: 0x02e, params: make([]int, 6), known: make([]bool, 6), why: "0x02e fleet gone by the end of the year"}}},
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
			{Kind: engine.EventLevelUp, Player: 0, Item: 2, Count: 5},
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
		{"whole sequence", seq(0, whole...), "skip: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"slots compared before the unknown one", seq(0, b36, q5, q6, m(0x050, 0xfffe, 5, 1, 3), empt),
			"message 0x050 (focus 65534, params [5 1 3]) = sent as 0x050 (focus 65534, params [5 2 ?]); message 0x050 (focus 65534, params [5 2 ?]) = sent, want not sent\nalso skipped: message_sequence 0x08f and 0x090 bombing fleet not compared"},
		{"out of order", seq(0, b36, q6, q5, lv, empt),
			"message 0x03f (focus 5, params [5]) = sent before 0x03f (focus 6, params [6]), want after; message 0x03f (focus 5, params [5]) = sent, want not sent\nalso skipped: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"other count", seq(0, m(0x036, 7, 3, 7), q5, q6, lv, empt),
			"message 0x036 (focus 7, params [3 7]) = sent as 0x036 (focus 7, params [2 7]); message 0x036 (focus 7, params [2 7]) = sent, want not sent\nalso skipped: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"other focus", seq(0, m(0x036, 6, 2, 7), q5, q6, lv, empt),
			"message 0x036 (focus 6, params [2 7]) = sent as 0x036 (focus 7, params [2 7]); message 0x036 (focus 7, params [2 7]) = sent, want not sent\nalso skipped: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"other slot count", seq(0, m(0x036, 7, 2, 7, 0), q5, q6, lv, empt),
			"message 0x036 (focus 7, params [2 7 0]) = sent as 0x036 (focus 7, params [2 7]); message 0x036 (focus 7, params [2 7]) = sent, want not sent\nalso skipped: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"repeat not sent", seq(0, b36, q5, q5, q6, lv, empt),
			"message 0x03f (focus 5, params [5]) = sent as 0x03f (focus 6, params [6])\nalso skipped: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"not sent", seq(0, b36, m(0x02e, 7, 1), q5, q6, lv, empt),
			"message 0x02e (focus 7, params [1]) = not sent\nalso skipped: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"extra, none a sample", seq(0, b36, q6, lv, empt),
			"message 0x03f (focus 5, params [5]) = sent, want not sent\nalso skipped: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"extra allowed beside a sample", seq(0, b36, sample(m(0x0a1, 0, 1)), q6, lv, empt), "skip: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"a sample is not compared", seq(0, b36, sample(m(0x03f, 9, 9)), q5, q6, lv, empt), "skip: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"a sample's order is not compared", seq(0, sample(q6), b36, q5, lv, empt), "skip: message_sequence 0x050 slot 3 (field researched next) not compared"},
		{"samples only", seq(0, sample(b36)), "skip: message_sequence of sample entries only"},
		{"unmapped entry", seq(0, m(0x07b, 7, 7, 8), b36, q5, q6, lv, empt), "skip: message_sequence entry 0x07b has no event mapping"},
		{"unmapped entry, mismatch", seq(0, m(0x07b, 7, 7, 8), b36, q6, q5, lv, empt),
			"message 0x03f (focus 5, params [5]) = sent before 0x03f (focus 6, params [6]), want after; message 0x03f (focus 5, params [5]) = sent, want not sent\nalso skipped: message_sequence entry 0x07b has no event mapping"},
		{"empty list, messages sent", seq(0),
			"message 0x036 (focus 7, params [2 7]) = sent, want not sent; message 0x03f (focus 5, params [5]) = sent, want not sent; message 0x03f (focus 6, params [6]) = sent, want not sent; message 0x050 (focus 65534, params [5 2 ?]) = sent, want not sent; message 0x090 (focus 8, params [? 8]) = sent, want not sent"},
		{"queue not loaded", seq(2, m(0x036, 4, 3, 4), m(0x03f, 4, 4), m(0x026, 4, 4, 12, 0)), "skip: message_sequence entry from a production queue not loaded"},
		{"queue not loaded, other message missing", seq(2, m(0x036, 4, 3, 4), m(0x026, 4, 4, 11, 0)),
			"message 0x026 (focus 4, params [4 11 0]) = sent as 0x026 (focus 4, params [4 12 0]); message 0x026 (focus 4, params [4 12 0]) = sent, want not sent\nalso skipped: message_sequence entry from a production queue not loaded"},
		{"computer player keeps 0x08f only", seq(3, m(0x08f, 0x8000|3<<9|4, 3<<9|4, 8)), "skip: message_sequence 0x08f and 0x090 bombing fleet not compared"},
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
	if got := l.pvCheck(g, seq(0, whole...)); got != "skip: message_sequence 0x050 slot 3 (field researched next) not compared" {
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

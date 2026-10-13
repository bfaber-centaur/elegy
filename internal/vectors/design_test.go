package vectors

import (
	"encoding/json"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestDesignEquals pins the design and starbase_design check: hull, mass
// and slots by hull slot, null for an empty one.
func TestDesignEquals(t *testing.T) {
	scout := engine.Design{Hull: engine.Hull{Name: "Scout"}, Mass: 19,
		Slots:   []engine.Slot{{Part: engine.Part{Name: "Bat Scanner"}, Count: 1}, {Part: engine.Part{Name: "Long Hump 6"}, Count: 1}},
		SlotPos: []int{1, 0}}
	station := engine.Design{Hull: engine.Hull{Name: "Space Station"}, Mass: 630}
	g := &engine.Game{
		Designs:     []engine.Design{scout, station, {Hull: engine.Hull{Name: "Scout"}, Slots: []engine.Slot{{Count: 1}}}},
		DesignSlots: []engine.DesignSlot{{Owner: 0, Slot: 8, Design: 0}, {Owner: 0, Slot: 8, Starbase: true, Design: 1}, {Owner: 1, Slot: 2, Design: 2}},
	}
	const want = `"hull": "Scout", "slots": [{"count": 1, "part": "Long Hump 6"}, {"count": 1, "part": "Bat Scanner"}, null], "mass": 19`
	for _, c := range []struct {
		name           string
		owner, slot    int
		starbase       bool
		eq             string
		mismatch, skip string
	}{
		{"match", 0, 8, false, `{` + want + `}`, "", ""},
		{"starbase slot of the same number", 0, 8, true, `{"hull": "Space Station", "mass": 630}`, "", ""},
		{"ship design is not the starbase one", 0, 8, true, `{"hull": "Scout"}`, "hull = Space Station, want Scout", ""},
		{"hull", 0, 8, false, `{"hull": "Frigate"}`, "hull = Scout, want Frigate", ""},
		{"mass", 0, 8, false, `{"mass": 20}`, "mass = 19, want 20", ""},
		{"empty slot filled", 0, 8, false, `{"slots": [{"count": 1, "part": "Long Hump 6"}, null, null]}`, "slots = [1 Long Hump 6, 1 Bat Scanner, -], want [1 Long Hump 6, -, -]", ""},
		{"count", 0, 8, false, `{"slots": [{"count": 2, "part": "Long Hump 6"}, {"count": 1, "part": "Bat Scanner"}]}`, "slots = [1 Long Hump 6, 1 Bat Scanner], want [2 Long Hump 6, 1 Bat Scanner]", ""},
		{"a part past the expected slots", 0, 8, false, `{"slots": [{"count": 1, "part": "Long Hump 6"}]}`, "slots = [1 Long Hump 6, 1 Bat Scanner], want [1 Long Hump 6]", ""},
		{"no positions", 1, 2, false, `{"slots": [null]}`, "", "design slots without hull-slot positions"},
		{"another player's slot of the same number", 1, 8, false, `{"hull": "Scout"}`, "no design in slot 8 of player 1", ""},
		{"no such design", 1, 3, false, `{"hull": "Scout"}`, "no design in slot 3 of player 1", ""},
		{"other field", 0, 8, false, `{"armor": 5}`, "", "design field armor"},
		{"malformed hull", 0, 8, false, `{"hull": 3}`, "hull: malformed expectation 3", ""},
		{"malformed mass", 0, 8, false, `{"mass": null}`, "mass: malformed expectation null", ""},
		{"malformed slots", 0, 8, false, `{"slots": {}}`, "slots: malformed expectation {}", ""},
	} {
		pvWant(t, c.name, designEquals(g, c.owner, c.slot, c.starbase, pvEq(t, c.eq)), c.mismatch, c.skip)
	}
}

// TestPlayerCounts pins the player's design counts.
func TestPlayerCounts(t *testing.T) {
	var l pvLoaded
	g := &engine.Game{Players: []engine.Player{{}, {}}, DesignSlots: []engine.DesignSlot{{Owner: 0}, {Owner: 0}, {Owner: 0, Starbase: true}, {Owner: 1}}}
	pvWant(t, "match", l.playerEquals(g, 0, pvEq(t, `{"counts": {"ship_designs": 2, "starbase_designs": 1}}`), nil), "", "")
	pvWant(t, "ships", l.playerEquals(g, 0, pvEq(t, `{"counts": {"ship_designs": 3}}`), nil), "counts ship_designs = 2, want 3", "")
	pvWant(t, "fewer ships", l.playerEquals(g, 0, pvEq(t, `{"counts": {"ship_designs": 1}}`), nil), "counts ship_designs = 2, want 1", "")
	pvWant(t, "starbases", l.playerEquals(g, 1, pvEq(t, `{"counts": {"starbase_designs": 1}}`), nil), "counts starbase_designs = 0, want 1", "")
	pvWant(t, "other count", l.playerEquals(g, 0, pvEq(t, `{"counts": {"fleets": 1}}`), nil), "", "player counts fleets")
	pvWant(t, "malformed", l.playerEquals(g, 0, pvEq(t, `{"counts": 3}`), nil), "counts: malformed expectation 3", "")
	pvWant(t, "empty", l.playerEquals(g, 0, pvEq(t, `{"counts": {}}`), nil), "counts: malformed expectation {}", "")
}

// TestDesignPlanned pins that a planned player's design expectations are
// skipped through pvCheck, and a played player's are compared.
func TestDesignPlanned(t *testing.T) {
	l := pvLoaded{g: engine.Game{Players: []engine.Player{{}, {Computer: true}}}}
	g := &engine.Game{
		Designs:     []engine.Design{{Hull: engine.Hull{Name: "Scout"}}},
		DesignSlots: []engine.DesignSlot{{Owner: 0, Slot: 2, Design: 0}, {Owner: 1, Slot: 2, Design: 0}},
	}
	i := func(n int) *int { return &n }
	for _, c := range []struct {
		kind  string
		owner int
		want  string
	}{
		{"design", 1, "skip: " + pvPlannedWhy},
		{"starbase_design", 1, "skip: " + pvPlannedWhy},
		{"design", 0, "hull = Scout, want Frigate"},
	} {
		e := pvExpect{Kind: c.kind, Owner: i(c.owner), Slot: i(2), Equals: json.RawMessage(`{"hull": "Frigate"}`)}
		if got := l.pvCheck(g, e); got != c.want {
			t.Errorf("%s of player %d: %q, want %q", c.kind, c.owner, got, c.want)
		}
	}
}

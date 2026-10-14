package vectors

import (
	"encoding/json"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

// TestTraderVectorIDs pins that a Trader is compared by its vector id
// when the ids are not its dense index (RG-005 y1: slots 2 and 3 only).
// Before, ids 2 and 3 read as missing although both Traders were loaded.
func TestTraderVectorIDs(t *testing.T) {
	raw := []json.RawMessage{
		json.RawMessage(`{"kind": "trader", "id": 3, "x": 1778, "y": 1492, "destination": [2020, 1020], "warp": 12, "met": [], "offer": "Alien Miner"}`),
		json.RawMessage(`{"kind": "trader", "id": 2, "x": 1428, "y": 1932, "destination": [1332, 2180], "warp": 9, "met": [1], "offer": "research"}`),
	}
	ids := pvTraderIDs(raw)
	m := pvMaps{Traders: ids}
	obj, err := pvObjects{}.Load(&engine.Game{}, raw, m)
	if err != nil {
		t.Fatal(err)
	}
	g := &engine.Game{Objects: obj}
	object := func(eq string) pvObjExpect { return pvObjExpect{Kind: "object", Equals: json.RawMessage(eq)} }
	for _, c := range []struct{ eq, want string }{
		{`{"kind": "trader", "id": 2, "x": 1428, "y": 1932, "met": [1], "offer": "research"}`, ""},
		{`{"kind": "trader", "id": 3, "x": 1778, "y": 1492, "met": [], "offer": "Alien Miner"}`, ""},
		{`{"kind": "trader", "id": 3, "x": 1428}`, "trader: x = 1778, want 1428"},
		{`{"kind": "trader", "id": 4, "x": 1}`, "trader 4 missing"},
	} {
		if got := (pvObjects{}).Check(g, object(c.eq), m); got != c.want {
			t.Errorf("%s: %q, want %q", c.eq, got, c.want)
		}
	}
	id := 3
	if got := (pvObjects{}).Check(g, pvObjExpect{Kind: "trader", ID: &id, Equals: json.RawMessage(`{"warp": 12}`)}, m); got != "" {
		t.Errorf("trader form: %q", got)
	}
	gone := pvObjExpect{Kind: "object_gone", Subject: json.RawMessage(`{"kind": "trader", "id": 3}`)}
	if got := (pvObjects{}).Check(g, gone, m); got != "trader still exists" {
		t.Errorf("object_gone of a loaded Trader: %q", got)
	}
	// have is how many Traders Elegy holds after the run.
	left := "skip: a Trader left during the run; Elegy's dense Trader list does not keep the original's slots"
	appeared := "skip: a Trader appeared during the run; Elegy's dense Trader list does not keep the original's slots"
	for _, c := range []struct {
		loaded             []int
		appeared, have, id int
		want               int
		why                string
	}{
		{[]int{2, 3}, 0, 2, 2, 0, ""}, {[]int{2, 3}, 0, 2, 3, 1, ""},
		{[]int{2, 3}, 0, 2, 4, -1, ""}, {[]int{2, 3}, 0, 2, 0, -1, ""},
		// RG-005 y4: both loaded Traders left and one appeared in slot 0.
		{[]int{2, 3}, 1, 1, 0, -1, left}, {[]int{2, 3}, 1, 1, 2, -1, left},
		{[]int{2, 3}, 1, 3, 3, 1, ""}, {[]int{2, 3}, 1, 3, 0, -1, appeared},
		{[]int{2, 3}, 0, 1, 3, -1, left}, {[]int{2, 3}, 0, 0, 3, -1, ""},
		{[]int{2, 3}, 1, 0, 0, -1, ""},
		{nil, 0, 0, 1, -1, ""}, {[]int{0, 1}, 0, 2, 1, 1, ""}, {[]int{0, 1}, 0, 2, 2, -1, ""},
	} {
		got, why := pvTraderIndex(pvMaps{Traders: c.loaded, TradersAppeared: c.appeared}, c.have, c.id)
		if got != c.want || why != c.why {
			t.Errorf("pvTraderIndex(%v, appeared %d, have %d, %d) = %d %q, want %d %q", c.loaded, c.appeared, c.have, c.id, got, why, c.want, c.why)
		}
	}
	// A Trader that left makes a loaded id's comparison a skip, never a
	// match or a miss by the wrong Trader.
	m.TradersAppeared = 0
	g.Objects.(*objects.Space).Traders = g.Objects.(*objects.Space).Traders[:1]
	if got := (pvObjects{}).Check(g, gone, m); got != left {
		t.Errorf("object_gone after a departure: %q", got)
	}
}

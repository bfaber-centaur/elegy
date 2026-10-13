package vectors

import (
	"encoding/json"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

// TestObjectWormhole pins the whole-object form of a wormhole expectation
// (README.md "object"), and that the wormhole form reads the same fields.
func TestObjectWormhole(t *testing.T) {
	s := &objects.Space{Wormholes: []objects.Wormhole{{Ends: [2]objects.WormholeEnd{
		{Pos: engine.Point{X: 1100, Y: 1200}, Class: 1, Years: 3, Known: []bool{false, true}, DestKnown: []bool{true}},
		{Pos: engine.Point{X: 1300, Y: 1400}, Class: 2, Years: 3},
	}}}}
	g := &engine.Game{Objects: s}
	// Vector ends 7 and 9 are the wormhole's ends 1 and 0.
	m := pvMaps{EndID: map[int]int{9: 0, 7: 1}}
	end9 := `{"kind": "wormhole", "id": 9, "x": 1100, "y": 1200, "partner": 7, "stability_class": 1, "years_since_jump": 3, "known_to": [1], "destination_known_to": [0]}`
	object := func(eq string) pvObjExpect { return pvObjExpect{Kind: "object", Equals: json.RawMessage(eq)} }
	for _, c := range []struct {
		name, eq, want string
	}{
		{"match", end9, ""},
		{"other end", `{"kind": "wormhole", "id": 7, "x": 1300, "y": 1400, "partner": 9, "stability_class": 2, "years_since_jump": 3, "known_to": [], "destination_known_to": []}`, ""},
		{"position", `{"kind": "wormhole", "id": 9, "x": 1101, "y": 1200}`, "wormhole: x = 1100, want 1101"},
		{"partner", `{"kind": "wormhole", "id": 9, "partner": 8}`, "wormhole: partner = 7, want 8"},
		{"id", `{"kind": "wormhole", "id": 7, "x": 1100}`, "wormhole: x = 1300, want 1100"},
		{"class", `{"kind": "wormhole", "id": 7, "stability_class": 1}`, "wormhole: stability_class = 2, want 1"},
		{"years", `{"kind": "wormhole", "id": 9, "years_since_jump": 0}`, "wormhole: years_since_jump = 3, want 0"},
		{"known", `{"kind": "wormhole", "id": 9, "known_to": [0, 1]}`, "wormhole: known_to = [1], want [0 1]"},
		{"destination known", `{"kind": "wormhole", "id": 9, "destination_known_to": []}`, "wormhole: destination_known_to = [0], want []"},
		{"missing end", `{"kind": "wormhole", "id": 8, "x": 1}`, "wormhole end 8 missing"},
		{"unknown field", `{"kind": "wormhole", "id": 9, "mass": 1}`, "skip: wormhole field mass"},
	} {
		if got := (pvObjects{}).Check(g, object(c.eq), m); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	id := 9
	if got := (pvObjects{}).Check(g, pvObjExpect{Kind: "wormhole", ID: &id, Equals: json.RawMessage(end9)}, m); got != "" {
		t.Errorf("wormhole form: %q", got)
	}
	id = 8
	if got := (pvObjects{}).Check(g, pvObjExpect{Kind: "wormhole", ID: &id, Equals: json.RawMessage(`{"x": 1}`)}, m); got != "wormhole end 8 missing" {
		t.Errorf("wormhole form, missing end: %q", got)
	}
	// An end id past Elegy's wormholes is missing too.
	if got := (pvObjects{}).Check(g, object(`{"kind": "wormhole", "id": 3, "x": 1}`), pvMaps{EndID: map[int]int{3: 2}}); got != "wormhole end 3 missing" {
		t.Errorf("end past the wormholes: %q", got)
	}
}

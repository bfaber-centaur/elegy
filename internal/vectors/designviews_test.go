package vectors

import (
	"encoding/json"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// The sc corpus has two players, one starbase design per player block and
// no design absent from the viewer's file, so it cannot tell apart the
// owner filter, the slot order or an absent design. A three-player view
// does.
func TestDesignViews(t *testing.T) {
	g := &engine.Game{DesignSlots: []engine.DesignSlot{
		{Owner: 1, Starbase: true, Slot: 3, Design: 0},
		{Owner: 1, Starbase: true, Slot: 1, Design: 1},
		{Owner: 2, Starbase: true, Slot: 0, Design: 2},
		{Owner: 1, Slot: 4, Design: 3},
		{Owner: 1, Slot: 5, Design: 4},
	}}
	v := engine.PlayerView{
		Player:  0,
		Designs: []engine.DesignSighting{{Design: 0, Full: true}, {Design: 1}, {Design: 2}, {Design: 3, Full: true}},
		Players: []engine.PlayerSighting{{Player: 1, Hab: &[3]engine.EnvRange{}}, {Player: 2}},
	}
	design := map[[2]int]int{{1, 4}: 3, {1, 5}: 4}
	for _, c := range []struct {
		kind      string
		owner, id int
		want      string
	}{
		{"other_players", 0, 0, `{"foreign_designs":{"full":1,"partial":0},"foreign_starbase_designs":{"full":1,"partial":2}}`},
		{"player", 0, 1, `{"player_block":"full","starbase_designs_full":[false,true]}`},
		{"player", 0, 2, `{"player_block":"partial","starbase_designs_full":[false]}`},
		{"player", 0, 3, `{"player_block":null,"starbase_designs_full":[]}`},
		{"design", 1, 4, `{"design_full":true}`},
		{"design", 1, 5, `{"design_full":null}`},
		{"design", 2, 0, `{"design_full":null}`},
	} {
		got := map[string]any{}
		pvDesignViews(g, v, c.kind, c.owner, c.id, design, got)
		b, _ := json.Marshal(got)
		if string(b) != c.want {
			t.Errorf("%s %d/%d: %s, want %s", c.kind, c.owner, c.id, b, c.want)
		}
	}
}

package game

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestLoadOldTerraformBuiltEvent: a version-12 save written before
// terraform units sent one EventTerraformed per click holds last year's
// batch EventBuilt of ItemTerraform. Such a save still loads, and the
// event reaches its player's Report unchanged.
func TestLoadOldTerraformBuiltEvent(t *testing.T) {
	g := newSmoke(t, smokeSeed)
	var buf bytes.Buffer
	if err := g.Save(&buf); err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	var last map[string]json.RawMessage
	if err := json.Unmarshal(doc["last"], &last); err != nil {
		t.Fatal(err)
	}
	planet := -1
	for _, p := range g.State.Planets {
		if p.Owner == 0 {
			planet = p.ID
			break
		}
	}
	old := engine.Event{Kind: engine.EventBuilt, Player: 0, Planet: planet, Fleet: -1, Item: engine.ItemTerraform, Count: 2}
	ev, err := json.Marshal([]engine.Event{old})
	if err != nil {
		t.Fatal(err)
	}
	last["events"] = ev
	if doc["last"], err = json.Marshal(last); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("a save with an old terraform EventBuilt: %v", err)
	}
	r, err := loaded.Report(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Events) != 1 || r.Events[0].Kind != engine.EventBuilt || r.Events[0].Item != engine.ItemTerraform || r.Events[0].Count != 2 || r.Events[0].Planet != planet {
		t.Errorf("report events %+v, want %+v", r.Events, old)
	}
}

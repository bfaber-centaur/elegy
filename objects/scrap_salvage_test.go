package objects

import (
	"slices"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// A fleet scrapped in deep space leaves salvage numbered from its owner's
// pool shared with packets (MESSAGES.md "Object ids in slots", MEASURED
// SV-1): with the owner's packet 0 in flight the salvage takes 1, and the
// scrap event names it (0x05b's salvage slot).
func TestScrapSalvageSharesThePacketPool(t *testing.T) {
	l := newLab(t)
	l.g.Planets = []engine.Planet{{ID: 7, Pos: at(900, 0), Owner: engine.NoOwner}}
	l.g.Objects = &Space{Packets: []Packet{{Owner: 0, Number: 0, Pos: origin, Target: 7, Warp: 5, Cargo: engine.Minerals{1000, 0, 0}}}}
	fi := l.fleet(0, at(0, 300), "Tank", 1)
	l.g.Fleets[fi].Task = engine.Task{Kind: engine.TaskScrap}
	r, err := engine.GenerateTurn(withRules(*l.g), nil, top{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(space(r.Game).Packets); n != 1 {
		t.Fatalf("%d packets, want packet 0 still in flight", n)
	}
	if len(r.Game.Salvage) != 1 || r.Game.Salvage[0].Owner != 0 || r.Game.Salvage[0].Number != 1 {
		t.Fatalf("salvage %+v, want player 0's number 1", r.Game.Salvage)
	}
	i := slices.IndexFunc(r.Events, func(e engine.Event) bool { return e.Kind == engine.EventFleetScrapped })
	if i < 0 || !slices.Equal(r.Events[i].Axes, []int{1}) {
		t.Errorf("scrap event %v, want Axes [1]", r.Events)
	}
}

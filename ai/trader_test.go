package ai

import (
	"slices"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
)

// traderLevels meet the Multi Function Pod's requirements (energy,
// propulsion and electronics 11) and the Meta Morph's.
var traderLevels = [engine.NumFields]int{11, 10, 11, 26, 11, 10}

func hasPart(fs []engine.SlotFill, part string) bool {
	return slices.ContainsFunc(fs, func(f engine.SlotFill) bool { return f.Part == part })
}

// AI.md §5 "Mystery Trader items" (BINARY-ONLY): a Trader part heads
// class 12, and the AI takes it only when it owns it and its tech meets
// the part's requirements.
func TestClassPartTrader(t *testing.T) {
	cat := engine.Components()
	race := engine.Race{PRT: engine.PRTHyperExpansion}
	low := traderLevels
	low[engine.Energy] = 10
	for _, c := range []struct {
		name   string
		levels [engine.NumFields]int
		owned  map[string]bool
		want   bool
	}{
		{"owned", traderLevels, map[string]bool{"Multi Function Pod": true}, true},
		{"not owned", traderLevels, nil, false},
		{"owned, energy short", low, map[string]bool{"Multi Function Pod": true}, false},
	} {
		comp, ok := classPart(cat, 12, race, c.levels, c.owned)
		if !ok {
			t.Fatalf("%s: no part", c.name)
		}
		if got := comp.Name == "Multi Function Pod"; got != c.want {
			t.Errorf("%s: took %s", c.name, comp.Name)
		}
	}
}

// A Meta Morph of Robotoid's list 0 takes the Multi Function Pod in
// slot 4 when the player owns it, and the design keeps it when the View
// is brought up to date (ReadDesign with the owned items); without it the
// slot takes another part.
func TestShipDesignTraderPart(t *testing.T) {
	for _, owned := range []bool{true, false} {
		var res Result
		v := heView(2450, traderLevels)
		if owned {
			v.TraderItems = map[string]bool{"Multi Function Pod": true}
		}
		s := newShipDesigns(v, top{}, &res)
		if !s.store(3, "Meta Morph", robotoidLists[0]) {
			t.Fatalf("owned %v: no design", owned)
		}
		if got := hasPart(s.slots[3].fills, "Multi Function Pod"); got != owned {
			t.Errorf("owned %v: fills %v", owned, s.slots[3].fills)
		}
		s.syncView(v)
		d, ok := v.ship(3)
		if !ok {
			t.Fatalf("owned %v: slot 3 not in the view", owned)
		}
		kept := slices.ContainsFunc(d.Design.Slots, func(sl engine.Slot) bool { return sl.Part.Name == "Multi Function Pod" && sl.Count > 0 })
		if kept != owned {
			t.Errorf("owned %v: view design slots %+v", owned, d.Design.Slots)
		}
	}
}

// The starbase upkeep (researchAndStarbases) takes Langston Shell (class
// 37) for its variant-2 Space Station only when the player owns it, at
// levels meeting its requirements (energy 12, propulsion 9, electronics 9).
func TestStarbaseTraderPart(t *testing.T) {
	station, err := engine.Components().NewDesign("Starbase", spaceStation, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, owned := range []bool{true, false} {
		v := &View{Year: 2450, Self: engine.Player{Race: engine.Race{PRT: engine.PRTClaimAdjuster},
			Research: engine.ResearchState{Levels: [engine.NumFields]int{12, 0, 9, 0, 9, 0}}}}
		v.Starbases = []Design{{Slot: 0, Index: 20, Design: station, Created: 2400}}
		v.Planets = []engine.Planet{{ID: 1, Owner: 0, HasStarbase: true, StarbaseDesign: 20}}
		if owned {
			v.TraderItems = map[string]bool{"Langston Shell": true}
		}
		var res Result
		researchAndStarbases(Rototill, v, top{}, &res)
		var first []engine.SlotFill
		for _, o := range res.Orders {
			if d, ok := o.(engine.DesignOrder); ok && d.Slot == 2 {
				first = d.Fills
			}
		}
		if first == nil {
			t.Fatalf("owned %v: no slot-2 design in %v", owned, res.Orders)
		}
		if got := hasPart(first, "Langston Shell"); got != owned {
			t.Errorf("owned %v: fills %v", owned, first)
		}
	}
}

// ViewOf carries the player's own Trader parts.
func TestViewOfTraderItems(t *testing.T) {
	v := ViewOf(game.Report{TraderItems: map[string]bool{"Multi Function Pod": true}}, Expert)
	if !v.TraderItems["Multi Function Pod"] || len(v.TraderItems) != 1 {
		t.Errorf("TraderItems %v", v.TraderItems)
	}
}

package engine

import (
	"errors"
	"testing"
)

// shipyard is launchGame with design slots: ship slots 0 (Scout) and 1
// (QJ5 Scout), starbase slots 0 (Orbital Fort), 1 (Space Dock) and 2
// (Space Station, the planets' starbase), rich surface minerals, and the
// owner's race.
func shipyard(t *testing.T) *Game {
	g := launchGame(t)
	cat := Components()
	for _, h := range []string{"Orbital Fort", "Space Dock"} {
		d, err := cat.NewDesign(h, h, nil)
		if err != nil {
			t.Fatal(err)
		}
		g.Designs = append(g.Designs, d)
	}
	g.DesignSlots = []DesignSlot{
		{Owner: 0, Slot: 0, Design: 0}, {Owner: 0, Slot: 1, Design: 1},
		{Owner: 0, Starbase: true, Slot: 0, Design: 3}, {Owner: 0, Starbase: true, Slot: 1, Design: 4},
		{Owner: 0, Starbase: true, Slot: 2, Design: 2},
	}
	g.Players[0].Race = Race{MineCost: 5, FactoryCost: 10}
	for i := range g.Planets {
		g.Planets[i].Surface = Minerals{10000, 10000, 10000}
	}
	return g
}

func shipyardRun(g *Game, pi, resources int) (int, []Event) {
	pl := &g.Players[g.Planets[pi].Owner]
	return g.PlanetProduction(pi, ProductionInput{Colony: Colony{Race: pl.Race}, Resources: resources})
}

func (g *Game) ownerCost(d int) Cost {
	return designCost(g.Designs[d], g.Players[0].Race, g.Players[0].Research.Levels)
}

func TestConfirmedProductionOneFleetPerItem(t *testing.T) {
	// PRODUCTION-LAUNCH.md "The new fleet" (CONFIRMED SL-01): a queue of
	// 2 Scouts, 1 Scout and 1 other design makes three fleets of 2, 1 and
	// 1 ships with full tanks; the finished queue is removed.
	g := shipyard(t)
	p := &g.Planets[0]
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 0, Count: 2}, {Kind: ItemShip, Slot: 0, Count: 1}, {Kind: ItemShip, Slot: 1, Count: 1}}
	_, ev := shipyardRun(g, 0, 10000)
	if len(g.Fleets) != 3 || p.HasQueue {
		t.Fatalf("fleets %+v, queue %v %+v", g.Fleets, p.HasQueue, p.Queue)
	}
	for i, want := range []Stack{{Design: 0, Count: 2}, {Design: 0, Count: 1}, {Design: 1, Count: 1}} {
		if f := g.Fleets[i]; f.Stacks[0] != want || f.Fuel != 50*want.Count || f.Number != i+1 {
			t.Errorf("fleet %d: %+v", i, f)
		}
	}
	built := 0
	for _, e := range ev {
		if e.Kind == EventShipsBuilt {
			built++
		}
	}
	if built != 3 {
		t.Errorf("events %+v, want three ships-built", ev)
	}
	// Each slot counts the ships ever built of its design (DesignSlot.Built).
	if a, b := g.DesignSlots[0].Built, g.DesignSlots[1].Built; a != 3 || b != 1 {
		t.Errorf("built counts %d and %d, want 3 and 1", a, b)
	}
}

func TestProductionShipPartial(t *testing.T) {
	// KERNEL.md "Production" (PQ-001 model): a ship item is spent on like
	// any non-auto item. Two of three Scouts complete and launch as one
	// fleet; the third takes its partial percentage and the queue stops.
	g := shipyard(t)
	p := &g.Planets[0]
	c := g.ownerCost(0)
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 0, Count: 3}, {Kind: ItemFactory, Count: 1}}
	shipyardRun(g, 0, 2*c.Resources+c.Resources/2)
	want := largestPercent(c.Resources, c.Resources/2)
	if len(g.Fleets) != 1 || g.Fleets[0].Stacks[0].Count != 2 {
		t.Fatalf("fleets %+v", g.Fleets)
	}
	if len(p.Queue) != 2 || p.Queue[0].Count != 1 || p.Queue[0].Percent != want || p.Factories != 0 {
		t.Errorf("queue %+v, factories %d; want Scout ×1 @%d%% and no factory", p.Queue, p.Factories, want)
	}
}

func TestPredictionShipWithoutStarbase(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Can the planet build it" (BINARY-ONLY): without
	// a starbase nothing is built, the resources stay spent and the item is
	// removed.
	g := shipyard(t)
	p := &g.Planets[1]
	p.HasStarbase, p.StarbaseHull = false, 0
	c := g.ownerCost(0)
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 0, Count: 1}}
	research, _ := shipyardRun(g, 1, c.Resources+7)
	if len(g.Fleets) != 0 || p.HasQueue || research != 7 || p.Surface[Ironium] != 10000-c.Minerals[Ironium] {
		t.Errorf("fleets %d, queue %v, research %d, Fe %d", len(g.Fleets), p.HasQueue, research, p.Surface[Ironium])
	}
}

func TestMeasuredProductionChargesReplacement(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Cost of a replacement" (MEASURED SL-12 for a
	// different hull): a Space Station item where a Space Dock stands is
	// charged the replacement cost, not the fresh one.
	g := shipyard(t)
	p := &g.Planets[0]
	p.StarbaseDesign, p.StarbaseHull = 4, 2
	race, levels := g.Players[0].Race, g.Players[0].Research.Levels
	want, _ := StarbaseReplacementCost(g.Designs[2], g.Designs[4], race, levels)
	if fresh := StarbaseBuildCost(g.Designs[2], race, levels); fresh == want {
		t.Fatalf("replacement %+v equals the fresh cost", want)
	}
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemStarbase, Slot: 2, Count: 1}}
	research, _ := shipyardRun(g, 0, want.Resources+3)
	if p.StarbaseDesign != 2 || research != 3 || p.Surface[Germanium] != 10000-want.Minerals[Germanium] {
		t.Errorf("starbase %d, research %d, Ge %d; want design 2, 3, %d", p.StarbaseDesign, research, p.Surface[Germanium], 10000-want.Minerals[Germanium])
	}
}

func TestSameHullReplacementSameParts(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Cost of a replacement" (BINARY-ONLY): the same
	// hull and the same parts cost nothing.
	g := shipyard(t)
	race, levels := g.Players[0].Race, g.Players[0].Research.Levels
	d, err := Components().NewDesign("Armed Station", "Space Station", []SlotFill{{Slot: 1, Part: "Laser", Count: 4}})
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := StarbaseReplacementCost(d, d, race, levels); !ok || c != (Cost{}) {
		t.Errorf("same design: %+v %v, want nothing", c, ok)
	}
	// Against the bare Station only the hull is credited.
	c, ok := StarbaseReplacementCost(d, g.Designs[2], race, levels)
	laser := designCost(d, race, levels).Resources - designCost(g.Designs[2], race, levels).Resources
	if !ok || c.Resources != (laser+1)/2 {
		t.Errorf("against a bare Station: %+v %v, want %d resources", c, ok, (laser+1)/2)
	}
}

func TestConfirmedEarlierHullClearsShips(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Queued ships" (CONFIRMED SL-12): an Orbital
	// Fort replacing a Space Station removes every ship item and resets
	// every starbase item left; items before it were built already
	// (MEASURED once).
	g := shipyard(t)
	p := &g.Planets[0]
	race, levels := g.Players[0].Race, g.Players[0].Research.Levels
	fort, _ := StarbaseReplacementCost(g.Designs[3], g.Designs[2], race, levels)
	p.HasQueue, p.Queue = true, []QueueItem{
		{Kind: ItemShip, Slot: 0, Count: 1},
		{Kind: ItemStarbase, Slot: 0, Count: 1},
		{Kind: ItemShip, Slot: 1, Count: 1},
		{Kind: ItemStarbase, Slot: 1, Count: 1, Percent: 30},
	}
	shipyardRun(g, 0, g.ownerCost(0).Resources+fort.Resources)
	if len(g.Fleets) != 1 || p.StarbaseDesign != 3 || p.StarbaseHull != 1 {
		t.Fatalf("fleets %d, starbase %d hull %d", len(g.Fleets), p.StarbaseDesign, p.StarbaseHull)
	}
	// With no resources left, the Space Dock item's partial starts from 0%
	// (the PQ-001 formula gives ⌊100/c⌋ − 1 at nothing available), not
	// from the 30% it had.
	dock, _ := StarbaseReplacementCost(g.Designs[4], g.Designs[3], race, levels)
	want := largestPercent(dock.Resources, 0)
	if len(p.Queue) != 1 || p.Queue[0].Kind != ItemStarbase || p.Queue[0].Percent != want || want >= 30 {
		t.Errorf("queue %+v, want the Space Dock item at %d%%", p.Queue, want)
	}
}

func TestQueueOrderDesignItems(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Can the planet build it" (Elegy's chosen rule):
	// a ship item the starbase's dock could not build is refused; an empty
	// slot is refused (ASSUMPTION P3); a ship item at a Station is taken.
	g := shipyard(t)
	g.Planets[1].StarbaseDesign, g.Planets[1].StarbaseHull = 3, 1
	errs, _ := apply(g, 0,
		QueueOrder{Planet: 1, Queue: []QueueItem{{Kind: ItemShip, Slot: 0, Count: 1}}},
		QueueOrder{Planet: 2, Queue: []QueueItem{{Kind: ItemShip, Slot: 0, Count: 1}}},
		QueueOrder{Planet: 1, Queue: []QueueItem{{Kind: ItemShip, Slot: 5, Count: 1}}},
		QueueOrder{Planet: 2, Queue: []QueueItem{{Kind: ItemStarbase, Slot: 2, Count: 1}}},
	)
	for i, want := range []error{nil, ErrOutOfRange, ErrOutOfRange, nil} {
		if !errors.Is(errs[i], want) {
			t.Errorf("order %d: %v, want %v", i, errs[i], want)
		}
	}
}

func TestMeasuredDesignDeleteDropsQueue(t *testing.T) {
	// ORDERS.md "Design delete effect" (MEASURED CO-07): a queue entry
	// building the deleted design is dropped even with progress; other
	// entries stay. A queue left empty is removed (ASSUMPTION P4).
	g := shipyard(t)
	g.Planets[0].HasQueue, g.Planets[0].Queue = true, []QueueItem{{Kind: ItemShip, Slot: 1, Count: 2, Percent: 66}, {Kind: ItemMine, Count: 1}}
	g.Planets[1].HasQueue, g.Planets[1].Queue = true, []QueueItem{{Kind: ItemShip, Slot: 1, Count: 1}}
	if errs, _ := apply(g, 0, DeleteDesignOrder{Slot: 1}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	if q := g.Planets[0].Queue; len(q) != 1 || q[0].Kind != ItemMine {
		t.Errorf("planet 1 queue %+v", q)
	}
	if g.Planets[1].HasQueue {
		t.Errorf("planet 2 queue %+v, want removed", g.Planets[1].Queue)
	}
}

func TestMeasuredDesignEditQueueBuildsEdited(t *testing.T) {
	// ORDERS.md "Design change into an occupied slot" (MEASURED CO-08): a
	// design used only by a queue entry is overwritten in place, and the
	// entry builds the edited design.
	g := shipyard(t)
	g.Designs = append(g.Designs, g.Designs[0])
	g.DesignSlots[0].Design = len(g.Designs) - 1
	p := &g.Planets[0]
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 0, Count: 1}}
	if errs, _ := apply(g, 0, DesignOrder{Slot: 0, Name: "Edited", Hull: "Scout", Fills: []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}}}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	shipyardRun(g, 0, 10000)
	if len(g.Fleets) != 1 || g.Designs[g.Fleets[0].Stacks[0].Design].Name != "Edited" {
		t.Errorf("fleets %+v", g.Fleets)
	}
}

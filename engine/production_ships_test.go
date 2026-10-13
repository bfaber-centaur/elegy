package engine

import (
	"errors"
	"fmt"
	"reflect"
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

func TestStarbaseWithoutDriverDropsPacketItems(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Starbases", mass driver (stars-elegy 35c48f0,
	// CONFIRMED by prediction in PD-1; mixed items BINARY-ONLY): a starbase without a mass driver built where there
	// was none clears the packet destination and speed and silently removes
	// the queued mineral packet items; Auto Mineral Packets and other items
	// stay. With a driver on the new starbase, the destination and the
	// items are kept and the speed becomes the driver's, 5. A driverless
	// starbase replacing one with a driver changes neither the settings nor
	// the items. A starbase the owner lacks the tech for is not built and
	// changes nothing.
	for _, tt := range []struct {
		name                      string
		driver, noTech, oldDriver bool
	}{
		{"no driver", false, false, false},
		{"Mass Driver 5", true, false, false},
		{"not built", false, true, false},
		{"driver before, none now", false, false, true},
	} {
		g := shipyard(t)
		if tt.oldDriver {
			d, err := Components().NewDesign("Station", "Space Station", []SlotFill{{Slot: 0, Part: "Mass Driver 5", Count: 1}})
			if err != nil {
				t.Fatal(err)
			}
			g.Designs[2] = d // the planets' starbase, starbase slot 2
		}
		if tt.noTech {
			g.Players[0].Research.Levels = [NumFields]int{}
			d, err := Components().NewDesign("Fort", "Orbital Fort", []SlotFill{{Slot: 1, Part: "Gatling Gun", Count: 1}})
			if err != nil {
				t.Fatal(err)
			}
			g.Designs[3] = d
		}
		if tt.driver {
			g.Players[0].Race.PRT = PRTPacketPhysics // mass drivers are PP-only
			d, err := Components().NewDesign("Fort", "Orbital Fort", []SlotFill{{Slot: 0, Part: "Mass Driver 5", Count: 1}})
			if err != nil {
				t.Fatal(err)
			}
			g.Designs[3] = d
		}
		p := &g.Planets[0]
		p.HasPacketDest, p.PacketDest, p.PacketSpeed = true, 2, 6
		race, levels := g.Players[0].Race, g.Players[0].Research.Levels
		fort, _ := StarbaseReplacementCost(g.Designs[3], g.Designs[2], race, levels)
		p.HasQueue, p.Queue = true, []QueueItem{
			{Kind: ItemStarbase, Slot: 0, Count: 1},
			{Kind: ItemIroniumPacket, Count: 2},
			{Kind: ItemAutoPackets, Count: 5},
			{Kind: ItemMixedPacket, Count: 1, Percent: 40},
			{Kind: ItemMine, Count: 3},
		}
		shipyardRun(g, 0, fort.Resources)
		if tt.noTech {
			if p.StarbaseDesign != 2 || len(p.Queue) != 4 || p.Queue[0].Kind != ItemIroniumPacket || !p.HasPacketDest || p.PacketSpeed != 6 {
				t.Errorf("%s: starbase %d, queue %+v, destination %v, speed %d", tt.name, p.StarbaseDesign, p.Queue, p.HasPacketDest, p.PacketSpeed)
			}
			continue
		}
		if p.StarbaseDesign != 3 {
			t.Fatalf("%s: starbase %d", tt.name, p.StarbaseDesign)
		}
		var kinds []ItemKind
		for _, it := range p.Queue {
			kinds = append(kinds, it.Kind)
		}
		wantKinds := []ItemKind{ItemAutoPackets, ItemMine}
		wantDest, wantSpeed := false, 0
		switch {
		case tt.oldDriver:
			// The settings and items stay; production then stops at the
			// Ironium item, which a planet without objects cannot launch.
			wantKinds = []ItemKind{ItemIroniumPacket, ItemAutoPackets, ItemMixedPacket, ItemMine}
			wantDest, wantSpeed = true, 6
		case tt.driver:
			wantKinds = []ItemKind{ItemIroniumPacket, ItemAutoPackets, ItemMixedPacket, ItemMine}
			wantDest, wantSpeed = true, 5
		}
		if fmt.Sprint(kinds) != fmt.Sprint(wantKinds) || p.HasPacketDest != wantDest || p.PacketSpeed != wantSpeed {
			t.Errorf("%s: queue %v, destination %v (%d), speed %d; want %v, %v, %d", tt.name, kinds, p.HasPacketDest, p.PacketDest, p.PacketSpeed, wantKinds, wantDest, wantSpeed)
		}
	}
}

func TestProductionRefusesUnentitledDesign(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Can the planet build it" (stars-elegy 35c48f0,
	// CONFIRMED by prediction in FB-1; FB-1's own rows are
	// TestConfirmedFB1RefusedDesigns): a design holding a part the owner's race may
	// not use is not built when a unit completes; the owner is told the
	// plans were lost. The spend is a normal build's. An item with units
	// left is emptied and production stops at it; it is dropped, with its
	// progress, when production next reaches it. An item whose units all
	// completed is removed and production goes on.
	fuelMizer := func(g *Game) {
		d, err := Components().NewDesign("FM Scout", "Scout", []SlotFill{{Slot: 0, Part: "Fuel Mizer", Count: 1}})
		if err != nil {
			t.Fatal(err)
		}
		g.Designs[1] = d // ship slot 1; Fuel Mizer needs IFE, which the race lacks
	}
	for _, tt := range []struct {
		name  string
		count int
	}{{"units left", 3}, {"all completed", 1}} {
		g := shipyard(t)
		fuelMizer(g)
		p := &g.Planets[0]
		unit := g.ownerCost(1)
		p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 1, Count: tt.count}, {Kind: ItemMine, Count: 1}}
		before := p.Surface
		_, ev := shipyardRun(g, 0, 2*unit.Resources+unit.Resources/2)
		lost := 0
		for _, e := range ev {
			if e.Kind == EventPlansLost {
				lost++
			}
		}
		built := min(tt.count, 2)
		spent := Minerals{before[0] - p.Surface[0], before[1] - p.Surface[1], before[2] - p.Surface[2]}
		if len(g.Fleets) != 0 || lost != 1 || spent[0] < built*unit.Minerals[0] || g.DesignSlots[1].Built != 0 {
			t.Errorf("%s: fleets %d, plans lost %d, spent %v for %d units of %v, slot built %d", tt.name, len(g.Fleets), lost, spent, built, unit.Minerals, g.DesignSlots[1].Built)
		}
		if tt.count == 1 {
			// Removed; the Mine item after it is built.
			if p.HasQueue && len(p.Queue) != 0 || p.Mines != 1 {
				t.Errorf("%s: queue %+v, mines %d", tt.name, p.Queue, p.Mines)
			}
			continue
		}
		if len(p.Queue) != 2 || p.Queue[0].Kind != ItemShip || p.Queue[0].Count != 0 || p.Queue[0].Percent == 0 || p.Mines != 0 {
			t.Fatalf("%s: queue %+v, mines %d; want the ship item emptied with its progress, production stopped", tt.name, p.Queue, p.Mines)
		}
		// Next year the empty item is dropped, with its progress, and
		// nothing more is paid or lost for it.
		_, ev = shipyardRun(g, 0, 1000)
		for _, e := range ev {
			if e.Kind == EventPlansLost {
				t.Errorf("%s, next year: plans lost again", tt.name)
			}
		}
		if p.HasQueue && len(p.Queue) != 0 || p.Mines != 1 || len(g.Fleets) != 0 {
			t.Errorf("%s, next year: queue %+v, mines %d, fleets %d", tt.name, p.Queue, p.Mines, len(g.Fleets))
		}
	}

	// A starbase: the old starbase and the packet settings stay, there is
	// no message of its own, and the item is removed.
	g := shipyard(t)
	d, err := Components().NewDesign("Fort", "Orbital Fort", []SlotFill{{Slot: 0, Part: "Mass Driver 5", Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	g.Designs[3] = d // starbase slot 0; Mass Driver 5 is PP-only
	p := &g.Planets[0]
	p.HasPacketDest, p.PacketDest, p.PacketSpeed = true, 2, 6
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemStarbase, Slot: 0, Count: 1}, {Kind: ItemMine, Count: 1}}
	_, ev := shipyardRun(g, 0, 10000)
	for _, e := range ev {
		if e.Kind == EventStarbaseBuilt || e.Kind == EventPlansLost {
			t.Errorf("starbase: event %+v", e)
		}
	}
	if p.StarbaseDesign != 2 || !p.HasPacketDest || p.PacketSpeed != 6 || p.Mines != 1 || p.HasQueue && len(p.Queue) != 0 {
		t.Errorf("starbase: design %d, destination %v, speed %d, mines %d, queue %+v", p.StarbaseDesign, p.HasPacketDest, p.PacketSpeed, p.Mines, p.Queue)
	}
	// Two starbase units, resources for one and a half: the item is
	// emptied with its progress and production stops at it.
	race, levels := g.Players[0].Race, g.Players[0].Research.Levels
	fort, _ := StarbaseReplacementCost(d, g.Designs[2], race, levels)
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemStarbase, Slot: 0, Count: 2}, {Kind: ItemMine, Count: 1}}
	shipyardRun(g, 0, fort.Resources*3/2)
	if p.StarbaseDesign != 2 || len(p.Queue) != 2 || p.Queue[0].Count != 0 || p.Queue[0].Percent == 0 || p.Mines != 1 {
		t.Errorf("two starbase units: design %d, queue %+v, mines %d", p.StarbaseDesign, p.Queue, p.Mines)
	}
}

func TestCanBuildDesign(t *testing.T) {
	// PRODUCTION-LAUNCH.md "Can the planet build it" (stars-elegy 35c48f0):
	// the hull and every part researched and allowed to the race (parts
	// CONFIRMED by prediction in FB-1; the research half and the hull rows
	// BINARY-ONLY) (COMPONENTS.md "Who can build what").
	g := launchGame(t)
	c := Components()
	for _, tt := range []struct {
		name  string
		hull  string
		fills []SlotFill
		race  Race
		level int
		want  bool
	}{
		{"all allowed", "Scout", []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}}, Race{}, 26, true},
		{"part above tech", "Scout", []SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}}, Race{}, 0, false},
		// Long Hump 6 needs propulsion 3: one level short is refused, the
		// level itself builds.
		{"part one level short", "Scout", []SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}}, Race{}, 2, false},
		{"part at its level", "Scout", []SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}}, Race{}, 3, true},
		{"part forbidden to the race", "Scout", []SlotFill{{Slot: 0, Part: "Fuel Mizer", Count: 1}}, Race{}, 26, false},
		{"part allowed by an LRT", "Scout", []SlotFill{{Slot: 0, Part: "Fuel Mizer", Count: 1}}, Race{LRT: LRTs{ImprovedFuelEfficiency: true}}, 26, true},
		{"Mystery Trader part not owned", "Scout", []SlotFill{{Slot: 0, Part: "Enigma Pulsar", Count: 1}}, Race{}, 26, false},
		{"hull forbidden to the race", "Space Dock", nil, Race{}, 26, false},
		{"hull allowed by an LRT", "Space Dock", nil, Race{LRT: LRTs{ImprovedStarbases: true}}, 26, true},
	} {
		d, err := c.NewDesign(tt.name, tt.hull, tt.fills)
		if err != nil {
			t.Fatal(err)
		}
		g.Players[0].Race = tt.race
		for f := range NumFields {
			g.Players[0].Research.Levels[f] = tt.level
		}
		if got := g.canBuild(0, d); got != tt.want {
			t.Errorf("%s: %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestStarbaseQueueChangesKeepEarlierItems(t *testing.T) {
	// The queue changes after a starbase item at index i (afterEarlierHull,
	// dropPacketItems) keep that item and every item before it: those were
	// processed already this year, and production removes or keeps the
	// starbase item itself by its count.
	q := []QueueItem{
		{Kind: ItemAutoAlchemy, Count: 1},
		{Kind: ItemIroniumPacket, Count: 1},
		{Kind: ItemStarbase, Slot: 0, Count: 1, Percent: 20},
		{Kind: ItemShip, Slot: 0, Count: 1},
		{Kind: ItemBoraniumPacket, Count: 1},
		{Kind: ItemStarbase, Slot: 1, Count: 1, Percent: 30},
		{Kind: ItemAutoPackets, Count: 1},
	}
	for _, tt := range []struct {
		name string
		f    func(*production, int)
		want []QueueItem
	}{
		{"earlier hull", (*production).afterEarlierHull, []QueueItem{q[0], q[1], {Kind: ItemStarbase, Slot: 0, Count: 1}, q[4], {Kind: ItemStarbase, Slot: 1, Count: 1}, q[6]}},
		{"no driver", (*production).dropPacketItems, []QueueItem{q[0], q[1], q[2], q[3], q[5], q[6]}},
	} {
		p := Planet{Queue: append([]QueueItem(nil), q...)}
		tt.f(&production{planet: &p}, 2)
		if !reflect.DeepEqual(p.Queue, tt.want) {
			t.Errorf("%s: queue %+v, want %+v", tt.name, p.Queue, tt.want)
		}
	}
}

func TestConfirmedFB1RefusedDesigns(t *testing.T) {
	// PARITY.md FB-1 (CONFIRMED by prediction, stars-elegy 35c48f0): a JOAT
	// at tech 26 queues designs it may not build (placed by state edit),
	// FB-1 read the resources spent from the mines built after the item;
	// here they are what is left for research.
	scout := func(t *testing.T, extra ...SlotFill) Design {
		d, err := Components().NewDesign("Scout", "Scout", append([]SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}}, extra...))
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	setup := func(t *testing.T, d Design) (*Game, *Planet) {
		g := shipyard(t)
		g.Players[0].Race.PRT = PRTJackOfAllTrades
		g.Designs[1] = d // ship slot 1
		return g, &g.Planets[0]
	}
	spent := func(before, after Minerals) Minerals {
		return Minerals{before[0] - after[0], before[1] - after[1], before[2] - after[2]}
	}
	for _, tt := range []struct {
		name      string
		d         Design
		resources int
		spent     Minerals
	}{
		{"a: Gun Scout (Mini Gun, Inner Strength only)", scout(t, SlotFill{Slot: 2, Part: "Mini Gun", Count: 1}), 5, Minerals{2, 5, 2}},
		{"b: MT Scout (unowned Multi Function Pod)", scout(t, SlotFill{Slot: 2, Part: "Multi Function Pod", Count: 1}), 9, Minerals{4, 1, 4}},
	} {
		// One unit from 0%: 0x04f, no fleet, the whole unit cost spent,
		// and the item removed.
		g, p := setup(t, tt.d)
		p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 1, Count: 1}}
		before := p.Surface
		left, ev := shipyardRun(g, 0, 200)
		lost := 0
		for _, e := range ev {
			if e.Kind == EventPlansLost {
				lost++
			}
		}
		if lost != 1 || len(g.Fleets) != 0 || spent(before, p.Surface) != tt.spent || 200-left != tt.resources || p.HasQueue && len(p.Queue) != 0 {
			t.Errorf("%s: plans lost %d, fleets %d, spent %v and %d resources (want %v and %d), queue %+v", tt.name, lost, len(g.Fleets), spent(before, p.Surface), 200-left, tt.spent, tt.resources, p.Queue)
		}
	}

	// d: Gun Scout ×3 with 6 resources (FB-1: "about 6", 39% after):
	// one unit and the progress on the next spent (2/6/2 kT), no mines, an
	// empty item left; next year it is dropped and a mine is built.
	g, p := setup(t, scout(t, SlotFill{Slot: 2, Part: "Mini Gun", Count: 1}))
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 1, Count: 3}, {Kind: ItemMine, Count: 1}}
	before := p.Surface
	shipyardRun(g, 0, 6)
	if got := spent(before, p.Surface); got != (Minerals{2, 6, 2}) || p.Mines != 0 || len(p.Queue) != 2 || p.Queue[0].Count != 0 || p.Queue[0].Percent != 39 {
		t.Errorf("d, year 1: spent %v, mines %d, queue %+v", got, p.Mines, p.Queue)
	}
	before = p.Surface
	shipyardRun(g, 0, 5)
	if got := spent(before, p.Surface); got != (Minerals{}) || p.Mines != 1 || p.HasQueue && len(p.Queue) != 0 {
		t.Errorf("d, year 2: spent %v, mines %d, queue %+v", got, p.Mines, p.Queue)
	}

	// c: Fort to Fort MD5 (Packet Physics only) from 0%: the Fort kept,
	// 6/5/5 kT spent, no message of its own.
	g = shipyard(t)
	g.Players[0].Race.PRT = PRTJackOfAllTrades
	md5, err := Components().NewDesign("Fort MD5", "Orbital Fort", []SlotFill{{Slot: 0, Part: "Mass Driver 5", Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	g.Designs[4] = md5 // starbase slot 1
	p = &g.Planets[0]
	p.StarbaseDesign, p.StarbaseHull = 3, 1 // an Orbital Fort
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemStarbase, Slot: 1, Count: 1}}
	before = p.Surface
	_, ev := shipyardRun(g, 0, 200)
	for _, e := range ev {
		if e.Kind == EventStarbaseBuilt || e.Kind == EventPlansLost {
			t.Errorf("c: event %+v", e)
		}
	}
	if got := spent(before, p.Surface); got != (Minerals{6, 5, 5}) || p.StarbaseDesign != 3 || p.HasQueue && len(p.Queue) != 0 {
		t.Errorf("c: spent %v, starbase %d, queue %+v", got, p.StarbaseDesign, p.Queue)
	}
}

func TestMeasuredStarbaseBuiltNamesDesignSlot(t *testing.T) {
	// MESSAGES.md 0x0cd "Design slot" (stars-elegy 898c526, MEASURED
	// SL-12): the message names the owner's starbase design slot, so the
	// event carries the slot (5), not the design index (3).
	g := shipyard(t)
	g.DesignSlots[2].Slot = 5 // the Orbital Fort
	p := &g.Planets[0]
	race, levels := g.Players[0].Race, g.Players[0].Research.Levels
	fort, _ := StarbaseReplacementCost(g.Designs[3], g.Designs[2], race, levels)
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemStarbase, Slot: 5, Count: 1}}
	_, ev := shipyardRun(g, 0, fort.Resources)
	var got []Event
	for _, e := range ev {
		if e.Kind == EventStarbaseBuilt {
			got = append(got, e)
		}
	}
	if p.StarbaseDesign != 3 || len(got) != 1 || len(got[0].Axes) != 1 || got[0].Axes[0] != 5 || got[0].Planet != p.ID || got[0].Player != 0 {
		t.Errorf("starbase %d, events %+v; want design 3 and one EventStarbaseBuilt naming slot 5", p.StarbaseDesign, got)
	}
}

func TestMeasuredShipsBuiltNamesDesignSlot(t *testing.T) {
	// MESSAGES.md 0x0cd "Design slot" (stars-elegy 898c526, MEASURED for
	// 0x02f): the build message names the owner's ship design slot, so
	// the event carries the slot (4), not the design index (1).
	g := shipyard(t)
	g.DesignSlots[1].Slot = 4 // the QJ5 Scout
	p := &g.Planets[0]
	p.HasQueue, p.Queue = true, []QueueItem{{Kind: ItemShip, Slot: 4, Count: 2}}
	_, ev := shipyardRun(g, 0, 2*g.ownerCost(1).Resources)
	var got []Event
	for _, e := range ev {
		if e.Kind == EventShipsBuilt {
			got = append(got, e)
		}
	}
	if len(got) != 1 || len(got[0].Axes) != 1 || got[0].Axes[0] != 4 || got[0].Count != 2 || got[0].Planet != p.ID || len(g.Fleets) != 1 || got[0].Fleet != g.Fleets[0].ID {
		t.Errorf("events %+v, fleets %d; want one EventShipsBuilt of 2 ships naming slot 4 and the new fleet", got, len(g.Fleets))
	}
}

package engine

import (
	"reflect"
	"testing"
)

// lossQueue is a queue with one item of each kind loseStarbase treats
// differently.
func lossQueue() []QueueItem {
	return []QueueItem{
		{Kind: ItemShip, Count: 2, Percent: 66, Slot: 3},
		{Kind: ItemFactory, Count: 5, Percent: 20},
		{Kind: ItemIroniumPacket, Count: 1, Percent: 10},
		{Kind: ItemMixedPacket, Count: 1},
		{Kind: ItemAutoPackets, Count: 2},
		{Kind: ItemStarbase, Count: 1, Percent: 40, Slot: 1},
		{Kind: ItemAutoMines, Count: 3},
	}
}

// lossStripped is lossQueue after loseStarbase.
func lossStripped() []QueueItem {
	return []QueueItem{
		{Kind: ItemFactory, Count: 5, Percent: 20},
		{Kind: ItemAutoPackets, Count: 2},
		{Kind: ItemStarbase, Count: 1, Slot: 1},
		{Kind: ItemAutoMines, Count: 3},
	}
}

func TestMeasuredCO07bDeleteStarbaseStripsQueue(t *testing.T) {
	// PARITY.md "Queued ships after deleting the starbase design"
	// (stars-elegy ee47fff; MEASURED CO-07b for the outcome, BINARY-ONLY
	// for the strip): deleting the starbase design in use removes the
	// starbase and, at order time, the planet's ship and non-automatic
	// packet items; a starbase item left restarts at 0%; planetary and
	// automatic items stay. A queue left empty is removed (ASSUMPTION P4).
	g := ordersGame()
	if errs, _ := apply(g, 0, DesignOrder{Starbase: true, Slot: 0, Name: "Fort", Hull: "Orbital Fort"}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	sb, _ := g.PlayerDesign(0, true, 0)
	g.Planets[0].HasStarbase, g.Planets[0].StarbaseDesign, g.Planets[0].StarbaseHull = true, sb, 1
	g.Planets[0].StarbaseDock, g.Planets[0].StarbaseDamage = true, 30
	g.Planets[0].HasQueue, g.Planets[0].Queue = true, lossQueue()
	g.Planets[1].HasQueue, g.Planets[1].Queue = true, lossQueue() // player 1's: untouched
	if errs, _ := apply(g, 0, DeleteDesignOrder{Starbase: true, Slot: 0}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	if p := g.Planets[0]; p.HasStarbase || p.StarbaseHull != 0 || p.StarbaseDock || p.StarbaseDamage != 0 || !p.HasQueue || !reflect.DeepEqual(p.Queue, lossStripped()) {
		t.Errorf("planet 1: starbase %v, queue %v %+v, want stripped %+v", p.HasStarbase, p.HasQueue, p.Queue, lossStripped())
	}
	if !reflect.DeepEqual(g.Planets[1].Queue, lossQueue()) {
		t.Errorf("player 1's queue %+v changed", g.Planets[1].Queue)
	}

	g = ordersGame()
	apply(g, 0, DesignOrder{Starbase: true, Slot: 0, Name: "Fort", Hull: "Orbital Fort"})
	sb, _ = g.PlayerDesign(0, true, 0)
	g.Planets[0].HasStarbase, g.Planets[0].StarbaseDesign, g.Planets[0].StarbaseHull = true, sb, 1
	g.Planets[0].HasQueue, g.Planets[0].Queue = true, []QueueItem{{Kind: ItemShip, Count: 2, Percent: 66}, {Kind: ItemShip, Count: 1, Slot: 2}}
	apply(g, 0, DeleteDesignOrder{Starbase: true, Slot: 0})
	if p := g.Planets[0]; p.HasQueue || len(p.Queue) != 0 {
		t.Errorf("CO-07b's ships-only queue: %v %+v, want no queue", p.HasQueue, p.Queue)
	}
}

func TestConfirmedBattleLossStripsQueue(t *testing.T) {
	// COMBAT.md (CONFIRMED CB-047: a destroyed Station's queued
	// Destroyers were gone, its factories stayed); the packet items and
	// the starbase item's restart are BINARY-ONLY (stars-elegy ee47fff).
	station := testDesign(Hull{Name: "Space Station", Armor: 500, Starbase: true}, 100)
	d := []Design{station}
	sb := tokenValues(station, Race{}, true, designCost(station, Race{}, [NumFields]int{}))
	sb.player, sb.design, sb.ships, sb.fleet, sb.planet = 1, 0, 1, -1, 0
	sb.dead = true
	tb := newTestBattle(panicRand{}, d, &sb)
	tb.g.Planets = []Planet{{Owner: 1, HasStarbase: true, StarbaseHull: 1, StarbaseDock: true, StarbaseDamage: 30, HasQueue: true, Queue: lossQueue()}}
	tb.loc.planet = 0
	tb.finish()
	if p := tb.g.Planets[0]; p.HasStarbase || p.StarbaseHull != 0 || p.StarbaseDock || p.StarbaseDamage != 0 || !reflect.DeepEqual(p.Queue, lossStripped()) {
		t.Errorf("starbase %v, queue %+v, want stripped %+v", p.HasStarbase, p.Queue, lossStripped())
	}
}

package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// packetPlanet sets up planet 3 owned by player 0 with a Mass Driver 7
// starbase (when driver), packet destination planet 7 (when dest), the
// given queue and surface minerals, and space objects.
func packetPlanet(t *testing.T, driver, dest bool, surface engine.Minerals, queue ...engine.QueueItem) *lab {
	t.Helper()
	l := newLab(t)
	sb := station(t, l, "Mass Driver 7", "")
	l.g.Planets = []engine.Planet{
		{ID: 3, Pos: origin, Owner: 0, Surface: surface, HasQueue: true, Queue: queue, HasStarbase: driver, StarbaseDesign: sb},
		{ID: 7, Pos: at(50, 0), Owner: engine.NoOwner},
	}
	if dest {
		l.g.Planets[0].HasPacketDest, l.g.Planets[0].PacketDest = true, 7
	}
	l.g.Objects = &Space{}
	return l
}

func produce(l *lab, resources int) (int, []engine.Event) {
	p := &l.g.Planets[0]
	in := engine.ProductionInput{Colony: engine.NewColony(p, &l.g.Players[p.Owner]), Resources: resources, GrownPop: p.Population}
	return l.g.PlanetProduction(0, in)
}

func kinds(ev []engine.Event) []engine.EventKind {
	var k []engine.EventKind
	for _, e := range ev {
		k = append(k, e.Kind)
	}
	return k
}

// Production charges a packet item what the launch spends (KERNEL.md
// "Item costs"; OBJECTS.md "Launch", MEASURED OB-028, OB-029).
func TestPacketItemCostMatchesLaunch(t *testing.T) {
	for _, prt := range []engine.PRT{engine.PRTJackOfAllTrades, engine.PRTPacketPhysics, engine.PRTInterstellarTraveler} {
		race := engine.Race{PRT: prt}
		res := 10
		if prt == engine.PRTPacketPhysics {
			res = 5
		}
		for k, mineral := range map[engine.ItemKind]int{
			engine.ItemIroniumPacket: engine.Ironium, engine.ItemBoraniumPacket: engine.Boranium,
			engine.ItemGermaniumPacket: engine.Germanium, engine.ItemMixedPacket: Mixed, engine.ItemAutoPackets: Mixed,
		} {
			_, spend := PacketItem(race, mineral == Mixed)
			want := engine.Cost{Resources: res}
			for m := range engine.NumMinerals {
				if mineral == Mixed || mineral == m {
					want.Minerals[m] = spend
				}
			}
			if got := engine.ItemCost(race, k); got != want {
				t.Errorf("PRT %d item %d: cost %+v, want %+v", prt, k, got, want)
			}
		}
	}
}

// A packet item without a destination is removed whatever its count, with
// the cancel message and then "completed its orders", no packet and the
// minerals unchanged (KERNEL.md "Packet items", MEASURED OB-028-F). The
// resources go to research.
func TestMeasuredPacketItemNoDestination(t *testing.T) {
	for _, c := range []struct {
		name         string
		driver, dest bool
	}{{"no destination", true, false}, {"no driver", false, true}} {
		l := packetPlanet(t, c.driver, c.dest, engine.Minerals{500, 0, 0}, engine.QueueItem{Kind: engine.ItemIroniumPacket, Count: 3})
		research, ev := produce(l, 100)
		p := l.g.Planets[0]
		if got := kinds(ev); len(got) != 2 || got[0] != engine.EventPacketNoDriver || got[1] != engine.EventQueueCompleted {
			t.Errorf("%s: events %v", c.name, ev)
		}
		if p.HasQueue || len(p.Queue) != 0 || research != 100 || p.Surface != (engine.Minerals{500, 0, 0}) || len(l.g.Objects.(*Space).Packets) != 0 {
			t.Errorf("%s: queue %v %v, research %d, surface %v, packets %v", c.name, p.HasQueue, p.Queue, research, p.Surface, l.g.Objects.(*Space).Packets)
		}
	}
}

// ASSUMPTION P6: a destination naming no planet is no destination.
func TestPacketDestinationNamingNoPlanet(t *testing.T) {
	l := packetPlanet(t, true, true, engine.Minerals{500, 0, 0}, engine.QueueItem{Kind: engine.ItemIroniumPacket, Count: 1})
	l.g.Planets[0].PacketDest = 99
	if _, ev := produce(l, 100); len(ev) == 0 || ev[0].Kind != engine.EventPacketNoDriver {
		t.Errorf("events %v", ev)
	}
}

// The units completed in a year leave as one launch; a partial unit stays
// as a percentage and stops the queue (KERNEL.md "Packet items").
func TestPacketItemUnitLoop(t *testing.T) {
	l := packetPlanet(t, true, true, engine.Minerals{300, 0, 0},
		engine.QueueItem{Kind: engine.ItemIroniumPacket, Count: 3}, engine.QueueItem{Kind: engine.ItemMine, Count: 1})
	_, ev := produce(l, 100)
	p := l.g.Planets[0]
	pk := l.g.Objects.(*Space).Packets
	if len(pk) != 1 || pk[0].Cargo != (engine.Minerals{200, 0, 0}) || pk[0].Target != 7 {
		t.Fatalf("packets %+v, want one of 200 kT", pk)
	}
	// 300 − 220 = 80 kT of 110 → 72%; resources 100 − 20 = 80 of 10 → 100%.
	if p.Surface[0] != 300-220-110*72/100 || len(p.Queue) != 2 || p.Queue[0] != (engine.QueueItem{Kind: engine.ItemIroniumPacket, Count: 1, Percent: 72}) || p.Mines != 0 {
		t.Errorf("surface %v, queue %+v, mines %d", p.Surface, p.Queue, p.Mines)
	}
	if len(ev) != 1 || ev[0].Kind != engine.EventBuilt || ev[0].Item != engine.ItemIroniumPacket || ev[0].Count != 2 {
		t.Errorf("events %+v", ev)
	}
}

// Auto Mineral Packets builds Mixed Mineral Packets up to its count and
// keeps the count; a mineral-short unit is skipped and the walk goes on; a
// unit short only of resources leaves a Mixed Mineral Packet ×1 partial at
// the front and stops the queue; without a destination it builds nothing,
// sends nothing and stays (KERNEL.md "Packet items").
func TestAutoMineralPackets(t *testing.T) {
	auto := engine.QueueItem{Kind: engine.ItemAutoPackets, Count: 5}
	factory := engine.QueueItem{Kind: engine.ItemFactory, Count: 1}

	// Built: two units of 10 resources each; the 5 left make a 59% partial
	// (the PQ-001 percentage rule).
	l := packetPlanet(t, true, true, engine.Minerals{1000, 1000, 1000}, auto, factory)
	_, ev := produce(l, 25)
	p := l.g.Planets[0]
	if pk := l.g.Objects.(*Space).Packets; len(pk) != 1 || pk[0].Cargo != (engine.Minerals{80, 80, 80}) {
		t.Errorf("packets %+v", pk)
	}
	if len(p.Queue) != 3 || p.Queue[0] != (engine.QueueItem{Kind: engine.ItemMixedPacket, Count: 1, Percent: 59}) || p.Queue[1] != auto {
		t.Errorf("queue %+v", p.Queue)
	}
	if got := kinds(ev); len(got) != 1 || got[0] != engine.EventBuilt {
		t.Errorf("events %v", ev)
	}

	// Mineral-short: skipped with nothing spent; the factory is built.
	l = packetPlanet(t, true, true, engine.Minerals{40, 1000, 1000}, auto, factory)
	produce(l, 100)
	p = l.g.Planets[0]
	if len(l.g.Objects.(*Space).Packets) != 0 || p.Factories != 1 || p.Surface[0] != 40 || len(p.Queue) != 1 || p.Queue[0] != auto {
		t.Errorf("mineral-short: factories %d, surface %v, queue %+v", p.Factories, p.Surface, p.Queue)
	}

	// No destination: nothing, no message, stays.
	l = packetPlanet(t, true, false, engine.Minerals{1000, 1000, 1000}, auto)
	_, ev = produce(l, 100)
	p = l.g.Planets[0]
	if len(l.g.Objects.(*Space).Packets) != 0 || len(p.Queue) != 1 || p.Surface != (engine.Minerals{1000, 1000, 1000}) {
		t.Errorf("no destination: queue %+v, surface %v", p.Queue, p.Surface)
	}
	for _, e := range ev {
		if e.Kind == engine.EventPacketNoDriver {
			t.Errorf("no destination: events %v", ev)
		}
	}
}

// ASSUMPTION P7: an Auto Alchemy prefix before a removed packet item stays
// and stands before the next item; here the mine finishes and leaves with
// its prefix (KERNEL.md "Auto Alchemy before a multi-count item").
func TestPacketItemRemovedBehindAlchemyPrefix(t *testing.T) {
	l := packetPlanet(t, true, false, engine.Minerals{},
		engine.QueueItem{Kind: engine.ItemAutoAlchemy, Count: 1},
		engine.QueueItem{Kind: engine.ItemMixedPacket, Count: 1},
		engine.QueueItem{Kind: engine.ItemMine, Count: 1})
	_, ev := produce(l, 100)
	p := l.g.Planets[0]
	if len(ev) == 0 || ev[0].Kind != engine.EventPacketNoDriver || len(p.Queue) != 0 || p.Mines != 1 {
		t.Errorf("events %v, queue %+v, mines %d", ev, p.Queue, p.Mines)
	}
}

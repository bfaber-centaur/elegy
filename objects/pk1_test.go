package objects

import (
	"slices"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// PK-1, year 2 (stars-elegy PRODUCTION-LAUNCH.md "A starbase destroyed in
// battle", e1b2ad9, MEASURED): each planet queues a Catcher 7 (Orbital
// Fort, one Mass Driver 7) at 99% and then one ironium packet. A planet
// whose starbase was destroyed the year before has no packet settings
// (loseStarbase, ELEGY CHOICE matching PK-1): the starbase is built and
// the packet item is cancelled (0x129), with no launch and no ironium
// spent on it. A control whose starbase stayed launches to its old
// destination at its old warp 9.
func TestMeasuredPK1RebuiltStarbaseKeepsNoPacketSettings(t *testing.T) {
	l := newLab(t)
	d, err := engine.Components().NewDesign("Catcher 7", "Orbital Fort", []engine.SlotFill{{Slot: 0, Part: "Mass Driver 7", Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	l.g.Designs = append(l.g.Designs, d)
	sb := len(l.g.Designs) - 1
	l.g.DesignSlots = []engine.DesignSlot{{Owner: 0, Starbase: true, Slot: 0, Design: sb}}
	for f := range l.g.Players[0].Research.Levels {
		l.g.Players[0].Research.Levels[f] = 26
	}
	queue := func() []engine.QueueItem {
		return []engine.QueueItem{{Kind: engine.ItemStarbase, Slot: 0, Count: 1, Percent: 99}, {Kind: engine.ItemIroniumPacket, Count: 1}}
	}
	surface := engine.Minerals{5000, 5000, 5000}
	l.g.Planets = []engine.Planet{
		{ID: 4, Pos: origin, Owner: 0, Surface: surface, HasQueue: true, Queue: queue()},
		{ID: 3, Pos: at(0, 50), Owner: 0, Surface: surface, HasQueue: true, Queue: queue(),
			HasStarbase: true, StarbaseDesign: sb, StarbaseHull: 1, HasPacketDest: true, PacketDest: 17, PacketSpeed: 9},
		{ID: 17, Pos: at(50, 0), Owner: engine.NoOwner},
	}
	l.g.Objects = &Space{}
	run := func(pi int) []engine.EventKind {
		p := &l.g.Planets[pi]
		_, ev := l.g.PlanetProduction(pi, engine.ProductionInput{Colony: engine.NewColony(p, &l.g.Players[0]), Resources: 1000, GrownPop: p.Population})
		var k []engine.EventKind
		for _, e := range ev {
			if e.Kind != engine.EventQueueCompleted {
				k = append(k, e.Kind)
			}
		}
		return k
	}

	victim := run(0)
	if i := slices.Index(victim, engine.EventPacketNoDriver); i < 0 || slices.Contains(victim, engine.EventBuilt) || len(l.g.Objects.(*Space).Packets) != 0 {
		t.Errorf("rebuilt planet: events %v, packets %+v; want the starbase built, the packet item cancelled, no launch", victim, l.g.Objects.(*Space).Packets)
	}
	if p := l.g.Planets[0]; !p.HasStarbase || p.HasPacketDest || p.HasQueue {
		t.Errorf("rebuilt planet: starbase %v, destination %v, queue %v %+v", p.HasStarbase, p.HasPacketDest, p.HasQueue, p.Queue)
	}

	control := run(1)
	pk := l.g.Objects.(*Space).Packets
	if slices.Contains(control, engine.EventPacketNoDriver) || len(pk) != 1 || pk[0].Target != 17 || pk[0].From != 3 || pk[0].Cargo[engine.Ironium] == 0 {
		t.Errorf("control: events %v, packets %+v; want one ironium launch to planet 17", control, pk)
	}
}

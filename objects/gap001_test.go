package objects

import (
	"slices"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// gap001Produce is one GAP-001 packet planet of player 0, an Interstellar
// Traveler, with 5000 kT of each mineral and a Mass Driver 7 starbase
// (driver) aimed at a planet (dest), producing with the given resources.
// Resources are the population / 1000 the case assumes (20 or 37; a
// pre-run check), with no research tax (player 0's research is 0%).
func gap001Produce(t *testing.T, driver, dest bool, resources int, queue ...engine.QueueItem) (*lab, int, []engine.Event) {
	t.Helper()
	l := packetPlanet(t, driver, dest, engine.Minerals{5000, 5000, 5000}, queue...)
	l.g.Players[0].Race.PRT = engine.PRTInterstellarTraveler
	research, ev := produce(l, resources)
	return l, research, ev
}

func gap001Packets(l *lab) []engine.Minerals {
	var out []engine.Minerals
	for _, p := range l.g.Objects.(*Space).Packets {
		if p.Warp != 7 || p.Class != 1 {
			out = append(out, engine.Minerals{-1, -1, -1})
			continue
		}
		out = append(out, p.Cargo)
	}
	return out
}

// GAP-001 P (KERNEL.md "Item costs", "Packet items", "Production";
// OBJECTS.md "Launch"; stars-elegy 8addf9e). Every packet is warp 7,
// class 1.
func TestPredictionGAP001PacketCosts(t *testing.T) {
	q := func(k engine.ItemKind, n int) engine.QueueItem { return engine.QueueItem{Kind: k, Count: n} }
	built := engine.EventBuilt
	done := engine.EventQueueCompleted

	// P1: one IT mixed item launches 40/40/40 and spends 48 of each and
	// 10 resources (BINARY-ONLY); the other 10 resources go to research.
	l, research, ev := gap001Produce(t, true, true, 20, q(engine.ItemMixedPacket, 1))
	p := l.g.Planets[0]
	if pk := gap001Packets(l); !slices.Equal(pk, []engine.Minerals{{40, 40, 40}}) || p.Surface != (engine.Minerals{4952, 4952, 4952}) ||
		p.HasQueue || research != 10 || !slices.Equal(kinds(ev), []engine.EventKind{built, done}) {
		t.Errorf("P1: packets %v, surface %v, queue %v %+v, research %d, events %v", pk, p.Surface, p.HasQueue, p.Queue, research, ev)
	}

	// P2: ironium ×20 with 37 resources. Three units (30 resources,
	// 3 × 120 kT) launch as one 300 kT packet. The fourth has 7 of 10
	// resources: max(trunc(8·100/10) − 1, trunc(7·100/10)) = 79%, which
	// charges trunc(120·79/100) = 94 kT. Left: ×17 at 79%, 4546 kT.
	l, _, ev = gap001Produce(t, true, true, 37, q(engine.ItemIroniumPacket, 20))
	p = l.g.Planets[0]
	if pk := gap001Packets(l); !slices.Equal(pk, []engine.Minerals{{300, 0, 0}}) || p.Surface != (engine.Minerals{4546, 5000, 5000}) ||
		len(p.Queue) != 1 || p.Queue[0] != (engine.QueueItem{Kind: engine.ItemIroniumPacket, Count: 17, Percent: 79}) || !slices.Equal(kinds(ev), []engine.EventKind{built}) {
		t.Errorf("P2: packets %v, surface %v, queue %+v, events %v", pk, p.Surface, p.Queue, ev)
	}

	// P3: mixed ×20 with 37 resources: 120/120/120 launches, 3 × 48 + 37
	// (trunc(48·79/100)) = 181 kT of each spent, ×17 at 79% left.
	l, _, ev = gap001Produce(t, true, true, 37, q(engine.ItemMixedPacket, 20))
	p = l.g.Planets[0]
	if pk := gap001Packets(l); !slices.Equal(pk, []engine.Minerals{{120, 120, 120}}) || p.Surface != (engine.Minerals{4819, 4819, 4819}) ||
		len(p.Queue) != 1 || p.Queue[0] != (engine.QueueItem{Kind: engine.ItemMixedPacket, Count: 17, Percent: 79}) || !slices.Equal(kinds(ev), []engine.EventKind{built}) {
		t.Errorf("P3: packets %v, surface %v, queue %+v, events %v", pk, p.Surface, p.Queue, ev)
	}
}

// GAP-001 A (KERNEL.md "Packet items": "No driver or no destination",
// "Auto Mineral Packets"; MESSAGES.md 0x03e, 0x129; stars-elegy 8addf9e).
func TestPredictionGAP001AutoAndDriverless(t *testing.T) {
	q := func(k engine.ItemKind, n int) engine.QueueItem { return engine.QueueItem{Kind: k, Count: n} }
	auto := func(n int) engine.QueueItem { return q(engine.ItemAutoPackets, n) }
	built := engine.EventBuilt
	done := engine.EventQueueCompleted

	// A1 with 20 resources: two mixed units launch together (80/80/80).
	// The third has 0 resources: the percentage rule gives
	// max(trunc(1·100/10) − 1, 0) = 9%, costing 0 resources and
	// trunc(48·9/100) = 4 kT, so a hidden Mixed ×1 at 9% goes to the
	// front and the queue stops (ASSUMPTION: a unit is tried with 0
	// resources left). With 30 or more resources all three launch
	// (120/120/120), the auto item stays at ×3 and 0x03e is sent.
	l, _, ev := gap001Produce(t, true, true, 20, auto(3))
	p := l.g.Planets[0]
	if pk := gap001Packets(l); !slices.Equal(pk, []engine.Minerals{{80, 80, 80}}) || p.Surface != (engine.Minerals{4900, 4900, 4900}) ||
		!slices.Equal(p.Queue, []engine.QueueItem{{Kind: engine.ItemMixedPacket, Count: 1, Percent: 9}, auto(3)}) || !slices.Equal(kinds(ev), []engine.EventKind{built}) {
		t.Errorf("A1 (20): packets %v, surface %v, queue %+v, events %v", pk, p.Surface, p.Queue, ev)
	}
	l, _, ev = gap001Produce(t, true, true, 30, auto(3))
	p = l.g.Planets[0]
	if pk := gap001Packets(l); !slices.Equal(pk, []engine.Minerals{{120, 120, 120}}) || p.Surface != (engine.Minerals{4856, 4856, 4856}) ||
		!slices.Equal(p.Queue, []engine.QueueItem{auto(3)}) || !slices.Equal(kinds(ev), []engine.EventKind{built, done}) {
		t.Errorf("A1 (30): packets %v, surface %v, queue %+v, events %v", pk, p.Surface, p.Queue, ev)
	}

	// A2: ×20 with 37 resources: three units launch (120/120/120); the
	// fourth, short only of resources, leaves a hidden Mixed ×1 at 79% at
	// the front and the queue stops; no 0x03e.
	l, _, ev = gap001Produce(t, true, true, 37, auto(20))
	p = l.g.Planets[0]
	if pk := gap001Packets(l); !slices.Equal(pk, []engine.Minerals{{120, 120, 120}}) || p.Surface != (engine.Minerals{4819, 4819, 4819}) ||
		!slices.Equal(p.Queue, []engine.QueueItem{{Kind: engine.ItemMixedPacket, Count: 1, Percent: 79}, auto(20)}) || !slices.Equal(kinds(ev), []engine.EventKind{built}) {
		t.Errorf("A2: packets %v, surface %v, queue %+v, events %v", pk, p.Surface, p.Queue, ev)
	}

	// A3: no driver. Ironium ×2, then mixed ×1: each item is removed whole
	// with 0x129, then 0x03e; no packet, minerals unchanged, all 20
	// resources to research.
	l, research, ev := gap001Produce(t, false, true, 20, q(engine.ItemIroniumPacket, 2), q(engine.ItemMixedPacket, 1))
	p = l.g.Planets[0]
	cancel := engine.EventPacketNoDriver
	if pk := gap001Packets(l); len(pk) != 0 || p.Surface != (engine.Minerals{5000, 5000, 5000}) || p.HasQueue || research != 20 ||
		!slices.Equal(kinds(ev), []engine.EventKind{cancel, cancel, done}) || ev[0].Item != engine.ItemIroniumPacket || ev[1].Item != engine.ItemMixedPacket {
		t.Errorf("A3: packets %v, surface %v, queue %v %+v, research %d, events %+v", pk, p.Surface, p.HasQueue, p.Queue, research, ev)
	}

	// A4 (no driver) and A5 (no destination), Auto Mineral Packets ×5
	// alone: nothing built or spent, no 0x129, the item stays, and 0x03e
	// is sent (MESSAGES.md 0x03e: the end of the queue was reached with no
	// auto item held back by missing minerals; the auto-only case is
	// inferred there, not observed).
	for _, c := range []struct {
		name         string
		driver, dest bool
	}{{"A4", false, true}, {"A5", true, false}} {
		l, research, ev := gap001Produce(t, c.driver, c.dest, 20, auto(5))
		p := l.g.Planets[0]
		if pk := gap001Packets(l); len(pk) != 0 || p.Surface != (engine.Minerals{5000, 5000, 5000}) || research != 20 ||
			!slices.Equal(p.Queue, []engine.QueueItem{auto(5)}) || !slices.Equal(kinds(ev), []engine.EventKind{done}) {
			t.Errorf("%s: packets %v, surface %v, queue %+v, research %d, events %v", c.name, pk, p.Surface, p.Queue, research, ev)
		}
	}
}

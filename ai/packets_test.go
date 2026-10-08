package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// shotView is a tiny galaxy (W = 400) with Cybertron's starbase at
// (100, 200) carrying driver, so w = rating + 3, and the planets given
// (relative to the corner (1000, 1000)) as ids 2, 3, ...
func shotView(t *testing.T, driver string, planets ...engine.Point) *View {
	t.Helper()
	v := caView(t, 2420)
	sb, err := engine.Components().NewDesign("Driver", "Space Station", []engine.SlotFill{fill(0, driver, 1)})
	if err != nil {
		t.Fatal(err)
	}
	v.Starbases = append(v.Starbases, Design{Slot: 9, Index: 40, Design: sb})
	home := engine.Point{X: 1100, Y: 1200}
	v.Universe = []PlanetPos{{ID: 1, Pos: home}, {ID: 99, Pos: engine.Point{X: 1400, Y: 1400}}}
	for i, p := range planets {
		v.Universe = append(v.Universe, PlanetPos{ID: 2 + i, Pos: engine.Point{X: 1000 + p.X, Y: 1000 + p.Y}})
	}
	v.Planets[0].Pos, v.Planets[0].StarbaseDesign = home, 40
	v.Planets[0].Surface = engine.Minerals{500, 100, 100}
	return v
}

func shot(t *testing.T, v *View, legacy packetLegacy, draws ...int) (*cyberTurn, map[int]bool, *script) {
	t.Helper()
	rng := &script{t: t, draws: draws}
	ct := cyberTest(t, v, rng)
	marked := map[int]bool{}
	ct.scannerShot(&v.Planets[0], 0, marked, legacy)
	if len(rng.draws) != 0 {
		t.Errorf("%d scripted draws left", len(rng.draws))
	}
	return ct, marked, rng
}

// §6 scanner shot, direction 1 (drawn 0): from (100, 200) the +x −y
// diagonal meets y = 0 at (300, 0); the slide Random(120) − 60 = 10 gives
// (310, 0); the inset Random(64) = 20 gives (310, 20). Planet 2 at
// (312, 20) is nearest: the destination is set at warp 8 and one
// ironium packet goes to the front of the queue. Clean marks name the
// destination; the LEGACY BUG marks the planet one id higher.
func TestScannerShot(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		v := shotView(t, "Mass Driver 5", engine.Point{X: 312, Y: 20}, engine.Point{X: 300, Y: 40})
		ct, marked, rng := shot(t, v, packetLegacy{markNextID: legacy}, 0, 70, 20)
		if got := rng.bounds; len(got) != 3 || got[0] != 7 || got[1] != 120 || got[2] != 64 {
			t.Errorf("draw bounds %v, want [7 120 64]", got)
		}
		set := ordersOf[engine.PlanetSettingsOrder](ct.res.Orders)
		if len(set) != 1 || set[0].PacketDest != 2 || set[0].PacketSpeed != 8 || !set[0].HasPacketDest {
			t.Errorf("settings %+v, want packets to planet 2 at warp 8", set)
		}
		q := ct.q.get(&v.Planets[0])
		if len(q) == 0 || q[0].Kind != engine.ItemIroniumPacket || q[0].Count != 1 {
			t.Errorf("queue %+v, want one ironium packet first", q)
		}
		want := 2
		if legacy {
			want = 3
		}
		if len(marked) != 1 || !marked[want] {
			t.Errorf("legacy %v: marks %v, want planet %d", legacy, marked, want)
		}
	}
}

// From (30, 200), direction 2 (drawn 2) meets y = 0 at (30, 0); a slide
// of −60 runs 30 past the corner, so the point continues down the x = 0
// edge to (0, 30); the inset then applies to x: (5, 30).
func TestScannerShotPastCorner(t *testing.T) {
	v := shotView(t, "Mass Driver 5", engine.Point{X: 5, Y: 31}, engine.Point{X: 40, Y: 0})
	home := engine.Point{X: 1030, Y: 1200}
	v.Universe[0].Pos, v.Planets[0].Pos = home, home
	ct, _, _ := shot(t, v, packetLegacy{}, 2, 0, 5)
	set := ordersOf[engine.PlanetSettingsOrder](ct.res.Orders)
	if len(set) != 1 || set[0].PacketDest != 2 {
		t.Errorf("settings %+v, want packets to planet 2", set)
	}
}

// A destination nearer than w² ly is no shot at all, after all three
// draws; with Ultra Driver 11 (w = 14) the LEGACY BUG's overflow passes
// it.
func TestScannerShotTooNear(t *testing.T) {
	// Direction 4 (drawn 4): (0, 200); no slide (60); inset 0.
	near := engine.Point{X: 50, Y: 200} // 50 ly from home
	v := shotView(t, "Mass Driver 5", near)
	ct, marked, _ := shot(t, v, packetLegacy{}, 4, 60, 0)
	if len(ct.res.Orders) != 0 || len(ct.q.get(&v.Planets[0])) != 0 || len(marked) != 0 {
		t.Errorf("orders %v; want none for a planet 50 ly away at w = 8 (64 ly)", ct.res.Orders)
	}
	for _, legacy := range []bool{false, true} {
		v := shotView(t, "Ultra Driver 11", near)
		ct, _, _ := shot(t, v, packetLegacy{overflow: legacy}, 4, 60, 0)
		got := len(ordersOf[engine.PlanetSettingsOrder](ct.res.Orders))
		if (got == 1) != legacy {
			t.Errorf("w = 14, overflow %v: %d settings orders", legacy, got)
		}
	}
}

// A planet Cybertron owns is no shot.
func TestScannerShotOwnPlanet(t *testing.T) {
	v := shotView(t, "Mass Driver 5", engine.Point{X: 312, Y: 20})
	v.Planets = append(v.Planets, engine.Planet{ID: 2, Pos: v.Universe[2].Pos, Owner: 0})
	v.Known[2] = engine.PlanetReport{Planet: 2, Level: engine.ReportOwn, Owner: 0}
	ct, _, _ := shot(t, v, packetLegacy{}, 0, 70, 20)
	if len(ct.res.Orders) != 0 {
		t.Errorf("orders %v, want none", ct.res.Orders)
	}
}

// Attack packets: an expert with minerals to spare aims at the nearest
// other player's planet in range that it can kill, queues the packets
// and marks the target; the scanner shot is not tried.
func TestAttackPacket(t *testing.T) {
	v := shotView(t, "Mass Driver 5", engine.Point{X: 150, Y: 200})
	v.Known[2] = engine.PlanetReport{Planet: 2, Level: engine.ReportNormal, Owner: 1, PopEstimate: 4000}
	v.Planets[0].Surface = engine.Minerals{3000, 2000, 1000}
	ct := cyberTest(t, v, &script{t: t})
	marked := map[int]bool{}
	if !ct.attackPacket(&v.Planets[0], 0, marked) {
		t.Fatal("no attack")
	}
	// pop 10: 16000·140 / (64·95) = 368.4; 50 ly at w² = 64 with q 0.75:
	// ÷ 0.75^0.78125 → 460.0 kT, 7 packets, all ironium (the most left).
	q := ct.q.get(&v.Planets[0])
	n := 0
	for _, it := range q {
		if it.Kind == engine.ItemIroniumPacket {
			n += it.Count
		}
	}
	set := ordersOf[engine.PlanetSettingsOrder](ct.res.Orders)
	if n != 7 || len(set) != 1 || set[0].PacketDest != 2 || set[0].PacketSpeed != 8 || !marked[2] {
		t.Errorf("queue %+v, settings %+v, marks %v; want 7 ironium packets to planet 2 at warp 8", q, set, marked)
	}
}

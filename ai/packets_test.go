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
// destination (Elegy's rules); jrc3-faithful's LEGACY BUG marks the
// planet one id higher.
func TestScannerShot(t *testing.T) {
	for _, rules := range []engine.Ruleset{engine.ElegyRules(), engine.FaithfulRules()} {
		legacy := rules.ID == engine.FaithfulRules().ID
		v := shotView(t, "Mass Driver 5", engine.Point{X: 312, Y: 20}, engine.Point{X: 300, Y: 40})
		v.Rules = rules
		ct, marked, rng := shot(t, v, packetLegacyOf(v.Rules), 0, 70, 20)
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
			t.Errorf("%s: marks %v, want planet %d", rules.ID, marked, want)
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

// Attack packets, cybertron.md §6's worked example: need 284 at
// f = 0.75^0.807 sends 358 kT, 6 packets (multiplying by f would send 4).
// The expert aims at the nearest other player's planet it can kill,
// queues the packets and marks the target; the scanner shot is not tried.
func TestAttackPacket(t *testing.T) {
	// Planet 2 at (51, 8) from home: D = 51.62 ly, D/w² = 0.8066 at w = 8.
	v := shotView(t, "Mass Driver 5", engine.Point{X: 151, Y: 208})
	// pop 800/400 = 2: 16000·min(1000, 108)/(64·95) = 284.2 → 284.
	v.Known[2] = engine.PlanetReport{Planet: 2, Level: engine.ReportNormal, Owner: 1, PopEstimate: 800}
	v.Planets[0].Surface = engine.Minerals{3000, 2000, 1000}
	v.Planets[0].Population, v.Planets[0].Factories = 3000, 200
	ct := cyberTest(t, v, &script{t: t})
	marked := map[int]bool{}
	if !ct.attackPacket(&v.Planets[0], 0, marked) {
		t.Fatal("no attack")
	}
	q := ct.q.get(&v.Planets[0])
	n := 0
	for _, it := range q {
		if it.Kind == engine.ItemIroniumPacket {
			n += it.Count
		}
	}
	set := ordersOf[engine.PlanetSettingsOrder](ct.res.Orders)
	if n != 6 || len(set) != 1 || set[0].PacketDest != 2 || set[0].PacketSpeed != 8 || !marked[2] {
		t.Errorf("queue %+v, settings %+v, marks %v; want 6 ironium packets to planet 2 at warp 8", q, set, marked)
	}
}

// The budget C = min(M, 70·((R/2 − 5)/5)) limits targets: with too few
// resources the same planet does not qualify.
func TestAttackPacketBudget(t *testing.T) {
	v := shotView(t, "Mass Driver 5", engine.Point{X: 151, Y: 208})
	v.Known[2] = engine.PlanetReport{Planet: 2, Level: engine.ReportNormal, Owner: 1, PopEstimate: 800}
	v.Planets[0].Surface = engine.Minerals{3000, 2000, 1000}
	v.Planets[0].Population, v.Planets[0].Factories = 100, 0
	ct := cyberTest(t, v, &script{t: t})
	if ct.attackPacket(&v.Planets[0], 0, map[int]bool{}) {
		t.Errorf("attacked with %d resources", v.available(&v.Planets[0], 0).Resources)
	}
}

// The packet LEGACY BUGs come from the game's ruleset: off under Elegy's
// rules, on under jrc3-faithful's.
func TestPacketLegacyFromRules(t *testing.T) {
	if got := packetLegacyOf(engine.ElegyRules()); got != (packetLegacy{}) {
		t.Errorf("elegy: %+v, want both off", got)
	}
	if got := packetLegacyOf(engine.FaithfulRules()); got != (packetLegacy{markNextID: true, overflow: true}) {
		t.Errorf("jrc3-faithful: %+v, want both on", got)
	}
}

// A target with a starbase in view qualifies once its design is known in
// full: a Mass Driver 5 catcher (c = 5) at w = 8 raises the kill mass to
// 16000·108/((64 − 25)·95) = 466, so A = 466/0.75^0.807 = 587 kT, 9
// packets. Seen only partially, or with a catcher as fast as w, the
// planet is no target.
func TestAttackPacketCatcher(t *testing.T) {
	catcher := func(t *testing.T, driver string) engine.Design {
		t.Helper()
		d, err := engine.Components().NewDesign("Their Fort", "Space Station", []engine.SlotFill{fill(0, driver, 1)})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	for _, tc := range []struct {
		name   string
		driver string // "" when the design is not known in full
		want   int    // packets; 0 for no attack
	}{
		{"known", "Mass Driver 5", 9},
		{"partial", "", 0},
		{"as fast", "Super Driver 8", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := shotView(t, "Mass Driver 5", engine.Point{X: 151, Y: 208})
			v.Known[2] = engine.PlanetReport{Planet: 2, Level: engine.ReportNormal, Owner: 1, PopEstimate: 800, Starbase: true, StarbaseDesign: 77}
			if tc.driver != "" {
				v.Foreign = map[int]engine.Design{77: catcher(t, tc.driver)}
			}
			v.Planets[0].Surface = engine.Minerals{3000, 2000, 1000}
			v.Planets[0].Population, v.Planets[0].Factories = 3000, 200
			ct := cyberTest(t, v, &script{t: t})
			marked := map[int]bool{}
			attacked := ct.attackPacket(&v.Planets[0], 0, marked)
			n := 0
			for _, it := range ct.q.get(&v.Planets[0]) {
				if it.Kind >= engine.ItemIroniumPacket && it.Kind <= engine.ItemIroniumPacket+2 {
					n += it.Count
				}
			}
			if attacked != (tc.want > 0) || n != tc.want {
				t.Errorf("attacked %v with %d packets, want %d (resources %d)", attacked, n, tc.want, v.available(&v.Planets[0], 0).Resources)
			}
		})
	}
}

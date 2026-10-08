package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// station adds a Space Station starbase design with the given orbital
// slot parts (slots 0 and 10; "" leaves a slot empty).
func station(t *testing.T, l *lab, a, b string) int {
	t.Helper()
	var fills []engine.SlotFill
	for i, name := range []string{a, b} {
		if name != "" {
			fills = append(fills, engine.SlotFill{Slot: 10 * i, Part: name, Count: 1})
		}
	}
	d, err := engine.Components().NewDesign("SB", "Space Station", fills)
	if err != nil {
		t.Fatal(err)
	}
	l.g.Designs = append(l.g.Designs, d)
	return len(l.g.Designs) - 1
}

// Driver warp, packet warp and decay class (OBJECTS.md "Launch",
// CONFIRMED OB-028).
func TestConfirmedPacketLaunchRules(t *testing.T) {
	l := newLab(t)
	for _, c := range []struct {
		a, b  string
		dw, t int
	}{
		{"Mass Driver 7", "Mass Driver 7", 7, 1}, // OB-028-C
		{"Mass Driver 7", "Mass Driver 5", 7, 0}, // OB-028-D
		{"Mass Driver 7", "", 7, 0},
	} {
		dw, tt, ok := DriverWarp(l.g.Designs[station(t, l, c.a, c.b)])
		if !ok || dw != c.dw || tt != c.t {
			t.Errorf("%s + %s: Dw %d t %d", c.a, c.b, dw, tt)
		}
	}
	if _, _, ok := DriverWarp(l.g.Designs[station(t, l, "", "")]); ok {
		t.Error("no driver")
	}
	for _, c := range []struct{ set, dw, t, w int }{{11, 7, 0, 7}, {0, 7, 1, 8}, {0, 7, 0, 7}, {9, 7, 0, 9}, {10, 7, 0, 10}, {4, 7, 0, 7}} {
		if got := PacketWarp(c.set, c.dw, c.t); got != c.w {
			t.Errorf("setting %d, Dw %d, t %d: warp %d, want %d", c.set, c.dw, c.t, got, c.w)
		}
	}
	// OB-028-A: warp 9 from a Mass Driver 7 is class 2; IT warp 7 class 1,
	// warp 10 class 3 (OB-028-H, I).
	for _, c := range []struct {
		w, dw, t int
		it       bool
		k        int
	}{{9, 7, 0, false, 2}, {7, 7, 0, true, 1}, {10, 7, 0, true, 3}, {8, 7, 1, false, 0}, {13, 7, 0, false, 3}} {
		if got := DecayClass(c.w, c.dw, c.t, c.it); got != c.k {
			t.Errorf("W %d Dw %d t %d IT %v: class %d, want %d", c.w, c.dw, c.t, c.it, got, c.k)
		}
	}
}

// Decay (OBJECTS.md "Flight and decay", CONFIRMED OB-003, OB-023, OB-028).
func TestConfirmedPacketDecay(t *testing.T) {
	one := func(kt int) engine.Minerals { return engine.Minerals{kt, 0, 0} }
	for _, c := range []struct {
		name  string
		kt, k int
		pp    bool
		share float64
		want  int
	}{
		{"OB-028 class 2 launch year", 100, 2, false, 0.5, 88},
		{"OB-028 class 3 launch year", 100, 3, false, 0.5, 75},
		{"OB-028 IT class 1 launch year", 100, 1, false, 0.5, 90},
		{"OB-028-G 70% of the launch flight", 500, 3, false, 0.35, 413},
		{"OB-023 PP class 1", 1000, 1, true, 1, 950},
		{"OB-023 PP class 2", 1000, 2, true, 1, 880},
		{"OB-023 PP class 3", 1000, 3, true, 1, 750},
		{"OB-023 minimum", 50, 1, false, 1, 40},
		{"OB-023 PP minimum", 50, 1, true, 1, 45},
		{"OB-003-C 5% of a year", 100, 1, false, 0.05, 90},
		{"class 0", 100, 0, false, 1, 100},
		{"minimum capped", 4, 1, false, 1, 0},
	} {
		if got := Decay(one(c.kt), c.k, c.pp, c.share)[0]; got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
	if got := Decay(engine.Minerals{0, 100, 0}, 2, false, 1); got[0] != 0 || got[1] != 75 {
		t.Errorf("empty mineral decayed: %v", got)
	}
}

// impactLab is a lab with planet 0 owned by player 1 at at(100, 0).
func impactLab(t *testing.T, pop int) *lab {
	l := newLab(t)
	l.g.Planets = []engine.Planet{{ID: 1, Pos: at(100, 0), Owner: 1, Population: pop}}
	return l
}

// Impact (OBJECTS.md "Impact", CONFIRMED OB-003, OB-009, OB-022, OB-030-A).
func TestConfirmedPacketImpact(t *testing.T) {
	pk := Packet{Owner: 0, Warp: 10, Cargo: engine.Minerals{1000, 0, 0}}
	// Unowned planet: one ninth arrives, no damage.
	l := impactLab(t, 0)
	l.g.Planets[0].Owner = engine.NoOwner
	if im := hit(l.g, ImpactContext{}, pk, 0, &script{}); im.Added[0] != 111 || im.Damage != 0 {
		t.Errorf("unowned: %+v", im)
	}
	if got := (Impact{}); got.Emptied {
		t.Fatal()
	}
	// OB-009: 1000 kT at warp 10 into 1000 units, no driver: 625 killed.
	l = impactLab(t, 1000)
	if im := hit(l.g, ImpactContext{}, pk, 0, &script{}); im.Killed != 625 || l.g.Planets[0].Population != 375 || im.Added[0] != 111 {
		t.Errorf("no driver: %+v", im)
	}
	// Against a Mass Driver 7 catcher: q = 490, +546, 318 killed.
	l = impactLab(t, 1000)
	l.g.Planets[0].HasStarbase, l.g.Planets[0].StarbaseDesign = true, station(t, l, "Mass Driver 7", "")
	if im := hit(l.g, ImpactContext{}, pk, 0, &script{}); im.Caught != 490 || im.Added[0] != 546 || im.Killed != 318 {
		t.Errorf("MD7 catcher: %+v", im)
	}
	// OB-022: an IT target halves c²: 24, 240‰, +324, 475 killed; with 50
	// SDI defenses 418 killed and defenses 50 → 30.
	l = impactLab(t, 1000)
	l.g.Players[1].Race.PRT = engine.PRTInterstellarTraveler
	l.g.Planets[0].HasStarbase, l.g.Planets[0].StarbaseDesign = true, station(t, l, "Mass Driver 7", "")
	if im := hit(l.g, ImpactContext{}, pk, 0, &script{}); im.Caught != 240 || im.Added[0] != 324 || im.Killed != 475 {
		t.Errorf("IT catcher: %+v", im)
	}
	l.g.Planets[0].Population, l.g.Planets[0].Defenses = 1000, 50
	ctx := ImpactContext{DefenseShare: func(int) float64 { return 0.8801 }}
	if im := hit(l.g, ctx, pk, 0, &script{}); im.Killed != 418 || l.g.Planets[0].Defenses != 30 {
		t.Errorf("IT with SDI: %+v, defenses %d", im, l.g.Planets[0].Defenses)
	}
	// A packet no faster than the catcher: all of it, no damage.
	l = impactLab(t, 1000)
	l.g.Planets[0].HasStarbase, l.g.Planets[0].StarbaseDesign = true, station(t, l, "Mass Driver 7", "Mass Driver 7")
	if im := hit(l.g, ImpactContext{}, Packet{Warp: 8, Cargo: pk.Cargo}, 0, &script{}); im.Added[0] != 1000 || im.Killed != 0 {
		t.Errorf("caught: %+v", im)
	}
	// OB-030-A: Alternate Reality planets take no damage.
	l = impactLab(t, 1000)
	l.g.Players[1].Race.PRT = engine.PRTAlternateReality
	if im := hit(l.g, ImpactContext{}, pk, 0, &script{}); im.Killed != 0 || im.Added[0] != 111 {
		t.Errorf("AR: %+v", im)
	}
	// OB-009-E: an owner's own packet damages its own planet; damage at
	// or above the population empties it.
	l = impactLab(t, 600)
	if im := hit(l.g, ImpactContext{}, Packet{Owner: 1, Warp: 10, Cargo: pk.Cargo}, 0, &script{}); !im.Emptied || l.g.Planets[0].Population != 0 {
		t.Errorf("own packet: %+v", im)
	}
}

// Defense loss with a small dmg (OBJECTS.md "Impact" step 7): Dk is 0,
// then 1 when rand(20) < dmg.
func TestPredictionPacketDefenses(t *testing.T) {
	l := impactLab(t, 1000)
	l.g.Planets[0].Defenses = 10
	pk := Packet{Warp: 10, Cargo: engine.Minerals{16, 0, 0}} // dmg 10
	if im := hit(l.g, ImpactContext{}, pk, 0, &script{9}); im.Damage != 10 || im.DefensesLost != 1 {
		t.Errorf("rand(20) = 9 < 10: %+v", im)
	}
	l.g.Planets[0].Defenses = 10
	if im := hit(l.g, ImpactContext{}, pk, 0, &script{10}); im.DefensesLost != 0 {
		t.Errorf("rand(20) = 10: %+v", im)
	}
}

// Launch, merge and flight (OBJECTS.md "Launch", "Flight and decay",
// CONFIRMED OB-028-B, E: two items, one 200 kT packet; OB-028 A–I: half
// a year's flight on the launch year).
func TestConfirmedPacketFlight(t *testing.T) {
	l := newLab(t)
	sb := station(t, l, "Mass Driver 7", "")
	l.g.Planets = []engine.Planet{
		{ID: 1, Pos: at(0, 0), Owner: 0, HasStarbase: true, StarbaseDesign: sb, Surface: engine.Minerals{1000, 1000, 1000}},
		{ID: 2, Pos: at(100, 0), Owner: engine.NoOwner},
	}
	s := &Space{}
	o := PacketOrder{Dest: 2, Speed: 9}
	a := s.Launch(l.g, 0, o, engine.Ironium, 1)
	b := s.Launch(l.g, 0, o, engine.Ironium, 1)
	if a.NoDriver || !b.Merged || len(s.Packets) != 1 || s.Packets[0].Cargo[0] != 200 || s.Packets[0].Class != 2 || a.Spend[0] != 110 {
		t.Fatalf("launch: %+v %+v %+v", a, b, s.Packets)
	}
	// A mixed item launches 40 kT of each; the merge rule does not look
	// at the minerals.
	mixed := &Space{}
	if m := mixed.Launch(l.g, 0, o, Mixed, 1); m.Merged || m.Cargo != (engine.Minerals{40, 40, 40}) || m.Spend != (engine.Minerals{44, 44, 44}) {
		t.Errorf("mixed: %+v", m)
	}
	// Launch year: ⌊81/2⌋ = 40 ly, half a year's decay (class 2: 200 → 175).
	if im := s.FlyLaunched(l.g, ImpactContext{}, &script{}); len(im) != 0 || s.Packets[0].Pos != at(40, 0) || s.Packets[0].Cargo[0] != 175 || s.Packets[0].New {
		t.Fatalf("launch year: %+v", s.Packets)
	}
	// Next year: 81 ly would pass 60 ly to go; it arrives after 60/81 of
	// a year: 175 loses ⌊175·25·0.7407/100⌋ = 32 → 143; unowned: +15.
	ims := s.MovePackets(l.g, ImpactContext{}, &script{})
	if len(ims) != 1 || len(s.Packets) != 0 || ims[0].Added[0] != 15 || l.g.Planets[1].Surface[0] != 15 {
		t.Errorf("arrival: %+v", ims)
	}
	// No driver or no destination: nothing launched (OB-028-F).
	l.g.Planets[0].StarbaseDesign = station(t, l, "", "")
	if r := s.Launch(l.g, 0, o, engine.Ironium, 1); !r.NoDriver || len(s.Packets) != 0 {
		t.Errorf("no driver: %+v", r)
	}
}

package objects

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// rhino is a fleet scanner with R 50 and no penetration.
func rhino(pos engine.Point) Scanner { return Scanner{Pos: pos, R: 50, Fleet: true} }

// Minefield sight (SCANNING.md "Space objects", CONFIRMED OB-018).
func TestConfirmedMinefieldSight(t *testing.T) {
	l := newLab(t)
	sp := &Space{Minefields: []Minefield{{Owner: 1, Pos: at(12, 0), Count: 100}}}
	if got := sp.Scan(l.g, 0, []Scanner{rhino(origin)}); !reflect.DeepEqual(got.Minefields, []int{0}) || !reflect.DeepEqual(got.Owners, []int{1}) {
		t.Errorf("12 ly with R 50: %+v", got)
	}
	// A quarter of normal range: 13 ly is not seen when unknown.
	sp = &Space{Minefields: []Minefield{{Owner: 1, Pos: at(13, 0), Count: 100}}}
	if got := sp.Scan(l.g, 0, []Scanner{rhino(origin)}); got.Minefields != nil || got.Owners != nil || sp.Minefields[0].KnownBy(0) {
		t.Errorf("13 ly unknown: %+v", got)
	}
	// Known: seen within the full range, 50 ly but not 51 (OB-018 E–G).
	for _, c := range []struct{ dx, n int }{{50, 1}, {51, 0}} {
		sp = &Space{Minefields: []Minefield{{Owner: 1, Pos: at(c.dx, 0), Count: 100, Known: []bool{true}}}}
		if got := sp.Scan(l.g, 0, []Scanner{rhino(origin)}); len(got.Minefields) != c.n {
			t.Errorf("known at %d ly: %+v", c.dx, got)
		}
	}
	// Inside the field, 30 ly from its centre (OB-018-C); a planet there
	// does not get the inside rule (BINARY-ONLY).
	sp = &Space{Minefields: []Minefield{{Owner: 1, Pos: at(30, 0), Count: 1000}}}
	if got := sp.Scan(l.g, 0, []Scanner{rhino(origin)}); len(got.Minefields) != 1 || !sp.Minefields[0].KnownBy(0) {
		t.Errorf("inside: %+v", got)
	}
	sp.Minefields[0].Known = nil
	if got := sp.Scan(l.g, 0, []Scanner{{Pos: origin, R: 50}}); got.Minefields != nil {
		t.Errorf("planet inside: %+v", got)
	}
	// Own fields are always listed and make nobody known, but ownership
	// does not make a field known: only the owner's own sight does
	// (SCANNING.md "Space objects", MEASURED MF-13a, MF-13c: a lone field
	// beyond every scanner ends the year with an empty known set).
	sp = &Space{Minefields: []Minefield{{Owner: 0, Pos: at(900, 0), Count: 100}, {Owner: 0, Pos: at(10, 0), Count: 100}}}
	if got := sp.Scan(l.g, 0, []Scanner{rhino(origin)}); !reflect.DeepEqual(got.Minefields, []int{0, 1}) || got.Owners != nil ||
		sp.Minefields[0].KnownBy(0) || !sp.Minefields[1].KnownBy(0) {
		t.Errorf("own: %+v, known %v %v", got, sp.Minefields[0].Known, sp.Minefields[1].Known)
	}
	// MF-13c: a field the owner's fleet sits inside is known to the owner
	// only.
	sp = &Space{Minefields: []Minefield{{Owner: 1, Pos: at(500, 0), Count: 400}}}
	sp.Scan(l.g, 1, []Scanner{rhino(at(500, 0))})
	sp.Scan(l.g, 0, []Scanner{rhino(origin)})
	if m := sp.Minefields[0]; !m.KnownBy(1) || m.KnownBy(0) {
		t.Errorf("laid under the owner's fleet: %v", m.Known)
	}
}

// Wormhole sight (SCANNING.md "Space objects", CONFIRMED OB-011-H,
// OB-017, OB-018 H, I, WT batch; the known band is BINARY-ONLY).
func TestConfirmedWormholeSight(t *testing.T) {
	l := newLab(t)
	hole := func(dx int, known bool) *Space {
		e := WormholeEnd{Pos: at(dx, 0)}
		if known {
			e.Known = []bool{true}
		}
		return &Space{Wormholes: []Wormhole{{Ends: [2]WormholeEnd{e, {Pos: at(5000, 0)}}}}}
	}
	for _, c := range []struct {
		dx    int
		known bool
		sc    Scanner
		seen  bool
	}{
		{12, false, rhino(origin), true},
		{13, false, rhino(origin), false},
		{40, true, rhino(origin), true},  // BINARY-ONLY band R/4..R
		{51, true, rhino(origin), false}, // OB-011-H: never beyond R
		{27, false, Scanner{Pos: origin, R: 50, P: 33, Fleet: true}, true},
		{52, false, Scanner{Pos: origin, R: 50, P: 33, Fleet: true}, false},
	} {
		sp := hole(c.dx, c.known)
		got := sp.Scan(l.g, 0, []Scanner{c.sc})
		if seen := len(got.Wormholes) == 1; seen != c.seen || got.Owners != nil || (seen && (got.Wormholes[0] != WormholeSighting{0, 0} || !sp.Wormholes[0].Ends[0].KnownBy(0))) {
			t.Errorf("end at %d ly, known %v: %+v", c.dx, c.known, got)
		}
	}
}

// Packets and the Trader (CONFIRMED OB-018 J, K, OB-012, OB-011-J).
func TestConfirmedPacketAndTraderSight(t *testing.T) {
	l := newLab(t)
	sp := &Space{
		Packets: []Packet{{Owner: 1, Pos: at(48, 0), Warp: 7}, {Owner: 1, Pos: at(53, 0), Warp: 7}},
		Traders: []Trader{{Pos: at(3000, 3000)}},
	}
	got := sp.Scan(l.g, 0, []Scanner{rhino(origin)})
	if !reflect.DeepEqual(got.Packets, []int{0}) || !reflect.DeepEqual(got.Traders, []int{0}) || !reflect.DeepEqual(got.Owners, []int{1}) {
		t.Errorf("JOAT viewer: %+v", got)
	}
	l.g.Players[0].Race.PRT = engine.PRTPacketPhysics
	if got := sp.Scan(l.g, 0, nil); !reflect.DeepEqual(got.Packets, []int{0, 1}) {
		t.Errorf("PP viewer: %+v", got)
	}
}

// A PP player's packets scan with R = P = warp² (CONFIRMED OB-012): a
// warp-5 packet sees a minefield at 20 ly and misses one at 30.
func TestConfirmedPacketScanners(t *testing.T) {
	l := newLab(t)
	sp := &Space{
		Packets:    []Packet{{Owner: 0, Pos: origin, Warp: 5}, {Owner: 1, Pos: origin, Warp: 9}},
		Minefields: []Minefield{{Owner: 1, Pos: at(20, 0), Count: 16}, {Owner: 1, Pos: at(30, 0), Count: 16}},
	}
	if sc := sp.PacketScanners(l.g, 0); sc != nil {
		t.Errorf("JOAT packet scanners: %+v", sc)
	}
	l.g.Players[0].Race.PRT = engine.PRTPacketPhysics
	if sc := sp.PacketScanners(l.g, 0); !reflect.DeepEqual(sc, []Scanner{{Pos: origin, R: 25, P: 25}}) {
		t.Errorf("PP packet scanners: %+v", sc)
	}
	if got := sp.Scan(l.g, 0, nil); !reflect.DeepEqual(got.Minefields, []int{0}) {
		t.Errorf("packet sight: %+v", got)
	}
}

// Space Demolition fields see other players' fleets inside them that are
// not in orbit (CONFIRMED OB-014-B); a cloaked fleet needs rand(100) ≥ c
// (BINARY-ONLY).
func TestConfirmedDemolitionSight(t *testing.T) {
	l := newLab(t)
	l.g.Players[0].Race.PRT = engine.PRTSpaceDemolition
	l.g.Planets = []engine.Planet{{ID: 1, Pos: at(5, 0), Owner: engine.NoOwner}}
	sp := &Space{Minefields: []Minefield{{Owner: 0, Pos: origin, Count: 2500}}}
	inside := l.fleet(1, at(10, 0), "Laser DD", 1)
	l.fleet(1, at(5, 0), "Laser DD", 1) // orbiting
	l.fleet(1, at(100, 0), "Laser DD", 1)
	l.fleet(0, at(1, 0), "Laser DD", 1)
	none := func(int) int { return 0 }
	if got := sp.DemolitionSightings(l.g, 0, none, &count{}); !reflect.DeepEqual(got, []int{inside}) {
		t.Errorf("SD: %v", got)
	}
	cloak := func(fi int) int { return 30 }
	if got := sp.DemolitionSightings(l.g, 0, cloak, &script{29}); got != nil {
		t.Errorf("cloaked, rand 29: %v", got)
	}
	if got := sp.DemolitionSightings(l.g, 0, cloak, &script{30}); !reflect.DeepEqual(got, []int{inside}) {
		t.Errorf("cloaked, rand 30: %v", got)
	}
	l.g.Players[0].Race.PRT = engine.PRTJackOfAllTrades
	if got := sp.DemolitionSightings(l.g, 0, none, &count{}); got != nil {
		t.Errorf("not SD: %v", got)
	}
}

// Hits and sweeps teach the field (BINARY-ONLY).
func TestPredictionMinefieldKnowledge(t *testing.T) {
	l := newLab(t)
	sp := &Space{Minefields: []Minefield{{Owner: 1, Kind: Standard, Pos: origin, Count: 400}, {Owner: 1, Kind: Heavy, Pos: origin, Count: 400}}}
	fi := l.fleet(0, at(10, 0), "Laser DD", 1)
	sp.LearnHit(l.g, &l.g.Fleets[fi], Stop{Hit: true, Kind: Standard}, at(10, 0))
	if !sp.Minefields[0].KnownBy(0) || sp.Minefields[1].KnownBy(0) {
		t.Errorf("hit: %+v", sp.Minefields)
	}
	// A known field 40 ly away is now seen by a Rhino.
	if got := sp.Scan(l.g, 0, []Scanner{rhino(at(40, 0))}); !reflect.DeepEqual(got.Minefields, []int{0}) {
		t.Errorf("after hit: %+v", got)
	}
}

// Learning a field never reaches a shallow copy's Known (CloneObjects
// copies Space.Minefields without their Known slices).
func TestElegyDecisionKnownCopyOnWrite(t *testing.T) {
	l := newLab(t)
	sp := &Space{Minefields: []Minefield{{Owner: 1, Pos: at(10, 0), Count: 100, Known: []bool{false, true}}}}
	c := sp.CloneObjects().(*Space)
	c.Scan(l.g, 0, []Scanner{rhino(origin)})
	if !c.Minefields[0].KnownBy(0) || sp.Minefields[0].KnownBy(0) {
		t.Errorf("clone %v, original %v", c.Minefields[0].Known, sp.Minefields[0].Known)
	}
}

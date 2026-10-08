package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// gateLab has gated planets at origin (player 0) and at(dx, 0) (player
// 1, who lists player 0 as a friend), both with gate.
func gateLab(t *testing.T, gate string, dx int) *lab {
	t.Helper()
	l := newLab(t)
	d, err := engine.Components().NewDesign("Gate", "Space Station", []engine.SlotFill{{Slot: 0, Part: gate, Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	l.g.Designs = append(l.g.Designs, d)
	sb := len(l.g.Designs) - 1
	l.g.Planets = []engine.Planet{
		{ID: 1, Pos: origin, Owner: 0, HasStarbase: true, StarbaseDesign: sb, Population: 100},
		{ID: 2, Pos: at(dx, 0), Owner: 1, HasStarbase: true, StarbaseDesign: sb},
	}
	l.g.Players[1].Relations = []engine.Relation{engine.RelationFriend, engine.RelationFriend}
	return l
}

// Gate limits and danger (OBJECTS.md "Stargates", CONFIRMED GT-001).
func TestConfirmedGateDanger(t *testing.T) {
	sg := func(name string) Gate {
		d, err := engine.Components().NewDesign("G", "Space Station", []engine.SlotFill{{Slot: 10, Part: name, Count: 1}})
		if err != nil {
			t.Fatal(err)
		}
		g, ok := StarbaseGate(d)
		if !ok {
			t.Fatalf("%s: no gate", name)
		}
		return g
	}
	if g := sg("Stargate 100/250"); g != (Gate{100, 250}) {
		t.Errorf("100/250: %+v", g)
	}
	if g := sg("Stargate any/any"); g != (Gate{0, anyRange}) {
		t.Errorf("any/any: %+v", g)
	}
	for _, c := range []struct {
		d, r, m, ms, md, pct int
	}{
		{1250, 250, 500, 100, 100, 100}, // GT-001: lost
		{380, 250, 50, 100, 100, 13},    // OB-021: 13%
		{250, 250, 100, 100, 100, 0},
		{100, 250, 150, 100, 100, 23}, // (500 − 150)·25 = 8750, twice → 7656
		{100, anyRange, 1000, 0, 0, 0},
	} {
		if got := GateDanger(c.d, c.r, c.m, c.ms, c.md); got != c.pct {
			t.Errorf("d %d R %d mass %d: %d%%, want %d", c.d, c.r, c.m, got, c.pct)
		}
	}
}

// A jump with losses (OB-021): 5 Laser Destroyers at 13% took 65/500
// each; the fleet lands on the destination; a friend's gate works.
func TestConfirmedGateJump(t *testing.T) {
	l := gateLab(t, "Stargate 100/250", 380)
	fi := l.fleet(0, origin, "Laser DD", 5)
	j := Jump(l.g, fi, at(380, 0), &script{50, 50, 50, 50, 50})
	f := l.g.Fleets[fi]
	if j.Refused != GateOK || f.Pos != at(380, 0) || j.Designs[0].Pct != 13 || f.Stacks[0].Count != 5 || f.Stacks[0].Damage != (engine.Damage{Pct: 100, Units: 65}) {
		t.Errorf("jump %+v, fleet %+v", j, f)
	}
	// ⌊13/3⌋ = 4: rand(100) = 3 destroys a ship; fuel scales 100 → 67
	// when one of three goes (OB-021).
	l = gateLab(t, "Stargate 100/250", 380)
	fi = l.fleet(0, origin, "Laser DD", 3)
	l.g.Fleets[fi].Fuel = 100
	j = Jump(l.g, fi, at(380, 0), &script{3, 50, 50})
	if f := l.g.Fleets[fi]; j.Designs[0].Destroyed != 1 || f.Stacks[0].Count != 2 || f.Fuel != 67 {
		t.Errorf("one lost: %+v, fleet %+v", j, f)
	}
}

// Refusals (OBJECTS.md "Refusal order", CONFIRMED GT-001, GT-003).
func TestConfirmedGateRefusals(t *testing.T) {
	// Source before destination: deep space fleet, deep space waypoint
	// (GT-003 R6).
	l := gateLab(t, "Stargate 100/250", 200)
	fi := l.fleet(0, at(5, 5), "Laser DD", 1)
	if j := Jump(l.g, fi, at(7, 7), &script{}); j.Refused != NoSourceGate {
		t.Errorf("deep space: %+v", j)
	}
	fi = l.fleet(0, origin, "Laser DD", 1)
	if j := Jump(l.g, fi, at(7, 7), &script{}); j.Refused != NoDestinationPlanet {
		t.Errorf("deep space waypoint (GT-001 M): %+v", j)
	}
	// One-sided friendship (GT-001 D/E): player 1 listing 0 as an enemy
	// closes its gates to 0, whatever 0 thinks of 1.
	l.g.Players[1].Relations = []engine.Relation{engine.RelationEnemy, engine.RelationFriend}
	if j := Jump(l.g, fi, at(200, 0), &script{}); j.Refused != DestinationNotFriend {
		t.Errorf("enemy destination: %+v", j)
	}
	// Range: 1250 ly at 5× 250 is lost, 1251 refused (GT-001); the
	// refused freighter still unloads its minerals (OB-021).
	l = gateLab(t, "Stargate 100/250", 1251)
	fi = l.fleet(0, origin, "Freighter", 1)
	l.g.Fleets[fi].Cargo.Minerals = engine.Minerals{100, 0, 0}
	if j := Jump(l.g, fi, at(1251, 0), &script{}); j.Refused != TooFar || l.g.Fleets[fi].Pos != origin || l.g.Planets[0].Surface[0] != 100 || l.g.Fleets[fi].Cargo.Minerals[0] != 0 {
		t.Errorf("too far: %+v, fleet %+v", j, l.g.Fleets[fi])
	}
	l = gateLab(t, "Stargate 100/250", 1250)
	fi = l.fleet(0, origin, "Laser DD", 1)
	if j := Jump(l.g, fi, at(1250, 0), &script{}); !j.FleetLost || !j.Designs[0].Lost {
		t.Errorf("5× range: %+v", j)
	}
	// Colonists from a friend's planet: refused, minerals unloaded there
	// (GT-001 F, F2).
	l = gateLab(t, "Stargate 100/250", 200)
	l.g.Planets[0].Owner = 1
	l.g.Players[1].Relations = []engine.Relation{engine.RelationFriend, engine.RelationFriend}
	fi = l.fleet(0, origin, "Freighter", 1)
	l.g.Fleets[fi].Cargo = engine.Cargo{Minerals: engine.Minerals{30, 0, 0}, Colonists: 10}
	if j := Jump(l.g, fi, at(200, 0), &script{}); j.Refused != ForeignColonists || l.g.Fleets[fi].Cargo.Colonists != 10 || l.g.Planets[0].Surface[0] != 30 {
		t.Errorf("foreign colonists: %+v", j)
	}
}

// Interstellar Traveler (OB-022): cargo kept, no ships destroyed.
func TestConfirmedGateIT(t *testing.T) {
	l := gateLab(t, "Stargate 100/250", 380)
	l.g.Players[0].Race.PRT = engine.PRTInterstellarTraveler
	fi := l.fleet(0, origin, "Freighter", 2)
	l.g.Fleets[fi].Cargo.Minerals = engine.Minerals{50, 0, 0}
	c := &count{}
	j := Jump(l.g, fi, at(380, 0), c)
	if j.Refused != GateOK || l.g.Fleets[fi].Cargo.Minerals[0] != 50 || c.n != 0 || l.g.Fleets[fi].Stacks[0].Count != 2 {
		t.Errorf("IT: %+v, %d draws", j, c.n)
	}
}

// Mixed fleets (LEGACY BUG, GT-001 H2, GT-002, GT-003): a 500 kT ship and
// a safe Laser DD through a 100 kT gate: the lost design counts twice
// against two designs, so the whole fleet goes; with the switch off the
// Laser DD survives.
func TestConfirmedGateMixedFleet(t *testing.T) {
	for _, legacy := range []bool{true, false} {
		l := gateLab(t, "Stargate 100/250", 100)
		heavy := l.g.Designs[l.designs["Laser DD"]]
		heavy.Mass = 500
		l.g.Designs = append(l.g.Designs, heavy)
		fi := l.fleet(0, origin, "Laser DD", 1)
		l.g.Fleets[fi].Stacks = append(l.g.Fleets[fi].Stacks, engine.Stack{Design: len(l.g.Designs) - 1, Count: 1})
		l.g.Rules.Legacy.GateMixedFleetLoss = legacy
		j := Jump(l.g, fi, at(100, 0), &script{})
		if j.FleetLost != legacy || len(l.g.Fleets[fi].Stacks) != 1 {
			t.Errorf("legacy %v: %+v, stacks %+v", legacy, j, l.g.Fleets[fi].Stacks)
		}
	}
}

// Damage with old damage, fuel and cargo after losses, and skipped
// designs (OBJECTS.md "Stargates", BINARY-ONLY; fits OB-021, GT-001).
func TestPredictionGateLosses(t *testing.T) {
	// 4 Laser DDs (armor 200) at 50% / u 100: D = 2, Dm = 40. 13%: the
	// first ship is destroyed (3 < 4) and rand(500) = 50 < 100 lowers D to
	// 1; avg = ⌊(26·3 + 40·1)/3⌋ = 39, u = 97.
	l := gateLab(t, "Stargate 100/250", 380)
	fi := l.fleet(0, origin, "Laser DD", 4)
	l.g.Fleets[fi].Stacks[0].Damage = engine.Damage{Pct: 50, Units: 100}
	Jump(l.g, fi, at(380, 0), &script{3, 50, 50, 50, 50})
	if st := l.g.Fleets[fi].Stacks[0]; st.Count != 3 || st.Damage != (engine.Damage{Pct: 100, Units: 97}) {
		t.Errorf("damaged design: %+v", st)
	}
	// Nw + Dm ≥ A: D more ships go. u 450: Dm 180, 26 + 180 ≥ 200, so 2
	// of 4 are destroyed; avg = ⌊(26·2 + 180·2)/2⌋ = 206.
	l = gateLab(t, "Stargate 100/250", 380)
	fi = l.fleet(0, origin, "Laser DD", 4)
	l.g.Fleets[fi].Stacks[0].Damage = engine.Damage{Pct: 50, Units: 450}
	j := Jump(l.g, fi, at(380, 0), &script{50, 50, 50, 50})
	if st := l.g.Fleets[fi].Stacks[0]; st.Count != 2 || j.Designs[0].Destroyed != 2 || st.Damage.Units != 515 {
		t.Errorf("extra destruction: %+v %+v", j, st)
	}
	// Two of three lost from 100 fuel keeps 34.
	l = gateLab(t, "Stargate 100/250", 380)
	fi = l.fleet(0, origin, "Laser DD", 3)
	l.g.Fleets[fi].Fuel = 100
	Jump(l.g, fi, at(380, 0), &script{0, 0, 50})
	if f := l.g.Fleets[fi]; f.Stacks[0].Count != 1 || f.Fuel != 34 {
		t.Errorf("two lost: %+v", f)
	}
	// A design at 0% makes no draws and takes no damage.
	l = gateLab(t, "Stargate 100/250", 200)
	fi = l.fleet(0, origin, "Laser DD", 3)
	c := &count{}
	if j := Jump(l.g, fi, at(200, 0), c); c.n != 0 || j.Designs[0].Pct != 0 || l.g.Fleets[fi].Stacks[0].Damage != (engine.Damage{}) {
		t.Errorf("0%%: %+v, %d draws", j, c.n)
	}
	// Cargo lost by capacity: half of 20 kT, ⌊c·10/20⌋ each, then 1 kT
	// from ironium.
	cg := engine.Cargo{Minerals: engine.Minerals{10, 7, 3}}
	loseShare(&cg, 1, 2)
	if cg != (engine.Cargo{Minerals: engine.Minerals{4, 4, 2}}) {
		t.Errorf("cargo share: %+v", cg)
	}
}

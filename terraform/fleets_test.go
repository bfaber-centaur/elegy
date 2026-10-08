package terraform

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

type lab struct {
	g       *engine.Game
	designs map[string]int
}

// newLab is two players with the KX-002 habitat, friends of themselves
// and enemies of each other, one planet per call to l.planet.
func newLab(t *testing.T) *lab {
	t.Helper()
	g := &engine.Game{Players: []engine.Player{
		{Race: race(), Relations: []engine.Relation{engine.RelationFriend, engine.RelationEnemy}},
		{Race: race(), Relations: []engine.Relation{engine.RelationEnemy, engine.RelationFriend}},
	}}
	l := &lab{g: g, designs: map[string]int{}}
	cat := engine.Components()
	add := func(name, hull string, fills ...engine.SlotFill) {
		d, err := cat.NewDesign(name, hull, fills)
		if err != nil {
			t.Fatal(err)
		}
		l.designs[name] = len(g.Designs)
		g.Designs = append(g.Designs, d)
	}
	hump := engine.SlotFill{Slot: 0, Part: "Long Hump 6", Count: 1}
	add("Midget Miner", "Midget Miner", hump, engine.SlotFill{Slot: 1, Part: "Robo-Midget Miner", Count: 2})
	add("Adjuster", "Midget Miner", hump, engine.SlotFill{Slot: 1, Part: "Orbital Adjuster", Count: 2})
	add("Super", "Midget Miner", hump, engine.SlotFill{Slot: 1, Part: "Robo-Super-Miner", Count: 2})
	add("Mini-Miner", "Mini-Miner", hump, engine.SlotFill{Slot: 2, Part: "Robo-Mini-Miner", Count: 1}, engine.SlotFill{Slot: 3, Part: "Robo-Mini-Miner", Count: 1})
	add("Robo", "Midget Miner", hump, engine.SlotFill{Slot: 1, Part: "Robo-Miner", Count: 2})
	return l
}

func (l *lab) planet(owner int, env [3]int, conc [3]int) int {
	p := engine.Planet{ID: len(l.g.Planets), Pos: engine.Point{X: 100 * (len(l.g.Planets) + 1), Y: 100}, Owner: owner, Env: env, OrigEnv: env}
	for k := range conc {
		p.Deposits[k].Concentration = conc[k]
	}
	l.g.Planets = append(l.g.Planets, p)
	return len(l.g.Planets) - 1
}

func (l *lab) fleet(owner, pi int, design string, n int) int {
	f := engine.Fleet{ID: len(l.g.Fleets) + 1, Number: len(l.g.Fleets), Owner: owner, Pos: l.g.Planets[pi].Pos,
		Stacks: []engine.Stack{{Design: l.designs[design], Count: n}}}
	l.g.Fleets = append(l.g.Fleets, f)
	return len(l.g.Fleets) - 1
}

// script is an engine.Rand returning fixed values, then n − 1.
type script []int

func (s *script) Intn(n int) int {
	if len(*s) == 0 {
		return n - 1
	}
	v := (*s)[0]
	*s = (*s)[1:]
	return v
}

// count is an engine.Rand that never hits and counts its draws.
type count struct{ n int }

func (c *count) Intn(n int) int { c.n++; return n - 1 }

// Mining rates (CONFIRMED CS-003-B X1/X2, KB-1A).
func TestConfirmedMiningRate(t *testing.T) {
	l := newLab(t)
	pi := l.planet(engine.NoOwner, [3]int{50, 50, 50}, [3]int{100, 100, 100})
	if got := MiningRate(l.g, &l.g.Fleets[l.fleet(0, pi, "Midget Miner", 1)]); got != 10 {
		t.Errorf("two Robo-Midget: %d", got)
	}
	if got := MiningRate(l.g, &l.g.Fleets[l.fleet(0, pi, "Adjuster", 1)]); got != 0 {
		t.Errorf("two Orbital Adjusters: %d", got)
	}
	// KB-1A's 4,320 points (2 Robo-Super, 54 a ship, × 80) are capped
	// at 4,000.
	if got := MiningRate(l.g, &l.g.Fleets[l.fleet(0, pi, "Super", 80)]); got != MaxMiningRate {
		t.Errorf("KB-1A fleet: %d", got)
	}
}

// Remote mining output (KERNEL.md "Remote mining"; CONFIRMED CS-003-B X1,
// X2, T-35, KB-1A, KB-1B).
func TestConfirmedRemoteMine(t *testing.T) {
	l := newLab(t)
	x2 := l.planet(engine.NoOwner, [3]int{75, 47, 62}, [3]int{100, 100, 100})
	x1 := l.planet(engine.NoOwner, [3]int{88, 82, 1}, [3]int{100, 100, 100})
	kb := l.planet(engine.NoOwner, [3]int{50, 50, 50}, [3]int{68, 78, 76})
	t35 := l.planet(engine.NoOwner, [3]int{50, 50, 50}, [3]int{100, 50, 25})
	owned := l.planet(1, [3]int{50, 50, 50}, [3]int{100, 100, 100})
	own := l.planet(0, [3]int{50, 50, 50}, [3]int{100, 100, 100})
	cases := []struct {
		name  string
		fi    int
		gain  engine.Minerals
		mined bool
	}{
		{"CS-003-B-X2", l.fleet(0, x2, "Midget Miner", 1), engine.Minerals{10, 10, 10}, true},
		{"CS-003-B-X1", l.fleet(0, x1, "Adjuster", 1), engine.Minerals{}, false},
		{"KB-1A cap", l.fleet(0, kb, "Super", 80), engine.Minerals{2720, 3120, 3040}, true},
		// T-35: 24 robot points at 100/50/25 mined 24/12/6.
		{"T-35 unowned", l.fleet(0, t35, "Robo", 1), engine.Minerals{24, 12, 6}, true},
		{"T-35 another player's planet", l.fleet(0, owned, "Midget Miner", 1), engine.Minerals{}, false},
		{"T-35 own non-AR planet", l.fleet(0, own, "Midget Miner", 1), engine.Minerals{}, false},
	}
	for _, c := range cases {
		rng := &count{}
		before := l.g.Planets[planetAt(l.g, l.g.Fleets[c.fi].Pos)].Surface
		gain, mined := RemoteMine(l.g, c.fi, rng)
		after := l.g.Planets[planetAt(l.g, l.g.Fleets[c.fi].Pos)].Surface
		if gain != c.gain || mined != c.mined || after != (engine.Minerals{before[0] + gain[0], before[1] + gain[1], before[2] + gain[2]}) {
			t.Errorf("%s: gain %v mined %v", c.name, gain, mined)
		}
		if !c.mined && rng.n != 0 {
			t.Errorf("%s: %d draws without mining", c.name, rng.n)
		}
	}
	// Depletion follows planetary mining: KB-1A's 4,000 points at 68
	// lose concentration.
	if d := l.g.Planets[kb].Deposits[0]; d.Concentration >= 68 && d.Fraction == 0 {
		t.Errorf("KB-1A deposit not depleted: %+v", d)
	}
}

// An Alternate Reality planet is mined by its owner's own miners, and
// only theirs (CONFIRMED KB-1B; another player's miners ASSUMPTION R1).
func TestConfirmedRemoteMineAR(t *testing.T) {
	l := newLab(t)
	l.g.Players[0].Race.PRT = engine.PRTAlternateReality
	ar := l.planet(0, [3]int{50, 50, 50}, [3]int{100, 100, 100})
	own := l.fleet(0, ar, "Midget Miner", 1)
	other := l.fleet(1, ar, "Midget Miner", 1)
	if gain, ok := RemoteMine(l.g, own, &count{}); !ok || gain != (engine.Minerals{10, 10, 10}) {
		t.Errorf("own miners at AR planet: %v %v", gain, ok)
	}
	if gain, ok := RemoteMine(l.g, other, &count{}); ok || gain != (engine.Minerals{}) {
		t.Errorf("other player's miners at AR planet: %v %v", gain, ok)
	}
}

// SL-starbases year 2 (parity regression; rule CONFIRMED KB-1B): a
// Mini-Miner with two Robo-Mini-Miners (8 points) mines its owner's AR
// planet 4 at 62/10/87 as its own step: 496, 80 and 696 hundredths, so
// 4/0/6 kT plus the random +1 on each remainder, 5/1/7 when all three
// draws hit.
func TestConfirmedRemoteMineSLStarbases(t *testing.T) {
	l := newLab(t)
	l.g.Players[0].Race.PRT = engine.PRTAlternateReality
	pi := l.planet(0, [3]int{50, 50, 50}, [3]int{62, 10, 87})
	fi := l.fleet(0, pi, "Mini-Miner", 1)
	if m := MiningRate(l.g, &l.g.Fleets[fi]); m != 8 {
		t.Fatalf("rate %d", m)
	}
	rng := script{95, 79, 95}
	if gain, ok := RemoteMine(l.g, fi, &rng); !ok || gain != (engine.Minerals{5, 1, 7}) || len(rng) != 0 {
		t.Errorf("hits: %v %v", gain, ok)
	}
	l.g.Planets[pi].Deposits = [3]engine.Deposit{{Concentration: 62}, {Concentration: 10}, {Concentration: 87}}
	if gain, _ := RemoteMine(l.g, fi, &script{96, 80, 96}); gain != (engine.Minerals{4, 0, 6}) {
		t.Errorf("misses: %v", gain)
	}
}

// Orbital Adjusters (KERNEL.md "Terraforming"; CONFIRMED KX-005 T0–T2):
// planet 60/60/60, fleet owner's reach gravity 11, temperature 7,
// radiation 3.
func TestConfirmedAdjusters(t *testing.T) {
	setup := func(fleetOwner int, starbase bool) (*lab, int) {
		l := newLab(t)
		l.g.Players[fleetOwner].Research.Levels = levels(5, 1, 10, 0, 0, 3)
		pi := l.planet(0, [3]int{60, 60, 60}, [3]int{60, 60, 60})
		l.g.Planets[pi].HasStarbase = starbase
		l.fleet(fleetOwner, pi, "Adjuster", 1) // two adjusters: two clicks
		return l, pi
	}
	// Own planet, with or without a starbase: two clicks → 60/58/60.
	for _, sb := range []bool{false, true} {
		l, pi := setup(0, sb)
		adj := Adjust(l.g)
		if p := l.g.Planets[pi]; p.Env != [3]int{60, 58, 60} || len(adj) != 1 || adj[0].Hostile || !adj[0].HabChanged {
			t.Errorf("own, starbase %v: %v %+v", sb, p.Env, adj)
		}
	}
	// Enemy without a starbase: gravity toward 71 scores 137 against 67;
	// two clicks → 62/60/60.
	l, pi := setup(1, false)
	// (With this habitat temperature toward 67 scores 86 and radiation
	// toward 63 scores 67.)
	if s := [3]int{score(race(), [3]int{60, 60, 60}, Gravity, 71), score(race(), [3]int{60, 60, 60}, Temperature, 67), score(race(), [3]int{60, 60, 60}, Radiation, 63)}; s != [3]int{137, 86, 67} {
		t.Errorf("hostile scores %v", s)
	}
	adj := Adjust(l.g)
	if p := l.g.Planets[pi]; p.Env != [3]int{62, 60, 60} || len(adj) != 1 || !adj[0].Hostile || adj[0].Axes[0] != Gravity {
		t.Errorf("enemy: %v %+v", p.Env, adj)
	}
	// Enemy at a planet with a starbase: nothing.
	l, pi = setup(1, true)
	if adj := Adjust(l.g); adj != nil || l.g.Planets[pi].Env != [3]int{60, 60, 60} {
		t.Errorf("enemy at a starbase: %+v", adj)
	}
	// A friend improves.
	l, pi = setup(1, true)
	l.g.Players[1].Relations[0] = engine.RelationFriend
	if Adjust(l.g); l.g.Planets[pi].Env != [3]int{60, 58, 60} {
		t.Errorf("friend: %v", l.g.Planets[pi].Env)
	}
	// Unowned planets are not adjusted (CS-003-B X1).
	l = newLab(t)
	pi = l.planet(engine.NoOwner, [3]int{88, 82, 1}, [3]int{88, 82, 1})
	l.fleet(0, pi, "Adjuster", 1)
	if adj := Adjust(l.g); adj != nil || l.g.Planets[pi].Env != [3]int{88, 82, 1} {
		t.Errorf("unowned: %+v", adj)
	}
}

// PP packet terraforming (OBJECTS.md "Impact"; axes CONFIRMED OB-029-T1..T3,
// draws BINARY-ONLY).
func TestPredictionPacketTerraform(t *testing.T) {
	l := newLab(t)
	l.g.Players[1].Race.PRT = engine.PRTPacketPhysics
	l.g.Players[1].Research.Levels = levels(10, 10, 10, 0, 0, 3) // ±11 on every axis
	pi := l.planet(engine.NoOwner, [3]int{70, 70, 70}, [3]int{70, 70, 70})
	// 1000 kT of ironium uncaught: ten chunks; successes on chunks 1, 4
	// and 7, the second permanent.
	rng := script{0, 5, 199, 199, 0, 0, 199, 199, 0, 5, 199, 199, 199}
	r := PacketTerraform(l.g, pi, 1, Uncaught(engine.Minerals{1000, 0, 0}, 0), &rng)
	p := l.g.Planets[pi]
	if r.Successes != [3]int{3, 0, 0} || r.Permanent != [3]int{1, 0, 0} || p.OrigEnv != [3]int{69, 70, 70} || p.Env != [3]int{67, 70, 70} {
		t.Errorf("ironium: %+v env %v orig %v", r, p.Env, p.OrigEnv)
	}
	if len(rng) != 0 {
		t.Errorf("draws left: %v", rng)
	}
	// The move stops at the reach around the original value: 59 of 70
	// with ±11, and at the PP player's ideal.
	pi = l.planet(engine.NoOwner, [3]int{60, 52, 70}, [3]int{})
	l.g.Planets[pi].OrigEnv = [3]int{70, 52, 70}
	all := script{}
	for range 13 {
		all = append(all, 0, 9)
	}
	r = PacketTerraform(l.g, pi, 1, engine.Minerals{1000, 300, 0}, &all)
	if p := l.g.Planets[pi]; p.Env != [3]int{59, 50, 70} || r.Moved != [3]int{-1, -2, 0} || r.Successes != [3]int{10, 3, 0} {
		t.Errorf("limits: %v %+v", p.Env, r)
	}
	// A partial last chunk: 250 kT is draws < 100, < 100, < 50.
	pi = l.planet(engine.NoOwner, [3]int{70, 70, 70}, [3]int{70, 70, 70})
	r = PacketTerraform(l.g, pi, 1, engine.Minerals{0, 0, 250}, &script{99, 9, 99, 9, 50})
	if r.Successes != [3]int{0, 0, 2} {
		t.Errorf("partial chunk: %+v", r)
	}
	// A catcher's share is not terraformed: q = 490 leaves 510 of 1000.
	if u := Uncaught(engine.Minerals{1000, 300, 0}, 490); u != (engine.Minerals{510, 153, 0}) {
		t.Errorf("uncaught %v", u)
	}
	// An immune axis moves toward 99 from an original of 50 or more, the
	// current value by half the count.
	l.g.Players[1].Race.Env[Gravity] = engine.EnvRange{Immune: true}
	pi = l.planet(engine.NoOwner, [3]int{70, 70, 70}, [3]int{70, 70, 70})
	r = PacketTerraform(l.g, pi, 1, engine.Minerals{300, 0, 0}, &script{0, 0, 0, 9, 0, 9})
	if p := l.g.Planets[pi]; p.Env != [3]int{71, 70, 70} || p.OrigEnv != [3]int{71, 70, 70} {
		t.Errorf("immune: %v %v %+v", p.Env, p.OrigEnv, r)
	}
	// Original 50 goes toward 99; a permanent success moves the original
	// too (OBJECTS.md "PP terraforming").
	pi = l.planet(engine.NoOwner, [3]int{50, 70, 70}, [3]int{})
	l.g.Planets[pi].OrigEnv = [3]int{50, 70, 70}
	PacketTerraform(l.g, pi, 1, engine.Minerals{300, 0, 0}, &script{0, 0, 0, 9, 0, 9})
	if p := l.g.Planets[pi]; p.Env != [3]int{51, 70, 70} || p.OrigEnv != [3]int{51, 70, 70} {
		t.Errorf("immune at 50: %v %v", p.Env, p.OrigEnv)
	}
	// Below 50 goes toward 1.
	pi = l.planet(engine.NoOwner, [3]int{40, 70, 70}, [3]int{})
	l.g.Planets[pi].OrigEnv = [3]int{49, 70, 70}
	PacketTerraform(l.g, pi, 1, engine.Minerals{200, 0, 0}, &script{0, 9, 0, 9})
	if p := l.g.Planets[pi]; p.Env != [3]int{39, 70, 70} {
		t.Errorf("immune below 50: %v", p.Env)
	}
	// Gated: with every other axis at its limit (the ideal), the current
	// value does not move; the permanent move still does.
	pi = l.planet(engine.NoOwner, [3]int{70, 50, 50}, [3]int{})
	l.g.Planets[pi].OrigEnv = [3]int{70, 50, 50}
	r = PacketTerraform(l.g, pi, 1, engine.Minerals{200, 0, 0}, &script{0, 0, 0, 9})
	if p := l.g.Planets[pi]; p.Env != [3]int{70, 50, 50} || p.OrigEnv != [3]int{71, 50, 50} || r.Moved != [3]int{} {
		t.Errorf("immune, no limit elsewhere: %v %v %+v", p.Env, p.OrigEnv, r)
	}
}

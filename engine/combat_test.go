package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

// Combat tests follow stars-elegy docs/COMBAT.md statuses (see
// kernel_test.go): TestConfirmed* vectors come from the CB-000..CB-019
// records in PARITY.md "Combat"; TestPrediction* pin BINARY-ONLY rules.
//
// Elegy has no part catalogue, so the parts below carry the values the
// records name (Laser 10 damage range 1, Beta Torpedo 12 damage 45%,
// Jihad 85 damage 20%, Frigate armor 45 initiative 4).

var (
	tLaser   = Part{Name: "Laser", Kind: PartBeam, Damage: 10, Range: 1, Initiative: 9}
	tPhaser  = Part{Name: "Colloidal Phaser", Kind: PartBeam, Damage: 26, Range: 3, Initiative: 4}
	tGatling = Part{Name: "Gatling Gun", Kind: PartBeam, Damage: 13, Range: 2, Initiative: 12, Gatling: true}
	tSapper  = Part{Name: "Pulsed Sapper", Kind: PartBeam, Damage: 82, Range: 3, Initiative: 14, Sapper: true}
	tBeta    = Part{Name: "Beta Torpedo", Kind: PartTorpedo, Damage: 12, Range: 4, Accuracy: 45}
	tJihad   = Part{Name: "Jihad Missile", Kind: PartTorpedo, Damage: 85, Range: 5, Accuracy: 20, Missile: true}
	tBSC     = Part{Name: "Battle Super Computer", Kind: PartElectrical, Computer: 30, Initiative: 2}
	tJammer  = func(f int) Part { return Part{Name: "Jammer", Kind: PartElectrical, Jammer: f} }
	tFrigate = Hull{Name: "Frigate", Mass: 8, Armor: 45, Initiative: 4}
)

// testDesign is a design on a hull with the given slots.
func testDesign(h Hull, cost int, slots ...Slot) Design {
	h.Cost.Resources = cost
	return Design{Name: h.Name, Mass: h.Mass, Hull: h, Slots: slots}
}

// testBattle is a two-player battle in which each player attacks the
// other, with one token per design given, all on (0,0) unless moved.
type testBattle struct {
	*battle
}

func newTestBattle(rng Rand, designs []Design, toks ...*token) testBattle {
	g := &Game{Players: make([]Player, 2), Designs: designs}
	sets := attackSets{{1: true}, {0: true}}
	b := &battle{g: g, rng: rng, loc: location{planet: -1}, sets: sets, players: []int{0, 1}, involved: 2,
		tokens: toks, killed: map[int]bool{}, in: map[int]bool{0: true, 1: true}}
	return testBattle{b}
}

// tok makes a ship token of design i of designs for player p.
func tok(designs []Design, i, p, ships int) *token {
	t := tokenValues(designs[i], Race{}, false, designCost(designs[i], Race{}, [NumFields]int{}))
	t.player, t.fleet, t.stack, t.planet, t.design, t.ships = p, -1, 0, -1, i, ships
	t.tactic, t.primary, t.secondary = TacticMaximizeDamage, TargetAny, TargetAny
	return &t
}

func TestConfirmedStartSquares(t *testing.T) {
	// CB-001..CB-019: player 0 on (1,4), player 1 on (8,5).
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1})}
	g := &Game{
		Players: []Player{
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEnemies}}, Relations: []Relation{RelationFriend, RelationEnemy}},
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEnemies}}, Relations: []Relation{RelationEnemy, RelationFriend}},
		},
		Designs: d,
		Fleets: []Fleet{
			{ID: 1, Owner: 0, Pos: Point{5, 5}, Stacks: []Stack{{Design: 0, Count: 1}}},
			{ID: 2, Owner: 1, Pos: Point{5, 5}, Stacks: []Stack{{Design: 0, Count: 1}}},
		},
	}
	loc := g.locations()[0]
	sets, inv, n := g.whoFights(loc, locationHistory{})
	b := &battle{g: g, rng: &seqRand{}, loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}}
	b.setup(map[int]bool{})
	for _, tk := range b.tokens {
		want := [2]int{1, 4}
		if tk.player == 1 {
			want = [2]int{8, 5}
		}
		if got := [2]int{tk.x, tk.y}; got != want {
			t.Errorf("player %d starts on %v, want %v", tk.player, got, want)
		}
	}
}

func TestConfirmedTokenValues(t *testing.T) {
	// Laser Frigate: initiative 4 + Laser 9 = weapon initiative 13, armor 45.
	lf := tokenValues(testDesign(tFrigate, 0, Slot{tLaser, 1}), Race{}, false, Cost{})
	if lf.initiative != 4 || lf.weapons[0].init != 13 || lf.armor != 45 {
		t.Errorf("Laser Frigate: init %d weapon init %d armor %d, want 4 13 45", lf.initiative, lf.weapons[0].init, lf.armor)
	}
	// Battle Super Computer: computer 30%, initiative +2.
	bc := tokenValues(testDesign(tFrigate, 0, Slot{tBeta, 1}, Slot{tBSC, 1}), Race{}, false, Cost{})
	if bc.computer != 30 || bc.initiative != 6 {
		t.Errorf("BSC: computer %d initiative %d, want 30 6", bc.computer, bc.initiative)
	}
	// Jammer 20 and Jammer 50 (factors 80, 50).
	for _, c := range []struct{ f, want int }{{80, 20}, {50, 50}} {
		j := tokenValues(testDesign(tFrigate, 0, Slot{tJammer(c.f), 1}), Race{}, false, Cost{})
		if j.jammer != c.want {
			t.Errorf("jammer factor %d: %d%%, want %d%%", c.f, j.jammer, c.want)
		}
	}
	// Capacitors compound: an Energy (10) and a Flux (20) Capacitor → 132%.
	cp := tokenValues(testDesign(tFrigate, 0, Slot{Part{Capacitor: 10}, 1}, Slot{Part{Capacitor: 20}, 1}), Race{}, false, Cost{})
	if cp.capacitor != 132 {
		t.Errorf("capacitor %d%%, want 132%%", cp.capacitor)
	}
	// Beam Deflector 90%.
	df := tokenValues(testDesign(tFrigate, 0, Slot{Part{Deflector: true}, 1}), Race{}, false, Cost{})
	if df.deflector != 90 {
		t.Errorf("deflector %d%%, want 90%%", df.deflector)
	}
}

func TestConfirmedRegeneratingShields(t *testing.T) {
	// CB-007: 2 Mole-skin shields → 70; a Destroyer (armor 200) with 2
	// Tritanium (50 each) → armor 250.
	rs := Race{}
	rs.LRT.RegeneratingShields = true
	mole := Part{Name: "Mole-skin Shield", Kind: PartShield, Shield: 25}
	trit := Part{Name: "Tritanium", Kind: PartArmor, Armor: 50}
	destroyer := Hull{Name: "Destroyer", Mass: 30, Armor: 200, Initiative: 3}
	tk := tokenValues(testDesign(destroyer, 0, Slot{mole, 2}, Slot{trit, 2}), rs, false, Cost{})
	if tk.shield != 70 || tk.armor != 250 {
		t.Errorf("RS: shield %d armor %d, want 70 250", tk.shield, tk.armor)
	}
	// CB-008: +10% of the maximum per later round, none once at 0.
	tb := newTestBattle(&seqRand{}, nil, &token{shield: 20, maxShield: 70, regen: true, ships: 1}, &token{shield: 0, maxShield: 70, regen: true, ships: 1})
	tb.regenerate()
	if tb.tokens[0].shield != 27 || tb.tokens[1].shield != 0 {
		t.Errorf("regen: %d %d, want 27 0", tb.tokens[0].shield, tb.tokens[1].shield)
	}
}

func TestConfirmedEnergyDampener(t *testing.T) {
	// CB-002 C8: speed code 2 → 0 with a dampener in the battle.
	engine := Engine{Name: "Test", Fuel: [11]int{0, 0, 0, 0, 0, 0, 200, 200, 200, 200, 200}} // w = 5
	d := testDesign(tFrigate, 10, Slot{tLaser, 1})
	d.Engine, d.Engines = engine, 1
	if s := speedCode(d, Race{}, d.Mass, false); s != 1 {
		t.Fatalf("speed code %d, want 1", s)
	}
	d.Slots = append(d.Slots, Slot{Part{Thrust: 1}, 1})
	if s := speedCode(d, Race{}, d.Mass, false); s != 2 {
		t.Fatalf("speed code %d, want 2", s)
	}
	damp := d
	damp.Slots = append(append([]Slot(nil), d.Slots...), Slot{Part{Dampener: true}, 1})
	designs := []Design{d, damp}
	g := &Game{
		Players: []Player{
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEveryone}}},
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEveryone}}},
		},
		Designs: designs,
		Fleets: []Fleet{
			{ID: 1, Owner: 0, Stacks: []Stack{{Design: 0, Count: 1}}},
			{ID: 2, Owner: 1, Stacks: []Stack{{Design: 1, Count: 1}}},
		},
	}
	loc := g.locations()[0]
	sets, inv, n := g.whoFights(loc, locationHistory{})
	b := &battle{g: g, rng: &seqRand{}, loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}}
	b.setup(map[int]bool{})
	for _, tk := range b.tokens {
		if tk.speed != 0 {
			t.Errorf("token of player %d: speed %d, want 0", tk.player, tk.speed)
		}
	}
}

func TestConfirmedMovesPerRound(t *testing.T) {
	// CB-000..CB-008: speed code 1 moves 1,1,0,1; code 5 moves 2,2,1,2.
	for _, c := range []struct {
		s    int
		want []int
	}{{1, []int{1, 1, 0, 1}}, {5, []int{2, 2, 1, 2}}} {
		var got []int
		for r := range 4 {
			got = append(got, movesInRound(c.s, r))
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("speed %d: %v, want %v", c.s, got, c.want)
		}
	}
}

func TestConfirmedDisengageLeavesOnEighthMove(t *testing.T) {
	// P-10: a disengaging token leaves the board on its 8th move.
	d := []Design{testDesign(tFrigate, 10)}
	runner := tok(d, 0, 0, 1)
	runner.tactic, runner.counter = TacticDisengage, 7
	runner.x, runner.y = 1, 4
	enemy := tok(d, 0, 1, 1)
	enemy.x, enemy.y = 8, 5
	tb := newTestBattle(&seqRand{}, d, runner, enemy)
	for move := 1; move <= 8; move++ {
		runner.moves = 1
		tb.step(runner)
		if left := runner.left; left != (move == 8) {
			t.Fatalf("move %d: left = %v", move, left)
		}
	}
}

func TestConfirmedHitChance(t *testing.T) {
	// CB-001: Beta Torpedo (45%) with no computer, BSC, BSC vs Jammer 20,
	// BSC vs Jammer 50.
	for _, c := range []struct{ comp, jam, want int }{{0, 0, 45}, {30, 0, 62}, {30, 20, 51}, {30, 50, 36}} {
		if p := hitChance(45, c.comp, c.jam); p != c.want {
			t.Errorf("hitChance(45, %d, %d) = %d, want %d", c.comp, c.jam, p, c.want)
		}
	}
}

func TestConfirmedLargeSalvoHits(t *testing.T) {
	// CB-001: 202 Beta torpedoes hit exactly 90 / 125 / 103 / 72 times,
	// with no random draw.
	for _, c := range []struct {
		firer  []Slot
		jammer int
		want   int
	}{
		{[]Slot{{tBeta, 1}}, 0, 90},
		{[]Slot{{tBeta, 1}, {tBSC, 1}}, 0, 125},
		{[]Slot{{tBeta, 1}, {tBSC, 1}}, 80, 103},
		{[]Slot{{tBeta, 1}, {tBSC, 1}}, 50, 72},
	} {
		target := testDesign(Hull{Armor: 100000}, 10)
		if c.jammer > 0 {
			target.Slots = []Slot{{tJammer(c.jammer), 1}}
		}
		d := []Design{testDesign(tFrigate, 10, c.firer...), target}
		f, e := tok(d, 0, 0, 202), tok(d, 1, 1, 1)
		tb := newTestBattle(panicRand{}, d, f, e)
		tb.torpedoes(0, f.weapons[0])
		hits := 0
		for _, h := range tb.hits {
			hits += h.Hits
		}
		if hits != c.want {
			t.Errorf("%v vs jammer %d: %d hits, want %d", c.firer, c.jammer, hits, c.want)
		}
	}
}

// panicRand fails a test that expects no random draws.
type panicRand struct{}

func (panicRand) Intn(int) int { panic("unexpected random draw") }

func TestConfirmedMissileDoubleDamage(t *testing.T) {
	// CB-002 (P-20): 202 Jihads at 20% hit 40 times for 6800 on two
	// 3650-armor Hulks with no shields: one killed, the survivor 432/500.
	hulk := testDesign(Hull{Name: "Hulk", Armor: 3650}, 100)
	d := []Design{testDesign(tFrigate, 10, Slot{tJihad, 1}), hulk}
	f, e := tok(d, 0, 0, 202), tok(d, 1, 1, 2)
	tb := newTestBattle(panicRand{}, d, f, e)
	tb.torpedoes(0, f.weapons[0])
	if len(tb.hits) != 1 {
		t.Fatalf("%d hit records, want 1 (no miss record against 0 shields)", len(tb.hits))
	}
	h := tb.hits[0]
	if h.Hits != 40 || h.Armor != 6800 || h.Kills != 1 || e.ships != 1 || e.dmg != (Damage{Pct: 100, Units: 432}) {
		t.Errorf("hit %+v, survivor %d ships %+v; want 40 hits, 6800, 1 kill, 1 ship at 432", h, e.ships, e.dmg)
	}
}

func TestConfirmedOneKillPerMissile(t *testing.T) {
	// CB-009 K1: 202 Jihads at 20% (6800 damage, armor for 272 Small
	// Freighters) killed 202, leaving the survivors undamaged.
	sf := testDesign(Hull{Name: "Small Freighter", Armor: 25}, 10)
	d := []Design{testDesign(tFrigate, 10, Slot{tJihad, 1}), sf}
	f, e := tok(d, 0, 0, 202), tok(d, 1, 1, 1000)
	tb := newTestBattle(panicRand{}, d, f, e)
	tb.torpedoes(0, f.weapons[0])
	if e.ships != 798 || e.dmg.Pct != 0 {
		t.Errorf("survivors %d with %+v, want 798 undamaged", e.ships, e.dmg)
	}
}

func TestConfirmedTorpedoMissesOnShields(t *testing.T) {
	// CB-009 K8 (Q-14): 14 Beta misses on a shielded target do 21 to
	// shields, recorded before any hit.
	d := []Design{testDesign(tFrigate, 10, Slot{tBeta, 1}), testDesign(Hull{Armor: 1000}, 10, Slot{Part{Shield: 1000}, 1})}
	f, e := tok(d, 0, 0, 14), tok(d, 1, 1, 1)
	tb := newTestBattle(highRand{}, d, f, e) // every rand(100) is 99: all miss
	tb.torpedoes(0, f.weapons[0])
	if len(tb.hits) != 1 || tb.hits[0].Misses != 14 || tb.hits[0].Shield != 21 || e.shield != 979 {
		t.Errorf("hits %+v, shield %d; want 14 misses doing 21", tb.hits, e.shield)
	}
}

func TestConfirmedBeamDropoff(t *testing.T) {
	// CB-011..013, CB-016: a Laser Station (8 Lasers, 80) hits at distance
	// 2 for 64, the Laser's own range giving 80%. A Laser at distance 1
	// does 90%.
	station := testDesign(Hull{Name: "Space Station", Armor: 500, Initiative: 14, Starbase: true}, 100, Slot{tLaser, 8})
	target := testDesign(Hull{Armor: 10000}, 10)
	d := []Design{station, target, testDesign(tFrigate, 10, Slot{tLaser, 1})}
	sb := tokenValues(station, Race{}, true, Cost{})
	sb.player, sb.design, sb.ships, sb.fleet, sb.planet = 0, 0, 1, -1, 0
	sb.primary, sb.secondary = TargetAny, TargetAny
	e := tok(d, 1, 1, 1)
	e.x = 2
	tb := newTestBattle(panicRand{}, d, &sb, e)
	tb.beam(0, sb.weapons[0])
	if len(tb.hits) != 1 || tb.hits[0].Armor != 64 {
		t.Errorf("station hits %+v, want one hit of 64", tb.hits)
	}
	f := tok(d, 2, 0, 1)
	e2 := tok(d, 1, 1, 1)
	e2.x = 1
	tb = newTestBattle(panicRand{}, d, f, e2)
	tb.beam(0, f.weapons[0])
	if len(tb.hits) != 1 || tb.hits[0].Armor != 9 {
		t.Errorf("laser hits %+v, want one hit of 9", tb.hits)
	}
}

func TestConfirmedCarryRescaled(t *testing.T) {
	// CB-010 (Q-7): after a kill the next target gets R' = min(R−1,
	// R·L/dp), again through deflector and dropoff.
	weak := testDesign(Hull{Armor: 30}, 10)
	defl := testDesign(Hull{Armor: 10000}, 1, Slot{Part{Deflector: true}, 1})
	d := []Design{testDesign(tFrigate, 10, Slot{tPhaser, 1}), weak, defl}
	f := tok(d, 0, 0, 4) // R = 4 × 26 = 104
	a, c := tok(d, 1, 1, 1), tok(d, 2, 1, 1)
	tb := newTestBattle(panicRand{}, d, f, a, c)
	tb.beam(0, f.weapons[0])
	// First hit: dp 104 on the 30-armor ship (more attractive), leftover
	// 74. R' = min(103, 104·74/104) = 74, × 90% deflector = 66.
	if len(tb.hits) != 2 || tb.hits[0].Leftover != 74 || tb.hits[1].Armor != 66 {
		t.Errorf("hits %+v, want leftover 74 then 66", tb.hits)
	}
}

func TestConfirmedGatlingHitsEveryTarget(t *testing.T) {
	// CB-002/003/005 (P-13): one shot hits every eligible target with full
	// damage and no dropoff.
	target := testDesign(Hull{Armor: 1000}, 10)
	d := []Design{testDesign(tFrigate, 10, Slot{tGatling, 1}), target}
	f := tok(d, 0, 0, 1)
	a, c := tok(d, 1, 1, 1), tok(d, 1, 1, 2)
	a.x, c.x = 2, 1
	tb := newTestBattle(panicRand{}, d, f, a, c)
	tb.gatling(0, f.weapons[0])
	if len(tb.hits) != 2 || tb.hits[0].Armor != 13 || tb.hits[1].Armor != 13 {
		t.Errorf("hits %+v, want 13 on each", tb.hits)
	}
}

func TestConfirmedSapperShieldsOnly(t *testing.T) {
	// CB-002 (P-15): sappers hit only shields; no hit on unshielded ships.
	d := []Design{testDesign(tFrigate, 10, Slot{tSapper, 1}), testDesign(Hull{Armor: 100}, 1000), testDesign(Hull{Armor: 100}, 10, Slot{Part{Shield: 50}, 1})}
	f := tok(d, 0, 0, 1)
	bare, shielded := tok(d, 1, 1, 1), tok(d, 2, 1, 1)
	tb := newTestBattle(panicRand{}, d, f, bare, shielded)
	tb.beam(0, f.weapons[0])
	if len(tb.hits) != 1 || tb.hits[0].Target != 2 || tb.hits[0].Armor != 0 || shielded.shield != 0 || bare.dmg != (Damage{}) {
		t.Errorf("hits %+v", tb.hits)
	}
}

func TestConfirmedTargetChoice(t *testing.T) {
	// CB-009 K4–K7 (Q-6).
	plain := testDesign(Hull{Armor: 45}, 10)
	dear := testDesign(Hull{Armor: 45}, 20)
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), plain, dear}
	choose := func(targets ...*token) int {
		f := tok(d, 0, 0, 1)
		tb := newTestBattle(panicRand{}, d, append([]*token{f}, targets...)...)
		return tb.choose(f, f.weapons[0])
	}
	if got := choose(tok(d, 1, 1, 3), tok(d, 1, 1, 5)); got != 2 {
		t.Errorf("5 ships vs 3: chose %d, want 2", got)
	}
	if got := choose(tok(d, 1, 1, 1), tok(d, 2, 1, 1)); got != 2 {
		t.Errorf("dearer design: chose %d, want 2", got)
	}
	hurt := tok(d, 1, 1, 1)
	hurt.dmg = Damage{Pct: 100, Units: 100}
	if got := choose(tok(d, 1, 1, 1), hurt); got != 2 {
		t.Errorf("damaged stack: chose %d, want 2", got)
	}
	if got := choose(tok(d, 1, 1, 1), tok(d, 1, 1, 1)); got != 1 {
		t.Errorf("identical stacks: chose %d, want 1", got)
	}
}

func TestConfirmedStarbaseDamageSteps(t *testing.T) {
	// CB-014/015 (Q-5): an unarmed Space Station's armor went through 90,
	// 190, …, 490 per 500 under 100-damage hits, then died at the next.
	// (The first armor hit was 90 after the shields took the rest.)
	station := testDesign(Hull{Name: "Space Station", Armor: 500, Starbase: true}, 100)
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), station}
	sb := tokenValues(station, Race{}, true, Cost{})
	sb.player, sb.design, sb.ships, sb.fleet, sb.planet = 1, 1, 1, -1, -1
	sb.dmg = Damage{Pct: 100}
	tb := newTestBattle(panicRand{}, d, tok(d, 0, 0, 1), &sb)
	var got []int
	for i, dp := range []int{90, 100, 100, 100, 100, 100} {
		tb.damage(&sb, dp, 0, false, -1)
		if sb.dead {
			if i != 5 {
				t.Fatalf("destroyed at hit %d", i)
			}
			break
		}
		got = append(got, sb.dmg.Units)
	}
	if want := []int{90, 190, 290, 390, 490}; !reflect.DeepEqual(got, want) || !sb.dead {
		t.Errorf("units %v dead %v, want %v then destroyed", got, sb.dead, want)
	}
}

func TestConfirmedSalvage(t *testing.T) {
	// CB-001 B1: kill events of 3, 4 and 3 Small Freighters (4 Fe, 5 Ge
	// each) in deep space leave 10 Fe, 13 Ge.
	sf := testDesign(Hull{Name: "Small Freighter", Armor: 25}, 10)
	sf.Hull.Cost.Minerals = Minerals{4, 0, 5}
	d := []Design{sf}
	e := tok(d, 0, 1, 10)
	tb := newTestBattle(panicRand{}, d, e)
	for _, k := range []int{3, 4, 3} {
		tb.killEvent(e, k)
	}
	for _, add := range tb.pending {
		tb.addSalvage(add)
	}
	if !reflect.DeepEqual(tb.salvage, []Minerals{{10, 0, 13}}) {
		t.Errorf("deep space salvage %v, want one object [10 0 13]", tb.salvage)
	}
	// CB-011..013 S6/S7 (Q-13): six in one kill event at a planet give
	// 6 Fe / 8 Ge with a starbase, 4 / 5 without, and no salvage object.
	for _, c := range []struct {
		starbase bool
		want     Minerals
	}{{true, Minerals{6, 0, 8}}, {false, Minerals{4, 0, 5}}} {
		tb := newTestBattle(panicRand{}, d, e)
		tb.g.Planets = []Planet{{HasStarbase: c.starbase}}
		tb.loc.planet = 0
		tb.killEvent(e, 6)
		if got := tb.g.Planets[0].Surface; got != c.want || len(tb.pending) != 0 {
			t.Errorf("starbase %v: surface %v pending %v, want %v", c.starbase, got, tb.pending, c.want)
		}
	}
}

func TestConfirmedRepair(t *testing.T) {
	// CB-017 (Q-12): units repaired per turn by situation, pct kept.
	ft := testDesign(Hull{Name: "Fuel Transport", FuelTransport: true, RepairBonus: 25}, 10)
	sfx := testDesign(Hull{Name: "Super-Fuel Xport", FuelTransport: true, RepairBonus: 50}, 10)
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), ft, sfx}
	g := &Game{
		Players: make([]Player, 2),
		Designs: d,
		Planets: []Planet{
			{Pos: Point{10, 0}, Owner: 1},
			{Pos: Point{20, 0}, Owner: 0},
			{Pos: Point{30, 0}, Owner: 0, StarbaseHull: 1},
			{Pos: Point{40, 0}, Owner: 0, StarbaseHull: 2},
			{Pos: Point{50, 0}, Owner: 0, StarbaseHull: 3},
		},
	}
	at := []Point{{0, 0}, {10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}, {0, 0}, {0, 0}}
	extra := []int{-1, -1, -1, -1, -1, -1, 1, 2}
	want := []int{10, 15, 25, 40, 100, 100, 35, 60}
	for i, p := range at {
		st := []Stack{{Design: 0, Count: 2, Damage: Damage{Pct: 50, Units: 400}}}
		if extra[i] >= 0 {
			st = append(st, Stack{Design: extra[i], Count: 1})
		}
		g.Fleets = append(g.Fleets, Fleet{ID: i + 1, Owner: 0, Pos: p, Stacks: st})
	}
	repair(g, nil, battleResult{})
	for i, f := range g.Fleets {
		if got := f.Stacks[0].Damage; got != (Damage{Pct: 50, Units: 400 - want[i]}) {
			t.Errorf("fleet at %v (extra %d): %+v, want units %d pct 50", at[i], extra[i], got, 400-want[i])
		}
	}
}

// combatLabGame is a two-player deep-space game in which player 0's laser
// frigates meet player 1's frigates; the players are mutual enemies.
func combatLabGame(defender Design, attackers, defenders int) Game {
	eng := Engine{Name: "Test"} // every fuel entry 0: w = 9
	att := testDesign(tFrigate, 20, Slot{tLaser, 2})
	att.Engine, att.Engines = eng, 1
	defender.Engine, defender.Engines = eng, 1
	plan := []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Secondary: TargetAny, Attack: AttackEnemies}}
	race := pgRace()
	return Game{
		Year: 2400,
		Players: []Player{
			{Race: race, Plans: plan, Relations: []Relation{RelationFriend, RelationEnemy}},
			{Race: race, Plans: plan, Relations: []Relation{RelationEnemy, RelationFriend}},
		},
		Designs: []Design{att, defender},
		Fleets: []Fleet{
			{ID: 1, Owner: 0, Pos: Point{100, 100}, Stacks: []Stack{{Design: 0, Count: attackers}}},
			{ID: 2, Owner: 1, Pos: Point{100, 100}, Stacks: []Stack{{Design: 1, Count: defenders}}},
		},
	}
}

func TestConfirmedTechFromBattleSameTurn(t *testing.T) {
	// CB-018 (Q-11): weapons 3, research 0%. Destroying frigates that need
	// weapons 4 raised weapons to 4 in the same turn for some random
	// streams and not others, with every accumulator 0.
	prey := testDesign(tFrigate, 20, Slot{tLaser, 1})
	prey.Slots[0].Part.TechReq[Weapons] = 4
	outcomes := map[int]bool{}
	for seed := int64(1); seed <= 40; seed++ {
		game := combatLabGame(prey, 6, 3)
		for p := range game.Players {
			for f := range NumFields {
				game.Players[p].Research.Levels[f] = MaxTechLevel
			}
		}
		game.Players[0].Research.Levels[Weapons] = 3
		res, err := GenerateTurn(game, nil, Jrc3(), rand.New(rand.NewSource(seed)))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range res.Game.Fleets {
			if f.Owner == 1 {
				t.Fatalf("seed %d: player 1's fleet survived", seed)
			}
		}
		r := res.Game.Players[0].Research
		if r.Accumulated != ([NumFields]int{}) || (r.Levels[Weapons] != 3 && r.Levels[Weapons] != 4) {
			t.Fatalf("seed %d: research %+v", seed, r)
		}
		outcomes[r.Levels[Weapons]] = true
	}
	if !outcomes[3] || !outcomes[4] {
		t.Errorf("outcomes %v, want both weapons 3 and 4 across streams", outcomes)
	}
}

func TestConfirmedOnlyFleetsStartBattles(t *testing.T) {
	// CB-002 C9/C10, CB-003/004 S2, CB-006: an armed starbase does not
	// start a battle with a visitor that attacks nobody.
	station := testDesign(Hull{Name: "Space Station", Armor: 500, Starbase: true}, 100, Slot{tLaser, 8})
	visitor := testDesign(tFrigate, 10)
	g := &Game{
		Players: []Player{
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEveryone}}, Relations: []Relation{RelationFriend, RelationEnemy}},
			{Relations: []Relation{RelationEnemy, RelationFriend}},
		},
		Designs: []Design{station, visitor},
		Planets: []Planet{{Pos: Point{5, 5}, Owner: 0, HasStarbase: true, StarbaseDesign: 0}},
		Fleets:  []Fleet{{ID: 1, Owner: 1, Pos: Point{5, 5}, Stacks: []Stack{{Design: 1, Count: 1}}}},
	}
	if _, inv, _ := g.whoFights(g.locations()[0], locationHistory{}); inv != nil {
		t.Errorf("lone starbase started a battle: %v", inv)
	}
}

func TestConfirmedStarbaseJoinsWithPlan0(t *testing.T) {
	// CB-011 (Q-1): player 1 sees player 0 as neutral; its armed fleet
	// attacks enemies. Player 0's armed station with plan 0 "enemies"
	// fights, and so do "everyone" and "player 1" when the previous
	// location had a battle (CB-012, CB-013: X = player 0). With plan 0
	// "nobody", or an unarmed station, there is no battle.
	laserStation := testDesign(Hull{Name: "Space Station", Armor: 500, Starbase: true}, 100, Slot{tLaser, 8})
	bare := testDesign(Hull{Name: "Space Station", Armor: 500, Starbase: true}, 100)
	frig := testDesign(tFrigate, 10, Slot{tLaser, 1})
	for _, c := range []struct {
		station Design
		attack  AttackWho
		battle  bool
	}{
		{laserStation, AttackEnemies, true},
		{laserStation, AttackEveryone, true},
		{laserStation, AttackPlayer, true},
		{laserStation, AttackNobody, false},
		{bare, AttackEnemies, false},
	} {
		g := &Game{
			Players: []Player{
				{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: c.attack, Player: 1}}, Relations: []Relation{RelationFriend, RelationEnemy}},
				{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEnemies}}, Relations: []Relation{RelationNeutral, RelationFriend}},
			},
			Designs: []Design{c.station, frig},
			Planets: []Planet{{Pos: Point{5, 5}, Owner: 0, HasStarbase: true, StarbaseDesign: 0}},
			Fleets:  []Fleet{{ID: 1, Owner: 1, Pos: Point{5, 5}, Stacks: []Stack{{Design: 1, Count: 1}}}},
		}
		_, inv, _ := g.whoFights(g.locations()[0], locationHistory{any: true, battle: true})
		if (inv != nil) != c.battle {
			t.Errorf("station armed %v, plan 0 attack %d: involved %v, want battle %v", c.station.armed(), c.attack, inv, c.battle)
		}
	}
}

func TestConfirmedStarbaseIsArmedTarget(t *testing.T) {
	// CB-011..013 S4/S5 (Q-4), LEGACY BUG: an unarmed station matches
	// "armed" and "any", not "unarmed".
	sb := tokenValues(testDesign(Hull{Name: "Space Station", Armor: 500, Starbase: true}, 100), Race{}, true, Cost{})
	if sb.matches(TargetUnarmed) || !sb.matches(TargetArmed) || !sb.matches(TargetAny) {
		t.Errorf("unarmed station class %d", sb.class)
	}
}

func TestPredictionBattleTurn(t *testing.T) {
	// A whole battle through GenerateTurn: deterministic for a given
	// stream, the stronger side wins, deep-space salvage is left, and the
	// fleets that fought get no repair.
	prey := testDesign(tFrigate, 20, Slot{tLaser, 1})
	prey.Hull.Cost.Minerals = Minerals{30, 0, 9}
	run := func() TurnResult {
		res, err := GenerateTurn(combatLabGame(prey, 6, 3), nil, Jrc3(), rand.New(rand.NewSource(7)))
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	a, b := run(), run()
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same stream gave different results")
	}
	if len(a.Game.Fleets) != 1 || a.Game.Fleets[0].Owner != 0 {
		t.Fatalf("fleets after battle: %+v", a.Game.Fleets)
	}
	if len(a.Game.Salvage) != 1 || a.Game.Salvage[0].Pos != (Point{100, 100}) || a.Game.Salvage[0].Minerals == (Minerals{}) {
		t.Errorf("salvage %+v", a.Game.Salvage)
	}
	battles := 0
	for _, e := range a.Events {
		if e.Kind == EventBattle {
			battles++
		}
	}
	if battles != 2 {
		t.Errorf("%d battle events, want one per player", battles)
	}
}

func TestPredictionTokenCap(t *testing.T) {
	// 256 tokens at most: 255/players stacks per player, then left-out
	// fleets added back while room remains.
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1})}
	g := &Game{Players: make([]Player, 2), Designs: d}
	var fleets []int
	for i := range 300 {
		owner := 0
		if i >= 200 {
			owner = 1
		}
		g.Fleets = append(g.Fleets, Fleet{ID: i, Owner: owner, Stacks: []Stack{{Design: 0, Count: 1}}})
		fleets = append(fleets, i)
	}
	b := &battle{g: g, loc: location{planet: -1}, players: []int{0, 1}}
	in, missed := b.capTokens(fleets)
	// Quota 127 each: 127 of player 0 and all 100 of player 1, then 28
	// more of player 0 up to 255.
	if len(in) != 255 || !reflect.DeepEqual(missed, []int{0}) {
		t.Errorf("%d fleets in, missed %v; want 255 and [0]", len(in), missed)
	}
}

func TestPredictionLegacyPlan0Recipient(t *testing.T) {
	// X is player 0 after a battle, else the owner of the previous
	// location's last fleet; at the first location it has no effect.
	if x, ok := legacyPlan0Recipient(1, locationHistory{any: true, battle: true, lastOwner: 1}); !ok || x != 0 {
		t.Errorf("after a battle: %d %v, want 0", x, ok)
	}
	if x, ok := legacyPlan0Recipient(0, locationHistory{any: true, lastOwner: 1}); !ok || x != 1 {
		t.Errorf("after no battle: %d %v, want 1", x, ok)
	}
	if _, ok := legacyPlan0Recipient(0, locationHistory{}); ok {
		t.Error("first location: want no recipient")
	}
}

// plan0Game is the CB-022 setup: player 0's Laser Station planet; player
// 1, who sees player 0 as neutral, has an armed fleet attacking "enemies"
// there and, when hauler is set, a lone hauler with a lower fleet number
// elsewhere.
func plan0Game(attack AttackWho, hauler bool) *Game {
	station := testDesign(Hull{Name: "Space Station", Armor: 500, Initiative: 14, Starbase: true}, 100, Slot{tLaser, 8})
	frig := testDesign(tFrigate, 10, Slot{tLaser, 1})
	g := &Game{
		Players: []Player{
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: attack, Player: 1}}, Relations: []Relation{RelationFriend, RelationEnemy}},
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Secondary: TargetAny, Attack: AttackEnemies}}, Relations: []Relation{RelationNeutral, RelationFriend}},
		},
		Designs: []Design{station, frig, testDesign(Hull{Name: "Hauler", Armor: 20}, 5)},
		Planets: []Planet{{ID: 1, Pos: Point{5, 5}, Owner: 0, HasStarbase: true, StarbaseDesign: 0}},
		Fleets:  []Fleet{{ID: 2, Owner: 1, Pos: Point{5, 5}, Stacks: []Stack{{Design: 1, Count: 5}}}},
	}
	if hauler {
		g.Fleets = append(g.Fleets, Fleet{ID: 1, Owner: 1, Pos: Point{50, 50}, Stacks: []Stack{{Design: 2, Count: 1}}})
	}
	return g
}

func TestConfirmedPlan0OnePlayerBattle(t *testing.T) {
	// CB-022 (R-10): after the lone hauler's location (no battle), plan 0
	// "player 1" and "everyone" give a one-player battle: two tokens, the
	// station on (4,4), the visitor on (1,4), no actions. Plan 0
	// "enemies" is an ordinary battle; without the hauler, "player 1"
	// gives no battle.
	for _, c := range []struct {
		attack   AttackWho
		hauler   bool
		battle   bool
		involved int
	}{
		{AttackPlayer, true, true, 1},
		{AttackEveryone, true, true, 1},
		{AttackEnemies, true, true, 2},
		{AttackPlayer, false, false, 0},
	} {
		g := plan0Game(c.attack, c.hauler)
		var prev locationHistory
		var b *battle
		for _, loc := range g.locations() {
			sets, inv, n := g.whoFights(loc, prev)
			prev = locationHistory{any: true, battle: inv != nil, lastOwner: g.Fleets[loc.fleets[len(loc.fleets)-1]].Owner}
			if inv != nil {
				b = &battle{g: g, rng: rand.New(rand.NewSource(1)), loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}}
			}
		}
		if (b != nil) != c.battle {
			t.Errorf("attack %d hauler %v: battle %v, want %v", c.attack, c.hauler, b != nil, c.battle)
			continue
		}
		if b == nil {
			continue
		}
		if b.involved != c.involved || len(b.players) != 2 {
			t.Errorf("attack %d: n %d, players %v; want n %d, players [0 1]", c.attack, b.involved, b.players, c.involved)
		}
		b.setup(map[int]bool{})
		if c.involved == 1 {
			for _, tk := range b.tokens {
				want := [2]int{1, 4}
				if tk.starbase {
					want = [2]int{4, 4}
				}
				if got := [2]int{tk.x, tk.y}; got != want {
					t.Errorf("attack %d: token of player %d on %v, want %v", c.attack, tk.player, got, want)
				}
			}
		}
		b.fight()
		if c.involved == 1 {
			if len(b.hits) != 0 {
				t.Errorf("attack %d: %d hits, want none", c.attack, len(b.hits))
			}
			for _, tk := range b.tokens {
				if !tk.starbase && (tk.x != 1 || tk.y != 4) {
					t.Errorf("attack %d: visitor moved to (%d,%d)", c.attack, tk.x, tk.y)
				}
			}
		} else if len(b.hits) == 0 {
			t.Errorf("attack %d: ordinary battle has no hits", c.attack)
		}
	}
}

func TestConfirmedTechAttemptLocation(t *testing.T) {
	// CB-021, CB-012: in a two-token, two-player battle at player 0's
	// planet, player 0 attempts (even having lost nothing) and the
	// attacker at another player's planet does not.
	d := []Design{testDesign(tFrigate, 10)}
	tb := newTestBattle(&seqRand{draws: []int{50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, Weapons, 50}}, d, tok(d, 0, 0, 1), tok(d, 0, 1, 1))
	tb.g.Planets = []Planet{{Owner: 0}}
	tb.loc.planet = 0
	tb.seen[Weapons] = 5
	ev := tb.techAttempts(map[int]bool{})
	if len(ev) != 1 || ev[0].Player != 0 || tb.g.Players[1].Research.Accumulated[Weapons] != 0 {
		t.Errorf("events %+v", ev)
	}
	if got := tb.g.Players[0].Research.Accumulated[Weapons]; got != ResearchLevelCost(1, 0, ResearchNormal, false) {
		t.Errorf("gain %d, want the next level's cost", got)
	}
}

func TestPredictionTechAttemptRules(t *testing.T) {
	// COMBAT.md "Tech from battle" (stars-elegy #31). countRand returns 0,
	// so each attempt stops after its first draw: draws = attempts.
	d := []Design{testDesign(tFrigate, 10)}
	setup := func() (testBattle, *countRand) {
		n := &countRand{}
		tb := newTestBattle(n, d, tok(d, 0, 2, 1), tok(d, 0, 3, 1))
		tb.g.Players = make([]Player, 4)
		tb.players, tb.involved = []int{2, 3}, 2
		// Players 0 and 1 have fleets at the location but are not in the
		// battle: observers = 0b11.
		tb.g.Fleets = []Fleet{{Owner: 0}, {Owner: 1}, {Owner: 2}, {Owner: 3}}
		tb.loc.fleets = []int{0, 1, 2, 3}
		return tb, n
	}

	// Deep space, n = 2: both participants attempt. LEGACY BUG: player 1
	// qualifies (1 AND 0b11 ≠ 0), player 0 never does.
	tb, n := setup()
	tb.techAttempts(map[int]bool{})
	if n.n != 3 {
		t.Errorf("%d attempts, want 3 (players 1, 2, 3)", n.n)
	}

	// n = 2: a wiped-out participant makes no attempt.
	tb, n = setup()
	tb.tokens[1].ships, tb.tokens[1].dead = 0, true
	tb.techAttempts(map[int]bool{})
	if n.n != 2 {
		t.Errorf("%d attempts, want 2 (players 1, 2)", n.n)
	}

	// n ≠ 2: every participant attempts, whatever it lost.
	tb, n = setup()
	tb.involved = 3
	tb.tokens[1].ships, tb.tokens[1].dead = 0, true
	tb.techAttempts(map[int]bool{})
	if n.n != 3 {
		t.Errorf("%d attempts, want 3", n.n)
	}
}

func TestPredictionStartSquareFlatTable(t *testing.T) {
	// Entry n(n−1)/2 + rank of the flattened table: n = 1 gives (4,4) then
	// (1,4); a third player in P with n = 2 runs into row 3.
	for _, c := range []struct{ n, r, x, y int }{{1, 0, 4, 4}, {1, 1, 1, 4}, {2, 1, 8, 5}, {2, 2, 4, 1}} {
		if got := startSquare(c.n, c.r); got != [2]int{c.x, c.y} {
			t.Errorf("startSquare(%d, %d) = %v, want (%d,%d)", c.n, c.r, got, c.x, c.y)
		}
	}
}

func TestPredictionDesignCost(t *testing.T) {
	laser := Part{Kind: PartBeam, Cost: Cost{Resources: 5, Minerals: Minerals{0, 6, 0}}}
	laser.TechReq[Weapons] = 1
	hull := Hull{Cost: Cost{Resources: 10, Minerals: Minerals{3, 0, 1}}}
	d := Design{Hull: hull, Slots: []Slot{{laser, 2}}}
	var lv [NumFields]int
	lv[Weapons] = 3
	// Hull: no requirement, m = lowest level 0: unchanged. Laser: m = 2,
	// d = 8%: 5 → 5 − round(0.4) = 5, 6 → 6 − round(0.48) = 6.
	if got := designCost(d, Race{}, lv); got != (Cost{Resources: 20, Minerals: Minerals{3, 12, 1}}) {
		t.Errorf("cost %+v", got)
	}
	// m = 19 levels: d = 75%: 5 → 5 − round(3.75) = 1; 6 → 6 − round(4.5) = 1.
	lv[Weapons] = 20
	if got := designCost(d, Race{}, lv); got != (Cost{Resources: 12, Minerals: Minerals{3, 2, 1}}) {
		t.Errorf("miniaturized cost %+v", got)
	}
	// War Monger weapons −¼ after miniaturization; IS +¼.
	lv[Weapons] = 1
	wm := Race{PRT: PRTWarMonger}
	if got := designCost(Design{Slots: []Slot{{Part{Kind: PartBeam, Cost: Cost{Resources: 40}, TechReq: laser.TechReq}, 1}}}, wm, lv); got.Resources != 30 {
		t.Errorf("WM cost %+v", got)
	}
	is := Race{PRT: PRTInnerStrength}
	if got := designCost(Design{Slots: []Slot{{Part{Kind: PartBeam, Cost: Cost{Resources: 40}, TechReq: laser.TechReq}, 1}}}, is, lv); got.Resources != 50 {
		t.Errorf("IS cost %+v", got)
	}
	// Bleeding Edge doubles a part at exactly its requirement.
	bet := Race{}
	bet.LRT.BleedingEdgeTech = true
	if got := designCost(Design{Slots: []Slot{{Part{Kind: PartBeam, Cost: Cost{Resources: 40}, TechReq: laser.TechReq}, 1}}}, bet, lv); got.Resources != 80 {
		t.Errorf("BET cost %+v", got)
	}
}

func TestPredictionCargoShare(t *testing.T) {
	// Losing 1 of 4 freighters (capacity 10 each) with cargo 7 Fe, 3 Bo,
	// 0 Ge, 2 colonists: moved = 12·10/40 = 3; shares 1, 0, 0, 0 by
	// truncation, then the remainder 2 goes 1 kT to Fe and 1 to Bo.
	fr := testDesign(Hull{Name: "Freighter", Armor: 25}, 10)
	fr.CargoCapacity = 10
	d := []Design{fr}
	e := tok(d, 0, 1, 3)
	e.fleet = 0
	tb := newTestBattle(panicRand{}, d, e)
	tb.g.Fleets = []Fleet{{Owner: 1, Cargo: Cargo{Minerals: Minerals{7, 3, 0}, Colonists: 2}, Stacks: []Stack{{Design: 0, Count: 4}}}}
	got := tb.cargoShare(e, 1)
	if got != (Minerals{2, 1, 0}) || tb.g.Fleets[0].Cargo != (Cargo{Minerals: Minerals{5, 2, 0}, Colonists: 2}) {
		t.Errorf("share %v, cargo left %+v", got, tb.g.Fleets[0].Cargo)
	}
}

func TestPredictionFuelShare(t *testing.T) {
	// COMBAT.md "Salvage" (stars-elegy #31): each kill event destroys
	// F · lost fuel capacity / capacity before, from the fuel held then.
	// 1 of 4 tankers (200 mg each) with 700 mg: 700·200/800 = 175 lost;
	// then 1 of the 3 left: 525·200/600 = 175 lost.
	tk := testDesign(Hull{Name: "Tanker", Armor: 25}, 10)
	tk.FuelCapacity = 200
	d := []Design{tk}
	e := tok(d, 0, 1, 3)
	e.fleet = 0
	tb := newTestBattle(panicRand{}, d, e)
	tb.g.Fleets = []Fleet{{Owner: 1, Fuel: 700, Stacks: []Stack{{Design: 0, Count: 4}}}}
	tb.cargoShare(e, 1)
	if got := tb.g.Fleets[0].Fuel; got != 525 {
		t.Errorf("fuel after first kill %d, want 525", got)
	}
	e.ships = 2
	tb.cargoShare(e, 1)
	if got := tb.g.Fleets[0].Fuel; got != 350 {
		t.Errorf("fuel after second kill %d, want 350", got)
	}
	e.ships = 0
	tb.cargoShare(e, 2)
	if got := tb.g.Fleets[0].Fuel; got != 0 {
		t.Errorf("fuel after the fleet died %d, want 0", got)
	}
}

func TestPredictionOutPlayerStillFires(t *testing.T) {
	// COMBAT.md "Rounds" step 5 and "Firing" (stars-elegy #31): step 5
	// only decides whether the battle ends. Player 0 attacks nobody and
	// player 1 attacks only player 0, so both are out; players 2 and 3
	// fight on. Player 1's token still fires, at player 0.
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), testDesign(Hull{Armor: 10000}, 10)}
	p0, p1, p2, p3 := tok(d, 1, 0, 1), tok(d, 0, 1, 1), tok(d, 0, 2, 1), tok(d, 1, 3, 1)
	tb := newTestBattle(panicRand{}, d, p0, p1, p2, p3)
	tb.g.Players = make([]Player, 4)
	tb.players = []int{0, 1, 2, 3}
	tb.sets = attackSets{{}, {0: true}, {3: true}, {2: true}}
	tb.checkIn()
	if tb.in[0] || tb.in[1] || len(tb.in) != 2 {
		t.Fatalf("in after step 5: %v, want only 2 and 3", tb.in)
	}
	tb.fire()
	if p0.dmg.Units == 0 {
		t.Errorf("player 1 (out) did not fire at player 0")
	}
	if p3.dmg.Units == 0 {
		t.Errorf("player 2 did not fire at player 3")
	}
}

func TestPredictionDumpCargo(t *testing.T) {
	// COMBAT.md "Setup steps" 2 (stars-elegy #31): in deep space the full
	// dump is the salvage object's first addition, no quarter lost, and
	// the object exists with no kills. Colonists and fuel stay aboard.
	fr := testDesign(Hull{Name: "Freighter", Armor: 25}, 10)
	fr.CargoCapacity = 100
	plan := []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEnemies, DumpCargo: true}}
	g := &Game{
		Players: []Player{{Plans: plan}, {Plans: plan}},
		Designs: []Design{fr},
		Fleets: []Fleet{{ID: 1, Owner: 0, Fuel: 50, Cargo: Cargo{Minerals: Minerals{40, 0, 3}, Colonists: 5},
			Stacks: []Stack{{Design: 0, Count: 1}}}},
	}
	b := &battle{g: g, rng: &seqRand{}, loc: location{planet: -1, fleets: []int{0}}, players: []int{0}, involved: 1, killed: map[int]bool{}}
	b.setup(map[int]bool{})
	b.finish()
	f := g.Fleets[0]
	if f.Cargo != (Cargo{Colonists: 5}) || f.Fuel != 50 {
		t.Errorf("fleet after dump %+v fuel %d", f.Cargo, f.Fuel)
	}
	if len(g.Salvage) != 1 || g.Salvage[0].Minerals != (Minerals{40, 0, 3}) {
		t.Errorf("salvage %+v", g.Salvage)
	}
}

func TestPredictionEstimateDrawsAt200(t *testing.T) {
	// One ship with one torpedo simulates exactly 200 torpedoes: 200 draws.
	d := []Design{testDesign(tFrigate, 10, Slot{tBeta, 1}), testDesign(Hull{Armor: 10000}, 10)}
	f, e := tok(d, 0, 0, 1), tok(d, 1, 1, 1)
	n := &countRand{}
	tb := newTestBattle(n, d, f, e)
	tb.estimate(f, e, 0, false)
	if n.n != 200 {
		t.Errorf("%d draws, want 200", n.n)
	}
	f2 := tok(d, 0, 0, 2)
	n.n = 0
	tb.estimate(f2, e, 0, false)
	if n.n != 0 {
		t.Errorf("400 torpedoes: %d draws, want 0", n.n)
	}
}

type countRand struct{ n int }

func (c *countRand) Intn(int) int { c.n++; return 0 }

func TestPredictionRepairOthers(t *testing.T) {
	// "Moved" rate 5, Inner Strength doubling r for fleets, starbase
	// repair 50 (Inner Strength 75, Interstellar Traveler 50), and no
	// repair after fighting.
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1})}
	g := &Game{
		Players: make([]Player, 3),
		Designs: d,
		Planets: []Planet{
			{Pos: Point{10, 0}, Owner: 0, HasStarbase: true, StarbaseDamage: 300},
			{Pos: Point{20, 0}, Owner: 2, HasStarbase: true, StarbaseDamage: 300},
			{Pos: Point{30, 0}, Owner: 1, HasStarbase: true, StarbaseDamage: 300},
		},
	}
	g.Players[1].Race.PRT = PRTInnerStrength
	g.Players[2].Race.PRT = PRTInnerStrength
	g.Players[0].Race.PRT = PRTInterstellarTraveler
	dmg := Damage{Pct: 100, Units: 200}
	g.Fleets = []Fleet{
		{ID: 1, Owner: 0, Stacks: []Stack{{Design: 0, Count: 1, Damage: dmg}}},
		{ID: 2, Owner: 1, Stacks: []Stack{{Design: 0, Count: 1, Damage: dmg}}},
		{ID: 3, Owner: 0, Stacks: []Stack{{Design: 0, Count: 1, Damage: dmg}}},
	}
	repair(g, map[int]bool{1: true}, battleResult{fleets: map[int]bool{3: true}, bases: map[int]bool{}})
	got := []int{g.Fleets[0].Stacks[0].Damage.Units, g.Fleets[1].Stacks[0].Damage.Units, g.Fleets[2].Stacks[0].Damage.Units,
		g.Planets[0].StarbaseDamage, g.Planets[1].StarbaseDamage, g.Planets[2].StarbaseDamage}
	if want := []int{195, 180, 200, 250, 225, 225}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPredictionEmptySalvageGetsTokenAmount(t *testing.T) {
	// An all-zero deep-space addition becomes rand(10) of each, redrawn
	// until the total is above 0.
	d := []Design{testDesign(tFrigate, 10)}
	tb := newTestBattle(&seqRand{draws: []int{0, 0, 0, 3, 4, 5}}, d)
	tb.pending = []Minerals{{}}
	tb.finish()
	if len(tb.g.Salvage) != 1 || tb.g.Salvage[0].Minerals != (Minerals{3, 4, 5}) {
		t.Errorf("salvage %+v", tb.g.Salvage)
	}
}

func TestPredictionSalvageLimit(t *testing.T) {
	// 3000 steps of 10 kT per object; a mineral that does not fit fills
	// the object and the rest goes into a new object.
	tb := newTestBattle(panicRand{}, nil)
	tb.addSalvage(Minerals{29995, 20, 0})
	if want := []Minerals{{29995, 0, 0}, {0, 20, 0}}; !reflect.DeepEqual(tb.salvage, want) {
		t.Errorf("salvage %v, want %v", tb.salvage, want)
	}
	tb = newTestBattle(panicRand{}, nil)
	tb.addSalvage(Minerals{0, 0, 100})
	tb.addSalvage(Minerals{29950, 0, 0})
	if want := []Minerals{{29950, 0, 50}, {0, 0, 50}}; !reflect.DeepEqual(tb.salvage, want) {
		t.Errorf("salvage %v, want %v", tb.salvage, want)
	}
}

func TestPredictionTorpedoHitsPerTarget(t *testing.T) {
	// Hits are drawn afresh for each target from the unfired torpedoes:
	// 10 Betas kill a 5-armor ship with 1 torpedo, then draw 9 more for
	// the next target.
	d := []Design{testDesign(tFrigate, 10, Slot{tBeta, 1}), testDesign(Hull{Armor: 5}, 1000), testDesign(Hull{Armor: 10000}, 10)}
	f := tok(d, 0, 0, 10)
	n := &countRand{}
	tb := newTestBattle(n, d, f, tok(d, 1, 1, 1), tok(d, 2, 1, 1))
	tb.torpedoes(0, f.weapons[0])
	if n.n != 19 || len(tb.hits) != 2 || tb.hits[0].Hits != 1 || tb.hits[1].Hits != 9 {
		t.Errorf("%d draws, hits %+v; want 19 draws, 1 then 9 hits", n.n, tb.hits)
	}
}

func TestPredictionAlternateRealityStarbaseLoss(t *testing.T) {
	// Destroying an Alternate Reality race's starbase leaves the planet
	// uninhabited; any destroyed starbase leaves the planet without one.
	station := testDesign(Hull{Name: "Orbital Fort", Armor: 100, Starbase: true}, 10)
	d := []Design{station}
	sb := tokenValues(station, Race{}, true, Cost{})
	sb.player, sb.planet, sb.fleet, sb.dead = 0, 0, -1, true
	tb := newTestBattle(panicRand{}, d, &sb)
	tb.g.Players[0].Race.PRT = PRTAlternateReality
	tb.g.Planets = []Planet{{Owner: 0, Population: 50, HasStarbase: true, StarbaseHull: 1, StarbaseDock: true}}
	tb.finish()
	p := tb.g.Planets[0]
	if p.HasStarbase || p.StarbaseHull != 0 || p.StarbaseDock || p.Owner != NoOwner || p.Population != 0 {
		t.Errorf("planet after AR starbase loss: %+v", p)
	}
}

func TestPredictionStarbaseJammer(t *testing.T) {
	// A starbase's jammer is reduced by a quarter: Jammer 20 → 15.
	sb := tokenValues(testDesign(Hull{Starbase: true, Armor: 500}, 10, Slot{tJammer(80), 1}), Race{}, true, Cost{})
	if sb.jammer != 15 {
		t.Errorf("starbase jammer %d, want 15", sb.jammer)
	}
}

func TestPredictionPlan0AbsentPlayer(t *testing.T) {
	// Plan-0 X is a player of the game not at the location (C = player 0,
	// whose lone hauler was examined first): C's set names B, so Q = {B,
	// C} and n = 2. A (player 1, the station) starts on (1,4), B (player
	// 2) on (8,5), and the battle ends after round 0's movement with no
	// shots.
	station := testDesign(Hull{Name: "Space Station", Armor: 500, Initiative: 14, Starbase: true}, 100, Slot{tLaser, 8})
	frig := testDesign(tFrigate, 10, Slot{tLaser, 1})
	g := &Game{
		Players: []Player{
			{},
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackPlayer, Player: 2}}},
			{Plans: []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Secondary: TargetAny, Attack: AttackEnemies}}, Relations: []Relation{RelationNeutral, RelationNeutral, RelationFriend}},
		},
		Designs: []Design{station, frig, testDesign(Hull{Name: "Hauler", Armor: 20}, 5)},
		Planets: []Planet{{ID: 1, Pos: Point{5, 5}, Owner: 1, HasStarbase: true, StarbaseDesign: 0}},
		Fleets: []Fleet{
			{ID: 1, Owner: 0, Pos: Point{50, 50}, Stacks: []Stack{{Design: 2, Count: 1}}},
			{ID: 2, Owner: 2, Pos: Point{5, 5}, Stacks: []Stack{{Design: 1, Count: 5}}},
		},
	}
	var prev locationHistory
	var b *battle
	for _, loc := range g.locations() {
		sets, inv, n := g.whoFights(loc, prev)
		prev = locationHistory{any: true, battle: inv != nil, lastOwner: g.Fleets[loc.fleets[len(loc.fleets)-1]].Owner}
		if inv != nil {
			b = &battle{g: g, rng: rand.New(rand.NewSource(1)), loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}}
		}
	}
	if b == nil || b.involved != 2 || !reflect.DeepEqual(b.players, []int{1, 2}) {
		t.Fatalf("battle %+v", b)
	}
	b.setup(map[int]bool{})
	for _, tk := range b.tokens {
		want := [2]int{8, 5}
		if tk.starbase {
			want = [2]int{1, 4}
		}
		if got := [2]int{tk.x, tk.y}; got != want {
			t.Errorf("token of player %d on %v, want %v", tk.player, got, want)
		}
	}
	b.fight()
	if len(b.hits) != 0 || b.round != 0 {
		t.Errorf("%d hits, ended in round %d; want none in round 0", len(b.hits), b.round)
	}
}

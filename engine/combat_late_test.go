package engine

import (
	"reflect"
	"testing"
)

// Late-game combat paths that RG-001's battles never reach: big
// starbase hulls, late-tech parts in token values, starbase damage on a
// large armor value, gatling deflectors and repair at the large docks.
// Designs come from the component table (engine/data/components.json).
// Statuses follow stars-elegy docs/COMBAT.md, as in combat_test.go.

// lateDesign builds a design from the component table.
func lateDesign(t *testing.T, hull string, fills ...SlotFill) Design {
	t.Helper()
	d, err := Components().NewDesign(hull, hull, fills)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestPredictionLateShipTokenValues(t *testing.T) {
	// COMBAT.md "Token values". Each formula is CONFIRMED for the parts
	// its case names (BSC, Jammer 20 and 50, one Energy and one Flux
	// Capacitor, one Beam Deflector, Mole-skin with RS); these parts and
	// combinations are BINARY-ONLY.
	d := lateDesign(t, "Nubian",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 3},
		SlotFill{Slot: 1, Part: "Battle Computer", Count: 1},
		SlotFill{Slot: 2, Part: "Battle Super Computer", Count: 1},
		SlotFill{Slot: 3, Part: "Battle Nexus", Count: 1},
		SlotFill{Slot: 4, Part: "Multi Contained Munition", Count: 1},
		SlotFill{Slot: 5, Part: "Langston Shell", Count: 1},
		SlotFill{Slot: 6, Part: "Mega Poly Shell", Count: 1},
		SlotFill{Slot: 7, Part: "Multi Function Pod", Count: 1},
		SlotFill{Slot: 8, Part: "Jammer 30", Count: 1},
		SlotFill{Slot: 9, Part: "Beam Deflector", Count: 2},
		SlotFill{Slot: 10, Part: "Flux Capacitor", Count: 3},
		SlotFill{Slot: 11, Part: "Croby Sharmor", Count: 1},
		SlotFill{Slot: 12, Part: "Fielded Kelarium", Count: 1})
	hull := d.Hull
	tk := tokenValues(d, Race{}, false, Cost{})
	// Computers compound, the Multi Contained Munition's 10 included:
	// 20 → 44 → 72 → 74. Initiative +1 +2 +3; the munition is a weapon,
	// so its own initiative is its slot's, not the token's.
	// Jammers: 10000·95·80·90·70 per 100⁴ = 4788 → 100 − 48 = 52.
	// Deflectors 1000 → 900 → 810; Flux 1000 → 1200 → 1440 → 1728.
	// Shields 125 + 100 + 60 + 50; armor + 65 + 400 + 65 + 175.
	got := []int{tk.computer, tk.initiative, tk.weapons[0].init, tk.jammer, tk.deflector, tk.capacitor, tk.shield, tk.armor, tk.class}
	want := []int{74, hull.Initiative + 6, min(63, 6+hull.Initiative+6), 52, 81, 172, 335, hull.Armor + 705, classArmed}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("computer, initiative, weapon initiative, jammer, deflector, capacitor, shield, armor, class: %v, want %v", got, want)
	}
	// Regenerating Shields: shields × 7/5; armor parts count half
	// (Mega Poly 200, Fielded Kelarium 87). The shield parts' armor
	// (Langston Shell, Croby Sharmor) is not an armor part's, and counts
	// whole (CONFIRMED for Croby Sharmor, CB-056, stars-elegy b0ca924; the
	// Langston Shell is not observed).
	rs := tokenValues(d, Race{LRT: LRTs{RegeneratingShields: true}}, false, Cost{})
	if rs.shield != 469 || rs.maxShield != 469 || rs.armor != hull.Armor+65+200+65+87 {
		t.Errorf("RS: shield %d max %d armor %d, want 469 469 %d", rs.shield, rs.maxShield, rs.armor, hull.Armor+417)
	}
}

func TestPredictionDeathStarTokenValues(t *testing.T) {
	// COMBAT.md "Token values" and "Starbases in battle" on the Death
	// Star (armor 1500, initiative 18 in the component table).
	nexus := lateDesign(t, "Death Star",
		SlotFill{Slot: 1, Part: "Heavy Blaster", Count: 32},
		SlotFill{Slot: 2, Part: "Battle Nexus", Count: 4},
		SlotFill{Slot: 3, Part: "Battle Nexus", Count: 4},
		SlotFill{Slot: 6, Part: "Battle Nexus", Count: 4},
		SlotFill{Slot: 8, Part: "Battle Nexus", Count: 4})
	tk := tokenValues(nexus, Race{}, true, Cost{})
	// Initiative 18 + 16·3 = 66, at most 63; the weapon's 5 + 63 also stops
	// at 63. Sixteen Nexus: 50, 75, 87, 93, 96, 98, 99, 99, … Reach: the
	// Heavy Blaster's 3, +1 on a starbase. CB-056's Nexus Star token had
	// initiative 63 and computer 99 (CONFIRMED, stars-elegy b0ca924); the
	// weapon initiative, reach and armor stay predictions.
	got := []int{tk.initiative, tk.weapons[0].init, tk.computer, tk.weapons[0].reach, tk.armor, tk.class}
	if want := []int{63, 63, 99, 4, 1500, classArmed}; !reflect.DeepEqual(got, want) {
		t.Errorf("initiative, weapon initiative, computer, reach, armor, class: %v, want %v", got, want)
	}
	// Five Jammer 50s: 10000 → 312, 100 − 3 = 97, at most 95, then a quarter
	// off on a starbase: 95 − 23 = 72. The cap comes first; a quarter off 97
	// first would give 73 (CONFIRMED 72, CB-056, stars-elegy b0ca924).
	jam := lateDesign(t, "Death Star",
		SlotFill{Slot: 12, Part: "Jammer 50", Count: 4},
		SlotFill{Slot: 14, Part: "Jammer 50", Count: 1})
	if j := tokenValues(jam, Race{}, true, Cost{}).jammer; j != 72 {
		t.Errorf("Death Star with five Jammer 50s: jammer %d, want 72", j)
	}
	// Six Flux Capacitors: 1000 → … → 2487 → 2984, at most 2550: 255%
	// (CONFIRMED, CB-056, stars-elegy b0ca924).
	flux := lateDesign(t, "Death Star",
		SlotFill{Slot: 2, Part: "Flux Capacitor", Count: 4},
		SlotFill{Slot: 3, Part: "Flux Capacitor", Count: 2})
	if c := tokenValues(flux, Race{}, true, Cost{}).capacitor; c != 255 {
		t.Errorf("six Flux Capacitors: capacitor %d%%, want 255%%", c)
	}
	// Thirty Complete Phase Shields and twenty Superlatanium: shields
	// 15000, armor 1500 + 30000; with RS 21000 and 1500 + 15000.
	wall := lateDesign(t, "Death Star",
		SlotFill{Slot: 4, Part: "Complete Phase Shield", Count: 30},
		SlotFill{Slot: 11, Part: "Superlatanium", Count: 20})
	w := tokenValues(wall, Race{}, true, Cost{})
	r := tokenValues(wall, Race{LRT: LRTs{RegeneratingShields: true}}, true, Cost{})
	if got := []int{w.shield, w.armor, r.shield, r.armor}; !reflect.DeepEqual(got, []int{15000, 31500, 21000, 16500}) {
		t.Errorf("shield, armor, RS shield, RS armor: %v, want [15000 31500 21000 16500]", got)
	}
}

func TestPredictionDeathStarDamageSteps(t *testing.T) {
	// COMBAT.md "Damage", "Starbases" (CONFIRMED on a Space Station,
	// CB-011..015): total = damage + units·armor/500; below the armor,
	// units = total·500/armor, at least one step more than before; else
	// destroyed, with nothing left over. On the Death Star's 1500 armor a
	// unit is 3 damage, so a small hit still takes one step.
	ds := lateDesign(t, "Death Star")
	cases := []struct {
		units, dp int
		want      int // units after, -1 destroyed
	}{
		{0, 1, 1},       // 1·500/1500 = 0, one step more
		{10, 2, 11},     // 32·500/1500 = 10, one step more
		{10, 6, 12},     // 36·500/1500 = 12
		{300, 599, 499}, // 1499: just below the armor
		{300, 600, -1},  // 1500: destroyed
		// At 499 units a hit below one unit (1498 < 1500) takes the 500th
		// step, and the next hit destroys it (MEASURED, CB-056, 2 streams,
		// stars-elegy b0ca924: a starbase went 499 → 500, then was destroyed).
		{499, 1, 500},
		{500, 1, -1},
	}
	for _, c := range cases {
		d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), ds}
		sb := tokenValues(ds, Race{}, true, Cost{})
		sb.player, sb.design, sb.ships, sb.fleet, sb.planet = 1, 1, 1, -1, -1
		sb.dmg = Damage{Pct: 100, Units: c.units}
		tb := newTestBattle(panicRand{}, d, tok(d, 0, 0, 1), &sb)
		h, _ := tb.damage(&sb, c.dp, 0, false, -1)
		got := sb.dmg.Units
		if sb.dead {
			got = -1
		}
		if got != c.want || h.Leftover != 0 || (c.want < 0) != (h.Kills == 1) {
			t.Errorf("units %d, hit %d: %d (kills %d, leftover %d), want %d", c.units, c.dp, got, h.Kills, h.Leftover, c.want)
		}
	}
}

func TestConfirmedBeamStopsAtStarbase(t *testing.T) {
	// COMBAT.md "Damage", "Starbases" (CONFIRMED, CB-052, 6 streams): a
	// one-slot phaser stack put 234 into an Orbital Fort with 100 armor,
	// and its fire action held only the Fort's record, though an enemy
	// stack on the same square was in range.
	fort := lateDesign(t, "Orbital Fort")
	d := []Design{testDesign(tFrigate, 10, Slot{tPhaser, 1}), fort, testDesign(tFrigate, 10)}
	f := tok(d, 0, 0, 9) // 9 × 26 = 234
	f.primary, f.secondary = TargetStarbase, TargetAny
	sb := tokenValues(fort, Race{}, true, Cost{})
	sb.player, sb.design, sb.ships, sb.fleet, sb.planet = 1, 1, 1, -1, -1
	sb.dmg = Damage{Pct: 100}
	ship := tok(d, 2, 1, 5)
	tb := newTestBattle(panicRand{}, d, f, &sb, ship)
	tb.beam(0, f.weapons[0])
	if len(tb.hits) != 1 || tb.hits[0].Target != 1 || !sb.dead || ship.dmg != (Damage{}) || ship.ships != 5 {
		t.Errorf("hits %+v, Fort dead %v, ship %d at %+v; want one hit, on the Fort only", tb.hits, sb.dead, ship.ships, ship.dmg)
	}
}

func TestGatlingDeflectorPerTarget(t *testing.T) {
	// COMBAT.md "Beams", "Gatling" (CONFIRMED, CB-002, CB-003, CB-005):
	// each target takes ships × damage × count × capacitor/100 × that
	// target's deflector/100. Gatling Neutrino Cannon (80), 3 ships, 2 per
	// slot, one Flux Capacitor (120%): 576, then 90% → 518 and 81% → 466.
	gnc := Part{Name: "Gatling Neutrino Cannon", Kind: PartBeam, Damage: 80, Range: 2, Initiative: 13, Gatling: true}
	flux := Part{Name: "Flux Capacitor", Kind: PartElectrical, Capacitor: 20}
	defl := Part{Name: "Beam Deflector", Kind: PartMechanical, Deflector: true}
	big := Hull{Name: "Wall", Armor: 100000}
	d := []Design{
		testDesign(tFrigate, 10, Slot{gnc, 2}, Slot{flux, 1}),
		testDesign(big, 10),
		testDesign(big, 10, Slot{defl, 1}),
		testDesign(big, 10, Slot{defl, 2}),
		testDesign(Hull{Name: "Wall Station", Armor: 100000, Starbase: true}, 10),
	}
	f := tok(d, 0, 0, 3)
	sb := tokenValues(d[4], Race{}, true, Cost{})
	sb.player, sb.design, sb.ships, sb.fleet, sb.planet = 1, 4, 1, -1, -1
	sb.dmg = Damage{Pct: 100}
	tb := newTestBattle(panicRand{}, d, f, tok(d, 1, 1, 1), &sb, tok(d, 2, 1, 1), tok(d, 3, 1, 1))
	tb.gatling(0, f.weapons[0])
	var got []int
	for _, h := range tb.hits {
		got = append(got, h.Armor)
	}
	// A starbase among the targets takes its share and the shot goes on to
	// the next target (CONFIRMED in one stream only, CB-056, stars-elegy
	// b0ca924: the starbase was token 0, and three ships after it took 466,
	// 576 and 518 for deflectors 81, 100 and 90).
	if want := []int{576, 576, 518, 466}; !reflect.DeepEqual(got, want) {
		t.Errorf("gatling hits %v, want %v", got, want)
	}
}

func TestPredictionRepairLargeDocks(t *testing.T) {
	// COMBAT.md "Repair" (CONFIRMED, CB-017 for its locations; CB-024 for
	// Inner Strength): at its own planet with a dock, r = 100; the Ultra
	// Station and Death Star have docks (component table). A fleet at a
	// planet whose starbase fought this turn gets 25, and that starbase is
	// not repaired. CB-056 CONFIRMED the plain lines (stars-elegy b0ca924,
	// fleets 400 → 300, starbases 300 → 250); Inner Strength at these docks
	// and the starbase that fought stay predictions.
	hullNumber := func(name string) int { return lateDesign(t, name).Hull.StarbaseNumber }
	g := &Game{
		Rules:   ElegyRules(),
		Players: make([]Player, 2),
		Designs: []Design{testDesign(tFrigate, 10, Slot{tLaser, 1})},
		Planets: []Planet{
			{Pos: Point{10, 0}, Owner: 0, HasStarbase: true, StarbaseHull: hullNumber("Ultra Station"), StarbaseDamage: 300},
			{Pos: Point{20, 0}, Owner: 0, HasStarbase: true, StarbaseHull: hullNumber("Death Star"), StarbaseDamage: 300},
			{Pos: Point{30, 0}, Owner: 1, HasStarbase: true, StarbaseHull: hullNumber("Death Star"), StarbaseDamage: 300},
			{Pos: Point{40, 0}, Owner: 0, HasStarbase: true, StarbaseHull: hullNumber("Death Star"), StarbaseDamage: 300},
		},
	}
	g.Players[1].Race.PRT = PRTInnerStrength
	for i, p := range []Point{{10, 0}, {20, 0}, {30, 0}, {40, 0}} {
		g.Fleets = append(g.Fleets, Fleet{ID: i + 1, Owner: g.Planets[i].Owner, Pos: p,
			Stacks: []Stack{{Design: 0, Count: 2, Damage: Damage{Pct: 50, Units: 400}}}})
	}
	repair(g, nil, battleResult{fleets: map[int]bool{}, bases: map[int]bool{3: true}})
	var got []int
	for i := range g.Fleets {
		got = append(got, g.Fleets[i].Stacks[0].Damage.Units)
	}
	for _, p := range g.Planets {
		got = append(got, p.StarbaseDamage)
	}
	// Fleets: 400 − 100, 400 − 100, 400 − 200 (Inner Strength), 400 − 25.
	// Starbases: 300 − 50, 300 − 50, 300 − 75, the one that fought 300.
	if want := []int{300, 300, 200, 375, 250, 250, 225, 300}; !reflect.DeepEqual(got, want) {
		t.Errorf("units %v, want %v", got, want)
	}
}

func TestConfirmedCB056RegeneratingShieldsCroby(t *testing.T) {
	// CB-056 (CONFIRMED, stars-elegy b0ca924): a Regenerating Shields race's
	// Battleship with 8 Croby Sharmor and 2 Tritanium. Armor 2000 + 8·65
	// (whole, a shield part) + 2·50/2 = 2570; shields 8·60·7/5 = 672. The
	// record gave shields 672 and bounded the armor to 2546–2581, which
	// excludes 2306 (each Croby Sharmor's armor halved, 65 → 32).
	d := lateDesign(t, "Battleship",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		SlotFill{Slot: 2, Part: "Croby Sharmor", Count: 8},
		SlotFill{Slot: 8, Part: "Tritanium", Count: 2})
	tk := tokenValues(d, Race{LRT: LRTs{RegeneratingShields: true}}, false, Cost{})
	if tk.armor != 2570 || tk.shield != 672 {
		t.Errorf("armor %d shield %d, want 2570 672", tk.armor, tk.shield)
	}
}

func TestMeasuredRG002RegeneratingShieldsArmor(t *testing.T) {
	// RG-002-Y6 (MEASURED, one stream; stars-elegy fe0ccc5), the battle at
	// (1241,1313): player 2 has Regenerating Shields, and its Rogue with
	// 2 Organic Armor (175 each) took Delta Torpedo hits putting 26, 13,
	// 13, 26, 26, 13 and 26 on its armor (shields 252 = 3 × 60 × 7/5).
	// The recorded damage units, 21 32 43 64 85 96 117, fit only armor 625
	// or 626: 450 + 350/2 (halved per slot, stars-elegy COMBAT.md "Armor",
	// BINARY-ONLY), not 450 + 2 × 87 = 624 (per part, which gives 42 at
	// the third hit). One slot does not tell per slot from the parts'
	// total halved once (TestMeasuredCB057RegeneratingShieldsTwoSlots).
	d := lateDesign(t, "Rogue",
		SlotFill{Slot: 0, Part: "Trans-Galactic Fuel Scoop", Count: 2},
		SlotFill{Slot: 1, Part: "Wolverine Diffuse Shield", Count: 3},
		SlotFill{Slot: 2, Part: "Cargo Pod", Count: 2},
		SlotFill{Slot: 3, Part: "Pick Pocket Scanner", Count: 1},
		SlotFill{Slot: 4, Part: "Organic Armor", Count: 2},
		SlotFill{Slot: 5, Part: "Beta Torpedo", Count: 2},
		SlotFill{Slot: 6, Part: "Maneuvering Jet", Count: 2},
		SlotFill{Slot: 7, Part: "Stealth Cloak", Count: 1},
		SlotFill{Slot: 8, Part: "Battle Computer", Count: 1})
	tk := tokenValues(d, Race{LRT: LRTs{RegeneratingShields: true}}, false, Cost{})
	if tk.armor != 625 || tk.shield != 252 {
		t.Fatalf("armor %d shield %d, want 625 252", tk.armor, tk.shield)
	}
	ds := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), d}
	tk.player, tk.design, tk.ships, tk.fleet, tk.planet = 2, 1, 1, -1, -1
	tk.shield = 0
	tb := newTestBattle(panicRand{}, ds, tok(ds, 0, 0, 1), &tk)
	var got []int
	for _, dp := range []int{26, 13, 13, 26, 26, 13, 26} {
		tb.damage(&tk, dp, 0, false, -1)
		got = append(got, tk.dmg.Units)
	}
	if want := []int{21, 32, 43, 64, 85, 96, 117}; !reflect.DeepEqual(got, want) {
		t.Errorf("damage units %v, want %v", got, want)
	}
}

// CB-057 (stars-elegy COMBAT.md; PARITY.md "Round 11"). Predicted in elegy
// 89cbc6a before the runs, but Elegy's numbers for the exact run designs
// were pushed after the runs began, so the results are MEASURED (oracle)
// and consistent with 89cbc6a, not CONFIRMED by prediction.

func TestMeasuredCB057RegeneratingShieldsTwoSlots(t *testing.T) {
	// Q1. RS halves each armor slot's armor, truncated per slot
	// (stars-elegy COMBAT.md "Armor"). MEASURED (CB-057, 6 streams: the
	// two-slot Destroyer fits 372–374, the one-slot one 375–376). A Destroyer (armor
	// 200) with one Organic Armor (175) in its general-purpose slot and one
	// in its armor slot: 200 + 87 + 87 = 374, where the parts' total
	// halved once gives 375. Control: both in the armor slot, 200 + 175 =
	// 375 either way. A Battleship takes armor only in its armor slot: an
	// Organic Armor in its shield slot is dropped when the design is read
	// (COMBAT.md "Parts the game drops from a design"), leaving 2000 + 87.
	rs := Race{LRT: LRTs{RegeneratingShields: true}}
	two := lateDesign(t, "Destroyer",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 1},
		SlotFill{Slot: 3, Part: "Organic Armor", Count: 1},
		SlotFill{Slot: 4, Part: "Organic Armor", Count: 1})
	one := lateDesign(t, "Destroyer",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 1},
		SlotFill{Slot: 4, Part: "Organic Armor", Count: 2})
	bb, err := Components().NewDesignAnyKind("BB", "Battleship", []SlotFill{
		{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		{Slot: 2, Part: "Organic Armor", Count: 1},
		{Slot: 8, Part: "Organic Armor", Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	got := []int{
		tokenValues(battleDesign(two), rs, false, Cost{}).armor,
		tokenValues(battleDesign(one), rs, false, Cost{}).armor,
		tokenValues(battleDesign(bb), rs, false, Cost{}).armor,
	}
	if want := []int{374, 375, 2087}; !reflect.DeepEqual(got, want) {
		t.Errorf("RS armor: two slots, one slot, Battleship with a shield-slot Organic Armor: %v, want %v", got, want)
	}
}

func TestMeasuredCB057NexusTorpedoes(t *testing.T) {
	// Q2. MEASURED (CB-057, 6 streams: 24 salvos all hit, no miss
	// record, and the battle replays only without draws for them). A
	// Death Star with sixteen Battle Nexus (computer 99) and 32
	// Delta Torpedoes (accuracy 60) at an unjammed stack of 20 Battleships
	// with 6 Superlatanium (no shields, armor 11000 each): p = 100 −
	// 1·40/100 = 100, so all 32 hit with no draw (panicRand), and 32·26/2
	// goes to shields (none) and again to armor: 832, no kill. A firer
	// with computer 0 still draws, so its hits are the same whether or not
	// the Death Star fired first.
	ds := lateDesign(t, "Death Star",
		SlotFill{Slot: 1, Part: "Delta Torpedo", Count: 32},
		SlotFill{Slot: 2, Part: "Battle Nexus", Count: 4},
		SlotFill{Slot: 3, Part: "Battle Nexus", Count: 4},
		SlotFill{Slot: 6, Part: "Battle Nexus", Count: 4},
		SlotFill{Slot: 8, Part: "Battle Nexus", Count: 4})
	bb := lateDesign(t, "Battleship",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		SlotFill{Slot: 8, Part: "Superlatanium", Count: 6})
	d := []Design{ds, bb}
	sb := tokenValues(ds, Race{}, true, Cost{})
	sb.player, sb.design, sb.ships, sb.fleet, sb.planet = 0, 0, 1, -1, -1
	sb.tactic, sb.primary, sb.secondary = TacticMaximizeDamage, TargetAny, TargetAny
	e := tokenValues(bb, Race{}, false, Cost{})
	e.player, e.design, e.ships, e.fleet, e.planet = 1, 1, 20, -1, -1
	if sb.computer != 99 || e.jammer != 0 || e.shield != 0 || e.armor != 11000 {
		t.Fatalf("computer %d, target jammer %d shield %d armor %d", sb.computer, e.jammer, e.shield, e.armor)
	}
	tb := newTestBattle(panicRand{}, d, &sb, &e)
	tb.torpedoes(0, sb.weapons[0])
	if len(tb.hits) != 1 || tb.hits[0].Hits != 32 || tb.hits[0].Misses != 0 || tb.hits[0].Armor != 832 || tb.hits[0].Kills != 0 {
		t.Errorf("hits %+v, want one record of 32 hits, 0 misses, armor 832, no kill", tb.hits)
	}
}

func TestMeasuredCB057TargetCostCap(t *testing.T) {
	// Q3. COMBAT.md "Target choice": cost × ships is ×100 below 100000,
	// else 10^7; a beam's score is that × 100/(A + S + 1). Costs are
	// Elegy's for a race with no cost LRT at tech 26 in every field
	// (resources + boranium per ship).
	//   S1: 10000 Destroyers with one Long Hump 6: cost 11, armor 200;
	//       cost × ships 110000, A = 2000000.
	//   S2: 16000 Battleships with 4 Trans-Galactic Drives and 12 Omega
	//       Torpedoes: cost 640, armor 2000; cost × ships 10240000,
	//       A = 32000000.
	// Capped, both score 10^7, so the beam picks S1 (lower A). With no
	// cap (or cost × ships uncapped above 100000) S2's 640/2000 per ship
	// beats S1's 11/200 and the beam picks S2. MEASURED (CB-057, 4
	// streams, 14 shots: S1 every time): a common cap at or below
	// 10,240,000. Any common cap gives this choice, so the value 10^7
	// stays BINARY-ONLY (TestPredictionTargetChoiceCostCap).
	var lv [NumFields]int
	for i := range lv {
		lv[i] = 26
	}
	dd := lateDesign(t, "Destroyer", SlotFill{Slot: 0, Part: "Long Hump 6", Count: 1})
	bb := lateDesign(t, "Battleship",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		SlotFill{Slot: 3, Part: "Omega Torpedo", Count: 6},
		SlotFill{Slot: 4, Part: "Omega Torpedo", Count: 6})
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), dd, bb}
	f := tok(d, 0, 0, 1)
	s1, s2 := tok(d, 1, 1, 10000), tok(d, 2, 1, 16000)
	s1.cost = designCost(dd, Race{}, lv).Resources + designCost(dd, Race{}, lv).Minerals[Boranium]
	s2.cost = designCost(bb, Race{}, lv).Resources + designCost(bb, Race{}, lv).Minerals[Boranium]
	if got := []int{s1.cost, s1.armor, s2.cost, s2.armor}; !reflect.DeepEqual(got, []int{11, 200, 640, 2000}) {
		t.Fatalf("cost, armor of S1 and S2: %v, want [11 200 640 2000]", got)
	}
	// S2 first in token order, so a tie would pick it.
	tb := newTestBattle(panicRand{}, d, f, s2, s1)
	if got := tb.choose(f, f.weapons[0]); got != 2 {
		t.Errorf("chose token %d, want 2 (S1: both capped at 10^7, lower armor)", got)
	}
}

func TestPredictionTorpedoCertainHit(t *testing.T) {
	// COMBAT.md "Torpedoes and missiles": with c > j, p = 100 −
	// (100 − (c − j))·(100 − acc)/100, and p ≥ 100 means every torpedo
	// hits with no draw. A computer of 99 (sixteen Battle Nexus, CONFIRMED
	// on CB-056's Nexus Star) against no jammer gives 100 − ⌊(100 − acc)/100⌋
	// = 100 for every torpedo. The "all hit" branch is unobserved.
	for _, acc := range []int{20, 45, 80} {
		if p := hitChance(acc, 99, 0); p != 100 {
			t.Errorf("hitChance(%d, 99, 0) = %d, want 100", acc, p)
		}
	}
	// A Jammer 20 brings it back below 100: 100 − 21·55/100 = 89.
	if p := hitChance(45, 99, 20); p != 89 {
		t.Errorf("hitChance(45, 99, 20) = %d, want 89", p)
	}
	// Ten Beta Torpedo ships (20 torpedoes, at most 200, so normally 20
	// draws) with computer 99 at an unjammed wall: 20 hits, no draw; with
	// no shields both halves of 20·12/2 go to armor (CS-003-C2): 240.
	d := []Design{testDesign(tFrigate, 10, Slot{tBeta, 2}), testDesign(Hull{Armor: 100000}, 10)}
	f, e := tok(d, 0, 0, 10), tok(d, 1, 1, 1)
	f.computer = 99
	tb := newTestBattle(panicRand{}, d, f, e)
	tb.torpedoes(0, f.weapons[0])
	if len(tb.hits) != 1 || tb.hits[0].Hits != 20 || tb.hits[0].Misses != 0 || tb.hits[0].Armor != 240 {
		t.Errorf("hits %+v, want one record of 20 hits, 0 misses, 240 armor", tb.hits)
	}
}

func TestPredictionTargetChoiceCostCap(t *testing.T) {
	// COMBAT.md "Target choice", "Cost": cost × ships is multiplied by
	// 100 below 100000, and is 10^7 otherwise. Late-game stacks reach the
	// cap: 20000 ships at cost 10 and at cost 20 both score 10^7, so at
	// equal toughness the first in token order wins, though the second is
	// dearer. With 1000 ships each the dearer stack is chosen (CONFIRMED
	// at small sizes, CB-009). The cap itself is unobserved.
	plain := testDesign(Hull{Armor: 45}, 10)
	dear := testDesign(Hull{Armor: 45}, 20)
	d := []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), plain, dear}
	choose := func(n int) int {
		f := tok(d, 0, 0, 1)
		tb := newTestBattle(panicRand{}, d, f, tok(d, 1, 1, n), tok(d, 2, 1, n))
		return tb.choose(f, f.weapons[0])
	}
	if got := choose(1000); got != 2 {
		t.Errorf("1000 ships each: chose %d, want 2 (the dearer)", got)
	}
	if got := choose(20000); got != 1 {
		t.Errorf("20000 ships each: chose %d, want 1 (both at the cap)", got)
	}
	// Just below and at the cap: 11111 ships at cost 9 score 99999 × 100,
	// at cost 10 (111110) the cap 10^7, a little more.
	nine := testDesign(Hull{Armor: 45}, 9)
	d = []Design{d[0], nine, plain}
	if got := choose(11111); got != 2 {
		t.Errorf("11111 ships at cost 9 and 10: chose %d, want 2 (10^7 above 9999900)", got)
	}
	// The cap's value, with mixed toughness: 10000 ships of armor 60 at
	// cost 10 (100000, capped: 10^7·100/600001 = 1666) against 11111 of
	// armor 45 at cost 9 (9999900·100/499996 = 1999): the second wins.
	// A cap of 2×10^7 would give the first 3333.
	d = []Design{d[0], testDesign(Hull{Armor: 60}, 10), nine}
	f := tok(d, 0, 0, 1)
	tb := newTestBattle(panicRand{}, d, f, tok(d, 1, 1, 10000), tok(d, 2, 1, 11111))
	if got := tb.choose(f, f.weapons[0]); got != 2 {
		t.Errorf("capped armor 600000 against uncapped armor 499995: chose %d, want 2", got)
	}
	// Exact ties at the cap: one ship of armor 49 at cost 100000 (capped:
	// 10^7·100/50 = 2·10^7) and one of armor 48 at cost 98000 (9800000·
	// 100/49 = 2·10^7). The first in token order wins either way round; a
	// cap of 10^7 − 1 (19999998) or 10^7 + 1 (20000002) breaks one order.
	capped, under := testDesign(Hull{Armor: 49}, 100000), testDesign(Hull{Armor: 48}, 98000)
	for _, order := range [][2]Design{{capped, under}, {under, capped}} {
		d = []Design{d[0], order[0], order[1]}
		f := tok(d, 0, 0, 1)
		a, b := tok(d, 1, 1, 1), tok(d, 2, 1, 1)
		if sa, sb := attractiveness(f, f.weapons[0], a), attractiveness(f, f.weapons[0], b); sa != 20_000_000 || sb != 20_000_000 {
			t.Fatalf("tie scores %d %d, want 20000000 each", sa, sb)
		}
		tb := newTestBattle(panicRand{}, d, f, a, b)
		if got := tb.choose(f, f.weapons[0]); got != 1 {
			t.Errorf("tie at the cap, armor %d first: chose %d, want 1", order[0].Hull.Armor, got)
		}
	}
}

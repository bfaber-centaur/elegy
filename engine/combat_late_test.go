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
	// whole (ASSUMPTION A13).
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
	// Initiative 18 + 16·3 = 66, at most 63; the weapon's 5 + 63 also
	// stops at 63. Sixteen Nexus: 50, 75, 87, 93, 96, 98, 99, 99, …
	// Reach: the Heavy Blaster's 3, +1 on a starbase.
	got := []int{tk.initiative, tk.weapons[0].init, tk.computer, tk.weapons[0].reach, tk.armor, tk.class}
	if want := []int{63, 63, 99, 4, 1500, classArmed}; !reflect.DeepEqual(got, want) {
		t.Errorf("initiative, weapon initiative, computer, reach, armor, class: %v, want %v", got, want)
	}
	// Five Jammer 50s: 10000 → 312, 100 − 3 = 97, at most 95, then a
	// quarter off on a starbase (BINARY-ONLY): 95 − 23 = 72. The cap comes
	// first; a quarter off 97 first would give 73.
	jam := lateDesign(t, "Death Star",
		SlotFill{Slot: 12, Part: "Jammer 50", Count: 4},
		SlotFill{Slot: 14, Part: "Jammer 50", Count: 1})
	if j := tokenValues(jam, Race{}, true, Cost{}).jammer; j != 72 {
		t.Errorf("Death Star with five Jammer 50s: jammer %d, want 72", j)
	}
	// Six Flux Capacitors: 1000 → … → 2487 → 2984, at most 2550: 255%.
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
		// ASSUMPTION A11: at 499 units a hit below one unit (1498 < 1500)
		// would need a 500th step, which is destroyed; COMBAT.md's rule
		// does not say (an oracle question).
		{499, 1, -1},
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
	// ASSUMPTION A12: a starbase among the targets takes its share and the
	// shot goes on to the next target. COMBAT.md says a beam stops at a
	// starbase; a gatling shot past one is unobserved (an oracle question).
	if want := []int{576, 576, 518, 466}; !reflect.DeepEqual(got, want) {
		t.Errorf("gatling hits %v, want %v", got, want)
	}
}

func TestPredictionRepairLargeDocks(t *testing.T) {
	// COMBAT.md "Repair" (CONFIRMED, CB-017 for its locations; CB-024 for
	// Inner Strength): at its own planet with a dock, r = 100; the Ultra
	// Station and Death Star have docks (component table). A fleet at a
	// planet whose starbase fought this turn gets 25, and that starbase
	// is not repaired.
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

func TestPredictionCB056RegeneratingShieldsCroby(t *testing.T) {
	// CB-056 Q3 (ASSUMPTION A13): a Regenerating Shields race's Battleship
	// with 8 Croby Sharmor and 2 Tritanium. Elegy: armor 2000 + 8·65 (whole,
	// a shield part) + 2·50/2 = 2570; shields 8·60·7/5 = 672. If the
	// original halves the Croby Sharmor's armor too: 2310.
	d := lateDesign(t, "Battleship",
		SlotFill{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		SlotFill{Slot: 2, Part: "Croby Sharmor", Count: 8},
		SlotFill{Slot: 8, Part: "Tritanium", Count: 2})
	tk := tokenValues(d, Race{LRT: LRTs{RegeneratingShields: true}}, false, Cost{})
	if tk.armor != 2570 || tk.shield != 672 {
		t.Errorf("armor %d shield %d, want 2570 672", tk.armor, tk.shield)
	}
}

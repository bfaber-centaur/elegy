package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// Mine damage on late-game fleets (OBJECTS.md "Hits on moving fleets",
// damage CONFIRMED by MF-9 and OB-010-S on one-engine Destroyers). The
// engine count and shields worth less than half the damage: CONFIRMED by
// CB-056 (66 fleets, 2 streams; OBJECTS.md at stars-elegy b0ca924).
func TestConfirmedMineDamageLateFleets(t *testing.T) {
	newBB := func(l *lab, name, eng string) {
		d, err := engine.Components().NewDesign(name, "Battleship", []engine.SlotFill{
			{Slot: 0, Part: eng, Count: 4},
			{Slot: 2, Part: "Mole-skin Shield", Count: 1},
			{Slot: 8, Part: "Superlatanium", Count: 2},
		})
		if err != nil {
			t.Fatal(err)
		}
		l.designs[name] = len(l.g.Designs)
		l.g.Designs = append(l.g.Designs, d)
	}
	// Battleship: armor 2000 + 2 × 1500 = 5000, shield 25, four engines.
	// Tank: Destroyer, armor 200 + 3000 = 3200, no shield, one engine.
	cases := []struct {
		name   string
		kind   MineKind
		stacks []any
		units  []int
	}{
		// 5 × 500 × 4 = 10000; shields take 5 × 25 = 125 (below half);
		// (10000 − 125)/5 = 1975 → 1975·500/5000 = 197.
		{"five Battleships heavy", Heavy, []any{"BB", 5}, []int{197}},
		// No fuel at warp 4: 5 × 600 × 4 = 12000 → 2375 → 237.
		{"five scoop Battleships heavy", Heavy, []any{"Scoop BB", 5}, []int{237}},
		// Two ships, standard: the 300 shortfall goes to the first design
		// before its engines multiply: (100 + 300) × 4 = 1600, less 25 →
		// 157; the Tank takes 100 → 15.
		{"Battleship then Tank", Standard, []any{"BB", 1, "Tank", 1}, []int{157, 15}},
		// The Tank first: (100 + 300) × 1 = 400 → 62 (as MF-9's Tank + Cloak
		// Tank); the Battleship 100 × 4 = 400, less 25 → 37.
		{"Tank then Battleship", Standard, []any{"Tank", 1, "BB", 1}, []int{62, 37}},
	}
	for _, c := range cases {
		l := newLab(t)
		newBB(l, "BB", "Trans-Galactic Drive")
		newBB(l, "Scoop BB", "Trans-Galactic Fuel Scoop")
		fi := l.fleet(0, origin, c.stacks...)
		MineDamage(l.g, &l.g.Fleets[fi], c.kind, nil)
		f := l.g.Fleets[fi]
		for i, want := range c.units {
			if got := f.Stacks[i].Damage; got != (engine.Damage{Pct: 100, Units: want}) {
				t.Errorf("%s stack %d: %+v, want 100%% at %d/500", c.name, i, got, want)
			}
		}
	}
	// Mines lost on large fields: max(50, ⌊N/100⌋) (CONFIRMED to 6000,
	// OB-010-S; MF-01's 50,000 field ended at the count this rule gives,
	// PARITY.md "Which field loses mines").
	for _, c := range []struct{ n, lost int }{{10000, 100}, {50000, 500}} {
		if got := MinesLost(c.n); got != c.lost {
			t.Errorf("%d mines: loses %d, want %d", c.n, got, c.lost)
		}
	}
}

// CB-056 (MEASURED, 23 fleets; OBJECTS.md at stars-elegy b0ca924): mine
// damage on a Regenerating Shields race uses the battle's shields and armor.
// Five Battleships (shield 25 × 7/5 = 35; armor 2000 + 2 × 1500/2 = 3500) in
// a heavy field take (10000 − 175)/5 = 1965 → 280/500. The plain sums (armor
// 5000, shield 25) would give 197.
func TestMeasuredMineDamageRegeneratingShields(t *testing.T) {
	l := newLab(t)
	l.g.Players[0].Race.LRT.RegeneratingShields = true
	d, err := engine.Components().NewDesign("BB", "Battleship", []engine.SlotFill{
		{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		{Slot: 2, Part: "Mole-skin Shield", Count: 1},
		{Slot: 8, Part: "Superlatanium", Count: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	l.designs["BB"] = len(l.g.Designs)
	l.g.Designs = append(l.g.Designs, d)
	fi := l.fleet(0, origin, "BB", 5)
	hits := MineDamage(l.g, &l.g.Fleets[fi], Heavy, nil)
	if got := l.g.Fleets[fi].Stacks[0].Damage; got != (engine.Damage{Pct: 100, Units: 280}) {
		t.Errorf("RS Battleships: %+v, want 100%% at 280/500", got)
	}
	if len(hits) != 1 || hits[0].Absorbed != 175 {
		t.Errorf("hits %+v, want one with 175 absorbed", hits)
	}
}

// With Regenerating Shields a shield part's armor counts whole in mine damage
// too, as in battle (CB-056 for battle; for mines a prediction that follows
// the RS mine-damage result). Five Battleships with 8 Croby Sharmor and 2
// Tritanium: shield 8·60·7/5 = 672, armor 2000 + 520 + 50 = 2570; heavy:
// (10000 − 3360)/5 = 1328 → 258/500. Croby Sharmor halved (armor 2306) would
// give 287.
func TestPredictionMineDamageRegeneratingShieldsCroby(t *testing.T) {
	l := newLab(t)
	l.g.Players[0].Race.LRT.RegeneratingShields = true
	d, err := engine.Components().NewDesign("BB", "Battleship", []engine.SlotFill{
		{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		{Slot: 2, Part: "Croby Sharmor", Count: 8},
		{Slot: 8, Part: "Tritanium", Count: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	l.designs["BB"] = len(l.g.Designs)
	l.g.Designs = append(l.g.Designs, d)
	fi := l.fleet(0, origin, "BB", 5)
	MineDamage(l.g, &l.g.Fleets[fi], Heavy, nil)
	if got := l.g.Fleets[fi].Stacks[0].Damage; got != (engine.Damage{Pct: 100, Units: 258}) {
		t.Errorf("RS Croby Battleships: %+v, want 100%% at 258/500", got)
	}
}

// With Regenerating Shields each armor slot's armor is halved and
// truncated once per slot, as the battle token does (stars-elegy
// COMBAT.md "Armor", BINARY-ONLY): three Fielded Kelarium (175 each) in
// one slot add ⌊525/2⌋ = 262, not 3 × 87 = 261 (per part); one Organic
// Armor (175) in each of a Destroyer's two armor slots adds 87 + 87 =
// 174, not ⌊350/2⌋ = 175 (the parts' total halved once).
func TestBinaryRegeneratingShieldsArmorPerSlot(t *testing.T) {
	bb, err := engine.Components().NewDesign("BB", "Battleship", []engine.SlotFill{
		{Slot: 0, Part: "Trans-Galactic Drive", Count: 4},
		{Slot: 8, Part: "Fielded Kelarium", Count: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a, plain := designArmor(bb, true), designArmor(bb, false); a != 2000+262 || plain != 2000+525 {
		t.Errorf("Battleship armor RS %d plain %d, want 2262 2525", a, plain)
	}
	dd, err := engine.Components().NewDesign("DD", "Destroyer", []engine.SlotFill{
		{Slot: 0, Part: "Trans-Galactic Drive", Count: 1},
		{Slot: 3, Part: "Organic Armor", Count: 1},
		{Slot: 4, Part: "Organic Armor", Count: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a, plain := designArmor(dd, true), designArmor(dd, false); a != 200+174 || plain != 200+350 {
		t.Errorf("Destroyer armor RS %d plain %d, want 374 550", a, plain)
	}
}

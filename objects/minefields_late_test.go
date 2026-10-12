package objects

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// Mine damage on late-game fleets (OBJECTS.md "Hits on moving fleets",
// damage CONFIRMED by MF-9 and OB-010-S on one-engine Destroyers). The
// engine count and shields worth less than half the damage are terms of
// the same rule that no observed fleet exercised: these are predictions.
func TestPredictionMineDamageLateFleets(t *testing.T) {
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

// CB-056 Q7: mine damage on a Regenerating Shields race uses the design's
// plain shield and armor sums (Elegy's reading; OBJECTS.md does not say).
// Five Battleships (armor 5000, shield 25) in a heavy field take 197/500,
// as without RS. A battle-style reading (shields ×7/5 = 35, armor parts
// halved: 2000 + 1500 = 3500) would give (10000 − 175)/5 = 1965 → 280.
func TestPredictionMineDamageRegeneratingShields(t *testing.T) {
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
	MineDamage(l.g, &l.g.Fleets[fi], Heavy, nil)
	if got := l.g.Fleets[fi].Stacks[0].Damage; got != (engine.Damage{Pct: 100, Units: 197}) {
		t.Errorf("RS Battleships: %+v, want 100%% at 197/500", got)
	}
}

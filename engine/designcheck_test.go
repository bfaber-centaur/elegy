package engine

import (
	"slices"
	"testing"
)

// slotNames is a design's parts as "count part" strings, in slot order.
func slotNames(d Design) []string {
	var out []string
	for _, s := range d.Slots {
		out = append(out, string(rune('0'+s.Count))+" "+s.Part.Name)
	}
	return out
}

// TestConfirmedDesignCheck: at year generation every ship design is
// re-checked against its owner's tech; a part the owner lacks the tech
// for is removed, except Mystery Trader parts and parts only some
// primary racial traits may build (stars-elegy TAKEOVER.md "Design parts
// dropped when the year is generated"; PARITY.md "Design check
// (CONFIRMED, T-37)"). The JOAT owner's settings are the ones run: tech 3
// in every field (PARITY.md "Design parts dropped", MEASURED), and
// electronics 7 with every other field 26 (CONFIRMED, TK-104-A,
// TK-105-A).
func TestConfirmedDesignCheck(t *testing.T) {
	type row struct {
		hull  string
		fills []SlotFill
		kept  bool
	}
	lh6 := SlotFill{0, "Long Hump 6", 1}
	tests := []struct {
		name   string
		levels [NumFields]int
		rows   []row
	}{
		{"tech 3", [NumFields]int{3, 3, 3, 3, 3, 3}, []row{
			{"Mini Bomber", []SlotFill{lh6, {1, "Cherry Bomb", 2}}, false},
			{"Mini Bomber", []SlotFill{lh6, {1, "Smart Bomb", 2}}, false},    // every race but IS
			{"Mini Bomber", []SlotFill{lh6, {1, "Peerless Bomb", 2}}, false}, // every race but IS
			{"Mini Bomber", []SlotFill{lh6, {1, "LBU-17 Bomb", 1}}, false},
			{"Mini Bomber", []SlotFill{lh6, {1, "LBU-32 Bomb", 1}}, false},
			{"Mini Bomber", []SlotFill{lh6, {1, "Lady Finger Bomb", 1}}, true},
			{"Mini Bomber", []SlotFill{lh6, {1, "Hush-a-Boom", 1}}, true},                 // Mystery Trader
			{"Mini Bomber", []SlotFill{lh6, {1, "Retro Bomb", 1}}, true},                  // CA only
			{"Colony Ship", []SlotFill{lh6, {1, "Orbital Construction Module", 1}}, true}, // AR only
			{"Scout", []SlotFill{lh6, {2, "Multi Contained Munition", 1}}, true},          // Mystery Trader
		}},
		{"electronics 7", [NumFields]int{26, 26, 26, 26, 7, 26}, []row{
			{"Mini Bomber", []SlotFill{lh6, {1, "Cherry Bomb", 2}}, true},
			{"Mini Bomber", []SlotFill{lh6, {1, "LBU-17 Bomb", 1}}, false}, // electronics 8
			{"Mini Bomber", []SlotFill{lh6, {1, "LBU-32 Bomb", 1}}, false}, // electronics 10
			{"Mini Bomber", []SlotFill{lh6, {1, "Hush-a-Boom", 1}}, true},  // electronics 12
			{"Mini Bomber", []SlotFill{lh6, {1, "Retro Bomb", 1}}, true},
		}},
	}
	for _, tt := range tests {
		l := newTKLab(t, 3)
		l.g.Players[0].Research.Levels = tt.levels
		l.g.Players[0].Research.Current = Energy
		for k, r := range tt.rows {
			l.g.DesignSlots = append(l.g.DesignSlots, DesignSlot{Owner: 0, Slot: k, Design: l.design(r.hull, r.fills...)})
		}
		before := slices.Clone(l.g.Designs)
		r := l.turn(&seqRand{})
		for k, row := range tt.rows {
			got, was := r.Game.Designs[k], before[k]
			if row.kept {
				if !slices.Equal(slotNames(got), slotNames(was)) || got.Mass != was.Mass {
					t.Errorf("%s: %s %v: slots %v mass %d, want it kept: %v mass %d", tt.name, row.hull, row.fills[1].Part, slotNames(got), got.Mass, slotNames(was), was.Mass)
				}
				continue
			}
			bare, err := Components().NewDesign(row.hull, row.hull, []SlotFill{lh6})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(slotNames(got), slotNames(bare)) || !slices.Equal(got.SlotPos, bare.SlotPos) || got.Mass != bare.Mass {
				t.Errorf("%s: %s %v: slots %v at %v mass %d, want %v at %v mass %d", tt.name, row.hull, row.fills[1].Part, slotNames(got), got.SlotPos, got.Mass, slotNames(bare), bare.SlotPos, bare.Mass)
			}
		}
		if !slices.Equal(slotNames(l.g.Designs[0]), slotNames(before[0])) {
			t.Errorf("%s: the game passed in was changed", tt.name)
		}
	}
}

// TestMeasuredDesignCheckShipsAct: ships of a design whose bombs were
// removed do not bomb that year, and a kept part works (PARITY.md "Design
// parts dropped", MEASURED: "slots written back empty and those ships
// did not bomb").
func TestMeasuredDesignCheckShipsAct(t *testing.T) {
	pop := func(bomb string) int {
		l := newTKLab(t, 3)
		l.g.Players[0].Research.Levels = [NumFields]int{3, 3, 3, 3, 3, 3}
		pi := l.planet(1, 0, 920)
		if bomb != "" {
			d := l.design("Mini Bomber", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, bomb, 1})
			l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: d}}
			l.fleet(0, pi, 0, Stack{Design: d, Count: 10})
		}
		return l.turn(&seqRand{}).Game.Planets[pi].Population
	}
	none := pop("")
	if got := pop("Cherry Bomb"); got != none {
		t.Errorf("Cherry bombers above tech: population %d, want %d as with no bombers", got, none)
	}
	if got := pop("Lady Finger Bomb"); got >= none {
		t.Errorf("Lady Finger bombers within tech: population %d, want below %d", got, none)
	}
}

// TestAssumptionDesignCheck covers the design check's ASSUMPTION T15
// rows (NOT RUN): capacities lose what a removed part added, an emptied
// engine slot gets the basic engine at the slot's capacity, a misslotted
// part is removed with no mass change, a lesser-racial-trait part is
// checked like any other, starbase designs and other owners' designs are
// left alone, and fleets keep fuel and cargo.
func TestAssumptionDesignCheck(t *testing.T) {
	cat := Components()
	l := newTKLab(t, 3)
	l.g.Players[0].Research.Levels = [NumFields]int{3, 3, 3, 3, 3, 3}
	// Player 1 has weapons 26, and its own Cherry design stays
	// while player 0's identical one goes.
	l.g.Players[1].Research.Levels = [NumFields]int{3, 26, 3, 3, 3, 3}
	cargo := l.design("Medium Freighter", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Super Cargo Pod", 1})
	fuel := l.design("Scout", SlotFill{0, "Long Hump 6", 1}, SlotFill{2, "Super Fuel Tank", 1})
	tgd := l.design("Large Freighter", SlotFill{0, "Trans-Galactic Drive", 2}, SlotFill{1, "Cargo Pod", 2})
	lrt5 := l.design("Mini-Miner", SlotFill{0, "Long Hump 6", 1}, SlotFill{2, "Robo-Maxi-Miner", 1})
	odd, err := cat.NewDesignAnyKind("odd", "Frigate", []SlotFill{{0, "Long Hump 6", 1}, {2, "Cherry Bomb", 1}, {3, "Tritanium", 1}})
	if err != nil {
		t.Fatal(err)
	}
	l.g.Designs = append(l.g.Designs, odd)
	misslot := len(l.g.Designs) - 1
	fort := l.design("Orbital Fort", SlotFill{1, "Yakimora Light Phaser", 2})
	mine := l.design("Mini Bomber", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Cherry Bomb", 2})
	theirs := l.design("Mini Bomber", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Cherry Bomb", 2})
	l.g.DesignSlots = []DesignSlot{
		{Owner: 0, Slot: 0, Design: cargo}, {Owner: 0, Slot: 1, Design: fuel}, {Owner: 0, Slot: 2, Design: tgd},
		{Owner: 0, Slot: 4, Design: lrt5}, {Owner: 0, Slot: 5, Design: misslot},
		{Owner: 0, Starbase: true, Slot: 0, Design: fort}, {Owner: 0, Slot: 6, Design: mine},
		{Owner: 1, Slot: 0, Design: theirs},
	}
	before := slices.Clone(l.g.Designs)
	pi := l.planet(0, 0, 1000)
	f := l.fleet(0, pi, 0, Stack{Design: fuel, Count: 1}, Stack{Design: cargo, Count: 1})
	l.g.Fleets[f].Fuel = before[fuel].FuelCapacity + before[cargo].FuelCapacity
	l.g.Fleets[f].Cargo = Cargo{Minerals: Minerals{before[cargo].CargoCapacity, 0, 0}}
	r := l.turn(&seqRand{})
	got := r.Game.Designs

	if d := got[cargo]; d.CargoCapacity != before[cargo].CargoCapacity-100 || d.Mass != before[cargo].Mass-7 || len(d.Slots) != 1 {
		t.Errorf("Super Cargo Pod: cargo %d mass %d slots %v, want %d, %d, the engine only", d.CargoCapacity, d.Mass, slotNames(d), before[cargo].CargoCapacity-100, before[cargo].Mass-7)
	}
	if d := got[fuel]; d.FuelCapacity != before[fuel].FuelCapacity-500 || d.Mass != before[fuel].Mass-before[fuel].Slots[1].Part.Mass {
		t.Errorf("Super Fuel Tank: fuel %d mass %d, want %d, %d", d.FuelCapacity, d.Mass, before[fuel].FuelCapacity-500, before[fuel].Mass-before[fuel].Slots[1].Part.Mass)
	}
	want, err := cat.NewDesign("tgd", "Large Freighter", []SlotFill{{0, basicEngine, 2}, {1, "Cargo Pod", 2}})
	if err != nil {
		t.Fatal(err)
	}
	if d := got[tgd]; !slices.Equal(slotNames(d), slotNames(want)) || !slices.Equal(d.SlotPos, want.SlotPos) || d.Engine.Name != basicEngine || d.Engines != 2 ||
		d.Mass != want.Mass || d.FuelCapacity != want.FuelCapacity || d.CargoCapacity != want.CargoCapacity {
		t.Errorf("Trans-Galactic Drive: %v at %v engine %s ×%d mass %d fuel %d cargo %d, want %v at %v, %s ×2, %d, %d, %d",
			slotNames(d), d.SlotPos, d.Engine.Name, d.Engines, d.Mass, d.FuelCapacity, d.CargoCapacity, slotNames(want), want.SlotPos, basicEngine, want.Mass, want.FuelCapacity, want.CargoCapacity)
	}
	if d := got[lrt5]; len(d.Slots) != 1 || d.Mass != before[lrt5].Mass-240 {
		t.Errorf("Robo-Maxi-Miner (not with OBRM; construction 7): %v mass %d, want removed, mass %d", slotNames(d), d.Mass, before[lrt5].Mass-240)
	}
	if d := got[misslot]; !slices.Equal(slotNames(d), []string{"1 Long Hump 6", "1 Tritanium"}) || !slices.Equal(d.SlotPos, []int{0, 3}) || d.Mass != before[misslot].Mass {
		t.Errorf("misslotted Cherry: %v at %v mass %d, want the engine and Tritanium at [0 3], mass %d", slotNames(d), d.SlotPos, d.Mass, before[misslot].Mass)
	}
	if d := got[fort]; !slices.Equal(slotNames(d), slotNames(before[fort])) {
		t.Errorf("starbase design: %v, want kept", slotNames(d))
	}
	if d := got[mine]; len(d.Slots) != 1 {
		t.Errorf("player 0's Cherry design: %v, want the bombs removed", slotNames(d))
	}
	if d := got[theirs]; !slices.Equal(slotNames(d), slotNames(before[theirs])) {
		t.Errorf("player 1's Cherry design: %v, want kept at weapons 26", slotNames(d))
	}
	for _, fl := range r.Game.Fleets {
		if fl.ID == l.g.Fleets[f].ID && (fl.Fuel != l.g.Fleets[f].Fuel || fl.Cargo != l.g.Fleets[f].Cargo) {
			t.Errorf("fleet fuel %d cargo %+v, want %d, %+v", fl.Fuel, fl.Cargo, l.g.Fleets[f].Fuel, l.g.Fleets[f].Cargo)
		}
	}
}

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
// left alone, and fleets keep fuel and cargo (cargo above the new
// capacity is CONFIRMED by prediction, stars-elegy #200 DC-6).
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

// TestAssumptionDesignCheckEachField: the check compares every field. For
// each field a Scout carries a part needing that field and no other, one
// level above the owner's (every other field 26): the part is removed
// (an engine is back-filled with the basic engine); at the part's level
// it stays.
func TestAssumptionDesignCheckEachField(t *testing.T) {
	cat := Components()
	parts := [NumFields]SlotFill{
		Energy:       {2, "Wolverine Diffuse Shield", 1},
		Weapons:      {2, "Yakimora Light Phaser", 1},
		Propulsion:   {0, "Daddy Long Legs 7", 1},
		Construction: {2, "Strobnium", 1},
		Electronics:  {1, "Possum Scanner", 1},
		Biotech:      {2, "Organic Armor", 1},
	}
	for f, fill := range parts {
		c, _ := cat.Lookup(fill.Part)
		p, err := c.Part()
		if err != nil {
			t.Fatal(err)
		}
		for _, short := range []bool{true, false} {
			l := newTKLab(t, 3)
			l.g.Players[0].Research.Levels = [NumFields]int{26, 26, 26, 26, 26, 26}
			l.g.Players[0].Research.Levels[f] = p.TechReq[f]
			if short {
				l.g.Players[0].Research.Levels[f]--
			}
			fills := []SlotFill{fill}
			if fill.Slot != 0 {
				fills = append([]SlotFill{{0, "Long Hump 6", 1}}, fills...)
			}
			d := l.design("Scout", fills...)
			l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: d}}
			got := l.turn(&seqRand{}).Game.Designs[d]
			kept := slices.Contains(slotNames(got), "1 "+fill.Part)
			if kept == short {
				t.Errorf("field %d, %s, level %d: slots %v, want it kept %v", f, fill.Part, l.g.Players[0].Research.Levels[f], slotNames(got), !short)
			}
		}
	}
}

// TestAssumptionDesignCheckBeforeLevelUp: a part whose field rises to
// its level from stored research at the step 2.4 level-up is still
// removed that year (CONFIRMED by prediction, stars-elegy #200 DC-1); the
// check runs ahead of that level-up, with the tech from before the year's
// research (ASSUMPTION T15; DC-1, inferred).
func TestAssumptionDesignCheckBeforeLevelUp(t *testing.T) {
	l := newTKLab(t, 3)
	l.g.Players[0].Research.Levels = [NumFields]int{26, 13, 26, 26, 26, 26} // Cherry needs weapons 14
	l.g.Players[0].Research.Accumulated[Weapons] = 1 << 20
	d := l.design("Mini Bomber", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Cherry Bomb", 2})
	l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: d}}
	r := l.turn(&seqRand{})
	if lv := r.Game.Players[0].Research.Levels[Weapons]; lv < 14 {
		t.Fatalf("weapons %d at the end of the year, want 14 or more from the step 2.4 level-up", lv)
	}
	if got := r.Game.Designs[d]; slices.Contains(slotNames(got), "2 Cherry Bomb") {
		t.Errorf("slots %v, want the Cherry Bombs removed before the level-up", slotNames(got))
	}
}

// TestAssumptionDesignCheckAfterOrders: the check runs at step 1a.2,
// after the year's orders, so a split that year shares cargo by the
// capacities before the check: the split-off ship's Super Cargo Pod
// (above tech) still counts, and the design loses it afterwards
// (ASSUMPTION T15).
func TestAssumptionDesignCheckAfterOrders(t *testing.T) {
	l := newTKLab(t, 3)
	l.g.Players[0].Research.Levels = [NumFields]int{3, 3, 3, 3, 3, 3}
	pi := l.planet(0, 0, 1000)
	super := l.design("Medium Freighter", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Super Cargo Pod", 1})
	plain := l.design("Medium Freighter", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Cargo Pod", 1})
	l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: super}, {Owner: 0, Slot: 1, Design: plain}}
	fi := l.fleet(0, pi, 0, Stack{Design: super, Count: 1}, Stack{Design: plain, Count: 1})
	l.g.Fleets[fi].Cargo = Cargo{Minerals: Minerals{300, 0, 0}}
	src := l.g.Fleets[fi].ID
	cs, cp := l.g.Designs[super].CargoCapacity, l.g.Designs[plain].CargoCapacity
	want := 300 * cs / (cs + cp)
	if want == 300*(cs-100)/(cs-100+cp) {
		t.Fatal("the two orders give the same share")
	}
	orders := []PlayerOrders{{Player: 0, GameID: l.g.ID, Year: l.g.Year, Orders: []Order{SplitOrder{Fleet: src, Ships: []Stack{{Design: super, Count: 1}}}}}}
	r, err := GenerateTurn(withRules(l.g), orders, &seqRand{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range r.Game.Fleets {
		if f.Owner == 0 && f.ID != src {
			found = true
			if f.Cargo.Minerals[0] != want {
				t.Errorf("split-off ship's ironium %d, want %d (Super Cargo Pod still counted)", f.Cargo.Minerals[0], want)
			}
		}
	}
	if !found {
		t.Fatal("no split-off fleet")
	}
	if got := r.Game.Designs[super]; got.CargoCapacity != cs-100 {
		t.Errorf("design cargo %d, want %d after the check", got.CargoCapacity, cs-100)
	}
}

// TestConfirmedDesignCheckReceived: a design received with a fleet
// transfer is not checked against the recipient's tech in the transfer
// year or the next (CONFIRMED by prediction, stars-elegy #200, DC-5;
// DC-2 MEASURED), while the recipient's own identical design loses the
// part; the giver keeps its own copy (MEASURED, DC-2 and DC-5).
// ASSUMPTION T15 (NOT RUN): the giver's original is still checked against
// the giver's tech (here lowered before the second year), and the copy
// stays exempt in the third year.
func TestConfirmedDesignCheckReceived(t *testing.T) {
	cat := Components()
	c, _ := cat.Lookup("Possum Scanner")
	p, err := c.Part()
	if err != nil {
		t.Fatal(err)
	}
	low := [NumFields]int{26, 26, 26, 26, p.TechReq[Electronics] - 1, 26}
	l := newTKLab(t, 3)
	l.g.Players[1].Research.Levels = low
	l.g.Players[1].Relations[0] = RelationNeutral
	pi := l.planet(0, 0, 1000)
	scout := func() int {
		return l.design("Scout", SlotFill{0, "Long Hump 6", 1}, SlotFill{1, "Possum Scanner", 1})
	}
	given, kept, own := scout(), scout(), scout()
	l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: given}, {Owner: 0, Slot: 1, Design: kept}, {Owner: 1, Slot: 0, Design: own}}
	fi := l.fleet(0, pi, 0, Stack{Design: given, Count: 1})
	l.g.Fleets[fi].Task = Task{Kind: TaskTransferFleet, Player: 1}
	l.fleet(0, pi, 0, Stack{Design: kept, Count: 1})

	has := func(g Game, d int) bool { return slices.Contains(slotNames(g.Designs[d]), "1 Possum Scanner") }
	g := l.g
	copyOf := -1
	for year := 1; year <= 3; year++ {
		if year == 2 {
			// The giver loses the tech too: its original is checked.
			g.Players[0].Research.Levels = low
		}
		r, err := GenerateTurn(withRules(g), nil, &seqRand{})
		if err != nil {
			t.Fatal(err)
		}
		g = r.Game
		if year == 1 {
			for _, ds := range g.DesignSlots {
				if ds.Owner == 1 && ds.Received {
					copyOf = ds.Design
				}
			}
			if copyOf < 0 {
				t.Fatalf("no received slot after the transfer: %+v", g.DesignSlots)
			}
			if has(g, own) {
				t.Errorf("year 1: the recipient's own design kept the Possum Scanner")
			}
			if d, ok := g.PlayerDesign(0, false, 0); !ok || d != given || !has(g, given) {
				t.Errorf("year 1: the giver's slot 0 holds %d %v, want its own copy %d with the scanner", d, ok, given)
			}
		}
		if !has(g, copyOf) {
			t.Errorf("year %d: the received copy lost the Possum Scanner: %v", year, slotNames(g.Designs[copyOf]))
		}
		if wantGiver := year < 2; has(g, kept) != wantGiver {
			t.Errorf("year %d: the giver's original kept the scanner %v, want %v", year, has(g, kept), wantGiver)
		}
	}
}

// TestDesignSlotReceivedCleared: a design order that replaces a received
// slot's design clears the mark, so the new design is checked like any
// other (ASSUMPTION T15).
func TestDesignSlotReceivedCleared(t *testing.T) {
	l := newTKLab(t, 3)
	d := l.design("Scout", SlotFill{0, "Long Hump 6", 1})
	l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: d, Received: true}}
	g := withRules(l.g)
	if err := (DesignOrder{Slot: 0, Name: "S", Hull: "Scout", Fills: []SlotFill{{0, "Long Hump 6", 1}}}).apply(&g, 0, &Applied{}); err != nil {
		t.Fatal(err)
	}
	if g.DesignSlots[0].Received {
		t.Error("the replaced slot is still marked received")
	}
	if !l.g.DesignSlots[0].Received {
		t.Error("the game passed in was changed")
	}
}

// TestConfirmedDesignCheckBeforeLoads: the check runs before the
// waypoint-0 load tasks, so a ship whose Super Cargo Pod is removed loads
// nothing that year while a Cargo Pod ship on the same task loads; a ship
// already holding more than the new capacity keeps its cargo (CONFIRMED
// by prediction, stars-elegy #200 DC-6; the new capacity of 0 is
// inferred there from the stripped design).
func TestConfirmedDesignCheckBeforeLoads(t *testing.T) {
	l := newTKLab(t, 3)
	l.g.Players[0].Research.Levels = [NumFields]int{26, 26, 26, 8, 26, 26} // the Super Cargo Pod needs construction 9
	pi := l.planet(0, 0, 100000)
	l.g.Planets[pi].Surface = Minerals{1000, 0, 0}
	super := l.design("Scout", SlotFill{0, "Long Hump 6", 1}, SlotFill{2, "Super Cargo Pod", 1})
	plain := l.design("Scout", SlotFill{0, "Long Hump 6", 1}, SlotFill{2, "Cargo Pod", 1})
	l.g.DesignSlots = []DesignSlot{{Owner: 0, Slot: 0, Design: super}, {Owner: 0, Slot: 1, Design: plain}}
	load := Task{Kind: TaskTransport}
	load.Transport[Ironium] = Transport{Action: LoadAll}
	p := l.g.Planets[pi]
	var fleets []int
	for _, d := range []int{super, plain, super} {
		fi := l.fleet(0, pi, 0, Stack{Design: d, Count: 1})
		l.g.Fleets[fi].Waypoints = []Waypoint{{Pos: p.Pos, Target: TargetPlanet, ID: p.ID, Task: load}}
		l.g.Fleets[fi].Task = load
		fleets = append(fleets, fi)
	}
	l.g.Fleets[fleets[2]].Cargo.Minerals[Ironium] = 80
	if l.g.Designs[super].CargoCapacity == 0 || l.g.Designs[plain].CargoCapacity == 0 {
		t.Fatal("a pod adds no cargo capacity")
	}
	r := l.turn(&seqRand{})
	cargo := func(fi int) int {
		id := l.g.Fleets[fi].ID
		for _, f := range r.Game.Fleets {
			if f.ID == id {
				return f.Cargo.Minerals[Ironium]
			}
		}
		t.Fatalf("fleet %d is gone", id)
		return 0
	}
	if c := r.Game.Designs[super].CargoCapacity; c != 0 {
		t.Fatalf("design cargo capacity %d, want 0 with the pod removed", c)
	}
	if got := cargo(fleets[0]); got != 0 {
		t.Errorf("Super Cargo Pod ship loaded %d, want 0", got)
	}
	if got := cargo(fleets[1]); got == 0 {
		t.Error("Cargo Pod ship loaded 0, want a load")
	}
	if got := cargo(fleets[2]); got != 80 {
		t.Errorf("Super Cargo Pod ship holding 80 kT has %d, want 80 kept", got)
	}
}

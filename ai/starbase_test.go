package ai

import (
	"reflect"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

func fills(o engine.Order) []engine.SlotFill { return o.(engine.DesignOrder).Fills }

func fill(slot int, part string, n int) engine.SlotFill {
	return engine.SlotFill{Slot: slot, Part: part, Count: n}
}

func TestStarbaseDesignsFirstYear(t *testing.T) {
	in := StarbaseInput{
		Personality: Rototill,
		Year:        2400,
		Race:        engine.Race{PRT: engine.PRTClaimAdjuster},
	}
	in.Designs[0] = &SlotDesign{Hull: spaceStation, Name: "Starbase", Created: 2400, Picture: 0}
	in.Built[0] = true
	// Pictures are free, so only name draws: one per design, no clashes.
	r := &script{t: t, draws: []int{0, 1, 2, 3}}
	orders, made := StarbaseDesigns(in, r)
	var slots []int
	for _, o := range orders {
		slots = append(slots, o.(engine.DesignOrder).Slot)
	}
	if !reflect.DeepEqual(slots, []int{2, 4, 1, 3}) {
		t.Fatalf("slots %v, want 2 4 1 3", slots)
	}
	wantPics := []int{1, 2, 0, 1}
	for i, m := range made {
		if m.Picture != wantPics[i] || m.Created != 2400 || m.Name != starbaseNames[i] {
			t.Errorf("design %d: %+v", i, m)
		}
	}
	// Variant 2 Space Station: 16-slots halved, small slots at their max.
	v2 := fills(orders[0])
	want2 := []engine.SlotFill{
		fill(0, "Battle Computer", 1), fill(1, "Alpha Torpedo", 8), fill(2, "Mole-skin Shield", 8),
		fill(3, "Alpha Torpedo", 8), fill(4, "Tritanium", 8), fill(5, "Mole-skin Shield", 8),
		fill(6, "Battle Computer", 3), fill(7, "Laser", 8), fill(8, "Battle Computer", 3),
		fill(9, "Laser", 8), fill(10, "Battle Computer", 1), fill(11, "Tritanium", 8),
	}
	if !reflect.DeepEqual(v2, want2) {
		t.Errorf("variant 2:\n got %v\nwant %v", v2, want2)
	}
	// Variant 3 keeps every maximum.
	for _, f := range fills(orders[1]) {
		if f.Count != map[int]int{0: 1, 6: 3, 8: 3, 10: 1}[f.Slot] && f.Count != 16 {
			t.Errorf("variant 3 slot %d count %d", f.Slot, f.Count)
		}
	}
	// Variant 1 Orbital Fort: 12-slots quartered; the one-item slot keeps 1.
	want1 := []engine.SlotFill{
		fill(0, "Battle Computer", 1), fill(1, "Alpha Torpedo", 3), fill(2, "Tritanium", 3),
		fill(3, "Laser", 3), fill(4, "Mole-skin Shield", 3),
	}
	if v1 := fills(orders[2]); !reflect.DeepEqual(v1, want1) {
		t.Errorf("variant 1 fort:\n got %v\nwant %v", v1, want1)
	}
	// Next year nothing is missing: no orders, no draws.
	for _, m := range made {
		in.Designs[m.Slot] = &SlotDesign{Hull: slotHull(m.Slot), Name: m.Name, Created: m.Created, Picture: m.Picture}
	}
	in.Year = 2401
	if orders, _ := StarbaseDesigns(in, &script{t: t}); len(orders) != 0 {
		t.Errorf("2401: %d orders", len(orders))
	}
}

// Variant 1 Space Station: a three-item electrical slot loses one.
func TestStarbaseVariantOne(t *testing.T) {
	in := StarbaseInput{Personality: Robotoid, Year: 2450, Race: engine.Race{PRT: engine.PRTHyperExpansion}}
	for _, k := range []int{0, 2, 4} {
		in.Designs[k] = &SlotDesign{Hull: spaceStation, Name: starbaseNames[k], Created: 2400, Picture: k / 2}
	}
	for _, k := range []int{1, 3} {
		in.Designs[k] = &SlotDesign{Hull: orbitalFort, Name: starbaseNames[k], Created: 2410, Picture: k / 2}
	}
	in.Built[0] = true
	// Family B Space Stations 5, 7, 9; the forts are 40 years old too,
	// so 6 and 8 follow. Pictures: station 3 is free, then all four are
	// used and each costs a draw before the name's draw.
	r := &script{t: t, draws: []int{5, 3, 6, 2, 7, 8, 9}}
	orders, made := StarbaseDesigns(in, r)
	var slots []int
	for _, o := range orders {
		slots = append(slots, o.(engine.DesignOrder).Slot)
	}
	if !reflect.DeepEqual(slots, []int{5, 7, 9, 6, 8}) {
		t.Fatalf("slots %v", slots)
	}
	if made[0].Picture != 3 || made[1].Picture != 3 || made[2].Picture != 2 || made[3].Picture != 2 || made[4].Picture != 3 {
		t.Errorf("pictures %+v", made)
	}
	for _, f := range fills(orders[0]) {
		want := map[int]int{0: 1, 6: 2, 8: 2, 10: 1}[f.Slot]
		if want == 0 {
			want = 4
		}
		if f.Count != want {
			t.Errorf("variant 1 station slot %d: count %d, want %d", f.Slot, f.Count, want)
		}
	}
}

// Step 2 waits for an old family nobody builds from, and is off before
// year index 50.
func TestStarbaseFamilySwitchBlocked(t *testing.T) {
	in := StarbaseInput{Personality: Rototill, Year: 2460, Race: engine.Race{PRT: engine.PRTClaimAdjuster}}
	for k := range 10 {
		in.Designs[k] = &SlotDesign{Hull: slotHull(k), Name: starbaseNames[k], Created: 2400, Picture: k % 4}
	}
	in.Designs[4].Created = 2405 // newest station: family A, so B is "other"
	in.Built[7] = true           // a starbase of family B
	in.Designs[8].Created = 2430 // newest fort: family B, only 30 years old
	if orders, _ := StarbaseDesigns(in, &script{t: t}); len(orders) != 0 {
		t.Errorf("blocked switch made %d orders", len(orders))
	}
	in.Built[7] = false
	in.Year = 2449
	if orders, _ := StarbaseDesigns(in, &script{t: t}); len(orders) != 0 {
		t.Errorf("year index 49 made %d orders", len(orders))
	}
}

// Cybertron keeps the mass driver in a variant-1 Orbital Fort; another
// personality with the same race would leave that slot empty.
func TestStarbaseCybertronMassDriver(t *testing.T) {
	in := StarbaseInput{Personality: Cybertron, Year: 2400, Race: engine.Race{PRT: engine.PRTPacketPhysics}}
	in.Levels[engine.Energy] = 4
	in.Designs[0] = &SlotDesign{Hull: spaceStation, Created: 2400}
	in.Designs[2] = &SlotDesign{Hull: spaceStation, Created: 2400, Picture: 1}
	in.Designs[3] = &SlotDesign{Hull: orbitalFort, Created: 2400}
	orders, _ := StarbaseDesigns(in, &script{t: t, draws: []int{0}})
	if f := fills(orders[0]); f[0] != (engine.SlotFill{Slot: 0, Part: "Mass Driver 5", Count: 1}) {
		t.Errorf("Cybertron fort slot 0: %+v", f[0])
	}
	in.Personality = Robotoid
	orders, _ = StarbaseDesigns(in, &script{t: t, draws: []int{0}})
	if f := fills(orders[0]); f[0].Slot == 0 {
		t.Errorf("Robotoid fort slot 0 kept %+v", f[0])
	}
}

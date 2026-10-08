package ai

import (
	"slices"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// Every Robotoid list has one class per slot of the hull it is used with
// (robotoid.md "Robotoid class lists").
func TestRobotoidListsFitHulls(t *testing.T) {
	hulls := map[string][]int{
		"Meta Morph":  {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
		"Privateer":   {14, 15},
		"Destroyer":   {16, 17, 18, 19, 20, 21, 22, 23},
		"B-52 Bomber": {24, 25},
		"Frigate":     {26},
		"Battleship":  {27, 28, 29, 30, 31, 32, 33, 34, 36},
		"Nubian":      {37},
	}
	cat := engine.Components()
	for name, lists := range hulls {
		c, _ := cat.Lookup(name)
		h, err := c.Hull()
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range lists {
			if len(robotoidLists[l]) != len(h.Slots) {
				t.Errorf("list %d has %d classes, %s has %d slots", l, len(robotoidLists[l]), name, len(h.Slots))
			}
		}
	}
}

func heView(year int, levels [engine.NumFields]int) *View {
	race := engine.Race{PRT: engine.PRTHyperExpansion, LRT: engine.LRTs{ImprovedFuelEfficiency: true}} // AI.md §3: HE has IFE
	return &View{Year: year, Self: engine.Player{Race: race, Research: engine.ResearchState{Levels: levels}}}
}

func designOrders(res Result) []engine.Order {
	var out []engine.Order
	for _, o := range res.Orders {
		switch o.(type) {
		case engine.DesignOrder, engine.DeleteDesignOrder:
			out = append(out, o)
		}
	}
	return out
}

// Step 1: with propulsion 2 and construction 4, slot 11 gets a Privateer
// of list 14; slot 12 waits for construction 7.
func TestRobotoidFirstFreighter(t *testing.T) {
	var res Result
	v := heView(2415, [engine.NumFields]int{0, 0, 2, 4, 0, 0})
	s := newShipDesigns(v, &script{t: t, draws: []int{0}}, &res)
	s.robotoidDesigns(Expert, nil)
	ds := designOrders(res)
	if len(ds) != 1 {
		t.Fatalf("orders %+v", ds)
	}
	d := ds[0].(engine.DesignOrder)
	if d.Slot != 11 || d.Hull != "Privateer" || d.Starbase {
		t.Errorf("design %+v", d)
	}
	if res.Designs[0].Picture != 0 || res.Designs[0].Created != 2415 {
		t.Errorf("stored %+v", res.Designs[0])
	}
}

// Step 2 before Nubians are available: the Nubian fails without a draw,
// then Destroyer list 16 + Random(4), and the first success ends the
// rounds. Step 1 has already made slot 11's Privateer.
func TestRobotoidSlot14Destroyer(t *testing.T) {
	var res Result
	v := heView(2442, [engine.NumFields]int{2, 5, 6, 6, 6, 0})
	// Draws: slot 11's name, list 16 + 1 (list 17 needs class 3: no part,
	// so the round fails), next round list 16 + 0, slot 14's name.
	r := &script{t: t, draws: []int{0, 1, 0, 0}}
	s := newShipDesigns(v, r, &res)
	s.robotoidDesigns(Expert, nil)
	ds := designOrders(res)
	if len(ds) != 2 {
		t.Fatalf("orders %+v", ds)
	}
	d := ds[1].(engine.DesignOrder)
	if d.Slot != 14 || d.Hull != "Destroyer" || len(r.draws) != 0 {
		t.Errorf("design %+v, draws left %v", d, r.draws)
	}
	if want := []int{8, 4, 4, 16}; !slices.Equal(r.bounds, want) {
		t.Errorf("draws Random%v, want Random%v", r.bounds, want)
	}
}

// Step 7: at harder or expert, with the tech and no slot-0 ships, slot 0
// is deleted and replaced by a Frigate of list 26.
func TestRobotoidFrigate(t *testing.T) {
	var res Result
	v := heView(2435, [engine.NumFields]int{6, 0, 6, 6, 5, 4})
	v.Ships = []Design{{Slot: 0, Design: engine.Design{Name: "Scout", Hull: engine.Hull{Name: "Scout"}}, Created: 2400}}
	s := newShipDesigns(v, top{}, &res)
	s.robotoidDesigns(Harder, map[int]int{})
	ds := designOrders(res)
	n := len(ds)
	if n < 2 || ds[n-2] != (engine.DeleteDesignOrder{Slot: 0}) {
		t.Fatalf("orders %+v", ds)
	}
	if d := ds[n-1].(engine.DesignOrder); d.Slot != 0 || d.Hull != "Frigate" {
		t.Errorf("design %+v", d)
	}
	// Standard level: no Frigate.
	res = Result{}
	s = newShipDesigns(v, top{}, &res)
	s.robotoidDesigns(Standard, map[int]int{})
	for _, o := range designOrders(res) {
		if d, ok := o.(engine.DesignOrder); ok && d.Slot == 0 {
			t.Errorf("standard level made %+v", d)
		}
	}
}

// Ageing: the newest design is chosen first; an old design with no ships
// is deleted, one with ships is marked obsolete.
func TestAgeGroup(t *testing.T) {
	var res Result
	v := heView(2470, [engine.NumFields]int{})
	v.Ships = []Design{
		{Slot: 2, Design: engine.Design{Hull: engine.Hull{Name: "Meta Morph"}}, Created: 2410},
		{Slot: 3, Design: engine.Design{Hull: engine.Hull{Name: "Meta Morph"}}, Created: 2415},
		{Slot: 4, Design: engine.Design{Hull: engine.Hull{Name: "Meta Morph"}}, Created: 2450},
	}
	s := newShipDesigns(v, top{}, &res)
	obsolete := map[int]bool{}
	newest, ships := s.ageGroup([]int{2, 3, 4, 5}, 50, map[int]int{3: 7, 4: 2}, obsolete)
	if newest != 4 || ships != 9 || !obsolete[3] || obsolete[4] || s.slots[2].present || !s.slots[3].present {
		t.Errorf("newest %d ships %d obsolete %v slots %+v", newest, ships, obsolete, s.slots[2:5])
	}
	if ds := designOrders(res); len(ds) != 1 || ds[0] != (engine.DeleteDesignOrder{Slot: 2}) {
		t.Errorf("orders %+v", ds)
	}
}

// Every Cybertron list has one class per slot of its hull.
func TestCybertronListsFitHulls(t *testing.T) {
	cat := engine.Components()
	for i, l := range cybertronLists {
		c, ok := cat.Lookup(l.hull)
		if !ok {
			t.Fatalf("list %d: no hull %s", i, l.hull)
		}
		h, _ := c.Hull()
		if len(l.classes) != len(h.Slots) {
			t.Errorf("list %d has %d classes, %s has %d slots", i, len(l.classes), l.hull, len(h.Slots))
		}
	}
}

func ppView(year int, levels [engine.NumFields]int) *View {
	race := engine.Race{PRT: engine.PRTPacketPhysics, LRT: engine.LRTs{ImprovedFuelEfficiency: true}} // AI.md §3: PP has IFE
	return &View{Year: year, Self: engine.Player{Race: race, Research: engine.ResearchState{Levels: levels}}}
}

// Cybertron step 1: at expert after year index 5 a Scout in slot 0 with no
// ships is deleted; the Frigate is then tried every year.
func TestCybertronSlotZero(t *testing.T) {
	var res Result
	v := ppView(2406, [engine.NumFields]int{})
	v.Ships = []Design{{Slot: 0, Design: engine.Design{Hull: engine.Hull{Name: "Scout"}}, Created: 2400}}
	s := newShipDesigns(v, top{}, &res)
	s.cybertronDesigns(Expert, 6, map[int]int{})
	if ds := designOrders(res); len(ds) != 1 || ds[0] != (engine.DeleteDesignOrder{Slot: 0}) {
		t.Errorf("orders %+v", ds)
	}
	res = Result{}
	s = newShipDesigns(v, top{}, &res)
	s.cybertronDesigns(Expert, 5, map[int]int{})
	if ds := designOrders(res); len(ds) != 0 {
		t.Errorf("year index 5: %+v", ds)
	}
}

// A range tries its lists without repeats: Random(m) over the lists left.
func TestCybertronRange(t *testing.T) {
	var res Result
	v := ppView(2431, [engine.NumFields]int{})
	r := &script{t: t, draws: []int{4, 3, 2, 1, 0}}
	s := newShipDesigns(v, r, &res)
	// At tech 0 no Destroyer list can be built (class 8 needs Fuel Mizer).
	if s.cyberRange(4, 0, 4) {
		t.Fatal("built at tech 0")
	}
	if want := []int{5, 4, 3, 2, 1}; !slices.Equal(r.bounds, want) {
		t.Errorf("draws Random%v, want Random%v", r.bounds, want)
	}
}

// Step 2 at the 2431 tech: Destroyer list 0 is tried first (y ≤ 75).
func TestCybertronDestroyer(t *testing.T) {
	var res Result
	v := ppView(2431, [engine.NumFields]int{3, 5, 2, 3, 3, 3})
	s := newShipDesigns(v, top{}, &res)
	s.cybertronDesigns(Expert, 31, map[int]int{})
	var slots []int
	for _, o := range designOrders(res) {
		if d, ok := o.(engine.DesignOrder); ok {
			slots = append(slots, d.Slot)
			if d.Slot == 4 && d.Hull != "Destroyer" {
				t.Errorf("slot 4: %+v", d)
			}
		}
	}
	if !slices.Contains(slots, 4) || !slices.Contains(slots, 14) {
		t.Errorf("design slots %v, want 4 and 14 (AIX 2431)", slots)
	}
}

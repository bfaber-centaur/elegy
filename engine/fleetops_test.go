package engine

import (
	"reflect"
	"testing"
)

// opsGame has two designs and no fleets.
func opsGame() *Game {
	return &Game{
		Players: make([]Player, 2),
		Designs: []Design{testDesign(tFrigate, 10, Slot{tLaser, 1}), testDesign(Hull{Armor: 20, CargoCapacity: 100}, 20)},
	}
}

func TestConfirmedMergeDamageDilution(t *testing.T) {
	// ORDERS.md "Merge" (FO): 10 ships at 100 units/50% merged with 10 at
	// 200 units/20% gave 45 units at 35% (D = 5 and 2; units divided by
	// all 20 ships).
	a := Stack{Count: 10, Damage: Damage{Pct: 50, Units: 100}}
	b := Stack{Count: 10, Damage: Damage{Pct: 20, Units: 200}}
	if got := mergeDamage(a, b, true); got != (Damage{Pct: 35, Units: 45}) {
		t.Errorf("legacy dilution: %+v, want 35%% 45 units", got)
	}
	// Switched off: divided by the 7 damaged ships, ceil(900/7) = 129.
	if got := mergeDamage(a, b, false); got != (Damage{Pct: 35, Units: 129}) {
		t.Errorf("no dilution: %+v, want 35%% 129 units", got)
	}
	// One damaged stack keeps its units: D = 5 of 20 ships, 25%.
	if got := mergeDamage(a, Stack{Count: 10}, true); got != (Damage{Pct: 25, Units: 100}) {
		t.Errorf("one damaged: %+v, want 25%% 100 units", got)
	}
}

func TestConfirmedMergeTaskOverflow(t *testing.T) {
	// ORDERS.md "Merge" (FO), the LEGACY BUG: 32000+767 → 32767 kept;
	// 32000+768 and 32000+1000 → no ships, cargo and fuel stay. Elegy's
	// chosen rule (switch off) holds the stack to 32766.
	for _, tt := range []struct {
		add            int
		legacy, chosen int // ships after the merge; 0 = no ships at all
	}{{767, 32767, 32766}, {768, 0, 32766}, {1000, 0, 32766}} {
		for _, overflow := range []bool{true, false} {
			dst := Fleet{Stacks: []Stack{{Design: 0, Count: 32000}}, Fuel: 10, Cargo: Cargo{Minerals: Minerals{5, 0, 0}}}
			src := Fleet{Stacks: []Stack{{Design: 0, Count: tt.add}}, Fuel: 20, Cargo: Cargo{Colonists: 7}}
			absorb(&dst, &src, overflow)
			want := tt.chosen
			if overflow {
				want = tt.legacy
			}
			got := 0
			for _, s := range dst.Stacks {
				got += s.Count
			}
			if got != want || dst.Fuel != 30 || dst.Cargo != (Cargo{Minerals: Minerals{5, 0, 0}, Colonists: 7}) {
				t.Errorf("32000+%d overflow=%v: %d ships, fuel %d, cargo %+v; want %d, 30, 5 Fe and 7 colonists",
					tt.add, overflow, got, dst.Fuel, dst.Cargo, want)
			}
		}
	}
}

func TestConfirmedMergeTask(t *testing.T) {
	// ORDERS.md "Merge" (FO-01..07): ships add per design, cargo and fuel
	// add into the target, which keeps its id; a target elsewhere refuses
	// the task and clears it, both fleets unchanged.
	g := opsGame()
	g.Fleets = []Fleet{
		{ID: 1, Owner: 0, Pos: Point{10, 10}, Stacks: []Stack{{Design: 0, Count: 2}}, Fuel: 50},
		{ID: 2, Owner: 0, Pos: Point{10, 10}, Stacks: []Stack{{Design: 0, Count: 3}, {Design: 1, Count: 1}}, Fuel: 40,
			Cargo: Cargo{Minerals: Minerals{0, 30, 0}}, Task: Task{Kind: TaskMerge, Fleet: 1}},
		{ID: 3, Owner: 0, Pos: Point{205, 10}, Stacks: []Stack{{Design: 1, Count: 1}}, Task: Task{Kind: TaskMerge, Fleet: 1}},
	}
	events := g.loadPass()
	if len(g.Fleets) != 2 || g.Fleets[0].ID != 1 {
		t.Fatalf("fleets %+v, want fleets 1 and 3", g.Fleets)
	}
	f := g.Fleets[0]
	if !reflect.DeepEqual(f.Stacks, []Stack{{Design: 0, Count: 5}, {Design: 1, Count: 1}}) || f.Fuel != 90 || f.Cargo.Minerals != (Minerals{0, 30, 0}) {
		t.Errorf("merged fleet %+v", f)
	}
	if far := g.Fleets[1]; far.Task.Kind != TaskNone || far.Stacks[0].Count != 1 {
		t.Errorf("fleet 3 %+v: want its task cleared and its ship kept", far)
	}
	if want := []Event{{Kind: EventMergeRefused, Player: 0, Planet: -1, Fleet: 3}}; !reflect.DeepEqual(events, want) {
		t.Errorf("events %+v, want %+v", events, want)
	}
}

func TestPredictionMergeTaskForeignTarget(t *testing.T) {
	// TAKEOVER.md "Other waypoint tasks": the ordering fleet merges into
	// its own fleet. ASSUMPTION O2: another player's fleet is refused.
	g := opsGame()
	g.Fleets = []Fleet{
		{ID: 1, Owner: 1, Stacks: []Stack{{Design: 0, Count: 2}}},
		{ID: 2, Owner: 0, Stacks: []Stack{{Design: 0, Count: 3}}, Task: Task{Kind: TaskMerge, Fleet: 1}},
	}
	g.loadPass()
	if len(g.Fleets) != 2 || g.Fleets[0].Stacks[0].Count != 2 || g.Fleets[1].Task.Kind != TaskNone {
		t.Errorf("fleets %+v", g.Fleets)
	}
}

func TestPredictionMergeOrder(t *testing.T) {
	// ORDERS.md "Merge": the merge order joins co-located fleets of the
	// owner, holding each design to 32766 (BINARY-ONLY); ownership is
	// checked on every order (chosen rule).
	g := opsGame()
	g.Fleets = []Fleet{
		{ID: 1, Owner: 0, Stacks: []Stack{{Design: 0, Count: 32000}}},
		{ID: 2, Owner: 0, Stacks: []Stack{{Design: 0, Count: 1000}, {Design: 1, Count: 2}}, Fuel: 9},
		{ID: 3, Owner: 1, Stacks: []Stack{{Design: 0, Count: 1}}},
		{ID: 4, Owner: 0, Pos: Point{1, 0}, Stacks: []Stack{{Design: 0, Count: 1}}},
	}
	if err := g.MergeFleets(0, 1, []int{3}); err == nil {
		t.Error("merging another player's fleet: want an error")
	}
	if err := g.MergeFleets(0, 1, []int{4}); err == nil {
		t.Error("merging a fleet elsewhere: want an error")
	}
	if err := g.MergeFleets(0, 1, []int{2}); err != nil {
		t.Fatal(err)
	}
	if len(g.Fleets) != 3 || !reflect.DeepEqual(g.Fleets[0].Stacks, []Stack{{Design: 0, Count: 32766}, {Design: 1, Count: 2}}) || g.Fleets[0].Fuel != 9 {
		t.Errorf("fleets %+v", g.Fleets)
	}
}

func TestConfirmedDesignLegality(t *testing.T) {
	// ORDERS.md "Design legality (Mystery Trader parts kept)": the original
	// keeps a part the owner is not entitled to (Anti Matter Torpedo with no
	// Mystery Trader items; the same path keeps race-restricted parts) and
	// drops only parts above its research tech (LEGACY BUG). Elegy's chosen
	// rule drops every part the owner is not entitled to.
	c := Components()
	fills := []SlotFill{
		{Slot: 0, Part: "Quick Jump 5", Count: 1},
		{Slot: 1, Part: "Anti Matter Torpedo", Count: 1},
		{Slot: 2, Part: "Mini Gun", Count: 1}, // Inner Strength only
		{Slot: 3, Part: "Laser", Count: 1},
	}
	var top [NumFields]int
	for f := range top {
		top[f] = MaxTechLevel
	}
	names := func(d Design) []string {
		var n []string
		for _, s := range d.Slots {
			n = append(n, s.Part.Name)
		}
		return n
	}
	for _, tt := range []struct {
		name     string
		techOnly bool
		trader   map[string]bool
		want     []string
	}{
		{"legacy keeps", true, nil, []string{"Quick Jump 5", "Anti Matter Torpedo", "Mini Gun", "Laser"}},
		{"chosen drops", false, nil, []string{"Quick Jump 5", "Laser"}},
		{"chosen, trader item owned", false, map[string]bool{"Anti Matter Torpedo": true}, []string{"Quick Jump 5", "Anti Matter Torpedo", "Laser"}},
	} {
		d, err := c.readDesign("D", "Destroyer", fills, Race{}, top, tt.trader, tt.techOnly)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got := names(d); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: parts %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestPredictionDesignTechStrip(t *testing.T) {
	// ORDERS.md "Design legality (tech strip)" (BINARY-ONLY): a part above
	// the owner's tech is dropped under both settings, and the stored mass
	// is that of the parts that remain. ASSUMPTION O3: a design whose hull
	// is not available is rejected.
	c := Components()
	var lv [NumFields]int
	lv[Construction] = 3
	fills := []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}, {Slot: 1, Part: "Beta Torpedo", Count: 1}}
	want, err := c.NewDesign("D", "Destroyer", fills[:1])
	if err != nil {
		t.Fatal(err)
	}
	for _, techOnly := range []bool{true, false} {
		d, err := c.readDesign("D", "Destroyer", fills, Race{}, lv, nil, techOnly)
		if err != nil || len(d.Slots) != 1 || d.Mass != want.Mass {
			t.Errorf("techOnly=%v: %+v, %v; want only the engine, mass %d", techOnly, d.Slots, err, want.Mass)
		}
	}
	if _, err := c.ReadDesign("D", "Destroyer", fills, Race{}, [NumFields]int{}, nil); err == nil {
		t.Error("hull above tech: want an error")
	}
}

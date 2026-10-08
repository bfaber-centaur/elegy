package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

func robotoidTest(t *testing.T, v *View, rng engine.Rand) *robotoidTurn {
	t.Helper()
	rt := &robotoidTurn{turn: newTurn(v, rng, &Result{}), d1415: -1, d1113: -1, d910: -1, d25: -1, d67: -1}
	rt.params()
	return rt
}

// §1 step 3: potency and armada size, and the age limit.
func TestRobotoidParams(t *testing.T) {
	for _, c := range []struct{ y, p, a, l int }{{50, 4, 6, 50}, {120, 4, 6, 70}, {150, 5, 8, 70}, {300, 13, 12, 100}} {
		rt := robotoidTest(t, caView(t, FirstYear+c.y), top{})
		if rt.potency != c.p || rt.size != c.a || rt.limit != c.l {
			t.Errorf("y %d: P %d A %d T %d, want %d %d %d", c.y, rt.potency, rt.size, rt.limit, c.p, c.a, c.l)
		}
	}
}

// §1 step 4: a Nubian in slot 14 is never marked obsolete.
func TestRobotoidAgeingNubian(t *testing.T) {
	v := caView(t, 2460)
	home := v.Universe[0].Pos
	nub := v.Ships[0]
	nub.Slot, nub.Index, nub.Design.Hull.Name = 14, 44, "Nubian"
	v.Ships = append(v.Ships, nub)
	v.Fleets = []engine.Fleet{fleet(100, 1, home, 44, 1)}
	rt := robotoidTest(t, v, top{})
	rt.ageing()
	if rt.obsolete[14] || rt.d1415 != 14 {
		t.Errorf("obsolete %v, D1415 %d; want 14 unmarked and current", rt.obsolete, rt.d1415)
	}
}

// AI.md §11 "Fleet classes": a warship is an attack fleet; an unarmed
// Frigate stops the walk even before a warship in a later slot.
func TestAttackFleet(t *testing.T) {
	v := caView(t, 2460)
	cat := engine.Components()
	dd, err := cat.NewDesign("D", "Destroyer", []engine.SlotFill{fill(0, "Long Hump 6", 1)})
	if err != nil {
		t.Fatal(err)
	}
	fr, err := cat.NewDesign("F", "Frigate", []engine.SlotFill{fill(0, "Long Hump 6", 1)})
	if err != nil {
		t.Fatal(err)
	}
	v.Ships = append(v.Ships, Design{Slot: 4, Index: 34, Design: dd}, Design{Slot: 3, Index: 33, Design: fr})
	rt := robotoidTest(t, v, top{})
	home := v.Universe[0].Pos
	war := fleet(100, 1, home, 34, 1)
	both := fleet(101, 2, home, 34, 1)
	both.Stacks = append(both.Stacks, engine.Stack{Design: 33, Count: 1})
	if !rt.attackFleet(&war) {
		t.Error("a Destroyer fleet is not an attack fleet")
	}
	if rt.attackFleet(&both) {
		t.Error("an unarmed Frigate in an earlier slot did not stop the walk")
	}
}

// §3 step 1: freighters when fewer than 0.8 F are alive.
func TestRobotoidFreighters(t *testing.T) {
	v := caView(t, 2430)
	v.Planets[0].Population = 3000
	v.Ships = append(v.Ships, Design{Slot: 11, Index: 41, Design: v.Ships[2].Design, Created: 2415})
	rt := robotoidTest(t, v, top{})
	rt.d1113, rt.hubs = 11, []int{1} // F = 4
	rt.produce(&v.Planets[0], 0)
	q := rt.q.get(&v.Planets[0])
	if len(q) == 0 || q[0].Kind != engine.ItemShip || q[0].Slot != 11 || q[0].Count != 1 {
		t.Errorf("queue %+v, want one ship of slot 11 first", q)
	}
}

// AI.md §6 steps 3 and 4: transports go to the nearest hub with room,
// then a hub with fewer than 4 takes one from a hub with 2 more.
func TestAssignHubs(t *testing.T) {
	v := caView(t, 2430)
	pv, err := engine.Components().NewDesign("Raider", "Privateer", []engine.SlotFill{fill(0, "Long Hump 6", 1)})
	if err != nil {
		t.Fatal(err)
	}
	v.Ships = append(v.Ships, Design{Slot: 11, Index: 41, Design: pv})
	home := v.Universe[0].Pos
	for i := range 3 {
		v.Fleets = append(v.Fleets, fleet(100+i, 1+i, home, 41, 1))
	}
	rt := robotoidTest(t, v, top{})
	rt.hubs = []int{1, 5}
	got := rt.assignHubs()
	n := map[int]int{}
	for _, h := range got {
		n[h]++
	}
	if n[1] != 2 || n[5] != 1 || got[102] != 5 {
		t.Errorf("assignment %v, want two at hub 1 and fleet 102 moved to hub 5", got)
	}
}

package ai

import "github.com/bfaber-centaur/elegy/engine"

// robotoidLists are Robotoid's class lists (robotoid.md "Robotoid class
// lists"): one AI part class per hull slot, in hull-slot order. List 35 is
// never used.
var robotoidLists = [38][]int{
	0:  {8, 4, 10, 10, 13, 9, 9},
	1:  {8, 10, 5, 4, 13, 12, 15},
	2:  {8, 10, 4, 7, 13, 12, 14},
	3:  {8, 10, 3, 3, 13, 12, 14},
	4:  {8, 9, 1, 1, 11, 11, 12},
	5:  {8, 0, 9, 10, 13, 11, 12},
	6:  {8, 9, 0, 0, 10, 11, 12},
	7:  {8, 1, 9, 12, 12, 11, 11},
	8:  {8, 10, 16, 16, 3, 12, 2},
	9:  {8, 16, 4, 3, 14, 12, 13},
	10: {8, 3, 16, 10, 16, 12, 14},
	11: {8, 16, 1, 11, 12, 10, 10},
	12: {8, 16, 11, 12, 16, 1, 0},
	13: {8, 10, 16, 16, 11, 0, 0},
	14: {8, 10, 15, 4, 4},
	15: {8, 9, 11, 0, 0},
	16: {8, 4, 4, 4, 17, 18, 19},
	17: {8, 3, 3, 14, 17, 18, 19},
	18: {8, 4, 3, 2, 17, 18, 20},
	19: {8, 4, 4, 5, 17, 18, 20},
	20: {8, 0, 0, 10, 17, 18, 19},
	21: {8, 0, 0, 11, 17, 18, 19},
	22: {8, 1, 1, 11, 17, 18, 19},
	23: {8, 1, 1, 11, 17, 18, 11},
	24: {24, 21, 23, 23, 23, 12, 10},
	25: {24, 21, 22, 22, 22, 12, 10},
	26: {24, 26, 25, 10},
	27: {8, 11, 10, 1, 1, 1, 1, 2, 9, 11, 19},
	28: {8, 13, 10, 1, 1, 0, 0, 0, 9, 11, 19},
	29: {8, 13, 10, 0, 0, 1, 1, 0, 9, 11, 19},
	30: {8, 13, 10, 0, 0, 0, 0, 3, 9, 11, 19},
	31: {8, 13, 10, 4, 4, 4, 4, 4, 9, 20, 19},
	32: {8, 13, 10, 4, 3, 3, 7, 2, 9, 20, 19},
	33: {8, 13, 10, 2, 3, 7, 7, 3, 9, 20, 19},
	34: {8, 13, 10, 4, 4, 3, 3, 5, 9, 20, 19},
	36: {8, 13, 10, 33, 33, 33, 33, 33, 17, 20, 19},
	37: {8, 10, 10, 7, 5, 20, 20, 4, 4, 19, 4, 2, 3},
}

// techAtLeast reports whether every listed field is at its level: pairs
// of (field, level).
func techAtLeast(l [engine.NumFields]int, req ...int) bool {
	for i := 0; i+1 < len(req); i += 2 {
		if l[req[i]] < req[i+1] {
			return false
		}
	}
	return true
}

// random is robotoid.md's "random list" step: up to 5 tries, each drawing
// Random(k) and building list base + r, stopping at the first success. A
// failed try costs only its draw.
func (s *shipDesigns) random(slot int, hull string, base, k int) bool {
	for range 5 {
		if s.store(slot, hull, robotoidLists[base+s.rng.Intn(k)]) {
			return true
		}
	}
	return false
}

// robotoidDesigns is robotoid.md §2 (CONFIRMED AI-8): the steps run in
// order every year, each needing its target slot empty and its tech.
// lvl is the AI level; alive the ships alive per slot.
//
// Reproduced LEGACY BUGs (robotoid.md §2): step 2 writes slot 14 five
// times when the Nubian can be built; slots 12 and 13 test the previous
// slot's age without checking it holds a design. Step 7's yearly delete
// of an already empty slot 0 is not written: Elegy's engine would reject
// it, and it changes nothing.
func (s *shipDesigns) robotoidDesigns(lvl Level, alive map[int]int) {
	l := s.lvls
	empty := func(k int) bool { return !s.slots[k].present }
	// Step 1: freighters.
	for _, k := range []int{11, 12, 13} {
		if !empty(k) || !techAtLeast(l, engine.Propulsion, 2, engine.Construction, 3*k-29) {
			continue
		}
		if k > 11 && s.age(k-1) < 15 {
			continue
		}
		if l[engine.Construction] < 10 {
			list := 15
			if k == 11 {
				list = 14
			}
			s.store(k, "Privateer", robotoidLists[list])
		} else {
			s.random(k, "Meta Morph", 8, 6)
		}
	}
	// Step 2: slot 14.
	if empty(14) && techAtLeast(l, engine.Weapons, 5, engine.Electronics, 6, engine.Construction, 6, engine.Propulsion, 6, engine.Energy, 2) {
		for range 5 {
			if s.store(14, "Nubian", robotoidLists[37]) {
				continue // LEGACY BUG: the rounds go on
			}
			if s.store(14, "Destroyer", robotoidLists[16+s.rng.Intn(4)]) {
				break
			}
		}
	}
	// Step 3: slot 15.
	if empty(15) && techAtLeast(l, engine.Electronics, 10, engine.Construction, 8, engine.Propulsion, 9, engine.Weapons, 14) {
		if !s.store(15, "Nubian", robotoidLists[37]) {
			s.random(15, "Destroyer", 20, 4)
		}
	}
	// Step 4: warships 2–5.
	for _, k := range []int{2, 3, 4, 5} {
		if !empty(k) || !techAtLeast(l, engine.Weapons, 10, engine.Construction, 10, engine.Propulsion, 9, engine.Energy, 6) {
			continue
		}
		if k > 2 && (empty(k-1) || s.age(k-1) < 13) {
			continue
		}
		base := 0
		if k == 3 || k == 5 {
			base = 4
		}
		s.random(k, "Meta Morph", base, 4)
	}
	// Step 5: battleships 6 and 7.
	for _, k := range []int{6, 7} {
		if !empty(k) || !techAtLeast(l, engine.Biotech, 4, engine.Electronics, 10, engine.Construction, 12, engine.Propulsion, 12, engine.Energy, 6, engine.Weapons, 15) {
			continue
		}
		if k == 7 && (empty(6) || s.age(6) < 21) {
			continue
		}
		base := 27
		if k == 7 {
			base = 31
		}
		s.random(k, "Battleship", base, 4)
	}
	// Step 6: armada 9 and 10.
	for _, k := range []int{9, 10} {
		if !empty(k) || !techAtLeast(l, engine.Weapons, 14) {
			continue
		}
		if k == 10 && (empty(9) || s.age(9) < 16) {
			continue
		}
		if !s.store(k, "Battleship", robotoidLists[36]) {
			list := 24
			if k == 10 {
				list = 25
			}
			s.store(k, "B-52 Bomber", robotoidLists[list])
		}
	}
	// Step 7: the Frigate in slot 0.
	if lvl >= Harder && s.slots[0].hull != "Frigate" && alive[0] == 0 &&
		techAtLeast(l, engine.Biotech, 4, engine.Electronics, 5, engine.Construction, 6, engine.Propulsion, 6, engine.Energy, 6) {
		s.delete(0)
		s.store(0, "Frigate", robotoidLists[26])
	}
}

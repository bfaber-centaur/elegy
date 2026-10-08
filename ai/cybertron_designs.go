package ai

// cybertronLists are Cybertron's class lists (cybertron.md "Cybertron
// class lists"), with the hull each is built on.
var cybertronLists = [36]struct {
	hull    string
	classes []int
}{
	0:  {"Destroyer", []int{8, 4, 4, 18, 17, 18, 20}},
	1:  {"Destroyer", []int{8, 4, 4, 5, 17, 18, 20}},
	2:  {"Destroyer", []int{8, 4, 4, 4, 17, 18, 19}},
	3:  {"Destroyer", []int{8, 3, 3, 14, 17, 18, 19}},
	4:  {"Destroyer", []int{8, 4, 3, 2, 17, 18, 20}},
	5:  {"Destroyer", []int{8, 0, 0, 18, 17, 18, 19}},
	6:  {"Destroyer", []int{8, 0, 0, 10, 17, 18, 19}},
	7:  {"Destroyer", []int{8, 0, 0, 11, 17, 18, 19}},
	8:  {"Destroyer", []int{8, 1, 1, 11, 17, 18, 19}},
	9:  {"Destroyer", []int{8, 1, 1, 11, 17, 18, 11}},
	10: {"Privateer", []int{44, 10, 15, 4, 4}},
	11: {"Privateer", []int{44, 17, 11, 0, 0}},
	12: {"Frigate", []int{24, 26, 25, 10}},
	13: {"B-52 Bomber", []int{8, 21, 23, 23, 23, 12, 10}},
	14: {"B-52 Bomber", []int{8, 21, 22, 22, 22, 12, 10}},
	15: {"Battleship", []int{8, 14, 10, 33, 33, 33, 33, 33, 17, 20, 19}},
	16: {"Nubian", []int{8, 18, 20, 33, 33, 33, 33, 33, 33, 33, 33, 33, 33}},
	17: {"Cruiser", []int{8, 20, 19, 4, 4, 13, 17}},
	18: {"Cruiser", []int{8, 20, 19, 4, 3, 3, 17}},
	19: {"Cruiser", []int{8, 20, 19, 3, 2, 10, 17}},
	20: {"Cruiser", []int{8, 19, 11, 0, 0, 0, 17}},
	21: {"Cruiser", []int{8, 19, 11, 0, 0, 18, 17}},
	22: {"Cruiser", []int{8, 19, 11, 0, 0, 10, 17}},
	23: {"Cruiser", []int{8, 19, 11, 1, 1, 11, 17}},
	24: {"Cruiser", []int{8, 19, 11, 1, 1, 0, 17}},
	25: {"Cruiser", []int{8, 19, 11, 1, 1, 10, 17}},
	26: {"Battleship", []int{8, 18, 10, 2, 2, 3, 3, 2, 17, 20, 20}},
	27: {"Battleship", []int{8, 20, 10, 2, 2, 3, 3, 2, 17, 20, 20}},
	28: {"Battleship", []int{8, 18, 10, 0, 0, 3, 3, 2, 17, 20, 11}},
	29: {"Battleship", []int{8, 18, 10, 1, 1, 0, 0, 1, 17, 11, 11}},
	30: {"Battleship", []int{8, 11, 10, 1, 1, 0, 0, 1, 17, 11, 11}},
	31: {"Battleship", []int{8, 20, 10, 1, 1, 2, 2, 1, 17, 11, 11}},
	32: {"Battleship", []int{8, 11, 10, 1, 1, 1, 1, 1, 17, 11, 11}},
	33: {"Nubian", []int{8, 11, 11, 1, 1, 1, 20, 20, 2, 3, 3, 15, 19}},
	34: {"Nubian", []int{8, 11, 11, 1, 1, 1, 1, 1, 1, 19, 19, 15, 19}},
	35: {"Nubian", []int{8, 20, 20, 2, 2, 2, 3, 3, 3, 19, 19, 15, 19}},
}

func (s *shipDesigns) cyber(slot, list int) bool {
	l := cybertronLists[list]
	return s.store(slot, l.hull, l.classes)
}

// cyberRange is cybertron.md §2's range a..b: the lists are tried in
// random order without repeats, each try drawing Random(m) over the m
// lists left, stopping at the first success.
//
// ASSUMPTION A16: Random(m) indexes the lists left in increasing order,
// and the one tried is removed. cybertron.md checked ranges as sets of
// allowed outcomes (AI-19).
func (s *shipDesigns) cyberRange(slot, a, b int) bool {
	var left []int
	for l := a; l <= b; l++ {
		left = append(left, l)
	}
	for len(left) > 0 {
		i := s.rng.Intn(len(left))
		l := left[i]
		left = append(left[:i:i], left[i+1:]...)
		if s.cyber(slot, l) {
			return true
		}
	}
	return false
}

// cybertronDesigns is cybertron.md §2 (CONFIRMED AI-19): the steps run in
// order every year. Every rule reading a slot's age first checks that the
// slot holds a design. y is the year index.
func (s *shipDesigns) cybertronDesigns(lvl Level, y int, alive map[int]int) {
	present := func(k int) bool { return s.slots[k].present }
	// Step 1: slot 0's Frigate.
	if present(0) && s.slots[0].hull != "Frigate" && lvl >= Harder && alive[0] == 0 && y > 5 {
		s.delete(0)
	}
	if !present(0) {
		s.cyber(0, 12)
	}
	// Steps 2 and 3: Destroyers 4 and 5.
	if !present(4) && y > 30 {
		if y > 75 || !s.cyber(4, 0) {
			s.cyberRange(4, 0, 4)
		}
	}
	if !present(5) && present(4) && s.age(4) > 20 {
		if y > 75 || !s.cyber(5, 5) {
			s.cyberRange(5, 5, 9)
		}
	}
	// Steps 4 and 5: Privateers 2 and 3.
	if !present(2) && y > 20 {
		s.cyber(2, 10)
	}
	if !present(3) && present(2) && s.age(2) > 20 {
		s.cyber(3, 11)
	}
	// Step 6: warship groups 6–9 and 10–13.
	for _, g := range []int{6, 10} {
		if present(g) {
			continue
		}
		if !(g == 6 && y > 40) && !(present(6) && s.age(6) > 30) {
			continue
		}
		s.warshipGroup(g)
	}
	// Steps 7 and 8: guards 14 and 15.
	if !present(14) && y > 30 {
		s.guard(14)
	}
	if !present(15) && present(14) && s.age(14) > 20 {
		s.guard(15)
	}
}

// warshipGroup is step 6 for the group starting at slot g.
//
// ASSUMPTION A17: when the big ships leave the target at g, slot g gets
// the "first" Cruiser range (the last 9 lists); "slot g itself range
// 17..19" applies when the fill reaches g from above.
func (s *shipDesigns) warshipGroup(g int) {
	t := g + 2
	for _, r := range [][2]int{{33, 35}, {29, 32}, {26, 28}} {
		if s.cyberRange(t, r[0], r[1]) {
			t--
		}
	}
	for k := t; k >= g; k-- {
		switch {
		case k == t:
			n := 3 * (g - t + 3)
			s.cyberRange(k, 25-n+1, 25)
		case k == g:
			s.cyberRange(k, 17, 19)
		default:
			s.cyberRange(k, 20, 22)
		}
	}
	for _, l := range []int{16, 15, 14, 13} {
		if s.cyber(g+3, l) {
			break
		}
	}
}

// guard is steps 7 and 8: Battleship range 26..32, else Cruiser range
// 17..25, else Destroyer range 0..9.
func (s *shipDesigns) guard(k int) {
	if !s.cyberRange(k, 26, 32) && !s.cyberRange(k, 17, 25) {
		s.cyberRange(k, 0, 9)
	}
}

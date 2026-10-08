package ai

import (
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// Starbase design slots and families (AI.md §5, CONFIRMED AI-2). A
// family's slots are its variants 1, 2, 3 in order; the Orbital Fort
// families have variants 1 and 3.
var (
	stationA = []int{0, 2, 4}
	stationB = []int{5, 7, 9}
	fortA    = []int{1, 3}
	fortB    = []int{6, 8}
)

const (
	spaceStation = "Space Station"
	orbitalFort  = "Orbital Fort"
)

// Hull slots' AI part classes, in hull order (AI.md §5).
var starbaseClasses = map[string][]int{
	spaceStation: {34, 35, 37, 0, 17, 10, 11, 38, 19, 36, 34, 9},
	orbitalFort:  {34, 0, 17, 36, 37},
}

// starbaseNames are Elegy's own 13 starbase names (AI.md §5 "Name":
// "Elegy may use its own 13 names").
var starbaseNames = [13]string{
	"Bastion", "Citadel", "Rampart", "Bulwark", "Keep", "Redoubt", "Watchtower",
	"Anchorage", "Haven", "Outpost", "Sentinel", "Harbor", "Aegis",
}

// SlotDesign is one of the player's own designs as a planner sees it.
type SlotDesign struct {
	Hull string
	Name string
	// Created is the calendar year the design was stored.
	Created int
	// Picture is the design's picture, 0..3 (AI.md §5 "Picture").
	Picture int
}

// NewDesign is a design a planner stored this turn, with the picture and
// creation year the engine's design order does not carry.
type NewDesign struct {
	Starbase bool
	Slot     int
	Name     string
	Picture  int
	Created  int
}

// StarbaseInput is what the starbase-design step reads.
type StarbaseInput struct {
	Personality Personality
	Year        int // calendar year
	Race        engine.Race
	Levels      [engine.NumFields]int
	// Designs are the player's starbase design slots; nil is empty.
	Designs [10]*SlotDesign
	// Built marks the slots an own starbase is built from.
	Built [10]bool
}

// StarbaseDesigns is the yearly starbase-design upkeep of Robotoid,
// Rototill and Cybertron (AI.md §5 "Each year", CONFIRMED AI-2), run
// before the personality's own work:
//
//  1. If slot 2 is empty, create slot 2 (variant 2) and slot 4 (variant
//     3). If slot 1 is empty, create slot 1 (variant 1). If slot 3 is
//     empty, create slot 3 (variant 3).
//  2. From year index 50: when the newest Space Station design of slots
//     0, 2, 4, 5, 7, 9 (ties: the later slot) is 40 or more years old and
//     no own starbase is built from the other family's slots, create the
//     other family as variants 1, 2, 3; the same for the Orbital Forts of
//     slots 1, 3, 6, 8 with variants 1 and 3.
//
// It returns the design orders and the designs stored. A design created
// earlier in the turn counts for later pictures and names.
//
// ASSUMPTION A2: step 1 creates slot 4 with slot 2 whether or not slot 4
// holds a design, as AI.md §5 says; a computer player's slot 4 is empty
// whenever slot 2 is.
func StarbaseDesigns(in StarbaseInput, rng engine.Rand) ([]engine.Order, []NewDesign) {
	s := &starbaseRun{in: in, rng: rng, designs: in.Designs}
	y := in.Year - FirstYear
	if s.designs[2] == nil {
		s.create(2, 2)
		s.create(4, 3)
	}
	if s.designs[1] == nil {
		s.create(1, 1)
	}
	if s.designs[3] == nil {
		s.create(3, 3)
	}
	if y >= 50 {
		s.switchFamily(stationA, stationB)
		s.switchFamily(fortA, fortB)
	}
	return s.orders, s.made
}

type starbaseRun struct {
	in      StarbaseInput
	rng     engine.Rand
	designs [10]*SlotDesign
	orders  []engine.Order
	made    []NewDesign
}

// switchFamily is step 2 for one hull: a and b are the hull's two families.
func (s *starbaseRun) switchFamily(a, b []int) {
	newest, created := -1, 0
	for _, k := range mergeSlots(a, b) {
		if d := s.designs[k]; d != nil && (newest < 0 || d.Created >= created) {
			newest, created = k, d.Created
		}
	}
	if newest < 0 || s.in.Year-created < 40 {
		return
	}
	other := b
	for _, k := range b {
		if k == newest {
			other = a
		}
	}
	for _, k := range other {
		if s.in.Built[k] {
			return
		}
	}
	for i, k := range other {
		v := i + 1
		if len(other) == 2 && i == 1 {
			v = 3
		}
		s.create(k, v)
	}
}

// mergeSlots is the slots of both families in increasing order, so the
// later slot wins a tie.
func mergeSlots(a, b []int) []int {
	out := append(append([]int(nil), a...), b...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// slotHull is the hull of a starbase slot's family (AI.md §5 "Creating a
// design").
func slotHull(slot int) string {
	switch slot {
	case 1, 3, 6, 8:
		return orbitalFort
	}
	return spaceStation
}

// create builds variant v of the slot's hull (AI.md §5 "Creating a
// design"). Nothing is created, and nothing drawn, when the hull is not
// available to the race or a hull slot's class has no part the race can
// build now. Counts start at the hull slot's maximum; for variants 1 and
// 2 a slot whose maximum is 4 or more gets max >> (3 − v); for variant 1
// only, except Cybertron, a slot with fewer than 4 that holds an orbital
// part or more than one item loses one.
//
// ASSUMPTION A3: "holds an orbital part or more than one item" is read
// as the chosen part being an orbital part (a mass driver, which only
// Packet Physics races can build, so the clause never applies to the
// three implemented personalities) or the count being above 1. A slot
// reduced to 0 is left empty.
func (s *starbaseRun) create(slot, v int) {
	cat := engine.Components()
	hullName := slotHull(slot)
	hc, ok := cat.Lookup(hullName)
	if !ok {
		return
	}
	if ok, err := hc.Buildable(s.in.Race, s.in.Levels, false); !ok || err != nil {
		return
	}
	hull, err := hc.Hull()
	if err != nil {
		return
	}
	var fills []engine.SlotFill
	for i, hs := range hull.Slots {
		comp, ok := classPart(cat, starbaseClasses[hullName][i], s.in.Race, s.in.Levels)
		if !ok {
			return
		}
		n := hs.Max
		if v < 3 && n >= 4 {
			n = n >> (3 - v)
		}
		if v == 1 && s.in.Personality != Cybertron && hs.Max < 4 && (comp.Category == engine.CatOrbital || n > 1) {
			n--
		}
		if n > 0 {
			fills = append(fills, engine.SlotFill{Slot: i, Part: comp.Name, Count: n})
		}
	}
	pic := s.picture(slot, hullName)
	name := s.name(slot, hullName)
	s.designs[slot] = &SlotDesign{Hull: hullName, Name: name, Created: s.in.Year, Picture: pic}
	s.orders = append(s.orders, engine.DesignOrder{Starbase: true, Slot: slot, Name: name, Hull: hullName, Fills: fills, Picture: pic})
	s.made = append(s.made, NewDesign{Starbase: true, Slot: slot, Name: name, Picture: pic, Created: s.in.Year})
}

// picture is the first of the hull's four pictures not used by another
// starbase design of the same hull, else Random(4) (AI.md §5 "Picture").
//
// ASSUMPTION A4: a design the new one replaces still counts as using its
// picture and name, as for ship designs (AI.md §10 "Picture").
func (s *starbaseRun) picture(slot int, hull string) int {
	var used [4]bool
	for _, d := range s.designs {
		if d != nil && d.Hull == hull && d.Picture >= 0 && d.Picture < 4 {
			used[d.Picture] = true
		}
	}
	for i, u := range used {
		if !u {
			return i
		}
	}
	return s.rng.Intn(4)
}

// name tries Random(13) up to 20 times for a name unused among starbase
// designs of the same hull; after 20 failures the last name tried gets a
// number from Random(100) (AI.md §5 "Name"). ASSUMPTION A5: the numbered
// form is "<name> <n>"; AI.md calls the exact form BINARY-ONLY.
func (s *starbaseRun) name(slot int, hull string) string {
	var name string
	for range 20 {
		name = starbaseNames[s.rng.Intn(len(starbaseNames))]
		taken := false
		for _, d := range s.designs {
			if d != nil && d.Hull == hull && d.Name == name {
				taken = true
			}
		}
		if !taken {
			return name
		}
	}
	return fmt.Sprintf("%s %d", name, s.rng.Intn(100))
}

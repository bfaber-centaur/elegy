package ai

import (
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// shipSlot is one of the player's 16 ship design slots during a turn. A
// slot deleted earlier in the turn keeps its old hull and creation year
// (robotoid.md "Notation"); an empty slot read at the start of a turn has
// creation year index 0 under clean per-player state (AI.md §1).
type shipSlot struct {
	present bool
	hull    string
	name    string
	created int // calendar year; FirstYear (year index 0) for never
	picture int
	fills   []engine.SlotFill // set for a design stored this turn
}

// shipDesigns is the planner's picture of its ship design slots and the
// design orders it has given.
type shipDesigns struct {
	slots [16]shipSlot
	year  int
	race  engine.Race
	lvls  [engine.NumFields]int
	// trader is the player's Mystery Trader parts (View.TraderItems).
	trader map[string]bool
	rng    engine.Rand
	res    *Result
	// q is the turn's queues, which a delete changes; nil leaves them.
	q *queues
}

func newShipDesigns(v *View, rng engine.Rand, res *Result) *shipDesigns {
	s := &shipDesigns{year: v.Year, race: v.Self.Race, lvls: v.Self.Research.Levels, trader: v.TraderItems, rng: rng, res: res}
	for k := range s.slots {
		s.slots[k].created = FirstYear // year index 0
	}
	for _, d := range v.Ships {
		if d.Slot >= 0 && d.Slot < len(s.slots) {
			s.slots[d.Slot] = shipSlot{present: true, hull: d.Design.Hull.Name, name: d.Design.Name, created: d.Created, picture: d.Picture}
		}
	}
	return s
}

// age is the year index minus the slot's creation year index.
func (s *shipDesigns) age(k int) int { return s.year - s.slots[k].created }

// delete writes a delete order for a present slot, and drops the slot's
// queue entries as the order will (queues.dropShip).
func (s *shipDesigns) delete(k int) {
	if !s.slots[k].present {
		return
	}
	s.slots[k].present = false
	s.res.Orders = append(s.res.Orders, engine.DeleteDesignOrder{Slot: k})
	if s.q != nil {
		s.q.dropShip(k)
	}
}

// build is the ship-design builder (AI.md §10 "Builder"): the hull must be
// available to the race, and each hull slot takes, at its maximum count,
// the first part of its class the race can build now. Any slot without a
// part means no design and no random draws. classes names one AI part
// class per hull slot.
func (s *shipDesigns) build(hull string, classes []int) ([]engine.SlotFill, bool) {
	cat := engine.Components()
	hc, ok := cat.Lookup(hull)
	if !ok {
		return nil, false
	}
	if ok, err := hc.Buildable(s.race, s.lvls, s.trader[hull]); !ok || err != nil {
		return nil, false
	}
	h, err := hc.Hull()
	if err != nil || len(h.Slots) != len(classes) {
		return nil, false
	}
	fills := make([]engine.SlotFill, len(h.Slots))
	for i, hs := range h.Slots {
		comp, ok := classPart(cat, classes[i], s.race, s.lvls, s.trader)
		if !ok {
			return nil, false
		}
		fills[i] = engine.SlotFill{Slot: i, Part: comp.Name, Count: hs.Max}
	}
	return fills, true
}

// store builds and stores a design into slot k (AI.md §10 "Storing a
// design"): when the slot holds a design, a delete order is written
// first; if the build then fails nothing more is written and the slot
// stays empty. The replaced design still counts for the picture and name.
// It reports success.
func (s *shipDesigns) store(k int, hull string, classes []int) bool {
	old := s.slots[k]
	s.delete(k)
	fills, ok := s.build(hull, classes)
	if !ok {
		return false
	}
	s.slots[k] = old // counts for the picture and name
	pic := s.picture(hull)
	name := s.name(hull)
	s.slots[k] = shipSlot{present: true, hull: hull, name: name, created: s.year, picture: pic, fills: fills}
	s.res.Orders = append(s.res.Orders, engine.DesignOrder{Slot: k, Name: name, Hull: hull, Fills: fills, Picture: pic})
	s.res.Designs = append(s.res.Designs, NewDesign{Slot: k, Name: name, Picture: pic, Created: s.year})
	return true
}

// picture is the first of the hull's four pictures not used by another
// non-empty ship design of the same hull (the design being replaced
// still counts), else Random(4) (AI.md §10 "Picture").
func (s *shipDesigns) picture(hull string) int {
	var used [4]bool
	for _, d := range s.slots {
		if d.present && d.hull == hull && d.picture >= 0 && d.picture < 4 {
			used[d.picture] = true
		}
	}
	for i, u := range used {
		if !u {
			return i
		}
	}
	return s.rng.Intn(4)
}

// name tries Random(n) from the hull role's name group up to 20 times for
// a name unlike every non-empty ship design's (any hull; the design being
// replaced counts); after 20 failures Random(100) then Random(n), the
// name with the number appended, unchecked (AI.md §10 "Name"). The names
// are Elegy's own; ASSUMPTION A5's "<name> <n>" form applies.
func (s *shipDesigns) name(hull string) string {
	group := nameGroup(hull)
	taken := func(n string) bool {
		for _, d := range s.slots {
			if d.present && d.name == n {
				return true
			}
		}
		return false
	}
	for range 20 {
		if n := group[s.rng.Intn(len(group))]; !taken(n) {
			return n
		}
	}
	num := s.rng.Intn(100)
	return fmt.Sprintf("%s %d", group[s.rng.Intn(len(group))], num)
}

// Name groups by hull role, with the original's group sizes (AI.md §10).
var (
	warshipNames   = []string{"Vindicator", "Paladin", "Juggernaut", "Leviathan", "Colossus", "Dominion", "Tempest", "Monarch", "Sovereign", "Titan", "Warden", "Avenger", "Vanguard", "Retribution", "Conqueror", "Imperator"}
	destroyerNames = []string{"Lancer", "Rapier", "Saber", "Cutlass", "Falchion", "Scimitar", "Gladius", "Halberd", "Pike", "Glaive", "Javelin", "Spear", "Trident", "Partisan", "Dirk", "Stiletto"}
	scoutNames     = []string{"Seeker", "Pathfinder", "Wayfarer", "Lookout", "Ranger", "Outrider", "Picket", "Skirmisher", "Harrier", "Swift"}
	bomberNames    = []string{"Hammer", "Anvil", "Thunder", "Quake", "Ruin", "Havoc", "Cinder", "Ember", "Blight", "Scourge", "Ravager", "Rubble"}
	freighterNames = []string{"Hauler", "Drayman", "Carrier", "Porter", "Packhorse", "Barge", "Tender", "Lighter"}
	minerNames     = []string{"Delver", "Digger", "Prospector", "Sapper", "Driller", "Excavator", "Quarry", "Collier"}
	privateerNames = []string{"Corsair", "Buccaneer", "Rover", "Marauder", "Freebooter", "Raider", "Brigand", "Picaroon"}
	colonyNames    = []string{"Pioneer", "Settler", "Homestead", "Frontier", "Founder", "Ark", "Hope", "Promise"}
	otherNames     = []string{"Utility", "Auxiliary", "Courier", "Cutter", "Skiff", "Pinnace", "Shuttle", "Launch"}
)

func nameGroup(hull string) []string {
	switch hull {
	case "Cruiser", "Battle Cruiser", "Battleship", "Dreadnought":
		return warshipNames
	case "Destroyer":
		return destroyerNames
	case "Scout", "Frigate":
		return scoutNames
	case "Mini Bomber", "B-17 Bomber", "Stealth Bomber", "B-52 Bomber":
		return bomberNames
	case "Small Freighter", "Medium Freighter", "Large Freighter", "Super Freighter":
		return freighterNames
	case "Midget Miner", "Mini-Miner", "Miner", "Maxi-Miner", "Ultra-Miner":
		return minerNames
	case "Privateer", "Rogue", "Galleon":
		return privateerNames
	case "Mini-Colony Ship", "Colony Ship":
		return colonyNames
	}
	return otherNames
}

// ageGroup is AI.md §10 "Ageing" for one group of slots with age limit l:
// the group's newest design (latest creation year, ties to the lower
// slot) is chosen before any deletion; each design older than l years
// with no ship alive is deleted, and one with ships alive is marked
// obsolete. It returns the newest slot (−1 when the group is empty) and
// the ships alive in the group (at most 32,000).
func (s *shipDesigns) ageGroup(slots []int, l int, alive map[int]int, obsolete map[int]bool) (newest, ships int) {
	newest = -1
	for _, k := range slots {
		if d := s.slots[k]; d.present && (newest < 0 || d.created > s.slots[newest].created) {
			newest = k
		}
	}
	for _, k := range slots {
		if !s.slots[k].present {
			continue
		}
		ships += alive[k]
		if s.age(k) <= l {
			continue
		}
		if alive[k] == 0 {
			s.delete(k)
		} else {
			obsolete[k] = true
		}
	}
	return newest, min(ships, 32000)
}

// syncView brings the planner's View up to date with this turn's design
// orders, so later steps (production, costs) see the slots as the
// engine will once the orders apply: deleted slots are gone and stored
// designs are present, with no design index yet (no ship has one).
func (s *shipDesigns) syncView(v *View) {
	var ships []Design
	for _, d := range v.Ships {
		if d.Slot >= 0 && d.Slot < len(s.slots) && s.slots[d.Slot].present && s.slots[d.Slot].fills == nil {
			ships = append(ships, d)
		}
	}
	for k, sl := range s.slots {
		if !sl.present || sl.fills == nil {
			continue
		}
		d, err := engine.Components().ReadDesign(sl.name, sl.hull, sl.fills, s.race, s.lvls, s.trader, v.Rules)
		if err != nil {
			continue
		}
		ships = append(ships, Design{Slot: k, Index: -1, Design: d, Created: sl.created, Picture: sl.picture})
	}
	v.Ships = ships
}

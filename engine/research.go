package engine

import (
	"sort"
	"sync"
)

// Research fields, in field order.
const (
	Energy = iota
	Weapons
	Propulsion
	Construction
	Electronics
	Biotech
	NumFields
)

// MaxTechLevel is the highest level of a field.
const MaxTechLevel = 26

// Next-field choices other than a specific field index.
const (
	NextSameField   = -1
	NextLowestField = -2
)

// researchBase is base[L] for L = 1..26.
var researchBase = [MaxTechLevel + 1]int{0,
	50, 80, 130, 210, 340, 550, 890, 1440, 2330, 3770,
	6100, 9870, 13850, 18040, 22440, 27050, 31870, 36900, 42140, 47590,
	53250, 59120, 65200, 71490, 77990, 84700}

// ResearchState is a player's technology.
type ResearchState struct {
	Levels      [NumFields]int
	Accumulated [NumFields]int
	Current     int
	// Next is the field to switch to after a level is gained in Current:
	// NextSameField, NextLowestField, or a field index.
	Next int
	// MaxLevel caps every field; 0 means MaxTechLevel. (KERNEL.md mentions
	// a level-10 cap for "a capped player".)
	MaxLevel int
}

func (s ResearchState) maxLevel() int {
	if s.MaxLevel == 0 {
		return MaxTechLevel
	}
	return s.MaxLevel
}

func (s ResearchState) levelSum() int {
	sum := 0
	for _, l := range s.Levels {
		sum += l
	}
	return sum
}

// ResearchLevelCost is the research needed to reach level from level−1,
// given the sum of the player's six current levels.
//
// KERNEL.md "Level cost": CONFIRMED for the normal setting (levels 3–9 of
// one field, PG) and the other two settings (KX-002 R1, R2); slower tech's
// doubling with the half-scale storage of AddResearch is CONFIRMED (KX-003
// S2).
func ResearchLevelCost(level, levelSum int, setting ResearchCost, slowerTech bool) int {
	c := researchBase[level] + 10*levelSum
	switch setting {
	case ResearchExpensive:
		c = 2*c - c/4
	case ResearchCheap:
		c = c / 2
	}
	if slowerTech {
		c *= 2
	}
	return c
}

// AddResearch adds one year's research resources to a player's state and
// applies level-ups.
//
// KERNEL.md "Allocation": accumulation and single level-ups with "same
// field" are CONFIRMED (PG); several levels per year, field switching,
// Generalized Research's split and research lost at level 26 are CONFIRMED
// (KX-002 G1, R2–R6); only the current field switching under Generalized
// Research, and the level-10 cap, are BINARY-ONLY. With Generalized
// Research every field receives its share when the current field's turn
// comes in the first of the field-order passes (BINARY-ONLY; see below).
//
// Under slower tech the stored accumulation is kept at half scale
// (KERNEL.md "Slower tech", CONFIRMED KX-003 S2, KX-005 R2): the year works
// on twice the stored value plus the research, against the doubled cost,
// and stores half of what is left, rounded up. A Generalized Research
// share for another field adds half of it, rounded down.
func AddResearch(s ResearchState, race Race, resources int, slowerTech bool) ResearchState {
	s, _ = addResearch(s, race, resources, slowerTech)
	return s
}

// addResearch is AddResearch, also returning the field of each level
// gained, in the order gained.
func addResearch(s ResearchState, race Race, resources int, slowerTech bool) (ResearchState, []int) {
	var gained []int
	scale := 1
	if slowerTech {
		scale = 2
		for f := range NumFields {
			s.Accumulated[f] *= 2
		}
	}
	add := func(field, amount int) {
		if s.Levels[field] >= s.maxLevel() {
			return // research into a maxed field is lost
		}
		s.Accumulated[field] += amount
	}
	var shares [NumFields]int
	if race.LRT.GeneralizedResearch {
		for f := range NumFields {
			if f == s.Current {
				shares[f] = (resources + 1) / 2
			} else {
				share := (3*resources + 19) / 20
				shares[f] = share / scale * scale
			}
		}
	} else {
		shares[s.Current] = resources
	}

	// levelOnce raises field by one level when its accumulation covers
	// the cost.
	levelOnce := func(field int) bool {
		if s.Levels[field] >= s.maxLevel() {
			return false
		}
		cost := ResearchLevelCost(s.Levels[field]+1, s.levelSum(), race.ResearchCosts[field], slowerTech)
		if s.Accumulated[field] < cost {
			return false
		}
		s.Accumulated[field] -= cost
		s.Levels[field]++
		gained = append(gained, field)
		return true
	}
	levelUp := func(field int) bool {
		gained := false
		for levelOnce(field) {
			gained = true
		}
		return gained
	}

	// While the year's choice switches fields ("lowest field", an
	// explicit field, or the level-26 switch below), the current field
	// takes one level at a time and the choice is made again after each.
	// CONFIRMED only for the level-26 switch (KB-2A P0: weapons 0 → 1 then
	// propulsion 0 → 1, not weapons 2). For a stored "lowest field" and an
	// explicit field it is inferred from KERNEL.md "Allocation": "when a
	// level is gained ... the leftover moves to the new field".
	lowest := false
	step := func(field int) bool {
		if !lowest && s.Next == NextSameField {
			return levelUp(field)
		}
		return levelOnce(field)
	}

	// Only a level-up in the current field switches fields. The leftover
	// moves to the new field, which is checked for level-ups the same
	// year, and may switch again. An explicit next field is used once and
	// then resets to "same field"; "lowest field" stays set.
	//
	// A current field that reaches level 26 under "same field" switches
	// as if "lowest field" were chosen, for the rest of the year, while
	// the stored choice stays "same field" (KERNEL.md "Allocation",
	// CONFIRMED KX-005 R3, KB-2A). ASSUMPTION: a field reaching the
	// level-10 cap of a capped player does not switch; KERNEL.md gives
	// only level 26, and the level-10 cap itself is BINARY-ONLY.
	switchField := func() {
		if s.Next == NextSameField && !lowest {
			if s.Levels[s.Current] < MaxTechLevel {
				return
			}
			lowest = true
		}
		next := s.Next
		if lowest || next == NextLowestField {
			next = 0
			for f := range NumFields {
				if s.Levels[f] < s.Levels[next] {
					next = f
				}
			}
		} else {
			s.Next = NextSameField
		}
		if next != s.Current {
			s.Accumulated[next] += s.Accumulated[s.Current]
			s.Accumulated[s.Current] = 0
			s.Current = next
		}
	}

	// The fields are checked in passes, in field order. In the first
	// pass the current field receives this year's research when its turn
	// comes, and under Generalized Research every other field receives
	// its share at the same moment, so the fields before the current one
	// see their shares only in the second pass, which always follows
	// (BINARY-ONLY, stars-elegy 397ebfc, for players with and without
	// Generalized Research; with nothing added the first affordable field
	// in field order levels, with shares arriving the field after the
	// current one comes first, wrapping round: lane F's GR-ORDER cases and
	// RG-001 2433, where electronics, then biotech, then energy
	// levelled). KERNEL.md repeats a pass after that only when a level in
	// the current field applies the next-field choice; Elegy repeats
	// after any level-up (ELEGY CHOICE: the outcome is the same, since a
	// level only raises costs, so the extra pass levels nothing).
	added := false
	for changed := true; changed; {
		changed = false
		for f := range NumFields {
			if f != s.Current {
				changed = levelUp(f) || changed
				continue
			}
			if !added {
				for g := range NumFields {
					add(g, shares[g])
				}
				added = true
				changed = true // the second pass
			}
			if step(f) {
				changed = true
				switchField()
			}
		}
	}
	if slowerTech {
		for f := range NumFields {
			s.Accumulated[f] = (s.Accumulated[f] + 1) / 2
		}
	}
	return s, gained
}

// LevelUpCheck is the second research level-up check of the turn, after
// battles (KERNEL.md "Turn order" step 6): every field whose accumulated
// research covers its next level rises, in field order. It does not
// switch the current field.
func LevelUpCheck(s ResearchState, race Race, slowerTech bool) ResearchState {
	for f := range NumFields {
		for s.Levels[f] < s.maxLevel() {
			// The stored value is at half scale under slower tech.
			cost := ResearchLevelCost(s.Levels[f]+1, s.levelSum(), race.ResearchCosts[f], false)
			if s.Accumulated[f] < cost {
				break
			}
			s.Accumulated[f] -= cost
			s.Levels[f]++
		}
	}
	return s
}

// researchSpent is how much of a year's research resources go into each
// field before any level-up: all of it to the current field, or under
// Generalized Research half (rounded up) to the current field and 15%
// (rounded up) to each other field (KERNEL.md "Allocation"). A field at
// its maximum level still counts what is directed into it, though
// AddResearch loses it (ASSUMPTION: KERNEL.md "Super Stealth" counts
// "every player's research in that field" and does not say whether lost
// research counts).
func researchSpent(s ResearchState, race Race, resources int) [NumFields]int {
	var spent [NumFields]int
	if !race.LRT.GeneralizedResearch {
		spent[s.Current] = resources
		return spent
	}
	for f := range NumFields {
		if f == s.Current {
			spent[f] = (resources + 1) / 2
		} else {
			spent[f] = (3*resources + 19) / 20
		}
	}
	return spent
}

// EventResearchStolen: a Super Stealth Player gained Count research in
// field Item (a field index, not an item kind) from what every player
// spent there this year.
const EventResearchStolen EventKind = EventWarp10FleetLost + 1

// stealResearch is KERNEL.md "Allocation", Super Stealth (stars-elegy
// fda4173, CONFIRMED KX-003 S3L): after every player's research, each
// Super Stealth player gains, per field, s = trunc(trunc(spent/players)/2)
// when s > 1, where spent is every player's research in that field this
// year, its own included, and players the players still in the game;
// one message per field. Under slower tech ceil(s/2) is stored
// (CONFIRMED, KB-2B). Generalized Research shares count in full. After
// every player's normal level-ups, the fields the stolen research levels
// up are checked in field order, energy first (BINARY-ONLY, KERNEL.md in
// stars-elegy 2f214f5).
func (g *Game) stealResearch(spent [][NumFields]int) []Event {
	players := 0
	var total [NumFields]int
	for i, pl := range g.Players {
		if pl.Dead {
			continue
		}
		players++
		for f := range NumFields {
			total[f] += spent[i][f]
		}
	}
	// No research is stolen unless more than one player is still in the
	// game (BINARY-ONLY, stars-elegy 2f214f5; MEASURED: the one-player
	// Super Stealth PG, KX-001 and KX-002 games kept their own research
	// unchanged, where KERNEL.md's formula would give them half of it
	// again).
	if players < 2 {
		return nil
	}
	var events []Event
	for i := range g.Players {
		pl := &g.Players[i]
		if pl.Dead || pl.Race.PRT != PRTSuperStealth {
			continue
		}
		for f := range NumFields {
			s := total[f] / players / 2
			if s <= 1 {
				continue
			}
			stored := s
			if g.SlowerTech {
				stored = (s + 1) / 2
			}
			pl.Research.Accumulated[f] += stored
			events = append(events, Event{Kind: EventResearchStolen, Player: i, Planet: -1, Fleet: -1, Item: ItemKind(f), Count: s})
		}
		before := pl.Research.Levels
		pl.Research = LevelUpCheck(pl.Research, pl.Race, g.SlowerTech)
		events = append(events, g.levelEvents(i, before, fieldOrder(before, pl.Research.Levels))...)
	}
	return events
}

// Research level-up messages.
const (
	// EventLevelUp: Player gained a level in field Item (a field index,
	// not an item kind); Count is the new level (MESSAGES.md 0x050, or
	// 0x136 for a Generalized Research player).
	EventLevelUp EventKind = EventBattleSummary + 1 + iota
	// EventPartAvailable: that level made a component available to
	// Player; Item is the field, Count the component's position in
	// Components().Components (MESSAGES.md 0x05f, or 0x078, 0x0d0, 0x145
	// or 0x157 by its kind).
	EventPartAvailable
)

// levelEvents is the level-up messages for the levels player gained since
// its levels were before, one field index per level in gained, in the
// order the levels were gained (MESSAGES.md 0x050 and 0x05f, CONFIRMED
// KX-003, KX-004, KX-005): for each level, its message, then one per
// component the race may use (owned, for a Mystery Trader part) whose
// requirement in that field is the new level and whose other requirements
// the player then meets (MEASURED KX-005 R1; BINARY-ONLY beyond it).
//
// The order follows the research chain, not field order: the current
// field's levels first, then the field the leftover moved to, each level
// followed by its own parts (MESSAGES.md "Clarified 2026-10-13 (EM-1)" at
// stars-elegy d89b8a3, MEASURED in RG-002 2468: biotech 7, its parts, then
// weapons 8, its parts). A part is announced by whichever requirement is
// met last (same clarification, MEASURED across years); ASSUMPTION: within
// one year "then" is the levels reached so far in that order, so a part
// needing two fields that both rose is announced at the later one.
//
// Terraforming components are announced like other parts, by any field
// (MESSAGES.md "Clarified 2026-10-13 (EM-1)" at stars-elegy d89b8a3,
// MEASURED: Gravity ±7 on propulsion 5, Radiation ±7 on weapons 5).
func (g *Game) levelEvents(player int, before [NumFields]int, gained []int) []Event {
	pl := &g.Players[player]
	cat := Components()
	trader := g.traderItems(player)
	levels := before
	var events []Event
	for _, f := range gained {
		levels[f]++
		events = append(events, Event{Kind: EventLevelUp, Player: player, Planet: -1, Fleet: -1, Item: ItemKind(f), Count: levels[f]})
		for _, i := range announceOrder() {
			c := cat.Components[i]
			if c.TechReq[f] != levels[f] || unannounced(pl.Race, c) {
				continue
			}
			if ok, err := c.Buildable(pl.Race, levels, trader[c.Name]); err != nil || !ok {
				continue
			}
			events = append(events, Event{Kind: EventPartAvailable, Player: player, Planet: -1, Fleet: -1, Item: ItemKind(f), Count: i})
		}
	}
	return events
}

// announceCategories is the order in which a level's components are
// announced after its level message (MESSAGES.md "Order of the component
// messages within one level", stars-elegy f2778b3): BINARY-ONLY as a
// whole; the pairs that met in one observed level are MEASURED (KX-003,
// KX-005, RG-001), and they put every 0x05f category seen before ship
// hulls and planetary items.
var announceCategories = []string{
	CatEngine, CatScanner, CatShield, CatArmor, CatBeam, CatTorpedo, CatBomb,
	CatMiningRobot, CatMineLayer, CatOrbital,
	CatStarbaseHull,
	CatElectrical, CatMechanical, CatTerraform,
	CatHull,
	CatPlanetary,
}

// announceOrder is the catalog's component indices in announcement
// order: by category (announceCategories), then by the item's index in
// its category (BINARY-ONLY, MEASURED only for ship hulls: RG-001 2434,
// Medium Freighter then Destroyer). Planetary items so run scanners,
// defenses, then the Genesis Device.
var announceOrder = sync.OnceValue(func() []int {
	cs := Components().Components
	rank := map[string]int{}
	for r, c := range announceCategories {
		rank[c] = r
	}
	order := make([]int, len(cs))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		ca, cb := cs[order[a]], cs[order[b]]
		if rank[ca.Category] != rank[cb.Category] {
			return rank[ca.Category] < rank[cb.Category]
		}
		return ca.Index < cb.Index
	})
	return order
})

// unannounced reports a component a race may use but is never told
// about: a Total Terraforming race gets no message for Gravity, Temp or
// Radiation Terraform ±3 (terraforming indices 8, 12 and 16; MESSAGES.md
// at stars-elegy f2778b3, BINARY-ONLY).
func unannounced(race Race, c Component) bool {
	return race.LRT.TotalTerraforming && c.Category == CatTerraform && (c.Index == 8 || c.Index == 12 || c.Index == 16)
}

// fieldOrder is the levels gained from before to after, in field order:
// the order of LevelUpCheck, which raises each field in turn.
func fieldOrder(before, after [NumFields]int) []int {
	var gained []int
	for f := range NumFields {
		for range after[f] - before[f] {
			gained = append(gained, f)
		}
	}
	return gained
}

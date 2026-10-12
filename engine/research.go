package engine

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
// Research every field is checked before any switch (the order is the
// code's choice; KERNEL.md does not give it).
//
// Under slower tech the stored accumulation is kept at half scale
// (KERNEL.md "Slower tech", CONFIRMED KX-003 S2, KX-005 R2): the year works
// on twice the stored value plus the research, against the doubled cost,
// and stores half of what is left, rounded up. A Generalized Research
// share for another field adds half of it, rounded down.
func AddResearch(s ResearchState, race Race, resources int, slowerTech bool) ResearchState {
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
	if race.LRT.GeneralizedResearch {
		for f := range NumFields {
			if f == s.Current {
				add(f, (resources+1)/2)
			} else {
				share := (3*resources + 19) / 20
				add(f, share/scale*scale)
			}
		}
	} else {
		add(s.Current, resources)
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

	gained := step(s.Current)
	for f := range NumFields {
		if f != s.Current {
			levelUp(f)
		}
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
	for gained {
		if s.Next == NextSameField && !lowest {
			if s.Levels[s.Current] < MaxTechLevel {
				break
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
		gained = step(s.Current)
	}
	if slowerTech {
		for f := range NumFields {
			s.Accumulated[f] = (s.Accumulated[f] + 1) / 2
		}
	}
	return s
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
		pl.Research = LevelUpCheck(pl.Research, pl.Race, g.SlowerTech)
	}
	return events
}

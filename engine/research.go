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
// one field, PG); other settings and slower tech are BINARY-ONLY.
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
// Generalized Research and the level cap are BINARY-ONLY. With Generalized
// Research every field is checked before any switch (the order is the
// code's choice; KERNEL.md does not give it).
func AddResearch(s ResearchState, race Race, resources int, slowerTech bool) ResearchState {
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
				add(f, (3*resources+19)/20)
			}
		}
	} else {
		add(s.Current, resources)
	}

	levelUp := func(field int) bool {
		gained := false
		for s.Levels[field] < s.maxLevel() {
			cost := ResearchLevelCost(s.Levels[field]+1, s.levelSum(), race.ResearchCosts[field], slowerTech)
			if s.Accumulated[field] < cost {
				break
			}
			s.Accumulated[field] -= cost
			s.Levels[field]++
			gained = true
		}
		return gained
	}

	gained := levelUp(s.Current)
	for f := range NumFields {
		if f != s.Current {
			levelUp(f)
		}
	}

	// Only a level-up in the current field switches fields. The leftover
	// moves to the new field, which is checked for level-ups the same
	// year, and may switch again. An explicit next field is used once and
	// then resets to "same field"; "lowest field" stays set.
	for gained && s.Next != NextSameField {
		next := s.Next
		if next == NextLowestField {
			next = 0
			for f := range NumFields {
				if s.Levels[f] < s.Levels[next] {
					next = f
				}
			}
		} else {
			s.Next = NextSameField
		}
		if next == s.Current {
			break
		}
		s.Accumulated[next] += s.Accumulated[s.Current]
		s.Accumulated[s.Current] = 0
		s.Current = next
		gained = levelUp(s.Current)
	}
	return s
}

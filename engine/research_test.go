package engine

import "testing"

func TestConfirmedResearchPG(t *testing.T) {
	// PG003: energy current, next "same field", electronics 5. Research each
	// year is trunc(P/10) + 10 from the previous year's population.
	want := map[int][2]int{ // year → energy level, accumulated
		2408: {2, 123}, 2409: {2, 186}, 2410: {3, 54}, 2413: {4, 7},
		2417: {5, 15}, 2421: {5, 638}, 2422: {6, 182}, 2426: {7, 163},
		2431: {8, 389}, 2435: {8, 2274}, 2436: {9, 343},
	}
	s := ResearchState{Current: Energy, Next: NextSameField}
	s.Levels[Energy] = 2
	s.Levels[Electronics] = 5
	s.Accumulated[Energy] = 65
	race := pgRace()
	for year := 2408; year <= 2436; year++ {
		pop := pgPopulation[year-1-2400][0]
		s = AddResearch(s, race, pop/10+10, false)
		if w, ok := want[year]; ok && (s.Levels[Energy] != w[0] || s.Accumulated[Energy] != w[1]) {
			t.Errorf("%d: energy %d/%d, want %d/%d", year, s.Levels[Energy], s.Accumulated[Energy], w[0], w[1])
		}
	}
}

func TestConfirmedResearchLevelCost(t *testing.T) {
	if got := ResearchLevelCost(9, 8+5, ResearchNormal, false); got != 2460 {
		t.Errorf("cost(9) = %d, want 2460", got)
	}
}

func TestPredictionResearchLevelCost(t *testing.T) {
	// c = 50 + 10·0 = 50.
	for _, tt := range []struct {
		setting ResearchCost
		slower  bool
		want    int
	}{
		{ResearchNormal, true, 100},
		{ResearchExpensive, true, 176},
	} {
		if got := ResearchLevelCost(1, 0, tt.setting, tt.slower); got != tt.want {
			t.Errorf("%+v: %d, want %d", tt, got, tt.want)
		}
	}
}

func TestPredictionResearchSeveralLevels(t *testing.T) {
	s := ResearchState{Current: Weapons, Next: NextSameField}
	// cost(1) = 50, cost(2) = 80 + 10 = 90; 150 → level 2, 10 left.
	s = AddResearch(s, pgRace(), 150, false)
	if s.Levels[Weapons] != 2 || s.Accumulated[Weapons] != 10 {
		t.Errorf("weapons %d/%d, want 2/10", s.Levels[Weapons], s.Accumulated[Weapons])
	}
}

func TestPredictionResearchNextLowestField(t *testing.T) {
	s := ResearchState{Current: Energy, Next: NextLowestField}
	s.Levels = [NumFields]int{0, 1, 0, 0, 0, 0}
	s = AddResearch(s, pgRace(), 60, false)
	// Energy 0 → 1 (cost 50 + 10·1 = 60), leaves 0; lowest is propulsion.
	if s.Levels[Energy] != 1 || s.Current != Propulsion || s.Accumulated[Energy] != 0 {
		t.Errorf("state %+v", s)
	}
	s = ResearchState{Current: Energy, Next: Biotech}
	s = AddResearch(s, pgRace(), 70, false)
	if s.Current != Biotech || s.Accumulated[Biotech] != 20 || s.Accumulated[Energy] != 0 {
		t.Errorf("state %+v", s)
	}
}

// TestConfirmedSlowerTech: under slower tech the stored accumulation is
// kept at half scale (KERNEL.md "Slower tech", KX-003 S2 vectors; KX-005
// R2 for a Generalized Research share).
func TestConfirmedSlowerTech(t *testing.T) {
	start := ResearchState{Levels: [NumFields]int{3, 3, 3, 3, 3, 3}, Current: Energy, Next: NextSameField}
	for _, c := range []struct {
		research   [2]int
		levels     [2]int
		accumulate [2]int
	}{
		{[2]int{435, 485}, [2]int{3, 4}, [2]int{218, 71}},
		{[2]int{937, 954}, [2]int{4, 5}, [2]int{79, 26}},
	} {
		s := start
		for y := range 2 {
			s = AddResearch(s, Race{}, c.research[y], true)
			if s.Levels[Energy] != c.levels[y] || s.Accumulated[Energy] != c.accumulate[y] {
				t.Errorf("research %v year %d: level %d stored %d, want %d and %d",
					c.research, y+1, s.Levels[Energy], s.Accumulated[Energy], c.levels[y], c.accumulate[y])
			}
		}
	}

	// A Generalized Research share o = 300 for another field stores 150.
	gr := Race{}
	gr.LRT.GeneralizedResearch = true
	s := ResearchState{Levels: [NumFields]int{20, 20, 20, 20, 20, 20}, Current: Energy, Next: NextSameField}
	s = AddResearch(s, gr, 1995, true)
	if s.Accumulated[Energy] != 499 || s.Accumulated[Weapons] != 150 {
		t.Errorf("stored energy %d, weapons %d; want 499 and 150", s.Accumulated[Energy], s.Accumulated[Weapons])
	}
}

// TestResearchSameFieldAtLevel26 checks KERNEL.md "Allocation": a current
// field reaching level 26 under "same field" moves research, with the
// leftover, to the lowest field that year, and the stored choice stays
// "same field" (CONFIRMED, KX-005 R3: energy 25 → 26, weapons current
// with 15 stored).
func TestResearchSameFieldAtLevel26(t *testing.T) {
	s := ResearchState{Current: Energy, Next: NextSameField}
	s.Levels = [NumFields]int{25, 3, 3, 3, 3, 3}
	s.Accumulated[Energy] = 85080
	// cost(26) = 84700 + 10·(25+15) = 85100; 85080 + 35 leaves 15.
	s = AddResearch(s, pgRace(), 35, false)
	if s.Levels[Energy] != MaxTechLevel || s.Current != Weapons || s.Next != NextSameField {
		t.Errorf("energy level %d, current %d, next %d; want 26, weapons, same field", s.Levels[Energy], s.Current, s.Next)
	}
	if s.Accumulated != [NumFields]int{0, 15, 0, 0, 0, 0} {
		t.Errorf("accumulated %v, want 15 in weapons", s.Accumulated)
	}
}

// TestResearchSameFieldAtLevel26RestOfYear checks that after the level-26
// switch the year goes on as if "lowest field" were chosen (KERNEL.md
// "Allocation", CONFIRMED KB-2A P0: research 1,135; energy 26, weapons
// 0 → 1, propulsion 0 → 1, then back to weapons with 185 stored, the
// choice still "same field"; keeping "same field" would have given
// weapons 2 with 155). The other levels are not in the spec: the costs
// fix their sum at 15 (weapons 1 costs 50 + 10·41 = 460, propulsion 1
// costs 470, 1,115 − 930 = 185), and propulsion leading back to weapons
// needs each of them at 1 or more, so 5 each here.
func TestResearchSameFieldAtLevel26RestOfYear(t *testing.T) {
	s := ResearchState{Current: Energy, Next: NextSameField}
	s.Levels = [NumFields]int{25, 0, 0, 5, 5, 5}
	// cost(26) = 84700 + 10·(25+15) = 85100, leaving 1,115.
	s.Accumulated[Energy] = 85100 + 1115 - 1135
	s = AddResearch(s, pgRace(), 1135, false)
	if s.Levels != [NumFields]int{26, 1, 1, 5, 5, 5} {
		t.Errorf("levels %v, want energy 26, weapons 1, propulsion 1", s.Levels)
	}
	if s.Current != Weapons || s.Next != NextSameField || s.Accumulated != [NumFields]int{0, 185, 0, 0, 0, 0} {
		t.Errorf("current %d, next %d, accumulated %v; want weapons, same field, 185 in weapons", s.Current, s.Next, s.Accumulated)
	}
}

// TestResearchSameFieldBelowLevel26 checks that a level other than 26
// keeps "same field" in place: energy 24 → 25 stays current.
func TestResearchSameFieldBelowLevel26(t *testing.T) {
	s := ResearchState{Current: Energy, Next: NextSameField}
	s.Levels = [NumFields]int{24, 3, 3, 3, 3, 3}
	// cost(25) = 77990 + 10·(24+15) = 78380.
	s.Accumulated[Energy] = 78380
	s = AddResearch(s, pgRace(), 10, false)
	if s.Levels[Energy] != 25 || s.Current != Energy || s.Accumulated[Energy] != 10 {
		t.Errorf("energy level %d, current %d, accumulated %v; want 25, energy, 10 in energy", s.Levels[Energy], s.Current, s.Accumulated)
	}
}

// TestResearchSameFieldAtCapLevel10 pins an ASSUMPTION (KERNEL.md gives
// the switch only at level 26; the level-10 cap is BINARY-ONLY): a capped
// player's field reaching 10 under "same field" stays current.
func TestResearchSameFieldAtCapLevel10(t *testing.T) {
	s := ResearchState{Current: Energy, Next: NextSameField, MaxLevel: 10}
	s.Levels = [NumFields]int{9, 3, 3, 3, 3, 3}
	// cost(10) = 3770 + 10·(9+15) = 4010.
	s.Accumulated[Energy] = 4010
	s = AddResearch(s, pgRace(), 15, false)
	if s.Levels[Energy] != 10 || s.Current != Energy || s.Next != NextSameField {
		t.Errorf("energy level %d, current %d, next %d; want 10, energy, same field", s.Levels[Energy], s.Current, s.Next)
	}
}

// TestResearchExplicitFieldOneLevel checks that an explicit next field
// takes the current field one level before switching (inferred from
// KERNEL.md "Allocation"; see AddResearch): energy 1 for 50, then weapons
// 1 for 60 with 40 left, not energy 2 for 50 + 90.
func TestResearchExplicitFieldOneLevel(t *testing.T) {
	s := ResearchState{Current: Energy, Next: Weapons}
	s = AddResearch(s, pgRace(), 150, false)
	if s.Levels != [NumFields]int{1, 1, 0, 0, 0, 0} || s.Current != Weapons || s.Next != NextSameField || s.Accumulated != [NumFields]int{0, 40, 0, 0, 0, 0} {
		t.Errorf("levels %v, current %d, next %d, accumulated %v; want energy 1, weapons 1, weapons current with 40, same field", s.Levels, s.Current, s.Next, s.Accumulated)
	}
}

// TestResearchStoredLowestOneLevel checks that a stored "lowest field"
// takes one level at a time (inferred, as above): energy 1 for 90, weapons
// 1 for 100, then back to energy with 30, not energy 2 for 90 + 130.
func TestResearchStoredLowestOneLevel(t *testing.T) {
	s := ResearchState{Current: Energy, Next: NextLowestField}
	s.Levels = [NumFields]int{0, 0, 1, 1, 1, 1}
	s = AddResearch(s, pgRace(), 220, false)
	if s.Levels != [NumFields]int{1, 1, 1, 1, 1, 1} || s.Current != Energy || s.Next != NextLowestField || s.Accumulated != [NumFields]int{30, 0, 0, 0, 0, 0} {
		t.Errorf("levels %v, current %d, next %d, accumulated %v; want all 1, energy current with 30, lowest field", s.Levels, s.Current, s.Next, s.Accumulated)
	}
}

// TestResearchLowestPicksCurrentAgain checks that when "lowest field"
// picks the current field again, it keeps its leftover and steps again:
// energy 1 for 100, energy is still lowest, energy 2 for 140.
func TestResearchLowestPicksCurrentAgain(t *testing.T) {
	s := ResearchState{Current: Energy, Next: NextLowestField}
	s.Levels = [NumFields]int{0, 1, 1, 1, 1, 1}
	s = AddResearch(s, pgRace(), 245, false)
	if s.Levels != [NumFields]int{2, 1, 1, 1, 1, 1} || s.Current != Weapons || s.Accumulated != [NumFields]int{0, 5, 0, 0, 0, 0} {
		t.Errorf("levels %v, current %d, accumulated %v; want energy 2, weapons current with 5", s.Levels, s.Current, s.Accumulated)
	}
}

// TestResearchSlowerTechWhileSwitching checks the doubled cost while a
// stored "lowest field" switches: energy 1 for 180, weapons 1 for 200,
// back to energy with 11, stored at half scale as 6.
func TestResearchSlowerTechWhileSwitching(t *testing.T) {
	s := ResearchState{Current: Energy, Next: NextLowestField}
	s.Levels = [NumFields]int{0, 0, 1, 1, 1, 1}
	s = AddResearch(s, pgRace(), 391, true)
	if s.Levels != [NumFields]int{1, 1, 1, 1, 1, 1} || s.Current != Energy || s.Accumulated != [NumFields]int{6, 0, 0, 0, 0, 0} {
		t.Errorf("levels %v, current %d, accumulated %v; want all 1, energy current with 6", s.Levels, s.Current, s.Accumulated)
	}
}

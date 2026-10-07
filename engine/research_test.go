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

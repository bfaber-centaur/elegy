package engine

import "testing"

// KX-002 vectors from stars-elegy KERNEL.md (CONFIRMED). The race is PG's:
// center 50, range 15..85 on every axis, growth 10%.

func TestConfirmedHabitabilityKX002(t *testing.T) {
	for _, tt := range []struct {
		name string
		env  [3]int
		want int
	}{
		{"H1", [3]int{60, 50, 50}, 92},
		{"H2", [3]int{70, 70, 50}, 58},
		{"H3", [3]int{80, 80, 80}, 3},
		{"H4", [3]int{85, 50, 50}, 41},
		{"H5", [3]int{90, 50, 50}, -5},
		{"H6", [3]int{10, 95, 50}, -15},
	} {
		if got := Habitability(pgRace(), tt.env); got != tt.want {
			t.Errorf("%s: Habitability(%v) = %d, want %d", tt.name, tt.env, got, tt.want)
		}
	}
}

func TestConfirmedMaxPopulationKX002(t *testing.T) {
	he, joat, obrm := pgRace(), pgRace(), pgRace()
	he.PRT = PRTHyperExpansion
	joat.PRT = PRTJackOfAllTrades
	obrm.LRT.OnlyBasicRemoteMining = true
	for _, tt := range []struct {
		name string
		race Race
		hab  int
		want int
	}{
		{"H1 hab 92", pgRace(), 92, 9200},
		{"H2 hab 58", pgRace(), 58, 5800},
		{"H3 hab 3", pgRace(), 3, 500},
		{"H4 hab 41", pgRace(), 41, 4100},
		{"H5 hostile", pgRace(), -5, 500},
		{"H6 hostile", pgRace(), -15, 500},
		{"P1 HE", he, 100, 5000},
		{"P2 JOAT", joat, 100, 12000},
		{"P3 OBRM", obrm, 100, 11000},
	} {
		if got := MaxPopulation(tt.race, tt.hab, 0); got != tt.want {
			t.Errorf("%s: max = %d, want %d", tt.name, got, tt.want)
		}
	}
	if got := he.growthRate(); got != 20 {
		t.Errorf("P1: HE growth rate = %d, want 20", got)
	}
}

func TestConfirmedPopulationGrowthKX002(t *testing.T) {
	for _, tt := range []struct {
		name              string
		p, k, max, g, hab int
		wantP, wantK      int
	}{
		{"H3 crowded, g < 1000", 300, 0, 500, 10, 3, 300, 24},
		{"H4 crowded, g < 1000", 3000, 0, 4100, 10, 41, 3015, 60},
		{"P1 HE, crowded, g ≥ 1000", 3000, 0, 5000, 20, 100, 3168, 0},
		{"P2 JOAT, crowded", 10005, 0, 12000, 10, 100, 10045, 2},
		{"P3 OBRM, crowded", 10005, 0, 11000, 10, 100, 10015, 0},
		{"G3 zero growth adds 1 to the carry", 9995, 0, 10000, 10, 100, 9995, 1},
		{"G2 within 10 units of max: frozen", 10005, 0, 10000, 10, 100, 10005, 0},
		{"G1 overcrowded deaths, g = −84", 12000, 0, 10000, 10, 100, 11899, 20},
		{"hostile hab −5", 1000, 0, 500, 10, -5, 995, 0},
		{"hostile hab −15", 1234, 10, 500, 10, -15, 1215, 59},
	} {
		gp, gk := GrowPopulation(tt.p, tt.k, tt.max, tt.g, tt.hab)
		if gp != tt.wantP || gk != tt.wantK {
			t.Errorf("%s: (%d,%d) → (%d,%d), want (%d,%d)", tt.name, tt.p, tt.k, gp, gk, tt.wantP, tt.wantK)
		}
	}
}

func TestConfirmedCapsKX002(t *testing.T) {
	colony := func(hab int) Colony {
		return Colony{Race: pgRace(), Hab: hab, MaxPop: MaxPopulation(pgRace(), hab, 0)}
	}
	if got := colony(58).MaxFactories(); got != 580 {
		t.Errorf("C1: max factories at hab 58 = %d, want 580", got)
	}
	if got := colony(3).MaxDefenses(); got != 12 {
		t.Errorf("C2: max defenses at hab 3 = %d, want 12", got)
	}
	if got := colony(41).MaxMines(); got != 410 {
		t.Errorf("C3: max mines at hab 41 = %d, want 410", got)
	}
}

func TestConfirmedEffectivePopulationKX002(t *testing.T) {
	// G1: P 12,000 at max 10,000 → E 11,000, so 1,100 resources (R0 10).
	if got := pgColony().Resources(12000, 0); got != 1100 {
		t.Errorf("Resources(12000) = %d, want 1100", got)
	}
}

func TestConfirmedMiningKX002(t *testing.T) {
	// N2: homeworld, 1000 working mines, eff 10, ironium concentration 20
	// (the homeworld floor of 30 sets output and depletion).
	years := []struct {
		conc, frac [3]int
		gain       [3]int // surface gain in the year that ends here
	}{
		{[3]int{20, 113, 84}, [3]int{242, 86, 157}, [3]int{}},
		{[3]int{20, 104, 79}, [3]int{88, 73, 34}, [3]int{300, 1130, 840}},
		{[3]int{19, 96, 74}, [3]int{189, 11, 77}, [3]int{300, 1040, 790}},
		{[3]int{19, 88, 70}, [3]int{35, 241, 11}, [3]int{300, 960, 740}},
	}
	for i := 0; i+1 < len(years); i++ {
		from, to := years[i], years[i+1]
		for m := range NumMinerals {
			d := Deposit{Concentration: from.conc[m], Fraction: from.frac[m]}
			gain, out := MineYear(d, 1000, 10, true, panicRand{})
			if out.Concentration != to.conc[m] || out.Fraction != to.frac[m] || gain != to.gain[m] {
				t.Errorf("%d mineral %d: gain %d deposit %+v, want gain %d conc %d frac %d",
					2407+i, m, gain, out, to.gain[m], to.conc[m], to.frac[m])
			}
		}
	}
}

func TestConfirmedResearchKX002(t *testing.T) {
	// R1: "costs 75% more", c = 210 → 368 (level 4, levels summing to 0).
	if got := ResearchLevelCost(4, 0, ResearchExpensive, false); got != 368 {
		t.Errorf("R1: cost = %d, want 368", got)
	}
	// R2: "costs 50% less", c = 200, 290, 430, 650 → 100, 145, 215, 325.
	for _, tt := range []struct{ level, sum, want int }{
		{3, 7, 100}, {4, 8, 145}, {5, 9, 215}, {6, 10, 325},
	} {
		if got := ResearchLevelCost(tt.level, tt.sum, ResearchCheap, false); got != tt.want {
			t.Errorf("R2: cost(%d, sum %d) = %d, want %d", tt.level, tt.sum, got, tt.want)
		}
	}
	// R5: Generalized Research, 211 → 106 to the current field, 32 each to
	// the others.
	r := pgRace()
	r.LRT.GeneralizedResearch = true
	// (Energy at level 10, so no field gains a level.)
	s := ResearchState{Current: Energy, Next: NextSameField}
	s.Levels[Energy] = 10
	s = AddResearch(s, r, 211, false)
	if s.Accumulated != [NumFields]int{106, 32, 32, 32, 32, 32} {
		t.Errorf("R5: accumulated %v, want 106 and 32 each", s.Accumulated)
	}
	// R6: research into a field at level 26 is lost.
	s = ResearchState{Current: Energy, Next: NextSameField}
	s.Levels[Energy] = MaxTechLevel
	if s = AddResearch(s, pgRace(), 1000, false); s.Accumulated[Energy] != 0 {
		t.Errorf("R6: accumulated %d at level 26", s.Accumulated[Energy])
	}
}

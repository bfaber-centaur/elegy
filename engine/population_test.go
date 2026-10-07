package engine

import "testing"

func TestConfirmedHabitabilityAtCenter(t *testing.T) {
	if got := Habitability(pgRace(), [3]int{50, 50, 50}); got != 100 {
		t.Errorf("hab = %d, want 100", got)
	}
}

func TestPredictionHabitability(t *testing.T) {
	tests := []struct {
		env  [3]int
		want int
	}{
		{[3]int{60, 50, 50}, 92},
		{[3]int{70, 50, 50}, 79},
		{[3]int{85, 50, 50}, 41},
		{[3]int{70, 70, 50}, 58},
		{[3]int{80, 80, 80}, 3},
		{[3]int{90, 50, 50}, -5},
		{[3]int{10, 95, 50}, -15},
	}
	for _, tt := range tests {
		if got := Habitability(pgRace(), tt.env); got != tt.want {
			t.Errorf("Habitability(%v) = %d, want %d", tt.env, got, tt.want)
		}
	}
}

func TestConfirmedMaxPopulation(t *testing.T) {
	if got := MaxPopulation(pgRace(), 100, 0); got != 10_000 {
		t.Errorf("max = %d, want 10000", got)
	}
}

func TestPredictionMaxPopulation(t *testing.T) {
	he, joat, obrm, joatObrm, ar := pgRace(), pgRace(), pgRace(), pgRace(), pgRace()
	he.PRT = PRTHyperExpansion
	joat.PRT = PRTJackOfAllTrades
	obrm.LRT.OnlyBasicRemoteMining = true
	joatObrm.PRT = PRTJackOfAllTrades
	joatObrm.LRT.OnlyBasicRemoteMining = true
	ar.PRT = PRTAlternateReality
	tests := []struct {
		name string
		race Race
		hab  int
		sb   int
		want int
	}{
		{"HE hab 100", he, 100, 0, 5000},
		{"JOAT hab 100", joat, 100, 0, 12000},
		{"OBRM hab 100", obrm, 100, 0, 11000},
		{"hab 3", pgRace(), 3, 0, 500},
		{"hostile", pgRace(), -20, 0, 500},
		{"JOAT+OBRM hab 79", joatObrm, 79, 0, 10428},
		{"AR no starbase", ar, 100, 0, 0},
		{"AR hull 3", ar, 7, 3, 10000},
	}
	for _, tt := range tests {
		if got := MaxPopulation(tt.race, tt.hab, tt.sb); got != tt.want {
			t.Errorf("%s: max = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// pgPopulation is PG-001..003: (P, k) for 2400..2436, G 10, hab 100,
// max 10,000.
var pgPopulation = [][2]int{
	{250, 0}, {275, 0}, {302, 50}, {332, 70}, {365, 90}, {402, 40}, {442, 60},
	{486, 80}, {535, 40}, {588, 90}, {647, 70}, {712, 40}, {783, 60}, {861, 90},
	{948, 0}, {1042, 80}, {1147, 0}, {1261, 70}, {1387, 80}, {1526, 50}, {1679, 10},
	{1847, 0}, {2031, 70}, {2234, 80}, {2458, 20}, {2704, 0}, {2958, 17}, {3218, 47},
	{3479, 12}, {3740, 4}, {3998, 10}, {4253, 97}, {4500, 64}, {4739, 14}, {4971, 35},
	{5190, 7}, {5402, 86},
}

func TestConfirmedPopulationGrowthPG(t *testing.T) {
	for i := 0; i+1 < len(pgPopulation); i++ {
		p, k := pgPopulation[i][0], pgPopulation[i][1]
		gp, gk := GrowPopulation(p, k, 10_000, 10, 100)
		if want := pgPopulation[i+1]; gp != want[0] || gk != want[1] {
			t.Errorf("%d→%d: (%d,%d) → (%d,%d), want (%d,%d)", 2400+i, 2401+i, p, k, gp, gk, want[0], want[1])
		}
	}
}

func TestPredictionPopulationGrowth(t *testing.T) {
	tests := []struct {
		name              string
		p, k, max, g, hab int
		wantP, wantK      int
	}{
		{"uncrowded", 1000, 0, 7900, 15, 79, 1118, 50},
		{"crowded, g ≥ 1000 quantization", 5000, 30, 10000, 15, 100, 5330, 30},
		{"crowded, g < 1000", 3000, 0, 8600, 10, 86, 3194, 70},
		{"zero growth adds 1 to the carry", 9995, 0, 10000, 10, 100, 9995, 1},
		{"within 10 units of max: frozen", 10005, 0, 10000, 10, 100, 10005, 0},
		{"overcrowded deaths", 12000, 0, 10000, 10, 100, 11949, 60},
		{"hostile hab −5", 1000, 0, 500, 10, -5, 995, 0},
		{"hostile hab −15", 1234, 10, 500, 10, -15, 1215, 59},
	}
	for _, tt := range tests {
		gp, gk := GrowPopulation(tt.p, tt.k, tt.max, tt.g, tt.hab)
		if gp != tt.wantP || gk != tt.wantK {
			t.Errorf("%s: (%d,%d) → (%d,%d), want (%d,%d)", tt.name, tt.p, tt.k, gp, gk, tt.wantP, tt.wantK)
		}
	}
}

func TestPredictionHyperExpansionDoublesGrowth(t *testing.T) {
	r := pgRace()
	r.PRT = PRTHyperExpansion
	if got := r.growthRate(); got != 20 {
		t.Errorf("growth rate = %d, want 20", got)
	}
}

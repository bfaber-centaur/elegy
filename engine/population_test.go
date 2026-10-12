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
		{[3]int{70, 50, 50}, 79},
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
	joatObrm, ar := pgRace(), pgRace()
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

// TestOvercrowdingFloor checks the −300 floor of the overcrowded branch
// (stars-elegy 397ebfc KERNEL.md "Population growth": g = 4·max(−300,
// trunc(c/−10) + 99), deaths at most 12% a year), which no oracle case
// has reached; it needs P at about 4 times max. The branch is CONFIRMED
// above the floor (KX-002 G1); the floor itself is BINARY-ONLY.
func TestOvercrowdingFloor(t *testing.T) {
	for _, c := range []struct {
		pop, maxPop, wantPop int
	}{
		// c = 3980: trunc(−398) + 99 = −299, just above the floor;
		// g = −1196, t = −476008: 39800 − 4760, carry −8 → 35039, 92.
		{39800, 10000, 35039},
		// c = 4000: −400 + 99 = −301, floored to −300; g = −1200,
		// 40000 − 4800 (without the floor: −1204, 35184).
		{40000, 10000, 35200},
		// c = 5000: floored; 12% die.
		{50000, 10000, 44000},
	} {
		if got, _ := GrowPopulation(c.pop, 0, c.maxPop, 10, 100); got != c.wantPop {
			t.Errorf("P %d, max %d: population %d, want %d", c.pop, c.maxPop, got, c.wantPop)
		}
	}
}

// TestBareRecordDropped checks the year's file round trip for unowned
// planets (stars-elegy 9a355fe TAKEOVER.md "Capture: what a planet
// keeps", BINARY-ONLY): with no mines, factories, defenses or scanner the
// carry and the leftover setting are lost; any one of them keeps the
// record and the carry.
func TestBareRecordDropped(t *testing.T) {
	for _, c := range []struct {
		name      string
		set       func(*Planet)
		wantCarry int
	}{
		{"bare", func(*Planet) {}, 0},
		{"mines", func(p *Planet) { p.Mines = 1 }, 37},
		{"factories", func(p *Planet) { p.Factories = 1 }, 37},
		{"defenses", func(p *Planet) { p.Defenses = 1 }, 37},
		{"scanner", func(p *Planet) { p.HasScanner = true }, 37},
	} {
		l := newTKLab(t, 3)
		pi := l.planet(NoOwner, 0, 0)
		p := &l.g.Planets[pi]
		p.GrowthCarry, p.LeftoverOnly = 37, true
		c.set(p)
		got := l.turn(&seqRand{}).Game.Planets[pi]
		if got.GrowthCarry != c.wantCarry || got.LeftoverOnly != (c.wantCarry != 0) {
			t.Errorf("%s: carry %d, leftover %v; want %d, %v", c.name, got.GrowthCarry, got.LeftoverOnly, c.wantCarry, c.wantCarry != 0)
		}
	}
	// An owned planet keeps its record.
	l := newTKLab(t, 3)
	pi := l.planet(1, 0, 1000)
	l.g.Planets[pi].GrowthCarry = 37
	if got := l.turn(&seqRand{}).Game.Planets[pi]; got.GrowthCarry == 0 {
		t.Errorf("owned bare planet: carry 0, want the year's growth from 37")
	}
}

// TestColonyBeforeGrowthKeepsCarry checks the branch no run has observed
// (stars-elegy 9a355fe, BINARY-ONLY): a planet colonized in the first
// order step, before the year's growth, grows from the carry it holds in
// memory, so the colony's population and carry run 37 hundredths ahead of
// the same colony on a planet with carry 0.
func TestColonyBeforeGrowthKeepsCarry(t *testing.T) {
	grown := func(carry int) int {
		l := newTKLab(t, 3)
		pi := l.planet(NoOwner, 0, 0)
		l.g.Planets[pi].GrowthCarry = carry
		fi := l.fleet(0, pi, 0, Stack{Design: l.colonyShip(), Count: 1})
		l.g.Fleets[fi].Cargo.Colonists = 25
		l.g.Fleets[fi].Task = colonizeTask
		p := l.turn(&seqRand{}).Game.Planets[pi]
		if p.Owner != 0 {
			t.Fatalf("carry %d: owner %d, want the colony of player 0", carry, p.Owner)
		}
		return 100*p.Population + p.GrowthCarry
	}
	if a, b := grown(37), grown(0); a-b != 37 {
		t.Errorf("colony grown from carry 37: %d hundredths, from 0: %d; want 37 apart", a, b)
	}
}

package engine

import "testing"

func TestConfirmedResourcesPG(t *testing.T) {
	// PG-001..003: R0 10, F 10, 10 factories: trunc(P/10) + 10.
	c := pgColony()
	for _, tt := range []struct{ pop, want int }{{486, 58}, {1042, 114}, {2704, 280}, {5190, 529}} {
		if got := c.Resources(tt.pop, 10); got != tt.want {
			t.Errorf("Resources(%d) = %d, want %d", tt.pop, got, tt.want)
		}
	}
	// PQ-001: no factories gives trunc(P/10).
	if got := c.Resources(1050, 0); got != 105 {
		t.Errorf("Resources(1050, no factories) = %d, want 105", got)
	}
}

func TestConfirmedOperableInstallations(t *testing.T) {
	c := pgColony()
	// PQ-001 C09: 550 units after growth operate 55 mines and factories.
	if got := c.OperableMines(550); got != 55 {
		t.Errorf("operable mines = %d, want 55", got)
	}
	if got := c.OperableFactories(550); got != 55 {
		t.Errorf("operable factories = %d, want 55", got)
	}
	// PQ-001 C13: ceil(1111/25) = 45 defenses.
	if got := c.OperableDefenses(1111); got != 45 {
		t.Errorf("operable defenses = %d, want 45", got)
	}
	// PG mining: all 10 installed mines work.
	if got := c.WorkingMines(486, 10); got != 10 {
		t.Errorf("working mines = %d, want 10", got)
	}
}

func TestPredictionCapsAndResources(t *testing.T) {
	c := pgColony()
	if got := c.MaxMines(); got != 1000 {
		t.Errorf("max mines = %d, want 1000", got)
	}
	if got := c.MaxFactories(); got != 1000 {
		t.Errorf("max factories = %d, want 1000", got)
	}
	if got := c.MaxDefenses(); got != 100 {
		t.Errorf("max defenses = %d, want 100", got)
	}
	low := Colony{Race: pgRace(), Hab: 2, MaxPop: 500}
	if got := low.MaxDefenses(); got != 10 {
		t.Errorf("max defenses at hab 2 = %d, want 10", got)
	}
	if got := low.MaxMines(); got != 50 {
		t.Errorf("max mines at max 500 = %d, want 50", got)
	}
	// Over max: E = min(2·max, max + (P−max)/2).
	if got := c.Resources(12000, 0); got != 1100 {
		t.Errorf("Resources(12000) = %d, want 1100", got)
	}
	if got := c.Resources(40000, 0); got != 2000 {
		t.Errorf("Resources(40000) = %d, want 2000", got)
	}
	// A positive result below 1 becomes 1; no population gives 0.
	if got := c.Resources(5, 0); got != 1 {
		t.Errorf("Resources(5) = %d, want 1", got)
	}
	if got := c.Resources(0, 10); got != 0 {
		t.Errorf("Resources(0) = %d, want 0", got)
	}
	// Alternate Reality: trunc(sqrt((E/R0)·max(1, energy))·max(25, hab)·0.1 + 0.999).
	ar := pgRace()
	ar.PRT = PRTAlternateReality
	arc := Colony{Race: ar, Hab: 40, MaxPop: 10000, EnergyTech: 4}
	// sqrt(100·4) = 20; 20·40·0.1 = 80.
	if got := arc.Resources(1000, 0); got != 80 {
		t.Errorf("AR Resources = %d, want 80", got)
	}
	if got := arc.WorkingMines(1000, 0); got != 31 {
		t.Errorf("AR working mines = %d, want 31", got)
	}
}

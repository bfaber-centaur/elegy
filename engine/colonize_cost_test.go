package engine

import "testing"

func TestColonizeCostBeforeResearch(t *testing.T) {
	// A colony ship arriving this year colonizes after the year's level-up
	// checks. Its ⌊3/4⌋ minerals are priced at the owner's tech from
	// before them (MEASURED, RG-001 2434; the year's start rather than
	// step 2.4's levels is an ASSUMPTION): 18/6/17 at tech 3 (T-30), not
	// the tech-26 4/1/5 the accumulated research reaches this turn.
	l := newTKLab(t, 3)
	for f := range NumFields {
		l.g.Players[0].Research.Levels[f] = 3
		l.g.Players[0].Research.Accumulated[f] = 1 << 30
	}
	pi := l.planet(NoOwner, 0, 0)
	l.arriving(0, pi, l.colonyShip(), 25, colonizeTask)
	g := l.turn(&seqRand{}).Game
	if lv := g.Players[0].Research.Levels; lv[Energy] != MaxTechLevel {
		t.Fatalf("levels %v: the research did not reach tech %d", lv, MaxTechLevel)
	}
	if p := g.Planets[pi]; p.Owner != 0 || p.Surface != (Minerals{18, 6, 17}) {
		t.Errorf("owner %d surface %v, want player 0 and 18/6/17 (tech 3)", p.Owner, p.Surface)
	}
	if g.costLevels != nil {
		t.Error("costLevels kept after the turn")
	}
}

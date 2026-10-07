package engine

import (
	"errors"
	"reflect"
	"testing"
)

// KX-001 (stars-elegy PARITY.md "KX-001"; KERNEL.md "Item costs", "Auto
// Alchemy before a multi-count item", AR maximum population and
// resources). Oracle-measured: ground truth.

func TestConfirmedProductionKX001(t *testing.T) {
	m500 := Minerals{500, 500, 500}
	aa := q(ItemAutoAlchemy, 1, 0)
	ma := func(r *Race) { r.LRT.MineralAlchemy = true }
	cases := []pqCase{
		{name: "A1", pop: 9000, defenses: 10, minerals: Minerals{100, 100, 0},
			queue:         []QueueItem{aa, q(ItemFactory, 5, 0), q(ItemMine, 2, 0)},
			wantFactories: 2, wantDefenses: 10, wantMinerals: Minerals{108, 108, 0},
			wantQueue:  []QueueItem{q(ItemMineralAlchemy, 1, 78), aa, q(ItemFactory, 3, 24), q(ItemMine, 2, 0)},
			wantEvents: []EventKind{EventAlchemy, EventBuilt}},
		{name: "A2", pop: 9000, defenses: 10, minerals: Minerals{100, 100, 6},
			queue:         []QueueItem{aa, q(ItemFactory, 5, 0), q(ItemMine, 2, 0)},
			wantFactories: 3, wantDefenses: 10, wantMinerals: Minerals{108, 108, 2},
			wantQueue: []QueueItem{q(ItemMineralAlchemy, 1, 68), aa, q(ItemFactory, 2, 24), q(ItemMine, 2, 0)}},
		{name: "A3", pop: 9000, defenses: 10, minerals: Minerals{100, 100, 0},
			queue:         []QueueItem{aa, q(ItemAutoFactories, 5, 0), q(ItemMine, 2, 0)},
			wantFactories: 2, wantDefenses: 10, wantMinerals: Minerals{108, 108, 0},
			wantQueue: []QueueItem{q(ItemMineralAlchemy, 1, 80), aa, q(ItemAutoFactories, 5, 0), q(ItemMine, 2, 0)}},
		{name: "A4", pop: 8200, defenses: 10, minerals: Minerals{100, 100, 0},
			queue:         []QueueItem{aa, q(ItemFactory, 2, 0), q(ItemMine, 2, 0)},
			wantFactories: 2, wantDefenses: 10, wantMinerals: Minerals{108, 108, 0},
			wantQueue: []QueueItem{q(ItemMine, 2, 19)}},
		{name: "M1", pop: 2600, defenses: 10, minerals: Minerals{100, 100, 100}, race: ma,
			queue:        []QueueItem{aa},
			wantDefenses: 10, wantMinerals: Minerals{110, 110, 110},
			wantQueue:  []QueueItem{q(ItemMineralAlchemy, 1, 43), aa},
			wantEvents: []EventKind{EventAlchemy}},
		{name: "M2", pop: 3000, defenses: 10, minerals: Minerals{100, 100, 0}, race: ma,
			queue:         []QueueItem{aa, q(ItemFactory, 5, 0), q(ItemMine, 2, 0)},
			wantFactories: 2, wantDefenses: 10, wantMinerals: Minerals{111, 111, 3},
			wantQueue: []QueueItem{q(ItemMineralAlchemy, 1, 15), aa, q(ItemFactory, 3, 24), q(ItemMine, 2, 0)}},
		// stars-elegy #18: the prefix stays with an auto item (A5); any short
		// mineral blocks an auto item even when resources are lower (A6 with
		// a prefix, A7 without).
		{name: "A5", pop: 9000, defenses: 10, minerals: Minerals{100, 100, 0},
			queue:         []QueueItem{aa, q(ItemAutoFactories, 2, 0), q(ItemMine, 2, 0)},
			wantFactories: 2, wantMines: 2, wantDefenses: 10, wantMinerals: Minerals{108, 108, 0},
			wantQueue: []QueueItem{aa, q(ItemAutoFactories, 2, 0)}, wantResearch: 70, wantCompleted: true,
			wantEvents: []EventKind{EventAlchemy, EventBuilt, EventBuilt}},
		{name: "A6", pop: 10, defenses: 10, minerals: Minerals{100, 100, 0},
			queue:        []QueueItem{aa, q(ItemAutoFactories, 2, 0)},
			wantDefenses: 10, wantMinerals: Minerals{100, 100, 0},
			wantQueue: []QueueItem{q(ItemMineralAlchemy, 1, 1), aa, q(ItemAutoFactories, 2, 0)}},
		{name: "A7", pop: 10, defenses: 10, minerals: Minerals{100, 100, 0},
			queue:        []QueueItem{q(ItemAutoFactories, 2, 0), q(ItemMine, 1, 0)},
			wantDefenses: 10, wantMinerals: Minerals{100, 100, 0},
			wantQueue: []QueueItem{q(ItemAutoFactories, 2, 0), q(ItemMine, 1, 39)}},
		// M3's race was over its advantage-point budget and the game raised
		// colonists per resource to 2,400 before production (not modelled);
		// the case starts from the degraded race.
		{name: "M3", pop: 310, defenses: 10, minerals: m500,
			race: func(r *Race) {
				r.ColonistsPerResource, r.FactoryCost, r.MineCost, r.FactoryLessGermanium = 2400, 7, 3, true
			},
			queue:         []QueueItem{q(ItemFactory, 3, 0), q(ItemMine, 4, 0)},
			wantFactories: 1, wantDefenses: 10, wantMinerals: Minerals{500, 500, 495},
			wantQueue: []QueueItem{q(ItemFactory, 2, 84), q(ItemMine, 4, 0)}},
		{name: "M3b", pop: 310, defenses: 10, minerals: m500,
			race: func(r *Race) {
				r.FactoryCost, r.MineCost, r.FactoryLessGermanium = 15, 8, true
			},
			queue:         []QueueItem{q(ItemFactory, 2, 0), q(ItemMine, 4, 0)},
			wantFactories: 2, wantDefenses: 10, wantMinerals: Minerals{500, 500, 494},
			wantQueue: []QueueItem{q(ItemMine, 4, 24)}},
		{name: "M4", pop: 400, defenses: 10, minerals: m500,
			race:         func(r *Race) { r.PRT = PRTInnerStrength },
			queue:        []QueueItem{q(ItemDefenses, 5, 0)},
			wantDefenses: 14, wantMinerals: Minerals{487, 487, 487},
			wantQueue: []QueueItem{q(ItemDefenses, 1, 54)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runPQ(t, tc) })
	}
}

func TestConfirmedItemCosts(t *testing.T) {
	r := pgRace()
	if got := ItemCost(r, ItemFactory); got != (Cost{10, Minerals{0, 0, 4}}) {
		t.Errorf("PG factory %v", got)
	}
	r.FactoryLessGermanium = true
	if got := ItemCost(r, ItemAutoFactories); got != (Cost{10, Minerals{0, 0, 3}}) {
		t.Errorf("factory with less germanium %v", got)
	}
	r.PRT = PRTInnerStrength
	if got := ItemCost(r, ItemAutoDefenses); got != (Cost{9, Minerals{3, 3, 3}}) {
		t.Errorf("Inner Strength defenses %v", got)
	}
	r.LRT.MineralAlchemy = true
	if got := ItemCost(r, ItemAutoAlchemy); got != (Cost{Resources: 25}) {
		t.Errorf("Mineral Alchemy %v", got)
	}
}

// kxAR is PG001 2407 with player 1 made Alternate Reality: Endeavor with
// its Space Station (hull 3, maximum 10,000), or without a starbase.
func kxAR(starbase bool) Game {
	g := pgHomeworld()
	g.Players[0].Race.PRT = PRTAlternateReality
	if starbase {
		g.Planets[0].StarbaseHull = 3
	}
	return g
}

func TestConfirmedAlternateRealityKX001(t *testing.T) {
	t.Run("Z2 starbase kept", func(t *testing.T) {
		r, err := GenerateTurn(kxAR(true), nil, Jrc3(), &seqRand{})
		if err != nil {
			t.Fatal(err)
		}
		p := r.Game.Planets[0]
		if p.Population != 535 || p.GrowthCarry != 40 {
			t.Errorf("population (%d,%d), want (535,40)", p.Population, p.GrowthCarry)
		}
		// Research 99 (floating E/R0), added to energy's 65.
		if got := r.Game.Players[0].Research.Accumulated[Energy]; got != 65+99 {
			t.Errorf("energy %d, want 164", got)
		}
		// 22 mines; observed +7/+25/+19 (both random +1s came up).
		start := pgDeposits[0].surface
		if got := (Minerals{p.Surface[0] - start[0], p.Surface[1] - start[1], p.Surface[2] - start[2]}); got != (Minerals{7, 25, 19}) {
			t.Errorf("mined %v, want [7 25 19]", got)
		}
	})
	t.Run("Z3 hostile, no starbase", func(t *testing.T) {
		g := kxAR(false)
		g.Players[0].Race.Env[Gravity] = EnvRange{Center: 85, Low: 70, High: 100}
		if h := Habitability(g.Players[0].Race, g.Planets[0].Env); h != -15 {
			t.Fatalf("hab %d, want -15", h)
		}
		// The player shuffle's draw, then mining.
		r, err := GenerateTurn(g, nil, Jrc3(), &seqRand{draws: []int{0, 0, 0, 99}})
		if err != nil {
			t.Fatal(err)
		}
		p := r.Game.Planets[0]
		if p.Population != 479 || p.GrowthCarry != 51 {
			t.Errorf("population (%d,%d), want (479,51)", p.Population, p.GrowthCarry)
		}
		if got := r.Game.Players[0].Research.Accumulated[Energy]; got != 65+1 {
			t.Errorf("energy %d, want 66 (1 resource)", got)
		}
		start := pgDeposits[0].surface
		if got := (Minerals{p.Surface[0] - start[0], p.Surface[1] - start[1], p.Surface[2] - start[2]}); got != (Minerals{7, 25, 18}) {
			t.Errorf("mined %v, want [7 25 18]", got)
		}
	})
}

// Z1: the original cannot generate the year (divide by zero). Elegy
// decision: GenerateTurn refuses it with a typed error and changes nothing.
func TestElegyDecisionZeroMaxPopulation(t *testing.T) {
	g := kxAR(false)
	before := g.clone()
	_, err := GenerateTurn(g, nil, Jrc3(), highRand{})
	var z *ZeroMaxPopulationError
	if !errors.As(err, &z) || z.Planet != 7 {
		t.Fatalf("err = %v, want *ZeroMaxPopulationError for planet 7", err)
	}
	if !reflect.DeepEqual(g, before) {
		t.Error("input game changed")
	}
}

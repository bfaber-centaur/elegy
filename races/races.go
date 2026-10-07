// Package races scores, repairs and generates race designs: the advantage
// points rule, legality and repairs at game creation, the wizard's Random
// race, and the yearly check of a running game.
//
// Every rule comes from stars-elegy docs/RACES.md (habitability from
// KERNEL.md); comments name the section and its status. What RACES.md
// leaves open is marked ASSUMPTION.
package races

import (
	"github.com/bfaber-centaur/elegy/engine"
)

// Design is a race as the race wizard and race files describe it: the
// engine's race settings plus the settings only creation and scoring use.
type Design struct {
	engine.Race
	Name string

	// ExpensiveAt3 is "expensive fields start at tech 3".
	ExpensiveAt3 bool
	// Spend is the leftover-points spend, 0–4 in the wizard (0 surface
	// minerals, 1 concentrations, 2 mines, 3 factories, 4 defenses); 5
	// and 6 are accepted and act as 0 (RACES.md "At game creation").
	Spend int
	// Stat15 is an unused race field that must be 0.
	Stat15 int
	// Random marks the wizard's Random race template, replaced by a
	// generated race at creation.
	Random bool
	// Tampered is the per-race flag a repair or a penalty sets.
	Tampered bool
}

// Setting ranges (RACES.md "Race settings").
const (
	MinGrowth, MaxGrowth = 1, 20
	MinColonists         = 700
	MaxColonists         = 2500
	MinFactoryOutput     = 5
	MaxFactoryOutput     = 15
	MinFactoryCost       = 5
	MaxFactoryCost       = 25
	MinOperated          = 5
	MaxOperated          = 25
	MinMineOutput        = 5
	MaxMineOutput        = 25
	MinMineCost          = 2
	MaxMineCost          = 15
	MaxSpend             = 6
)

// Default is the wizard's default (Humanoid) race: JOAT, growth 15, 15–85
// centre 50 on every axis, the standard economy and research, no LRTs. It
// scores 25 points (RACES.md "Race settings").
func Default() Design {
	r := engine.Race{
		PRT: engine.PRTJackOfAllTrades, GrowthRate: 15,
		ColonistsPerResource: 1000, FactoryOutput: 10, FactoryCost: 10, FactoriesOperated: 10,
		MineOutput: 10, MineCost: 5, MinesOperated: 10,
	}
	for i := range r.Env {
		r.Env[i] = engine.EnvRange{Center: 50, Low: 15, High: 85}
	}
	return Design{Race: r}
}

// lrtList is the lesser racial traits in RACES.md order with their point
// values (RACES.md "Traits", CONFIRMED). Negative values are the "good"
// traits.
var lrtList = []struct {
	code  string
	value int
	field func(*engine.LRTs) *bool
}{
	{"IFE", -235, func(l *engine.LRTs) *bool { return &l.ImprovedFuelEfficiency }},
	{"TT", -25, func(l *engine.LRTs) *bool { return &l.TotalTerraforming }},
	{"ARM", -159, func(l *engine.LRTs) *bool { return &l.AdvancedRemoteMining }},
	{"ISB", -201, func(l *engine.LRTs) *bool { return &l.ImprovedStarbases }},
	{"GR", 40, func(l *engine.LRTs) *bool { return &l.GeneralizedResearch }},
	{"UR", -240, func(l *engine.LRTs) *bool { return &l.UltimateRecycling }},
	{"MA", -155, func(l *engine.LRTs) *bool { return &l.MineralAlchemy }},
	{"NRSE", 160, func(l *engine.LRTs) *bool { return &l.NoRamScoopEngines }},
	{"CE", 240, func(l *engine.LRTs) *bool { return &l.CheapEngines }},
	{"OBRM", 255, func(l *engine.LRTs) *bool { return &l.OnlyBasicRemoteMining }},
	{"NAS", 325, func(l *engine.LRTs) *bool { return &l.NoAdvancedScanners }},
	{"LSP", 180, func(l *engine.LRTs) *bool { return &l.LowStartingPopulation }},
	{"BET", 70, func(l *engine.LRTs) *bool { return &l.BleedingEdgeTech }},
	{"RS", 30, func(l *engine.LRTs) *bool { return &l.RegeneratingShields }},
}

// NumLRTs is the number of lesser racial traits.
var NumLRTs = len(lrtList)

// HasLRT reports whether the race has lesser trait i (RACES.md order).
func (d *Design) HasLRT(i int) bool { return *lrtList[i].field(&d.Race.LRT) }

// SetLRT sets lesser trait i (RACES.md order).
func (d *Design) SetLRT(i int, on bool) { *lrtList[i].field(&d.Race.LRT) = on }

// LRTCode is lesser trait i's abbreviation.
func LRTCode(i int) string { return lrtList[i].code }

// prtCost is subtracted from P (RACES.md "Traits", CONFIRMED).
var prtCost = map[engine.PRT]int{
	engine.PRTHyperExpansion: 40, engine.PRTSuperStealth: 95, engine.PRTWarMonger: 45,
	engine.PRTClaimAdjuster: 10, engine.PRTInnerStrength: -100, engine.PRTSpaceDemolition: -150,
	engine.PRTPacketPhysics: 120, engine.PRTInterstellarTraveler: 180,
	engine.PRTAlternateReality: 90, engine.PRTJackOfAllTrades: -66,
}

// AllPRTs are the ten primary racial traits in RACES.md order.
var AllPRTs = []engine.PRT{
	engine.PRTHyperExpansion, engine.PRTSuperStealth, engine.PRTWarMonger,
	engine.PRTClaimAdjuster, engine.PRTInnerStrength, engine.PRTSpaceDemolition,
	engine.PRTPacketPhysics, engine.PRTInterstellarTraveler,
	engine.PRTAlternateReality, engine.PRTJackOfAllTrades,
}

func validPRT(p engine.PRT) bool {
	_, ok := prtCost[p]
	return ok
}

// researchLevel is r_f: 0 for "costs 75% extra", 1 normal, 2 "costs 50%
// less".
func researchLevel(c engine.ResearchCost) int {
	switch c {
	case engine.ResearchExpensive:
		return 0
	case engine.ResearchCheap:
		return 2
	}
	return 1
}

func researchCost(level int) engine.ResearchCost {
	switch level {
	case 0:
		return engine.ResearchExpensive
	case 2:
		return engine.ResearchCheap
	}
	return engine.ResearchNormal
}

func clamp(v, lo, hi int) int { return max(lo, min(hi, v)) }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

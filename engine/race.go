package engine

// Axis indexes the three environment axes.
const (
	Gravity = iota
	Temperature
	Radiation
)

// EnvRange is a race's tolerance on one environment axis, on the 0–100
// internal scale.
type EnvRange struct {
	Center, Low, High int
	Immune            bool
}

// PRT is a primary racial trait. Only the traits the kernel rules mention
// are named; the zero value is an ordinary race.
type PRT int

const (
	PRTOther PRT = iota
	PRTHyperExpansion
	PRTJackOfAllTrades
	PRTAlternateReality
	PRTInnerStrength
	PRTWarMonger
	PRTInterstellarTraveler
	PRTClaimAdjuster
	PRTSuperStealth
	PRTPacketPhysics
	PRTSpaceDemolition
)

// LRTs is the set of lesser racial traits the kernel rules mention.
type LRTs struct {
	OnlyBasicRemoteMining bool
	GeneralizedResearch   bool
	MineralAlchemy        bool
	RegeneratingShields   bool
	CheapEngines          bool
	BleedingEdgeTech      bool
	NoAdvancedScanners    bool
	ImprovedStarbases     bool
}

// ResearchCost is a race's research-cost setting for one field.
type ResearchCost int

const (
	ResearchNormal    ResearchCost = iota
	ResearchExpensive              // "costs 75% more"
	ResearchCheap                  // "costs 50% less"
)

// Race holds the race settings the peaceful kernel uses. Values are the race
// wizard's values unless noted.
type Race struct {
	Env        [3]EnvRange
	GrowthRate int // percent per year
	PRT        PRT
	LRT        LRTs

	ColonistsPerResource int // colonists, e.g. 1000
	FactoryOutput        int // resources per 10 factories
	FactoriesOperated    int // per 10,000 colonists
	MinesOperated        int // per 10,000 colonists
	MineOutput           int // the wizard's mine output setting (eff)

	FactoryCost int // resources per factory
	MineCost    int // resources per mine
	// FactoryLessGermanium is "factories cost 1 kT less germanium".
	FactoryLessGermanium bool

	ResearchCosts [NumFields]ResearchCost
}

// colonistsPerResourceUnits is R0: the wizard value in units of 100.
func (r Race) colonistsPerResourceUnits() int {
	return r.ColonistsPerResource / 100
}

// growthRate is G, doubled for Hyper-Expansion (BINARY-ONLY).
func (r Race) growthRate() int {
	if r.PRT == PRTHyperExpansion {
		return 2 * r.GrowthRate
	}
	return r.GrowthRate
}

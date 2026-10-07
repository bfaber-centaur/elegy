package engine

import "math"

// Mineral indexes.
const (
	Ironium = iota
	Boranium
	Germanium
	NumMinerals
)

// Minerals is an amount of each mineral in kT.
type Minerals [NumMinerals]int

// Cost is a production cost.
type Cost struct {
	Resources int
	Minerals  Minerals
}

// Colony is the derived per-year view of an owned planet that the economy
// rules read: the owner's race and research levels plus the planet's
// habitability and maximum population.
type Colony struct {
	Race       Race
	EnergyTech int
	Hab        int
	MaxPop     int
}

// NewColony derives the economy view of planet p owned by player.
func NewColony(p *Planet, player *Player) Colony {
	hab := Habitability(player.Race, p.Env)
	return Colony{
		Race:       player.Race,
		EnergyTech: player.Research.Levels[Energy],
		Hab:        hab,
		MaxPop:     MaxPopulation(player.Race, hab, p.StarbaseHull),
	}
}

func (c Colony) alternateReality() bool { return c.Race.PRT == PRTAlternateReality }

// MaxMines, MaxFactories and MaxDefenses are the planet maximums
// (KERNEL.md "Caps", BINARY-ONLY).
func (c Colony) MaxMines() int {
	if c.alternateReality() {
		return 0
	}
	return max(10, c.MaxPop*c.Race.MinesOperated/100)
}

func (c Colony) MaxFactories() int {
	if c.alternateReality() {
		return 0
	}
	return max(10, c.MaxPop*c.Race.FactoriesOperated/100)
}

func (c Colony) MaxDefenses() int {
	if c.alternateReality() {
		return 0
	}
	return min(100, max(10, 4*c.Hab))
}

// OperableMines, OperableFactories and OperableDefenses are the
// installations population pop (units) can operate. CONFIRMED for the
// production caps of auto items (PQ-001 C04, C09, C13, C14).
func (c Colony) OperableMines(pop int) int {
	return max(1, min(c.MaxMines(), pop*c.Race.MinesOperated/100))
}

func (c Colony) OperableFactories(pop int) int {
	return max(1, min(c.MaxFactories(), pop*c.Race.FactoriesOperated/100))
}

func (c Colony) OperableDefenses(pop int) int {
	return min(c.MaxDefenses(), 1000, (pop+24)/25)
}

// WorkingMines is the number of mines that mine this year.
// CONFIRMED for non-AR (PG mining) and AR (KX-001 Z2, Z3).
func (c Colony) WorkingMines(pop, installed int) int {
	if c.alternateReality() {
		return int(math.Sqrt(float64(pop)))
	}
	return min(installed, c.OperableMines(pop))
}

// effectivePop is E: population above the maximum counts half, up to twice
// the maximum (BINARY-ONLY above max).
func (c Colony) effectivePop(pop int) int {
	if pop <= c.MaxPop {
		return pop
	}
	return min(2*c.MaxPop, c.MaxPop+(pop-c.MaxPop)/2)
}

// Resources is the planet's resources for the year from population pop
// (units) and installed factories.
//
// KERNEL.md "Resources per planet": CONFIRMED for R0 10, F 10, 10 factories
// (PG) and for no factories (PQ-001); AR CONFIRMED at one point (KX-001 Z2,
// Z3); the over-max rule is BINARY-ONLY.
func (c Colony) Resources(pop, factories int) int {
	if pop <= 0 {
		return 0
	}
	e := c.effectivePop(pop)
	var r int
	if c.alternateReality() {
		// E/R0 is a floating-point division (CONFIRMED, KX-001 Z2).
		ep := float64(e) / float64(c.Race.colonistsPerResourceUnits())
		r = int(math.Sqrt(ep*float64(max(1, c.EnergyTech)))*float64(max(25, c.Hab))*0.1 + 0.999)
	} else {
		n := min(factories, c.OperableFactories(pop))
		r = e/c.Race.colonistsPerResourceUnits() + (c.Race.FactoryOutput*n+9)/10
	}
	return max(1, r)
}

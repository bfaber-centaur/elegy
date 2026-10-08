package terraform

import "github.com/bfaber-centaur/elegy/engine"

// Rules is the engine's Terraformer: production's terraform items use the
// functions of this package (docs/TERRAFORM-STATUS.md, "Turn wiring").
type Rules struct{}

var _ engine.Terraformer = Rules{}

func (Rules) Reach(race engine.Race, levels [engine.NumFields]int) [3]int {
	return Reach(race, levels)
}

func (Rules) Capacity(p engine.Planet, race engine.Race, reach [3]int) int {
	return Capacity(p, race, reach)
}

func (Rules) Improve(p *engine.Planet, race engine.Race, reach [3]int) int {
	return Improve(p, race, reach)
}

func (Rules) UnitCost(race engine.Race) int { return UnitCost(race) }

func (Rules) AutoUnits(count, capacity int, minOnly bool, popChange, hab int) int {
	return AutoUnits(count, capacity, minOnly, popChange, hab)
}

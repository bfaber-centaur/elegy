package ai

import (
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/terraform"
)

// noExtra is a Rand whose draw never adds the random extra kT of mining.
type noExtra struct{}

func (noExtra) Intn(n int) int { return n - 1 }

// funds is what a planet has to spend this year: minerals and resources.
type funds struct {
	Minerals  engine.Minerals
	Resources int
}

// available is AI.md §7's "available": the surface minerals plus this
// year's expected mining (ESTIMATES.md "Mining rate", CONFIRMED ES-001:
// the year's mining without the random extra kT), and the resources left
// after the research budget.
func (v *View) available(p *engine.Planet, budget int) funds {
	col := engine.NewColony(p, &v.Self)
	eff := v.Self.Race.MineOutput
	if v.Self.Race.PRT == engine.PRTAlternateReality {
		eff = 10
	}
	working := col.WorkingMines(p.Population, p.Mines)
	f := funds{Minerals: p.Surface}
	for m := range engine.NumMinerals {
		gain, _ := engine.MineYear(p.Deposits[m], working, eff, p.Homeworld, noExtra{})
		f.Minerals[m] += gain
	}
	r := col.Resources(p.Population, p.Factories)
	f.Resources = r - r*budget/100
	return f
}

// queueCost is the full cost of a queue: every item's unit cost times its
// count, less what is already spent on the first unit.
//
// ASSUMPTION A12: auto items count at their count, as plain items do; a
// starbase item counts its full build cost. AI.md §7 says only "the full
// cost of its current queue".
func (v *View) queueCost(q []engine.QueueItem) funds {
	var f funds
	for _, it := range q {
		c := v.itemCost(it)
		spent := func(x int) int { return x * it.Percent / 100 }
		f.Resources += c.Resources*it.Count - spent(c.Resources)
		for m := range engine.NumMinerals {
			f.Minerals[m] += c.Minerals[m]*it.Count - spent(c.Minerals[m])
		}
	}
	return f
}

// itemCost is one unit of a queue item for the player.
func (v *View) itemCost(it engine.QueueItem) engine.Cost {
	race, levels := v.Self.Race, v.Self.Research.Levels
	switch it.Kind {
	case engine.ItemShip:
		if d, ok := v.ship(it.Slot); ok {
			return designCost(d.Design, race, levels)
		}
		return engine.Cost{}
	case engine.ItemStarbase:
		for _, d := range v.Starbases {
			if d.Slot == it.Slot {
				return engine.StarbaseBuildCost(d.Design, race, levels)
			}
		}
		return engine.Cost{}
	case engine.ItemTerraform, engine.ItemAutoMinTerraform, engine.ItemAutoMaxTerraform:
		return engine.Cost{Resources: terraform.UnitCost(race)}
	}
	return engine.ItemCost(race, it.Kind)
}

// designCost is a ship design's cost for the player (COMPONENTS.md "Cost
// for an owner", CONFIRMED CS-001): the hull's and each part's owner cost
// from the component table.
func designCost(d engine.Design, race engine.Race, levels [engine.NumFields]int) engine.Cost {
	cat := engine.Components()
	var c engine.Cost
	add := func(name string, n int) {
		comp, ok := cat.Lookup(name)
		if !ok {
			return
		}
		pc := comp.OwnerCost(race, levels)
		c.Resources += n * pc.Resources
		for m := range engine.NumMinerals {
			c.Minerals[m] += n * pc.Minerals[m]
		}
	}
	add(d.Hull.Name, 1)
	for _, s := range d.Slots {
		add(s.Part.Name, s.Count)
	}
	return c
}

// sub is a − b.
func (a funds) sub(b funds) funds {
	a.Resources -= b.Resources
	for m := range engine.NumMinerals {
		a.Minerals[m] -= b.Minerals[m]
	}
	return a
}

// short reports whether some mineral of a is below b's.
func (a funds) short(b funds) bool {
	for m := range engine.NumMinerals {
		if a.Minerals[m] < b.Minerals[m] {
			return true
		}
	}
	return false
}

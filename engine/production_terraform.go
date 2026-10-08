package engine

// Terraform Environment (planetary item 12) and Auto Min / Auto Max
// Terraform go through Game.Terraform (terraformer.go).

// terraform reports whether k builds Terraform Environment units.
func (k ItemKind) terraform() bool { return k.real() == ItemTerraform }

// terraformModelled reports whether terraform items can be built:
// production runs for a game with terraforming rules.
func (pr *production) terraformModelled() bool {
	return pr.g != nil && pr.g.Terraform != nil
}

// reach is the owner's terraforming reach at the tech levels before this
// year's research (KERNEL.md "Terraforming", tech used, CONFIRMED KX-005).
func (pr *production) reach() [3]int {
	return pr.g.Terraform.Reach(pr.in.Colony.Race, pr.g.Players[pr.planet.Owner].Research.Levels)
}

// terraformCapacity is the improving clicks still available on the planet
// for its owner.
func (pr *production) terraformCapacity() int {
	return pr.g.Terraform.Capacity(*pr.planet, pr.in.Colony.Race, pr.reach())
}

// autoTerraformUnits is how many units an Auto Min or Auto Max Terraform
// item with count builds this year (KERNEL.md "Terraforming", CONFIRMED
// KX-005). The population change is this year's growth: production runs
// before the year's population is written (engine/turn.go), so the
// planet still holds last year's population.
func (pr *production) autoTerraformUnits(k ItemKind, count int) int {
	p := pr.planet
	return pr.g.Terraform.AutoUnits(count, pr.terraformCapacity(), k == ItemAutoMinTerraform,
		pr.in.GrownPop-p.Population, Habitability(pr.in.Colony.Race, p.Env))
}

// terraformUnits applies n completed Terraform Environment units, one
// click each.
func (pr *production) terraformUnits(n int) {
	reach := pr.reach()
	for range n {
		pr.g.Terraform.Improve(pr.planet, pr.in.Colony.Race, reach)
	}
	pr.event(EventBuilt, ItemTerraform, n)
}

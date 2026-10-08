package engine

// Terraformer is the terraforming rules production uses for Terraform
// Environment (planetary item 12) and Auto Min / Auto Max Terraform
// (KERNEL.md "Terraforming"). Package terraform implements it
// (terraform.Rules); the engine cannot import that package. Each method is
// the terraform function of the same name (docs/TERRAFORM-STATUS.md).
type Terraformer interface {
	// Reach is how far race at these tech levels terraforms each axis.
	Reach(race Race, levels [NumFields]int) [3]int
	// Capacity is the improving clicks still available on p.
	Capacity(p Planet, race Race, reach [3]int) int
	// Improve makes one improving click on p and returns the axis moved,
	// or -1.
	Improve(p *Planet, race Race, reach [3]int) int
	// UnitCost is a Terraform Environment unit's resource cost.
	UnitCost(race Race) int
	// AutoUnits is how many units an auto item builds this year.
	AutoUnits(count, capacity int, minOnly bool, popChange, hab int) int
}

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

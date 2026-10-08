package engine

// TaskRemoteMine is the "remote mining" waypoint task (KERNEL.md "Remote
// mining", TAKEOVER.md "Other waypoint tasks").
const TaskRemoteMine TaskKind = TaskRoute + 4

// Remote terraforming and packet disclosure messages (MESSAGES.md 0x12c,
// 0x15a; OBJECTS.md "Impact" step 4). Elegy's wording.
const (
	// EventPlanetImproved: Fleet's Orbital Adjusters improved Planet.
	// Player = told (the fleet owner, and the planet owner when it is
	// someone else and the planet owner's value changed), Count = the
	// planet owner's habitability after.
	EventPlanetImproved EventKind = iota + EventRaceHacked + 1
	// EventPlanetDegraded: as EventPlanetImproved, by a fleet that is not
	// the planet owner's friend.
	EventPlanetDegraded
	// EventPacketDesignSeen: Player's packet hit Planet, whose starbase
	// design (Count, a design index) Player now knows in full.
	EventPacketDesignSeen
)

// Terraformer is the terraforming, remote-mining and Orbital Adjuster
// rules, which the terraform package implements (it imports this package,
// so the engine reaches it through this interface).
type Terraformer interface {
	// MiningRate is fleet f's remote-mining rate (KERNEL.md "Remote
	// mining").
	MiningRate(g *Game, f *Fleet) int
	// RemoteMine runs one year of remote mining by fleet fi at the planet
	// it orbits (step 6c.2).
	RemoteMine(g *Game, fi int, rng Rand) (Minerals, bool)
	// Adjust runs the Orbital Adjusters (step 7.4) and returns their
	// messages.
	Adjust(g *Game) []Event

	// Production's terraform items (KERNEL.md "Terraforming";
	// production_terraform.go), each the terraform function of the same
	// name (docs/TERRAFORM-STATUS.md).

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

// defaultTask is a new fleet's task at its build planet (PRODUCTION-LAUNCH.md
// "Default task", CONFIRMED SL-11): remote mining for an Alternate Reality
// owner's fleet that can mine, else none.
func (g *Game) defaultTask(f *Fleet) Task {
	if g.Terraform != nil && g.Players[f.Owner].Race.PRT == PRTAlternateReality && g.Terraform.MiningRate(g, f) > 0 {
		return Task{Kind: TaskRemoteMine}
	}
	return Task{}
}

package engine

// Ship parts and hulls. A design carries the values of its hull and
// parts; catalog.go builds them from the component table.

// PartKind is a part's category (COMPONENTS.md "category"). PartOther is
// a part without a catalogue category, as tests build them.
type PartKind int

const (
	PartOther PartKind = iota
	PartBeam
	PartTorpedo
	PartBomb
	PartArmor
	PartShield
	PartElectrical
	PartMechanical
	PartEngine
	PartStargate
	PartScanner
	PartMiningRobot
	PartMineLayer
	PartMassDriver
	// PartPlanetary (planetary scanners, defenses, Genesis Device) and
	// PartTerraform are not ship parts; they appear only for their cost.
	PartPlanetary
	PartTerraform
)

// Part is one ship part. Fields that do not apply are zero.
type Part struct {
	Name string
	Kind PartKind
	Mass int

	// Weapons (PartBeam, PartTorpedo).
	Damage   int
	Range    int
	Accuracy int // torpedoes, percent
	Gatling  bool
	Sapper   bool
	Missile  bool // Jihad, Juggernaut, Doomsday, Armageddon

	// Initiative is a weapon's own initiative, or for other parts the
	// initiative they add to the ship (Battle Computer 1, Battle Super
	// Computer 2, Battle Nexus 3).
	Initiative int

	Armor  int // includes Croby Sharmor and Langston Shell (65), Multi Cargo Pod (50)
	Shield int // includes Fielded Kelarium (50) and Mega Poly Shell (100)

	Computer  int  // percent: Battle Computer 20, Super 30, Nexus 50, Multi Contained Munition 10
	Jammer    int  // factor f: Jammer 20 → 80, Langston Shell 95, ...
	Capacitor int  // percent: Energy Capacitor 10, Flux Capacitor 20
	Deflector bool // Beam Deflector
	Dampener  bool // Energy Dampener

	// Thrust is the battle speed bonus: 1 per Maneuvering Jet and Multi
	// Function Pod, 2 per Overthruster. HalfThrust parts (Alien Miner)
	// count with Enigma Pulsars as (n + 1)/2.
	Thrust     int
	HalfThrust bool

	// Scanning (SCANNING.md). Scanner marks a part that scans (a scanner
	// part, or a part with a built-in scanner), with its normal and
	// penetrating ranges in ly. CargoScan parts (Pick Pocket, Robber
	// Baron) see the cargo of an enemy fleet at their exact position;
	// DetailedPlanetScan (Robber Baron) gives a detailed report of an
	// orbited planet.
	Scanner            bool
	ScanRange          int
	PenRange           int
	CargoScan          bool
	DetailedPlanetScan bool
	CloakPoints        int
	Tachyon            bool // Tachyon Detector

	// FuelCapacity (mg) and CargoCapacity (kT) the part adds to a ship:
	// fuel tanks, Anti-matter Generator, cargo pods.
	FuelCapacity  int
	CargoCapacity int

	Cost    Cost
	TechReq [NumFields]int
}

func (p Part) weapon() bool { return p.Kind == PartBeam || p.Kind == PartTorpedo }

// Hull is a ship or starbase hull.
type Hull struct {
	Name       string
	Mass       int
	Armor      int
	Initiative int
	Starbase   bool
	// FuelTransport marks the Fuel Transport and Super-Fuel Xport hulls,
	// whose ships are fuel-transport targets and add to repair.
	FuelTransport bool
	// RepairBonus is f in the repair rule: 25 for a Fuel Transport, 50
	// for a Super-Fuel Xport.
	RepairBonus int
	// JOATScanner marks the Scout, Frigate and Destroyer hulls, which
	// have a built-in scanner for a Jack of All Trades owner.
	JOATScanner bool
	Cost        Cost
	TechReq     [NumFields]int

	// From the component table (COMPONENTS.md "Hull", "Starbase hull").
	FuelCapacity  int // mg; ship hulls
	CargoCapacity int // kT; ship hulls
	Slots         []HullSlot
	// Dock is a starbase's dock capacity in kT of hull mass: 0 without a
	// dock, DockUnlimited for no limit.
	Dock int
	// StarbaseNumber is Planet.StarbaseHull for a starbase hull: its
	// catalogue index + 1.
	StarbaseNumber int
}

// DockUnlimited is Hull.Dock for a starbase whose dock has no limit.
const DockUnlimited = -1

// HullSlot is one slot of a hull: the part kinds it accepts and the most
// parts it holds. A ship hull's first slot is its engine slot, which must
// be filled to Max.
type HullSlot struct {
	Kinds []PartKind
	Max   int
}

// designCost is a design's per-ship cost for an owner with race and
// tech levels (COMBAT.md "Design cost", which points to stars-elegy
// COMPONENTS.md "Cost for an owner", CONFIRMED CS-001): the hull plus each
// slot's count × part cost, each adjusted for miniaturization, race and
// Bleeding Edge Technology.
// A starbase token's cost for target choice is this plain owner cost; the
// starbase build-cost reduction applies only to production (COMBAT.md
// "Target choice", BINARY-ONLY).
func designCost(d Design, race Race, levels [NumFields]int) Cost {
	c := itemCost(d.Hull.Cost, d.Hull.TechReq, PartOther, race, levels)
	for _, s := range d.Slots {
		pc := itemCost(s.Part.Cost, s.Part.TechReq, s.Part.Kind, race, levels)
		c.Resources += s.Count * pc.Resources
		for m := range NumMinerals {
			c.Minerals[m] += s.Count * pc.Minerals[m]
		}
	}
	return c
}

// StarbaseBuildCost is what production charges for a starbase design
// (COMPONENTS.md "Starbases", CONFIRMED CS-001): the owner cost, then with
// Improved Starbases or Alternate Reality each component c − c/5, then
// halved rounding up.
func StarbaseBuildCost(d Design, race Race, levels [NumFields]int) Cost {
	c := designCost(d, race, levels)
	isb := race.LRT.ImprovedStarbases || race.PRT == PRTAlternateReality
	f := func(v int) int {
		if isb {
			v -= v / 5
		}
		return (v + 1) / 2
	}
	c.Resources = f(c.Resources)
	for m := range NumMinerals {
		c.Minerals[m] = f(c.Minerals[m])
	}
	return c
}

// itemCost is one hull's, part's or planetary item's cost for an owner
// (COMPONENTS.md "Cost for an owner", CONFIRMED CS-001). Terraform and
// planetary items skip miniaturization and are never doubled by Bleeding
// Edge Technology.
func itemCost(base Cost, req [NumFields]int, kind PartKind, race Race, levels [NumFields]int) Cost {
	exempt := kind == PartTerraform || kind == PartPlanetary
	m, hasReq := 0, false
	for f := range NumFields {
		if req[f] > 0 {
			if v := levels[f] - req[f]; !hasReq || v < m {
				m = v
			}
			hasReq = true
		}
	}
	if !hasReq {
		m = levels[0]
		for _, l := range levels {
			m = min(m, l)
		}
	}
	bet := race.LRT.BleedingEdgeTech
	comps := []*int{&base.Resources, &base.Minerals[Ironium], &base.Minerals[Boranium], &base.Minerals[Germanium]}
	adjust := func(fn func(int) int) {
		for _, c := range comps {
			if *c != 0 {
				*c = fn(*c)
			}
		}
	}
	if m > 0 && !exempt {
		d := min(75, 4*min(m, 19))
		if bet {
			d = min(80, 5*min(m, 19))
		}
		adjust(func(c int) int { return max(1, c-(c*d+50)/100) })
	}
	weapon := kind == PartBeam || kind == PartTorpedo || kind == PartBomb
	switch {
	case race.PRT == PRTInterstellarTraveler && kind == PartStargate,
		race.PRT == PRTWarMonger && weapon:
		adjust(func(c int) int { return c - c/4 })
	case race.PRT == PRTInnerStrength && weapon:
		adjust(func(c int) int { return c + c/4 })
	case race.PRT == PRTClaimAdjuster && kind == PartTerraform:
		base.Resources /= 2
	case race.LRT.CheapEngines && kind == PartEngine:
		adjust(func(c int) int { return c - c/2 })
	}
	if bet && m <= 0 && hasReq && !exempt {
		adjust(func(c int) int { return 2 * c })
	}
	return base
}

// Slot is a number of one part in one hull slot.
type Slot struct {
	Part  Part
	Count int
}

// techReq is the highest requirement of the hull and every part.
func (d Design) techReq() [NumFields]int {
	r := d.Hull.TechReq
	for _, s := range d.Slots {
		for f := range NumFields {
			r[f] = max(r[f], s.Part.TechReq[f])
		}
	}
	return r
}

func (d Design) armed() bool {
	for _, s := range d.Slots {
		if s.Count > 0 && s.Part.weapon() {
			return true
		}
	}
	return false
}

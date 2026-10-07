package engine

// Ship parts and hulls as battles need them. Elegy has no built-in part
// catalogue yet: a design carries the values of its hull and parts.

// PartKind is a part's category as far as battle rules distinguish them.
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
	Cost        Cost
	TechReq     [NumFields]int
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

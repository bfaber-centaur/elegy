package ai

import "github.com/bfaber-centaur/elegy/engine"

// partClasses are the AI part classes (AI.md §5 "AI part classes"): each a
// list of parts in order of preference, named as in COMPONENTS.md. The
// table is transcribed from AI.md; the starbase classes are CONFIRMED with
// the starbase designs (AI-2), the ship-design classes with Robotoid's and
// Cybertron's designs (AI-8, AI-19).
var partClasses = [45][]string{
	0:  {"Anti Matter Torpedo", "Omega Torpedo", "Upsilon Torpedo", "Rho Torpedo", "Epsilon Torpedo", "Delta Torpedo", "Beta Torpedo", "Alpha Torpedo"},
	1:  {"Armageddon Missile", "Doomsday Missile", "Juggernaut Missile", "Jihad Missile"},
	2:  {"Multi Contained Munition", "Mega Disruptor", "Heavy Blaster", "Colloidal Phaser"},
	3:  {"Anti-Matter Pulverizer", "Disruptor", "Mark IV Blaster", "Phaser Bazooka"},
	4:  {"Streaming Pulverizer", "Myopic Disruptor", "Mini Blaster", "Yakimora Light Phaser", "X-Ray Laser", "Laser"},
	5:  {"Blunderbuss", "Bludgeon", "Blackjack"},
	6:  {"Big Mutha Cannon", "Gatling Neutrino Cannon", "Gatling Gun", "Mini Gun"},
	7:  {"Syncro Sapper", "Phased Sapper", "Pulsed Sapper"},
	8:  {"Galaxy Scoop", "Enigma Pulsar", "Trans-Galactic Mizer Scoop", "Trans-Galactic Super Scoop", "Trans-Galactic Fuel Scoop", "Sub-Galactic Fuel Scoop", "Radiating Hydro-Ram Scoop", "Fuel Mizer"},
	9:  {"Superlatanium", "Mega Poly Shell", "Valanium", "Depleted Neutronium", "Neutronium", "Fielded Kelarium", "Kelarium", "Organic Armor", "Strobnium", "Carbonic Armor", "Crobmnium", "Tritanium"},
	10: {"Complete Phase Shield", "Elephant Hide Fortress", "Langston Shell", "Gorilla Delagator", "Croby Sharmor", "Shadow Shield", "Bear Neutrino Barrier", "Wolverine Diffuse Shield", "Cow-hide Shield", "Mole-skin Shield"},
	11: {"Battle Nexus", "Battle Super Computer", "Battle Computer"},
	12: {"Multi Function Pod", "Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Beam Deflector", "Overthruster", "Maneuvering Jet"},
	13: {"Multi Function Pod", "Overthruster", "Maneuvering Jet", "Ultra-Stealth Cloak", "Beam Deflector", "Super-Stealth Cloak", "Stealth Cloak"},
	14: {"Flux Capacitor", "Energy Capacitor", "Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Super Fuel Tank", "Fuel Tank"},
	15: {"Beam Deflector", "Flux Capacitor", "Energy Capacitor", "Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Super Fuel Tank", "Fuel Tank"},
	16: {"Multi Cargo Pod", "Super Cargo Pod", "Cargo Pod"},
	17: {"Mega Poly Shell", "Superlatanium", "Valanium", "Mega Poly Shell", "Neutronium", "Depleted Neutronium", "Fielded Kelarium", "Kelarium", "Organic Armor", "Strobnium", "Carbonic Armor", "Crobmnium", "Tritanium"},
	18: {"Overthruster", "Maneuvering Jet", "Beam Deflector", "Super Fuel Tank", "Fuel Tank"},
	19: {"Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Battle Nexus", "Battle Super Computer", "Battle Computer"},
	20: {"Flux Capacitor", "Energy Capacitor", "Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Multi Function Pod", "Ultra-Stealth Cloak", "Super-Stealth Cloak", "Stealth Cloak", "Battle Nexus", "Battle Super Computer", "Battle Computer"},
	21: {"Hush-a-Boom", "Cherry Bomb", "M-80 Bomb", "M-70 Bomb", "Black Cat Bomb", "Lady Finger Bomb"},
	22: {"Hush-a-Boom", "Retro Bomb", "Annihilator Bomb", "Peerless Bomb", "Enriched Neutron Bomb", "Neutron Bomb", "Smart Bomb"},
	23: {"Hush-a-Boom", "Annihilator Bomb", "Peerless Bomb", "Enriched Neutron Bomb", "Neutron Bomb", "Smart Bomb", "Cherry Bomb", "M-80 Bomb", "M-70 Bomb", "Black Cat Bomb", "Lady Finger Bomb", "Retro Bomb"},
	24: {"Galaxy Scoop", "Radiating Hydro-Ram Scoop"},
	25: {"Mine Dispenser 130", "Mine Dispenser 80", "Mine Dispenser 50", "Mine Dispenser 40"},
	26: {"Elephant Scanner", "Robber Baron Scanner", "Dolphin Scanner", "Chameleon Scanner", "Ferret Scanner", "Gazelle Scanner", "Possum Scanner"},
	27: {"Robber Baron Scanner", "Pick Pocket Scanner", "Chameleon Scanner", "Pick Pocket Scanner", "Possum Scanner", "DNA Scanner", "Mole Scanner", "Rhino Scanner", "Bat Scanner"},
	28: {"Alien Miner", "Robo-Ultra-Miner", "Robo-Super-Miner", "Robo-Maxi-Miner", "Robo-Miner", "Robo-Mini-Miner", "Robo-Midget Miner"},
	29: {"Orbital Adjuster"},
	30: {"Trans-Star 10", "Enigma Pulsar", "Radiating Hydro-Ram Scoop", "Trans-Galactic Super Scoop", "Trans-Galactic Fuel Scoop", "Sub-Galactic Fuel Scoop", "Trans-Galactic Drive", "Alpha Drive 8", "Daddy Long Legs 7", "Long Hump 6"},
	31: {"Orbital Construction Module", "Colonization Module"},
	32: {"Speed Trap 50", "Speed Trap 30", "Speed Trap 20"},
	33: {"Multi Contained Munition"},
	34: {"Ultra Driver 13", "Ultra Driver 12", "Ultra Driver 11", "Ultra Driver 10", "Super Driver 9", "Super Driver 8", "Mass Driver 7", "Mass Driver 6", "Mass Driver 5", "Multi Function Pod", "Ultra-Stealth Cloak", "Super-Stealth Cloak", "Stealth Cloak", "Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Battle Nexus", "Battle Super Computer", "Battle Computer"},
	35: {"Armageddon Missile", "Omega Torpedo", "Doomsday Missile", "Upsilon Torpedo", "Rho Torpedo", "Epsilon Torpedo", "Delta Torpedo", "Beta Torpedo", "Alpha Torpedo"},
	36: {"Anti-Matter Pulverizer", "Streaming Pulverizer", "Disruptor", "Myopic Disruptor", "Mark IV Blaster", "Mini Blaster", "Phaser Bazooka", "Yakimora Light Phaser", "X-Ray Laser", "Laser"},
	37: {"Langston Shell", "Complete Phase Shield", "Elephant Hide Fortress", "Gorilla Delagator", "Langston Shell", "Bear Neutrino Barrier", "Shadow Shield", "Croby Sharmor", "Wolverine Diffuse Shield", "Cow-hide Shield", "Mole-skin Shield"},
	38: {"Mega Disruptor", "Heavy Blaster", "Colloidal Phaser", "Phaser Bazooka", "Laser"},
	39: {"Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Multi Function Pod", "Ultra-Stealth Cloak", "Super-Stealth Cloak", "Stealth Cloak"},
	40: {"Orbital Construction Module"},
	41: {"Mega Poly Shell", "Jammer 50", "Jammer 30", "Jammer 20", "Jammer 10", "Overthruster", "Maneuvering Jet", "Beam Deflector", "Super Fuel Tank", "Fuel Tank"},
	42: {"Alien Miner", "Robo-Ultra-Miner", "Robo-Super-Miner", "Robo-Maxi-Miner"},
	43: {"Alien Miner", "Robo-Ultra-Miner", "Robo-Midget Miner"},
	44: {"Galaxy Scoop", "Trans-Galactic Mizer Scoop", "Trans-Galactic Super Scoop", "Trans-Galactic Fuel Scoop", "Sub-Galactic Fuel Scoop", "Fuel Mizer"},
}

// classPart is the first part of class k the race can build now (AI.md
// §5: "the first part in the class's list that the race can build now";
// COMPONENTS.md "Who can build what"). The computer players own no
// Mystery Trader parts in Elegy. ok is false when none can be built.
func classPart(cat *engine.Catalog, k int, race engine.Race, levels [engine.NumFields]int) (engine.Component, bool) {
	for _, name := range partClasses[k] {
		comp, found := cat.Lookup(name)
		if !found {
			continue
		}
		if ok, err := comp.Buildable(race, levels, false); ok && err == nil {
			return comp, true
		}
	}
	return engine.Component{}, false
}

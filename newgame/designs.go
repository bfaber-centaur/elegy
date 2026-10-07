package newgame

import (
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// shipSpec is a starting design before part upgrades: a hull and the
// parts in each slot. The hulls, parts and counts are UNIVERSE.md
// "Starting ships" (CONFIRMED). ELEGY CHOICE: which hull slot holds each
// part when a hull has several slots that accept it; the spec names parts,
// not slots. Design names are also Elegy's (the spec names the hulls and
// parts only).
type shipSpec struct {
	name  string
	hull  string
	fills []engine.SlotFill
}

const (
	quickJump = "Quick Jump 5"
	bat       = "Bat Scanner"
)

func fill(slot int, part string, count int) engine.SlotFill {
	return engine.SlotFill{Slot: slot, Part: part, Count: count}
}

func scout(name, item string) shipSpec {
	return shipSpec{name, "Scout", []engine.SlotFill{fill(0, quickJump, 1), fill(1, bat, 1), fill(2, item, 1)}}
}

var (
	scoutFuel    = scout("Scout", "Fuel Tank")
	scoutXRay    = scout("Armed Scout", "X-Ray Laser")
	scoutCloak   = scout("Cloaked Scout", "Stealth Cloak")
	smallFreight = shipSpec{"Cloaked Freighter", "Small Freighter", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Transport Cloaking", 1), fill(2, "Mole-skin Shield", 1)}}
	mediumFreight = shipSpec{"Medium Freighter", "Medium Freighter", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, bat, 1), fill(2, "Tritanium", 1)}}
	destroyer = shipSpec{"Destroyer", "Destroyer", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Laser", 1), fill(2, "Alpha Torpedo", 1), fill(3, bat, 1),
		fill(4, "Tritanium", 2), fill(5, "Fuel Tank", 1), fill(6, "Battle Computer", 1)}}
	privateer = shipSpec{"Privateer", "Privateer", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Crobmnium", 2), fill(2, bat, 1), fill(3, "Laser", 1), fill(4, "Alpha Torpedo", 1)}}
	colonyShip = shipSpec{"Colony Ship", "Colony Ship", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Colonization Module", 1)}}
	colonyShipAR = shipSpec{"Colony Ship", "Colony Ship", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Orbital Construction Module", 1)}}
	miniColony = shipSpec{"Mini-Colony Ship", "Mini-Colony Ship", []engine.SlotFill{
		fill(0, "Settler's Delight", 1), fill(1, "Colonization Module", 1)}}
	miniBomber = shipSpec{"Mini Bomber", "Mini Bomber", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Lady Finger Bomb", 2)}}
	miniMiner = shipSpec{"Mini-Miner", "Mini-Miner", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, bat, 1), fill(2, "Robo-Mini-Miner", 1), fill(3, "Robo-Mini-Miner", 1)}}
	miniMinerCA = shipSpec{"Mini-Miner", "Mini-Miner", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, bat, 1), fill(2, "Orbital Adjuster", 1), fill(3, "Orbital Adjuster", 1)}}
	midgetMiner = shipSpec{"Midget Miner", "Midget Miner", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Robo-Midget Miner", 2)}}
	mineLayer = shipSpec{"Mine Layer", "Mini Mine Layer", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Mine Dispenser 40", 2), fill(2, "Mine Dispenser 40", 2), fill(3, bat, 1)}}
	speedTrapLayer = shipSpec{"Speed Trap Layer", "Mini Mine Layer", []engine.SlotFill{
		fill(0, quickJump, 1), fill(1, "Speed Trap 20", 2), fill(2, "Speed Trap 20", 2), fill(3, bat, 1)}}
)

// upgrades maps a basic part to the parts that replace it, first
// available first (UNIVERSE.md "Part upgrades", CONFIRMED for the cases
// run). A part with no available replacement stays.
var upgrades = map[string][]string{
	quickJump:           {"Radiating Hydro-Ram Scoop", "Alpha Drive 8", "Daddy Long Legs 7", "Fuel Mizer", "Long Hump 6"},
	bat:                 {"Possum Scanner", "Mole Scanner", "Rhino Scanner"},
	"Rhino Scanner":     {"Possum Scanner", "Mole Scanner", "Rhino Scanner"},
	"Tritanium":         {"Carbonic Armor", "Crobmnium"},
	"Crobmnium":         {"Carbonic Armor", "Crobmnium"},
	"Mole-skin Shield":  {"Wolverine Diffuse Shield", "Cow-hide Shield"},
	"Cow-hide Shield":   {"Wolverine Diffuse Shield", "Cow-hide Shield"},
	"Laser":             {"Yakimora Light Phaser", "X-Ray Laser"},
	"X-Ray Laser":       {"Yakimora Light Phaser", "X-Ray Laser"},
	"Alpha Torpedo":     {"Beta Torpedo"},
	"Lady Finger Bomb":  {"Black Cat Bomb"},
	"Robo-Midget Miner": {"Robo-Miner", "Robo-Midget Miner"},
	"Robo-Mini-Miner":   {"Robo-Miner", "Robo-Midget Miner"},
}

const ramScoop = "Radiating Hydro-Ram Scoop"

// upgrade returns the part a starting design carries in place of part.
func (g *generator) upgrade(part, hull string, race engine.Race, levels [engine.NumFields]int) (string, error) {
	for _, cand := range upgrades[part] {
		// The ram scoop is skipped on a Colony Ship when the race is
		// not radiation-immune and its radiation centre is 84 or less.
		rad := race.Env[engine.Radiation]
		if cand == ramScoop && hull == "Colony Ship" && !rad.Immune && rad.Center <= 84 {
			continue
		}
		c, ok := g.cat.Lookup(cand)
		if !ok {
			return "", fmt.Errorf("newgame: no component %q", cand)
		}
		ok, err := c.Buildable(race, levels, false)
		if err != nil {
			return "", err
		}
		if ok {
			return cand, nil
		}
	}
	return part, nil
}

// newDesign builds a design and appends it to Game.Designs.
func (g *generator) newDesign(s shipSpec, upgrade bool, race engine.Race, levels [engine.NumFields]int) (int, error) {
	fills := append([]engine.SlotFill(nil), s.fills...)
	if upgrade {
		for i := range fills {
			p, err := g.upgrade(fills[i].Part, s.hull, race, levels)
			if err != nil {
				return 0, err
			}
			fills[i].Part = p
		}
	}
	d, err := g.cat.NewDesign(s.name, s.hull, fills)
	if err != nil {
		return 0, err
	}
	g.res.Game.Designs = append(g.res.Game.Designs, d)
	return len(g.res.Game.Designs) - 1, nil
}

// Starbase designs (UNIVERSE.md "Starbases"). Starting starbase designs
// are not upgraded.
//
// PLACEHOLDER: the spec says design 0 is "a Space Station armed with
// lasers and Mole-skin shields" without counts or slots. Elegy fills both
// Space Station beam slots 1 and 3 with 16 Lasers and shield slot 2 with
// 16 Mole-skin Shields until the spec gives the loadout.
func spaceStation(name, orbital string) shipSpec {
	fills := []engine.SlotFill{fill(1, "Laser", 16), fill(2, "Mole-skin Shield", 16), fill(3, "Laser", 16)}
	if orbital != "" {
		fills = append([]engine.SlotFill{fill(0, orbital, 1)}, fills...)
	}
	return shipSpec{name, "Space Station", fills}
}

func orbitalFort(name, orbital string) shipSpec {
	var fills []engine.SlotFill
	if orbital != "" {
		fills = append(fills, fill(0, orbital, 1))
	}
	return shipSpec{name, "Orbital Fort", fills}
}

// starbaseSpecs are a player's starbase designs 0 and (when it has one)
// 1, and which of them orbits the homeworld.
func starbaseSpecs(prt engine.PRT, tiny bool) (specs []shipSpec, homeworld int) {
	switch prt {
	case engine.PRTAlternateReality:
		// Design 0 an empty Orbital Fort; design 1 a Space Station at
		// the homeworld. ELEGY CHOICE: design 1 has design 0's usual
		// loadout.
		return []shipSpec{orbitalFort("Orbital Fort", ""), spaceStation("Starbase", "")}, 1
	case engine.PRTPacketPhysics:
		specs = []shipSpec{spaceStation("Starbase", "Mass Driver 5")}
		if !tiny {
			specs = append(specs, orbitalFort("Accelerator Platform", "Mass Driver 5"))
		}
	case engine.PRTInterstellarTraveler:
		specs = []shipSpec{spaceStation("Starbase", "Stargate 100/250")}
		if !tiny {
			specs = append(specs, orbitalFort("Gate Platform", "Stargate 100/250"))
		}
	default:
		specs = []shipSpec{spaceStation("Starbase", "")}
	}
	return specs, 0
}

// startingShips is a player's starting ship list, one entry per ship in
// fleet order (UNIVERSE.md "Starting ships", CONFIRMED). An entry with
// atSecond is the second planet's Scout of ship design 0.
type shipEntry struct {
	spec     *shipSpec
	atSecond bool
}

func startingShips(ps PlayerSetup, levels [engine.NumFields]int, secondPlanet bool) ([]shipEntry, error) {
	one := func(s *shipSpec) shipEntry { return shipEntry{spec: s} }
	var list []shipEntry
	second := func() {
		if secondPlanet {
			list = append(list, shipEntry{atSecond: true})
		}
	}
	switch ps.Race.PRT {
	case engine.PRTHyperExpansion:
		list = []shipEntry{one(&scoutFuel), one(&miniColony), one(&miniColony), one(&miniColony)}
	case engine.PRTSuperStealth:
		if levels[engine.Energy] >= 2 {
			list = append(list, one(&scoutCloak))
		} else {
			list = append(list, one(&scoutFuel))
		}
		if !ps.Computer {
			list = append(list, one(&smallFreight))
		}
		list = append(list, one(&colonyShip))
	case engine.PRTWarMonger:
		list = append(list, one(&scoutXRay))
		if levels[engine.Construction] >= 3 {
			list = append(list, one(&destroyer), one(&miniBomber))
		}
		list = append(list, one(&colonyShip))
	case engine.PRTClaimAdjuster:
		list = []shipEntry{one(&scoutFuel), one(&colonyShip), one(&miniMinerCA)}
	case engine.PRTInnerStrength:
		list = []shipEntry{one(&scoutFuel), one(&colonyShip)}
	case engine.PRTSpaceDemolition:
		list = []shipEntry{one(&scoutFuel), one(&colonyShip), one(&mineLayer), one(&speedTrapLayer)}
	case engine.PRTPacketPhysics:
		list = []shipEntry{one(&scoutFuel), one(&colonyShip)}
		second()
	case engine.PRTInterstellarTraveler:
		list = []shipEntry{one(&scoutFuel), one(&colonyShip), one(&destroyer), one(&privateer)}
		second()
	case engine.PRTAlternateReality:
		list = []shipEntry{one(&scoutFuel), one(&colonyShipAR)}
	case engine.PRTJackOfAllTrades:
		list = []shipEntry{one(&scoutXRay), one(&scoutFuel), one(&colonyShip)}
		if levels[engine.Construction] < 4 {
			list = append(list, one(&mediumFreight))
		} else {
			list = append(list, one(&privateer))
		}
		list = append(list, one(&destroyer), one(&miniMiner))
	default:
		return nil, fmt.Errorf("%w: primary racial trait %d is not one of the ten", ErrSettings, ps.Race.PRT)
	}
	// Then a race with ARM and without OBRM gets two Midget Miners of
	// one design. ELEGY CHOICE: two one-ship fleets, like every other
	// starting ship.
	if l := ps.Race.LRT; l.AdvancedRemoteMining && !l.OnlyBasicRemoteMining {
		list = append(list, one(&midgetMiner), one(&midgetMiner))
	}
	return list, nil
}

// addDesignsAndFleets gives player i its starbase designs, homeworld
// starbase, second planet (PP and IT on a map larger than tiny), ship
// designs and fleets.
func (g *generator) addDesignsAndFleets(i int, ps PlayerSetup, start *PlayerStart) error {
	game := &g.res.Game
	pl := &game.Players[i]
	levels := pl.Research.Levels
	tiny := g.s.Size == Tiny

	specs, hwBase := starbaseSpecs(ps.Race.PRT, tiny)
	for _, s := range specs {
		d, err := g.newDesign(s, false, ps.Race, levels)
		if err != nil {
			return err
		}
		start.StarbaseDesigns = append(start.StarbaseDesigns, d)
	}
	hw := &game.Planets[start.Homeworld]
	g.setStarbase(hw, start.StarbaseDesigns[hwBase])

	secondPlanet := !tiny && (ps.Race.PRT == engine.PRTPacketPhysics || ps.Race.PRT == engine.PRTInterstellarTraveler)
	if secondPlanet {
		if err := g.setUpSecondPlanet(i); err != nil {
			return err
		}
	}

	ships, err := startingShips(ps, levels, secondPlanet)
	if err != nil {
		return err
	}
	// Each new design takes the next design slot; a repeated spec
	// reuses its design.
	made := map[*shipSpec]int{}
	for _, e := range ships {
		if e.atSecond {
			g.addFleet(i, start.ShipDesigns[0], game.Planets[start.SecondPlanet].Pos, start)
			continue
		}
		d, ok := made[e.spec]
		if !ok {
			d, err = g.newDesign(*e.spec, true, ps.Race, levels)
			if err != nil {
				return err
			}
			made[e.spec] = d
			start.ShipDesigns = append(start.ShipDesigns, d)
		}
		g.addFleet(i, d, hw.Pos, start)
	}
	return nil
}

// addFleet adds a one-ship fleet of design d with full fuel and battle
// plan 0 (UNIVERSE.md "Starting ships", CONFIRMED). ELEGY CHOICE: fleet
// ids are numbered across the whole game in creation order.
func (g *generator) addFleet(owner, d int, pos engine.Point, start *PlayerStart) {
	game := &g.res.Game
	f := engine.Fleet{
		ID:     len(game.Fleets),
		Owner:  owner,
		Pos:    pos,
		Stacks: []engine.Stack{{Design: d, Count: 1}},
		Fuel:   game.Designs[d].FuelCapacity,
		Plan:   0,
	}
	start.Fleets = append(start.Fleets, len(game.Fleets))
	game.Fleets = append(game.Fleets, f)
}

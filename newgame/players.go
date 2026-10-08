package newgame

import (
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/races"
)

// startingTech is energy/weapons/propulsion/construction/electronics/
// biotech by primary racial trait (UNIVERSE.md "Starting tech",
// CONFIRMED). HE and IS start at 0 in every field.
var startingTech = map[engine.PRT][engine.NumFields]int{
	engine.PRTSuperStealth:         {0, 0, 0, 0, 5, 0},
	engine.PRTWarMonger:            {1, 6, 1, 0, 0, 0},
	engine.PRTClaimAdjuster:        {1, 1, 1, 2, 0, 6},
	engine.PRTSpaceDemolition:      {0, 0, 2, 0, 0, 2},
	engine.PRTPacketPhysics:        {4, 0, 0, 0, 0, 0},
	engine.PRTInterstellarTraveler: {0, 0, 5, 5, 0, 0},
	engine.PRTAlternateReality:     {1, 0, 0, 0, 0, 0},
	engine.PRTJackOfAllTrades:      {3, 3, 3, 3, 3, 3},
}

// StartingTech is a player's tech levels at the start (UNIVERSE.md
// "Starting tech", CONFIRMED): the PRT's levels; with "expensive fields
// start at tech 3", every field set to "costs 75% extra" raised to 3
// (JOAT 4); then +1 propulsion each for CE and IFE.
func StartingTech(race engine.Race, expensiveAt3 bool) [engine.NumFields]int {
	t := startingTech[race.PRT]
	if expensiveAt3 {
		floor := 3
		if race.PRT == engine.PRTJackOfAllTrades {
			floor = 4
		}
		for f := range t {
			if race.ResearchCosts[f] == engine.ResearchExpensive {
				t[f] = max(t[f], floor)
			}
		}
	}
	if race.LRT.CheapEngines {
		t[engine.Propulsion]++
	}
	if race.LRT.ImprovedFuelEfficiency {
		t[engine.Propulsion]++
	}
	return t
}

// resolvePlayers applies RACES.md "At game creation" to every player
// (CONFIRMED RD-4..RD-6): Random races are generated; human races are
// repaired, and an illegal one becomes the default race with a computer
// name; computer races are not checked. L = min(50, points), 50 for
// computer players (UNIVERSE.md "Leftover advantage points"); spends 5
// and 6 act as 0.
func (g *generator) resolvePlayers() {
	g.players = make([]player, len(g.s.Players))
	for i, ps := range g.s.Players {
		d := ps.Race
		if d.Random {
			d = races.Generate(d.Name, g.rng)
			if d.Name == "Random" {
				d.Name = ""
			}
		}
		cr := races.AtCreation(d, ps.Computer)
		if cr.Replaced {
			cr.Race.Name = ""
		}
		p := player{
			Race: cr.Race.Race, Design: cr.Race, Points: cr.Points, Name: cr.Race.Name,
			Computer: ps.Computer, Level: ps.Level,
			Leftover: races.Leftover(cr.Points), Spend: Spend(races.EffectiveSpend(cr.Race.Spend)),
			ExpensiveAt3: cr.Race.ExpensiveAt3,
		}
		if ps.Computer {
			p.Leftover = 50
		}
		if p.Name == "" {
			// PLACEHOLDER: the original picks one of its 24 built-in
			// computer-player names, game data not in the spec.
			p.Name = fmt.Sprintf("Computer %d", i+1)
		}
		g.players[i] = p
	}
}

// homeworldPopulation is the starting homeworld population in units of
// 100 colonists, before a second planet takes its share (UNIVERSE.md
// "Homeworld", "BBS option", "Computer players", CONFIRMED).
//
// The expert +10% applies before the BBS factor, each truncating: the
// UG vectors' two expert players in BBS games (UG03 player 2, PP with LSP
// and growth 19: 175 → 192 → 921, homeworld 736 after the second planet;
// UG21 player 9, CA with LSP and growth 15: 175 → 192 → 768) match this
// order and not the reverse (739, 770). MEASURED from the vectors; the
// spec does not state the order yet.
func homeworldPopulation(p player, bbs bool) int {
	pop := 250
	if p.Race.LRT.LowStartingPopulation {
		pop = 175
	}
	if p.Computer && p.Level == Expert {
		pop += pop / 10
	}
	if bbs {
		k := 1
		if p.Race.PRT == engine.PRTHyperExpansion {
			k = 2
		}
		pop = pop * (p.Race.GrowthRate*k + 5) / 5
	}
	return pop
}

// homeworldEnv is the centre of the race's range on each axis,
// lo + (hi − lo)/2, or 1 + rand(99) on an immune axis (UNIVERSE.md
// "Homeworld", CONFIRMED).
func (g *generator) homeworldEnv(r engine.Race) [3]int {
	var e [3]int
	for axis, er := range r.Env {
		if er.Immune {
			e[axis] = 1 + g.rand(99)
			continue
		}
		e[axis] = er.Low + (er.High-er.Low)/2
	}
	return e
}

// surfaceDraw is a homeworld's starting surface minerals from the
// concentrations c (UNIVERSE.md "Shared starting minerals", CONFIRMED):
// per mineral 10 + rand(10·c), plus 155 + rand(150) when that is below
// 200; BBS adds a quarter.
//
// c is the reference planet's concentration as generated, not raised to
// 30; the floor applies only to the homeworlds' concentrations.
func (g *generator) surfaceDraw(c [engine.NumMinerals]int) engine.Minerals {
	var s engine.Minerals
	for m := range s {
		v := 10 + g.rand(10*max(1, c[m]))
		if v < 200 {
			v += 155 + g.rand(150)
		}
		if g.s.BBS {
			v += v / 4
		}
		s[m] = v
	}
	return s
}

// floorConcentrations raises each concentration to at least 30.
func floorConcentrations(c [engine.NumMinerals]int) [engine.NumMinerals]int {
	for m := range c {
		c[m] = max(c[m], 30)
	}
	return c
}

func concentrationsOf(p *engine.Planet) [engine.NumMinerals]int {
	var c [engine.NumMinerals]int
	for m := range c {
		c[m] = p.Deposits[m].Concentration
	}
	return c
}

// SurfaceSpend applies a surface-minerals spend of L points (UNIVERSE.md
// "Leftover advantage points", CONFIRMED UG16): 10·L kT, the smallest
// mineral (the last of equal smallest) getting L·10/4 plus the remainder,
// then all three L·10/4.
func SurfaceSpend(s engine.Minerals, l int) engine.Minerals {
	total := 10 * l
	q := total / 4
	low := 0
	for m := range s {
		if s[m] <= s[low] {
			low = m
		}
	}
	s[low] += q + (total - 4*q)
	for m := range s {
		s[m] += q
	}
	return s
}

// ConcentrationSpend applies a concentration spend of L points
// (UNIVERSE.md "Leftover advantage points", CONFIRMED UG20): e = L/2 (1
// when L is 1–2); the lowest concentration (the first of equal lowest)
// gains e, then all three gain (e + 1)/2.
func ConcentrationSpend(c [engine.NumMinerals]int, l int) [engine.NumMinerals]int {
	if l <= 0 {
		return c
	}
	e := max(1, l/2)
	low := 0
	for m := range c {
		if c[m] < c[low] {
			low = m
		}
	}
	c[low] += e
	for m := range c {
		c[m] += (e + 1) / 2
	}
	return c
}

// setUpPlayers makes the players and their homeworlds, second planets,
// designs and fleets. hws is each player's homeworld planet index.
func (g *generator) setUpPlayers(hws []int) error {
	game := &g.res.Game
	planets := game.Planets

	// The shared starting minerals: one draw per game from planet 0's
	// concentrations (LEGACY BUG, see Legacy.SharedHomeworldMinerals).
	ref := concentrationsOf(&planets[0])
	shared := g.surfaceDraw(ref)
	sharedConc := floorConcentrations(ref)

	humans := 0
	for _, ps := range g.players {
		if !ps.Computer {
			humans++
		}
	}

	for i, ps := range g.players {
		start := PlayerStart{Name: ps.Name, Race: ps.Design, Points: ps.Points, Homeworld: hws[i], SecondPlanet: -1}
		levels := StartingTech(ps.Race, ps.ExpensiveAt3)
		pl := engine.Player{
			Race: ps.Race,
			// Every player starts at 15% research, current field
			// energy, next field "same field" (UNIVERSE.md "Relations,
			// research and production", MEASURED).
			ResearchBudget: 15,
			Research:       engine.ResearchState{Levels: levels, Current: engine.Energy, Next: engine.NextSameField},
			Plans:          StartingPlans(),
			// The engine's rules that name computer players (a gifted
			// fleet is refused) read this flag.
			Computer: ps.Computer,
		}
		// The engine's Mystery Trader planet trades read the level
		// (engine.Player.Level, OBJECTS.md "Computer players' planets",
		// CONFIRMED O-53); a human player's stays 0.
		if ps.Computer {
			pl.Level = int(ps.Level)
		}
		// Relations (MEASURED): with exactly one human player every
		// player starts as an enemy of every other; with two or more,
		// every player is neutral (the engine's default). PLACEHOLDER:
		// a game with no human player was not run; Elegy leaves it
		// neutral.
		if humans == 1 {
			pl.Relations = make([]engine.Relation, len(g.players))
			for j := range pl.Relations {
				if j != i {
					pl.Relations[j] = engine.RelationEnemy
				}
			}
		}
		game.Players = append(game.Players, pl)
		g.setUpHomeworld(&planets[hws[i]], i, ps, shared, sharedConc)
		g.res.Players = append(g.res.Players, start)
	}

	// Designs, second planets and fleets, in player order, after every
	// homeworld is owned. ELEGY CHOICE: the order players pick their
	// second planets is not specified.
	for i, ps := range g.players {
		if err := g.addDesignsAndFleets(i, ps, &g.res.Players[i]); err != nil {
			return err
		}
	}
	return nil
}

// setUpHomeworld gives player i's homeworld its starting state
// (UNIVERSE.md "Homeworld", CONFIRMED).
func (g *generator) setUpHomeworld(hw *engine.Planet, i int, ps player, shared engine.Minerals, sharedConc [engine.NumMinerals]int) {
	ar := ps.Race.PRT == engine.PRTAlternateReality
	hw.Owner = i
	hw.Homeworld = true
	hw.Env = g.homeworldEnv(ps.Race)
	g.res.Artifact[hw.ID] = false
	hw.Population = homeworldPopulation(ps, g.s.BBS)
	// No planet has a production queue at the start (MEASURED).
	if !ar {
		hw.Mines, hw.Factories, hw.Defenses = 10, 10, 10
		hw.HasScanner = true
	}

	conc, surface := sharedConc, shared
	// Legacy.SharedHomeworldMinerals reproduces the original's LEGACY BUG
	// that every homeworld starts with the same surface minerals (one draw
	// per game) and the concentrations of planet 0 instead of its own
	// (UNIVERSE.md "Shared starting minerals", CONFIRMED UG16–UG21). Off,
	// each homeworld gets its own draw and its own concentrations, with
	// the same floor of 30.
	if !g.s.Rules.Legacy.SharedHomeworldMinerals {
		own := concentrationsOf(hw)
		conc, surface = floorConcentrations(own), g.surfaceDraw(own)
	}

	l := ps.Leftover
	switch ps.Spend {
	case SpendSurfaceMinerals:
		surface = SurfaceSpend(surface, l)
		if ps.Computer && (ps.Level == Harder || ps.Level == Expert) {
			conc = ConcentrationSpend(conc, 50)
		}
	case SpendConcentrations:
		conc = ConcentrationSpend(conc, l)
	// AR's mines, factories or defenses spend adds nothing
	// (CONFIRMED RD-7).
	case SpendMines:
		if !ar {
			hw.Mines += l / 2
		}
	case SpendFactories:
		if !ar {
			hw.Factories += l / 5
		}
	case SpendDefenses:
		if !ar {
			hw.Defenses += (l + 5) / 10
		}
	}
	for m := range engine.NumMinerals {
		hw.Deposits[m] = engine.Deposit{Concentration: conc[m]}
	}
	hw.Surface = surface
}

// secondBand reports whether d² lies in the second-planet band:
// (15W/100)² ≤ d² ≤ (23W/100)², each bound truncated before squaring
// (UNIVERSE.md "Second planet", BINARY-ONLY detail).
func secondBand(dd, w int) bool {
	lo, hi := 15*w/100, 23*w/100
	return dd >= lo*lo && dd <= hi*hi
}

// setUpSecondPlanet gives a PP or IT player its second planet
// (UNIVERSE.md "Second planet: PP and IT", CONFIRMED).
func (g *generator) setUpSecondPlanet(i int) error {
	game := &g.res.Game
	planets := game.Planets
	start := &g.res.Players[i]
	hw := &planets[start.Homeworld]
	race := game.Players[i].Race

	var band []int
	nearest, best := -1, 0
	for j := range planets {
		p := &planets[j]
		if p.Owner != engine.NoOwner {
			continue
		}
		dd := d2(p.Pos, hw.Pos)
		if secondBand(dd, g.w) {
			band = append(band, j)
		} else if nearest < 0 || dd < best {
			// Ties for nearest go to the first in list order.
			nearest, best = j, dd
		}
	}
	pick := nearest
	if len(band) > 0 {
		pick = band[g.rand(len(band))]
	}
	if pick < 0 {
		return fmt.Errorf("%w: no unowned planet for player %d's second planet", ErrNoPlacement, i)
	}
	sp := &planets[pick]

	// Redraw a poor environment, each axis 2 + rand(97), until the
	// planet reaches 10%, at most 100 times.
	redraws := 0
	for engine.Habitability(race, sp.Env) < 10 && redraws < 100 {
		for axis := range sp.Env {
			sp.Env[axis] = 2 + g.rand(97)
		}
		redraws++
	}
	// Legacy.SecondPlanetFallback reproduces the original's LEGACY BUG
	// that a second planet whose 100 environment redraws were all used
	// takes the homeworld's environment, even when the last redraw reached
	// 10% (UNIVERSE.md "Second planet", LEGACY BUG CONFIRMED UG29, UG30; a
	// success on exactly the 100th redraw is BINARY-ONLY). Off, the last
	// redraw is kept.
	if redraws == 100 && g.s.Rules.Legacy.SecondPlanetFallback {
		sp.Env = hw.Env
	}

	sp.Owner = i
	g.res.Artifact[pick] = false // ELEGY CHOICE: as on a homeworld.
	sp.Mines, sp.Factories = 10, 4
	sp.HasScanner = true
	for m := range sp.Surface {
		sp.Surface[m] = 100 + g.rand(200)
	}
	// 2/5 of the homeworld's population; the homeworld keeps 4/5.
	sp.Population = hw.Population * 2 / 5
	hw.Population = hw.Population * 4 / 5
	g.setStarbase(sp, start.StarbaseDesigns[1])
	start.SecondPlanet = pick
	return nil
}

// setStarbase puts starbase design d (a Game.Designs index) on p.
func (g *generator) setStarbase(p *engine.Planet, d int) {
	des := g.res.Game.Designs[d]
	p.HasStarbase = true
	p.StarbaseDesign = d
	p.StarbaseHull = des.Hull.StarbaseNumber
	p.StarbaseDock = des.Hull.Dock != 0
}

// StartingPlans are the five battle plans every player starts with
// (COMBAT.md "Battle plans", "Starting plans", MEASURED UG01..UG21: every
// player of every new game, 2 to 16 players, single-human and
// multi-human). "Default" attacks neutrals and enemies in every game
// (COMBAT.md, MEASURED UG01..UG21 and BP-2). Starting fleets use plan 0
// (UNIVERSE.md "Starting ships").
func StartingPlans() []engine.BattlePlan {
	return []engine.BattlePlan{
		{Name: "Default", Tactic: engine.TacticMaximizeRatio, Primary: engine.TargetArmed, Secondary: engine.TargetAny, Attack: engine.AttackNeutralsAndEnemies},
		{Name: "Kill Starbase", Tactic: engine.TacticMaximizeRatio, Primary: engine.TargetStarbase, Secondary: engine.TargetArmed, Attack: engine.AttackNeutralsAndEnemies},
		{Name: "Max-Defense", Tactic: engine.TacticMaximizeNet, Primary: engine.TargetArmed, Secondary: engine.TargetBombersFreighters, Attack: engine.AttackNeutralsAndEnemies},
		{Name: "Sniper", Tactic: engine.TacticDisengageIfChallenged, Primary: engine.TargetUnarmed, Secondary: engine.TargetNone, Attack: engine.AttackNeutralsAndEnemies},
		{Name: "Chicken", Tactic: engine.TacticDisengage, Primary: engine.TargetAny, Secondary: engine.TargetNone, Attack: engine.AttackNeutralsAndEnemies},
	}
}

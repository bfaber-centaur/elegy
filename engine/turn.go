package engine

import (
	"errors"
	"fmt"
	"sort"
)

// ErrNilRand is returned by GenerateTurn when no random source is given.
var ErrNilRand = errors.New("engine: GenerateTurn needs a non-nil Rand")

// ZeroMaxPopulationError is returned by GenerateTurn, under a ruleset with
// Legacy.ZeroMaxPopulationStop, for an Alternate Reality planet with
// population, habitability ≥ 0 and no starbase, whose maximum population
// is 0.
//
// The original stops with an integer divide by zero and generates no year
// (KERNEL.md "Maximum population", CONFIRMED KX-001 Z1, LEGACY BUG). With
// the switch, Elegy refuses the year with this error before changing
// anything, the closest it comes to the original. Without it (the Elegy
// ruleset) the year is generated (crowdingPermille).
type ZeroMaxPopulationError struct {
	Planet int // planet id
}

func (e *ZeroMaxPopulationError) Error() string {
	return fmt.Sprintf("engine: planet %d has population and a maximum population of 0 (Alternate Reality without a starbase)", e.Planet)
}

// checkGenerable reports a state the original cannot generate from. Nothing
// in movement or production changes a planet's maximum population, so the
// check runs once, before the year starts.
func checkGenerable(g *Game) error {
	for i := range g.Planets {
		p := &g.Planets[i]
		if p.Owner == NoOwner || p.Population <= 0 {
			continue
		}
		col := NewColony(p, &g.Players[p.Owner])
		if col.MaxPop == 0 && col.Hab >= 0 {
			return &ZeroMaxPopulationError{Planet: p.ID}
		}
	}
	return nil
}

// NoOwner marks an unowned planet.
const NoOwner = -1

type Game struct {
	// ID identifies the game; order files carry it (orders.go).
	ID   uint64
	Year int

	// Rules is the game's ruleset (ruleset.go), fixed for the whole game.
	// GenerateTurn refuses a game without one and passes it on unchanged.
	Rules Ruleset

	// SlowerTech is the game's slower-tech-advances option.
	SlowerTech bool
	// RandomEvents is the game's random events option (KERNEL.md "Game
	// options during a turn").
	RandomEvents bool
	// Size is the universe size, 0 tiny .. 4 huge (new-minerals chance).
	Size int
	// PublicScores is the game's public player scores option.
	PublicScores bool
	// Victory is the game's victory settings; Decided is set in a year in
	// which a player won (recomputed every year).
	Victory Victory
	Decided bool

	Players []Player
	Planets []Planet
	Designs []Design
	Fleets  []Fleet
	Salvage []Salvage

	// DesignSlots gives each player's design slots their designs
	// (orders_design.go).
	DesignSlots []DesignSlot

	// PlanetScanners is the catalogue of planetary scanners; a planet
	// with a scanner uses the best one its owner's tech allows.
	PlanetScanners []PlanetScanner
	// Defenses is the catalogue of planetary defense types, for the
	// defense coverage estimate.
	Defenses []DefenseType

	// Objects is the game's minefields, wormholes and Mystery Traders
	// (objects.go), or nil for none.
	Objects SpaceObjects
	// Races runs the yearly race check (racecheck.go), or nil to skip it.
	// A checker that keeps state of its own implements RaceCloner, so
	// GenerateTurn leaves the input game's checker unchanged.
	Races RaceChecker
	// Terraform is the terraforming and remote-mining rules
	// (terraformer.go): production's terraform items
	// (production_terraform.go), remote mining and the Orbital Adjusters.
	// Nil skips remote mining and the adjusters, and terraform items then
	// stop the queue.
	Terraform Terraformer
}

// RaceCloner is a RaceChecker that keeps state between years and copies
// it for GenerateTurn's copy of the game.
type RaceCloner interface {
	CloneRaces() RaceChecker
}

type Player struct {
	Race           Race
	Research       ResearchState
	ResearchBudget int // percent of resources taxed for research

	// Plans are the player's battle plans; plan 0 is the default plan,
	// also used by the player's starbases.
	Plans []BattlePlan
	// Relations is this player's view of each player, by player index.
	// Missing entries are neutral; a player is its own friend.
	Relations []Relation

	// DefaultQueue and DefaultLeftoverOnly are what a new colony of the
	// player starts with (TAKEOVER.md "Colonization").
	DefaultQueue        []QueueItem
	DefaultLeftoverOnly bool

	// Dead is set when the player has no planets and no ships (KERNEL.md
	// "Victory conditions").
	Dead bool
	// Computer marks a computer player, whose orders the game loop's
	// driver writes (game/, ai/). The engine reads the flag only for the
	// rules that name computer players (a gifted fleet is refused;
	// Mystery Trader trades).
	Computer bool
}

type Planet struct {
	ID        int
	Pos       Point
	Owner     int // player index, or NoOwner
	Homeworld bool
	Env       [3]int // gravity, temperature, radiation on the 0–100 scale
	// StarbaseHull is 0 without an owner's starbase, else 1..5 in hull
	// order (used by Alternate Reality).
	StarbaseHull int
	// StarbaseDock is set when the owner's starbase here can refuel fleets.
	StarbaseDock bool
	// HasStarbase marks a starbase that takes part in battles, of design
	// StarbaseDesign (an index into Game.Designs) with StarbaseDamage
	// units (1/500 of its armor) of damage.
	HasStarbase    bool
	StarbaseDesign int
	StarbaseDamage int
	// HasScanner marks a planetary scanner (its range comes from the
	// owner's tech, see Game.PlanetScanners).
	HasScanner bool

	// OrigEnv is the planet's original environment, which retro bombs
	// and a Claim Adjuster's lost planet return to (TAKEOVER.md). Game
	// states must set it; the zero value is 0/0/0.
	OrigEnv [3]int

	Population  int // units of 100 colonists
	GrowthCarry int // hundredths of a unit, 0..99 (StarsAPI excessPop)

	Mines, Factories, Defenses int
	Deposits                   [NumMinerals]Deposit
	Surface                    Minerals

	// HasQueue distinguishes a planet with a (possibly empty) production
	// queue from one without.
	HasQueue     bool
	Queue        []QueueItem
	LeftoverOnly bool // contribute only leftover resources to research

	// HasRoute and RouteTo are the planet's route destination, a planet
	// id (launch.go).
	HasRoute bool
	RouteTo  int

	// HasPacketDest and PacketDest are the planet's mass-driver packet
	// destination, a planet id, and PacketSpeed its packet-speed setting
	// (0 when unset), stored as the owner sends them (OBJECTS.md "The
	// settings"; production.go).
	HasPacketDest bool
	PacketDest    int
	PacketSpeed   int
}

type TurnResult struct {
	Game   Game
	Events []Event
	// Views is each player's knowledge at the end of the year
	// (SCANNING.md), by player index.
	Views []PlayerView
	// Scores is every player's score record for the year, by player.
	// Views[v].Scores holds the ones player v may see.
	Scores []ScoreRecord
	// Orders is the outcome of every order file and order applied at
	// the start of the year (YearOrders); a rejected order has an Err.
	Orders []OrderResult
}

// GenerateTurn advances the game one year with the J-RC3 peaceful kernel
// (KERNEL.md "Turn order"): the players' orders (YearOrders), the
// takeover tasks before movement
// (TAKEOVER.md), fleet movement, then mining for every planet,
// then per planet resources, research tax and production queue, then population
// growth for every planet, then starbase refuelling, then research
// level-ups and random events, then battles (COMBAT.md), bombing and the takeover tasks
// after movement (TAKEOVER.md), the after-movement research level-up check and
// repair. Each player's view of the result (SCANNING.md) comes last.
//
// Not yet modelled: following fleets and the waypoint check after the
// orders (step 1a.3), waypoint tasks other than unloads, colonize, load
// and merge, space objects
// other than battle salvage, the Mystery Trader, fuel generators, mine
// sweeping, terraforming other than the Claim Adjuster's year-end step,
// and remote mining. Scores and victory come after
// the year advances.
//
// The year runs under the game's own ruleset, game.Rules; a game without
// a valid one returns its Validate error (ErrNoRuleset for none).
//
// rng must not be nil: the turn's random draws (mining's +1, random events, battles) come only from
// it, and there is deliberately no hidden default generator. A nil rng
// returns ErrNilRand. A state the original cannot generate from returns
// a *ZeroMaxPopulationError (see checkGenerable) when the ruleset says to
// stop there.
//
// game is not modified; the returned Game is independent of it.
func GenerateTurn(
	game Game,
	orders []PlayerOrders,
	rng Rand,
) (TurnResult, error) {
	if rng == nil {
		return TurnResult{}, ErrNilRand
	}
	if err := game.Rules.Validate(); err != nil {
		return TurnResult{}, err
	}
	if err := checkGenerable(&game); game.Rules.Legacy.ZeroMaxPopulationStop && err != nil {
		return TurnResult{}, err
	}
	g := game.clone()
	var events []Event

	// The orders (KERNEL.md "Turn order" step 1): the player shuffle,
	// whose draws are the first of the year, then each player's file in
	// that order; gifts to other players are credited at the end of the
	// replay, before any waypoint task (TAKEOVER.md "Manual cargo
	// transfers to other players"). Step 1a's registration check has no
	// Elegy equivalent.
	applied := YearOrders(&g, orders, rng)
	events = append(events, applied.Events...)
	// The waypoint check after the orders (step 1a.3).
	g.waypointCheck()

	// Waypoint tasks before movement (TAKEOVER.md "Where each task
	// happens", step 2): unloads and colonize, then the drops. Colonists
	// put onto other players' planets by the orders come first in the
	// drop queue, in the order given (TAKEOVER.md "Order inside a phase",
	// MEASURED TK-501).
	// gained marks the players that gained tech this turn, from a capture
	// or a battle.
	gained := map[int]bool{}
	// Scraps (TaskScrap) run in their place among these tasks.
	recycled := scrapYear{}
	queue, ev := g.unloadTasks(g.phaseStart(), nil, func(i int) []Event { return g.scrap(i, rng, gained, recycled) })
	events = append(events, ev...)
	queue = append(append([]drop(nil), applied.drops...), queue...)
	events = append(events, g.resolveQueue(queue, rng, gained)...)
	// The first research level-up check (KERNEL.md "Turn order" step 2.4).
	for i := range g.Players {
		pl := &g.Players[i]
		pl.Research = LevelUpCheck(pl.Research, pl.Race, g.SlowerTech)
	}
	events = append(events, g.loadPass(false)...)

	// The race check (KERNEL.md "Turn order" step 2a).
	if g.Races != nil {
		events = append(events, g.Races.CheckRaces(&g)...)
	}

	// Objects move before fleets, then the waypoint check again
	// (step 3.2).
	if g.Objects != nil {
		events = append(events, g.Objects.MoveObjects(&g, rng)...)
		g.waypointCheck()
	}

	start := map[int]Point{}
	for _, f := range g.Fleets {
		start[f.ID] = f.Pos
	}
	moveEvents, gated := g.moveAll(rng)
	events = append(events, moveEvents...)
	for i := range g.Fleets {
		if f := &g.Fleets[i]; start[f.ID] != f.Pos {
			events = append(events, g.radiatingColonists(f)...)
		}
	}
	g.generateFuel()
	// Detonations and decay (step 3a).
	if g.Objects != nil {
		g.dropShiplessFleets()
		events = append(events, g.Objects.DecayObjects(&g)...)
		g.dropShiplessFleets()
	}

	research := make([]int, len(g.Players))

	order := make([]int, len(g.Planets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return g.Planets[order[a]].ID < g.Planets[order[b]].ID })

	// Every planet is mined before any planet's production, in id order
	// (KERNEL.md "Mining", BINARY-ONLY draw order), with the population
	// before growth.
	for _, i := range order {
		p := &g.Planets[i]
		if p.Owner == NoOwner || p.Population <= 0 {
			continue
		}
		player := &g.Players[p.Owner]
		col := NewColony(p, player)
		eff := player.Race.MineOutput
		if col.alternateReality() {
			eff = 10
		}
		working := col.WorkingMines(p.Population, p.Mines)
		for m := range NumMinerals {
			gain, d := MineYear(p.Deposits[m], working, eff, p.Homeworld, rng)
			p.Surface[m] += gain
			p.Deposits[m] = d
		}
	}

	// Growth and deaths only act on owned planets with population
	// (KERNEL.md "Population", BINARY-ONLY).
	grows := make([]bool, len(g.Planets))
	for _, i := range order {
		p := &g.Planets[i]
		if p.Owner == NoOwner || p.Population <= 0 {
			continue
		}
		grows[i] = true
		player := &g.Players[p.Owner]
		col := NewColony(p, player)

		// The production caps read the grown population (ASSUMPTION K7:
		// estimated from the environment before this year's production
		// terraforming).
		gp, _ := GrowPopulation(p.Population, p.GrowthCarry, col.MaxPop, player.Race.growthRate(), col.Hab)

		res := col.Resources(p.Population, p.Factories)
		if x := recycled[i]; x > 0 {
			// Ships scrapped here this year (Ultimate Recycling).
			added := recycledResources(res, x) - res
			res += added
			events = append(events, Event{Kind: EventScrapRecycled, Player: p.Owner, Planet: p.ID, Fleet: -1, Count: added})
		}
		r, ev := g.PlanetProduction(i, ProductionInput{
			Colony:         col,
			Resources:      res,
			GrownPop:       gp,
			ResearchBudget: player.ResearchBudget,
			LeftoverOnly:   p.LeftoverOnly,
		})
		research[p.Owner] += r
		events = append(events, ev...)
	}

	// Growth (step 4a) comes after every planet's production, with the
	// environment production terraformed (KERNEL.md "Turn order";
	// MEASURED KX-002 T1, T2).
	for i := range g.Planets {
		if !grows[i] {
			continue
		}
		p := &g.Planets[i]
		player := &g.Players[p.Owner]
		col := NewColony(p, player)
		p.Population, p.GrowthCarry = GrowPopulation(p.Population, p.GrowthCarry, col.MaxPop, player.Race.growthRate(), col.Hab)
		if p.Population <= 0 {
			// A planet whose population dies out is emptied
			// (TAKEOVER.md "Capture").
			events = append(events, g.emptyPlanet(i))
		}
	}

	for i := range g.Players {
		pl := &g.Players[i]
		pl.Research = AddResearch(pl.Research, pl.Race, research[i], g.SlowerTech)
	}
	// Research level-ups, then random events (KERNEL.md "Turn order"
	// steps 4b and 4c), then starbase refuelling (step 5.2).
	events = append(events, g.randomEvents(rng)...)
	if g.Objects != nil {
		// The Mystery Trader's appearance ends step 4c; wormholes move
		// after fleets, then the waypoint check again (step 5.1).
		events = append(events, g.Objects.TraderAppears(&g, rng)...)
		events = append(events, g.Objects.MoveObjectsAgain(&g, rng)...)
		g.waypointCheck()
	}
	refuelFleets(&g)

	// Battles, bombing, the waypoint tasks after movement, then the
	// post-movement research check, then repair (KERNEL.md "Turn order"
	// steps 6–7, TAKEOVER.md "Where each task happens" steps 4–5;
	// research gained in a battle levels up the same turn, CB-018,
	// CB-021).
	owned := g.phaseStart()
	fights := battles(&g, rng, gained)
	events = append(events, fights.events...)
	bombs := g.bombChecks()
	events = append(events, bombing(&g, rng)...)
	// Mystery Trader encounters (step 6b), then the tasks after movement
	// with mine laying (6c.2).
	var layers []MineLayer
	if g.Objects != nil {
		met := g.Objects.MeetTraders(&g, rng)
		events = append(events, met...)
		g.dropShiplessFleets()
		// A Trader's gift fleet counts as not moved this year
		// (OBJECTS-STATUS.md, the Mystery Trader's turn hooks).
		for _, e := range met {
			if e.Kind == EventTraderGift {
				if fi := g.fleetIndex(e.Fleet); fi >= 0 {
					start[e.Fleet] = g.Fleets[fi].Pos
				}
			}
		}
	}
	moved := map[int]bool{}
	for _, f := range g.Fleets {
		// A fleet built this year counts as moved (PRODUCTION-LAUNCH.md,
		// CONFIRMED SL-03).
		if p, ok := start[f.ID]; !ok || p != f.Pos {
			moved[f.ID] = true
		}
	}
	// Remote mining by a fleet that did not move this year, in its place
	// in fleet order (KERNEL.md "Remote mining", CONFIRMED T-35, KB-1B).
	queue, ev = g.unloadTasks(owned, func(i int) {
		if g.Terraform != nil && !moved[g.Fleets[i].ID] {
			g.Terraform.RemoteMine(&g, i, rng)
		}
	}, nil)
	events = append(events, ev...)
	// The layers are taken after the unload pass, which removes the
	// fleets that colonized or scrapped, so MineLayer's fleet indexes
	// stay valid through LayMines and endLayYear. In KERNEL.md "Turn
	// order" step 6c.2 each fleet runs one task, and a colonize or scrap
	// task is not mine laying. ASSUMPTION O15: a Space Demolition fleet
	// consumed in that pass, whose next waypoint is "lay mines", lays
	// nothing.
	if g.Objects != nil {
		layers = g.layers(moved)
	}
	if len(layers) > 0 {
		events = append(events, g.Objects.LayMines(&g, layers)...)
		g.endLayYear(layers)
	}
	events = append(events, g.resolveQueue(queue, rng, gained)...)
	for i := range g.Players {
		pl := &g.Players[i]
		pl.Research = LevelUpCheck(pl.Research, pl.Race, g.SlowerTech)
	}
	events = append(events, g.loadPass(true)...)
	// Mine sweeping (step 7.1).
	if g.Objects != nil {
		events = append(events, g.Objects.SweepMines(&g)...)
	}
	moved = map[int]bool{}
	for _, f := range g.Fleets {
		// A fleet launched this year counts as moved (PRODUCTION-LAUNCH.md,
		// SL-03).
		if p, ok := start[f.ID]; !ok || p != f.Pos {
			moved[f.ID] = true
		}
	}
	// A fleet that jumped by stargate gets no repair this year
	// (OBJECTS.md "Stargates").
	for id := range gated {
		fights.fleets[id] = true
	}
	repair(&g, moved, fights)
	// Claim Adjuster drift and year-end terraforming (KERNEL.md "Turn
	// order" step 7.3), with the levels reached this year.
	events = append(events, g.claimAdjusterYearEnd(rng)...)
	// Remote terraforming by Orbital Adjusters (step 7.4), after the Claim
	// Adjuster step (CONFIRMED OT-4).
	if g.Terraform != nil {
		events = append(events, g.Terraform.Adjust(&g)...)
	}
	// The end-of-year waypoint check (step 7a.2).
	g.waypointCheck()

	g.Year++
	scores := g.scores()
	events = append(events, g.decide(scores)...)

	// Knowledge is computed last, from the final state (SCANNING.md "When
	// knowledge is computed").
	//
	// ASSUMPTION O13: the space objects are seen, with their Space
	// Demolition cloak draws, before the population estimates draw.
	var sights []ObjectSight
	if g.Objects != nil {
		sights = g.Objects.SeeObjects(&g, g.ObjectScanners(), func(fi int) int { return g.fleetCloak(&g.Fleets[fi]) }, rng)
	}
	for _, e := range events {
		if e.Kind == EventPacketDesignSeen && e.Player >= 0 && e.Player < len(g.Players) {
			if len(sights) < len(g.Players) {
				sights = append(sights, make([]ObjectSight, len(g.Players)-len(sights))...)
			}
			sights[e.Player].Designs = append(sights[e.Player].Designs, e.Count)
		}
	}
	views := viewsWith(g, PopulationEstimates(g, rng), fights.seen, bombs, sights)
	for v := range views {
		views[v].Scores = g.visibleScores(v, scores)
	}
	// As each player's file is written: the retarget of other players'
	// fleets, then patrol intercepts (ORDERS.md).
	g.fileRetarget()
	g.patrol(views)

	return TurnResult{
		Game:   g,
		Events: events,
		Views:  views,
		Scores: scores,
		Orders: applied.Results,
	}, nil
}

func (g Game) clone() Game {
	c := g
	c.Players = append([]Player(nil), g.Players...)
	c.Planets = append([]Planet(nil), g.Planets...)
	for i := range c.Planets {
		c.Planets[i].Queue = append([]QueueItem(nil), g.Planets[i].Queue...)
	}
	c.Salvage = append([]Salvage(nil), g.Salvage...)
	c.Designs = append([]Design(nil), g.Designs...)
	c.DesignSlots = append([]DesignSlot(nil), g.DesignSlots...)
	c.Fleets = append([]Fleet(nil), g.Fleets...)
	for i := range c.Fleets {
		c.Fleets[i].Stacks = append([]Stack(nil), g.Fleets[i].Stacks...)
		c.Fleets[i].Waypoints = append([]Waypoint(nil), g.Fleets[i].Waypoints...)
	}
	if g.Objects != nil {
		c.Objects = g.Objects.CloneObjects()
	}
	if r, ok := g.Races.(RaceCloner); ok {
		c.Races = r.CloneRaces()
	}
	return c
}

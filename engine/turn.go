package engine

import (
	"errors"
	"fmt"
	"sort"
)

// ErrNilRand is returned by GenerateTurn when no random source is given.
var ErrNilRand = errors.New("engine: GenerateTurn needs a non-nil Rand")

// ZeroMaxPopulationError is returned by GenerateTurn for an Alternate
// Reality planet with population, habitability ≥ 0 and no starbase, whose
// maximum population is 0.
//
// ELEGY DECISION, not original behavior: the original stops with an
// integer divide by zero and generates no year (KERNEL.md "Maximum
// population", CONFIRMED KX-001 Z1, LEGACY BUG). Elegy refuses the year
// with this error before changing anything. To choose another rule,
// change checkGenerable (and the population rule) only.
type ZeroMaxPopulationError struct {
	Planet int // planet id
}

func (e *ZeroMaxPopulationError) Error() string {
	return fmt.Sprintf("engine: planet %d has population and a maximum population of 0 (Alternate Reality without a starbase)", e.Planet)
}

// checkGenerable reports a state the year cannot be generated from. Nothing
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
	Year int

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

	// PlanetScanners is the catalogue of planetary scanners; a planet
	// with a scanner uses the best one its owner's tech allows.
	PlanetScanners []PlanetScanner
	// Defenses is the catalogue of planetary defense types, for the
	// defense coverage estimate.
	Defenses []DefenseType
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
}

// Stubs: keep these small until real rules/orders require shape.
type PlayerOrders struct{}
type Ruleset interface{}
type JRC3Rules struct{}

func Jrc3() Ruleset {
	return JRC3Rules{}
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
}

// GenerateTurn advances the game one year with the J-RC3 peaceful kernel
// (KERNEL.md "Turn order"): the takeover tasks before movement
// (TAKEOVER.md), fleet movement, then mining for every planet,
// then per planet resources, research tax and production queue, then population
// growth for every planet, then starbase refuelling, then research
// level-ups and random events, then battles (COMBAT.md), bombing and the takeover tasks
// after movement (TAKEOVER.md), a second research level-up check and
// repair. Each player's view of the result (SCANNING.md) comes last.
//
// Not yet modelled: order application, waypoint tasks other than
// unloads and colonize, space objects
// other than battle salvage, the Mystery Trader, fuel generators, mine
// sweeping, terraforming other than the Claim Adjuster's year-end step,
// and remote mining. Scores and victory come after
// the year advances.
//
// rng must not be nil: the turn's random draws (mining's +1, random events, battles) come only from
// it, and there is deliberately no hidden default generator. A nil rng
// returns ErrNilRand. A state the original cannot generate from returns
// a *ZeroMaxPopulationError (see checkGenerable).
//
// game is not modified; the returned Game is independent of it.
func GenerateTurn(
	game Game,
	orders []PlayerOrders,
	rules Ruleset,
	rng Rand,
) (TurnResult, error) {
	if rng == nil {
		return TurnResult{}, ErrNilRand
	}
	if err := checkGenerable(&game); err != nil {
		return TurnResult{}, err
	}
	g := game.clone()
	var events []Event

	// Waypoint tasks before movement (TAKEOVER.md "Where each task
	// happens", step 2): unloads and colonize, then the drops.
	// gained marks the players that gained tech this turn, from a capture
	// or a battle.
	gained := map[int]bool{}
	queue, ev := g.unloadPhase(g.phaseStart())
	events = append(events, ev...)
	events = append(events, g.resolveQueue(queue, rng, gained)...)
	events = append(events, g.loadPass()...)

	start := map[int]Point{}
	for _, f := range g.Fleets {
		start[f.ID] = f.Pos
	}
	events = append(events, moveFleets(&g)...)
	for i := range g.Fleets {
		if f := &g.Fleets[i]; start[f.ID] != f.Pos {
			events = append(events, g.radiatingColonists(f)...)
		}
	}
	g.generateFuel()

	type growth struct{ pop, carry int }
	grown := make([]growth, len(g.Planets))
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

	for _, i := range order {
		p := &g.Planets[i]
		grown[i] = growth{p.Population, p.GrowthCarry}
		// Growth and deaths only act on owned planets with population
		// (KERNEL.md "Population", BINARY-ONLY).
		if p.Owner == NoOwner || p.Population <= 0 {
			continue
		}
		player := &g.Players[p.Owner]
		col := NewColony(p, player)

		// Growth is applied after every planet's production, but the
		// production caps read the grown population.
		gp, gc := GrowPopulation(p.Population, p.GrowthCarry, col.MaxPop, player.Race.growthRate(), col.Hab)
		grown[i] = growth{gp, gc}

		res := col.Resources(p.Population, p.Factories)
		r, ev := RunProduction(p, ProductionInput{
			Colony:         col,
			Resources:      res,
			GrownPop:       gp,
			ResearchBudget: player.ResearchBudget,
			LeftoverOnly:   p.LeftoverOnly,
		})
		research[p.Owner] += r
		events = append(events, ev...)
	}

	for i := range g.Planets {
		p := &g.Planets[i]
		starved := p.Owner != NoOwner && p.Population > 0 && grown[i].pop <= 0
		p.Population = grown[i].pop
		p.GrowthCarry = grown[i].carry
		if starved {
			// A planet whose population dies out is emptied
			// (TAKEOVER.md "Capture").
			events = append(events, g.emptyPlanet(i))
		}
	}

	refuelFleets(&g)

	for i := range g.Players {
		pl := &g.Players[i]
		pl.Research = AddResearch(pl.Research, pl.Race, research[i], g.SlowerTech)
	}
	// Random events end production (KERNEL.md "Turn order" step 4);
	// starbase refuelling (step 5) above does not depend on them.
	events = append(events, g.randomEvents(rng)...)

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
	queue, ev = g.unloadPhase(owned)
	events = append(events, ev...)
	events = append(events, g.resolveQueue(queue, rng, gained)...)
	for i := range g.Players {
		pl := &g.Players[i]
		pl.Research = LevelUpCheck(pl.Research, pl.Race, g.SlowerTech)
	}
	events = append(events, g.loadPass()...)
	moved := map[int]bool{}
	for _, f := range g.Fleets {
		// A fleet launched this year counts as moved (PRODUCTION-LAUNCH.md,
		// SL-03).
		if p, ok := start[f.ID]; !ok || p != f.Pos {
			moved[f.ID] = true
		}
	}
	repair(&g, moved, fights)
	// Claim Adjuster drift and year-end terraforming (KERNEL.md "Turn
	// order" step 7.3), with the levels reached this year.
	events = append(events, g.claimAdjusterYearEnd(rng)...)

	g.Year++
	scores := g.scores()
	events = append(events, g.decide(scores)...)

	// Knowledge is computed last, from the final state (SCANNING.md "When
	// knowledge is computed").
	views := views(g, PopulationEstimates(g, rng), fights.seen, bombs)
	for v := range views {
		views[v].Scores = g.visibleScores(v, scores)
	}

	return TurnResult{
		Game:   g,
		Events: events,
		Views:  views,
		Scores: scores,
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
	c.Fleets = append([]Fleet(nil), g.Fleets...)
	for i := range c.Fleets {
		c.Fleets[i].Stacks = append([]Stack(nil), g.Fleets[i].Stacks...)
		c.Fleets[i].Waypoints = append([]Waypoint(nil), g.Fleets[i].Waypoints...)
	}
	return c
}

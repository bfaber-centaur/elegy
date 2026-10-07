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

	Players []Player
	Planets []Planet
	Designs []Design
	Fleets  []Fleet
}

type Player struct {
	Race           Race
	Research       ResearchState
	ResearchBudget int // percent of resources taxed for research
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
}

// GenerateTurn advances the game one year with the J-RC3 peaceful kernel
// (KERNEL.md "Turn order"): fleet movement, then mining for every planet,
// then per planet resources, research tax and production queue, then population
// growth for every planet, then starbase refuelling, then research
// level-ups.
//
// Not yet modelled: order application, waypoint tasks, space objects,
// random events, fuel generators, battles, mine sweeping, repair, terraforming,
// remote mining and scores.
//
// rng must not be nil: the turn's random draws (mining's +1) come only from
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

	events = append(events, moveFleets(&g)...)

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
		g.Planets[i].Population = grown[i].pop
		g.Planets[i].GrowthCarry = grown[i].carry
	}

	refuelFleets(&g)

	for i := range g.Players {
		pl := &g.Players[i]
		pl.Research = AddResearch(pl.Research, pl.Race, research[i], g.SlowerTech)
	}

	g.Year++

	return TurnResult{
		Game:   g,
		Events: events,
	}, nil
}

func (g Game) clone() Game {
	c := g
	c.Players = append([]Player(nil), g.Players...)
	c.Planets = append([]Planet(nil), g.Planets...)
	for i := range c.Planets {
		c.Planets[i].Queue = append([]QueueItem(nil), g.Planets[i].Queue...)
	}
	c.Designs = append([]Design(nil), g.Designs...)
	c.Fleets = append([]Fleet(nil), g.Fleets...)
	for i := range c.Fleets {
		c.Fleets[i].Stacks = append([]Stack(nil), g.Fleets[i].Stacks...)
		c.Fleets[i].Waypoints = append([]Waypoint(nil), g.Fleets[i].Waypoints...)
	}
	return c
}

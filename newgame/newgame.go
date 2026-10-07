// Package newgame builds a new J-RC3 game (year 2400) from game settings:
// the galaxy's planets, the homeworlds, each player's starting tech,
// planets, designs and fleets, and the wormholes present at the start.
//
// Every rule comes from stars-elegy docs/UNIVERSE.md (with KERNEL.md
// "Habitability", COMPONENTS.md and OBJECTS.md "Wormholes" where it points
// to them); comments name the section and its status. What the spec leaves
// open is marked ELEGY CHOICE (no observable rule to follow, such as the
// random-draw order) or PLACEHOLDER (a rule the spec does not give yet;
// see docs/UNIVERSE-STATUS.md). LEGACY BUG rules sit behind one named
// variable each.
//
// The package reads the engine's state types and the component table and
// changes nothing in the engine.
package newgame

import (
	"errors"
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// Size is the universe size, tiny (0) to huge (4).
type Size int

const (
	Tiny Size = iota
	Small
	Medium
	Large
	Huge
)

// Density is the planet density, sparse (0) to packed (3).
type Density int

const (
	Sparse Density = iota
	Normal
	Dense
	Packed
)

// Positions is the player-positions setting, close (0) to distant (3).
type Positions int

const (
	Close Positions = iota
	Moderate
	Farther
	Distant
)

// Level is a computer player's level.
type Level int

const (
	Easy Level = iota
	Standard
	Harder
	Expert
)

// Spend is where a race's leftover advantage points go (UNIVERSE.md
// "Leftover advantage points").
type Spend int

const (
	SpendSurfaceMinerals Spend = iota
	SpendConcentrations
	SpendMines
	SpendFactories
	SpendDefenses
)

// PlayerSetup is one player of a new game.
type PlayerSetup struct {
	// Race is the player's race. For a computer player it is that
	// computer type's built-in race, which the caller supplies: the
	// original's computer races are game data UNIVERSE.md does not
	// specify ("Computer players").
	Race engine.Race
	// Name is the player's name. A computer player without a name gets
	// a PLACEHOLDER name ("Computer N"): the original's 24 built-in
	// names are game data not in the spec.
	Name string

	Computer bool
	Level    Level // computer players only

	// LeftoverPoints are the race-design points left unspent. How they
	// are computed is the race-design point system, not specified in
	// UNIVERSE.md; the caller supplies them. Computer players always use
	// 50.
	LeftoverPoints int
	Spend          Spend
	// ExpensiveAt3 is the race option "expensive fields start at tech 3".
	ExpensiveAt3 bool
}

// Settings are a new game's settings (UNIVERSE.md "Settings that shape a
// new game").
type Settings struct {
	Size      Size
	Density   Density
	Positions Positions
	Players   []PlayerSetup

	MaxMinerals    bool
	SlowerTech     bool
	BBS            bool // accelerated BBS play
	NoRandomEvents bool
	// ComputerAlliances and PublicScores change nothing at creation
	// (BINARY-ONLY); they are carried for later systems.
	ComputerAlliances bool
	PublicScores      bool
	Clumping          bool
}

// Width is the galaxy width W in light years (UNIVERSE.md "Conventions").
func (s Settings) Width() int { return (int(s.Size) + 1) * 400 }

// Origin is the galaxy's lowest coordinate on both axes.
const Origin = 1000

// Wormhole is a pair of linked wormhole ends.
type Wormhole struct {
	Ends [2]WormholeEnd
}

// WormholeEnd is one end of a wormhole (OBJECTS.md "Wormholes").
type WormholeEnd struct {
	Pos engine.Point
	// Stability is the end's stability class, 0..2 at creation.
	Stability int
	// Years since the end last jumped.
	Years int
}

// PlayerStart is where a player's starting objects are in the Game.
type PlayerStart struct {
	Name string
	// Homeworld and SecondPlanet are planet indexes; SecondPlanet is −1
	// without one.
	Homeworld    int
	SecondPlanet int
	// StarbaseDesigns and ShipDesigns index Game.Designs, in the player's
	// design-slot order (starbase design 0, 1; ship design 0, 1, ...).
	StarbaseDesigns []int
	ShipDesigns     []int
	// Fleets index Game.Fleets, in the player's fleet numbering order.
	Fleets []int
}

// Result is a new game.
type Result struct {
	Game engine.Game
	// NameIndex is each planet's name index (0..998) into the original's
	// 999-name list. The name texts are original game data, not in the
	// spec; PlanetName gives Elegy's placeholder text.
	NameIndex []int
	// Artifact marks planets that carry an artifact.
	Artifact  []bool
	Wormholes []Wormhole
	Players   []PlayerStart
}

// PlanetName is a PLACEHOLDER name for a name index: the original's name
// list is game data (UNIVERSE.md "Names").
func PlanetName(index int) string { return fmt.Sprintf("Planet %d", index+1) }

// Errors returned by Generate.
var (
	ErrNilRand     = errors.New("newgame: Generate needs a non-nil Rand")
	ErrSettings    = errors.New("newgame: invalid settings")
	ErrNoPlacement = errors.New("newgame: no homeworld placement exists")
)

// MaxPlayers is the most players a game holds.
const MaxPlayers = 16

func (s Settings) validate() error {
	switch {
	case s.Size < Tiny || s.Size > Huge:
		return fmt.Errorf("%w: size %d", ErrSettings, s.Size)
	case s.Density < Sparse || s.Density > Packed:
		return fmt.Errorf("%w: density %d", ErrSettings, s.Density)
	case s.Positions < Close || s.Positions > Distant:
		return fmt.Errorf("%w: player positions %d", ErrSettings, s.Positions)
	case len(s.Players) < 1 || len(s.Players) > MaxPlayers:
		return fmt.Errorf("%w: %d players", ErrSettings, len(s.Players))
	}
	for i, p := range s.Players {
		if p.Computer && (p.Level < Easy || p.Level > Expert) {
			return fmt.Errorf("%w: player %d level %d", ErrSettings, i, p.Level)
		}
		if p.Race.PRT < engine.PRTHyperExpansion || p.Race.PRT > engine.PRTSpaceDemolition {
			return fmt.Errorf("%w: player %d primary racial trait %d is not one of the ten", ErrSettings, i, p.Race.PRT)
		}
		if p.Spend < SpendSurfaceMinerals || p.Spend > SpendDefenses {
			return fmt.Errorf("%w: player %d spend %d", ErrSettings, i, p.Spend)
		}
	}
	return nil
}

// Generate builds a new game. Every random draw comes from rng, so the
// same settings and generator state give the same game; NewRand(seed) is
// Elegy's seeded generator. There is deliberately no default generator.
func Generate(s Settings, rng engine.Rand) (Result, error) {
	if rng == nil {
		return Result{}, ErrNilRand
	}
	if err := s.validate(); err != nil {
		return Result{}, err
	}
	gen := &generator{s: s, w: s.Width(), rng: rng, cat: engine.Components()}
	return gen.run()
}

type generator struct {
	s   Settings
	w   int
	rng engine.Rand
	cat *engine.Catalog

	res Result
}

func (g *generator) rand(n int) int { return g.rng.Intn(n) }

func (g *generator) run() (Result, error) {
	g.res.Game = engine.Game{
		Year:           2400,
		SlowerTech:     g.s.SlowerTech,
		PlanetScanners: g.cat.PlanetScanners(),
		Defenses:       g.cat.Defenses(),
	}

	pos := g.placePlanets()
	if g.s.Clumping {
		g.clump(pos, planetCount(g.s.Size, g.s.Density))
	}
	g.makePlanets(pos)

	hws, err := g.placeHomeworlds()
	if err != nil {
		return Result{}, err
	}
	if err := g.setUpPlayers(hws); err != nil {
		return Result{}, err
	}
	g.makeWormholes()
	return g.res, nil
}

func d2(a, b engine.Point) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}

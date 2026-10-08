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
	"github.com/bfaber-centaur/elegy/objects"
	"github.com/bfaber-centaur/elegy/races"
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
	// Race is the player's race design, with its name, leftover spend
	// and options. For a computer player it is that computer type's
	// built-in race (ComputerPlayer). A race marked Random is generated
	// at creation (races.Generate).
	Race races.Design

	Computer bool
	Level    Level // computer players only
}

// ComputerPlayer is a computer player of definition-file type 1–6 (HE,
// SS, IS, CA, PP, AR) at a level, with its built-in race (stars-elegy
// AI.md "Built-in races"). The race's leftover spend is the caller's
// (races.BuiltIn, ASSUMPTION B1).
func ComputerPlayer(typ int, level Level) (PlayerSetup, error) {
	d, err := races.BuiltIn(typ, int(level))
	if err != nil {
		return PlayerSetup{}, err
	}
	return PlayerSetup{Race: d, Computer: true, Level: level}, nil
}

// player is a PlayerSetup after the RACES.md creation rules: the race
// actually played, its name, leftover L and effective spend.
type player struct {
	Race         engine.Race
	Design       races.Design
	Points       int
	Name         string
	Computer     bool
	Level        Level
	Leftover     int
	Spend        Spend
	ExpensiveAt3 bool
}

// Settings are a new game's settings (UNIVERSE.md "Settings that shape a
// new game").
type Settings struct {
	// Rules is the game's ruleset (engine.Ruleset), which the new game
	// carries for its whole life (engine.Game.Rules). It must be valid.
	Rules engine.Ruleset

	Size      Size
	Density   Density
	Positions Positions
	Players   []PlayerSetup

	MaxMinerals    bool
	SlowerTech     bool
	BBS            bool // accelerated BBS play
	NoRandomEvents bool
	// SlowerTech, ComputerAlliances and PublicScores change nothing at
	// creation but the option word (CONFIRMED UG26–UG28); they are
	// carried for later systems.
	ComputerAlliances bool
	PublicScores      bool
	Clumping          bool

	// Victory is the victory-conditions dialog, in engine.Victory's
	// encoding. The game stores it with each disabled condition's value
	// as 0 (storedVictory).
	Victory engine.Victory
}

// Width is the galaxy width W in light years (UNIVERSE.md "Conventions").
func (s Settings) Width() int { return (int(s.Size) + 1) * 400 }

// Origin is the galaxy's lowest coordinate on both axes.
const Origin = 1000

// Wormhole and WormholeEnd are the space objects package's (OBJECTS.md
// "Wormholes").
type (
	Wormhole    = objects.Wormhole
	WormholeEnd = objects.WormholeEnd
)

// PlayerStart is where a player's starting objects are in the Game.
type PlayerStart struct {
	Name string
	// Race is the race the player starts with, after repairs, the
	// illegal-race replacement or Random generation (RACES.md "At game
	// creation"); Race.Tampered is the race's tampered flag. Points are
	// its advantage points.
	Race   races.Design
	Points int
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
	Artifact []bool
	// Wormholes are the new game's wormholes, also held by Game.Objects.
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
	if err := s.Rules.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrSettings, err)
	}
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
		// Human races are repaired at creation (an out-of-range PRT
		// becomes JOAT); computer races are taken as given, so they
		// must be well formed.
		if p.Computer && (p.Race.Race.PRT < engine.PRTHyperExpansion || p.Race.Race.PRT > engine.PRTSpaceDemolition) {
			return fmt.Errorf("%w: player %d primary racial trait %d is not one of the ten", ErrSettings, i, p.Race.Race.PRT)
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

	players []player

	res Result
}

func (g *generator) rand(n int) int { return g.rng.Intn(n) }

func (g *generator) run() (Result, error) {
	g.res.Game = engine.Game{
		Year:           2400,
		Rules:          g.s.Rules,
		SlowerTech:     g.s.SlowerTech,
		RandomEvents:   !g.s.NoRandomEvents,
		Size:           int(g.s.Size),
		PublicScores:   g.s.PublicScores,
		Victory:        storedVictory(g.s.Victory),
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
	// Races are settled when the players get their homeworlds, after
	// placement (RACES.md "At game creation" step 6).
	g.resolvePlayers()
	if err := g.setUpPlayers(hws); err != nil {
		return Result{}, err
	}
	// Every planet's original environment is its environment at
	// creation: the original value is the never-terraformed one
	// (stars-elegy KERNEL.md "Terraforming"), and nothing has terraformed
	// a new game.
	for i := range g.res.Game.Planets {
		p := &g.res.Game.Planets[i]
		p.OrigEnv = p.Env
	}
	g.makeWormholes()
	// The game carries its space objects into the turn (engine.Game.Objects);
	// a new game has only its wormholes. Result.Wormholes is the same slice.
	g.res.Game.Objects = &objects.Space{Wormholes: g.res.Wormholes}
	return g.res, nil
}

// storedVictory is the victory settings as a new game stores them: a
// disabled condition's value is stored as 0, which decodes as that
// condition's lowest value (tech: both the level and the field count);
// enabled conditions, the number needed and the minimum years keep
// their values.
//
// MEASURED: every UG vector's stored victory conditions (UG01-E ..
// UG30-E, stars-elegy vectors "ug"). UNIVERSE.md and KERNEL.md do not
// state the rule yet. It matters because a condition's flag is set in
// the score record when met even when disabled (KERNEL.md "Yearly score
// record").
func storedVictory(v engine.Victory) engine.Victory {
	vals := [engine.NumVictory]*int{&v.Planets, &v.TechLevel, &v.Score, &v.Lead, &v.Resources, &v.Capital, &v.Highest}
	for c, p := range vals {
		if !v.Enabled[c] {
			*p = 0
			if c == engine.VictoryTech {
				v.TechFields = 0
			}
		}
	}
	return v
}

func d2(a, b engine.Point) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}

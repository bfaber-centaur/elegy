package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/objects"
	"github.com/bfaber-centaur/elegy/races"
	"github.com/bfaber-centaur/elegy/terraform"
)

// Game is a running game: the authoritative state, its random stream and
// what the last generated year told each player.
//
// The whole game is a pure function of its creation settings, its seed
// and the orders submitted each year: the same inputs give the same
// state hash (Hash), and a game saved and loaded mid-way continues
// exactly as one that never stopped.
type Game struct {
	// State is the authoritative game. Its Objects, Races and Terraform
	// are the objects, races and terraform packages' implementations.
	State engine.Game
	// Seed is the seed the game was created with (informational; the
	// stream's position is in the generator).
	Seed uint64

	rng *newgame.Source
	// What the last year produced, for the players' reports.
	views   []engine.PlayerView
	events  []engine.Event
	results []engine.OrderResult
	// Every player's planet history (Report.History) and wormhole
	// sightings (Report.Wormholes).
	history   history
	wormholes wormholeHistory
	// designs is every player's knowledge of other players' designs
	// (Report.KnownDesigns).
	designs designHistory
	// nameIndex is each planet's name index, by planet index
	// (newgame.Result.NameIndex).
	nameIndex []int
	// levels is each player's computer-player level (newgame.PlayerSetup
	// Level), by player index; it means nothing for a human player.
	levels []newgame.Level
}

// New creates a game from a ruleset, new-game settings and a seed.
//
// ELEGY CHOICE: one generator, newgame.NewRand(seed), makes every draw
// of the game: creation, then the starting knowledge, then each year's
// turn. The original's stream is not part of the spec (stars-elegy
// UNIVERSE.md "Randomness and seeds").
//
// ELEGY CHOICE: the starting knowledge (the reports for the first year)
// is engine.Views of the new game, with population estimates drawn from
// the game's stream right after creation. The original writes each
// player's first file at creation; how it draws those estimates is not
// in the spec.
//
// rules is the game's ruleset (engine.Ruleset, docs/RULESET.md): it
// replaces s.Rules and stays with the game, and its save, for the game's
// whole life.
func New(rules engine.Ruleset, s newgame.Settings, seed uint64) (*Game, error) {
	s.Rules = rules
	rng := newgame.NewRand(seed)
	res, err := newgame.Generate(s, rng)
	if err != nil {
		return nil, err
	}
	st := res.Game
	st.ID = gameID(s, seed)
	designs := make([]races.Design, len(res.Players))
	computer := make([]bool, len(res.Players))
	for i, p := range res.Players {
		designs[i] = p.Race
		computer[i] = st.Players[i].Computer
	}
	// The yearly race check (RACES.md "In a running game") and the
	// terraforming and remote-mining rules run in every game.
	st.Races = &races.GameRaces{Designs: designs, Computer: computer}
	st.Terraform = terraform.Rules{}
	if st.Objects == nil {
		st.Objects = &objects.Space{}
	}
	g := &Game{State: st, Seed: seed, rng: rng, nameIndex: res.NameIndex}
	for _, p := range s.Players {
		g.levels = append(g.levels, p.Level)
	}
	g.views = engine.Views(st, engine.PopulationEstimates(st, rng))
	g.history = history(nil).record(st.Year, g.views)
	g.wormholes = wormholeHistory(nil).record(st.Year, g.views, space(st))
	g.designs = designHistory(nil).record(st.Year, g.views, st.Designs)
	return g, nil
}

// gameID is a new game's id, which order files carry (stars-elegy
// ORDERS.md "Wrong game or wrong year").
//
// ELEGY CHOICE: the id is the first eight bytes of the SHA-256 of the
// settings and seed, never 0, so it is reproducible and games made from
// different settings or seeds almost never share one. The spec does not
// say how the original picks a game id.
func gameID(s newgame.Settings, seed uint64) uint64 {
	b, err := json.Marshal(struct {
		Settings newgame.Settings
		Seed     uint64
	}{s, seed})
	if err != nil {
		panic(fmt.Sprintf("game: encoding settings: %v", err))
	}
	sum := sha256.Sum256(b)
	return max(1, binary.BigEndian.Uint64(sum[:8]))
}

// ComputerLevel is a computer player's level, and false for a player who
// is not a computer player. The engine marks computer players
// (engine.Player.Computer); the level is the new-game setting
// (newgame.PlayerSetup.Level), which the game keeps so a loaded game can
// run its computer players again.
func (g *Game) ComputerLevel(player int) (newgame.Level, bool) {
	if player < 0 || player >= len(g.State.Players) || !g.State.Players[player].Computer {
		return 0, false
	}
	return g.levels[player], true
}

// Report is player's report for the current year, with its planet
// history. Its Rand is nil: only Advance gives drivers the stream.
func (g *Game) Report(player int) (Report, error) {
	r, err := NewReport(g.State, player, g.views, g.events, g.results)
	if err != nil {
		return Report{}, err
	}
	r.History = g.history.list(player)
	for i, p := range g.State.Planets {
		u := UniversePlanet{ID: p.ID, Pos: p.Pos}
		if i < len(g.nameIndex) {
			u.NameIndex = g.nameIndex[i]
		}
		r.Universe = append(r.Universe, u)
	}
	if sp, ok := g.State.Objects.(*objects.Space); ok {
		r.Objects = sp.Report(&g.State, player, r.View.Objects)
		seen := map[int]bool{}
		for _, id := range r.View.Objects.Wormholes {
			seen[id] = true
		}
		for _, w := range g.wormholes.list(player) {
			wi, ei := w.End/2, w.End%2
			if wi >= len(sp.Wormholes) || !sp.Wormholes[wi].Ends[ei].KnownBy(player) {
				continue
			}
			if r.Self.Computer && !seen[w.End] {
				continue // SCANNING.md "Space objects", MEASURED SC-038
			}
			k := KnownWormholeEnd{End: w.End, Year: w.Year, Pos: w.Pos, Stability: w.Stability, Years: w.Years}
			if d, ok := sp.Destination(wi, ei, player, seen[objects.WormholeEndID(wi, 1-ei)]); ok {
				k.Destination = &d
			}
			r.Wormholes = append(r.Wormholes, k)
		}
	}
	for _, d := range g.designs.list(player) {
		k := KnownDesign{Index: d.Design, Year: d.Year, Hull: d.Hull, Mass: d.Mass, Full: d.Full != nil}
		if d.Full != nil {
			full := deepCopy(*d.Full)
			k.Design = &full
		}
		r.KnownDesigns = append(r.KnownDesigns, k)
	}
	return r, nil
}

// space is the game's space objects, or nil.
func space(g engine.Game) *objects.Space {
	sp, _ := g.Objects.(*objects.Space)
	return sp
}

// Year is what generating one year did.
type Year struct {
	// Orders are the order files submitted, one per living player.
	Orders []engine.PlayerOrders
	// Result is the engine's result for the year; Result.Game is the
	// game's new state.
	Result engine.TurnResult
}

// DriverError is a driver's failure to give orders; the year is not
// generated.
type DriverError struct {
	Player int
	Year   int
	Err    error
}

func (e *DriverError) Error() string {
	return fmt.Sprintf("game: year %d: player %d's driver: %v", e.Year, e.Player, e.Err)
}

func (e *DriverError) Unwrap() error { return e.Err }

// TurnError is the engine's refusal to generate a year.
type TurnError struct {
	Year int
	Err  error
}

func (e *TurnError) Error() string { return fmt.Sprintf("game: year %d: %v", e.Year, e.Err) }

func (e *TurnError) Unwrap() error { return e.Err }

// InvariantError lists what is wrong with the state a year produced
// (Check); the year is not committed.
type InvariantError struct {
	Year     int
	Problems []string
}

func (e *InvariantError) Error() string {
	return fmt.Sprintf("game: year %d produced an inconsistent state: %d problems, first: %s", e.Year, len(e.Problems), e.Problems[0])
}

// ErrDrivers is returned by Advance when the drivers do not match the
// players.
var ErrDrivers = errors.New("game: one driver per player is needed")

// Advance generates one year: every living player's driver gives orders
// from its report, in player order, then engine.GenerateTurn advances
// the game. drivers is indexed by player; a nil driver is Idle. Dead
// players are not asked.
//
// Drivers draw their random numbers from the game's stream through
// Report.Rand, before the year is generated, so every draw a driver
// makes shifts the turn's later draws (stars-elegy AI.md §1 "Random
// numbers"). The draws are part of the saved stream, so a replay with
// the same drivers is identical.
//
// On any error the game is unchanged: a driver failure (*DriverError),
// the engine's refusal (*TurnError) or a state that fails Check
// (*InvariantError).
func (g *Game) Advance(drivers []Driver) (Year, error) {
	if len(drivers) != len(g.State.Players) {
		return Year{}, fmt.Errorf("%w: %d drivers, %d players", ErrDrivers, len(drivers), len(g.State.Players))
	}
	// The stream advances only when the year is committed.
	rng := newgame.NewRand(g.rng.State())
	var files []engine.PlayerOrders
	for p, d := range drivers {
		if g.State.Players[p].Dead {
			continue
		}
		if d == nil {
			d = Idle
		}
		r, err := g.Report(p)
		if err != nil {
			return Year{}, err
		}
		r.Rand = rng
		orders, err := d.Orders(r)
		if err != nil {
			return Year{}, &DriverError{Player: p, Year: g.State.Year, Err: err}
		}
		files = append(files, engine.PlayerOrders{Player: p, GameID: g.State.ID, Year: g.State.Year, Orders: orders})
	}
	res, err := engine.GenerateTurn(g.State, files, rng)
	if err != nil {
		return Year{}, &TurnError{Year: g.State.Year, Err: err}
	}
	if probs := Check(res.Game); len(probs) > 0 {
		return Year{}, &InvariantError{Year: g.State.Year, Problems: probs}
	}
	g.State = res.Game
	g.rng = rng
	g.views, g.events = res.Views, res.Events
	g.results = normalizeResults(res.Orders)
	g.history = g.history.clone().record(res.Game.Year, res.Views)
	g.wormholes = g.wormholes.clone().record(res.Game.Year, res.Views, space(res.Game))
	g.designs = g.designs.clone().record(res.Game.Year, res.Views, res.Game.Designs)
	return Year{Orders: files, Result: res}, nil
}

// Run advances the game the given number of years with the same drivers.
// It stops at the first error, after the years already generated.
func (g *Game) Run(years int, drivers []Driver) error {
	for range years {
		if _, err := g.Advance(drivers); err != nil {
			return err
		}
	}
	return nil
}

// orderSentinels are the engine's order errors a result can carry; a
// saved result keeps which one it was (normalizeResults).
var orderSentinels = []error{
	engine.ErrWrongGame, engine.ErrOutOfDate, engine.ErrLaterYear, engine.ErrNoSuchPlayer,
	engine.ErrDuplicateFile, engine.ErrNotYours, engine.ErrNoSuchObject, engine.ErrOutOfRange,
	engine.ErrNotModelled, engine.ErrNotTogether, engine.ErrRefusedByOwner,
}

// orderError is an order result's error as the game keeps it: the
// message and the engine sentinel it wraps, if any. It survives a save
// and load unchanged, so a driver that tests errors.Is decides the same
// after a reload.
type orderError struct {
	msg      string
	sentinel error
}

func (e *orderError) Error() string { return e.msg }
func (e *orderError) Unwrap() error { return e.sentinel }

func sentinelIndex(err error) int {
	for i, s := range orderSentinels {
		if errors.Is(err, s) {
			return i
		}
	}
	return -1
}

func newOrderError(msg string, sentinel int) error {
	e := &orderError{msg: msg}
	if sentinel >= 0 && sentinel < len(orderSentinels) {
		e.sentinel = orderSentinels[sentinel]
	}
	return e
}

func normalizeResults(rs []engine.OrderResult) []engine.OrderResult {
	out := make([]engine.OrderResult, len(rs))
	for i, r := range rs {
		out[i] = r
		if r.Err != nil {
			out[i].Err = newOrderError(r.Err.Error(), sentinelIndex(r.Err))
		}
	}
	return out
}

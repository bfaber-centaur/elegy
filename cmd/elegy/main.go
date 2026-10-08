// Command elegy creates and advances Elegy games from the command line,
// with no user interface: games are save files and orders are order
// files (docs/GAME-LOOP.md).
//
//	elegy new -seed 1 -players 3 -size small -rules elegy -o game.json
//	elegy new -race default -race me.json -ai rototill:expert,cybertron -o game.json
//	elegy race > me.json                                   (a race file to edit)
//	elegy report -game game.json -player 0 > report.json
//	elegy orders -game game.json -player 0 > orders.json   (an empty order file to fill)
//	elegy turn -game game.json orders0.json orders2.json
//	elegy hash -game game.json
//	elegy play -seed 1 -players 3 -years 40                 (idle players; prints each year's hash)
//	elegy play -players 1 -ai robotoid,rototill,cybertron -years 40
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bfaber-centaur/elegy/ai"
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "elegy:", err)
		os.Exit(1)
	}
}

const usage = "usage: elegy new|race|report|orders|turn|hash|play [flags]; elegy <command> -h for flags"

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "new":
		return cmdNew(args, out)
	case "race":
		return cmdRace(args, out)
	case "report":
		return cmdReport(args, out)
	case "orders":
		return cmdOrders(args, out)
	case "turn":
		return cmdTurn(args, out)
	case "hash":
		return cmdHash(args, out)
	case "play":
		return cmdPlay(args, out)
	}
	return fmt.Errorf("unknown command %q; %s", cmd, usage)
}

// settingsFlags are the new-game flags shared by new and play.
type settingsFlags struct {
	seed      *uint64
	players   *int
	size      *string
	density   *string
	positions *string
	rules     *string
	races     *raceList
	computers *string
}

// raceList is the repeatable -race flag.
type raceList []string

func (l *raceList) String() string     { return strings.Join(*l, ",") }
func (l *raceList) Set(v string) error { *l = append(*l, v); return nil }

func addSettings(fs *flag.FlagSet) settingsFlags {
	races := &raceList{}
	fs.Var(races, "race", "a human player's race, once per human player: default, random or a race file (elegy race); replaces -players")
	return settingsFlags{
		seed:      fs.Uint64("seed", 1, "game seed"),
		players:   fs.Int("players", 2, "number of players, each with the default race"),
		size:      fs.String("size", "small", "universe size: tiny, small, medium, large, huge"),
		density:   fs.String("density", "normal", "planet density: sparse, normal, dense, packed"),
		positions: fs.String("positions", "moderate", "player positions: close, moderate, farther, distant"),
		rules:     fs.String("rules", engine.ElegyRulesID, "ruleset: "+engine.ElegyRulesID+" or "+engine.FaithfulRulesID),
		races:     races,
		computers: fs.String("ai", "", "computer players after the human players, comma-separated personality[:level]: robotoid, rototill, cybertron; easy, standard, harder, expert (default expert)"),
	}
}

func pick(name, v string, options []string) (int, error) {
	for i, o := range options {
		if strings.EqualFold(v, o) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("-%s %q: one of %s", name, v, strings.Join(options, ", "))
}

func (f settingsFlags) newGame(fs *flag.FlagSet) (*game.Game, error) {
	var rules engine.Ruleset
	switch strings.ToLower(*f.rules) {
	case engine.ElegyRulesID:
		rules = engine.ElegyRules()
	case engine.FaithfulRulesID:
		rules = engine.FaithfulRules()
	default:
		return nil, fmt.Errorf("-rules %q: one of %s, %s", *f.rules, engine.ElegyRulesID, engine.FaithfulRulesID)
	}
	size, err := pick("size", *f.size, []string{"tiny", "small", "medium", "large", "huge"})
	if err != nil {
		return nil, err
	}
	density, err := pick("density", *f.density, []string{"sparse", "normal", "dense", "packed"})
	if err != nil {
		return nil, err
	}
	positions, err := pick("positions", *f.positions, []string{"close", "moderate", "farther", "distant"})
	if err != nil {
		return nil, err
	}
	s := newgame.Settings{Size: newgame.Size(size), Density: newgame.Density(density), Positions: newgame.Positions(positions)}
	if len(*f.races) == 0 {
		for range *f.players {
			s.Players = append(s.Players, newgame.PlayerSetup{Race: races.Default()})
		}
	} else {
		set := false
		fs.Visit(func(fl *flag.Flag) { set = set || fl.Name == "players" })
		if set {
			return nil, errors.New("-players and -race both set; -race gives one human player each")
		}
		for _, v := range *f.races {
			d, err := raceOption(v)
			if err != nil {
				return nil, err
			}
			s.Players = append(s.Players, newgame.PlayerSetup{Race: d})
		}
	}
	if *f.computers != "" {
		for _, v := range strings.Split(*f.computers, ",") {
			ps, err := computerOption(strings.TrimSpace(v))
			if err != nil {
				return nil, err
			}
			s.Players = append(s.Players, ps)
		}
	}
	return game.New(rules, s, *f.seed)
}

// personalities are the computer players Elegy implements, with their
// definition-file types (stars-elegy AI.md table and "Project policy":
// Robotoid, Rototill and Cybertron are the approved three).
var personalities = []struct {
	name string
	typ  int
	prt  engine.PRT
	p    ai.Personality
}{
	{"robotoid", 1, engine.PRTHyperExpansion, ai.Robotoid},
	{"rototill", 4, engine.PRTClaimAdjuster, ai.Rototill},
	{"cybertron", 5, engine.PRTPacketPhysics, ai.Cybertron},
}

var levelNames = []string{"easy", "standard", "harder", "expert"}

// computerOption is one -ai entry, personality[:level].
//
// ELEGY CHOICE: the level defaults to expert, the level the ai package's
// own long games play.
func computerOption(v string) (newgame.PlayerSetup, error) {
	name, lvl, hasLevel := strings.Cut(v, ":")
	level := int(newgame.Expert)
	if hasLevel {
		var err error
		if level, err = pick("ai level", lvl, levelNames); err != nil {
			return newgame.PlayerSetup{}, err
		}
	}
	for _, c := range personalities {
		if strings.EqualFold(name, c.name) {
			return newgame.ComputerPlayer(c.typ, newgame.Level(level))
		}
	}
	return newgame.PlayerSetup{}, fmt.Errorf("-ai %q: one of robotoid, rototill, cybertron (Turindrone, Automitron and Macinti are not implemented)", name)
}

// drivers are the game's computer players' drivers, by player; nil for
// a human player. A computer player's personality is the one whose
// built-in races have its PRT (AI.md table: each definition-file type
// has one PRT).
func drivers(g *game.Game) ([]game.Driver, error) {
	out := make([]game.Driver, len(g.State.Players))
	for p, pl := range g.State.Players {
		lvl, ok := g.ComputerLevel(p)
		if !ok {
			continue
		}
		found := false
		for _, c := range personalities {
			if c.prt == pl.Race.PRT {
				out[p] = ai.NewDriver(c.p, ai.Level(lvl))
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("player %d is a computer player of PRT %d, which has no implemented personality", p, pl.Race.PRT)
		}
	}
	return out, nil
}

// The race file format. ELEGY CHOICE: a race file is Elegy's own
// versioned JSON document holding a races.Design, like the save and
// order files (docs/GAME-LOOP.md).
const (
	raceFormat  = "elegy-race"
	raceVersion = 1
)

type raceFile struct {
	Format  string       `json:"format"`
	Version int          `json:"version"`
	Race    races.Design `json:"race"`
}

// raceOption is one -race value: default, random or a race file.
func raceOption(v string) (races.Design, error) {
	switch strings.ToLower(v) {
	case "default":
		return races.Default(), nil
	case "random":
		return races.RandomTemplate("Random"), nil
	}
	b, err := os.ReadFile(v)
	if err != nil {
		return races.Design{}, fmt.Errorf("-race: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var f raceFile
	if err := dec.Decode(&f); err != nil {
		return races.Design{}, fmt.Errorf("-race %s: %w", v, err)
	}
	if f.Format != raceFormat || f.Version != raceVersion {
		return races.Design{}, fmt.Errorf("-race %s: format %q version %d; this build reads %q version %d", v, f.Format, f.Version, raceFormat, raceVersion)
	}
	return f.Race, nil
}

func cmdRace(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("race", flag.ContinueOnError)
	name := fs.String("name", "Humanoid", "race name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	d := races.Default()
	d.Name = *name
	b, err := json.MarshalIndent(raceFile{Format: raceFormat, Version: raceVersion, Race: d}, "", " ")
	if err != nil {
		return err
	}
	_, err = out.Write(append(b, '\n'))
	return err
}

func cmdNew(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	sf := addSettings(fs)
	path := fs.String("o", "game.json", "save file to write")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := sf.newGame(fs)
	if err != nil {
		return err
	}
	if err := writeGame(*path, g); err != nil {
		return err
	}
	return printStatus(out, g)
}

func cmdReport(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	path := fs.String("game", "game.json", "save file")
	player := fs.Int("player", 0, "player index")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := readGame(*path)
	if err != nil {
		return err
	}
	r, err := g.Report(*player)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		return err
	}
	_, err = out.Write(append(b, '\n'))
	return err
}

func cmdOrders(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("orders", flag.ContinueOnError)
	path := fs.String("game", "game.json", "save file")
	player := fs.Int("player", 0, "player index")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := readGame(*path)
	if err != nil {
		return err
	}
	if *player < 0 || *player >= len(g.State.Players) {
		return fmt.Errorf("no player %d", *player)
	}
	return game.EncodeOrders(out, game.OrderFile{GameID: g.State.ID, Year: g.State.Year, Player: *player})
}

func cmdTurn(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("turn", flag.ContinueOnError)
	path := fs.String("game", "game.json", "save file; the next year is written back to it")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := readGame(*path)
	if err != nil {
		return err
	}
	// Each file is one player's orders for this year; players with no
	// file keep their standing orders.
	byPlayer := map[int][]engine.Order{}
	for _, name := range fs.Args() {
		fh, err := os.Open(name)
		if err != nil {
			return err
		}
		f, err := game.DecodeOrders(fh)
		fh.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		switch {
		case f.GameID != g.State.ID:
			return fmt.Errorf("%s: orders for game %d, this is game %d", name, f.GameID, g.State.ID)
		case f.Year != g.State.Year:
			return fmt.Errorf("%s: orders for %d, the game is at %d", name, f.Year, g.State.Year)
		case f.Player < 0 || f.Player >= len(g.State.Players):
			return fmt.Errorf("%s: no player %d", name, f.Player)
		}
		if _, dup := byPlayer[f.Player]; dup {
			return fmt.Errorf("%s: a second order file for player %d", name, f.Player)
		}
		byPlayer[f.Player] = f.Orders
	}
	ds, err := drivers(g)
	if err != nil {
		return err
	}
	for p, orders := range byPlayer {
		if ds[p] != nil {
			return fmt.Errorf("an order file for player %d, a computer player", p)
		}
		ds[p] = game.DriverFunc(func(game.Report) ([]engine.Order, error) { return orders, nil })
	}
	y, err := g.Advance(ds)
	if err != nil {
		return err
	}
	printRejected(out, y)
	if err := writeGame(*path, g); err != nil {
		return err
	}
	return printStatus(out, g)
}

func cmdHash(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("hash", flag.ContinueOnError)
	path := fs.String("game", "game.json", "save file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := readGame(*path)
	if err != nil {
		return err
	}
	return printStatus(out, g)
}

func cmdPlay(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	sf := addSettings(fs)
	years := fs.Int("years", 40, "years to generate")
	path := fs.String("o", "", "save file to write at the end (none when empty)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := sf.newGame(fs)
	if err != nil {
		return err
	}
	if err := printStatus(out, g); err != nil {
		return err
	}
	ds, err := drivers(g)
	if err != nil {
		return err
	}
	unsupported := make([]int, len(ds))
	for range *years {
		y, err := g.Advance(ds)
		if err != nil {
			return err
		}
		printRejected(out, y)
		for p, d := range ds {
			if a, ok := d.(*ai.Driver); ok && !g.State.Players[p].Dead {
				unsupported[p] += len(a.Unsupported)
			}
		}
		if err := printStatus(out, g); err != nil {
			return err
		}
	}
	for p := range g.State.Players {
		who := "human"
		if a, ok := ds[p].(*ai.Driver); ok {
			who = fmt.Sprintf("%v %s, %d steps not supported yet", a.Personality, levelNames[a.Level], unsupported[p])
		}
		planets, ships := 0, 0
		for _, pp := range g.State.Planets {
			if pp.Owner == p {
				planets++
			}
		}
		for _, f := range g.State.Fleets {
			if f.Owner == p {
				for _, st := range f.Stacks {
					ships += st.Count
				}
			}
		}
		fmt.Fprintf(out, "player %d (%s): %d planets, %d ships\n", p, who, planets, ships)
	}
	if *path != "" {
		return writeGame(*path, g)
	}
	return nil
}

func printRejected(out io.Writer, y game.Year) {
	for _, o := range y.Result.Orders {
		if o.Err != nil {
			fmt.Fprintf(out, "player %d order %d rejected: %v\n", o.Player, o.Index, o.Err)
		}
	}
}

func printStatus(out io.Writer, g *game.Game) error {
	h, err := g.Hash()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "game %d year %d hash %s\n", g.State.ID, g.State.Year, h)
	return err
}

func readGame(path string) (*game.Game, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	g, err := game.Load(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return g, nil
}

// writeGame writes the save file through a temporary file, so a failed
// write never leaves half a game.
func writeGame(path string, g *game.Game) error {
	var buf bytes.Buffer
	if err := g.Save(&buf); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

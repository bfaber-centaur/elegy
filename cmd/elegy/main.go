// Command elegy creates and advances Elegy games from the command line,
// with no user interface: games are save files and orders are order
// files (docs/GAME-LOOP.md).
//
//	elegy new -seed 1 -players 3 -size small -rules elegy -o game.json
//	elegy report -game game.json -player 0 > report.json
//	elegy orders -game game.json -player 0 > orders.json   (an empty order file to fill)
//	elegy turn -game game.json orders0.json orders2.json
//	elegy hash -game game.json
//	elegy play -seed 1 -players 3 -years 40                 (idle players; prints each year's hash)
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

const usage = "usage: elegy new|report|orders|turn|hash|play [flags]; elegy <command> -h for flags"

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "new":
		return cmdNew(args, out)
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
}

func addSettings(fs *flag.FlagSet) settingsFlags {
	return settingsFlags{
		seed:      fs.Uint64("seed", 1, "game seed"),
		players:   fs.Int("players", 2, "number of players, each with the default race"),
		size:      fs.String("size", "small", "universe size: tiny, small, medium, large, huge"),
		density:   fs.String("density", "normal", "planet density: sparse, normal, dense, packed"),
		positions: fs.String("positions", "moderate", "player positions: close, moderate, farther, distant"),
		rules:     fs.String("rules", engine.ElegyRulesID, "ruleset: "+engine.ElegyRulesID+" or "+engine.FaithfulRulesID),
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

func (f settingsFlags) newGame() (*game.Game, error) {
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
	for range *f.players {
		s.Players = append(s.Players, newgame.PlayerSetup{Race: races.Default()})
	}
	return game.New(rules, s, *f.seed)
}

func cmdNew(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	sf := addSettings(fs)
	path := fs.String("o", "game.json", "save file to write")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g, err := sf.newGame()
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
	drivers := make([]game.Driver, len(g.State.Players))
	for p, orders := range byPlayer {
		drivers[p] = game.DriverFunc(func(game.Report) ([]engine.Order, error) { return orders, nil })
	}
	y, err := g.Advance(drivers)
	if err != nil {
		return err
	}
	for _, o := range y.Result.Orders {
		if o.Err != nil {
			fmt.Fprintf(out, "player %d order %d rejected: %v\n", o.Player, o.Index, o.Err)
		}
	}
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
	g, err := sf.newGame()
	if err != nil {
		return err
	}
	if err := printStatus(out, g); err != nil {
		return err
	}
	drivers := make([]game.Driver, len(g.State.Players))
	for range *years {
		if _, err := g.Advance(drivers); err != nil {
			return err
		}
		if err := printStatus(out, g); err != nil {
			return err
		}
	}
	if *path != "" {
		return writeGame(*path, g)
	}
	return nil
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

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

func cli(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	if err := run(args, &out); err != nil {
		t.Fatalf("elegy %s: %v", strings.Join(args, " "), err)
	}
	return out.String()
}

// TestCLIGame plays two years through the command line: a new game, an
// order file filled in for player 0, two turns, a report and the hash.
// The command-line game must match the same game played through the
// library.
func TestCLIGame(t *testing.T) {
	dir := t.TempDir()
	save := filepath.Join(dir, "game.json")
	cli(t, "new", "-seed", "5", "-players", "2", "-size", "tiny", "-o", save)

	var tmpl bytes.Buffer
	if err := run([]string{"orders", "-game", save, "-player", "0"}, &tmpl); err != nil {
		t.Fatal(err)
	}
	f, err := game.DecodeOrders(&tmpl)
	if err != nil {
		t.Fatal(err)
	}
	research := engine.ResearchOrder{Budget: 40, Field: 2, Next: engine.NextSameField}
	bad := engine.RenameOrder{Fleet: -1, Name: "nobody's"}
	f.Orders = []engine.Order{research, bad}
	ordersPath := filepath.Join(dir, "orders0.json")
	var buf bytes.Buffer
	if err := game.EncodeOrders(&buf, f); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ordersPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	out := cli(t, "turn", "-game", save, ordersPath)
	if !strings.Contains(out, "player 0 order 1 rejected") || !strings.Contains(out, "year 2401") {
		t.Fatalf("turn output:\n%s", out)
	}
	// The same file again is now out of date.
	if err := run([]string{"turn", "-game", save, ordersPath}, &bytes.Buffer{}); err == nil {
		t.Fatal("a 2400 order file was accepted in 2401")
	}
	cli(t, "turn", "-game", save)

	var rep bytes.Buffer
	if err := run([]string{"report", "-game", save, "-player", "0"}, &rep); err != nil {
		t.Fatal(err)
	}
	var r game.Report
	if err := json.Unmarshal(rep.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Year != 2402 || r.Player != 0 || r.Self.ResearchBudget != 40 || r.Self.Research.Current != 2 {
		t.Fatalf("report year %d player %d budget %d field %d", r.Year, r.Player, r.Self.ResearchBudget, r.Self.Research.Current)
	}

	// The library game with the same orders.
	g := newLibraryGame(t)
	drivers := []game.Driver{game.DriverFunc(func(game.Report) ([]engine.Order, error) {
		return []engine.Order{research, bad}, nil
	}), nil}
	if _, err := g.Advance(drivers); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Advance(make([]game.Driver, 2)); err != nil {
		t.Fatal(err)
	}
	h, _ := g.Hash()
	if got := cli(t, "hash", "-game", save); !strings.Contains(got, h) {
		t.Fatalf("command-line game hash %q, library game %s", got, h)
	}
}

func newLibraryGame(t *testing.T) *game.Game {
	t.Helper()
	s := newgame.Settings{Size: newgame.Tiny, Density: newgame.Normal, Positions: newgame.Moderate}
	for range 2 {
		s.Players = append(s.Players, newgame.PlayerSetup{Race: races.Default()})
	}
	g, err := game.New(engine.ElegyRules(), s, 5)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestCLIPlay(t *testing.T) {
	out := cli(t, "play", "-seed", "3", "-players", "3", "-years", "30")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 34 || !strings.Contains(lines[30], "year 2430") || !strings.HasPrefix(lines[33], "player 2 (human)") {
		t.Fatalf("play printed %d lines:\n%s", len(lines), out)
	}
	if again := cli(t, "play", "-seed", "3", "-players", "3", "-years", "30"); again != out {
		t.Fatal("two plays of the same seed differ")
	}
}

// TestCLIComputerPlayers plays 40-year games against each approved
// computer player alone, then all three together, through the command
// line. Every computer player's order must be accepted, every computer
// player must still hold a planet, and the same seed must replay to the
// same hashes.
func TestCLIComputerPlayers(t *testing.T) {
	for _, opponents := range []string{"robotoid", "rototill", "cybertron", "robotoid,rototill:harder,cybertron:standard"} {
		t.Run(opponents, func(t *testing.T) {
			args := []string{"play", "-seed", "7", "-players", "1", "-size", "small", "-ai", opponents, "-years", "40"}
			out := cli(t, args...)
			if strings.Contains(out, "rejected") {
				t.Errorf("rejected orders:\n%s", out)
			}
			n := strings.Count(opponents, ",") + 1
			lines := strings.Split(strings.TrimSpace(out), "\n")
			summary := lines[len(lines)-1-n:]
			if !strings.Contains(lines[len(lines)-2-n], "year 2440") || !strings.HasPrefix(summary[0], "player 0 (human)") {
				t.Fatalf("play output ends:\n%s", strings.Join(lines[len(lines)-2-n:], "\n"))
			}
			for _, line := range summary[1:] {
				t.Log(line)
				if strings.Contains(line, "human") || strings.Contains(line, ": 0 planets") {
					t.Errorf("computer player summary %q", line)
				}
			}
			if again := cli(t, args...); again != out {
				t.Fatal("two plays of the same seed differ")
			}
		})
	}
}

// TestCLIPlayerOptions checks race files, random races and computer
// players in new, and that turn runs a saved game's computer players.
func TestCLIPlayerOptions(t *testing.T) {
	dir := t.TempDir()
	racePath := filepath.Join(dir, "race.json")
	if err := os.WriteFile(racePath, []byte(cli(t, "race", "-name", "Testers")), 0o644); err != nil {
		t.Fatal(err)
	}
	save := filepath.Join(dir, "game.json")
	cli(t, "new", "-seed", "9", "-size", "tiny", "-race", racePath, "-race", "random", "-ai", "rototill:easy", "-o", save)
	g, err := readGame(save)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.State.Players) != 3 || g.State.Players[0].Race.PRT != engine.PRTJackOfAllTrades || g.State.Players[0].Computer || g.State.Players[1].Computer {
		t.Fatalf("players %+v", g.State.Players)
	}
	if lvl, ok := g.ComputerLevel(2); !ok || lvl != newgame.Easy || g.State.Players[2].Race.PRT != engine.PRTClaimAdjuster {
		t.Fatalf("player 2: level %v computer %v PRT %v", lvl, ok, g.State.Players[2].Race.PRT)
	}
	designs := g.State.Races.(*races.GameRaces).Designs
	if designs[0].Name != "Testers" || designs[1].Random {
		t.Fatalf("race 0 named %q; race 1 still the Random template: %v", designs[0].Name, designs[1].Random)
	}

	// turn plays the computer player; an order file for it is refused.
	cli(t, "turn", "-game", save)
	var tmpl bytes.Buffer
	if err := run([]string{"orders", "-game", save, "-player", "2"}, &tmpl); err != nil {
		t.Fatal(err)
	}
	orders := filepath.Join(dir, "orders2.json")
	if err := os.WriteFile(orders, tmpl.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"turn", "-game", save, orders}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "computer player") {
		t.Fatalf("order file for a computer player: err = %v", err)
	}
	g, err = readGame(save)
	if err != nil {
		t.Fatal(err)
	}
	if g.State.Year != 2401 {
		t.Fatalf("year %d after one turn", g.State.Year)
	}
}

func TestCLIErrors(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"fly"},
		{"new", "-size", "enormous"},
		{"new", "-ai", "macinti"},
		{"new", "-ai", "rototill:hardest"},
		{"new", "-players", "2", "-race", "default"},
		{"new", "-race", filepath.Join(t.TempDir(), "missing.json")},
		{"hash", "-game", filepath.Join(t.TempDir(), "missing.json")},
	} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Errorf("elegy %v: no error", args)
		}
	}
}

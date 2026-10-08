// Package regress plays long Elegy games against the computer players
// and checks, every year, that the game stays valid and deterministic.
//
// A Case is one game: a seed, a built-in ruleset and a list of computer
// opponents, played with one idle human player for a number of years.
// Run plays three copies of it in lockstep:
//
//   - the primary game;
//   - a replay from the same seed, which must hash the same every year;
//   - a copy saved and loaded at SaveAt, with fresh drivers, which must
//     hash the same as the primary every year after that.
//
// Every year must also pass game.Check (game.Advance refuses a year that
// fails it) and every check in Case.Checks. Run stops the case at the
// primary's first refused year or failed check, and the replay or reload
// copy at its first difference, and reports each failure with the year,
// the first differing JSON paths (game.Diff) and the command that
// reproduces it. Rejected computer orders are counted, and the case plays
// on.
//
// The harness reports what it finds. It does not decide whether a
// finding is an engine, AI or game-loop bug.
package regress

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bfaber-centaur/elegy/ai"
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/races"
)

// opponent is a computer player the harness plays against, by name: the
// three personalities Elegy implements (stars-elegy AI.md "Project
// policy"), with their definition-file type.
func opponent(name string) (typ int, p ai.Personality, ok bool) {
	switch name {
	case "robotoid":
		return 1, ai.Robotoid, true
	case "rototill":
		return 4, ai.Rototill, true
	case "cybertron":
		return 5, ai.Cybertron, true
	}
	return 0, 0, false
}

// ruleset is the built-in ruleset with the given id.
func ruleset(id string) (engine.Ruleset, bool) {
	switch id {
	case engine.ElegyRulesID:
		return engine.ElegyRules(), true
	case engine.FaithfulRulesID:
		return engine.FaithfulRules(), true
	}
	return engine.Ruleset{}, false
}

// A Check tests one year's game state and returns its problems.
type Check struct {
	Name  string
	Check func(engine.Game) []string
}

// Case is one long game.
type Case struct {
	Seed  uint64
	Rules string // a built-in ruleset id: "elegy" or "jrc3-faithful"
	// Opponents are computer players after the idle human player 0, by
	// name ("robotoid", "rototill" or "cybertron"), at expert level.
	Opponents []string
	Size      newgame.Size
	Years     int
	// SaveAt is the number of years played before the save and load;
	// 0 or ≥ Years skips the reload copy.
	SaveAt int
	// Checks run on every year's state, after game.Check.
	Checks []Check
}

// Name is the case's name, usable as a subtest name.
func (c Case) Name() string {
	return fmt.Sprintf("seed%d/%s/%s", c.Seed, c.Rules, strings.Join(c.Opponents, "+"))
}

// Command is the command that replays the case alone.
func (c Case) Command() string {
	return fmt.Sprintf("go test ./regress -run TestLongGames -count=1 -args -seeds=%d -rules=%s -ai=%s -years=%d -save-at=%d -size=%d",
		c.Seed, c.Rules, strings.Join(c.Opponents, "+"), c.Years, c.SaveAt, c.Size)
}

// Category is a kind of finding.
type Category string

const (
	// Failures.
	SetupFailed    Category = "setup failed"
	DriverFailed   Category = "driver error"
	TurnRefused    Category = "turn refused"
	InvariantFails Category = "invariant failed"  // game.Check
	CheckFails     Category = "year check failed" // Case.Checks
	ReplayDiffers  Category = "replay differs"    // same seed, other hash
	ReloadDiffers  Category = "reload differs"    // saved and loaded copy, other hash
	ReloadFailed   Category = "save or load failed"
	OrderRejected  Category = "computer order rejected"

	// Observations: counted, not failures.
	Unsupported Category = "unsupported computer steps"
)

// IsFailure reports whether a finding of the category fails the case.
func (c Category) IsFailure() bool { return c != Unsupported }

// Finding is one thing a case found.
type Finding struct {
	Category Category
	Year     int // the game year the finding is about
	Detail   string
	// Count is how many there were: for observations, and for rejected
	// orders, whose Year and Detail are the first.
	Count int
}

// Result is a case's outcome.
type Result struct {
	Case     Case
	Years    int // years the primary copy completed
	Hash     string
	Findings []Finding
}

// Failed reports whether any finding is a failure.
func (r Result) Failed() bool {
	for _, f := range r.Findings {
		if f.Category.IsFailure() {
			return true
		}
	}
	return false
}

// Report is the result, finding by finding, with the reproduction.
func (r Result) Report() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d of %d years", r.Case.Name(), r.Years, r.Case.Years)
	for _, f := range r.Findings {
		if f.Category.IsFailure() {
			fmt.Fprintf(&b, "\n  %s in %d: %s", f.Category, f.Year, f.Detail)
			if f.Count > 1 {
				fmt.Fprintf(&b, " (the first of %d)", f.Count)
			}
		} else {
			fmt.Fprintf(&b, "\n  %s: %d", f.Category, f.Count)
		}
	}
	if r.Failed() {
		fmt.Fprintf(&b, "\n  reproduce: %s", r.Case.Command())
	}
	return b.String()
}

// copyGame is one copy of the game with its own drivers.
type copyGame struct {
	g       *game.Game
	drivers []game.Driver
}

func (c Case) newGame() (*game.Game, error) {
	rules, ok := ruleset(c.Rules)
	if !ok {
		return nil, fmt.Errorf("no ruleset %q", c.Rules)
	}
	s := newgame.Settings{Size: c.Size, Density: newgame.Normal, Positions: newgame.Moderate,
		Players: []newgame.PlayerSetup{{Race: races.Default()}}}
	for _, name := range c.Opponents {
		typ, _, ok := opponent(name)
		if !ok {
			return nil, fmt.Errorf("no computer player %q", name)
		}
		ps, err := newgame.ComputerPlayer(typ, newgame.Expert)
		if err != nil {
			return nil, err
		}
		s.Players = append(s.Players, ps)
	}
	return game.New(rules, s, c.Seed)
}

func (c Case) drivers() []game.Driver {
	ds := make([]game.Driver, len(c.Opponents)+1)
	for i, name := range c.Opponents {
		_, p, _ := opponent(name)
		ds[i+1] = ai.NewDriver(p, ai.Expert)
	}
	return ds
}

// Run plays the case.
func Run(c Case) Result {
	res := Result{Case: c}
	fail := func(cat Category, year int, format string, args ...any) {
		res.Findings = append(res.Findings, Finding{Category: cat, Year: year, Detail: fmt.Sprintf(format, args...)})
	}
	primary, err := c.newGame()
	if err != nil {
		fail(SetupFailed, 0, "%v", err)
		return res
	}
	replay, err := c.newGame()
	if err != nil {
		fail(SetupFailed, 0, "%v", err)
		return res
	}
	a := &copyGame{primary, c.drivers()}
	b := &copyGame{replay, c.drivers()}
	var r *copyGame // the reloaded copy, from SaveAt
	unsupported, rejected := 0, -1
	// observed adds the observations made so far: a case that stops early
	// keeps those of the years it played.
	observed := func() Result {
		if unsupported > 0 {
			res.Findings = append(res.Findings, Finding{Category: Unsupported, Count: unsupported})
		}
		return res
	}

	for y := 0; y < c.Years; y++ {
		if c.SaveAt > 0 && y == c.SaveAt {
			var buf bytes.Buffer
			if err := a.g.Save(&buf); err != nil {
				fail(ReloadFailed, a.g.State.Year, "save: %v", err)
			} else if g, err := game.Load(&buf); err != nil {
				fail(ReloadFailed, a.g.State.Year, "load: %v", err)
			} else {
				r = &copyGame{g, c.drivers()}
				if d := differs(a.g, r.g); d != "" {
					fail(ReloadDiffers, a.g.State.Year, "right after the load: %s", d)
					r = nil
				}
			}
		}
		year := a.g.State.Year
		ya, err := a.g.Advance(a.drivers)
		if err != nil {
			fail(categorize(err), year, "%v", err)
			return observed()
		}
		res.Years++
		for _, o := range ya.Result.Orders {
			if o.Err == nil || o.Player == 0 {
				continue
			}
			if rejected < 0 {
				fail(OrderRejected, year, "player %d (%s) order %d: %v", o.Player, c.Opponents[o.Player-1], o.Index, o.Err)
				rejected = len(res.Findings) - 1
			}
			res.Findings[rejected].Count++
		}
		for _, d := range a.drivers {
			if ad, ok := d.(*ai.Driver); ok {
				unsupported += len(ad.Unsupported)
			}
		}
		for _, ch := range c.Checks {
			if probs := ch.Check(a.g.State); len(probs) > 0 {
				fail(CheckFails, year, "%s: %d problems, first: %s", ch.Name, len(probs), probs[0])
				return observed()
			}
		}
		if b != nil {
			if _, err := b.g.Advance(b.drivers); err != nil {
				fail(ReplayDiffers, year, "the replay failed where the primary did not: %v", err)
				b = nil
			} else if d := differs(a.g, b.g); d != "" {
				fail(ReplayDiffers, year, "%s", d)
				b = nil
			}
		}
		if r != nil {
			if _, err := r.g.Advance(r.drivers); err != nil {
				fail(ReloadDiffers, year, "the reloaded copy failed where the primary did not: %v", err)
				r = nil
			} else if d := differs(a.g, r.g); d != "" {
				fail(ReloadDiffers, year, "%s", d)
				r = nil
			}
		}
	}
	res.Hash, _ = a.g.Hash()
	return observed()
}

// differs is "" when two games hash the same, else their first
// differing paths.
func differs(x, y *game.Game) string {
	hx, errx := x.Hash()
	hy, erry := y.Hash()
	if errx != nil || erry != nil {
		return fmt.Sprintf("hash: %v, %v", errx, erry)
	}
	if hx == hy {
		return ""
	}
	diffs, err := game.Diff(x, y, 5)
	if err != nil {
		return fmt.Sprintf("hashes differ; diff: %v", err)
	}
	return "first differences: " + strings.Join(diffs, "; ")
}

func categorize(err error) Category {
	var de *game.DriverError
	var te *game.TurnError
	var ie *game.InvariantError
	switch {
	case errors.As(err, &de):
		return DriverFailed
	case errors.As(err, &te):
		return TurnRefused
	case errors.As(err, &ie):
		return InvariantFails
	}
	return TurnRefused
}

// Summary counts results by category: how many cases had each kind of
// finding, and the cases. Clean cases count under "clean".
func Summary(results []Result) string {
	cases := map[Category][]string{}
	clean := 0
	for _, r := range results {
		seen := map[Category]bool{}
		for _, f := range r.Findings {
			if !seen[f.Category] {
				seen[f.Category] = true
				cases[f.Category] = append(cases[f.Category], r.Case.Name())
			}
		}
		if !r.Failed() {
			clean++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d cases: %d with no failure", len(results), clean)
	cats := make([]string, 0, len(cases))
	for c := range cases {
		cats = append(cats, string(c))
	}
	sort.Strings(cats)
	for _, c := range cats {
		kind := "failure"
		if !Category(c).IsFailure() {
			kind = "observation"
		}
		fmt.Fprintf(&b, "\n  %s (%s): %d cases: %s", c, kind, len(cases[Category(c)]), strings.Join(cases[Category(c)], ", "))
	}
	return b.String()
}

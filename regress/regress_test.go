package regress

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
)

// Flags select a matrix other than the PR subset, for reproducing a case
// or for the long scheduled run:
//
//	go test ./regress -run TestLongGames -count=1 -args -seeds=7 -rules=elegy -ai=rototill -years=40 -save-at=17
var (
	seedsFlag  = flag.String("seeds", "", "comma-separated seeds (default: the PR subset)")
	rulesFlag  = flag.String("rules", "", "comma-separated rulesets (default: both)")
	aiFlag     = flag.String("ai", "", "comma-separated opponent sets, each personality+personality (default: each alone, then all three)")
	yearsFlag  = flag.Int("years", 0, "years per game")
	saveAtFlag = flag.Int("save-at", 0, "years before the save and load")
	sizeFlag   = flag.Int("size", -1, "universe size, 0 tiny .. 4 huge")
)

// The PR subset: 3 seeds × 2 rulesets × 4 opponent sets = 24 games of 60
// years on a small map, each played three times (primary, replay,
// reload after 23 years). 60 years reach Cybertron's first warship group
// (ai/cybertron.md §2).
var (
	prSeeds     = []uint64{1, 2, 3}
	prOpponents = [][]string{{"robotoid"}, {"rototill"}, {"cybertron"}, {"robotoid", "rototill", "cybertron"}}
	prYears     = 60
	prSaveAt    = 23
	prSize      = newgame.Small
)

func matrix(t *testing.T) []Case {
	seeds := prSeeds
	if *seedsFlag != "" {
		seeds = nil
		for _, s := range strings.Split(*seedsFlag, ",") {
			var v uint64
			if _, err := fmt.Sscan(s, &v); err != nil {
				t.Fatalf("-seeds %q: %v", s, err)
			}
			seeds = append(seeds, v)
		}
	}
	rules := []string{"elegy", "jrc3-faithful"}
	if *rulesFlag != "" {
		rules = strings.Split(*rulesFlag, ",")
	}
	opps := prOpponents
	if *aiFlag != "" {
		opps = nil
		for _, set := range strings.Split(*aiFlag, ",") {
			opps = append(opps, strings.Split(set, "+"))
		}
	}
	years, saveAt, size := prYears, prSaveAt, prSize
	if *yearsFlag > 0 {
		years = *yearsFlag
	}
	if *saveAtFlag > 0 {
		saveAt = *saveAtFlag
	}
	if *sizeFlag >= 0 {
		size = newgame.Size(*sizeFlag)
	}
	var cases []Case
	for _, seed := range seeds {
		for _, r := range rules {
			for _, o := range opps {
				cases = append(cases, matrixCase(seed, r, o, size, years, saveAt))
			}
		}
	}
	return cases
}

// matrixCase is one case of a TestLongGames matrix (the PR subset, the
// nightly run or a reproduction), with the year checks.
func matrixCase(seed uint64, rules string, opponents []string, size newgame.Size, years, saveAt int) Case {
	return Case{Seed: seed, Rules: rules, Opponents: opponents, Size: size, Years: years, SaveAt: saveAt, Checks: yearChecks()}
}

// TestLongGames plays the matrix and reports findings by category. Any
// failure fails the test, with its reproduction.
func TestLongGames(t *testing.T) {
	cases := matrix(t)
	results := make([]Result, len(cases))
	var wg sync.WaitGroup
	for i, c := range cases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = Run(c)
		}()
	}
	wg.Wait()
	for _, r := range results {
		if r.Failed() {
			t.Error(r.Report())
		} else {
			t.Log(r.Report())
		}
	}
	t.Log(Summary(results))
}

// The harness reports a failing year check with its year, category and
// reproduction, and stops the game there.
func TestHarnessReportsFailures(t *testing.T) {
	c := Case{Seed: 5, Rules: "elegy", Opponents: []string{"rototill"}, Size: newgame.Tiny, Years: 6, SaveAt: 3,
		Checks: []Check{{Name: "fails from 2403", Check: func(_ engine.Game, r engine.TurnResult) []string {
			if r.Game.Year >= 2403 {
				return []string{"planted problem"}
			}
			return nil
		}}}}
	r := Run(c)
	// Rototill's own unsupported steps (the warp re-pick's minefield rule,
	// ai ASSUMPTION A57) are observations, not failures; they are not
	// this test's.
	var found []Finding
	for _, f := range r.Findings {
		if f.Category.IsFailure() {
			found = append(found, f)
		}
	}
	if len(found) != 1 || found[0].Category != CheckFails || found[0].Year != 2402 {
		t.Fatalf("findings %+v: want exactly the one year-check failure, in the year played from 2402", r.Findings)
	}
	if r.Years != 3 {
		t.Errorf("played %d years: want the case stopped after the failing third year", r.Years)
	}
	rep := r.Report()
	for _, want := range []string{"planted problem", c.Command(), "seed5/elegy/rototill"} {
		if !strings.Contains(rep, want) {
			t.Errorf("report lacks %q:\n%s", want, rep)
		}
	}
	if s := Summary([]Result{r, Run(Case{Seed: 5, Rules: "elegy", Opponents: []string{"rototill"}, Size: newgame.Tiny, Years: 2})}); !strings.Contains(s, "2 cases: 1 with no failure") || !strings.Contains(s, string(CheckFails)+" (failure): 1 cases") {
		t.Errorf("summary:\n%s", s)
	}
	if bad := Run(Case{Seed: 5, Rules: "nonesuch", Years: 1}); len(bad.Findings) != 1 || bad.Findings[0].Category != SetupFailed {
		t.Errorf("unknown ruleset: %+v", bad.Findings)
	}
}

// A case stopped early keeps the observations of the years it played.
// The opponent is replaced by a driver that orders nothing and reports
// one unsupported step a year, so the test does not depend on what the
// computer players cannot order yet.
func TestStoppedCaseKeepsObservations(t *testing.T) {
	c := Case{Seed: 5, Rules: "elegy", Opponents: []string{"rototill"}, Size: newgame.Tiny, Years: 6,
		Checks: []Check{{Name: "fails from 2403", Check: func(_ engine.Game, r engine.TurnResult) []string {
			if r.Game.Year >= 2403 {
				return []string{"planted problem"}
			}
			return nil
		}}}}
	r := run(c, func() []game.Driver { return []game.Driver{nil, oneUnsupported{}} }, nil)
	if len(r.Findings) != 2 || r.Findings[0].Category != CheckFails || !reflect.DeepEqual(r.Findings[1], Finding{Category: Unsupported, Count: 3}) {
		t.Fatalf("findings %+v: want the year-check failure, then 3 unsupported steps for the 3 years played", r.Findings)
	}
}

// oneUnsupported orders nothing and reports one unsupported step.
type oneUnsupported struct{}

func (oneUnsupported) Orders(game.Report) ([]engine.Order, error) { return nil, nil }
func (oneUnsupported) unsupportedSteps() int                      { return 1 }

// The year checks include engine.CheckYear: a result whose year did not
// advance is reported.
func TestYearChecksRunCheckYear(t *testing.T) {
	g, err := Case{Seed: 5, Rules: "elegy", Opponents: []string{"rototill"}, Size: newgame.Tiny}.newGame()
	if err != nil {
		t.Fatal(err)
	}
	var probs []string
	for _, ch := range yearChecks() {
		probs = append(probs, ch.Check(g.State, engine.TurnResult{Game: g.State})...)
	}
	if len(probs) == 0 || !strings.Contains(strings.Join(probs, "; "), "year") {
		t.Fatalf("problems %q: want CheckYear's year problem", probs)
	}
}

// A matrix case runs engine.CheckYear: a starting fleet numbered 0,
// which game.Check accepts and the engine carries over, stops the case
// in its first year with a year-check failure naming CheckYear.
func TestMatrixCaseStopsOnCheckYear(t *testing.T) {
	c := matrixCase(5, "elegy", []string{"rototill"}, newgame.Tiny, 4, 2)
	r := run(c, c.drivers, func(g *game.Game) {
		for i := range g.State.Fleets {
			if g.State.Fleets[i].Owner == 1 {
				g.State.Fleets[i].Number = 0
				return
			}
		}
		t.Fatal("test setup: player 1 has no fleet")
	})
	var failures []Finding
	for _, f := range r.Findings {
		if f.Category.IsFailure() {
			failures = append(failures, f)
		}
	}
	if len(failures) != 1 || failures[0].Category != CheckFails || failures[0].Year != 2400 ||
		!strings.Contains(failures[0].Detail, "engine.CheckYear") || !strings.Contains(failures[0].Detail, "number 0") {
		t.Fatalf("failures %+v: want one CheckYear failure in the year played from 2400", failures)
	}
	if r.Years != 1 {
		t.Errorf("played %d years: want the case stopped after the first", r.Years)
	}
}

// differs names the first differing paths of two games, and nothing for
// equal ones.
func TestDiffers(t *testing.T) {
	c := Case{Seed: 5, Rules: "elegy", Opponents: []string{"rototill"}, Size: newgame.Tiny}
	a, err := c.newGame()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := c.newGame()
	if d := differs(a, b); d != "" {
		t.Fatalf("equal games differ: %s", d)
	}
	b.State.Players[1].ResearchBudget++
	if d := differs(a, b); !strings.Contains(d, "ResearchBudget") {
		t.Fatalf("differs = %q", d)
	}
}

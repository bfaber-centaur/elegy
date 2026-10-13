package engine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/internal/vectors"
)

// The battle-record comparison is in battle_parity_test.go, inside
// package engine; these tests generate the vectors' years, which needs
// package vectors.

// battleStarts generates the years of the vector at path ("cb/cb010.json")
// and returns each year's battle-phase game, indexed by year (index 0 is
// unused).
func battleStarts(t *testing.T, path string) []engine.BattleStart {
	return battleStartsVariant(t, path, 0)
}

// battleStartsVariant is battleStarts with seed variant k.
func battleStartsVariant(t *testing.T, path string, k int) []engine.BattleStart {
	t.Helper()
	v, err := vectors.ReadFile(filepath.Join("testdata", "vectors", filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	s, err := v.Start()
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if s.Skip != "" {
		return []engine.BattleStart{{}, {Skip: s.Skip}}
	}
	out := []engine.BattleStart{{}}
	// drawn counts the draws of the years generated so far: a year's
	// battles that follow a draw of an earlier year start from a game
	// those draws shaped (BattleStart.Downstream).
	drawn := &countRand{}
	turn := func(g engine.Game, orders []engine.PlayerOrders, rng engine.Rand) (engine.TurnResult, error) {
		before := drawn.n
		drawn.rng = rng
		return engine.GenerateTurnAtBattles(g, orders, drawn, func(g engine.Game, costLevels [][engine.NumFields]int) {
			out = append(out, engine.BattleStart{Game: g, CostLevels: costLevels, Fleets: s.FleetsIn(&g), Planets: s.Planets, Downstream: before, InYear: drawn.n - before})
		})
	}
	vectors.Generate(v, k, turn)
	return out
}

// countRand passes draws to rng and counts them.
type countRand struct {
	rng engine.Rand
	n   int
}

func (c *countRand) Intn(n int) int {
	c.n++
	return c.rng.Intn(n)
}

func TestBattleRecords(t *testing.T) {
	starts := map[string][]engine.BattleStart{}
	for _, path := range engine.BattleRecordVectors(t) {
		starts[path] = battleStarts(t, path)
	}
	// RG-002's and RG-003's records go to their ledgers
	// (battle_ledger_test.go).
	l := &ledgerTB{TB: t, lines: map[string]string{}}
	ungen := engine.CheckBattleRecords(l, starts, battleVariants(t))
	for corpus, n := range ungen {
		if n != 0 {
			t.Errorf("%s: %d records of a year not generated", corpus, n)
		}
	}
	// Each representative game's lines go to its own ledger.
	for _, path := range []string{rgBattleLedgers["rg002"], rgBattleLedgers["rg003"]} {
		got := map[string]string{}
		for k, class := range l.lines {
			if rgBattleLedger(k) == path {
				got[k] = class
			}
		}
		ledger := readBattleLedger(t, path)
		if os.Getenv("RG002_LEDGER") == "write" {
			writeBattleLedger(t, path, got, ledger)
			ledger = readBattleLedger(t, path)
		}
		checkBattleLedger(t, got, ledger)
	}
	checkRGBattleCoverage(t, l.lines, "rg002", rg002BattleRecords)
	checkRGBattleCoverage(t, l.lines, "rg003", rg003BattleRecords)
}

// checkRGBattleCoverage fails unless every battle record of the
// representative game named by its chained run (rg002, rg003) has both its
// expectations and at least one ledger line: the internal/vectors ledger
// leaves the `battle` and `battle_actions` expectations to these lines,
// so none may go unreported. want is the game's number of records.
func checkRGBattleCoverage(t *testing.T, lines map[string]string, game string, want int) {
	covered := map[string]bool{}
	for k := range lines {
		parts := strings.SplitN(k, " | ", 4)
		covered[strings.Join(parts[:3], " | ")] = true
	}
	n := 0
	for _, path := range engine.BattleRecordVectors(t) {
		vector, ok := strings.CutPrefix(strings.TrimSuffix(path, ".json"), "rg/")
		if !ok || !strings.HasPrefix(vector, game) {
			continue
		}
		for k, lacks := range engine.BattleRecordKeys(t, path) {
			n++
			if lacks != "" {
				t.Errorf("%s battle record %s %s: %s", game, vector, k, lacks)
			}
			if !covered[vector+" | "+k] {
				t.Errorf("%s battle record %s %s has no ledger line", game, vector, k)
			}
		}
	}
	if n != want {
		t.Errorf("%d %s battle records, want %d", n, game, want)
	}
}

// rg002BattleRecords is the number of battle records in the RG-002
// vectors: 44 in the chained run and 44 in the restarts, each a `battle`
// and a `battle_actions` expectation.
const rg002BattleRecords = 88

// rg003BattleRecords is the number of battle records in the RG-003
// vectors: 102 in the chained run and 102 in the restarts.
const rg003BattleRecords = 204

func TestBattleRecordHarness(t *testing.T) {
	engine.CheckBattleRecordHarness(t, battleStarts(t, "cb/cb010.json")[1], battleStarts(t, "cs/cs003c.json")[1])
}

// RG-001 2435: the Orbital Forts at the vector's planets 54 and 55 keep
// the damage the replayed battles give them (360 and 180, as in the
// original). Elegy's own battles differ only in the attacker's random
// walk (COMBAT.md "Disengaging").
func TestRG001StarbaseDamageKept(t *testing.T) {
	engine.CheckStarbaseDamageKept(t, battleStarts(t, "rg/rg001-y6.json")[1])
}

// A chained run's years after Elegy's own draws start downstream of them:
// RG-001 draws in its first year, so its later years' battles are
// samples; CB-010's single year and each per-year restart are not.
func TestBattleStartDownstream(t *testing.T) {
	chain := battleStarts(t, "rg/rg001.json")
	if len(chain) < 3 || chain[1].Downstream != 0 || chain[2].Downstream == 0 {
		t.Errorf("RG-001 chain: %d years, downstream draws %d (year 1), want 0 then more", len(chain)-1, chain[1].Downstream)
	}
	for i := 2; i < len(chain); i++ {
		if chain[i].Downstream <= chain[i-1].Downstream {
			t.Errorf("RG-001 year %d: %d draws before it, year %d: %d", i, chain[i].Downstream, i-1, chain[i-1].Downstream)
		}
	}
	for _, p := range []string{"cb/cb010.json", "rg/rg001-y6.json"} {
		if s := battleStarts(t, p); s[1].Downstream != 0 {
			t.Errorf("%s year 1: %d draws before it, want 0", p, s[1].Downstream)
		}
	}
}

// recordTB keeps CheckBattleRecords' errors and log lines.
type recordTB struct {
	testing.TB
	errs, logs []string
}

func (r *recordTB) Errorf(format string, args ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, args...))
}

func (r *recordTB) Logf(format string, args ...any) {
	r.logs = append(r.logs, fmt.Sprintf(format, args...))
}

// A sample that misses is logged, not failed: with every fleet removed
// from Elegy's battle-phase game of RG-001's last chained year, each of
// that year's battles misses as a sample, and the same game taken as a
// start with no earlier draws fails each of them instead.
func TestBattleRecordSampleMissIsNotFailure(t *testing.T) {
	const path = "rg/rg001.json"
	chain := battleStarts(t, path)
	last := len(chain) - 1
	marker := fmt.Sprintf(" y%d: ", last)
	// samples is the rg tally line's samples column, matched/compared.
	samples := func(r *recordTB) string {
		got := ""
		for _, l := range r.logs {
			if f := strings.Fields(l); len(f) > 0 && f[0] == "rg" {
				// The last matched/compared column is samples.
				for _, c := range f {
					if strings.Contains(c, "/") {
						got = c
					}
				}
			}
		}
		return got
	}
	var tally string
	run := func(downstream int) (failed, missed int) {
		starts := append([]engine.BattleStart(nil), chain...)
		starts[last].Game.Fleets = nil
		starts[last].Downstream = downstream
		r := &recordTB{TB: t}
		engine.CheckBattleRecords(r, map[string][]engine.BattleStart{path: starts}, engine.BattleVariants{})
		tally = samples(r)
		for _, e := range r.errs {
			if strings.Contains(e, marker) {
				failed++
			}
		}
		for _, l := range r.logs {
			if strings.Contains(l, marker) && strings.Contains(l, "missed") {
				missed++
			}
		}
		return failed, missed
	}
	if chain[last].Downstream == 0 {
		t.Fatalf("RG-001 year %d follows no draws", last)
	}
	r := &recordTB{TB: t}
	engine.CheckBattleRecords(r, map[string][]engine.BattleStart{path: chain}, engine.BattleVariants{})
	if got := samples(r); got != "5/7" {
		t.Errorf("unmodified chain: samples %s, want 5/7", got)
	}
	failed, missed := run(chain[last].Downstream)
	if failed != 0 || missed == 0 {
		t.Errorf("as a sample: %d failures, %d misses, want 0 and some", failed, missed)
	}
	// The last year's samples all miss now; the earlier years' still
	// match as before.
	if want := "3/7"; tally != want {
		t.Errorf("as a sample: samples %s, want %s", tally, want)
	}
	if f, m := run(0); f != missed || m != 0 {
		t.Errorf("with no earlier draws: %d failures, %d misses, want %d and 0", f, m, missed)
	}
}

// battleVariants generates a vector's other seed variants when
// CheckBattleRecords asks for them, once each.
func battleVariants(t *testing.T) engine.BattleVariants {
	cache := map[string][]engine.BattleStart{}
	return engine.BattleVariants{N: vectors.Seeds, Start: func(name string, k int) []engine.BattleStart {
		key := fmt.Sprintf("%s#%d", name, k)
		if s, ok := cache[key]; ok {
			return s
		}
		s := battleStartsVariant(t, name, k)
		cache[key] = s
		return s
	}}
}

// A restart's record that differs after the year's own draws is random
// when another seed variant matches it, and fails when none does or when
// the year drew nothing before its battles. RG-002-Y2's battle at
// (1208,1327) is taken out of the reference variant's game, as a mine
// hit on its fleets would; the other variants keep it.
func TestBattleRecordRandomInYear(t *testing.T) {
	const path = "rg/rg002-y2.json"
	const marker = " (1208,1327) y1: "
	base := battleStarts(t, path)
	if base[1].InYear == 0 {
		t.Fatalf("RG-002-Y2 draws nothing before its battles")
	}
	vs := battleVariants(t)
	run := func(inYear int, vs engine.BattleVariants) (failed, random int) {
		starts := append([]engine.BattleStart(nil), base...)
		g := starts[1].Game
		g.Fleets = nil
		for _, f := range base[1].Game.Fleets {
			if f.Pos != (engine.Point{X: 1208, Y: 1327}) {
				g.Fleets = append(g.Fleets, f)
			}
		}
		starts[1].Game, starts[1].InYear = g, inYear
		r := &recordTB{TB: t}
		engine.CheckBattleRecords(r, map[string][]engine.BattleStart{path: starts}, vs)
		for _, e := range r.errs {
			if strings.Contains(e, marker) {
				failed++
			}
		}
		for _, l := range r.logs {
			if strings.Contains(l, marker) && strings.Contains(l, "random") {
				random++
			}
		}
		return failed, random
	}
	if f, r := run(base[1].InYear, vs); f != 0 || r != 1 {
		t.Errorf("after the year's draws: %d failures, %d random, want 0 and 1", f, r)
	}
	if f, r := run(0, vs); f != 1 || r != 0 {
		t.Errorf("with no draws in the year: %d failures, %d random, want 1 and 0", f, r)
	}
	// only keeps variant k's game; the others lose every fleet.
	only := func(k int) engine.BattleVariants {
		return engine.BattleVariants{N: vs.N, Start: func(name string, j int) []engine.BattleStart {
			s := append([]engine.BattleStart(nil), vs.Start(name, j)...)
			if j != k {
				s[1].Game.Fleets = nil
			}
			return s
		}}
	}
	if f, r := run(base[1].InYear, only(-1)); f != 1 || r != 0 {
		t.Errorf("no variant matching: %d failures, %d random, want 1 and 0", f, r)
	}
	for _, k := range []int{1, vs.N - 1} {
		if f, r := run(base[1].InYear, only(k)); f != 0 || r != 1 {
			t.Errorf("only variant %d matching: %d failures, %d random, want 0 and 1", k, f, r)
		}
	}
}

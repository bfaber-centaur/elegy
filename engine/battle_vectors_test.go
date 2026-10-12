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
			out = append(out, engine.BattleStart{Game: g, CostLevels: costLevels, Fleets: s.FleetsIn(&g), Planets: s.Planets, Downstream: before})
		})
	}
	vectors.Generate(v, 0, turn)
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
	// RG-002's records go to its ledger (battle_ledger_test.go).
	l := &ledgerTB{TB: t, lines: map[string]string{}}
	ungen := engine.CheckBattleRecords(l, starts)
	for corpus, n := range ungen {
		if n != 0 {
			t.Errorf("%s: %d records of a year not generated", corpus, n)
		}
	}
	ledger := readBattleLedger(t)
	if os.Getenv("RG002_LEDGER") == "write" {
		writeBattleLedger(t, l.lines, ledger)
		ledger = readBattleLedger(t)
	}
	checkBattleLedger(t, l.lines, ledger)
}

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
				got = f[len(f)-1]
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
		engine.CheckBattleRecords(r, map[string][]engine.BattleStart{path: starts})
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
	engine.CheckBattleRecords(r, map[string][]engine.BattleStart{path: chain})
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

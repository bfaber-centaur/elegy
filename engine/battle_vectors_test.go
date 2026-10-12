package engine_test

import (
	"path/filepath"
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
	turn := func(g engine.Game, orders []engine.PlayerOrders, rng engine.Rand) (engine.TurnResult, error) {
		return engine.GenerateTurnAtBattles(g, orders, rng, func(g engine.Game, costLevels [][engine.NumFields]int) {
			out = append(out, engine.BattleStart{Game: g, CostLevels: costLevels, Fleets: s.FleetsIn(&g), Planets: s.Planets})
		})
	}
	vectors.Generate(v, 0, turn)
	return out
}

func TestBattleRecords(t *testing.T) {
	starts := map[string][]engine.BattleStart{}
	for _, path := range engine.BattleRecordVectors(t) {
		starts[path] = battleStarts(t, path)
	}
	ungen := engine.CheckBattleRecords(t, starts)
	for corpus, n := range ungen {
		if n != 0 {
			t.Errorf("%s: %d records of a year not generated", corpus, n)
		}
	}
}

func TestBattleRecordHarness(t *testing.T) {
	engine.CheckBattleRecordHarness(t, battleStarts(t, "cb/cb010.json")[1], battleStarts(t, "cs/cs003c.json")[1])
}

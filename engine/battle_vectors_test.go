package engine_test

import (
	"path/filepath"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/internal/vectors"
)

// The battle-record comparison is in battle_parity_test.go, inside
// package engine; these tests load the vectors' starts, which needs
// package vectors.

// battleStart builds the start of the vector at path ("cb/cb010.json").
func battleStart(t *testing.T, path string) engine.BattleStart {
	t.Helper()
	v, err := vectors.ReadFile(filepath.Join("testdata", "vectors", filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	s, err := v.Start()
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return engine.BattleStart{Game: s.Game, Fleets: s.Fleets, Planets: s.Planets, Skip: s.Skip}
}

func TestBattleRecords(t *testing.T) {
	starts := map[string]engine.BattleStart{}
	for _, path := range engine.BattleRecordVectors(t) {
		starts[path] = battleStart(t, path)
	}
	// RG-001 (rg/rg001.json) chains six years; its battles after the first
	// year need that year's generated start, which this test does not
	// build, so they are counted apart.
	later := engine.CheckBattleRecords(t, starts)
	if later["rg"] != 7 || later["cb"] != 0 {
		t.Errorf("later-year records by corpus: %v, want rg 7 and none in cb", later)
	}
}

func TestBattleRecordHarness(t *testing.T) {
	engine.CheckBattleRecordHarness(t, battleStart(t, "cb/cb010.json"), battleStart(t, "cs/cs003c.json"))
}

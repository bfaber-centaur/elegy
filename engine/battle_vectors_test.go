package engine_test

import (
	"path/filepath"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/internal/vectors"
)

func init() {
	engine.BattleStartOf = func(path string) (engine.BattleStart, error) {
		v, err := vectors.ReadFile(filepath.Join("testdata", "vectors", filepath.FromSlash(path)))
		if err != nil {
			return engine.BattleStart{}, err
		}
		s, err := v.Start()
		return engine.BattleStart{Game: s.Game, Fleets: s.Fleets, Planets: s.Planets, Skip: s.Skip}, err
	}
}

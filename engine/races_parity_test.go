package engine_test

import (
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/races"
)

// The parity harness's race check (KERNEL.md "Turn order" step 2a): each
// vector player's race with its wizard settings. This file is in package
// engine_test because package races imports the engine.

func init() {
	engine.SetParityRaces(func(g *engine.Game, settings []engine.PVRaceSettings) engine.RaceChecker {
		r := &races.GameRaces{}
		for i, s := range settings {
			r.Designs = append(r.Designs, races.Design{Race: g.Players[i].Race, ExpensiveAt3: s.ExpensiveAt3, Spend: s.Spend, Stat15: s.Stat15})
			r.Computer = append(r.Computer, s.Computer)
		}
		return r
	})
	engine.SetParityRaceSettings(func(c engine.RaceChecker, player int) (engine.PVRaceSettings, bool) {
		r, ok := c.(*races.GameRaces)
		if !ok || player >= len(r.Designs) {
			return engine.PVRaceSettings{}, false
		}
		d := r.Designs[player]
		return engine.PVRaceSettings{ExpensiveAt3: d.ExpensiveAt3, Spend: d.Spend, Stat15: d.Stat15, Computer: r.Computer[player]}, true
	})
}

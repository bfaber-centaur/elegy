package newgame

import (
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/objects"
)

// makeWormholes creates the wormholes, after the planets and players.
// Only games with random events have any (OBJECTS.md "Creation",
// UNIVERSE.md "Space objects at the start", CONFIRMED): rand(v) + m pairs
// by universe size, each end with its own stability class rand(3), 0
// years, and a position from up to 100 uniform tries (objects.Place).
func (g *generator) makeWormholes() {
	if g.s.NoRandomEvents {
		return
	}
	vm := objects.WormholePairs[g.s.Size]
	n := g.rand(vm[0]) + vm[1]
	for range n {
		var wh Wormhole
		for e := range wh.Ends {
			// ELEGY CHOICE: an end's class is drawn before its
			// position.
			wh.Ends[e].Class = g.rand(3)
			var partner *engine.Point
			if e == 1 {
				partner = &wh.Ends[0].Pos
			}
			wh.Ends[e].Pos = g.surroundings(partner).Place(objects.UniformTry(g.w, g.rng), nil)
		}
		g.res.Wormholes = append(g.res.Wormholes, wh)
	}
}

// surroundings are what a new end is judged against: the wormholes
// placed so far, every planet and fleet (OBJECTS.md "Placement badness",
// CONFIRMED at creation, UG01–UG21).
func (g *generator) surroundings(partner *engine.Point) objects.Surroundings {
	sr := objects.Surroundings{Width: g.w, Partner: partner}
	for _, wh := range g.res.Wormholes {
		for _, e := range wh.Ends {
			sr.Ends = append(sr.Ends, e.Pos)
		}
	}
	for _, pl := range g.res.Game.Planets {
		sr.Planets = append(sr.Planets, pl.Pos)
	}
	for _, f := range g.res.Game.Fleets {
		sr.Objects = append(sr.Objects, f.Pos)
	}
	return sr
}

func (g *generator) wormholeBadness(p engine.Point, partner *engine.Point) int {
	return g.surroundings(partner).Badness(p)
}

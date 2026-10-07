package newgame

import "github.com/bfaber-centaur/elegy/engine"

// wormholePairs is (v, m) by universe size: rand(v) + m pairs
// (OBJECTS.md "Wormholes: Creation", CONFIRMED OB-006, UG01–UG21).
var wormholePairs = [5][2]int{{3, 0}, {3, 1}, {5, 1}, {4, 3}, {5, 4}}

// wormholeTries is the number of uniform position tries per end.
const wormholeTries = 100

// makeWormholes creates the wormholes, after the planets and players.
// Only games with random events have any (OBJECTS.md "Creation",
// UNIVERSE.md "Space objects at the start", CONFIRMED).
func (g *generator) makeWormholes() {
	if g.s.NoRandomEvents {
		return
	}
	vm := wormholePairs[g.s.Size]
	n := g.rand(vm[0]) + vm[1]
	for range n {
		var wh Wormhole
		for e := range wh.Ends {
			// Each end gets its own stability class rand(3) and 0
			// years. ELEGY CHOICE: an end's class is drawn before its
			// position.
			wh.Ends[e].Stability = g.rand(3)
			var partner *engine.Point
			if e == 1 {
				partner = &wh.Ends[0].Pos
			}
			wh.Ends[e].Pos = g.placeWormholeEnd(partner)
		}
		g.res.Wormholes = append(g.res.Wormholes, wh)
	}
}

// placeWormholeEnd tries up to 100 uniform positions over the galaxy and
// keeps the first with no badness, else the least bad (OBJECTS.md
// "Placement badness", BINARY-ONLY). A try exactly on a planet, fleet or
// other wormhole end is rejected.
//
// ELEGY CHOICE: the spec lists the badness terms but not how they
// combine. Elegy scores each term by its band (worst band 4 for the
// partner, wormholes and planets, 1 for an edge) and sums them; ties keep
// the earliest try. "Within 10 ly of an edge" is read as less than 10 ly.
// If every try is rejected outright, the last try is used.
func (g *generator) placeWormholeEnd(partner *engine.Point) engine.Point {
	var best engine.Point
	bestBad := -1
	for range wormholeTries {
		p := engine.Point{X: Origin + g.rand(g.w+1), Y: Origin + g.rand(g.w+1)}
		bad, ok := g.wormholeBadness(p, partner)
		if !ok {
			if bestBad < 0 {
				best = p
			}
			continue
		}
		if bad == 0 {
			return p
		}
		if bestBad < 0 || bad < bestBad {
			best, bestBad = p, bad
		}
	}
	return best
}

func band(d int, limits [4]int) int {
	for i, l := range limits {
		if d < l {
			return 4 - i
		}
	}
	return 0
}

func (g *generator) wormholeBadness(p engine.Point, partner *engine.Point) (int, bool) {
	game := &g.res.Game
	for _, pl := range game.Planets {
		if pl.Pos == p {
			return 0, false
		}
	}
	for _, f := range game.Fleets {
		if f.Pos == p {
			return 0, false
		}
	}
	var ends []engine.Point
	for _, wh := range g.res.Wormholes {
		for _, e := range wh.Ends {
			ends = append(ends, e.Pos)
		}
	}
	for _, e := range ends {
		if e == p {
			return 0, false
		}
	}
	if partner != nil && *partner == p {
		return 0, false
	}

	bad := 0
	lo, hi := Origin, Origin+g.w
	if p.X-lo < 10 || hi-p.X < 10 || p.Y-lo < 10 || hi-p.Y < 10 {
		bad++
	}
	if partner != nil {
		bad += band(d2(p, *partner), [4]int{25, 100, 900, 4900})
	}
	worst := 0
	for _, e := range ends {
		worst = max(worst, band(d2(p, e), [4]int{16, 64, 225, 900}))
	}
	bad += worst
	worst = 0
	for _, pl := range game.Planets {
		worst = max(worst, band(d2(p, pl.Pos), [4]int{25, 100, 400, 784}))
	}
	return bad + worst, true
}

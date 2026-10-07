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

// placeWormholeEnd tries up to 100 positions 1000 + rand(W) on each axis
// and keeps the first try with the lowest badness (OBJECTS.md "Placement
// badness", CONFIRMED at creation, UG01–UG21). Badness is a set of four
// flags OR-ed together and compared as a number 0–15; a try exactly on a
// planet, fleet or other wormhole end is rejected and counts as 15.
func (g *generator) placeWormholeEnd(partner *engine.Point) engine.Point {
	var best engine.Point
	bestBad := 16
	for range wormholeTries {
		p := engine.Point{X: Origin + g.rand(g.w), Y: Origin + g.rand(g.w)}
		bad := g.wormholeBadness(p, partner)
		if bad < bestBad {
			best, bestBad = p, bad
		}
		if bad == 0 {
			break
		}
	}
	return best
}

// rejected is the badness of a try that lands exactly on an object.
const rejected = 15

// bandFlag is the flag a distance band sets: 8 for the closest band, then
// 4, 2, 1; 0 outside every band.
func bandFlag(d int, limits [4]int) int {
	for i, l := range limits {
		if d < l {
			return 8 >> i
		}
	}
	return 0
}

func (g *generator) wormholeBadness(p engine.Point, partner *engine.Point) int {
	game := &g.res.Game
	var ends []engine.Point
	for _, wh := range g.res.Wormholes {
		for _, e := range wh.Ends {
			ends = append(ends, e.Pos)
		}
	}
	for _, pl := range game.Planets {
		if pl.Pos == p {
			return rejected
		}
	}
	for _, f := range game.Fleets {
		if f.Pos == p {
			return rejected
		}
	}
	for _, e := range ends {
		if e == p {
			return rejected
		}
	}
	if partner != nil && *partner == p {
		return rejected
	}

	bad := 0
	// Within 10 ly of an edge: x or y below 1010 or above 1000 + W − 10.
	lo, hi := Origin+10, Origin+g.w-10
	if p.X < lo || p.X > hi || p.Y < lo || p.Y > hi {
		bad |= 4
	}
	if partner != nil {
		bad |= bandFlag(d2(p, *partner), [4]int{25, 100, 900, 4900})
	}
	for _, e := range ends {
		bad |= bandFlag(d2(p, e), [4]int{16, 64, 225, 900})
	}
	for _, pl := range game.Planets {
		bad |= bandFlag(d2(p, pl.Pos), [4]int{25, 100, 400, 784})
	}
	return bad
}

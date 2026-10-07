package newgame

import (
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// hwPicks is the number of random picks before a fallback (UNIVERSE.md
// "Homeworld placement", CONFIRMED).
const hwPicks = 50

// box is an inclusive square region of the galaxy.
type box struct{ lo, hi int }

func (b box) contains(p engine.Point) bool {
	return p.X >= b.lo && p.X <= b.hi && p.Y >= b.lo && p.Y <= b.hi
}

// dist2 is d² from p to the nearest point of the box.
func (b box) dist2(p engine.Point) int {
	axis := func(v int) int {
		switch {
		case v < b.lo:
			return b.lo - v
		case v > b.hi:
			return v - b.hi
		}
		return 0
	}
	dx, dy := axis(p.X), axis(p.Y)
	return dx*dx + dy*dy
}

// spacing are the homeworld distance limits, as d².
type spacing struct {
	v        int // the spacing unit
	min, max int
}

// homeworldSpacing is v and its limits for P players, player positions s
// and width W (UNIVERSE.md "Homeworld placement", CONFIRMED):
// a = W²/P − 6W, 0 if negative, else a·9/10; v = 6W + s·a/3;
// min 9v/10, max 7v/6.
func homeworldSpacing(w, players int, pos Positions) spacing {
	a := w*w/players - 6*w
	if a < 0 {
		a = 0
	} else {
		a = a * 9 / 10
	}
	v := 6*w + int(pos)*a/3
	return spacing{v: v, min: 9 * v / 10, max: 7 * v / 6}
}

// placeHomeworlds returns one planet index per player, in player order
// (UNIVERSE.md "Homeworld placement", CONFIRMED): the homeworld list is
// placed, then players are assigned to it in random order.
func (g *generator) placeHomeworlds() ([]int, error) {
	hws, _, err := g.homeworldList(len(g.s.Players))
	if err != nil {
		return nil, err
	}
	// Players are assigned in random order. ELEGY CHOICE: a
	// Fisher–Yates shuffle.
	for i := len(hws) - 1; i > 0; i-- {
		j := g.rand(i + 1)
		hws[i], hws[j] = hws[j], hws[i]
	}
	return hws, nil
}

// homeworldList places P homeworlds and returns them in placement order
// with the spacing limits they finally met.
func (g *generator) homeworldList(players int) ([]int, spacing, error) {
	planets := g.res.Game.Planets
	if len(planets) < players {
		return nil, spacing{}, fmt.Errorf("%w: %d planets for %d players", ErrNoPlacement, len(planets), players)
	}
	w := g.w
	sp := homeworldSpacing(w, players, g.s.Positions)
	step := sp.v / 35
	central := box{Origin + w/4, Origin + 3*w/4}
	var wide box
	switch {
	case players >= 5:
		wide = box{Origin + w/20, Origin + 19*w/20}
	case players >= 3:
		wide = box{Origin + w/10, Origin + 9*w/10}
	default:
		wide = box{Origin + 3*w/20, Origin + 17*w/20}
	}
	// Past these limits every planet in the wide box qualifies on
	// distance, so a further restart cannot help.
	farthest := 2 * w * w

	for {
		hws, ok := g.tryHomeworlds(players, central, wide, sp)
		if ok {
			return hws, sp, nil
		}
		if sp.min <= 0 && sp.max >= farthest {
			return nil, sp, fmt.Errorf("%w: too few planets in the placement box", ErrNoPlacement)
		}
		// No planet qualified: widen the limits by v/35 each and
		// restart from the first homeworld.
		sp.min -= step
		sp.max += step
		if step == 0 {
			sp.min, sp.max = 0, farthest
		}
	}
}

func (g *generator) tryHomeworlds(players int, central, wide box, sp spacing) ([]int, bool) {
	planets := g.res.Game.Planets
	n := len(planets)

	// The first homeworld: a random planet in the central box; after
	// 50 picks outside it, the pick nearest to the box by dx² + dy²
	// outside it; the first of equally near picks wins.
	first, best := -1, 0
	for range hwPicks {
		i := g.rand(n)
		d := central.dist2(planets[i].Pos)
		if d == 0 {
			first = i
			break
		}
		if first < 0 || d < best {
			first, best = i, d
		}
	}
	hws := []int{first}
	taken := map[int]bool{first: true}

	qualifies := func(i int) bool {
		p := planets[i].Pos
		if taken[i] || !wide.contains(p) {
			return false
		}
		near := false
		for _, h := range hws {
			d := d2(p, planets[h].Pos)
			if d < sp.min {
				return false
			}
			if d <= sp.max {
				near = true
			}
		}
		return near
	}

	for len(hws) < players {
		pick, last := -1, 0
		for range hwPicks {
			last = g.rand(n)
			if qualifies(last) {
				pick = last
				break
			}
		}
		if pick < 0 {
			// After 50 failed picks, the list is scanned from the
			// planet after the 50th pick, wrapping, and stops on
			// returning to that pick, which is not tested again.
			for k := 1; k < n; k++ {
				if i := (last + k) % n; qualifies(i) {
					pick = i
					break
				}
			}
		}
		if pick < 0 {
			return nil, false
		}
		hws = append(hws, pick)
		taken[pick] = true
	}
	return hws, true
}

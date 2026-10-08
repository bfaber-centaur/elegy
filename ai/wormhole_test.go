package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestPreferWormhole is AI.md §11's wormhole preference, with the 16-bit
// distance wrap (LEGACY BUG, "Wormhole distance arithmetic") on and off.
// The fleet is at (1000, 1000); ends lie along x. A draw of 89 takes a
// score-90 end and refuses a score-50 one; a draw of 49 takes either.
func TestPreferWormhole(t *testing.T) {
	from := engine.Point{X: 1000, Y: 1000}
	at := func(ly int) engine.Point { return engine.Point{X: 1000 + ly, Y: 1000} }
	end := func(id, ly int) Wormhole { return Wormhole{End: id, Pos: at(ly)} }
	cases := []struct {
		name   string
		wrap   bool
		ends   []Wormhole
		cand   int // candidate distance in ly; -1 for no candidate
		draws  []int
		want   int // End taken; -1 for none
		nodraw bool
	}{
		// 181 ly: 32,761 fits 16 bits, so both modes agree.
		{"181 near, wrap", true, []Wormhole{end(1, 181)}, 100, []int{89}, -1, false},
		{"181 near, wrap, 50", true, []Wormhole{end(1, 181)}, 100, []int{49}, 1, false},
		{"181 far, wrap", true, []Wormhole{end(1, 181)}, 10, nil, -1, true},
		{"181 far, exact", false, []Wormhole{end(1, 181)}, 10, nil, -1, true},
		// 182 ly: 33,124 wraps to −32,412, so the end passes both tests.
		{"182, wrap", true, []Wormhole{end(1, 182)}, 10, []int{89}, 1, false},
		{"182, exact", false, []Wormhole{end(1, 182)}, 10, nil, -1, true},
		// 255 ly: 65,025 wraps to −511.
		{"255, wrap", true, []Wormhole{end(1, 255)}, 10, []int{89}, 1, false},
		{"255, exact", false, []Wormhole{end(1, 255)}, 10, nil, -1, true},
		// 256 ly: 65,536 wraps to 0, still ≤ D.
		{"256, wrap", true, []Wormhole{end(1, 256)}, 10, []int{89}, 1, false},
		{"256, exact", false, []Wormhole{end(1, 256)}, 10, nil, -1, true},
		// 257 ly: 66,049 wraps to 513, past 4·D = 400.
		{"257, wrap", true, []Wormhole{end(1, 257)}, 10, nil, -1, true},
		// w = D scores 90; w = 4·D is considered.
		{"w = D", false, []Wormhole{end(1, 100)}, 100, []int{89}, 1, false},
		{"w = 4D", false, []Wormhole{end(1, 100)}, 50, []int{49}, 1, false},
		{"w = 4D, 90 refused", false, []Wormhole{end(1, 100)}, 50, []int{89}, -1, false},
		// No candidate: D = 99,999,999; both ends score 90 and the smaller
		// w wins: 10,000 exact, −25,536 (200 ly) wrapped.
		{"no candidate, exact", false, []Wormhole{end(1, 100), end(2, 200)}, -1, []int{89}, 1, false},
		{"no candidate, wrap", true, []Wormhole{end(1, 100), end(2, 200)}, -1, []int{89}, 2, false},
		{"no candidate, exact, order", false, []Wormhole{end(2, 200), end(1, 100)}, -1, []int{89}, 1, false},
		// A known end scores by its class, whatever w is.
		{"known, wrap", true, []Wormhole{{End: 1, Pos: at(200), Known: true, Class: 3}}, 10, []int{39}, 1, false},
		{"known, wrap, refused", true, []Wormhole{{End: 1, Pos: at(200), Known: true, Class: 3}}, 10, []int{40}, -1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := &View{Wormholes: c.ends}
			v.Rules.Legacy.AIWormholeDistanceWrap = c.wrap
			target, dd := 7, c.cand*c.cand
			if c.cand < 0 {
				target, dd = -1, 0
			}
			rng := &script{t: t, draws: c.draws}
			w, ok := v.preferWormhole(0, rng, from, target, dd)
			got := -1
			if ok {
				got = w.End
			}
			if got != c.want {
				t.Errorf("took end %d, want %d", got, c.want)
			}
			if c.nodraw != (len(rng.bounds) == 0) {
				t.Errorf("draws %v, want none: %v", rng.bounds, c.nodraw)
			}
			if len(rng.draws) != 0 {
				t.Errorf("unused draws %v", rng.draws)
			}
		})
	}
}

// TestPreferWormholeBuiltins: the wrap is on in both current built-in
// rulesets and off in the versions before it (docs/RULESET.md), so an
// unknown end 182 ly away is taken under the first and never considered
// under the second.
func TestPreferWormholeBuiltins(t *testing.T) {
	for _, r := range engine.Rulesets() {
		v := &View{Rules: r, Wormholes: []Wormhole{{End: 1, Pos: engine.Point{X: 1182, Y: 1000}}}}
		cur := r == engine.ElegyRules() || r == engine.FaithfulRules()
		var draws []int
		if cur {
			draws = []int{89}
		}
		_, ok := v.preferWormhole(0, &script{t: t, draws: draws}, engine.Point{X: 1000, Y: 1000}, 7, 100)
		if ok != cur {
			t.Errorf("%s v%d: took %v, want %v", r.ID, r.Version, ok, cur)
		}
	}
}

// TestPreferWormholeYear: only before year index 120 (AI.md §11).
func TestPreferWormholeYear(t *testing.T) {
	v := &View{Wormholes: []Wormhole{{End: 1, Pos: engine.Point{X: 1100, Y: 1000}}}}
	from := engine.Point{X: 1000, Y: 1000}
	if w, ok := v.preferWormhole(119, &script{t: t, draws: []int{89}}, from, -1, 0); !ok || w.End != 1 {
		t.Errorf("year 119: took %v %v, want end 1", w.End, ok)
	}
	if _, ok := v.preferWormhole(120, &script{t: t}, from, -1, 0); ok {
		t.Error("year 120: took a wormhole")
	}
}

package engine

import (
	"reflect"
	"testing"
)

// TestTurnHostileShrink checks the hostile-shrink message (MESSAGES.md
// 0x025 at stars-elegy de1bd6f, CONFIRMED cb5/cb037-owner, ob/ob007): a
// planet with a hab value below 0 for its owner that shrinks and survives
// reports its population before and after; an overcrowded planet with a
// hab value of 0 or more (0x026 instead) and one that dies out do not.
func TestTurnHostileShrink(t *testing.T) {
	for _, c := range []struct {
		name string
		env  [3]int
		pop  int
		want bool
	}{
		{"hostile", [3]int{0, 50, 50}, 20000, true},
		{"hostile, small", [3]int{0, 50, 50}, 486, true},
		{"hab 100, over capacity", [3]int{50, 50, 50}, 20000, false},
		{"hostile, dies out", [3]int{0, 0, 0}, 1, false},
	} {
		g := pgHomeworld()
		g.Planets[0].Env, g.Planets[0].Population = c.env, c.pop
		r, err := GenerateTurn(withRules(g), nil, highRand{})
		if err != nil {
			t.Fatal(err)
		}
		end := r.Game.Planets[0].Population
		var got []Event
		for _, ev := range r.Events {
			if ev.Kind == EventHostileShrink {
				got = append(got, ev)
			}
		}
		if !c.want {
			if got != nil {
				t.Errorf("%s: population %d → %d: unexpected %+v", c.name, c.pop, end, got)
			}
			continue
		}
		want := []Event{{Kind: EventHostileShrink, Player: 0, Planet: 7, Fleet: -1, Count: c.pop, Axes: []int{end}}}
		if end <= 0 || end >= c.pop || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: population %d → %d: events %+v, want %+v", c.name, c.pop, end, got, want)
		}
	}
}

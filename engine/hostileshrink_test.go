package engine

import (
	"reflect"
	"testing"
)

// TestTurnHostileShrink checks the hostile-shrink message (MESSAGES.md
// 0x025 at stars-elegy de1bd6f, CONFIRMED cb5/cb037-owner, ob/ob007): a
// planet with a hab value below 0 for its owner that shrinks and survives
// reports its population before and after; an overcrowded planet with a
// hab value of 0 or more (0x026 instead) does not. Nor does a hostile
// planet whose growth takes it to 0, which is emptied: the 0x025 row's
// trigger is a population that "stays above 0" (ASSUMPTION: the row does
// not say a die-out was observed).
func TestTurnHostileShrink(t *testing.T) {
	for _, c := range []struct {
		name string
		env  [3]int
		pop  int
		// carry is the growth carry; -1 keeps the homeworld's.
		carry int
		want  bool
	}{
		{"hostile", [3]int{0, 50, 50}, 20000, -1, true},
		{"hostile, small", [3]int{0, 50, 50}, 486, -1, true},
		{"hab 100, over capacity", [3]int{50, 50, 50}, 20000, -1, false},
		// hab −45: 1 colonist loses trunc(45·1/10) = 4 hundredths, which
		// the carry of 0 cannot cover, so one colonist dies.
		{"hostile, dies out", [3]int{0, 0, 0}, 1, 0, false},
	} {
		g := pgHomeworld()
		g.Planets[0].Env, g.Planets[0].Population = c.env, c.pop
		if c.carry >= 0 {
			g.Planets[0].GrowthCarry = c.carry
		}
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
		if c.name == "hostile, dies out" && (end > 0 || r.Game.Planets[0].Owner != NoOwner) {
			t.Errorf("%s: population %d → %d, owner %d: want the planet emptied", c.name, c.pop, end, r.Game.Planets[0].Owner)
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

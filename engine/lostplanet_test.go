package engine

import (
	"reflect"
	"testing"
)

// TestConfirmedLostPlanetMessage checks the cause growth gives a planet it
// empties (MESSAGES.md 0x023, 0x040 and "0x040 vs 0x023", stars-elegy
// 898c526): a planet that dies out during growth is EmptiedDiedOut
// (0x023); one that already has no colonists is EmptiedLeft (0x040), or,
// under Legacy.LostPlanetMessageCarry, EmptiedLeftShownDiedOut (0x023)
// when the planet growth handled just before it shrank (LEGACY BUG,
// CONFIRMED MG-002: after a planet that shrank from overcrowding, 0x023;
// after one that grew, 0x040; as the first planet, 0x040). The rows with
// an unowned planet or a second empty planet between are ASSUMPTION
// (they leave the carried value alone), as are the hostile shrink and
// die-out rows (both count as shrinking).
func TestConfirmedLostPlanetMessage(t *testing.T) {
	type pl struct {
		env   [3]int
		pop   int
		carry int
		owned bool
	}
	var (
		left    = pl{[3]int{50, 50, 50}, 0, 0, true}
		crowded = pl{[3]int{50, 50, 50}, 20000, 80, true}
		grows   = pl{[3]int{50, 50, 50}, 486, 80, true}
		hostile = pl{[3]int{0, 50, 50}, 20000, 80, true}
		dies    = pl{[3]int{0, 0, 0}, 1, 0, true}
		unowned = pl{[3]int{50, 50, 50}, 0, 0, false}
	)
	for _, c := range []struct {
		name    string
		planets []pl
		on, off []int // the EventPlanetEmptied counts, switch on and off
	}{
		{"first planet (MG-002 planet 0)", []pl{left}, []int{EmptiedLeft}, []int{EmptiedLeft}},
		{"after overcrowding (MG-002 planet 3)", []pl{crowded, left}, []int{EmptiedLeftShownDiedOut}, []int{EmptiedLeft}},
		{"after growth (MG-002 planet 6)", []pl{grows, left}, []int{EmptiedLeft}, []int{EmptiedLeft}},
		{"after a hostile shrink", []pl{hostile, left}, []int{EmptiedLeftShownDiedOut}, []int{EmptiedLeft}},
		{"dies out", []pl{dies}, []int{EmptiedDiedOut}, []int{EmptiedDiedOut}},
		{"after a die-out", []pl{dies, left}, []int{EmptiedDiedOut, EmptiedLeftShownDiedOut}, []int{EmptiedDiedOut, EmptiedLeft}},
		{"unowned planet between", []pl{crowded, unowned, left}, []int{EmptiedLeftShownDiedOut}, []int{EmptiedLeft}},
		{"second empty planet", []pl{crowded, left, left}, []int{EmptiedLeftShownDiedOut, EmptiedLeftShownDiedOut}, []int{EmptiedLeft, EmptiedLeft}},
		{"growth after the shrink", []pl{crowded, grows, left}, []int{EmptiedLeft}, []int{EmptiedLeft}},
	} {
		for _, on := range []bool{true, false} {
			g := pgHomeworld()
			if !on {
				// Off is no built-in ruleset (every current one has it
				// on), so a custom one.
				g.Rules.ID, g.Rules.Legacy.LostPlanetMessageCarry = "custom", false
			}
			hw := g.Planets[0]
			g.Planets = nil
			for k, q := range c.planets {
				p := hw
				p.ID, p.Pos, p.Homeworld = 7+k, Point{1100 + 20*k, 1100}, k == 0
				p.Env, p.Population, p.GrowthCarry = q.env, q.pop, q.carry
				if !q.owned {
					p.Owner, p.Mines, p.Factories, p.Defenses = NoOwner, 0, 0, 0
				}
				g.Planets = append(g.Planets, p)
			}
			r, err := GenerateTurn(g, nil, highRand{})
			if err != nil {
				t.Fatal(err)
			}
			var got []int
			for _, ev := range r.Events {
				if ev.Kind == EventPlanetEmptied {
					got = append(got, ev.Count)
				}
			}
			want := c.off
			if on {
				want = c.on
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s, switch %v: emptied %v, want %v", c.name, on, got, want)
			}
		}
	}
}

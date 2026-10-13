package engine

import (
	"reflect"
	"testing"
)

func TestMeasuredRG003SapperSkipsUnshieldedPrimary(t *testing.T) {
	// RG-003 year 8 at (1571,1434) (stars-elegy b0f7dfa, vectors/rg/
	// rg003-y8.json): player 1's Meta Morphs (plan 0: primary armed,
	// secondary any) had 2 Pulsed Sappers each. In round 6 the only armed
	// enemy in sapper reach was a starbase stripped of its shields, and
	// the sappers hit an unarmed shielded minelayer 3 squares away
	// instead (shield damage 800). A target scoring 0 is never chosen, and
	// with no primary-type target scoring above 0 the secondary type is
	// used. MEASURED (one battle; with this rule its 63 recorded hits
	// replay), then CONFIRMED by prediction (CB-062 Z1b, control Z1c,
	// 6 of 6 streams, stars-elegy #194 head 188ee81). A shielded starbase
	// is still the sapper's target.
	armed := testDesign(tFrigate, 10, Slot{tLaser, 1})
	shielded := testDesign(Hull{Armor: 1000}, 10, Slot{Part{Name: "Shield", Kind: PartShield, Shield: 200}, 1})
	d := []Design{testDesign(tFrigate, 10, Slot{tSapper, 2}), armed, shielded}
	for _, c := range []struct {
		name     string
		sbShield int
		mShield  int
		want     []int // target, shield
	}{
		// 2 × 82 at distance 3 of range 3: 147.
		{"unshielded starbase", 0, 200, []int{2, 147}},
		// The starbase's 50, then the carry to the minelayer (CB-060 S2).
		{"shielded starbase", 50, 200, []int{1, 50, 2, 99}},
		// The primary type first even when the minelayer, with fewer
		// shields, would score higher.
		{"minelayer more attractive", 1000, 20, []int{1, 154}}, // distance 2: 154
	} {
		f := tok(d, 0, 0, 1)
		f.primary, f.secondary = TargetArmed, TargetAny
		sb := tokenValues(armed, Race{}, true, designCost(armed, Race{}, [NumFields]int{}))
		sb.player, sb.fleet, sb.planet, sb.design, sb.ships = 1, -1, 0, 1, 1
		sb.shield = c.sbShield
		sb.x = 2
		m := tok(d, 2, 1, 1)
		m.x, m.shield = 3, c.mShield
		tb := newTestBattle(panicRand{}, d, f, &sb, m)
		tb.beam(0, f.weapons[0])
		var got []int
		for _, h := range tb.hits {
			got = append(got, h.Target, h.Shield)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: hits (target, shield) %v, want %v", c.name, got, c.want)
		}
	}
}

package engine

import (
	"errors"
	"testing"
)

// A FleetShips carries the FuelWrap setting of the ruleset it was made
// under and of no other: under a wrap-on ruleset the under-engined
// freighter's fuel range is the wrapped one (FM-105: 7 ly from 200 mg at
// warp 5), under a wrap-off ruleset the exact one. An invalid ruleset is
// refused, and a FleetShips not made by NewFleetShips has no setting.
func TestFleetShipsFuelWrapFromRuleset(t *testing.T) {
	d := Design{Name: "Large Freighter", Hull: Hull{Name: "Large Freighter", Slots: []HullSlot{{Kinds: []PartKind{PartEngine}, Max: 2}}},
		Mass: 134, Engines: 1, FuelCapacity: 2600}
	d.Engine.Fuel = [11]int{0, 0, 0, 0, 0, 0, 100, 100, 100, 100, 100}
	stacks := []ShipStack{{Index: 0, Design: d, Count: 1}}

	exact := ElegyRules()
	exact.ID = "elegy-exact-fuel"
	exact.Legacy.FuelWrap = false
	wrapRange := 0
	for _, r := range append(Rulesets(), exact) {
		s, err := NewFleetShips(r, stacks, 0, false)
		if err != nil {
			t.Fatalf("%s v%d: %v", r.ID, r.Version, err)
		}
		if s.wrap() != r.Legacy.FuelWrap {
			t.Errorf("%s v%d: wrap %v, ruleset %v", r.ID, r.Version, s.wrap(), r.Legacy.FuelWrap)
		}
		got, _ := s.FuelRange(200, 5)
		if r.Legacy.FuelWrap {
			wrapRange = got
			if got != 7 {
				t.Errorf("%s v%d: range %d ly, want 7 (wrapped)", r.ID, r.Version, got)
			}
		} else if got == 7 {
			t.Errorf("%s v%d: wrap-off range is the wrapped 7 ly", r.ID, r.Version)
		}
	}
	if wrapRange == 0 {
		t.Fatal("no built-in ruleset has FuelWrap on")
	}

	if _, err := NewFleetShips(Ruleset{}, stacks, 0, false); !errors.Is(err, ErrNoRuleset) {
		t.Errorf("no ruleset: %v", err)
	}
	bad := ElegyRules()
	bad.Legacy.FuelWrap = false // claims elegy's ID and version with other settings
	if _, err := NewFleetShips(bad, stacks, 0, false); !errors.Is(err, ErrRuleset) {
		t.Errorf("altered built-in: %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Error("a zero FleetShips computed a fuel range")
		}
	}()
	_, _ = FleetShips{Stacks: stacks}.FuelRange(200, 5)
}

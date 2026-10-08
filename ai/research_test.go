package ai

import (
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

func TestResearchBudget(t *testing.T) {
	var zero, all24 [engine.NumFields]int
	for f := range all24 {
		all24[f] = 24
	}
	cases := []struct {
		pers   Personality
		y      int
		levels [engine.NumFields]int
		want   int
	}{
		{Robotoid, 9, zero, 0},
		{Robotoid, 10, zero, 15},
		{Rototill, 19, zero, 0},
		{Rototill, 20, zero, 15},
		{Cybertron, 0, zero, 17},
		{Cybertron, 80, all24, 0},
	}
	for _, c := range cases {
		if got := ResearchBudget(c.pers, c.y, c.levels); got != c.want {
			t.Errorf("%v y=%d: budget %d, want %d", c.pers, c.y, got, c.want)
		}
	}
}

func TestResearchPlan(t *testing.T) {
	// Robotoid's first goal is P2. Two levels away: the next field keeps
	// its setting, and the order is written although nothing changes.
	rs := engine.ResearchState{Current: engine.Propulsion, Next: engine.NextSameField}
	o, write := Research(Robotoid, 0, rs, 0)
	if !write || o != (engine.ResearchOrder{Budget: 0, Field: engine.Propulsion, Next: engine.NextSameField}) {
		t.Errorf("P2 two levels away: %+v %v", o, write)
	}
	// One level away and not the last goal: next is the following goal's
	// field (C3).
	rs.Levels[engine.Propulsion] = 1
	if o, _ := Research(Robotoid, 12, rs, 0); o != (engine.ResearchOrder{Budget: 15, Field: engine.Propulsion, Next: engine.Construction}) {
		t.Errorf("P2 one level away: %+v", o)
	}
	// Cybertron's last goal is B26: one level away, but the last, so the
	// next field keeps its setting.
	var near [engine.NumFields]int
	for f := range near {
		near[f] = 26
	}
	near[engine.Biotech] = 25
	rs = engine.ResearchState{Levels: near, Current: engine.Energy, Next: engine.NextLowestField}
	if o, write := Research(Cybertron, 90, rs, 17); !write || o != (engine.ResearchOrder{Budget: 0, Field: engine.Biotech, Next: engine.NextLowestField}) {
		t.Errorf("last goal: %+v %v", o, write)
	}
}

func TestResearchNoPlan(t *testing.T) {
	// Rototill: the lowest field, first in field order on ties; the order
	// is written only when the field (or, ASSUMPTION A1, the budget)
	// changes.
	rs := engine.ResearchState{Levels: [engine.NumFields]int{3, 3, 3, 3, 3, 0}, Current: engine.Energy, Next: engine.NextSameField}
	o, write := Research(Rototill, 0, rs, 0)
	if !write || o.Field != engine.Biotech || o.Next != engine.NextSameField {
		t.Errorf("field change: %+v %v", o, write)
	}
	rs.Current = engine.Biotech
	if _, write := Research(Rototill, 5, rs, 0); write {
		t.Error("no change: order written")
	}
	if o, write := Research(Rototill, 20, rs, 0); !write || o.Budget != 15 {
		t.Errorf("budget change at year index 20: %+v %v", o, write)
	}
	rs.Levels = [engine.NumFields]int{4, 2, 2, 4, 4, 4}
	if o, _ := Research(Rototill, 25, rs, 15); o.Field != engine.Weapons {
		t.Errorf("tie: field %d, want weapons", o.Field)
	}
	// Every goal of a plan met: as with no plan.
	var all [engine.NumFields]int
	for f := range all {
		all[f] = 26
	}
	rs = engine.ResearchState{Levels: all, Current: engine.Energy}
	if _, write := Research(Robotoid, 150, rs, 0); write {
		t.Error("all goals met, field unchanged: order written")
	}
}

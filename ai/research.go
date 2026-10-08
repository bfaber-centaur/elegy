package ai

import "github.com/bfaber-centaur/elegy/engine"

// Research goals: a field and the level to reach in it.
type goal struct{ field, level int }

const (
	e  = engine.Energy
	w  = engine.Weapons
	p  = engine.Propulsion
	c  = engine.Construction
	el = engine.Electronics
	b  = engine.Biotech
)

// researchPlans are the personalities' research plans (AI.md §4,
// CONFIRMED AI-1). Rototill has none.
var researchPlans = map[Personality][]goal{
	Robotoid: {
		{p, 2}, {c, 3}, {w, 3}, {c, 4}, {e, 2}, {el, 3}, {p, 6}, {w, 5}, {c, 6}, {b, 4},
		{el, 5}, {e, 6}, {w, 7}, {c, 10}, {e, 6}, {el, 7}, {w, 10}, {p, 9}, {p, 12}, {c, 13},
		{w, 14}, {c, 16}, {e, 9}, {el, 10}, {p, 16}, {b, 10}, {e, 15}, {w, 20}, {p, 20}, {el, 16},
		{b, 12}, {w, 24}, {el, 19}, {c, 24}, {e, 22}, {c, 26},
	},
	Cybertron: {
		{c, 4}, {p, 2}, {el, 3}, {b, 3}, {w, 3}, {p, 6}, {c, 6}, {el, 6}, {e, 10}, {c, 10},
		{p, 8}, {c, 13}, {w, 6}, {b, 9}, {el, 9}, {w, 7}, {p, 9}, {c, 16}, {e, 14}, {w, 11},
		{el, 12}, {b, 11}, {p, 13}, {c, 18}, {w, 15}, {e, 18}, {el, 17}, {w, 17}, {c, 20}, {p, 17},
		{b, 18}, {el, 21}, {e, 23}, {p, 22}, {c, 21}, {w, 23}, {p, 26}, {el, 26}, {e, 26}, {w, 26},
		{c, 26}, {b, 26},
	},
}

// ResearchBudget is the personality's research budget in percent of
// resources for a year index (AI.md §4 "Budget", CONFIRMED AI-1): 0 once
// all six tech levels are 24 or more.
func ResearchBudget(pers Personality, y int, levels [engine.NumFields]int) int {
	all24 := true
	for _, l := range levels {
		if l < 24 {
			all24 = false
		}
	}
	if all24 {
		return 0
	}
	switch pers {
	case Robotoid:
		if y < 10 {
			return 0
		}
		return 15
	case Rototill:
		if y < 20 {
			return 0
		}
		return 15
	case Cybertron:
		return 17
	}
	return 0
}

// Research is the year's research choice (AI.md §4, CONFIRMED AI-1). It
// returns the research order and whether the original writes one this
// year.
//
// The first unmet goal of the personality's plan sets the current field;
// when that goal is exactly one level away and is not the last, the next
// field becomes the following goal's field, else it keeps its previous
// setting. While a goal is unmet the order is written every year. With no
// plan (Rototill), or every goal met, the current field is the lowest tech
// level (first in field order on ties) and the order is written only when
// the field changes.
//
// ASSUMPTION A1: in the no-plan case Elegy also writes the order when
// only the budget changes (Rototill at year index 20), so the budget takes
// effect. AI.md §4 does not say how the original's budget changes when no
// research order is written.
func Research(pers Personality, y int, rs engine.ResearchState, budget int) (engine.ResearchOrder, bool) {
	o := engine.ResearchOrder{
		Budget: ResearchBudget(pers, y, rs.Levels),
		Field:  rs.Current,
		Next:   rs.Next,
	}
	plan := researchPlans[pers]
	for i, g := range plan {
		if rs.Levels[g.field] >= g.level {
			continue
		}
		o.Field = g.field
		if g.level-rs.Levels[g.field] == 1 && i+1 < len(plan) {
			o.Next = plan[i+1].field
		}
		return o, true
	}
	low := 0
	for f := range engine.NumFields {
		if rs.Levels[f] < rs.Levels[low] {
			low = f
		}
	}
	o.Field = low
	return o, low != rs.Current || o.Budget != budget
}

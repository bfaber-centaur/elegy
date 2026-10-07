package engine

import "sort"

// Who fights (COMBAT.md "Where battles happen in the turn", "Who fights").

// location is a set of fleets at exactly the same position, with the
// planet there if any.
type location struct {
	pos    Point
	fleets []int // fleet indices, ordered by owner then fleet id
	planet int   // planet index, or -1
}

// locations groups fleets with ships by exact position, in the order of
// each location's first fleet, fleets ordered by owner then id.
func (g *Game) locations() []location {
	var order []int
	for i, f := range g.Fleets {
		if fleetShips(&f) > 0 {
			order = append(order, i)
		}
	}
	sort.SliceStable(order, func(a, b int) bool {
		fa, fb := &g.Fleets[order[a]], &g.Fleets[order[b]]
		if fa.Owner != fb.Owner {
			return fa.Owner < fb.Owner
		}
		return fa.ID < fb.ID
	})
	var locs []location
	at := map[Point]int{}
	for _, i := range order {
		p := g.Fleets[i].Pos
		j, ok := at[p]
		if !ok {
			j = len(locs)
			at[p] = j
			locs = append(locs, location{pos: p, planet: -1})
			for k := range g.Planets {
				if g.Planets[k].Pos == p {
					locs[j].planet = k
					break
				}
			}
		}
		locs[j].fleets = append(locs[j].fleets, i)
	}
	return locs
}

func fleetShips(f *Fleet) int {
	n := 0
	for _, s := range f.Stacks {
		n += s.Count
	}
	return n
}

func (g *Game) fleetArmed(f *Fleet) bool {
	for _, s := range f.Stacks {
		if s.Count > 0 && g.Designs[s.Design].armed() {
			return true
		}
	}
	return false
}

// battleStarbase returns the planet's starbase owner when the planet has a
// starbase that can take part in a battle.
func (g *Game) battleStarbase(planet int) (owner int, ok bool) {
	if planet < 0 {
		return 0, false
	}
	p := &g.Planets[planet]
	if !p.HasStarbase || p.Owner == NoOwner {
		return 0, false
	}
	return p.Owner, true
}

// attackSets holds, per player, the set of players it attacks.
type attackSets []map[int]bool

func (a attackSets) add(attacker, target int) bool {
	if a[attacker] == nil {
		a[attacker] = map[int]bool{}
	}
	if a[attacker][target] {
		return false
	}
	a[attacker][target] = true
	return true
}

// selected is the set of present players other than owner that plan's
// attack-who selects for owner.
func (g *Game) selected(owner int, plan BattlePlan, present []int) []int {
	var out []int
	for _, q := range present {
		if q == owner {
			continue
		}
		rel := g.relation(owner, q)
		switch plan.Attack {
		case AttackEnemies:
			if rel == RelationEnemy {
				out = append(out, q)
			}
		case AttackNeutralsAndEnemies:
			if rel != RelationFriend {
				out = append(out, q)
			}
		case AttackEveryone:
			out = append(out, q)
		case AttackPlayer:
			if q == plan.Player {
				out = append(out, q)
			}
		}
	}
	return out
}

// legacyPlan0Recipient reproduces the original's LEGACY BUG for a
// starbase whose owner's plan 0 attacks "everyone" or a named player:
// the attack set goes to another player X (COMBAT.md "LEGACY BUG: plan 0
// ...", BINARY-ONLY, consistent with CB-011..013). X is player 0 when
// the previously examined location had a battle, else the owner of that
// location's last fleet. For the first location the original's value is
// undetermined; Elegy uses the starbase's owner. Set legacyPlan0 to false
// to always give the set to the owner.
const legacyPlan0 = true

type locationHistory struct {
	any       bool // a location was examined before
	battle    bool // it had a battle
	lastOwner int  // owner of its last fleet
}

func legacyPlan0Recipient(owner int, plan BattlePlan, prev locationHistory) int {
	if !legacyPlan0 || !prev.any || (plan.Attack != AttackEveryone && plan.Attack != AttackPlayer) {
		return owner
	}
	if prev.battle {
		return 0
	}
	return prev.lastOwner
}

// whoFights decides a location's battle: the attack sets and the involved
// players (in player order), or no involved players when there is no
// battle.
func (g *Game) whoFights(loc location, prev locationHistory) (attackSets, []int) {
	sets := make(attackSets, len(g.Players))
	var present []int
	isPresent := map[int]bool{}
	addPresent := func(p int) {
		if !isPresent[p] {
			isPresent[p] = true
			present = append(present, p)
		}
	}
	for _, i := range loc.fleets {
		addPresent(g.Fleets[i].Owner)
	}
	sbOwner, hasSB := g.battleStarbase(loc.planet)
	if hasSB {
		addPresent(sbOwner)
	}
	sort.Ints(present)

	// 1–3. Aggressor fleets, and their owners' attack sets. Only fleets
	// start battles (CONFIRMED, CB-002, CB-003, CB-004, CB-006, Q-1).
	aggressor := false
	for _, i := range loc.fleets {
		f := &g.Fleets[i]
		plan := g.plan(f.Owner, f.Plan)
		if plan.Primary == TargetNone || plan.Attack == AttackNobody || !g.fleetArmed(f) {
			continue
		}
		aggressor = true
		for _, q := range g.selected(f.Owner, plan, present) {
			sets.add(f.Owner, q)
		}
	}
	if !aggressor {
		return nil, nil
	}

	// 4. An armed starbase joins with its owner's plan 0 (CONFIRMED,
	// CB-011, Q-1).
	if hasSB {
		p := &g.Planets[loc.planet]
		plan := g.plan(sbOwner, 0)
		if g.Designs[p.StarbaseDesign].armed() && plan.Attack != AttackNobody {
			x := legacyPlan0Recipient(sbOwner, plan, prev)
			for _, q := range g.selected(sbOwner, plan, present) {
				sets.add(x, q)
			}
		}
	}

	// 5. Retaliation and friends, until nothing changes (BINARY-ONLY;
	// firing back is CONFIRMED, CB-002, CB-009).
	involved := func(p int) bool {
		if len(sets[p]) > 0 {
			return true
		}
		for _, s := range sets {
			if s[p] {
				return true
			}
		}
		return false
	}
	for changed := true; changed; {
		changed = false
		for a := range sets {
			for b := range sets[a] {
				if sets.add(b, a) {
					changed = true
				}
			}
		}
		for _, f := range present {
			for _, p := range present {
				if f == p || g.relation(f, p) != RelationFriend || !involved(p) {
					continue
				}
				for q := range sets[p] {
					if q != f && sets.add(f, q) {
						changed = true
					}
				}
			}
		}
	}

	// 6. Involvement: a battle needs two or more involved players.
	var inv []int
	for _, p := range present {
		if involved(p) {
			inv = append(inv, p)
		}
	}
	if len(inv) < 2 {
		return nil, nil
	}
	return sets, inv
}

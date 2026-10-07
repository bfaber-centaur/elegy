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

// selects reports whether attack-who "enemies" or "neutrals and enemies"
// of owner selects player q.
func (g *Game) selects(owner int, a AttackWho, q int) bool {
	if q == owner {
		return false
	}
	rel := g.relation(owner, q)
	return rel == RelationEnemy || (a == AttackNeutralsAndEnemies && rel == RelationNeutral)
}

// write applies one plan's attack-who to the set of player x on behalf of
// owner (COMBAT.md "Who fights" steps 3 and 4): "enemies" and "neutrals
// and enemies" add every such player of the game, a named player is
// added, and "everyone" replaces the set with every player but owner.
func (g *Game) write(sets attackSets, x, owner int, plan BattlePlan) {
	switch plan.Attack {
	case AttackEnemies, AttackNeutralsAndEnemies:
		for q := range g.Players {
			if g.selects(owner, plan.Attack, q) {
				sets.add(x, q)
			}
		}
	case AttackPlayer:
		sets.add(x, plan.Player)
	case AttackEveryone:
		sets[x] = map[int]bool{}
		for q := range g.Players {
			if q != owner {
				sets[x][q] = true
			}
		}
	}
}

// legacyPlan0Recipient reproduces the original's LEGACY BUG for a
// starbase whose owner's plan 0 attacks "everyone" or a named player: the
// set is written to another player X (COMBAT.md "LEGACY BUG: plan 0 ...",
// CONFIRMED CB-011..013, CB-022, CB-035). X is player 0 when the previously
// examined location had a battle, else the owner of that location's last
// fleet. Set legacyPlan0 to false to write to the owner instead.
//
// For the first location of a turn X is not determined (COMBAT.md,
// BINARY-ONLY; never a player of the game in CB-022 and in 36 runs of
// CB-035). COMBAT.md's chosen rule for Elegy: there plan 0 contributes
// nothing (ok is false).
const legacyPlan0 = true

type locationHistory struct {
	any       bool // a location was examined before
	battle    bool // it had a battle
	lastOwner int  // owner of its last fleet
}

func legacyPlan0Recipient(owner int, prev locationHistory) (x int, ok bool) {
	switch {
	case !legacyPlan0:
		return owner, true
	case !prev.any:
		return 0, false
	case prev.battle:
		return 0, true
	}
	return prev.lastOwner, true
}

// whoFights decides a location's battle (COMBAT.md "Who fights", the
// procedure; BINARY-ONLY in its details). It returns the attack sets, the
// battle's player list P in player order and n, the number of involved
// players; P is nil when there is no battle.
func (g *Game) whoFights(loc location, prev locationHistory) (attackSets, []int, int) {
	np := len(g.Players)
	sets := make(attackSets, np)
	inP := map[int]bool{}
	for _, i := range loc.fleets {
		inP[g.Fleets[i].Owner] = true
	}
	sbOwner, hasSB := g.battleStarbase(loc.planet)
	if hasSB {
		inP[sbOwner] = true
	}

	// 3. Starbase plan 0 (CONFIRMED, CB-011, Q-1; LEGACY BUG CB-022).
	if hasSB && g.Designs[g.Planets[loc.planet].StarbaseDesign].armed() {
		plan := g.plan(sbOwner, 0)
		switch plan.Attack {
		case AttackEnemies, AttackNeutralsAndEnemies:
			g.write(sets, sbOwner, sbOwner, plan)
		case AttackEveryone, AttackPlayer:
			if x, ok := legacyPlan0Recipient(sbOwner, prev); ok && x >= 0 && x < np {
				g.write(sets, x, sbOwner, plan)
			}
		}
	}

	// 4–5. Aggressor fleets, in location order. Only fleets start battles
	// (CONFIRMED, CB-002, CB-003, CB-004, CB-006, Q-1).
	aggressor := false
	for _, i := range loc.fleets {
		f := &g.Fleets[i]
		plan := g.plan(f.Owner, f.Plan)
		if plan.Primary == TargetNone || plan.Attack == AttackNobody || !g.fleetArmed(f) {
			continue
		}
		aggressor = true
		g.write(sets, f.Owner, f.Owner, plan)
	}
	if !aggressor {
		return nil, nil, 0
	}

	// 6. The attacked set Q.
	Q := map[int]bool{}
	for _, s := range sets {
		for q := range s {
			if inP[q] {
				Q[q] = true
			}
		}
	}
	if len(Q) == 0 {
		return nil, nil, 0
	}

	// 7. Retaliation: one pass in player order (firing back CONFIRMED,
	// CB-002, CB-009).
	for i := range np {
		for q := range sets[i] {
			if Q[q] {
				Q[i] = true
				break
			}
		}
		if Q[i] {
			for j := range np {
				if sets[j][i] {
					sets.add(i, j)
				}
			}
		}
	}

	// 8. Friends: passes over the fleets until nothing changes.
	for changed := true; changed; {
		changed = false
		for _, i := range loc.fleets {
			p := g.Fleets[i].Owner
			if !inP[p] || Q[p] {
				continue
			}
			built := map[int]bool{}
			for f := range np {
				if f == p || g.relation(p, f) != RelationFriend || !Q[f] {
					continue
				}
				if built[f] {
					built = map[int]bool{} // two friends fight each other
					break
				}
				for q := range sets[f] {
					built[q] = true
				}
			}
			sets[p] = built
			if len(built) == 0 {
				delete(inP, p)
			} else {
				Q[p] = true
			}
			changed = true
		}
	}

	var players []int
	for p := range np {
		if inP[p] {
			players = append(players, p)
		}
	}
	return sets, players, len(Q)
}

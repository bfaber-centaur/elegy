package engine

import "sort"

// Battle movement (COMBAT.md "Movement order", "Disengaging", "Choosing a
// square", "Square score"). BINARY-ONLY except the moves per round, the
// disengage counter, and one mover against a station (CB-012, CB-019).

// move runs one round's movement: three phases a = 3, 2, 1, tokens in
// descending jittered weight (ties keep token order).
func (b *battle) move() {
	order := make([]int, 0, len(b.tokens))
	for i, t := range b.tokens {
		t.moves = 0
		if t.live() && !t.starbase {
			t.moves = movesInRound(t.speed, b.round)
		}
		order = append(order, i)
	}
	weight := func(t *token) int { return t.mass + t.mass*(t.jitter-7)*2/100 }
	sort.SliceStable(order, func(i, j int) bool {
		return weight(b.tokens[order[i]]) > weight(b.tokens[order[j]])
	})
	for a := 3; a >= 1; a-- {
		for _, i := range order {
			t := b.tokens[i]
			if t.live() && !t.starbase && t.moves >= a {
				b.step(t)
				t.moves--
			}
		}
	}
}

// step makes one single-square move.
func (b *battle) step(t *token) {
	// Every move the token is given counts, including one where it stays
	// on its square; the counter is lowered before the square is chosen
	// (COMBAT.md "Disengaging", BINARY-ONLY).
	if t.tactic == TacticDisengage {
		if t.counter == 0 {
			t.left = true // off the board: out of the battle, not destroyed
			return
		}
		t.counter--
	}
	radius, goal, hasGoal := b.radius(t)
	px, py := t.x, t.y
	best, bestDist, ties := 0, 0, 0
	for x := range boardSize {
		for y := range boardSize {
			d := dist(t.x, t.y, x, y)
			if d > radius {
				continue
			}
			s := b.score(t, x, y)
			switch {
			case ties == 0 || s < best || (s == best && d < bestDist):
				px, py, best, bestDist, ties = x, y, s, d, 1
			case s == best && d == bestDist:
				ties++
				if b.rng.Intn(ties) == 0 {
					px, py = x, y
				}
			}
		}
	}
	if hasGoal {
		px, py = goal[0], goal[1]
	}
	dx, dy := px-t.x, py-t.y
	if max(abs(dx), abs(dy)) <= 1 {
		t.x, t.y = px, py
		return
	}
	sx, sy := sign(dx), sign(dy)
	var nx, ny int
	switch {
	case abs(dx) == abs(dy):
		nx, ny = t.x+sx, t.y+sy
	case dx == 0 || dy == 0:
		var cand [3][2]int
		if dx == 0 {
			cand = [3][2]int{{t.x - 1, t.y + sy}, {t.x, t.y + sy}, {t.x + 1, t.y + sy}}
		} else {
			cand = [3][2]int{{t.x + sx, t.y - 1}, {t.x + sx, t.y}, {t.x + sx, t.y + 1}}
		}
		var scores [3]int
		low := 0
		for i, c := range cand {
			scores[i] = b.score(t, c[0], c[1])
			if i == 0 || scores[i] < low {
				low = scores[i]
			}
		}
		var tied [][2]int
		for i, c := range cand {
			if scores[i] == low {
				tied = append(tied, c)
			}
		}
		c := tied[0]
		if len(tied) > 1 {
			c = tied[b.rng.Intn(len(tied))]
		}
		nx, ny = c[0], c[1]
	default:
		diag := [2]int{t.x + sx, t.y + sy}
		straight := [2]int{t.x + sx, t.y}
		if abs(dy) >= abs(dx) {
			straight = [2]int{t.x, t.y + sy}
		}
		sd, ss := b.score(t, diag[0], diag[1]), b.score(t, straight[0], straight[1])
		c := straight
		if sd < ss || (sd == ss && b.rng.Intn(2) == 0) {
			c = diag
		}
		nx, ny = c[0], c[1]
	}
	if nx < 0 || ny < 0 || nx >= boardSize || ny >= boardSize {
		return // a step off the board leaves the token where it is
	}
	t.x, t.y = nx, ny
}

func sign(x int) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

// targetType is the type token t is currently looking for: its primary,
// or its secondary when no live enemy matches the primary.
func (b *battle) targetType(t *token) TargetType {
	for _, e := range b.tokens {
		if e.live() && b.attacks(t.player, e.player) && e.matches(t.primary) {
			return t.primary
		}
	}
	return t.secondary
}

// radius returns the square radius for one move of t and its goal square,
// if any.
func (b *battle) radius(t *token) (int, [2]int, bool) {
	tt := b.targetType(t)
	longest := t.longestReach()
	r := longest
	if (t.tactic == TacticMaximizeNet || t.tactic == TacticMaximizeDamage) && t.shortestReach() < longest {
		r = t.shortestReach()
	}
	reach := r + t.moves
	nonSapper := 0
	for _, w := range t.weapons {
		if w.part.Kind == PartBeam && !w.part.Sapper {
			nonSapper = max(nonSapper, w.reach)
		}
	}
	var enemies []*token
	for _, e := range b.tokens {
		if e.live() && b.attacks(t.player, e.player) && e.matches(tt) {
			enemies = append(enemies, e)
		}
	}
	effDist := func(e *token) int {
		d := dist(t.x, t.y, e.x, e.y)
		if e.moves >= t.moves {
			d++
		}
		return d
	}
	for _, e := range enemies {
		d := effDist(e)
		if longest == 3 && e.shield == 0 {
			if d <= t.moves+nonSapper {
				return t.moves, [2]int{}, false
			}
			continue
		}
		if d <= reach {
			return t.moves, [2]int{}, false
		}
	}
	var goal *token
	gd := 0
	for _, e := range enemies {
		if !b.canDamage(t, e) {
			continue
		}
		if d := effDist(e); goal == nil || d < gd {
			goal, gd = e, d
		}
	}
	if goal == nil {
		return 1, [2]int{}, false
	}
	return 1, [2]int{goal.x, goal.y}, true
}

// canDamage reports whether some weapon of t can hurt e: a sapper cannot
// hurt an unshielded target.
func (b *battle) canDamage(t, e *token) bool {
	for _, w := range t.weapons {
		if !w.part.Sapper || e.shield > 0 {
			return true
		}
	}
	return false
}

// tacticScore scores a square for a tactic; lower is better.
func tacticScore(tactic Tactic, give, take int) int {
	switch tactic {
	case TacticDisengage, TacticMinimizeDamage:
		return take
	case TacticDisengageIfChallenged, TacticMaximizeDamage:
		return -give
	}
	if give == 0 {
		return take
	}
	return min(-1, -give*100/(take+1))
}

// score is the square score of (qx, qy) for token t.
func (b *battle) score(t *token, qx, qy int) int {
	tt := b.targetType(t)
	give, take := 0, 0
	for _, e := range b.tokens {
		if !e.live() || !b.attacks(t.player, e.player) {
			continue
		}
		d := dist(qx, qy, e.x, e.y)
		lo, hi := d, d
		if e.moves >= t.moves {
			lo = max(0, d-1)
			for nx := max(0, e.x-1); nx <= min(boardSize-1, e.x+1); nx++ {
				for ny := max(0, e.y-1); ny <= min(boardSize-1, e.y+1); ny++ {
					hi = max(hi, dist(qx, qy, nx, ny))
				}
			}
		}
		bestS, bestGive, bestTake := 0, 0, 0
		for x := lo; x <= hi; x++ {
			tk := b.estimate(e, t, x, t.tactic == TacticDisengage)
			gv := 0
			if e.matches(tt) {
				gv = b.estimate(t, e, x, false)
			}
			if s := tacticScore(e.tactic, tk, gv); x == lo || s <= bestS {
				bestS, bestGive, bestTake = s, gv, tk
			}
		}
		give = max(give, bestGive)
		take += bestTake
	}
	s := tacticScore(t.tactic, give, take)
	if t.tactic == TacticDisengage {
		for _, o := range b.tokens {
			if o != t && o.live() && o.player == t.player && o.x == qx && o.y == qy {
				s += 2
			}
		}
		if qx == t.x && qy == t.y {
			s--
		}
	}
	return s
}

// estimate is a's estimated damage to e at distance x (COMBAT.md "Damage
// estimate"). It simulates ships × count × 200 torpedoes per slot; like a
// salvo, exactly 200 of them draw one rand(100) each (BINARY-ONLY quirk,
// kept for the random stream).
func (bt *battle) estimate(a, b *token, x int, ignoreRange bool) int {
	if !ignoreRange && x > a.longestReach() {
		return 0
	}
	total := 0
	for _, w := range a.weapons {
		if !ignoreRange && x > w.reach {
			continue
		}
		r := w.reach
		out := func(v int) int {
			if x > w.reach {
				return max(w.count, v/(x+10-r))
			}
			return v
		}
		p := w.part
		if p.Kind == PartBeam {
			v := p.Damage * w.count
			v = v * a.capacitor / 100
			// COMBAT.md: "If x > 0", with r including the starbase +1.
			// ASSUMPTION A7 (docs/COMBAT-STATUS.md): r is 0 only for a
			// ship's range-0 beam when range is ignored; the estimate then
			// skips the dropoff, as real fire does for range 0.
			if x > 0 && r > 0 {
				v = v + (x*v)/(-10)/r
			}
			v = v * b.deflector / 100
			if p.Sapper {
				v = min(v, b.shield*a.ships)
			}
			total += a.ships * out(v)
			continue
		}
		n := a.ships * w.count * 200
		pct := hitChance(p.Accuracy, a.computer, b.jammer)
		h := n
		switch {
		case pct >= 100:
		case n > 200:
			h = n * pct / 100
		default:
			h = 0
			for range n {
				if bt.rng.Intn(100) < pct {
					h++
				}
			}
		}
		v := p.Damage * h / 200
		if b.shield > 0 {
			v += p.Damage * (n - h) / 1600
		}
		total += out(v)
	}
	if !ignoreRange {
		total = min(total, max(1, (b.armor+b.shield)*b.ships-b.existingDamage()))
	}
	return total
}

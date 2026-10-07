package engine

import "sort"

// Firing and damage (COMBAT.md "Firing", "Target choice", "Beams",
// "Torpedoes and missiles", "Damage"). CONFIRMED by replay of every hit
// record in CB-001..CB-019 except where marked.

// fire runs one round's firing: weapon initiative levels from the highest
// to the lowest, tokens in reverse token order.
func (b *battle) fire() {
	seen := map[int]bool{}
	var levels []int
	for _, t := range b.tokens {
		if !t.live() {
			continue
		}
		for _, w := range t.weapons {
			if !seen[w.init] {
				seen[w.init] = true
				levels = append(levels, w.init)
			}
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(levels)))
	for _, level := range levels {
		for i := len(b.tokens) - 1; i >= 0; i-- {
			t := b.tokens[i]
			for _, w := range t.weapons {
				if w.init != level || !t.live() {
					continue
				}
				if b.playersIn() < 2 {
					return
				}
				switch {
				case w.part.Kind == PartTorpedo:
					b.torpedoes(i, w)
				case w.part.Gatling:
					b.gatling(i, w)
				default:
					b.beam(i, w)
				}
			}
		}
	}
}

// playersIn counts the players still in the battle: those that were in
// at the start of firing and still have a live token and a live token to
// attack. (Recomputing it during firing is Elegy's reading of "while at
// least two players are still in the battle".)
func (b *battle) playersIn() int {
	n := 0
	for p := range b.in {
		alive := false
		for _, t := range b.tokens {
			if t.live() && t.player == p {
				alive = true
				break
			}
		}
		if alive && b.hasPrey(p) {
			n++
		}
	}
	return n
}

// eligible returns the eligible targets of token t's weapon slot w, in
// token order.
func (b *battle) eligible(t *token, w weaponSlot) []int {
	pick := func(tt TargetType) []int {
		var out []int
		for j, e := range b.tokens {
			if e.live() && b.attacks(t.player, e.player) && dist(t.x, t.y, e.x, e.y) <= w.reach && e.matches(tt) {
				out = append(out, j)
			}
		}
		return out
	}
	if out := pick(t.primary); len(out) > 0 {
		return out
	}
	return pick(t.secondary)
}

// choose returns the most attractive eligible target, the first in token
// order on ties, or -1 when every score is 0.
func (b *battle) choose(t *token, w weaponSlot) int {
	best, bestScore := -1, 0
	for _, j := range b.eligible(t, w) {
		if s := attractiveness(t, w, b.tokens[j]); s > bestScore {
			best, bestScore = j, s
		}
	}
	return best
}

// attractiveness is the target-choice score (COMBAT.md "Target choice").
func attractiveness(t *token, w weaponSlot, e *token) int {
	cost := e.cost * e.ships
	if cost < 100000 {
		cost *= 100
	} else {
		cost = 10_000_000
	}
	A := e.toughArmor()
	S := e.shield * e.ships
	p := w.part
	if p.Kind == PartBeam {
		cost = cost * e.deflector / 100
		if p.Sapper {
			if S < 1 {
				return 0
			}
			return ceilDiv(cost*100, S)
		}
		return max(1, cost*100/(A+S+1))
	}
	c, j := t.computer, e.jammer
	var a int
	if c < j {
		a = p.Accuracy - (j-c)*p.Accuracy/100
	} else {
		a = p.Accuracy + (c-j)*(100-p.Accuracy)/100
	}
	if a <= 0 {
		return 0
	}
	m := 1
	if p.Missile {
		m = 2
	}
	X := A * 200 / a
	Y := S * 100 / (a/2 + (100-a)/8)
	Z := (A - Y*a/200) * 100 / (m * a)
	eff := min(X, Y+Z)
	if eff <= 0 {
		return 0
	}
	return max(1, cost/eff)
}

// hitChance is a torpedo's hit chance in percent, at least 1.
func hitChance(acc, c, j int) int {
	var p int
	if c > j {
		p = 100 - (100-(c-j))*(100-acc)/100
	} else {
		p = acc * (100 - (j - c)) / 100
	}
	return max(1, p)
}

func ceilDiv(a, b int) int { return (a + b - 1) / b }

func (b *battle) record(firer, target int, h BattleHit) {
	h.Round, h.Firer, h.Target = b.round, firer, target
	b.hits = append(b.hits, h)
}

// beam fires one beam slot, carrying damage to further targets after a
// kill.
func (b *battle) beam(fi int, w weaponSlot) {
	t := b.tokens[fi]
	p := w.part
	R := t.ships * p.Damage * w.count
	for R > 0 {
		ti := b.choose(t, w)
		if ti < 0 {
			return
		}
		e := b.tokens[ti]
		dp := R * t.capacitor / 100 * e.deflector / 100
		// Dropoff uses the part's own range, without the starbase +1.
		if x := dist(t.x, t.y, e.x, e.y); x > 0 && p.Range > 0 {
			dp = (100 - x*10/p.Range) * dp / 100
		}
		if dp <= 0 {
			return
		}
		h, ok := b.damage(e, dp, 0, p.Sapper, -1)
		if ok {
			b.record(fi, ti, h)
		}
		if !e.dead || h.Leftover <= 0 {
			return
		}
		R = min(R-1, R*h.Leftover/dp)
	}
}

// gatling hits every eligible target once, in token order, with no
// dropoff and no carry.
func (b *battle) gatling(fi int, w weaponSlot) {
	t := b.tokens[fi]
	for _, ti := range b.eligible(t, w) {
		e := b.tokens[ti]
		if !e.live() {
			continue
		}
		dp := t.ships * w.part.Damage * w.count * t.capacitor / 100 * e.deflector / 100
		if dp <= 0 {
			continue
		}
		if h, ok := b.damage(e, dp, 0, w.part.Sapper, -1); ok {
			b.record(fi, ti, h)
		}
	}
}

// torpedoes fires one torpedo or missile slot.
func (b *battle) torpedoes(fi int, w weaponSlot) {
	t := b.tokens[fi]
	p := w.part
	N := t.ships * w.count
	ti := b.choose(t, w)
	if ti < 0 {
		return
	}
	// The hit count uses the jammer of the first target. (COMBAT.md
	// computes p per target; the hits are drawn once per salvo. Elegy
	// draws against the first target chosen.)
	pct := hitChance(p.Accuracy, t.computer, b.tokens[ti].jammer)
	H := 0
	switch {
	case pct >= 100:
		H = N
	case N > 200:
		H = N * pct / 100
	default:
		for range N {
			if b.rng.Intn(100) < pct {
				H++
			}
		}
	}
	for N > 0 {
		if ti < 0 {
			if ti = b.choose(t, w); ti < 0 {
				return
			}
		}
		e := b.tokens[ti]
		d := p.Damage
		S := e.shield * e.ships
		if p.Missile && S < 1 {
			d *= 2
		}
		A := e.toughArmor()
		n, hits := N, H
		if e.ships < N && H*d > A {
			n = N
			for k := e.ships; k <= N; k++ {
				hk := ceilDiv(k*H, N)
				sp := max(0, S-(k-hk)*d/8)
				armor := hk * d / 2
				if sp < hk*d/2 {
					armor = hk*d - sp
				}
				if armor >= A {
					n = k
					break
				}
			}
			hits = ceilDiv(n*H, N)
		}
		misses := n - hits
		var h BattleHit
		recorded := false
		if misses > 0 && S > 0 {
			if mh, ok := b.damage(e, misses*d/8, 0, true, -1); ok {
				h.Shield += mh.Shield
				recorded = true
			}
		}
		if hits > 0 && e.live() {
			hh, _ := b.damage(e, hits*d/2, hits*d/2, false, n)
			h.Shield += hh.Shield
			h.Armor, h.Kills = hh.Armor, hh.Kills
			recorded = true
		}
		if recorded {
			h.Hits, h.Misses = hits, misses
			b.record(fi, ti, h)
		}
		N -= n
		H -= hits
		ti = -1
	}
}

// damage applies dp (and, for torpedo hits, the armor-only part extra) to
// token e. shieldOnly is a sapper or torpedo misses; limit caps kills (-1
// for none). ok is false when nothing was recorded.
func (b *battle) damage(e *token, dp, extra int, shieldOnly bool, limit int) (h BattleHit, ok bool) {
	S := e.shield * e.ships
	if shieldOnly && S == 0 {
		return h, false
	}
	if S > dp {
		e.shield = (S - dp) / e.ships
		h.Shield, dp = dp, 0
	} else {
		h.Shield = S
		dp -= S
		e.shield = 0
	}
	if shieldOnly || dp+extra == 0 {
		return h, true
	}
	dp += extra
	h.Armor = dp
	if e.tactic == TacticDisengageIfChallenged {
		e.tactic, e.counter = TacticDisengage, 7
	}
	if e.starbase {
		b.starbaseDamage(e, dp, extra, &h)
		return h, true
	}
	armor := e.armor
	damaged := 0
	if e.dmg.Units > 0 && e.dmg.Pct > 0 {
		damaged = max(1, e.ships*e.dmg.Pct/100)
	}
	per := max(1, e.dmg.Units*armor/500)
	fresh := e.ships - damaged
	kills := 0
	can := func() bool { return limit < 0 || kills < limit }
	for damaged > 0 && dp >= armor-per && can() {
		dp -= armor - per
		damaged--
		kills++
	}
	for fresh > 0 && dp >= armor && can() {
		dp -= armor
		fresh--
		kills++
	}
	surv := e.ships - kills
	if limit >= 0 && kills >= limit {
		dp = 0 // one kill per torpedo: the rest is lost
	}
	switch {
	case surv == 0:
		e.dead = true
		h.Leftover = max(0, dp-extra)
		e.dmg = Damage{}
	case dp > 0:
		if damaged > 0 {
			dp = ceilDiv(dp+damaged*per, surv)
		} else {
			dp /= surv
		}
		dp = max(1, dp)
		e.dmg = Damage{Pct: 100, Units: min(499, max(1, ceilDiv(dp*500, armor)))}
	default:
		e.dmg.Pct = ceilDiv(damaged*100, surv)
	}
	e.ships = surv
	h.Kills = kills
	if kills > 0 {
		b.killEvent(e, kills)
	}
	return h, true
}

// starbaseDamage applies armor damage to a starbase (COMBAT.md "Damage",
// "Starbases"). dp already includes extra.
func (b *battle) starbaseDamage(e *token, dp, extra int, h *BattleHit) {
	total := dp + e.dmg.Units*e.armor/500
	if total < e.armor {
		if u := max(e.dmg.Units+1, total*500/e.armor); u < 500 {
			e.dmg.Units = u
			return
		}
	}
	// Destroyed. The leftover for a beam carry is what remains past the
	// starbase's remaining armor (Elegy's reading; COMBAT.md states the
	// leftover for ship stacks).
	h.Leftover = max(0, total-e.armor-extra)
	h.Kills = 1
	e.dead, e.ships = true, 0
	b.killEvent(e, 1)
}

// killEvent records a kill event: salvage, seen tech and who lost ships.
// e.ships already excludes the kills.
func (b *battle) killEvent(e *token, kills int) {
	g := b.g
	d := g.Designs[e.design]
	b.killed[e.player] = true
	r := d.techReq()
	for f := range NumFields {
		b.seen[f] = max(b.seen[f], r[f])
	}
	var s Minerals
	for m := range NumMinerals {
		s[m] = d.Cost.Minerals[m] * kills / 3
	}
	if e.fleet >= 0 {
		f := &g.Fleets[e.fleet]
		capacity := kills * d.CargoCapacity
		fleetDead := true
		for _, o := range b.tokens {
			if o.fleet == e.fleet && o.ships > 0 {
				fleetDead = false
				capacity += o.ships * g.Designs[o.design].CargoCapacity
			}
		}
		for m := range NumMinerals {
			share := f.Cargo.Minerals[m]
			if !fleetDead {
				share = 0
				if capacity > 0 {
					share = f.Cargo.Minerals[m] * kills * d.CargoCapacity / capacity
				}
			}
			s[m] += share
			f.Cargo.Minerals[m] -= share
		}
		if fleetDead {
			f.Cargo.Colonists = 0
		} else if capacity > 0 {
			f.Cargo.Colonists -= f.Cargo.Colonists * kills * d.CargoCapacity / capacity
		}
	}
	if b.loc.planet >= 0 {
		pl := &g.Planets[b.loc.planet]
		num := 5
		if pl.HasStarbase {
			num = 8
		}
		for m := range NumMinerals {
			pl.Surface[m] += s[m] * num / 10
		}
		return
	}
	for m := range NumMinerals {
		b.salvage[m] += s[m] - s[m]/4
	}
	b.salvaged = true
}

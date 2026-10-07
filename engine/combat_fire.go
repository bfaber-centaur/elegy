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
			acts := false
			for _, w := range t.weapons {
				if w.init == level && t.live() {
					acts = true
				}
			}
			if !acts {
				continue
			}
			// Checked again before each token acts (COMBAT.md "Firing",
			// BINARY-ONLY): with one player left, no further token fires.
			if b.livePlayers() < 2 {
				return
			}
			for _, w := range t.weapons {
				if w.init != level || !t.live() {
					continue
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
		// Dropoff uses the part's own range, without the starbase +1; a
		// range-0 beam has none (CONFIRMED, CB-026).
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
	for N > 0 {
		ti := b.choose(t, w)
		if ti < 0 {
			return
		}
		// Hits are computed afresh for each target, from the torpedoes
		// still unfired and that target's jammer.
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
		// Misses do misses·d/8 to shields only, when that is above 0 and
		// the target has shields left (COMBAT.md "Torpedoes and missiles"
		// step 3).
		if dm := misses * d / 8; dm > 0 && S > 0 {
			mh, _ := b.damage(e, dm, 0, true, -1)
			h.Shield += mh.Shield
		}
		// Hits: h = hits·d/2, truncated once, to shields and again to
		// armor (step 4, CONFIRMED, CS-003-C2). Every target the salvo
		// reaches gets a hit record, even with 0 hits (BINARY-ONLY).
		if hits > 0 {
			hh, _ := b.damage(e, hits*d/2, hits*d/2, false, n)
			h.Shield += hh.Shield
			h.Armor, h.Kills = hh.Armor, hh.Kills
		}
		h.Hits, h.Misses = hits, misses
		b.record(fi, ti, h)
		N -= n
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
	damaged, per := 0, 0 // no damage: no damaged ships (CONFIRMED, CB-001 B3)
	if e.dmg.Units > 0 {
		damaged = max(1, e.ships*e.dmg.Pct/100)
		per = max(1, e.dmg.Units*armor/500)
	}
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
	// Destroyed. No damage is left over after a hit on a starbase.
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
		s[m] = e.minerals[m] * kills / 3
	}
	if e.fleet >= 0 {
		share := b.cargoShare(e, kills)
		for m := range NumMinerals {
			s[m] += share[m]
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
		s[m] -= s[m] / 4
	}
	b.pending = append(b.pending, s)
}

// salvageSteps is a salvage object's limit: 30000 kT in 10 kT steps.
const salvageSteps = 3000

// addSalvage adds minerals to this battle's deep-space salvage (COMBAT.md
// "Salvage", BINARY-ONLY in detail; the 30000 kT overflow CONFIRMED,
// CB-040). An all-zero addition becomes rand(10)
// of each, redrawn until above 0. The open object's minerals are taken
// out and re-added with the new ones, ironium, boranium, germanium; a
// mineral that does not fit fills the object to exactly 3000 steps, and
// the rest goes into a new object at the same position in a new pass.
func (b *battle) addSalvage(add Minerals) {
	for add == (Minerals{}) {
		for m := range NumMinerals {
			add[m] = b.rng.Intn(10)
		}
	}
	if len(b.salvage) == 0 {
		b.salvage = append(b.salvage, Minerals{})
	}
	last := len(b.salvage) - 1
	for m := range NumMinerals {
		add[m] += b.salvage[last][m]
	}
	b.salvage[last] = Minerals{}
	for {
		obj := &b.salvage[len(b.salvage)-1]
		used := 0
		full := false
		for m := range NumMinerals {
			a := add[m]
			if a == 0 {
				continue
			}
			if steps := (a + 9) / 10; used+steps <= salvageSteps {
				obj[m] += a
				used += steps
				add[m] = 0
				continue
			}
			fit := 10 * (salvageSteps - used)
			obj[m] += fit
			add[m] -= fit
			full = true
			break
		}
		if !full {
			return
		}
		b.salvage = append(b.salvage, Minerals{})
	}
}

// cargoShare removes the destroyed ships' share of their fleet's cargo
// and fuel and returns the cargo's minerals (COMBAT.md "Salvage",
// the cargo share BINARY-ONLY; the fuel share CONFIRMED, CB-023). The
// shares of colonists and fuel are destroyed. Each kill
// event takes its share from what the fleet holds at that moment.
// e.ships already excludes the kills.
func (b *battle) cargoShare(e *token, kills int) Minerals {
	g := b.g
	f := &g.Fleets[e.fleet]
	lost := kills * g.Designs[e.design].CargoCapacity
	lostFuel := kills * g.Designs[e.design].FuelCapacity
	before, beforeFuel, fleetDead := lost, lostFuel, true
	for _, o := range b.tokens {
		if o.fleet == e.fleet && !o.starbase && o.ships > 0 {
			fleetDead = false
			before += o.ships * g.Designs[o.design].CargoCapacity
			beforeFuel += o.ships * g.Designs[o.design].FuelCapacity
		}
	}
	// Fuel is shared like cargo, by fuel capacity: F · lost / before.
	if beforeFuel > 0 {
		f.Fuel -= f.Fuel * lostFuel / beforeFuel
	}
	cargo := [NumMinerals + 1]int{f.Cargo.Minerals[Ironium], f.Cargo.Minerals[Boranium], f.Cargo.Minerals[Germanium], f.Cargo.Colonists}
	var share [NumMinerals + 1]int
	if fleetDead {
		share = cargo
	} else if C := f.Cargo.mass(); C > 0 && before > 0 && lost > 0 {
		moved := C * lost / before
		left := moved
		for i := range cargo {
			share[i] = cargo[i] * moved / C
			left -= share[i]
		}
		for i := range cargo {
			if left > 0 && cargo[i]-share[i] > 0 {
				share[i]++
				left--
			}
		}
	}
	for m := range NumMinerals {
		f.Cargo.Minerals[m] -= share[m]
	}
	f.Cargo.Colonists -= share[NumMinerals]
	return Minerals{share[Ironium], share[Boranium], share[Germanium]}
}

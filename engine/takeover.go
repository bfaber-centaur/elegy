package engine

import (
	"math"
	"sort"
)

// Planet takeover (stars-elegy TAKEOVER.md, with the order and amount
// rules of stars-elegy #34 and #37): orbital bombing, transport unloads,
// colonist drops and ground combat, colonization, and what an emptied
// planet keeps.

// Takeover events.
const (
	EventBombed         EventKind = iota + EventBattleTech + 1 // Player = bomber, Planet, Count = population killed (units)
	EventPlanetEmptied                                         // Player = the old owner, Planet
	EventColonized                                             // Player = the new owner, Planet, Count = population landed (units)
	EventDropRepelled                                          // Player = attacker, Planet, Count = troops lost (units)
	EventDropRefused                                           // Player, Fleet, Planet: the colonists stay aboard
	EventColonizeFailed                                        // Player, Fleet, Planet (or -1)
)

// TaskKind is a waypoint task. Only the takeover tasks are modelled.
type TaskKind int

const (
	TaskNone      TaskKind = iota
	TaskTransport          // unload actions per cargo type
	TaskColonize
	TaskMerge // Merge with Fleet: join the fleet with id Task.Fleet
)

// Cargo types of a transport order: the three minerals by their Minerals
// index, then colonists. Fuel is never unloaded to a planet (KERNEL.md
// "Fuel cannot be unloaded onto a planet", CONFIRMED FM-101..105), so it
// has no action here.
const (
	CargoColonists = NumMinerals
	NumCargo       = NumMinerals + 1
)

// TransportAction is a transport order's action for one cargo type
// (TAKEOVER.md "Unload and load amounts"). Only the unload actions are
// modelled.
type TransportAction int

const (
	TransportNone TransportAction = iota
	UnloadAll                     // C
	UnloadExactly                 // min(v, C)
)

// Transport is one cargo type's action and amount v (kT; colonists in
// units of 100).
type Transport struct {
	Action TransportAction
	Amount int
}

// Task is the task of a fleet's current location (the original's
// waypoint 0), or of a waypoint.
type Task struct {
	Kind      TaskKind
	Transport [NumCargo]Transport // for TaskTransport
	Fleet     int                 // target fleet id, for TaskMerge
}

// planetOrder is the planet indices in planet-id order.
func (g *Game) planetOrder() []int {
	order := make([]int, len(g.Planets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return g.Planets[order[a]].ID < g.Planets[order[b]].ID })
	return order
}

// fleetOrder is the fleet indices in fleet order: by owner, then by fleet
// number (TAKEOVER.md "Order inside a phase", the order of COMBAT.md).
func (g *Game) fleetOrder() []int {
	order := make([]int, len(g.Fleets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		fa, fb := &g.Fleets[order[a]], &g.Fleets[order[b]]
		if fa.Owner != fb.Owner {
			return fa.Owner < fb.Owner
		}
		if fa.Number != fb.Number {
			return fa.Number < fb.Number
		}
		return fa.ID < fb.ID
	})
	return order
}

// roundHalfUp is TAKEOVER.md's round(x) for x ≥ 0.
func roundHalfUp(x float64) int { return int(math.Floor(x + 0.5)) }

// bestDefense is the coverage c (permille) of the best planetary defense
// player's current tech allows (TAKEOVER.md "Planetary defenses against
// bombs", CONFIRMED T-6..T-9; SCANNING.md uses the same choice).
func (g *Game) bestDefense(player int) (int, bool) {
	pl := &g.Players[player]
	v, ok := 0, false
	for _, d := range g.Defenses {
		allowed := true
		for f := range NumFields {
			if pl.Research.Levels[f] < d.TechReq[f] {
				allowed = false
			}
		}
		if allowed && (!ok || d.Coverage > v) {
			v, ok = d.Coverage, true
		}
	}
	return v, ok
}

// survival is the planet's bombing survival factors s and sSmart
// (TAKEOVER.md "Planetary defenses against bombs", CONFIRMED T-6..T-9):
// with n = min(installed, operable defenses) of the owner's best defense
// c, s = (1 − c/1000)^n and sSmart = (1 − c/2000)^n, both stored in single
// precision. Both are 1 for an unowned planet, a planet without defenses
// or an owner without a defense part.
func (g *Game) survival(p *Planet) (s, sSmart float32) {
	if p.Owner == NoOwner || p.Defenses <= 0 {
		return 1, 1
	}
	c, ok := g.bestDefense(p.Owner)
	if !ok {
		return 1, 1
	}
	n := float64(min(p.Defenses, NewColony(p, &g.Players[p.Owner]).OperableDefenses(p.Population)))
	return float32(math.Pow(1-float64(c)/1000, n)), float32(math.Pow(1-float64(c)/2000, n))
}

// bombsOwner reports whether a fleet plan's attack-who covers a planet
// owner for bombing (TAKEOVER.md "Who bombs", CONFIRMED T-3, T-19, T-20):
// "enemies" covers enemies, "enemies and neutrals" anyone not a friend,
// "everyone" anyone, and a named player only that player.
func (g *Game) bombsOwner(bomber int, plan BattlePlan, owner int) bool {
	rel := g.relation(bomber, owner)
	switch plan.Attack {
	case AttackEnemies:
		return rel == RelationEnemy
	case AttackNeutralsAndEnemies:
		return rel != RelationFriend
	case AttackEveryone:
		return true
	case AttackPlayer:
		return plan.Player == owner
	}
	return false
}

// bombPass is one player's bomb totals over every fleet it has in orbit
// (TAKEOVER.md "Bomb totals", CONFIRMED T-10..T-18). A, I, M and S are
// before defenses; M is in units of 100 colonists.
type bombPass struct {
	A, I, M, R int
	pi         float64 // Π, the product of smart survival rates
}

func (b *bombPass) add(p Part, count int) {
	switch p.Bomb {
	case BombNormal:
		b.A += count * p.KillRate
		b.I += count * p.InstallKill
		b.M += count * (p.MinKill / ColonistsPerUnit)
	case BombSmart:
		b.pi *= math.Pow(1-float64(p.KillRate)/1000, float64(count))
	case BombRetro:
		b.R += count
	}
}

// smart is S = min(1000, round(1000 − 1000·Π)).
func (b bombPass) smart() int { return min(1000, roundHalfUp(1000-1000*b.pi)) }

// bombing is the bombing phase, after every battle of the year
// (TAKEOVER.md "Where each task happens", step 4, and "Bombing order",
// BINARY-ONLY): fleets are walked in fleet order, and the first fleet of a
// player that qualifies at a planet triggers that player's single pass
// there, applied at once with its draws. Unowned planets and planets with
// a starbase of any kind are never bombed (CONFIRMED T-3); an emptied
// planet is unowned, so later passes skip it.
func bombing(g *Game, rng Rand) []Event {
	var events []Event
	done := map[[2]int]bool{} // player, planet index
	for _, i := range g.fleetOrder() {
		f := &g.Fleets[i]
		pi := g.planetAt(f.Pos)
		if pi < 0 {
			continue
		}
		p := &g.Planets[pi]
		key := [2]int{f.Owner, pi}
		if p.Owner == NoOwner || p.Owner == f.Owner || g.hasStarbase(pi) || done[key] {
			continue
		}
		if !g.bombsOwner(f.Owner, g.plan(f.Owner, f.Plan), p.Owner) {
			continue
		}
		done[key] = true
		events = append(events, g.bombPlanet(pi, f.Owner, g.bomberPass(f.Owner, pi), rng)...)
	}
	return events
}

// bomberPass sums every bomb of every fleet bomber has in orbit at planet
// pi (CONFIRMED T-20, TK-005: once one fleet qualifies, the player's
// fleets bomb as one, whatever the other fleets' plans, and the
// qualifying fleet needs no bombs).
func (g *Game) bomberPass(bomber, pi int) bombPass {
	p := &g.Planets[pi]
	pass := bombPass{pi: 1}
	for i := range g.Fleets {
		f := &g.Fleets[i]
		if f.Owner != bomber || f.Pos != p.Pos {
			continue
		}
		for _, st := range f.Stacks {
			for _, sl := range g.Designs[st.Design].Slots {
				pass.add(sl.Part, st.Count*sl.Count)
			}
		}
	}
	return pass
}

// bombPlanet applies one player's pass (TAKEOVER.md "Applying the pass",
// CONFIRMED T-10..T-16; the random roundings MEASURED). Draws: factories,
// defenses, population, each only when its remainder is non-zero.
func (g *Game) bombPlanet(pi, bomber int, pass bombPass, rng Rand) []Event {
	p := &g.Planets[pi]
	s, sSmart := g.survival(p)
	sf, ssf := float64(s), float64(sSmart)
	A := roundHalfUp(float64(pass.A) * sf)
	M := roundHalfUp(float64(pass.M) * sf)
	S := roundHalfUp(float64(pass.smart()) * ssf)
	I := roundHalfUp(float64(pass.I) * (1 - (1-sf)/2))

	if T := p.Mines + p.Factories + p.Defenses; I > 0 && T > 0 {
		lose := func(have int) int {
			n := I * have / T
			if r := I * have % T; r != 0 && rng.Intn(T) < r {
				n++
			}
			return min(n, have)
		}
		fl := lose(p.Factories)
		dl := lose(p.Defenses)
		// When both rounded-up kills exceed I, mines lose nothing
		// (BINARY-ONLY).
		ml := min(max(0, I-fl-dl), p.Mines)
		p.Factories -= fl
		p.Defenses -= dl
		p.Mines -= ml
	}

	killed := 0
	if P := p.Population; P > 0 {
		k1 := min(P*S/1000, P-1)
		x := (P - k1) * A
		k2 := x / 1000
		if r := x % 1000; r != 0 && rng.Intn(1000) <= r {
			k2++
		}
		k := k1 + k2
		if A > 0 && k == 0 {
			k = 1
		}
		k = min(max(k, M), P)
		p.Population -= k
		killed = k
	}

	if pass.R > 0 {
		// Retro bombs (CONFIRMED T-16): each axis moves toward its
		// original value by up to R clicks, separately.
		R := min(500, pass.R-int((1-sf)*float64(pass.R)/2))
		for a := range p.Env {
			d := p.OrigEnv[a] - p.Env[a]
			p.Env[a] += max(-R, min(R, d))
		}
	}

	events := []Event{{Kind: EventBombed, Player: bomber, Planet: p.ID, Fleet: -1, Count: killed}}
	if p.Population == 0 {
		events = append(events, g.emptyPlanet(pi))
	}
	return events
}

// emptyPlanet empties a planet that lost its whole population (TAKEOVER.md
// "Capture: what a planet keeps", CONFIRMED T-21, T-26, T-27): mines,
// factories, minerals, concentrations, environment and the growth carry
// stay; owner, population, defenses, scanner, queue, starbase and the
// leftover setting go. A Claim Adjuster owner's planet returns to its
// original environment (CONFIRMED, TK-116).
//
// Not modelled: ancient artifacts (Elegy has no random events).
func (g *Game) emptyPlanet(pi int) Event {
	p := &g.Planets[pi]
	old := p.Owner
	if old != NoOwner && g.Players[old].Race.PRT == PRTClaimAdjuster {
		p.Env = p.OrigEnv
	}
	p.Owner, p.Population = NoOwner, 0
	p.Defenses, p.HasScanner = 0, false
	p.HasQueue, p.Queue, p.LeftoverOnly = false, nil, false
	p.HasStarbase, p.StarbaseHull, p.StarbaseDock, p.StarbaseDamage = false, 0, false, 0
	return Event{Kind: EventPlanetEmptied, Player: old, Planet: p.ID, Fleet: -1}
}

// newColony gives an empty planet to player with pop units (TAKEOVER.md
// "Colonization"): the owner's default production queue, less the first
// three items for Alternate Reality and the fifth and sixth for Claim
// Adjuster, and the owner's default leftover setting (BINARY-ONLY).
//
// Not modelled: the starbase of the owner's first starbase design an
// Alternate Reality colony gets (Elegy's designs have no owner yet).
func (g *Game) newColony(pi, player, pop int) Event {
	p := &g.Planets[pi]
	pl := &g.Players[player]
	p.Owner, p.Population = player, pop
	p.HasQueue, p.Queue = true, nil
	for i, it := range pl.DefaultQueue {
		switch {
		case pl.Race.PRT == PRTAlternateReality && i < 3,
			pl.Race.PRT == PRTClaimAdjuster && (i == 4 || i == 5):
			continue
		}
		p.Queue = append(p.Queue, it)
	}
	p.LeftoverOnly = pl.DefaultLeftoverOnly
	return Event{Kind: EventColonized, Player: player, Planet: p.ID, Fleet: -1, Count: pop}
}

// drop is colonists queued to land on a planet (planet index).
type drop struct {
	planet, player, troops int
}

// phaseStart records which planets are owned at the start of a waypoint
// phase, before anything else in it (TAKEOVER.md "Order inside a phase":
// before step 2, and before the battles for steps 4–5).
func (g *Game) phaseStart() []bool {
	owned := make([]bool, len(g.Planets))
	for i, p := range g.Planets {
		owned[i] = p.Owner != NoOwner
	}
	return owned
}

// unloadPhase is a waypoint phase's unload pass (TAKEOVER.md "Where each
// task happens", steps 2 and 5): each fleet in fleet order carries out its
// whole task. It returns the drops queued, in the order they were made.
//
// Not modelled: scrap, remote mining, mine laying, and colonists given by
// manual cargo transfers (which would start the queue before movement).
// The load pass is loadPass.
func (g *Game) unloadPhase(owned []bool) (queue []drop, events []Event) {
	consumed := map[int]bool{}
	for _, i := range g.fleetOrder() {
		f := &g.Fleets[i]
		switch f.Task.Kind {
		case TaskTransport:
			events = append(events, g.unload(f, owned, &queue)...)
		case TaskColonize:
			if ev, ok := g.colonize(f, &queue); ok {
				consumed[f.ID] = true
			} else {
				events = append(events, ev)
			}
		}
	}
	g.removeFleets(consumed)
	return queue, events
}

func (g *Game) removeFleets(ids map[int]bool) {
	if len(ids) == 0 {
		return
	}
	fleets := g.Fleets[:0]
	for _, f := range g.Fleets {
		if !ids[f.ID] {
			fleets = append(fleets, f)
		}
	}
	g.Fleets = fleets
}

// unload runs a transport task's unload actions at the orbited planet
// (TAKEOVER.md "Unload and load amounts", BINARY-ONLY): "unload all"
// moves C, "unload exactly v" moves min(v, C). The actions then clear, so
// they happen once. Minerals join the planet's surface whoever owns it,
// with no relation check. On the owner's own planet colonists join the
// population at once, with no cap; colonists for any other planet follow
// "Unloading colonists on another player's planet".
//
// In deep space minerals are destroyed, with no salvage, and colonists
// are refused (BINARY-ONLY).
//
// ASSUMPTION T1: Elegy's task does not record what a waypoint pointed
// at, so a fleet away from any planet but sharing its position with
// another fleet or a salvage object keeps its cargo: unloads to fleets
// and salvage are not modelled. Anywhere else away from a planet is deep
// space.
func (g *Game) unload(f *Fleet, owned []bool, queue *[]drop) []Event {
	var events []Event
	pi := g.planetAt(f.Pos)
	for c := range NumCargo {
		t := &f.Task.Transport[c]
		amount := 0
		have := f.Cargo.Colonists
		if c < NumMinerals {
			have = f.Cargo.Minerals[c]
		}
		switch t.Action {
		case UnloadAll:
			amount = have
		case UnloadExactly:
			amount = min(t.Amount, have)
		default:
			continue
		}
		t.Action, t.Amount = TransportNone, 0
		if amount <= 0 {
			continue
		}
		if pi < 0 {
			switch {
			case !g.deepSpace(f):
			case c < NumMinerals:
				f.Cargo.Minerals[c] -= amount
			default:
				events = append(events, Event{Kind: EventDropRefused, Player: f.Owner, Planet: -1, Fleet: f.ID, Count: amount})
			}
			continue
		}
		p := &g.Planets[pi]
		switch {
		case c < NumMinerals:
			p.Surface[c] += amount
			f.Cargo.Minerals[c] -= amount
		case p.Owner == f.Owner:
			p.Population += amount
			f.Cargo.Colonists -= amount
		case c == CargoColonists:
			events = append(events, g.dropColonists(f, pi, amount, owned, queue)...)
		}
	}
	return events
}

// deepSpace reports a fleet away from any planet with no other fleet or
// salvage object at its position (ASSUMPTION T1).
func (g *Game) deepSpace(f *Fleet) bool {
	for _, o := range g.Fleets {
		if o.ID != f.ID && o.Pos == f.Pos {
			return false
		}
	}
	for _, s := range g.Salvage {
		if s.Pos == f.Pos {
			return false
		}
	}
	return true
}

// dropColonists is an unload of colonists on a planet the fleet's owner
// does not own (TAKEOVER.md "Unloading colonists on another player's
// planet", CONFIRMED T-4, T-28, T-29; the Alternate Reality refusal
// BINARY-ONLY). The player relation is not checked.
func (g *Game) dropColonists(f *Fleet, pi, amount int, owned []bool, queue *[]drop) []Event {
	p := &g.Planets[pi]
	switch {
	case p.Owner == NoOwner && !owned[pi],
		g.hasStarbase(pi),
		g.Players[f.Owner].Race.PRT == PRTAlternateReality:
		return []Event{{Kind: EventDropRefused, Player: f.Owner, Planet: p.ID, Fleet: f.ID, Count: amount}}
	}
	*queue = append(*queue, drop{pi, f.Owner, amount})
	f.Cargo.Colonists -= amount
	return nil
}

// colonize is a colonize order (TAKEOVER.md "Colonization", CONFIRMED
// T-1, T-30, T-31; requirements BINARY-ONLY): the fleet must orbit a
// planet that is unowned now, carry colonists and have a ship with a
// colonizing module. There is no habitability check. On success the whole
// fleet is consumed, the planet gains ⌊3C/4⌋ of each mineral of the
// fleet's cost for its owner this year plus its mineral cargo, and the
// colonists are queued as a drop; the colony's event comes when the drop
// is resolved.
//
// The order is tried once (TAKEOVER.md "Colonize is tried once",
// CONFIRMED TK-113): on any failure the task is cleared and the fleet
// keeps its cargo; there is no retry.
func (g *Game) colonize(f *Fleet, queue *[]drop) (ev Event, ok bool) {
	pi := g.planetAt(f.Pos)
	fail := Event{Kind: EventColonizeFailed, Player: f.Owner, Planet: -1, Fleet: f.ID}
	if pi >= 0 {
		fail.Planet = g.Planets[pi].ID
	}
	if pi < 0 || g.Planets[pi].Owner != NoOwner || f.Cargo.Colonists <= 0 || !g.canColonize(f) {
		f.Task = Task{}
		return fail, false
	}
	p := &g.Planets[pi]
	pl := &g.Players[f.Owner]
	var C Minerals
	for _, st := range f.Stacks {
		c := designCost(g.Designs[st.Design], pl.Race, pl.Research.Levels)
		for m := range NumMinerals {
			C[m] += st.Count * c.Minerals[m]
		}
	}
	for m := range NumMinerals {
		p.Surface[m] += 3*C[m]/4 + f.Cargo.Minerals[m]
	}
	*queue = append(*queue, drop{pi, f.Owner, f.Cargo.Colonists})
	return Event{}, true
}

func (g *Game) canColonize(f *Fleet) bool {
	for _, st := range f.Stacks {
		if st.Count <= 0 {
			continue
		}
		for _, sl := range g.Designs[st.Design].Slots {
			if sl.Count > 0 && sl.Part.Colonizer {
				return true
			}
		}
	}
	return false
}

// resolveQueue resolves queued drops (TAKEOVER.md "Order inside a phase",
// BINARY-ONLY): the queue is walked from the front, and the first
// unresolved entry's planet is resolved with all of that planet's
// entries together.
func (g *Game) resolveQueue(queue []drop, rng Rand, gained map[int]bool) []Event {
	var events []Event
	done := map[int]bool{}
	for _, d := range queue {
		if done[d.planet] {
			continue
		}
		done[d.planet] = true
		troops := make([]int, len(g.Players))
		for _, e := range queue {
			if e.planet == d.planet {
				troops[e.player] += e.troops
			}
		}
		events = append(events, g.resolveDrops(d.planet, troops, rng, gained)...)
	}
	return events
}

// groundStrength is a player's ground strength factor k (TAKEOVER.md
// "Ground combat"): 110, War Monger 165 and Alternate Reality 0 (both
// BINARY-ONLY).
func groundStrength(r Race) int {
	switch r.PRT {
	case PRTWarMonger:
		return 165
	case PRTAlternateReality:
		return 0
	}
	return 110
}

// resolveDrops resolves every drop on one planet, troops by player
// (TAKEOVER.md "Ground combat", CONFIRMED T-21..T-25, and "Several
// players dropping at once", LEGACY BUG, CONFIRMED T-32). A capture of
// an owned planet gives the winner one tech attempt against the old
// owner's levels (TAKEOVER.md "Capture", BINARY-ONLY).
func (g *Game) resolveDrops(pi int, troops []int, rng Rand, gained map[int]bool) []Event {
	p := &g.Planets[pi]
	s, _ := g.survival(p)
	s2 := s + (1-s)/4 // defenses are 75% as effective against troops
	strength := make([]int, len(g.Players))
	sum := 0
	for q, t := range troops {
		if t > 0 {
			strength[q] = int(float64(t*groundStrength(g.Players[q].Race)/100) * float64(s2))
			sum += strength[q]
		}
	}

	var events []Event
	repelled := func(skip int) {
		for q, t := range troops {
			if t > 0 && q != skip {
				events = append(events, Event{Kind: EventDropRepelled, Player: q, Planet: p.ID, Fleet: -1, Count: t})
			}
		}
	}
	D, captured := 0, false
	var oldLevels [NumFields]int
	if p.Owner != NoOwner {
		D = p.Population
		if g.Players[p.Owner].Race.PRT == PRTInnerStrength {
			D *= 2 // BINARY-ONLY
		}
		if D > sum {
			p.Population -= p.Population * sum / D
			repelled(-1)
			return events
		}
		// A tie goes to the attackers.
		captured, oldLevels = true, g.Players[p.Owner].Research.Levels
		events = append(events, g.emptyPlanet(pi))
	}

	w, best, second, tie := g.dropWinner(troops, strength)
	if tie {
		repelled(-1)
		return events
	}
	land := troops[w]
	if sum > 0 {
		land = max(1, troops[w]*((sum-D)*best/sum)/best)
	}
	if second > 0 {
		land = max(1, land*(best-second)/best)
	}
	repelled(w)
	events = append(events, g.newColony(pi, w, land))
	if captured {
		events = append(events, techAttempt(g, rng, w, oldLevels, gained)...)
	}
	return events
}

// legacyDropScan reproduces the original's LEGACY BUG in picking the
// winner of several players' drops (TAKEOVER.md "Several players
// dropping at once", CONFIRMED T-32): a scan in player order in which
// "second" only ever holds a lower-index player's strength, so only a
// higher-index winner is reduced. Set it to false to compare every
// winner against its strongest rival.
const legacyDropScan = true

// dropWinner picks the winner w among the dropping players, with the
// strengths best and second its landing is reduced by, and whether a tie
// leaves nobody landing.
//
// The scan starts at the first dropping player, so a lone attacker of
// strength 0 (Alternate Reality colonists, k = 0) is not a tie with an
// empty start; TAKEOVER.md does not give the start value.
func (g *Game) dropWinner(troops, strength []int) (w, best, second int, tie bool) {
	w = -1
	for q, t := range troops {
		if t <= 0 {
			continue
		}
		switch st := strength[q]; {
		case w < 0:
			w, best = q, st
		case st >= best:
			tie = st == best
			w, second, best = q, best, st
		}
	}
	if legacyDropScan {
		return w, best, second, tie
	}
	second, tie = 0, false
	for q, t := range troops {
		if t > 0 && q != w {
			second = max(second, strength[q])
			tie = tie || strength[q] == best
		}
	}
	return w, best, second, tie
}

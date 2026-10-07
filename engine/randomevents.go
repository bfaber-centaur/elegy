package engine

import "sort"

// Random events (stars-elegy KERNEL.md "Random events"): a comet strike, a
// planetary climate change and a new-minerals discovery, run at the end of
// production when the game's random events option is on. The Mystery
// Trader, which follows them, is not modelled.
//
// The rules and their draw order are CONFIRMED (KX-004); the probabilities
// are BINARY-ONLY.

// Random event messages. Player is the recipient, Planet the planet id.
const (
	// Count = comet size 0..3 (small, medium, large, huge). Axes, for the
	// owner's "colonists killed" form only, are the environment axes the
	// message names.
	EventCometStrike   EventKind = iota + EventColonistsLostInFlight + 1
	EventClimateChange           // to the owner; Axes = the axis
	EventNewMinerals             // to the owner; Count = the mineral (Minerals index)
)

// legacyCometAxes reproduces the original's LEGACY BUG in the comet
// strike message (KERNEL.md "Comet strike", CONFIRMED KX-004 S2): the
// owner's message names the axes of a shuffled order A, while the axes
// that move are the first ones in index order. Set it to false to name
// the axes that moved. Only the message differs.
const legacyCometAxes = true

// yearIndex is the year counted from 2400, before the year advances.
func (g *Game) yearIndex() int { return g.Year - 2400 }

// protected is KERNEL.md's "protected": owned, more than 50 units of
// population (after this year's growth) and year index below 20.
func (g *Game) protected(p *Planet) bool {
	return p.Owner != NoOwner && p.Population > 50 && g.yearIndex() < 20
}

// randomEvents runs the three events in order when the option is on.
func (g *Game) randomEvents(rng Rand) []Event {
	if !g.RandomEvents || len(g.Planets) == 0 {
		return nil
	}
	order := make([]int, len(g.Planets))
	for i := range order {
		order[i] = i
	}
	// "rand(planets)" indexes the planets in id order.
	sort.SliceStable(order, func(a, b int) bool { return g.Planets[order[a]].ID < g.Planets[order[b]].ID })
	var events []Event
	events = append(events, g.cometStrike(rng, order)...)
	events = append(events, g.climateChange(rng, order)...)
	return append(events, g.newMinerals(rng, order)...)
}

// cutQueue keeps only the automatic items of a planet's queue, with their
// counts.
func cutQueue(p *Planet) {
	var keep []QueueItem
	for _, it := range p.Queue {
		if it.Kind.auto() {
			keep = append(keep, it)
		}
	}
	p.Queue = keep
}

// shiftEnv moves an axis's current and original value by s, each clamped
// to 1..99.
func shiftEnv(p *Planet, axis, s int) {
	p.Env[axis] = min(99, max(1, p.Env[axis]+s))
	p.OrigEnv[axis] = min(99, max(1, p.OrigEnv[axis]+s))
}

// cometAxes is the axes the owner's comet message names: the first moved
// of the shuffled order a (legacy), or the moved axes in index order.
func cometAxes(a [3]int, moved int, legacy bool) []int {
	if legacy {
		return append([]int(nil), a[:moved]...)
	}
	return []int{0, 1, 2}[:moved]
}

// cometStrike is KERNEL.md "Comet strike". An Alternate Reality owner gets
// the plain message and loses no population (BINARY-ONLY).
func (g *Game) cometStrike(rng Rand, order []int) []Event {
	if rng.Intn(20) != 0 {
		return nil
	}
	p := &g.Planets[order[rng.Intn(len(order))]]
	if g.protected(p) || g.yearIndex() < 10 {
		return nil
	}
	e := rng.Intn(4)
	huge := e == 3
	a := [3]int{0, 1, 2}
	for i := range a {
		j := rng.Intn(3)
		a[i], a[j] = a[j], a[i]
	}
	var b [NumMinerals]int
	for i := range b {
		b[i] = 50 + rng.Intn(250)
	}
	bo := [3]int{0, 1, 2}
	j := rng.Intn(3)
	bo[0], bo[j] = bo[j], bo[0]
	j = 1 + rng.Intn(2)
	bo[1], bo[j] = bo[j], bo[1]

	killed := p.Owner != NoOwner && g.Players[p.Owner].Race.PRT != PRTAlternateReality
	moved := min(e, 2) + 1
	var events []Event
	for v := range g.Players {
		ev := Event{Kind: EventCometStrike, Player: v, Planet: p.ID, Fleet: -1, Count: e}
		if killed && v == p.Owner {
			ev.Axes = cometAxes(a, moved, legacyCometAxes)
		}
		events = append(events, ev)
	}
	if killed {
		p.Population -= p.Population * (20*e + 25) / 100
	}
	for k := range moved {
		m := bo[k]
		b[m] += 3000 + rng.Intn(17000)
		c := p.Deposits[m].Concentration + 50 + rng.Intn(50)
		if huge {
			c += 15 + rng.Intn(15)
		}
		p.Deposits[m].Concentration = min(200, c)
	}
	for i := range b {
		p.Surface[i] += b[i] / 16
	}
	for d := range moved {
		s := 3 + rng.Intn(3)
		if huge {
			s += 3 + rng.Intn(3)
		}
		if rng.Intn(2) != 0 {
			s = -s
		}
		shiftEnv(p, d, s)
	}
	cutQueue(p)
	return events
}

// climateChange is KERNEL.md "Planetary climate change".
func (g *Game) climateChange(rng Rand, order []int) []Event {
	if rng.Intn(20) != 0 {
		return nil
	}
	p := &g.Planets[order[rng.Intn(len(order))]]
	if g.protected(p) {
		return nil
	}
	axis := rng.Intn(3)
	var events []Event
	if p.Owner != NoOwner {
		events = append(events, Event{Kind: EventClimateChange, Player: p.Owner, Planet: p.ID, Fleet: -1, Axes: []int{axis}})
	}
	s := 3 + rng.Intn(3)
	if s == 3 {
		s = 6 + rng.Intn(3)
	}
	if rng.Intn(2) != 0 {
		s = -s
	}
	shiftEnv(p, axis, s)
	cutQueue(p)
	return events
}

// newMinerals is KERNEL.md "New minerals" (the cap at 180 BINARY-ONLY).
func (g *Game) newMinerals(rng Rand, order []int) []Event {
	if rng.Intn(15-g.Size) != 0 {
		return nil
	}
	p := &g.Planets[order[rng.Intn(len(order))]]
	if g.yearIndex() < 10 {
		return nil
	}
	m := rng.Intn(3)
	var events []Event
	if p.Owner != NoOwner {
		events = append(events, Event{Kind: EventNewMinerals, Player: p.Owner, Planet: p.ID, Fleet: -1, Count: m})
	}
	if p.Deposits[m].Concentration < 180 {
		p.Deposits[m].Concentration += 5 + rng.Intn(15)
	}
	return events
}
